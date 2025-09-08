package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/api"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// TestBasicRBAC tests the basic functionality of the RBAC system
func TestBasicRBAC(t *testing.T) {
	// Set development environment to disable HTTPS requirement
	t.Setenv("ENVIRONMENT", "development")

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

	t.Run("Default Admin Login", func(t *testing.T) {
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
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify the response has correct admin role
		user := response["user"].(map[string]interface{})
		assert.Equal(t, "admin", user["username"])
		assert.Equal(t, "admin", user["role"])

		// Verify tokens are present
		assert.NotEmpty(t, response["access_token"])
		assert.NotEmpty(t, response["refresh_token"])
	})

	t.Run("Access Control Test", func(t *testing.T) {
		// First, create a test user with viewer role
		registerPayload := map[string]interface{}{
			"username":   "viewer_test",
			"email":      "viewer@test.com",
			"password":   "viewer123",
			"role":       "viewer",
			"first_name": "Test",
			"last_name":  "Viewer",
		}

		// Add a small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)

		payloadBytes, _ := json.Marshal(registerPayload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Login as the viewer
		loginPayload := map[string]interface{}{
			"username": "viewer_test",
			"password": "viewer123",
		}

		time.Sleep(100 * time.Millisecond)

		payloadBytes, _ = json.Marshal(loginPayload)
		req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var loginResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)

		viewerToken := loginResponse["access_token"].(string)

		// Test that viewer can access read-only endpoints
		req, _ = http.NewRequest("GET", "/api/v1/clusters", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)

		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Test that viewer cannot access write endpoints
		clusterPayload := map[string]interface{}{
			"name":        "test-cluster",
			"description": "Test cluster",
		}

		payloadBytes, _ = json.Marshal(clusterPayload)
		req, _ = http.NewRequest("POST", "/api/v1/clusters", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+viewerToken)

		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Token Refresh", func(t *testing.T) {
		// Login with admin account
		loginPayload := map[string]interface{}{
			"username": "admin",
			"password": "admin123",
		}

		time.Sleep(100 * time.Millisecond)

		payloadBytes, _ := json.Marshal(loginPayload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var loginResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResponse)
		require.NoError(t, err)

		refreshToken := loginResponse["refresh_token"].(string)

		// Use refresh token to get new access token
		refreshPayload := map[string]interface{}{
			"refresh_token": refreshToken,
		}

		time.Sleep(100 * time.Millisecond)

		payloadBytes, _ = json.Marshal(refreshPayload)
		req, _ = http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &refreshResponse)
		require.NoError(t, err)

		newAccessToken := refreshResponse["access_token"].(string)
		assert.NotEmpty(t, newAccessToken)
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		// Test invalid login
		payload := map[string]interface{}{
			"username": "nonexistent",
			"password": "wrongpassword",
		}

		time.Sleep(100 * time.Millisecond)

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		// Test accessing protected endpoint without token
		req, _ := http.NewRequest("GET", "/api/v1/clusters", nil)

		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Test accessing protected endpoint with invalid token
		req, _ = http.NewRequest("GET", "/api/v1/clusters", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
