package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateFromRequest(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret:         []byte("test-secret-key-for-testing-purposes"),
		AccessTokenExpiry: 15 * time.Minute,
	}, testLogger)
	require.NoError(t, err)

	// Generate a valid token
	token, err := authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(t, err)

	tests := []struct {
		name           string
		authConfig     *AuthConfig
		queryToken     string
		headerToken    string
		expectUserID   string
		expectRole     string
		expectError    bool
		allowAnonymous bool
	}{
		{
			name:         "Auth disabled returns empty",
			authConfig:   nil,
			expectUserID: "",
			expectRole:   "",
			expectError:  false,
		},
		{
			name: "Valid token from query param",
			authConfig: &AuthConfig{
				Enabled:     true,
				AuthManager: authManager,
				Logger:      testLogger,
			},
			queryToken:   token,
			expectUserID: "test-user",
			expectRole:   middleware.RoleAdmin,
			expectError:  false,
		},
		{
			name: "Valid token from Authorization header",
			authConfig: &AuthConfig{
				Enabled:     true,
				AuthManager: authManager,
				Logger:      testLogger,
			},
			headerToken:  "Bearer " + token,
			expectUserID: "test-user",
			expectRole:   middleware.RoleAdmin,
			expectError:  false,
		},
		{
			name: "Missing token with AllowAnonymous",
			authConfig: &AuthConfig{
				Enabled:        true,
				AuthManager:    authManager,
				AllowAnonymous: true,
				Logger:         testLogger,
			},
			expectUserID: "anonymous",
			expectRole:   "viewer",
			expectError:  false,
		},
		{
			name: "Missing token without AllowAnonymous",
			authConfig: &AuthConfig{
				Enabled:        true,
				AuthManager:    authManager,
				AllowAnonymous: false,
				Logger:         testLogger,
			},
			expectError: true,
		},
		{
			name: "Invalid token",
			authConfig: &AuthConfig{
				Enabled:     true,
				AuthManager: authManager,
				Logger:      testLogger,
			},
			queryToken:  "invalid.token.here",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.WebSocketConfig{
				Host:            "localhost",
				Port:            8181,
				Path:            "/ws",
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
			}

			server := NewWithAuth(cfg, testLogger, nil, tt.authConfig)

			// Create request
			url := "http://localhost:8181/ws"
			if tt.queryToken != "" {
				url += "?token=" + tt.queryToken
			}
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			if tt.headerToken != "" {
				req.Header.Set("Authorization", tt.headerToken)
			}

			userID, role, err := server.authenticateFromRequest(req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectUserID, userID)
				assert.Equal(t, tt.expectRole, role)
			}
		})
	}
}

func TestClientHasRole(t *testing.T) {
	client := &Client{
		authenticated: true,
		role:          middleware.RoleOperator,
	}

	// Operator can access viewer endpoints
	assert.True(t, client.hasRole(middleware.RoleViewer))

	// Operator can access operator endpoints
	assert.True(t, client.hasRole(middleware.RoleOperator))

	// Operator cannot access admin endpoints
	assert.False(t, client.hasRole(middleware.RoleAdmin))

	// Unauthenticated client
	unauthClient := &Client{
		authenticated: false,
		role:          "",
	}
	assert.False(t, unauthClient.hasRole(middleware.RoleViewer))
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		userRole     string
		requiredRole string
		expected     bool
	}{
		{middleware.RoleAdmin, middleware.RoleAdmin, true},
		{middleware.RoleAdmin, middleware.RoleOperator, true},
		{middleware.RoleAdmin, middleware.RoleViewer, true},
		{middleware.RoleOperator, middleware.RoleAdmin, false},
		{middleware.RoleOperator, middleware.RoleOperator, true},
		{middleware.RoleOperator, middleware.RoleViewer, true},
		{middleware.RoleViewer, middleware.RoleAdmin, false},
		{middleware.RoleViewer, middleware.RoleOperator, false},
		{middleware.RoleViewer, middleware.RoleViewer, true},
	}

	for _, tt := range tests {
		t.Run(tt.userRole+"->"+tt.requiredRole, func(t *testing.T) {
			result := hasPermission(tt.userRole, tt.requiredRole)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{
		Code:    401,
		Message: "Authentication required",
	}

	assert.Equal(t, "Authentication required", err.Error())
	assert.Equal(t, ErrNoAuthToken.Error(), "Authentication token required")
}

func TestWebSocketAuthEndToEnd(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret:         []byte("test-secret-key-for-testing-purposes"),
		AccessTokenExpiry: 15 * time.Minute,
	}, testLogger)
	require.NoError(t, err)

	// Generate a valid token
	token, err := authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(t, err)

	cfg := &config.WebSocketConfig{
		Host:            "localhost",
		Port:            0, // Random port
		Path:            "/ws",
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	authConfig := &AuthConfig{
		Enabled:        true,
		AuthManager:    authManager,
		AllowAnonymous: false,
		Logger:         testLogger,
	}

	server := NewWithAuth(cfg, testLogger, nil, authConfig)

	// Start the hub in background
	go server.run()
	defer close(server.shutdown)

	// Create test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("Authenticated connection succeeds", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws?token=" + token
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer ws.Close()

		// Should receive welcome pong message
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := ws.ReadMessage()
		require.NoError(t, err)

		var message Message
		err = json.Unmarshal(msg, &message)
		require.NoError(t, err)
		assert.Equal(t, MessageTypePong, message.Type)
	})

	t.Run("Unauthenticated connection fails", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	})
}

func TestWebSocketAuthMessage(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret:         []byte("test-secret-key-for-testing-purposes"),
		AccessTokenExpiry: 15 * time.Minute,
	}, testLogger)
	require.NoError(t, err)

	// Generate a valid token
	token, err := authManager.GenerateToken("test-user", middleware.RoleOperator, middleware.TokenTypeAccess)
	require.NoError(t, err)

	cfg := &config.WebSocketConfig{
		Host:            "localhost",
		Port:            0,
		Path:            "/ws",
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	authConfig := &AuthConfig{
		Enabled:        true,
		AuthManager:    authManager,
		AllowAnonymous: true, // Allow anonymous to connect, but require auth for subscriptions
		Logger:         testLogger,
	}

	server := NewWithAuth(cfg, testLogger, nil, authConfig)

	// Start the hub in background
	go server.run()
	defer close(server.shutdown)

	// Create test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("Auth via message after connect", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer ws.Close()

		// Skip welcome message
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, _ = ws.ReadMessage()

		// Send auth message
		authPayload, _ := json.Marshal(AuthMessage{Token: token})
		authMsg := Message{
			Type:      MessageTypeAuth,
			Payload:   authPayload,
			Timestamp: time.Now(),
		}
		msgBytes, _ := json.Marshal(authMsg)
		err = ws.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)

		// Read auth response
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, respBytes, err := ws.ReadMessage()
		require.NoError(t, err)

		var respMsg Message
		err = json.Unmarshal(respBytes, &respMsg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeAuth, respMsg.Type)

		var authResp AuthResponseMessage
		err = json.Unmarshal(respMsg.Payload, &authResp)
		require.NoError(t, err)
		assert.True(t, authResp.Success)
		assert.Equal(t, "test-user", authResp.UserID)
		assert.Equal(t, middleware.RoleOperator, authResp.Role)
	})
}
