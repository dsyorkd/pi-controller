package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/api"
	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

func TestBasicSecurity(t *testing.T) {
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

	// Create secure server configuration
	apiConfig := &config.APIConfig{
		Host:         "localhost",
		Port:         8080,
		AuthEnabled:  true,
		CORSEnabled:  false,
		ReadTimeout:  "30s",
		WriteTimeout: "30s",
	}

	// Create server with security enabled
	server := api.New(apiConfig, testLogger, db)

	t.Run("Authentication Required", func(t *testing.T) {
		// Test that API requires authentication
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/clusters", nil)

		server.Router().ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
		assert.Equal(t, "Unauthorized", response["error"])
	})

	t.Run("Input Validation", func(t *testing.T) {
		// Test SQL injection prevention
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/clusters?limit='; DROP TABLE clusters; --", nil)
		req.Header.Set("Authorization", "Bearer fake-token")

		server.Router().ServeHTTP(w, req)

		// Should be blocked by input validation, not reach auth
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
		assert.Contains(t, response["message"], "malicious")
	})

	t.Run("GPIO Pin Validation", func(t *testing.T) {
		validator := middleware.NewValidator(middleware.DefaultValidationConfig(), testLogger)

		// Test system critical pins are blocked
		err := validator.ValidateGPIOPin(0) // I2C SDA
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "system-critical")

		err = validator.ValidateGPIOPin(1) // I2C SCL
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "system-critical")

		err = validator.ValidateGPIOPin(14) // UART TX
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "system-critical")

		err = validator.ValidateGPIOPin(15) // UART RX
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "system-critical")

		// Test valid pins work
		err = validator.ValidateGPIOPin(18)
		assert.NoError(t, err)
	})

	t.Run("Name Validation", func(t *testing.T) {
		validator := middleware.NewValidator(middleware.DefaultValidationConfig(), testLogger)

		// Test SQL injection in names
		err := validator.ValidateName("test'; DROP TABLE users; --", "cluster")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "malicious")

		// Test XSS in names
		err = validator.ValidateName("test<script>alert('xss')</script>", "cluster")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid characters")

		// Test valid names
		err = validator.ValidateName("test-cluster-01", "cluster")
		assert.NoError(t, err)
	})
}

func TestJWTAuthentication(t *testing.T) {
	testLogger := logger.Default()

	// Create auth manager with test secret
	authConfig := middleware.DefaultAuthConfig()
	authConfig.JWTSecret = []byte("test-secret-key-32-bytes-long!!!")

	authManager, err := middleware.NewAuthManager(authConfig, testLogger)
	require.NoError(t, err)

	t.Run("Generate and Validate Token", func(t *testing.T) {
		// Generate token
		token, err := authManager.GenerateToken("test-user", "admin", "access")
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate token
		claims, err := authManager.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "test-user", claims.UserID)
		assert.Equal(t, "admin", claims.Role)
		assert.Equal(t, "access", claims.TokenType)
	})

	t.Run("Invalid Token Rejected", func(t *testing.T) {
		// Test invalid token
		_, err := authManager.ValidateToken("invalid-token")
		assert.Error(t, err)

		// Test empty token
		_, err = authManager.ValidateToken("")
		assert.Error(t, err)
	})
}

func TestClusterCreationWithSecurity(t *testing.T) {
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

	// Create server with auth disabled for this test
	apiConfig := &config.APIConfig{
		Host:         "localhost",
		Port:         8080,
		AuthEnabled:  false, // Disable auth to test input validation only
		CORSEnabled:  false,
		ReadTimeout:  "30s",
		WriteTimeout: "30s",
	}

	server := api.New(apiConfig, testLogger, db)

	t.Run("Malicious Cluster Name Blocked", func(t *testing.T) {
		maliciousPayload := map[string]interface{}{
			"name":        "test'; DROP TABLE clusters; --",
			"description": "Test cluster",
		}

		payload, _ := json.Marshal(maliciousPayload)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/clusters", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")

		server.Router().ServeHTTP(w, req)

		// Should be blocked by input validation
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "error")
	})
}
