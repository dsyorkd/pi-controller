package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebUIConfig_GetAddress(t *testing.T) {
	tests := []struct {
		name string
		cfg  WebUIConfig
		want string
	}{
		{
			name: "default config",
			cfg:  WebUIConfig{Host: "0.0.0.0", Port: 3000},
			want: "0.0.0.0:3000",
		},
		{
			name: "localhost",
			cfg:  WebUIConfig{Host: "localhost", Port: 8080},
			want: "localhost:8080",
		},
		{
			name: "ipv6",
			cfg:  WebUIConfig{Host: "::", Port: 3000},
			want: ":::3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetAddress()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebUIConfig_GetBackendAPIURL(t *testing.T) {
	cfg := WebUIConfig{
		Backend: BackendConfig{
			API: BackendServiceConfig{
				URL:         "http://localhost:8080",
				InternalURL: "http://pi-controller-api:8080",
			},
		},
	}

	tests := []struct {
		name    string
		k8sMode bool
		want    string
	}{
		{
			name:    "binary mode uses external URL",
			k8sMode: false,
			want:    "http://localhost:8080",
		},
		{
			name:    "k8s mode uses internal URL",
			k8sMode: true,
			want:    "http://pi-controller-api:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetBackendAPIURL(tt.k8sMode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebUIConfig_GetBackendWebSocketURL(t *testing.T) {
	cfg := WebUIConfig{
		Backend: BackendConfig{
			WebSocket: BackendServiceConfig{
				URL:         "ws://localhost:8081",
				InternalURL: "ws://pi-controller-ws:8081",
			},
		},
	}

	tests := []struct {
		name    string
		k8sMode bool
		want    string
	}{
		{
			name:    "binary mode",
			k8sMode: false,
			want:    "ws://localhost:8081",
		},
		{
			name:    "k8s mode",
			k8sMode: true,
			want:    "ws://pi-controller-ws:8081",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetBackendWebSocketURL(tt.k8sMode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebUIConfig_GetBackendGRPCURL(t *testing.T) {
	cfg := WebUIConfig{
		Backend: BackendConfig{
			GRPC: BackendServiceConfig{
				URL:         "localhost:9090",
				InternalURL: "pi-controller-grpc:9090",
			},
		},
	}

	tests := []struct {
		name    string
		k8sMode bool
		want    string
	}{
		{
			name:    "binary mode",
			k8sMode: false,
			want:    "localhost:9090",
		},
		{
			name:    "k8s mode",
			k8sMode: true,
			want:    "pi-controller-grpc:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetBackendGRPCURL(tt.k8sMode)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebUIDefaults(t *testing.T) {
	t.Run("production defaults", func(t *testing.T) {
		cfg := getProductionDefaults()

		// WebUI should be enabled by default
		assert.True(t, cfg.WebUI.Enabled)
		assert.Equal(t, "0.0.0.0", cfg.WebUI.Host)
		assert.Equal(t, 3000, cfg.WebUI.Port)
		assert.Equal(t, "./web/dist", cfg.WebUI.StaticDir)
		assert.True(t, cfg.WebUI.SPAMode)

		// Backend URLs
		assert.Equal(t, "http://localhost:8080", cfg.WebUI.Backend.API.URL)
		assert.Equal(t, "http://pi-controller-api:8080", cfg.WebUI.Backend.API.InternalURL)
		assert.Equal(t, "/api", cfg.WebUI.Backend.API.Prefix)

		// Runtime config injection
		assert.True(t, cfg.WebUI.RuntimeConfig.Enabled)
		assert.Equal(t, "/config.js", cfg.WebUI.RuntimeConfig.Path)

		// Auth settings (production should be secure)
		assert.True(t, cfg.WebUI.Auth.Enabled)
		assert.True(t, cfg.WebUI.Auth.CookieSecure)
		assert.Equal(t, "strict", cfg.WebUI.Auth.CookieSameSite)
		assert.Equal(t, "24h", cfg.WebUI.Auth.SessionTimeout)

		// CORS
		assert.True(t, cfg.WebUI.CORS.Enabled)
		assert.Contains(t, cfg.WebUI.CORS.AllowedOrigins, "http://localhost:3000")
		assert.True(t, cfg.WebUI.CORS.Credentials)

		// Feature flags
		assert.True(t, cfg.WebUI.Features.GPIOControl)
		assert.True(t, cfg.WebUI.Features.ClusterManagement)
		assert.True(t, cfg.WebUI.Features.CertificateManagement)
		assert.False(t, cfg.WebUI.Features.Experimental)

		// Branding
		assert.Equal(t, "Pi Controller", cfg.WebUI.Branding.Title)
		assert.Equal(t, "#10b981", cfg.WebUI.Branding.PrimaryColor)
		assert.Equal(t, "dark", cfg.WebUI.Branding.Theme)

		// Cache
		assert.True(t, cfg.WebUI.Cache.Enabled)
		assert.Equal(t, "31536000", cfg.WebUI.Cache.StaticMaxAge)
		assert.Equal(t, "0", cfg.WebUI.Cache.HTMLMaxAge)

		// Compression
		assert.True(t, cfg.WebUI.Compression.Enabled)
		assert.Equal(t, 6, cfg.WebUI.Compression.Level)
		assert.Equal(t, 1024, cfg.WebUI.Compression.MinSize)

		// Security (production should be strict)
		assert.True(t, cfg.WebUI.Security.HSTSEnabled)
		assert.True(t, cfg.WebUI.Security.FrameDeny)
		assert.True(t, cfg.WebUI.Security.ContentTypeNoSniff)
		assert.True(t, cfg.WebUI.Security.XSSProtection)

		// Rate limiting
		assert.True(t, cfg.WebUI.RateLimit.Enabled)
		assert.Equal(t, 60, cfg.WebUI.RateLimit.RequestsPerMinute)
		assert.Equal(t, 100, cfg.WebUI.RateLimit.BurstSize)

		// Health checks
		assert.True(t, cfg.WebUI.Health.Enabled)
		assert.Equal(t, "/health", cfg.WebUI.Health.Path)
		assert.Equal(t, "/health/live", cfg.WebUI.Health.LivenessPath)
		assert.Equal(t, "/health/ready", cfg.WebUI.Health.ReadinessPath)

		// Resources
		assert.Equal(t, "500m", cfg.WebUI.Resources.CPULimit)
		assert.Equal(t, "512Mi", cfg.WebUI.Resources.MemoryLimit)
		assert.Equal(t, "100m", cfg.WebUI.Resources.CPURequest)
		assert.Equal(t, "128Mi", cfg.WebUI.Resources.MemoryRequest)
	})

	t.Run("development defaults", func(t *testing.T) {
		cfg := getDevelopmentDefaults()

		// Development should have relaxed security
		assert.False(t, cfg.WebUI.Auth.CookieSecure)
		assert.False(t, cfg.WebUI.Security.HSTSEnabled)
		assert.False(t, cfg.WebUI.Backend.TLS.Enabled)

		// Other settings should still be reasonable
		assert.True(t, cfg.WebUI.Enabled)
		assert.True(t, cfg.WebUI.Auth.Enabled)
	})
}

func TestFeatureFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    FeatureFlags
		validate func(*testing.T, FeatureFlags)
	}{
		{
			name: "all features enabled",
			flags: FeatureFlags{
				GPIOControl:           true,
				ClusterManagement:     true,
				CertificateManagement: true,
				RealTimeMetrics:       true,
				NodeDiscovery:         true,
				AdvancedNetworking:    true,
				Experimental:          true,
			},
			validate: func(t *testing.T, f FeatureFlags) {
				assert.True(t, f.GPIOControl)
				assert.True(t, f.ClusterManagement)
				assert.True(t, f.CertificateManagement)
				assert.True(t, f.RealTimeMetrics)
				assert.True(t, f.NodeDiscovery)
				assert.True(t, f.AdvancedNetworking)
				assert.True(t, f.Experimental)
			},
		},
		{
			name: "minimal features",
			flags: FeatureFlags{
				GPIOControl:       false,
				ClusterManagement: true,
				Experimental:      false,
			},
			validate: func(t *testing.T, f FeatureFlags) {
				assert.False(t, f.GPIOControl)
				assert.True(t, f.ClusterManagement)
				assert.False(t, f.Experimental)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.validate != nil {
				tt.validate(t, tt.flags)
			}
		})
	}
}

func TestBrandingConfig(t *testing.T) {
	tests := []struct {
		name     string
		branding BrandingConfig
		validate func(*testing.T, BrandingConfig)
	}{
		{
			name: "default branding",
			branding: BrandingConfig{
				Title:        "Pi Controller",
				PrimaryColor: "#10b981",
				Theme:        "dark",
			},
			validate: func(t *testing.T, b BrandingConfig) {
				assert.Equal(t, "Pi Controller", b.Title)
				assert.Equal(t, "#10b981", b.PrimaryColor)
				assert.Equal(t, "dark", b.Theme)
			},
		},
		{
			name: "custom branding",
			branding: BrandingConfig{
				Title:        "My Custom Controller",
				LogoURL:      "https://example.com/logo.png",
				FaviconURL:   "https://example.com/favicon.ico",
				PrimaryColor: "#ff0000",
				Theme:        "light",
			},
			validate: func(t *testing.T, b BrandingConfig) {
				assert.Equal(t, "My Custom Controller", b.Title)
				assert.Equal(t, "https://example.com/logo.png", b.LogoURL)
				assert.Equal(t, "https://example.com/favicon.ico", b.FaviconURL)
				assert.Equal(t, "#ff0000", b.PrimaryColor)
				assert.Equal(t, "light", b.Theme)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.validate != nil {
				tt.validate(t, tt.branding)
			}
		})
	}
}

func TestSecurityConfig(t *testing.T) {
	tests := []struct {
		name     string
		security SecurityConfig
		validate func(*testing.T, SecurityConfig)
	}{
		{
			name: "production security",
			security: SecurityConfig{
				HSTSEnabled:        true,
				HSTSMaxAge:         "31536000",
				FrameDeny:          true,
				ContentTypeNoSniff: true,
				XSSProtection:      true,
				CSPEnabled:         true,
				CSPDirectives:      "default-src 'self'",
			},
			validate: func(t *testing.T, s SecurityConfig) {
				assert.True(t, s.HSTSEnabled)
				assert.True(t, s.FrameDeny)
				assert.True(t, s.ContentTypeNoSniff)
				assert.True(t, s.XSSProtection)
				assert.True(t, s.CSPEnabled)
				assert.NotEmpty(t, s.CSPDirectives)
			},
		},
		{
			name: "development security",
			security: SecurityConfig{
				HSTSEnabled:        false,
				FrameDeny:          true,
				ContentTypeNoSniff: true,
				XSSProtection:      true,
				CSPEnabled:         false,
			},
			validate: func(t *testing.T, s SecurityConfig) {
				assert.False(t, s.HSTSEnabled)
				assert.False(t, s.CSPEnabled)
				assert.True(t, s.FrameDeny)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.validate != nil {
				tt.validate(t, tt.security)
			}
		})
	}
}

func TestCORSConfig(t *testing.T) {
	cfg := CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		Credentials:    true,
		MaxAge:         "12h",
	}

	assert.True(t, cfg.Enabled)
	assert.Len(t, cfg.AllowedOrigins, 2)
	assert.Contains(t, cfg.AllowedOrigins, "http://localhost:3000")
	assert.Len(t, cfg.AllowedMethods, 4)
	assert.True(t, cfg.Credentials)
}

func TestRateLimitConfig(t *testing.T) {
	cfg := RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         100,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 60, cfg.RequestsPerMinute)
	assert.Equal(t, 100, cfg.BurstSize)
}

func TestResourcesConfig(t *testing.T) {
	cfg := ResourcesConfig{
		CPULimit:      "500m",
		MemoryLimit:   "512Mi",
		CPURequest:    "100m",
		MemoryRequest: "128Mi",
	}

	assert.Equal(t, "500m", cfg.CPULimit)
	assert.Equal(t, "512Mi", cfg.MemoryLimit)
	assert.Equal(t, "100m", cfg.CPURequest)
	assert.Equal(t, "128Mi", cfg.MemoryRequest)
}

func TestHealthConfig(t *testing.T) {
	cfg := HealthConfig{
		Enabled:       true,
		Path:          "/health",
		LivenessPath:  "/health/live",
		ReadinessPath: "/health/ready",
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "/health", cfg.Path)
	assert.Equal(t, "/health/live", cfg.LivenessPath)
	assert.Equal(t, "/health/ready", cfg.ReadinessPath)
}