package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// AuthHandler handles authentication and user management endpoints
type AuthHandler struct {
	database    *storage.Database
	logger      logger.Interface
	authManager *middleware.AuthManager
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db *storage.Database, authManager *middleware.AuthManager, logger logger.Interface) *AuthHandler {
	return &AuthHandler{
		database:    db,
		logger:      logger.WithField("component", "auth_handler"),
		authManager: authManager,
	}
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required" validate:"min=3,max=50"`
	Password string `json:"password" binding:"required" validate:"min=6,max=100"`
}

// RegisterRequest represents the user registration request payload
type RegisterRequest struct {
	Username  string `json:"username" binding:"required" validate:"min=3,max=50"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required" validate:"min=6,max=100"`
	FirstName string `json:"first_name" validate:"max=50"`
	LastName  string `json:"last_name" validate:"max=50"`
	Role      string `json:"role" validate:"oneof=viewer operator admin"`
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

// UserInfo represents safe user information for API responses
type UserInfo struct {
	ID        uint              `json:"id"`
	Username  string            `json:"username"`
	Email     string            `json:"email"`
	Role      models.UserRole   `json:"role"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	IsActive  bool              `json:"is_active"`
	LastLogin *time.Time        `json:"last_login,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// RefreshTokenRequest represents the token refresh request payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login authenticates a user and returns JWT tokens
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid request format",
		})
		return
	}

	// Find user by username or email
	var user models.User
	if err := h.database.DB().Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.logger.WithField("username", req.Username).Warn("Login attempt with invalid username")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid credentials",
			})
			return
		}
		h.logger.WithError(err).Error("Database error during login")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Authentication failed",
		})
		return
	}

	// Check if account is locked
	if user.IsLocked() {
		h.logger.WithField("user_id", user.ID).Warn("Login attempt on locked account")
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":   "Account Locked",
			"message": "Account is temporarily locked due to too many failed login attempts",
		})
		return
	}

	// Check if account is active
	if !user.IsActive {
		h.logger.WithField("user_id", user.ID).Warn("Login attempt on inactive account")
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Forbidden",
			"message": "Account is deactivated",
		})
		return
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		// Increment failed login attempts
		user.IncrementFailedLogins()
		if err := h.database.DB().Save(&user).Error; err != nil {
			h.logger.WithError(err).Error("Failed to update failed login count")
		}

		h.logger.WithField("user_id", user.ID).Warn("Failed login attempt")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid credentials",
		})
		return
	}

	// Reset failed login attempts and update last login
	user.ResetFailedLogins()
	now := time.Now()
	user.LastLogin = &now
	if err := h.database.DB().Save(&user).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update user login info")
		// Don't fail the login for this non-critical error
	}

	// Generate tokens
	accessToken, err := h.authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), middleware.TokenTypeAccess)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate access token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to generate access token",
		})
		return
	}

	refreshToken, err := h.authManager.GenerateToken(fmt.Sprintf("%d", user.ID), string(user.Role), middleware.TokenTypeRefresh)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to generate refresh token",
		})
		return
	}

	// Create user info response
	userInfo := &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    15 * 60, // 15 minutes
		User:         userInfo,
	})
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid registration request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid request format",
		})
		return
	}

	// Normalize and validate username
	req.Username = strings.TrimSpace(strings.ToLower(req.Username))
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Set default role if not provided
	if req.Role == "" {
		req.Role = string(models.RoleViewer)
	}

	// Validate role
	if !models.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid role specified",
		})
		return
	}

	// Check if username or email already exists
	var existingUser models.User
	if err := h.database.DB().Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "Username or email already exists",
		})
		return
	} else if err != gorm.ErrRecordNotFound {
		h.logger.WithError(err).Error("Database error during registration check")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Registration failed",
		})
		return
	}

	// Create new user
	user := models.User{
		Username:  req.Username,
		Email:     req.Email,
		Role:      models.UserRole(req.Role),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		IsActive:  true,
	}

	// Set password
	if err := user.SetPassword(req.Password); err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create user",
		})
		return
	}

	// Save user to database
	if err := h.database.DB().Create(&user).Error; err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create user",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
	}).Info("User registered successfully")

	// Return user info (without password)
	userInfo := &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    userInfo,
	})
}

// RefreshToken generates new access token from refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid request format",
		})
		return
	}

	// Validate refresh token
	claims, err := h.authManager.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid or expired refresh token",
		})
		return
	}

	// Ensure it's a refresh token
	if claims.TokenType != middleware.TokenTypeRefresh {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid token type",
		})
		return
	}

	// Generate new access token
	accessToken, err := h.authManager.GenerateToken(claims.UserID, claims.Role, middleware.TokenTypeAccess)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate new access token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to generate access token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   15 * 60, // 15 minutes
	})
}

// GetProfile returns the current user's profile information
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid user ID",
		})
		return
	}

	var user models.User
	if err := h.database.DB().First(&user, uint(userID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Database error getting user profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to get user profile",
		})
		return
	}

	userInfo := &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
	}

	c.JSON(http.StatusOK, userInfo)
}

// Logout invalidates the current session (placeholder for token blacklisting)
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a real implementation, we'd want to add the token to a blacklist
	// For now, we just return success - the client should discard the token
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}