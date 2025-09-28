# Secure Authentication Implementation

This document describes the secure authentication system implemented for the Pi Controller web application using httpOnly cookies and CSRF protection.

## Overview

The secure authentication system replaces the previous localStorage-based token storage with a more secure approach using:

- **httpOnly Cookies**: Prevents XSS attacks by making tokens inaccessible to JavaScript
- **CSRF Protection**: Protects against Cross-Site Request Forgery attacks
- **Automatic Token Refresh**: Seamless token renewal without user intervention
- **Session Monitoring**: Automatic logout on session expiry or invalidation

## Security Features

### 1. httpOnly Cookies
- Authentication tokens are stored in httpOnly cookies
- Cookies are inaccessible to client-side JavaScript
- Prevents XSS-based token theft
- Automatic inclusion in API requests

### 2. CSRF Protection
- Unique CSRF tokens generated for each session
- CSRF tokens included in all API requests
- Server validates CSRF tokens on protected endpoints
- Prevents unauthorized requests from malicious sites

### 3. Secure Cookie Configuration
```http
Set-Cookie: pi_session=<token>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=3600
```

- `HttpOnly`: Prevents JavaScript access
- `Secure`: Only sent over HTTPS
- `SameSite=Strict`: Prevents cross-site request inclusion
- `Path=/`: Cookie scope
- `Max-Age`: Automatic expiry

### 4. Session Management
- Server-side session validation
- Automatic token refresh before expiry
- Session monitoring and cleanup
- Graceful logout handling

## Implementation Components

### Core Files

1. **`/utils/secureSession.ts`**
   - Secure session management utilities
   - CSRF token generation and management
   - Cookie-based authentication helpers

2. **`/services/secureApi.ts`**
   - Secure API service with cookie support
   - Automatic CSRF token inclusion
   - Token refresh handling

3. **`/components/auth/SecureAuthProvider.tsx`**
   - React context for secure authentication
   - Session state management
   - Automatic logout on expiry

4. **`/components/auth/SecureProtectedRoute.tsx`**
   - Protected route component
   - Role-based access control
   - Secure session validation

5. **`/pages/auth/SecureLoginPage.tsx`**
   - Secure login form
   - CSRF token inclusion
   - Enhanced security indicators

6. **`/SecureApp.tsx`**
   - Main application with secure auth
   - Demonstration of secure implementation

### Backend Requirements

The secure implementation requires corresponding backend support:

```go
// Required backend endpoints:
POST /api/v1/auth/login          // Login with secure cookie setting
POST /api/v1/auth/logout         // Logout with cookie clearing
POST /api/v1/auth/refresh        // Token refresh
GET  /api/v1/auth/session-status // Session validation
POST /api/v1/auth/set-cookies    // Secure cookie management
POST /api/v1/auth/clear-cookies  // Secure cookie clearing
```

## Migration from localStorage

### Before (Insecure)
```typescript
// Old localStorage-based approach
localStorage.setItem('authToken', token);
const token = localStorage.getItem('authToken');
```

### After (Secure)
```typescript
// New httpOnly cookie approach
await storeTokensSecurely(token, refreshToken, expiresIn);
const isValid = await getCurrentTokenSecurely();
```

## Usage Examples

### 1. Secure Login
```typescript
import { useSecureAuthContext } from './components/auth/SecureAuthProvider';

const LoginComponent = () => {
  const { login, isLoading, error } = useSecureAuthContext();

  const handleLogin = async (email: string, password: string) => {
    try {
      await login({ email, password });
      // Automatically redirected on success
    } catch (err) {
      // Error handled by context
    }
  };
};
```

### 2. Secure API Calls
```typescript
import { secureApiService } from './services/secureApi';

// Automatically includes CSRF tokens and cookies
const clusters = await secureApiService.clusters.getAll();
```

### 3. Protected Routes
```typescript
import { SecureProtectedRoute } from './components/auth/SecureProtectedRoute';

<SecureProtectedRoute requiredRole="admin">
  <AdminDashboard />
</SecureProtectedRoute>
```

## Security Benefits

### Vulnerability Mitigation

1. **XSS Protection**: Tokens in httpOnly cookies can't be accessed by malicious scripts
2. **CSRF Protection**: CSRF tokens prevent unauthorized cross-site requests
3. **Token Theft Prevention**: Secure cookies reduce attack surface
4. **Session Hijacking**: Secure cookie flags prevent interception
5. **Automatic Cleanup**: Server-managed expiry prevents stale sessions

### Compliance

- Follows OWASP security guidelines
- Implements defense-in-depth security
- Provides audit trail capabilities
- Supports security monitoring

## Configuration

### Environment Variables
```env
# Security settings
VITE_API_BASE_URL=https://your-api-domain.com/api/v1
VITE_CSRF_COOKIE_NAME=pi_csrf_token
VITE_SESSION_COOKIE_NAME=pi_session
```

### Build Configuration
```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false, // Set to true in production
      },
    },
  },
});
```

## Testing

### Unit Tests
```bash
npm run test -- --testPathPattern=secure
```

### Security Tests
```bash
npm run test:security
```

### Integration Tests
```bash
npm run test:integration -- --grep "secure auth"
```

## Production Deployment

### HTTPS Requirements
- All production deployments must use HTTPS
- Secure cookie flags require HTTPS
- Mixed content policies enforced

### Cookie Configuration
```nginx
# Nginx configuration for secure cookies
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header X-XSS-Protection "1; mode=block" always;
```

## Monitoring and Logging

### Security Events
- Failed login attempts
- CSRF token validation failures
- Session expiry events
- Unauthorized access attempts

### Metrics
- Session duration statistics
- Authentication success rates
- Security violation counts
- Token refresh frequency

## Troubleshooting

### Common Issues

1. **CSRF Token Mismatch**
   - Check token generation
   - Verify header inclusion
   - Validate server configuration

2. **Cookie Not Set**
   - Verify HTTPS in production
   - Check SameSite settings
   - Validate domain configuration

3. **Session Expired**
   - Check token refresh logic
   - Verify expiry settings
   - Monitor session duration

### Debug Mode
```typescript
// Enable debug logging
localStorage.setItem('DEBUG_SECURE_AUTH', 'true');
```

## Future Enhancements

- [ ] Multi-factor authentication (MFA)
- [ ] Session activity monitoring
- [ ] Device fingerprinting
- [ ] Rate limiting integration
- [ ] Advanced threat detection

## Support

For security-related issues or questions:
- Review this documentation
- Check the implementation examples
- Test with the provided secure components
- Validate backend endpoint compatibility