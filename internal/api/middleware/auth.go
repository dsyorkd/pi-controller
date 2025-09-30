package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// AuthorizationHeader is the header name for authorization
	AuthorizationHeader = "Authorization"
	// UserIDKey is the context key for user ID
	UserIDKey = "user_id"
	// UserRoleKey is the context key for user role
	UserRoleKey = "user_role"
	// TokenTypeKey is the context key for token type
	TokenTypeKey = "token_type"
	// CSRFTokenKey is the context key for CSRF token
	CSRFTokenKey = "csrf_token"
)

// Cookie names for secure session management
const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
	CSRFTokenCookie    = "csrf_token"
)

// Security headers
const (
	CSRFTokenHeader = "X-CSRF-Token"
	CSRFHeaderName  = "X-Requested-With"
)

// Role constants for authorization
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// Token types
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	TokenTypeAPI     = "api"
)

// JWTClaims represents JWT claims structure
type JWTClaims struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret          []byte        `yaml:"jwt_secret"`
	JWTSecretFromFile  string        `yaml:"jwt_secret_file"`
	AccessTokenExpiry  time.Duration `yaml:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `yaml:"refresh_token_expiry"`
	APIKeyExpiry       time.Duration `yaml:"api_key_expiry"`
	EnableIPWhitelist  bool          `yaml:"enable_ip_whitelist"`
	AllowedIPs         []string      `yaml:"allowed_ips"`
	RateLimitPerMinute int           `yaml:"rate_limit_per_minute"`
	RequireHTTPS       bool          `yaml:"require_https"`
	EnableAuditLog     bool          `yaml:"enable_audit_log"`
	CookieDomain       string        `yaml:"cookie_domain"`
	CookiePath         string        `yaml:"cookie_path"`
	EnableCSRF         bool          `yaml:"enable_csrf"`
}

// DefaultAuthConfig returns environment-aware default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	env := os.Getenv("PI_CONTROLLER_ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}

	// Use secure production defaults unless explicitly set to development
	requireHTTPS := true
	if env == "development" || env == "dev" {
		requireHTTPS = false // Allow HTTP for development ease
	}

	return &AuthConfig{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		APIKeyExpiry:       90 * 24 * time.Hour,
		EnableIPWhitelist:  false,
		AllowedIPs:         []string{"127.0.0.1", "::1"},
		RateLimitPerMinute: 100,
		RequireHTTPS:       requireHTTPS,
		EnableAuditLog:     true,
		CookieDomain:       "",
		CookiePath:         "/",
		EnableCSRF:         true,
	}
}

// AuthManager handles JWT authentication and authorization
type AuthManager struct {
	config *AuthConfig
	logger logger.Interface
	secret []byte
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig, logger logger.Interface) (*AuthManager, error) {
	if config == nil {
		config = DefaultAuthConfig()
	}

	am := &AuthManager{
		config: config,
		logger: logger.WithField("component", "auth"),
	}

	// Load JWT secret
	if err := am.loadSecret(); err != nil {
		return nil, fmt.Errorf("failed to load JWT secret: %w", err)
	}

	am.logger.Info("Authentication manager initialized successfully")
	return am, nil
}

// GetConfig returns the authentication configuration
func (am *AuthManager) GetConfig() *AuthConfig {
	return am.config
}

// loadSecret loads the JWT secret from config or file, generates if missing
func (am *AuthManager) loadSecret() error {
	// Try to load from file first
	if am.config.JWTSecretFromFile != "" {
		data, err := os.ReadFile(am.config.JWTSecretFromFile)
		if err != nil {
			return fmt.Errorf("failed to read JWT secret file: %w", err)
		}
		am.secret = data
		am.logger.Info("JWT secret loaded from file")
		return nil
	}

	// Use provided secret
	if len(am.config.JWTSecret) > 0 {
		am.secret = am.config.JWTSecret
		am.logger.Info("JWT secret loaded from config")
		return nil
	}

	// Generate secure random secret
	secret := make([]byte, 64)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	am.secret = secret

	am.logger.Warn("Generated new JWT secret - tokens will be invalidated on restart")
	am.logger.Info("Consider setting JWT_SECRET environment variable or jwt_secret_file config")

	return nil
}

// GenerateToken generates a JWT token for the given user
func (am *AuthManager) GenerateToken(userID, role, tokenType string) (string, error) {
	now := time.Now().UTC()
	var expiry time.Duration

	switch tokenType {
	case TokenTypeAccess:
		expiry = am.config.AccessTokenExpiry
	case TokenTypeRefresh:
		expiry = am.config.RefreshTokenExpiry
	case TokenTypeAPI:
		expiry = am.config.APIKeyExpiry
	default:
		return "", fmt.Errorf("invalid token type: %s", tokenType)
	}

	claims := JWTClaims{
		UserID:    userID,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "pi-controller",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	if am.config.EnableAuditLog {
		am.logger.WithFields(map[string]interface{}{
			"user_id":    userID,
			"role":       role,
			"token_type": tokenType,
			"expires_at": now.Add(expiry),
		}).Info("JWT token generated")
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (am *AuthManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Parse token with validation
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Additional validation
	if claims.UserID == "" {
		return nil, errors.New("missing user ID in token")
	}

	if claims.Role == "" {
		return nil, errors.New("missing role in token")
	}

	// Validate role
	if !isValidRole(claims.Role) {
		return nil, fmt.Errorf("invalid role: %s", claims.Role)
	}

	return claims, nil
}

// SetSecureCookies sets authentication tokens as httpOnly secure cookies
func (am *AuthManager) SetSecureCookies(c *gin.Context, accessToken, refreshToken string) error {
	// Set access token cookie (httpOnly, secure, sameSite)
	accessCookie := &http.Cookie{
		Name:     AccessTokenCookie,
		Value:    accessToken,
		Path:     am.config.CookiePath,
		Domain:   am.config.CookieDomain,
		MaxAge:   int(am.config.AccessTokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   am.config.RequireHTTPS,
		SameSite: http.SameSiteStrictMode,
	}

	// Set refresh token cookie (httpOnly, secure, sameSite)
	refreshCookie := &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    refreshToken,
		Path:     am.config.CookiePath,
		Domain:   am.config.CookieDomain,
		MaxAge:   int(am.config.RefreshTokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   am.config.RequireHTTPS,
		SameSite: http.SameSiteStrictMode,
	}

	// Apply cookies to response
	http.SetCookie(c.Writer, accessCookie)
	http.SetCookie(c.Writer, refreshCookie)

	// Generate and set CSRF token if enabled
	if am.config.EnableCSRF {
		csrfToken, err := am.GenerateCSRFToken()
		if err != nil {
			return fmt.Errorf("failed to generate CSRF token: %w", err)
		}

		csrfCookie := &http.Cookie{
			Name:     CSRFTokenCookie,
			Value:    csrfToken,
			Path:     am.config.CookiePath,
			Domain:   am.config.CookieDomain,
			MaxAge:   int(am.config.AccessTokenExpiry.Seconds()),
			HttpOnly: false, // CSRF token needs to be readable by JS
			Secure:   am.config.RequireHTTPS,
			SameSite: http.SameSiteStrictMode,
		}

		http.SetCookie(c.Writer, csrfCookie)
		c.Set(CSRFTokenKey, csrfToken)
	}

	return nil
}

// ClearSecureCookies removes authentication cookies
func (am *AuthManager) ClearSecureCookies(c *gin.Context) {
	cookies := []string{AccessTokenCookie, RefreshTokenCookie, CSRFTokenCookie}

	for _, cookieName := range cookies {
		cookie := &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     am.config.CookiePath,
			Domain:   am.config.CookieDomain,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   am.config.RequireHTTPS,
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(c.Writer, cookie)
	}
}

// GetTokenFromCookie extracts JWT token from httpOnly cookie
func (am *AuthManager) GetTokenFromCookie(c *gin.Context, cookieName string) (string, error) {
	cookie, err := c.Request.Cookie(cookieName)
	if err != nil {
		return "", fmt.Errorf("cookie not found: %w", err)
	}

	if cookie.Value == "" {
		return "", errors.New("empty cookie value")
	}

	return cookie.Value, nil
}

// GenerateCSRFToken generates a secure CSRF token
func (am *AuthManager) GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateCSRFToken validates CSRF token from header against cookie
func (am *AuthManager) ValidateCSRFToken(c *gin.Context) bool {
	if !am.config.EnableCSRF {
		return true // CSRF protection disabled
	}

	// Get CSRF token from header
	headerToken := c.GetHeader(CSRFTokenHeader)
	if headerToken == "" {
		// Also check X-Requested-With header for AJAX requests
		if c.GetHeader(CSRFHeaderName) == "XMLHttpRequest" {
			return true
		}
		return false
	}

	// Get CSRF token from cookie
	cookieToken, err := c.Request.Cookie(CSRFTokenCookie)
	if err != nil || cookieToken.Value == "" {
		return false
	}

	// Perform constant-time comparison
	return SecureCompare(headerToken, cookieToken.Value)
}

// Auth provides JWT authentication middleware with support for both cookies and headers
func (am *AuthManager) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check HTTPS requirement
		if am.config.RequireHTTPS && c.Request.Header.Get("X-Forwarded-Proto") != "https" && c.Request.TLS == nil {
			am.auditLog(c, "auth_failure", "HTTPS required", "")
			c.JSON(http.StatusUpgradeRequired, gin.H{
				"error":   "HTTPS Required",
				"message": "This API requires HTTPS",
			})
			c.Abort()
			return
		}

		// Check IP whitelist if enabled
		if am.config.EnableIPWhitelist && !am.isIPAllowed(c.ClientIP()) {
			am.auditLog(c, "auth_failure", "IP not whitelisted", "")
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "IP address not allowed",
			})
			c.Abort()
			return
		}

		// Extract token from cookie first, then fallback to authorization header
		var tokenString string
		var tokenSource string

		// Try to get token from httpOnly cookie first (more secure)
		if cookieToken, err := am.GetTokenFromCookie(c, AccessTokenCookie); err == nil && cookieToken != "" {
			tokenString = cookieToken
			tokenSource = "cookie"
		} else {
			// Fallback to authorization header for API clients
			authHeader := c.GetHeader(AuthorizationHeader)
			if authHeader == "" {
				am.auditLog(c, "auth_failure", "Missing authorization header and cookie", "")
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Unauthorized",
					"message": "Authorization header is required",
				})
				c.Abort()
				return
			}

			// Validate Bearer token format
			if !strings.HasPrefix(authHeader, "Bearer ") {
				am.auditLog(c, "auth_failure", "Invalid authorization header format", "")
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Unauthorized",
					"message": "Invalid authorization header format. Use: Bearer <token>",
				})
				c.Abort()
				return
			}

			// Extract token
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			tokenSource = "header"
		}

		if tokenString == "" {
			am.auditLog(c, "auth_failure", "Empty token", "")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Token is required",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := am.ValidateToken(tokenString)
		if err != nil {
			am.auditLog(c, "auth_failure", fmt.Sprintf("Token validation failed: %v", err), "")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// For cookie-based auth, validate CSRF token for state-changing operations
		if tokenSource == "cookie" && am.isStateChangingRequest(c.Request.Method) {
			if !am.ValidateCSRFToken(c) {
				am.auditLog(c, "csrf_failure", "CSRF token validation failed", claims.UserID)
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Forbidden",
					"message": "CSRF token validation failed",
				})
				c.Abort()
				return
			}
		}

		// Set context values
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)
		c.Set(TokenTypeKey, claims.TokenType)

		am.auditLog(c, "auth_success", fmt.Sprintf("Authentication successful via %s", tokenSource), claims.UserID)
		c.Next()
	}
}

// isStateChangingRequest determines if the HTTP method can modify state
func (am *AuthManager) isStateChangingRequest(method string) bool {
	stateChangingMethods := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, m := range stateChangingMethods {
		if method == m {
			return true
		}
	}
	return false
}

// RequireRole creates a middleware that requires specific role
func (am *AuthManager) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(UserRoleKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Invalid role format",
			})
			c.Abort()
			return
		}

		if !am.hasPermission(role, requiredRole) {
			userID := GetUserID(c)
			am.auditLog(c, "authz_failure", fmt.Sprintf("Insufficient permissions: required %s, has %s", requiredRole, role), userID)
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": fmt.Sprintf("Requires %s role", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasPermission checks if user role has permission for required role
func (am *AuthManager) hasPermission(userRole, requiredRole string) bool {
	// Admin can access everything
	if userRole == RoleAdmin {
		return true
	}

	// Operator can access operator and viewer endpoints
	if userRole == RoleOperator && (requiredRole == RoleOperator || requiredRole == RoleViewer) {
		return true
	}

	// Viewer can only access viewer endpoints
	if userRole == RoleViewer && requiredRole == RoleViewer {
		return true
	}

	return false
}

// isIPAllowed checks if IP is in whitelist
func (am *AuthManager) isIPAllowed(clientIP string) bool {
	for _, allowedIP := range am.config.AllowedIPs {
		if clientIP == allowedIP {
			return true
		}
	}
	return false
}

// auditLog logs authentication events for security monitoring
func (am *AuthManager) auditLog(c *gin.Context, eventType, message, userID string) {
	if !am.config.EnableAuditLog {
		return
	}

	am.logger.WithFields(map[string]interface{}{
		"event_type": eventType,
		"message":    message,
		"user_id":    userID,
		"client_ip":  c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"request_id": c.GetHeader("X-Request-ID"),
	}).Info("Auth event")
}

// isValidRole validates user role
func isValidRole(role string) bool {
	validRoles := []string{RoleAdmin, RoleOperator, RoleViewer}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

// GetUserID returns the user ID from the gin context
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(UserIDKey); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

// GetUserRole returns the user role from the gin context
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get(UserRoleKey); exists {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

// GetTokenType returns the token type from the gin context
func GetTokenType(c *gin.Context) string {
	if tokenType, exists := c.Get(TokenTypeKey); exists {
		if tt, ok := tokenType.(string); ok {
			return tt
		}
	}
	return ""
}

// GenerateSecureAPIKey generates a secure API key for external integrations
func GenerateSecureAPIKey() (string, error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as hex with prefix
	return "pk_" + hex.EncodeToString(bytes), nil
}

// SecureCompare performs constant-time string comparison to prevent timing attacks
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
