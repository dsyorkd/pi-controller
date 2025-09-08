package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user in the system with authentication and authorization details
type User struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	Username     string         `json:"username" gorm:"uniqueIndex;not null"`
	Email        string         `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string         `json:"-" gorm:"not null"` // Never expose password hash in JSON
	Role         UserRole       `json:"role" gorm:"not null;default:'viewer'"`
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	LastLogin    *time.Time     `json:"last_login,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// API Key support for service accounts
	APIKey       *string    `json:"-" gorm:"uniqueIndex"` // Never expose API key in JSON, make nullable
	APIKeyExpiry *time.Time `json:"api_key_expiry,omitempty"`

	// Security fields
	FailedLogins  int        `json:"-" gorm:"default:0"`
	LockedUntil   *time.Time `json:"-"`
	PasswordReset string     `json:"-"` // Password reset token
	ResetExpiry   *time.Time `json:"-"`
}

// UserRole defines the possible roles for users
type UserRole string

const (
	RoleViewer   UserRole = "viewer"   // Read-only access
	RoleOperator UserRole = "operator" // Read/write access to operations
	RoleAdmin    UserRole = "admin"    // Full administrative access
)

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword verifies the provided password against the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsValidRole returns true if the provided role is valid
func IsValidRole(role string) bool {
	validRoles := []UserRole{RoleViewer, RoleOperator, RoleAdmin}
	for _, validRole := range validRoles {
		if string(validRole) == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has permission for the required role
func (u *User) HasPermission(requiredRole string) bool {
	// Admin can access everything
	if u.Role == RoleAdmin {
		return true
	}

	// Operator can access operator and viewer endpoints
	if u.Role == RoleOperator && (requiredRole == string(RoleOperator) || requiredRole == string(RoleViewer)) {
		return true
	}

	// Viewer can only access viewer endpoints
	if u.Role == RoleViewer && requiredRole == string(RoleViewer) {
		return true
	}

	return false
}

// IsLocked returns true if the user account is temporarily locked
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && time.Now().Before(*u.LockedUntil)
}

// IncrementFailedLogins increments the failed login counter and locks account if threshold is reached
func (u *User) IncrementFailedLogins() {
	u.FailedLogins++

	// Lock account for 15 minutes after 5 failed attempts
	if u.FailedLogins >= 5 {
		lockUntil := time.Now().Add(15 * time.Minute)
		u.LockedUntil = &lockUntil
	}
}

// ResetFailedLogins resets the failed login counter and unlocks the account
func (u *User) ResetFailedLogins() {
	u.FailedLogins = 0
	u.LockedUntil = nil
}

// TableName returns the table name for the User model
func (u *User) TableName() string {
	return "users"
}
