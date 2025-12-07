package server

import (
	"context"
	"testing"
	"time"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthUnaryInterceptor_Disabled(t *testing.T) {
	cfg := &AuthConfig{
		Enabled: false,
	}

	interceptor := AuthUnaryInterceptor(cfg)

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, called, "handler should be called when auth is disabled")
}

func TestAuthUnaryInterceptor_SkipMethods(t *testing.T) {
	cfg := &AuthConfig{
		Enabled:     true,
		SkipMethods: []string{"/grpc.health.v1.Health/Check"},
	}

	interceptor := AuthUnaryInterceptor(cfg)

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, called, "handler should be called for skipped methods")
}

func TestAuthUnaryInterceptor_MissingMetadata(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret: []byte("test-secret-key-for-testing-purposes"),
	}, testLogger)
	require.NoError(t, err)

	cfg := &AuthConfig{
		Enabled:     true,
		AuthManager: authManager,
		Logger:      testLogger,
	}

	interceptor := AuthUnaryInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err = interceptor(context.Background(), nil, info, handler)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthUnaryInterceptor_MissingAuthHeader(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret: []byte("test-secret-key-for-testing-purposes"),
	}, testLogger)
	require.NoError(t, err)

	cfg := &AuthConfig{
		Enabled:     true,
		AuthManager: authManager,
		Logger:      testLogger,
	}

	interceptor := AuthUnaryInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Add metadata without authorization header
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, err = interceptor(ctx, nil, info, handler)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthUnaryInterceptor_InvalidTokenFormat(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret: []byte("test-secret-key-for-testing-purposes"),
	}, testLogger)
	require.NoError(t, err)

	cfg := &AuthConfig{
		Enabled:     true,
		AuthManager: authManager,
		Logger:      testLogger,
	}

	interceptor := AuthUnaryInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Add metadata with invalid authorization format
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
		"authorization": []string{"InvalidFormat token123"},
	})

	_, err = interceptor(ctx, nil, info, handler)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthUnaryInterceptor_ValidToken(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret:         []byte("test-secret-key-for-testing-purposes"),
		AccessTokenExpiry: 15 * time.Minute,
	}, testLogger)
	require.NoError(t, err)

	// Generate a valid token
	token, err := authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(t, err)

	cfg := &AuthConfig{
		Enabled:     true,
		AuthManager: authManager,
		Logger:      testLogger,
	}

	interceptor := AuthUnaryInterceptor(cfg)

	var capturedUserID string
	var capturedRole string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedUserID = GetUserIDFromContext(ctx)
		capturedRole = GetUserRoleFromContext(ctx)
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Add metadata with valid Bearer token
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
		"authorization": []string{"Bearer " + token},
	})

	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.Equal(t, "test-user", capturedUserID)
	assert.Equal(t, middleware.RoleAdmin, capturedRole)
}

func TestAuthUnaryInterceptor_InvalidToken(t *testing.T) {
	testLogger := logger.Default()
	authManager, err := middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret: []byte("test-secret-key-for-testing-purposes"),
	}, testLogger)
	require.NoError(t, err)

	cfg := &AuthConfig{
		Enabled:     true,
		AuthManager: authManager,
		Logger:      testLogger,
	}

	interceptor := AuthUnaryInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Add metadata with invalid token
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
		"authorization": []string{"Bearer invalid.token.here"},
	})

	_, err = interceptor(ctx, nil, info, handler)

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name         string
		userRole     string
		requiredRole string
		expectError  bool
		errorCode    codes.Code
	}{
		{
			name:         "Admin can access admin endpoints",
			userRole:     middleware.RoleAdmin,
			requiredRole: middleware.RoleAdmin,
			expectError:  false,
		},
		{
			name:         "Admin can access operator endpoints",
			userRole:     middleware.RoleAdmin,
			requiredRole: middleware.RoleOperator,
			expectError:  false,
		},
		{
			name:         "Admin can access viewer endpoints",
			userRole:     middleware.RoleAdmin,
			requiredRole: middleware.RoleViewer,
			expectError:  false,
		},
		{
			name:         "Operator can access operator endpoints",
			userRole:     middleware.RoleOperator,
			requiredRole: middleware.RoleOperator,
			expectError:  false,
		},
		{
			name:         "Operator can access viewer endpoints",
			userRole:     middleware.RoleOperator,
			requiredRole: middleware.RoleViewer,
			expectError:  false,
		},
		{
			name:         "Operator cannot access admin endpoints",
			userRole:     middleware.RoleOperator,
			requiredRole: middleware.RoleAdmin,
			expectError:  true,
			errorCode:    codes.PermissionDenied,
		},
		{
			name:         "Viewer can access viewer endpoints",
			userRole:     middleware.RoleViewer,
			requiredRole: middleware.RoleViewer,
			expectError:  false,
		},
		{
			name:         "Viewer cannot access operator endpoints",
			userRole:     middleware.RoleViewer,
			requiredRole: middleware.RoleOperator,
			expectError:  true,
			errorCode:    codes.PermissionDenied,
		},
		{
			name:         "Viewer cannot access admin endpoints",
			userRole:     middleware.RoleViewer,
			requiredRole: middleware.RoleAdmin,
			expectError:  true,
			errorCode:    codes.PermissionDenied,
		},
		{
			name:         "No role returns unauthenticated",
			userRole:     "",
			requiredRole: middleware.RoleViewer,
			expectError:  true,
			errorCode:    codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.userRole != "" {
				ctx = setAuthContext(ctx, "test-user", tt.userRole)
			}

			err := RequireRole(ctx, tt.requiredRole)

			if tt.expectError {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContextHelpers(t *testing.T) {
	t.Run("GetUserIDFromContext", func(t *testing.T) {
		ctx := setAuthContext(context.Background(), "user-123", middleware.RoleAdmin)
		assert.Equal(t, "user-123", GetUserIDFromContext(ctx))

		// Empty context
		assert.Equal(t, "", GetUserIDFromContext(context.Background()))
	})

	t.Run("GetUserRoleFromContext", func(t *testing.T) {
		ctx := setAuthContext(context.Background(), "user-123", middleware.RoleOperator)
		assert.Equal(t, middleware.RoleOperator, GetUserRoleFromContext(ctx))

		// Empty context
		assert.Equal(t, "", GetUserRoleFromContext(context.Background()))
	})

	t.Run("IsAuthenticated", func(t *testing.T) {
		ctx := setAuthContext(context.Background(), "user-123", middleware.RoleViewer)
		assert.True(t, IsAuthenticated(ctx))

		// Empty context
		assert.False(t, IsAuthenticated(context.Background()))
	})
}

func TestDefaultSkipMethods(t *testing.T) {
	skipMethods := DefaultSkipMethods()

	assert.Contains(t, skipMethods, "/grpc.health.v1.Health/Check")
	assert.Contains(t, skipMethods, "/grpc.health.v1.Health/Watch")
	assert.Contains(t, skipMethods, "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo")
}

func TestShouldSkipAuth(t *testing.T) {
	skipMethods := []string{
		"/grpc.health.v1.Health/Check",
		"CustomService/CustomMethod",
	}

	tests := []struct {
		method     string
		shouldSkip bool
	}{
		{"/grpc.health.v1.Health/Check", true},
		{"CustomService/CustomMethod", true},
		{"/some.package/grpc.health.v1.Health/Check", true}, // suffix match
		{"/test.Service/OtherMethod", false},
		{"/protected.Service/SecureMethod", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := shouldSkipAuth(tt.method, skipMethods)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}
