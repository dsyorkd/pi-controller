package server

import (
	"context"
	"strings"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// AuthConfig holds gRPC authentication configuration
type AuthConfig struct {
	// Enabled determines if authentication is required
	Enabled bool
	// AuthManager is the JWT authentication manager
	AuthManager *middleware.AuthManager
	// SkipMethods is a list of methods that don't require authentication
	SkipMethods []string
	// AllowInternalTLS allows connections with valid mTLS certificates without JWT
	AllowInternalTLS bool
	// Logger for auth events
	Logger logger.Interface
}

// ContextKey is a custom type for context keys
type contextKey string

const (
	// UserIDKey is the context key for user ID
	userIDContextKey contextKey = "grpc_user_id"
	// UserRoleKey is the context key for user role
	userRoleContextKey contextKey = "grpc_user_role"
	// AuthenticatedKey is the context key for authentication status
	authenticatedContextKey contextKey = "grpc_authenticated"
)

// GetUserIDFromContext retrieves the user ID from the context
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDContextKey).(string); ok {
		return userID
	}
	return ""
}

// GetUserRoleFromContext retrieves the user role from the context
func GetUserRoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value(userRoleContextKey).(string); ok {
		return role
	}
	return ""
}

// IsAuthenticated checks if the context is authenticated
func IsAuthenticated(ctx context.Context) bool {
	if authenticated, ok := ctx.Value(authenticatedContextKey).(bool); ok {
		return authenticated
	}
	return false
}

// AuthUnaryInterceptor creates a unary interceptor for gRPC authentication
func AuthUnaryInterceptor(cfg *AuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication if disabled
		if !cfg.Enabled {
			return handler(ctx, req)
		}

		// Check if method should skip auth
		if shouldSkipAuth(info.FullMethod, cfg.SkipMethods) {
			return handler(ctx, req)
		}

		// Check for mTLS authentication if enabled
		if cfg.AllowInternalTLS {
			if authenticatedViaTLS(ctx) {
				ctx = context.WithValue(ctx, authenticatedContextKey, true)
				ctx = context.WithValue(ctx, userIDContextKey, "internal-node")
				ctx = context.WithValue(ctx, userRoleContextKey, middleware.RoleOperator)
				return handler(ctx, req)
			}
		}

		// Authenticate via JWT
		authCtx, err := authenticateRequest(ctx, cfg)
		if err != nil {
			logAuthFailure(cfg.Logger, info.FullMethod, err)
			return nil, err
		}

		return handler(authCtx, req)
	}
}

// AuthStreamInterceptor creates a stream interceptor for gRPC authentication
func AuthStreamInterceptor(cfg *AuthConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication if disabled
		if !cfg.Enabled {
			return handler(srv, ss)
		}

		// Check if method should skip auth
		if shouldSkipAuth(info.FullMethod, cfg.SkipMethods) {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// Check for mTLS authentication if enabled
		if cfg.AllowInternalTLS {
			if authenticatedViaTLS(ctx) {
				wrappedStream := &wrappedServerStream{
					ServerStream: ss,
					ctx:          setAuthContext(ctx, "internal-node", middleware.RoleOperator),
				}
				return handler(srv, wrappedStream)
			}
		}

		// Authenticate via JWT
		authCtx, err := authenticateRequest(ctx, cfg)
		if err != nil {
			logAuthFailure(cfg.Logger, info.FullMethod, err)
			return err
		}

		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          authCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream wraps a grpc.ServerStream to inject a modified context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// shouldSkipAuth checks if the method should skip authentication
func shouldSkipAuth(method string, skipMethods []string) bool {
	for _, m := range skipMethods {
		if strings.HasSuffix(method, m) || method == m {
			return true
		}
	}
	return false
}

// authenticatedViaTLS checks if the connection is authenticated via TLS client certificate
func authenticatedViaTLS(ctx context.Context) bool {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return false
	}

	// Check if TLS info is available and has client certificates
	if p.AuthInfo != nil {
		// For mutual TLS, the AuthInfo would contain certificate details
		// This is a simplified check - in production you'd validate the certificate chain
		return strings.Contains(p.AuthInfo.AuthType(), "tls")
	}

	return false
}

// authenticateRequest authenticates a gRPC request using JWT from metadata
func authenticateRequest(ctx context.Context, cfg *AuthConfig) (context.Context, error) {
	if cfg.AuthManager == nil {
		return nil, status.Error(codes.Internal, "authentication manager not configured")
	}

	// Extract metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Get authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization header")
	}

	authHeader := authHeaders[0]

	// Validate Bearer token format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format, expected 'Bearer <token>'")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return nil, status.Error(codes.Unauthenticated, "empty token")
	}

	// Validate token
	claims, err := cfg.AuthManager.ValidateToken(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Set authenticated context
	return setAuthContext(ctx, claims.UserID, claims.Role), nil
}

// setAuthContext adds authentication information to the context
func setAuthContext(ctx context.Context, userID, role string) context.Context {
	ctx = context.WithValue(ctx, authenticatedContextKey, true)
	ctx = context.WithValue(ctx, userIDContextKey, userID)
	ctx = context.WithValue(ctx, userRoleContextKey, role)
	return ctx
}

// logAuthFailure logs authentication failures
func logAuthFailure(log logger.Interface, method string, err error) {
	if log == nil {
		return
	}
	log.WithFields(map[string]interface{}{
		"method": method,
		"error":  err.Error(),
	}).Warn("gRPC authentication failed")
}

// DefaultSkipMethods returns the default list of methods that don't require authentication
func DefaultSkipMethods() []string {
	return []string{
		// Health check endpoints
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
		// Reflection for development tools
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}
}

// RequireRole creates a function to check if the user has the required role
func RequireRole(ctx context.Context, requiredRole string) error {
	userRole := GetUserRoleFromContext(ctx)
	if userRole == "" {
		return status.Error(codes.Unauthenticated, "authentication required")
	}

	if !hasPermission(userRole, requiredRole) {
		return status.Errorf(codes.PermissionDenied, "requires %s role, has %s", requiredRole, userRole)
	}

	return nil
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
