package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	applogger "github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityMiddleware_SecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := applogger.New(applogger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tests := []struct {
		name            string
		config          *SecurityConfig
		expectedHeaders map[string]string
	}{
		{
			name:   "Default config applies all security headers",
			config: DefaultSecurityConfig(),
			expectedHeaders: map[string]string{
				"X-Content-Type-Options":            "nosniff",
				"X-Frame-Options":                   "DENY",
				"X-XSS-Protection":                  "1; mode=block",
				"X-Permitted-Cross-Domain-Policies": "none",
				"X-DNS-Prefetch-Control":            "off",
				"Permissions-Policy":                "camera=(), microphone=(), geolocation=()",
				"Referrer-Policy":                   "strict-origin-when-cross-origin",
			},
		},
		{
			name: "Custom frame options",
			config: &SecurityConfig{
				ContentTypeOptions: true,
				FrameOptions:       "SAMEORIGIN",
				XSSProtection:      true,
			},
			expectedHeaders: map[string]string{
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "SAMEORIGIN",
				"X-XSS-Protection":       "1; mode=block",
			},
		},
		{
			name: "Disabled options",
			config: &SecurityConfig{
				ContentTypeOptions: false,
				FrameOptions:       "",
				XSSProtection:      false,
			},
			expectedHeaders: map[string]string{
				// These headers should still be present (always added)
				"X-Permitted-Cross-Domain-Policies": "none",
				"X-DNS-Prefetch-Control":            "off",
				"Permissions-Policy":                "camera=(), microphone=(), geolocation=()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSecurityMiddleware(tt.config, logger)
			require.NotNil(t, sm)

			router := gin.New()
			router.Use(sm.SecurityHeaders())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			for header, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, w.Header().Get(header), "Header %s mismatch", header)
			}
		})
	}
}

func TestSecurityMiddleware_ContentSecurityPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := applogger.New(applogger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	config := DefaultSecurityConfig()
	sm := NewSecurityMiddleware(config, logger)

	router := gin.New()
	router.Use(sm.SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestSecurityMiddleware_EnforceHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := applogger.New(applogger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		enforceHTTPS   bool
		protoHeader    string
		expectedCode   int
		expectRedirect bool
	}{
		{
			name:           "HTTP blocked when HTTPS enforced",
			enforceHTTPS:   true,
			protoHeader:    "",
			expectedCode:   http.StatusMovedPermanently,
			expectRedirect: true,
		},
		{
			name:           "HTTPS allowed when HTTPS enforced",
			enforceHTTPS:   true,
			protoHeader:    "https",
			expectedCode:   http.StatusOK,
			expectRedirect: false,
		},
		{
			name:           "HTTP allowed when HTTPS not enforced",
			enforceHTTPS:   false,
			protoHeader:    "",
			expectedCode:   http.StatusOK,
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SecurityConfig{
				EnforceHTTPS:      tt.enforceHTTPS,
				ProxyTrustHeaders: []string{"X-Forwarded-Proto"},
			}
			sm := NewSecurityMiddleware(config, logger)

			router := gin.New()
			router.Use(sm.EnforceHTTPS())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			req.Host = "example.com"

			if tt.protoHeader != "" {
				req.Header.Set("X-Forwarded-Proto", tt.protoHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectRedirect {
				location := w.Header().Get("Location")
				assert.Contains(t, location, "https://")
			}
		})
	}
}

func TestSecurityMiddleware_HSTSHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := applogger.New(applogger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	config := &SecurityConfig{
		EnforceHTTPS:       true,
		StrictTransportSec: true,
		STSMaxAge:          31536000,
		STSIncludeSubdom:   true,
		STSPreload:         true,
		ProxyTrustHeaders:  []string{"X-Forwarded-Proto"},
	}
	sm := NewSecurityMiddleware(config, logger)

	router := gin.New()
	router.Use(sm.EnforceHTTPS())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-Proto", "https")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")
}

func TestSecurityMiddleware_HostValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, err := applogger.New(applogger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		allowedHosts []string
		requestHost  string
		expectedCode int
	}{
		{
			name:         "Allowed host succeeds",
			allowedHosts: []string{"example.com", "api.example.com"},
			requestHost:  "example.com",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Blocked host fails",
			allowedHosts: []string{"example.com"},
			requestHost:  "evil.com",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Wildcard subdomain match",
			allowedHosts: []string{"*.example.com"},
			requestHost:  "api.example.com",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Empty allowed hosts allows all",
			allowedHosts: []string{},
			requestHost:  "any-host.com",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Host with port stripped",
			allowedHosts: []string{"example.com"},
			requestHost:  "example.com:8080",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SecurityConfig{
				AllowedHosts: tt.allowedHosts,
			}
			sm := NewSecurityMiddleware(config, logger)

			router := gin.New()
			router.Use(sm.HostValidation())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			require.NoError(t, err)
			req.Host = tt.requestHost

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestGetSecureTLSConfig(t *testing.T) {
	tlsConfig := GetSecureTLSConfig()
	require.NotNil(t, tlsConfig)

	// Verify minimum TLS version is 1.2
	assert.Equal(t, uint16(0x0303), tlsConfig.MinVersion) // TLS 1.2

	// Verify maximum TLS version is 1.3
	assert.Equal(t, uint16(0x0304), tlsConfig.MaxVersion) // TLS 1.3

	// Verify cipher suites are configured
	assert.NotEmpty(t, tlsConfig.CipherSuites)

	// Verify curve preferences are set
	assert.NotEmpty(t, tlsConfig.CurvePreferences)

	// Verify insecure options are disabled
	assert.False(t, tlsConfig.InsecureSkipVerify)
}

func TestSecurityMiddleware_GetSecurityStats(t *testing.T) {
	logger, err := applogger.New(applogger.Config{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)

	config := DefaultSecurityConfig()
	sm := NewSecurityMiddleware(config, logger)

	stats := sm.GetSecurityStats()

	assert.Equal(t, true, stats["enforce_https"])
	assert.Equal(t, true, stats["strict_transport_sec"])
	assert.Equal(t, 31536000, stats["sts_max_age"])
	assert.Equal(t, true, stats["content_type_options"])
	assert.Equal(t, "DENY", stats["frame_options"])
	assert.Equal(t, true, stats["xss_protection"])
	assert.Equal(t, true, stats["has_csp"])
}
