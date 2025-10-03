package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestUser_SetPassword(t *testing.T) {
	user := &User{
		Username: "testuser",
	}

	password := "secretPassword123!"
	err := user.SetPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, password, user.PasswordHash)

	// Verify the hash is valid bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	assert.NoError(t, err)
}

func TestUser_CheckPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		check    string
		want     bool
	}{
		{
			name:     "correct password",
			password: "correctPassword123",
			check:    "correctPassword123",
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "correctPassword123",
			check:    "wrongPassword",
			want:     false,
		},
		{
			name:     "empty password",
			password: "correctPassword123",
			check:    "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Username: "testuser"}
			err := user.SetPassword(tt.password)
			assert.NoError(t, err)

			result := user.CheckPassword(tt.check)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		isActive bool
		want     bool
	}{
		{
			name:     "active user",
			isActive: true,
			want:     true,
		},
		{
			name:     "inactive user",
			isActive: false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{IsActive: tt.isActive}
			assert.Equal(t, tt.want, user.IsActive)
		})
	}
}

func TestUserRole_Constants(t *testing.T) {
	assert.Equal(t, UserRole("admin"), RoleAdmin)
	assert.Equal(t, UserRole("operator"), RoleOperator)
	assert.Equal(t, UserRole("viewer"), RoleViewer)
}

func TestUser_IsLocked(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		lockedUntil *time.Time
		want        bool
	}{
		{
			name:        "not locked",
			lockedUntil: nil,
			want:        false,
		},
		{
			name:        "locked in future",
			lockedUntil: func() *time.Time { t := now.Add(1 * time.Hour); return &t }(),
			want:        true,
		},
		{
			name:        "lock expired",
			lockedUntil: func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{LockedUntil: tt.lockedUntil}
			assert.Equal(t, tt.want, user.IsLocked())
		})
	}
}

func TestUser_BasicFields(t *testing.T) {
	user := &User{
		ID:        1,
		Username:  "admin",
		Email:     "admin@example.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      RoleAdmin,
		IsActive:  true,
	}

	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, "admin@example.com", user.Email)
	assert.Equal(t, "Admin", user.FirstName)
	assert.Equal(t, "User", user.LastName)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.True(t, user.IsActive)
}

func TestUser_HasPermission(t *testing.T) {
	tests := []struct {
		name         string
		userRole     UserRole
		requiredRole string
		want         bool
	}{
		{
			name:         "admin can access admin",
			userRole:     RoleAdmin,
			requiredRole: "admin",
			want:         true,
		},
		{
			name:         "admin can access operator",
			userRole:     RoleAdmin,
			requiredRole: "operator",
			want:         true,
		},
		{
			name:         "admin can access viewer",
			userRole:     RoleAdmin,
			requiredRole: "viewer",
			want:         true,
		},
		{
			name:         "operator can access operator",
			userRole:     RoleOperator,
			requiredRole: "operator",
			want:         true,
		},
		{
			name:         "operator can access viewer",
			userRole:     RoleOperator,
			requiredRole: "viewer",
			want:         true,
		},
		{
			name:         "operator cannot access admin",
			userRole:     RoleOperator,
			requiredRole: "admin",
			want:         false,
		},
		{
			name:         "viewer can access viewer",
			userRole:     RoleViewer,
			requiredRole: "viewer",
			want:         true,
		},
		{
			name:         "viewer cannot access operator",
			userRole:     RoleViewer,
			requiredRole: "operator",
			want:         false,
		},
		{
			name:         "viewer cannot access admin",
			userRole:     RoleViewer,
			requiredRole: "admin",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.userRole}
			assert.Equal(t, tt.want, user.HasPermission(tt.requiredRole))
		})
	}
}

func TestUser_PasswordHashEmpty(t *testing.T) {
	user := &User{
		Username: "testuser",
	}

	// CheckPassword should return false when password hash is empty
	result := user.CheckPassword("anypassword")
	assert.False(t, result)
}

func TestUser_SetPasswordErrors(t *testing.T) {
	user := &User{
		Username: "testuser",
	}

	// Test with empty password
	err := user.SetPassword("")
	assert.NoError(t, err) // bcrypt will hash even empty strings

	// Verify empty password can be checked
	assert.True(t, user.CheckPassword(""))
}

func TestUser_ConcurrentPasswordChecks(t *testing.T) {
	user := &User{Username: "testuser"}
	password := "testPassword123"
	err := user.SetPassword(password)
	assert.NoError(t, err)

	// Run multiple password checks concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result := user.CheckPassword(password)
			assert.True(t, result)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUser_IncrementFailedLogins(t *testing.T) {
	user := &User{Username: "testuser"}

	// Initial state
	assert.Equal(t, 0, user.FailedLogins)
	assert.Nil(t, user.LockedUntil)
	assert.False(t, user.IsLocked())

	// Increment 4 times - should not lock
	for i := 0; i < 4; i++ {
		user.IncrementFailedLogins()
	}
	assert.Equal(t, 4, user.FailedLogins)
	assert.Nil(t, user.LockedUntil)
	assert.False(t, user.IsLocked())

	// 5th increment should lock the account
	user.IncrementFailedLogins()
	assert.Equal(t, 5, user.FailedLogins)
	assert.NotNil(t, user.LockedUntil)
	assert.True(t, user.IsLocked())
}

func TestUser_ResetFailedLogins(t *testing.T) {
	now := time.Now()
	lockUntil := now.Add(15 * time.Minute)
	user := &User{
		Username:     "testuser",
		FailedLogins: 5,
		LockedUntil:  &lockUntil,
	}

	assert.Equal(t, 5, user.FailedLogins)
	assert.True(t, user.IsLocked())

	user.ResetFailedLogins()

	assert.Equal(t, 0, user.FailedLogins)
	assert.Nil(t, user.LockedUntil)
	assert.False(t, user.IsLocked())
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{name: "admin is valid", role: "admin", want: true},
		{name: "operator is valid", role: "operator", want: true},
		{name: "viewer is valid", role: "viewer", want: true},
		{name: "invalid role", role: "superuser", want: false},
		{name: "empty role", role: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidRole(tt.role))
		})
	}
}
