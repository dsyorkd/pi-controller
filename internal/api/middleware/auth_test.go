package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/logger"
)

func setupTestAuthManager() *AuthManager {
	config := &AuthConfig{
		JWTSecret:          []byte("test-secret-key-at-least-32-chars-long"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		APIKeyExpiry:       90 * 24 * time.Hour,
		RequireHTTPS:       false,
		EnableAuditLog:     true,
	}

	log := logger.Default()
	authManager, _ := NewAuthManager(config, log)
	return authManager
}

func setupTestRouter(authManager *AuthManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Public endpoint
	router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public"})
	})

	// Protected endpoints with different role requirements
	protected := router.Group("/api/v1")
	protected.Use(authManager.Auth())
	{
		protected.GET("/viewer", authManager.RequireRole(RoleViewer), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "viewer endpoint",
				"user_id": GetUserID(c),
				"role":    GetUserRole(c),
			})
		})

		protected.GET("/operator", authManager.RequireRole(RoleOperator), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "operator endpoint",
				"user_id": GetUserID(c),
				"role":    GetUserRole(c),
			})
		})

		protected.GET("/admin", authManager.RequireRole(RoleAdmin), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "admin endpoint",
				"user_id": GetUserID(c),
				"role":    GetUserRole(c),
			})
		})
	}

	return router
}

func TestAuthManager_GenerateAndValidateToken(t *testing.T) {
	authManager := setupTestAuthManager()

	tests := []struct {
		name      string
		userID    string
		role      string
		tokenType string
		wantErr   bool
	}{
		{
			name:      "valid access token for admin",
			userID:    "user123",
			role:      RoleAdmin,
			tokenType: TokenTypeAccess,
			wantErr:   false,
		},
		{
			name:      "valid access token for operator",
			userID:    "user456",
			role:      RoleOperator,
			tokenType: TokenTypeAccess,
			wantErr:   false,
		},
		{
			name:      "valid access token for viewer",
			userID:    "user789",
			role:      RoleViewer,
			tokenType: TokenTypeAccess,
			wantErr:   false,
		},
		{
			name:      "invalid token type",
			userID:    "user123",
			role:      RoleAdmin,
			tokenType: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := authManager.GenerateToken(tt.userID, tt.role, tt.tokenType)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate the generated token
			claims, err := authManager.ValidateToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.role, claims.Role)
			assert.Equal(t, tt.tokenType, claims.TokenType)
			assert.Equal(t, "pi-controller", claims.Issuer)
		})
	}
}

func TestAuthManager_ValidateToken_InvalidTokens(t *testing.T) {
	authManager := setupTestAuthManager()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "malformed.token.here",
		},
		{
			name:  "invalid signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcjEyMyIsInJvbGUiOiJhZG1pbiIsInRva2VuX3R5cGUiOiJhY2Nlc3MifQ.invalid_signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authManager.ValidateToken(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestAuth_NoAuthorizationHeader(t *testing.T) {
	authManager := setupTestAuthManager()
	router := setupTestRouter(authManager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["error"])
	assert.Equal(t, "Authorization header is required", response["message"])
}

func TestAuth_InvalidAuthorizationHeaderFormat(t *testing.T) {
	authManager := setupTestAuthManager()
	router := setupTestRouter(authManager)

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing Bearer prefix",
			header: "token123",
		},
		{
			name:   "wrong prefix",
			header: "Basic token123",
		},
		{
			name:   "empty token after Bearer",
			header: "Bearer ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
			req.Header.Set("Authorization", tt.header)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "Unauthorized", response["error"])
		})
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	authManager := setupTestAuthManager()
	router := setupTestRouter(authManager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["error"])
	assert.Equal(t, "Invalid or expired token", response["message"])
}

func TestAuth_ValidToken_AccessGranted(t *testing.T) {
	authManager := setupTestAuthManager()
	router := setupTestRouter(authManager)

	// Generate valid token
	token, err := authManager.GenerateToken("user123", RoleViewer, TokenTypeAccess)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "viewer endpoint", response["message"])
	assert.Equal(t, "user123", response["user_id"])
	assert.Equal(t, RoleViewer, response["role"])
}

func TestRBAC_RolePermissions(t *testing.T) {
	authManager := setupTestAuthManager()
	router := setupTestRouter(authManager)

	tests := []struct {
		name         string
		userRole     string
		endpoint     string
		expectedCode int
	}{
		// Viewer role tests
		{
			name:         "viewer can access viewer endpoint",
			userRole:     RoleViewer,
			endpoint:     "/api/v1/viewer",
			expectedCode: http.StatusOK,
		},
		{
			name:         "viewer cannot access operator endpoint",
			userRole:     RoleViewer,
			endpoint:     "/api/v1/operator",
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "viewer cannot access admin endpoint",
			userRole:     RoleViewer,
			endpoint:     "/api/v1/admin",
			expectedCode: http.StatusForbidden,
		},
		// Operator role tests
		{
			name:         "operator can access viewer endpoint",
			userRole:     RoleOperator,
			endpoint:     "/api/v1/viewer",
			expectedCode: http.StatusOK,
		},
		{
			name:         "operator can access operator endpoint",
			userRole:     RoleOperator,
			endpoint:     "/api/v1/operator",
			expectedCode: http.StatusOK,
		},
		{
			name:         "operator cannot access admin endpoint",
			userRole:     RoleOperator,
			endpoint:     "/api/v1/admin",
			expectedCode: http.StatusForbidden,
		},
		// Admin role tests
		{
			name:         "admin can access viewer endpoint",
			userRole:     RoleAdmin,
			endpoint:     "/api/v1/viewer",
			expectedCode: http.StatusOK,
		},
		{
			name:         "admin can access operator endpoint",
			userRole:     RoleAdmin,
			endpoint:     "/api/v1/operator",
			expectedCode: http.StatusOK,
		},
		{
			name:         "admin can access admin endpoint",
			userRole:     RoleAdmin,
			endpoint:     "/api/v1/admin",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate valid token for the user role
			token, err := authManager.GenerateToken("testuser", tt.userRole, TokenTypeAccess)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedCode == http.StatusForbidden {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "Forbidden", response["error"])
				assert.Contains(t, response["message"], "Requires")
			}
		})
	}
}

func TestAuth_ExpiredToken(t *testing.T) {
	// Create auth manager with very short token expiry
	config := &AuthConfig{
		JWTSecret:         []byte("test-secret-key-at-least-32-chars-long"),
		AccessTokenExpiry: 1 * time.Millisecond,
		RequireHTTPS:      false,
		EnableAuditLog:    true,
	}

	log := logger.Default()
	authManager, err := NewAuthManager(config, log)
	require.NoError(t, err)

	router := setupTestRouter(authManager)

	// Generate token
	token, err := authManager.GenerateToken("user123", RoleViewer, TokenTypeAccess)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", response["error"])
	assert.Equal(t, "Invalid or expired token", response["message"])
}

func TestAuth_PublicEndpoint_NoAuthRequired(t *testing.T) {
	authManager := setupTestAuthManager()
	router := setupTestRouter(authManager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/public", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "public", response["message"])
}

func TestAuth_HTTPSRequired(t *testing.T) {
	// Create auth manager with HTTPS requirement
	config := &AuthConfig{
		JWTSecret:         []byte("test-secret-key-at-least-32-chars-long"),
		AccessTokenExpiry: 15 * time.Minute,
		RequireHTTPS:      true,
		EnableAuditLog:    true,
	}

	log := logger.Default()
	authManager, err := NewAuthManager(config, log)
	require.NoError(t, err)

	router := setupTestRouter(authManager)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUpgradeRequired, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "HTTPS Required", response["error"])
	assert.Equal(t, "This API requires HTTPS", response["message"])
}

func TestAuth_IPWhitelist(t *testing.T) {
	// Create auth manager with IP whitelist
	config := &AuthConfig{
		JWTSecret:         []byte("test-secret-key-at-least-32-chars-long"),
		AccessTokenExpiry: 15 * time.Minute,
		EnableIPWhitelist: true,
		AllowedIPs:        []string{"192.168.1.100"},
		RequireHTTPS:      false,
		EnableAuditLog:    true,
	}

	log := logger.Default()
	authManager, err := NewAuthManager(config, log)
	require.NoError(t, err)

	router := setupTestRouter(authManager)

	// Test with IP not in whitelist (default test IP is usually 192.0.2.1)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/viewer", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Forbidden", response["error"])
	assert.Equal(t, "IP address not allowed", response["message"])
}

func TestGenerateSecureAPIKey(t *testing.T) {
	apiKey, err := GenerateSecureAPIKey()
	require.NoError(t, err)

	assert.NotEmpty(t, apiKey)
	assert.True(t, strings.HasPrefix(apiKey, "pk_"))
	assert.Equal(t, 67, len(apiKey)) // 3 (prefix) + 64 (hex encoded 32 bytes)
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name   string
		a      string
		b      string
		expect bool
	}{
		{
			name:   "identical strings",
			a:      "hello",
			b:      "hello",
			expect: true,
		},
		{
			name:   "different strings",
			a:      "hello",
			b:      "world",
			expect: false,
		},
		{
			name:   "empty strings",
			a:      "",
			b:      "",
			expect: true,
		},
		{
			name:   "different lengths",
			a:      "hello",
			b:      "hello world",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecureCompare(tt.a, tt.b)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestContextHelpers(t *testing.T) {
	// Create a test context with auth data
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Test empty context
	assert.Empty(t, GetUserID(c))
	assert.Empty(t, GetUserRole(c))
	assert.Empty(t, GetTokenType(c))

	// Set context values
	c.Set(UserIDKey, "user123")
	c.Set(UserRoleKey, RoleAdmin)
	c.Set(TokenTypeKey, TokenTypeAccess)

	// Test with values
	assert.Equal(t, "user123", GetUserID(c))
	assert.Equal(t, RoleAdmin, GetUserRole(c))
	assert.Equal(t, TokenTypeAccess, GetTokenType(c))
}

func TestValidRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{RoleAdmin, true},
		{RoleOperator, true},
		{RoleViewer, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := isValidRole(tt.role)
			assert.Equal(t, tt.valid, result)
		})
	}
}
