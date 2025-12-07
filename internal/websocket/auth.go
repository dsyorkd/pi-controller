package websocket

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
)

// AuthConfig holds WebSocket authentication configuration
type AuthConfig struct {
	// Enabled determines if authentication is required
	Enabled bool
	// AuthManager is the JWT authentication manager
	AuthManager *middleware.AuthManager
	// AllowAnonymous allows unauthenticated connections with limited access
	AllowAnonymous bool
	// Logger for auth events
	Logger logger.Interface
}

// AuthenticatedClient extends Client with authentication info
type AuthenticatedClient struct {
	*Client
	UserID        string
	Role          string
	Authenticated bool
}

// MessageTypeAuth is the authentication message type
const MessageTypeAuth MessageType = "auth"

// AuthMessage represents an authentication request
type AuthMessage struct {
	Token string `json:"token"`
}

// AuthResponseMessage represents an authentication response
type AuthResponseMessage struct {
	Success bool   `json:"success"`
	UserID  string `json:"user_id,omitempty"`
	Role    string `json:"role,omitempty"`
	Message string `json:"message,omitempty"`
}

// authenticateFromRequest authenticates a WebSocket connection from the HTTP request
func (s *Server) authenticateFromRequest(r *http.Request) (userID, role string, err error) {
	if s.authConfig == nil || !s.authConfig.Enabled {
		return "", "", nil
	}

	// Try to get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		if s.authConfig.AllowAnonymous {
			return "anonymous", "viewer", nil
		}
		return "", "", ErrNoAuthToken
	}

	// Validate token
	claims, err := s.authConfig.AuthManager.ValidateToken(token)
	if err != nil {
		return "", "", err
	}

	return claims.UserID, claims.Role, nil
}

// handleAuthMessage handles authentication messages from connected clients
func (c *Client) handleAuthMessage(msg Message) bool {
	if c.server.authConfig == nil || !c.server.authConfig.Enabled {
		c.authenticated = true
		return true
	}

	var authMsg AuthMessage
	if err := json.Unmarshal(msg.Payload, &authMsg); err != nil {
		c.sendAuthResponse(false, "", "", "Invalid auth message format")
		return false
	}

	if authMsg.Token == "" {
		c.sendAuthResponse(false, "", "", "Token is required")
		return false
	}

	// Validate token
	claims, err := c.server.authConfig.AuthManager.ValidateToken(authMsg.Token)
	if err != nil {
		c.server.logAuthFailure(c.id, err)
		c.sendAuthResponse(false, "", "", "Invalid or expired token")
		return false
	}

	c.userID = claims.UserID
	c.role = claims.Role
	c.authenticated = true

	c.server.logAuthSuccess(c.id, c.userID, c.role)
	c.sendAuthResponse(true, c.userID, c.role, "Authentication successful")
	return true
}

// sendAuthResponse sends an authentication response to the client
func (c *Client) sendAuthResponse(success bool, userID, role, message string) {
	authResp := AuthResponseMessage{
		Success: success,
		UserID:  userID,
		Role:    role,
		Message: message,
	}
	payload, _ := json.Marshal(authResp)

	msg := Message{
		Type:      MessageTypeAuth,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	c.server.sendToClient(c, msg)
}

// logAuthFailure logs WebSocket authentication failures
func (s *Server) logAuthFailure(clientID string, err error) {
	if s.authConfig == nil || s.authConfig.Logger == nil {
		return
	}
	s.authConfig.Logger.WithFields(map[string]interface{}{
		"client_id": clientID,
		"error":     err.Error(),
	}).Warn("WebSocket authentication failed")
}

// logAuthSuccess logs successful WebSocket authentication
func (s *Server) logAuthSuccess(clientID, userID, role string) {
	if s.authConfig == nil || s.authConfig.Logger == nil {
		return
	}
	s.authConfig.Logger.WithFields(map[string]interface{}{
		"client_id": clientID,
		"user_id":   userID,
		"role":      role,
	}).Info("WebSocket authentication successful")
}

// hasRole checks if the client has the required role
func (c *Client) hasRole(requiredRole string) bool {
	if !c.authenticated {
		return false
	}
	return hasPermission(c.role, requiredRole)
}

// hasPermission checks if user role has permission for required role
func hasPermission(userRole, requiredRole string) bool {
	// Admin can access everything
	if userRole == middleware.RoleAdmin {
		return true
	}

	// Operator can access operator and viewer endpoints
	if userRole == middleware.RoleOperator && (requiredRole == middleware.RoleOperator || requiredRole == middleware.RoleViewer) {
		return true
	}

	// Viewer can only access viewer endpoints
	if userRole == middleware.RoleViewer && requiredRole == middleware.RoleViewer {
		return true
	}

	return false
}

// ErrNoAuthToken is returned when no authentication token is provided
var ErrNoAuthToken = &AuthError{Code: 401, Message: "Authentication token required"}

// AuthError represents an authentication error
type AuthError struct {
	Code    int
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
