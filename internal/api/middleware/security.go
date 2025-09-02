package middleware

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	EnforceHTTPS        bool     `yaml:"enforce_https"`
	StrictTransportSec  bool     `yaml:"strict_transport_security"`
	STSMaxAge           int      `yaml:"sts_max_age"`
	STSIncludeSubdom    bool     `yaml:"sts_include_subdomains"`
	STSPreload          bool     `yaml:"sts_preload"`
	ContentTypeOptions  bool     `yaml:"content_type_options"`
	FrameOptions        string   `yaml:"frame_options"`
	XSSProtection       bool     `yaml:"xss_protection"`
	ContentSecPolicy    string   `yaml:"content_security_policy"`
	ReferrerPolicy      string   `yaml:"referrer_policy"`
	AllowedHosts        []string `yaml:"allowed_hosts"`
	ProxyTrustHeaders   []string `yaml:"proxy_trust_headers"`
}

// DefaultSecurityConfig returns secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnforceHTTPS:       true,
		StrictTransportSec: true,
		STSMaxAge:          31536000, // 1 year
		STSIncludeSubdom:   true,
		STSPreload:         true,
		ContentTypeOptions: true,
		FrameOptions:       "DENY",
		XSSProtection:      true,
		ContentSecPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; media-src 'self'; form-action 'self'; frame-ancestors 'none';",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		AllowedHosts:       []string{},
		ProxyTrustHeaders:  []string{"X-Forwarded-Proto", "X-Forwarded-For"},
	}
}

// SecurityMiddleware handles security headers and HTTPS enforcement
type SecurityMiddleware struct {
	config *SecurityConfig
	logger *logrus.Entry
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(config *SecurityConfig, logger *logrus.Logger) *SecurityMiddleware {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	sm := &SecurityMiddleware{
		config: config,
		logger: logger.WithField("component", "security"),
	}

	sm.logger.WithFields(logrus.Fields{
		"enforce_https":    config.EnforceHTTPS,
		"hsts_enabled":     config.StrictTransportSec,
		"frame_options":    config.FrameOptions,
		"xss_protection":   config.XSSProtection,
		"allowed_hosts":    len(config.AllowedHosts),
	}).Info("Security middleware initialized")

	return sm
}

// SecurityHeaders adds security headers to all responses
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		sm.addSecurityHeaders(c)

		c.Next()
	}
}

// EnforceHTTPS redirects HTTP requests to HTTPS and sets HSTS headers
func (sm *SecurityMiddleware) EnforceHTTPS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if HTTPS enforcement is enabled
		if !sm.config.EnforceHTTPS {
			c.Next()
			return
		}

		// Check if request is HTTPS
		if !sm.isHTTPS(c) {
			sm.logger.WithFields(logrus.Fields{
				"client_ip":   c.ClientIP(),
				"method":      c.Request.Method,
				"path":        c.Request.URL.Path,
				"user_agent":  c.GetHeader("User-Agent"),
			}).Warn("HTTP request blocked, HTTPS required")

			// Redirect to HTTPS
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(http.StatusMovedPermanently, httpsURL)
			c.Abort()
			return
		}

		// Add HSTS header for HTTPS requests
		if sm.config.StrictTransportSec {
			sm.addHSTSHeader(c)
		}

		c.Next()
	}
}

// HostValidation validates the Host header against allowed hosts
func (sm *SecurityMiddleware) HostValidation() gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(sm.config.AllowedHosts) == 0 {
			c.Next()
			return
		}

		host := sm.getHost(c)
		if !sm.isAllowedHost(host) {
			sm.logger.WithFields(logrus.Fields{
				"client_ip":  c.ClientIP(),
				"host":       host,
				"user_agent": c.GetHeader("User-Agent"),
			}).Warn("Request blocked: host not allowed")

			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request",
				"message": "Invalid host",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// addSecurityHeaders adds all configured security headers
func (sm *SecurityMiddleware) addSecurityHeaders(c *gin.Context) {
	// X-Content-Type-Options
	if sm.config.ContentTypeOptions {
		c.Header("X-Content-Type-Options", "nosniff")
	}

	// X-Frame-Options
	if sm.config.FrameOptions != "" {
		c.Header("X-Frame-Options", sm.config.FrameOptions)
	}

	// X-XSS-Protection
	if sm.config.XSSProtection {
		c.Header("X-XSS-Protection", "1; mode=block")
	}

	// Content-Security-Policy
	if sm.config.ContentSecPolicy != "" {
		c.Header("Content-Security-Policy", sm.config.ContentSecPolicy)
	}

	// Referrer-Policy
	if sm.config.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", sm.config.ReferrerPolicy)
	}

	// Additional security headers
	c.Header("X-Permitted-Cross-Domain-Policies", "none")
	c.Header("X-DNS-Prefetch-Control", "off")
	c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

// addHSTSHeader adds the Strict-Transport-Security header
func (sm *SecurityMiddleware) addHSTSHeader(c *gin.Context) {
	hsts := fmt.Sprintf("max-age=%d", sm.config.STSMaxAge)
	
	if sm.config.STSIncludeSubdom {
		hsts += "; includeSubDomains"
	}
	
	if sm.config.STSPreload {
		hsts += "; preload"
	}

	c.Header("Strict-Transport-Security", hsts)
}

// isHTTPS checks if the request is HTTPS
func (sm *SecurityMiddleware) isHTTPS(c *gin.Context) bool {
	// Direct TLS connection
	if c.Request.TLS != nil {
		return true
	}

	// Check proxy headers
	for _, header := range sm.config.ProxyTrustHeaders {
		if header == "X-Forwarded-Proto" {
			if c.GetHeader("X-Forwarded-Proto") == "https" {
				return true
			}
		}
		if header == "X-Forwarded-Ssl" {
			if c.GetHeader("X-Forwarded-Ssl") == "on" {
				return true
			}
		}
		if header == "X-Url-Scheme" {
			if c.GetHeader("X-Url-Scheme") == "https" {
				return true
			}
		}
	}

	return false
}

// getHost extracts the host from the request
func (sm *SecurityMiddleware) getHost(c *gin.Context) string {
	// Check X-Forwarded-Host first (if trusted)
	for _, header := range sm.config.ProxyTrustHeaders {
		if header == "X-Forwarded-Host" {
			if host := c.GetHeader("X-Forwarded-Host"); host != "" {
				return strings.Split(host, ",")[0] // Take first host if multiple
			}
		}
	}

	// Fall back to Host header
	return c.Request.Host
}

// isAllowedHost checks if a host is in the allowed hosts list
func (sm *SecurityMiddleware) isAllowedHost(host string) bool {
	// Remove port from host if present
	hostWithoutPort := strings.Split(host, ":")[0]

	for _, allowedHost := range sm.config.AllowedHosts {
		if allowedHost == host || allowedHost == hostWithoutPort {
			return true
		}
		// Support wildcard matching for subdomains
		if strings.HasPrefix(allowedHost, "*.") {
			domain := allowedHost[2:] // Remove *.
			if strings.HasSuffix(hostWithoutPort, "."+domain) || hostWithoutPort == domain {
				return true
			}
		}
	}

	return false
}

// GetSecureTLSConfig returns a secure TLS configuration
func GetSecureTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (automatically used when available)
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			
			// TLS 1.2 cipher suites
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   false,
		Renegotiation:           tls.RenegotiateNever,
		InsecureSkipVerify:      false,
	}
}

// ValidateTLSCertificates validates TLS certificate files
func ValidateTLSCertificates(certFile, keyFile string) error {
	if certFile == "" || keyFile == "" {
		return fmt.Errorf("TLS certificate and key files must be specified")
	}

	// Try to load the certificate pair
	_, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate pair: %w", err)
	}

	return nil
}

// GetSecurityStats returns security middleware statistics
func (sm *SecurityMiddleware) GetSecurityStats() map[string]interface{} {
	return map[string]interface{}{
		"enforce_https":          sm.config.EnforceHTTPS,
		"strict_transport_sec":   sm.config.StrictTransportSec,
		"sts_max_age":           sm.config.STSMaxAge,
		"content_type_options":   sm.config.ContentTypeOptions,
		"frame_options":          sm.config.FrameOptions,
		"xss_protection":         sm.config.XSSProtection,
		"has_csp":               sm.config.ContentSecPolicy != "",
		"allowed_hosts":          len(sm.config.AllowedHosts),
		"proxy_trust_headers":    len(sm.config.ProxyTrustHeaders),
	}
}
