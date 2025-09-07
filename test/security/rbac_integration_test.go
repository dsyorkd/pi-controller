package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/api"
	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// TestRBACIntegration tests the complete Role-Based Access Control system
func TestRBACIntegration(t *testing.T) {
	// Create test database
	dbConfig := &storage.Config{
		Path:            ":memory:",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "5m",
		LogLevel:        "error",
	}

	testLogger := logger.Default()
	db, err := storage.New(dbConfig, testLogger)
	require.NoError(t, err)
	defer db.Close()

	// Create server with authentication enabled
	apiConfig := &config.APIConfig{
		Host:         "localhost",
		Port:         8080,
		AuthEnabled:  true,
		CORSEnabled:  false,
		ReadTimeout:  "30s",
		WriteTimeout: "30s",
	}

	server := api.New(apiConfig, testLogger, db, nil)

	// Test data
	testUsers := []struct {
		username string
		email    string
		password string
		role     string
	}{
		{"admin_user", "admin@test.com", "admin123", "admin"},
		{"operator_user", "operator@test.com", "operator123", "operator"},
		{"viewer_user", "viewer@test.com", "viewer123", "viewer"},
	}

	// Create test users
	for _, user := range testUsers {
		registerUser(t, server, user.username, user.email, user.password, user.role)
	}

	// Test authentication flows
	t.Run("Authentication Flow", func(t *testing.T) {
		testAuthenticationFlow(t, server, testUsers)
	})

	// Test authorization with different roles
	t.Run("Role-Based Authorization", func(t *testing.T) {
		testRoleBasedAuthorization(t, server, testUsers)
	})

	// Test token refresh functionality
	t.Run("Token Refresh", func(t *testing.T) {
		testTokenRefresh(t, server, testUsers[0])
	})

	// Test account security features
	t.Run("Account Security", func(t *testing.T) {
		testAccountSecurity(t, server)
	})
}

func registerUser(t *testing.T, server *api.Server, username, email, password, role string) {
	payload := map[string]interface{}{
		"username":   username,
		"email":      email,
		"password":   password,
		"role":       role,
		"first_name": "Test",
		"last_name":  "User",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Failed to register user %s", username)
}

func loginUser(t *testing.T, server *api.Server, username, password string) (accessToken, refreshToken string) {
	payload := map[string]interface{}{
		"username": username,
		"password": password,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "Login failed for user %s", username)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	accessToken = response["access_token"].(string)
	refreshToken = response["refresh_token"].(string)
	
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	return accessToken, refreshToken
}

func testAuthenticationFlow(t *testing.T, server *api.Server, testUsers []struct {
	username string
	email    string
	password string
	role     string
}) {
	for _, user := range testUsers {
		t.Run(fmt.Sprintf("Login_%s", user.role), func(t *testing.T) {
			// Test successful login
			accessToken, refreshToken := loginUser(t, server, user.username, user.password)

			// Test accessing profile with valid token
			req, _ := http.NewRequest("GET", "/api/v1/auth/profile", nil)
			req.Header.Set("Authorization", "Bearer "+accessToken)

			w := httptest.NewRecorder()
			server.Router().ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var profile map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &profile)
			require.NoError(t, err)
			
			assert.Equal(t, user.username, profile["username"])
			assert.Equal(t, user.email, profile["email"])
			assert.Equal(t, user.role, profile["role"])

			// Test logout
			req, _ = http.NewRequest("POST", "/api/v1/auth/logout", nil)
			req.Header.Set("Authorization", "Bearer "+accessToken)

			w = httptest.NewRecorder()
			server.Router().ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			_ = refreshToken // Use refresh token to avoid unused variable error
		})
	}

	// Test invalid login
	t.Run("Invalid_Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "nonexistent",
			"password": "wrongpassword",
		}

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func testRoleBasedAuthorization(t *testing.T, server *api.Server, testUsers []struct {
	username string
	email    string
	password string
	role     string
}) {
	// Get tokens for each user
	tokens := make(map[string]string)
	for _, user := range testUsers {
		accessToken, _ := loginUser(t, server, user.username, user.password)
		tokens[user.role] = accessToken
	}

	// Test endpoints with different role requirements
	testCases := []struct {
		method         string
		endpoint       string
		requiredRole   string
		expectedStatus map[string]int // role -> expected status
		payload        map[string]interface{}
	}{
		{
			method:       "GET",
			endpoint:     "/api/v1/clusters",
			requiredRole: "viewer",
			expectedStatus: map[string]int{
				"admin":    http.StatusOK,
				"operator": http.StatusOK,
				"viewer":   http.StatusOK,
			},
		},
		{
			method:       "POST",
			endpoint:     "/api/v1/clusters",
			requiredRole: "operator",
			payload:      map[string]interface{}{"name": "test-cluster", "description": "Test cluster"},
			expectedStatus: map[string]int{
				"admin":    http.StatusOK,
				"operator": http.StatusOK,
				"viewer":   http.StatusForbidden,
			},
		},
		{
			method:       "DELETE",
			endpoint:     "/api/v1/clusters/1",
			requiredRole: "admin",
			expectedStatus: map[string]int{
				"admin":    http.StatusNotFound, // Cluster doesn't exist, but auth passes
				"operator": http.StatusForbidden,
				"viewer":   http.StatusForbidden,
			},
		},
		{
			method:       "GET",
			endpoint:     "/api/v1/nodes",
			requiredRole: "viewer",
			expectedStatus: map[string]int{
				"admin":    http.StatusOK,
				"operator": http.StatusOK,
				"viewer":   http.StatusOK,
			},
		},
		{
			method:       "POST",
			endpoint:     "/api/v1/nodes",
			requiredRole: "operator",
			payload:      map[string]interface{}{"name": "test-node", "ip_address": "192.168.1.100"},
			expectedStatus: map[string]int{
				"admin":    http.StatusOK,
				"operator": http.StatusOK,
				"viewer":   http.StatusForbidden,
			},
		},
		{
			method:       "DELETE",
			endpoint:     "/api/v1/nodes/1",
			requiredRole: "admin",
			expectedStatus: map[string]int{
				"admin":    http.StatusNotFound, // Node doesn't exist, but auth passes
				"operator": http.StatusForbidden,
				"viewer":   http.StatusForbidden,
			},
		},
		{
			method:       "GET",
			endpoint:     "/api/v1/gpio",
			requiredRole: "viewer",
			expectedStatus: map[string]int{
				"admin":    http.StatusOK,
				"operator": http.StatusOK,
				"viewer":   http.StatusOK,
			},
		},
		{
			method:       "POST",
			endpoint:     "/api/v1/gpio",
			requiredRole: "operator",
			payload:      map[string]interface{}{"name": "test-pin", "pin_number": 18, "node_id": 1},
			expectedStatus: map[string]int{
				"admin":    http.StatusBadRequest, // Node doesn't exist, but auth passes
				"operator": http.StatusBadRequest,
				"viewer":   http.StatusForbidden,
			},
		},
	}

	for _, tc := range testCases {
		for role, expectedStatus := range tc.expectedStatus {
			t.Run(fmt.Sprintf("%s_%s_as_%s", tc.method, tc.endpoint, role), func(t *testing.T) {
				var req *http.Request
				if tc.payload != nil {
					payloadBytes, _ := json.Marshal(tc.payload)
					req, _ = http.NewRequest(tc.method, tc.endpoint, bytes.NewBuffer(payloadBytes))
					req.Header.Set("Content-Type", "application/json")
				} else {
					req, _ = http.NewRequest(tc.method, tc.endpoint, nil)
				}

				req.Header.Set("Authorization", "Bearer "+tokens[role])

				w := httptest.NewRecorder()
				server.Router().ServeHTTP(w, req)

				assert.Equal(t, expectedStatus, w.Code,
					"Unexpected status for %s %s with role %s", tc.method, tc.endpoint, role)
			})
		}
	}

	// Test accessing endpoints without authentication
	t.Run("No_Authentication", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/clusters", nil)
		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test invalid token
	t.Run("Invalid_Token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/clusters", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func testTokenRefresh(t *testing.T, server *api.Server, user struct {
	username string
	email    string
	password string
	role     string
}) {
	// Login and get tokens
	_, refreshToken := loginUser(t, server, user.username, user.password)

	// Use refresh token to get new access token
	payload := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	newAccessToken := response["access_token"].(string)
	assert.NotEmpty(t, newAccessToken)

	// Test that new access token works
	req, _ = http.NewRequest("GET", "/api/v1/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+newAccessToken)

	w = httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test invalid refresh token
	payload = map[string]interface{}{
		"refresh_token": "invalid-refresh-token",
	}

	payloadBytes, _ = json.Marshal(payload)
	req, _ = http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func testAccountSecurity(t *testing.T, server *api.Server) {
	// Test account lockout after failed attempts
	t.Run("Account_Lockout", func(t *testing.T) {
		// Create a test user first
		registerUser(t, server, "lockout_test", "lockout@test.com", "password123", "viewer")

		// Make multiple failed login attempts
		for i := 0; i < 6; i++ {
			payload := map[string]interface{}{
				"username": "lockout_test",
				"password": "wrongpassword",
			}

			payloadBytes, _ := json.Marshal(payload)
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.Router().ServeHTTP(w, req)

			if i < 5 {
				assert.Equal(t, http.StatusUnauthorized, w.Code)
			} else {
				// After 5 failed attempts, account should be locked
				assert.Equal(t, http.StatusTooManyRequests, w.Code)
			}
		}

		// Even with correct password, account should be locked
		payload := map[string]interface{}{
			"username": "lockout_test",
			"password": "password123",
		}

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	// Test password validation
	t.Run("Password_Validation", func(t *testing.T) {
		// Test weak password
		payload := map[string]interface{}{
			"username": "weak_pass_user",
			"email":    "weak@test.com",
			"password": "123",
			"role":     "viewer",
		}

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test email validation
	t.Run("Email_Validation", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "invalid_email_user",
			"email":    "invalid-email",
			"password": "validpassword123",
			"role":     "viewer",
		}

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test duplicate username/email prevention
	t.Run("Duplicate_Prevention", func(t *testing.T) {
		// Register first user
		registerUser(t, server, "duplicate_test", "duplicate@test.com", "password123", "viewer")

		// Try to register with same username
		payload := map[string]interface{}{
			"username": "duplicate_test",
			"email":    "different@test.com",
			"password": "password123",
			"role":     "viewer",
		}

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		// Try to register with same email
		payload = map[string]interface{}{
			"username": "different_user",
			"email":    "duplicate@test.com",
			"password": "password123",
			"role":     "viewer",
		}

		payloadBytes, _ = json.Marshal(payload)
		req, _ = http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// TestDefaultAdminUser tests that the default admin user is created and works
func TestDefaultAdminUser(t *testing.T) {
	// Create test database
	dbConfig := &storage.Config{
		Path:            ":memory:",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "5m",
		LogLevel:        "error",
	}

	testLogger := logger.Default()
	db, err := storage.New(dbConfig, testLogger)
	require.NoError(t, err)
	defer db.Close()

	// Create server with authentication enabled
	apiConfig := &config.APIConfig{
		Host:         "localhost",
		Port:         8080,
		AuthEnabled:  true,
		CORSEnabled:  false,
		ReadTimeout:  "30s",
		WriteTimeout: "30s",
	}

	server := api.New(apiConfig, testLogger, db, nil)

	// Test default admin login
	payload := map[string]interface{}{
		"username": "admin",
		"password": "admin123",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify the response has correct admin role
	user := response["user"].(map[string]interface{})
	assert.Equal(t, "admin", user["username"])
	assert.Equal(t, "admin", user["role"])
}

// TestTokenExpiration tests JWT token expiration handling
func TestTokenExpiration(t *testing.T) {
	testLogger := logger.Default()

	// Create auth manager with very short token expiry for testing
	authConfig := middleware.DefaultAuthConfig()
	authConfig.JWTSecret = []byte("test-secret-key-32-bytes-long!!!")
	authConfig.AccessTokenExpiry = 1 * time.Millisecond // Very short expiry

	authManager, err := middleware.NewAuthManager(authConfig, testLogger)
	require.NoError(t, err)

	// Generate token
	token, err := authManager.GenerateToken("1", "admin", middleware.TokenTypeAccess)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired token
	_, err = authManager.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}