# Security Implementation Summary

This document provides a comprehensive overview of the security measures implemented in the pi-controller project.

## Table of Contents

- [Authentication & Authorization](#authentication--authorization)
- [gRPC Authentication](#grpc-authentication)
- [WebSocket Authentication](#websocket-authentication)
- [Security Headers](#security-headers)
- [GPIO Pin Safety](#gpio-pin-safety)
- [Rate Limiting](#rate-limiting)
- [Input Validation](#input-validation)
- [Security Compliance Checklist](#security-compliance-checklist)

---

## Authentication & Authorization

### JWT-Based Authentication

**Implementation:** `internal/api/middleware/auth.go`

The system uses industry-standard JWT (JSON Web Tokens) for authentication with the following features:

#### Token Types

- **Access Tokens**: Short-lived (15 minutes) for API access
- **Refresh Tokens**: Longer-lived (7 days) for token renewal
- **API Keys**: Service account tokens (90 days)

#### Security Features

1. **Secure Token Storage**
   - httpOnly cookies (protected from XSS)
   - SameSite=Strict (CSRF protection)
   - Secure flag (HTTPS-only in production)
   - Automatic token rotation on refresh

2. **Password Security**
   - bcrypt hashing with default cost (10 rounds)
   - Account lockout after 5 failed attempts (15-minute lockout)
   - Password reset tokens with expiry

3. **Session Management**
   - CSRF token generation and validation
   - Cookie-based auth for browsers
   - Bearer token auth for API clients
   - Dual-mode support (cookie + header)

### Role-Based Access Control (RBAC)

**Three-tier permission system:**

| Role | Permissions | Use Case |
|------|-------------|----------|
| `viewer` | Read-only access to all resources | Monitoring, dashboards |
| `operator` | Read/write access, GPIO control, node management | Day-to-day operations |
| `admin` | Full access including DELETE operations, cluster lifecycle | System administration |

**Permission Hierarchy:**

```
admin > operator > viewer
```

- Admin can access everything
- Operator can access operator + viewer endpoints
- Viewer can only access viewer endpoints

### API Route Protection

**File:** `internal/api/server.go`

All API routes require authentication when `AuthEnabled: true` in config:

```go
// Example: Cluster management with role-based access
clusters.GET("", s.requireRole("viewer"), clusterHandler.List)          // Read
clusters.POST("", s.requireRole("operator"), clusterHandler.Create)      // Create
clusters.PUT("/:id", s.requireRole("operator"), clusterHandler.Update)   // Update
clusters.DELETE("/:id", s.requireRole("admin"), clusterHandler.Delete)   // Delete (admin only)
```

**Critical DELETE Operations** (Task 67 ✅):

- All DELETE endpoints require `admin` role
- Includes: clusters, nodes, GPIO devices, certificates
- Prevents unauthorized resource deletion

### Audit Logging

**Enabled by default** when `EnableAuditLog: true`:

Logs all authentication events:

- Login attempts (success/failure)
- Token validation failures
- Authorization failures
- User actions with request context

---

## gRPC Authentication

### Interceptor-Based Authentication

**Implementation:** `internal/grpc/server/auth_interceptor.go`

The gRPC server uses interceptors to authenticate all RPC calls using JWT tokens.

#### Configuration

```go
authCfg := &server.AuthConfig{
    Enabled:          true,
    AuthManager:      authManager,  // Shared with HTTP API
    SkipMethods:      server.DefaultSkipMethods(),
    AllowInternalTLS: false,
}
```

#### Interceptor Chain

The server uses chained interceptors for both unary and streaming RPCs:

1. **Logging Interceptor** - Logs all RPC calls
2. **Auth Interceptor** - Validates JWT tokens from metadata

#### Token Validation

Tokens are extracted from gRPC metadata (headers):

```go
// Client sends token in metadata
md := metadata.Pairs("authorization", "Bearer "+token)
ctx := metadata.NewOutgoingContext(context.Background(), md)
```

#### Skipped Methods

Health check endpoints bypass authentication for load balancer compatibility:

```go
DefaultSkipMethods() = []string{
    "/grpc.health.v1.Health/Check",
    "/grpc.health.v1.Health/Watch",
    "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
}
```

#### Role-Based Access Control

Use `RequireRole` in RPC handlers:

```go
func (s *PiControllerServer) DeleteCluster(ctx context.Context, req *pb.DeleteClusterRequest) (*pb.Empty, error) {
    // Require admin role for DELETE operations
    if err := server.RequireRole(ctx, middleware.RoleAdmin); err != nil {
        return nil, err  // Returns codes.PermissionDenied
    }
    // ... implementation
}
```

#### Context Helpers

```go
// Get authenticated user info
userID := server.GetUserIDFromContext(ctx)
role := server.GetUserRoleFromContext(ctx)
isAuth := server.IsAuthenticated(ctx)
```

---

## WebSocket Authentication

### Connection-Time Authentication

**Implementation:** `internal/websocket/auth.go`

WebSocket connections can be authenticated at connection time or via messages after connection.

#### Authentication Methods

1. **Query Parameter** (for JavaScript clients that can't set headers)

   ```javascript
   const ws = new WebSocket('wss://host/ws?token=<jwt-token>')
   ```

2. **Authorization Header** (for clients that support it)

   ```javascript
   const ws = new WebSocket('wss://host/ws', {
     headers: { 'Authorization': 'Bearer <jwt-token>' }
   })
   ```

3. **Post-Connection Auth Message** (for re-authentication or token refresh)

   ```json
   {
     "type": "auth",
     "payload": { "token": "<jwt-token>" },
     "timestamp": "2025-01-01T00:00:00Z"
   }
   ```

#### Configuration

```go
authConfig := &websocket.AuthConfig{
    Enabled:        true,
    AuthManager:    authManager,  // Shared with HTTP API
    AllowAnonymous: false,        // Set true for read-only public access
    Logger:         logger,
}
```

#### Anonymous Access Mode

When `AllowAnonymous: true`:

- Unauthenticated connections are allowed with `viewer` role
- UserID is set to `anonymous`
- Can only subscribe to public topics
- Write operations still require authentication

#### Subscription Authorization

Subscriptions require authentication when auth is enabled:

```go
case MessageTypeSubscribe:
    if s.authConfig != nil && s.authConfig.Enabled && !c.authenticated {
        c.sendError(401, "Authentication required")
        return
    }
    // ... allow subscription
```

#### Role-Based Topic Access

```go
// Check if client can access a topic
if client.hasRole(middleware.RoleOperator) {
    // Can subscribe to operational topics
}
```

---

## Security Headers

### HTTP Security Headers Middleware

**Implementation:** `internal/api/middleware/security.go`

The API server applies security headers to all HTTP responses.

#### Headers Applied

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | `nosniff` | Prevents MIME type sniffing |
| `X-Frame-Options` | `DENY` | Prevents clickjacking |
| `X-XSS-Protection` | `1; mode=block` | Legacy XSS protection |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Controls referrer leakage |
| `Permissions-Policy` | `geolocation=(), microphone=(), camera=()` | Disables sensitive features |
| `Cache-Control` | `no-store, no-cache, must-revalidate` | Prevents sensitive data caching |
| `Pragma` | `no-cache` | HTTP/1.0 cache prevention |

#### Content Security Policy (CSP)

```
default-src 'self';
script-src 'self';
style-src 'self' 'unsafe-inline';
img-src 'self' data:;
font-src 'self';
connect-src 'self' wss:;
frame-ancestors 'none';
base-uri 'self';
form-action 'self'
```

#### HTTPS Enforcement

In production environment (`PI_CONTROLLER_ENVIRONMENT=production`):

- HTTP requests are redirected to HTTPS
- HSTS header is added: `Strict-Transport-Security: max-age=31536000; includeSubDomains`

#### Host Validation

Optional host validation prevents host header attacks:

```go
securityMiddleware := middleware.NewSecurityMiddleware(
    middleware.WithAllowedHosts([]string{"api.example.com", "localhost"}),
)
```

---

## GPIO Pin Safety

### Pin Safelisting (Task 61 ✅)

**Implementation:** `internal/config/gpio_config.go`

**Problem:** GPIO pins 0, 1, 14, 15 are system-critical (I2C, UART). Allowing user control could crash the system.

**Solution:** Strict pin safelist with validation.

#### System Critical Pins (Never Allowed)

| Pin | Function | Risk |
|-----|----------|------|
| 0 | I2C0 SDA | System I2C data line - crash risk |
| 1 | I2C0 SCL | System I2C clock line - crash risk |
| 2 | I2C1 SDA | Alternate I2C data line |
| 3 | I2C1 SCL | Alternate I2C clock line |
| 14 | UART TXD | Serial transmit - communication loss |
| 15 | UART RXD | Serial receive - communication loss |

#### Default Safe GPIO Pins

BCM GPIO pins 4-27 (excluding critical pins above):

```
4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27
```

#### Configuration

**Environment Variables:**

```bash
# Override safe pins (comma-separated)
export GPIO_SAFE_PINS="17,27,22,23,24,25"

# Disable safelist (NOT RECOMMENDED for production)
export GPIO_DISABLE_SAFELIST="true"
```

#### Validation Flow

```
User creates GPIO device
    ↓
config.ValidateGPIOPin(pinNumber)
    ↓
1. Check: pinNumber >= 0
2. Check: pinNumber <= 27 (BCM max)
3. Check: NOT in SystemCriticalPins
4. Check: IN safelist (if enabled)
    ↓
✅ Accept or ❌ Reject with reason
```

**Implementation in Service:** `internal/services/gpio.go:196-202`

```go
func (s *GPIOService) Create(req CreateGPIODeviceRequest) (*models.GPIODevice, error) {
    // CRITICAL SECURITY CHECK - Validate pin safety
    if err := config.ValidateGPIOPin(req.PinNumber); err != nil {
        return nil, errors.Wrapf(ErrValidationFailed, "pin safety check failed: %s", err.Error())
    }
    // ... rest of creation logic
}
```

---

## Rate Limiting

### Global API Rate Limiting

**Implementation:** `internal/api/middleware/ratelimit.go`

**Default Configuration:**

- **60 requests/minute** per client
- **Burst size:** 10 requests
- Tracking: By user ID (authenticated) or IP address
- Whitelisted: 127.0.0.1, ::1 (localhost)

### GPIO-Specific Rate Limiting (Task 68 ✅)

**Implementation:** `internal/api/middleware/gpio_ratelimit.go`

**Problem:** Rapid GPIO switching can damage connected hardware (relays, motors, etc.).

**Solution:** Stricter rate limits specifically for GPIO endpoints.

**Configuration:**

- **10 requests/second** (much stricter than global)
- **Burst size:** 20 requests
- Applied to ALL `/api/v1/gpio/*` endpoints
- Hardware protection priority

**Why Stricter?**

- Prevents rapid relay toggling (can cause coil burnout)
- Protects against GPIO DoS attacks
- Prevents motor controller damage
- Limits electrical stress on pins

**Applied in:** `internal/api/server.go:196`

```go
gpio := v1.Group("/gpio")
{
    // Apply GPIO-specific rate limiting to ALL GPIO endpoints
    gpio.Use(s.gpioRateLimiter.RateLimit())

    // ... GPIO handlers
}
```

### Rate Limit Response Headers

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1234567890
Retry-After: 1

{
  "error": "Rate Limit Exceeded",
  "message": "GPIO operation rate limit exceeded. Rapid GPIO switching can damage hardware.",
  "retry_after": 1,
  "limit": 10
}
```

---

## Input Validation

### Validation Middleware (Task 60 ✅)

**Implementation:** `internal/api/middleware/validation.go`

**Enabled by default** for all API requests.

#### Features

1. **HTTP Method Validation**
   - Only allows: GET, POST, PUT, DELETE, PATCH, OPTIONS
   - Returns 405 Method Not Allowed for invalid methods

2. **SQL Injection Protection**
   - Scans all inputs for SQL patterns
   - Blocks: quotes, semicolons, SQL keywords (DROP, DELETE, UNION, etc.)
   - Applied to: names, descriptions, query parameters

3. **XSS Protection**
   - Detects script tags, event handlers
   - Blocks: `<script>`, `javascript:`, `onload=`, etc.
   - Applied to all text inputs

4. **Input Sanitization**
   - Name validation: alphanumeric, dots, hyphens, underscores only
   - Max lengths: names (63 chars), descriptions (255 chars)
   - Hostname/IP validation with regex patterns
   - Port range validation (1-65535)

5. **Query Parameter Validation**
   - Limit: 0-1000 (prevents excessive queries)
   - Offset: non-negative integers only
   - Injection checking on all query params

#### Example Validations

```go
// Name validation
validator.ValidateName("my-cluster", "cluster")
// ✅ Allows: alphanumeric, dots, hyphens, underscores
// ❌ Rejects: spaces, special chars, SQL patterns

// Description validation
validator.ValidateDescription("Production cluster for IoT devices")
// ✅ Allows: normal text
// ❌ Rejects: <script>alert('xss')</script>

// GPIO pin validation
validator.ValidateGPIOPin(17)
// ✅ Allows: pins in safe range (4-27, excluding critical)
// ❌ Rejects: pin 0 (system critical I2C)
```

### 405 Method Not Allowed Handling

**Implementation:** `internal/api/server.go:97-99`

```go
// Add NoMethod handler to return 405 for unsupported methods
s.router.NoMethod(func(c *gin.Context) {
    c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
})
```

Properly returns HTTP 405 for invalid methods on existing endpoints.

---

## Security Compliance Checklist

### ✅ Completed Security Measures

| Category | Measure | Implementation | Status |
|----------|---------|----------------|--------|
| **Authentication** | JWT-based auth | middleware/auth.go | ✅ |
| | Token rotation | auth.go:369-387 | ✅ |
| | Password hashing (bcrypt) | models/user.go | ✅ |
| | Account lockout | models/user.go:98-105 | ✅ |
| | Secure cookies | auth.go:261-314 | ✅ |
| | CSRF protection | auth.go:349-382 | ✅ |
| **Authorization** | RBAC (3 roles) | middleware/auth.go:502-558 | ✅ |
| | DELETE protection | server.go:165,182,216 | ✅ Task 67 |
| **Input Validation** | SQL injection check | middleware/validation.go:266-284 | ✅ Task 60 |
| | XSS protection | middleware/validation.go:286-304 | ✅ Task 60 |
| | 405 Method handling | server.go:97-99 | ✅ Task 60 |
| | Field validation | validation.go:103-197 | ✅ Task 60 |
| **GPIO Safety** | Pin safelist | config/gpio_config.go | ✅ Task 61 |
| | System pin blocking | gpio_config.go:12-19 | ✅ Task 61 |
| | Pin validation | services/gpio.go:196-202 | ✅ Task 61 |
| **Rate Limiting** | Global API limits | middleware/ratelimit.go | ✅ |
| | GPIO-specific limits | middleware/gpio_ratelimit.go | ✅ Task 68 |
| | Hardware protection | server.go:196 | ✅ Task 68 |
| **TLS/HTTPS** | Auto-cert generation | tls/tls.go | ✅ |
| | Production cert support | tls/tls.go:30-80 | ✅ |
| | Secure cipher suites | tls/tls.go:223-234 | ✅ |
| | Certificate validation | tls/tls.go:192-220 | ✅ |
| **Logging** | Audit logging | auth.go:570-586 | ✅ |
| | Security events | Throughout | ✅ |
| **gRPC** | JWT authentication | grpc/server/auth_interceptor.go | ✅ |
| | Role-based access control | auth_interceptor.go:RequireRole | ✅ |
| | Health check bypass | auth_interceptor.go:DefaultSkipMethods | ✅ |
| **WebSocket** | Connection auth | websocket/auth.go | ✅ |
| | Message-based auth | auth.go:handleAuthMessage | ✅ |
| | Anonymous mode | auth.go:AllowAnonymous | ✅ |
| **Security Headers** | X-Content-Type-Options | middleware/security.go | ✅ |
| | X-Frame-Options | middleware/security.go | ✅ |
| | Content-Security-Policy | middleware/security.go | ✅ |
| | HSTS (production) | middleware/security.go | ✅ |

### ✅ TLS/HTTPS Implementation (COMPLETED)

**Implementation:** `internal/tls/tls.go`

**Features:**

- Auto-generation of self-signed certificates for development
- Production certificate support (load from files)
- ECDSA key generation (more efficient than RSA)
- TLS 1.2+ minimum with secure cipher suites
- Automatic certificate expiry checking and renewal
- Warning system for expired/expiring certificates

**Usage:**

**Development Mode:**

```yaml
app:
  environment: development
  data_dir: ./data

# TLS will auto-generate self-signed certificates
# No additional configuration needed
```

Certificates will be auto-generated at `./data/tls/server.crt` and `./data/tls/server.key`.

**Production Mode:**

```yaml
app:
  environment: production

api:
  tls_cert_file: /etc/pi-controller/tls/cert.pem
  tls_key_file: /etc/pi-controller/tls/key.pem
```

**Environment Variables:**

```bash
# Production TLS certificate paths
export TLS_CERT_FILE=/path/to/cert.pem
export TLS_KEY_FILE=/path/to/key.pem

# Or use Docker secrets
export TLS_CERT_FILE=/run/secrets/tls_cert
export TLS_KEY_FILE=/run/secrets/tls_key
```

**Certificate Requirements:**

- Valid x509 certificate in PEM format
- Private key in PEM format (RSA or ECDSA)
- Certificate must be valid (not expired)
- Must match configured hostnames

**Security Notes:**

- Auto-generated certificates are **ONLY** for development
- Production **MUST** use properly signed certificates (Let's Encrypt, corporate CA, etc.)
- Certificates are checked for expiry on startup
- Warning logged if certificate expires within 30 days

### 🔄 Additional Recommendations

1. **Token Blacklist** (Medium Priority)
   - Currently: tokens valid until expiry
   - Recommended: Add Redis-based token blacklist for logout
   - Prevents token reuse after logout

2. **Password Requirements** (Medium Priority)
   - Currently: minimum 6 characters
   - Recommended: Add complexity rules (uppercase, numbers, symbols)
   - Consider password strength meter

3. **2FA/MFA** (Low Priority)
   - Currently: Not implemented
   - Consider TOTP-based 2FA for admin accounts

4. **IP Whitelisting** (Optional)
   - Feature exists but disabled by default
   - Enable for production: `auth.EnableIPWhitelist: true`

---

## Configuration Example

### Secure Production Configuration

```yaml
api:
  host: 0.0.0.0
  port: 8443
  tls_cert_file: /etc/pi-controller/tls/cert.pem
  tls_key_file: /etc/pi-controller/tls/key.pem
  cors_enabled: true
  auth_enabled: true  # MUST BE TRUE

gpio:
  enabled: true
  mock_mode: false
  allowed_pins: [17, 27, 22, 23, 24, 25]  # Override defaults
  restricted_pins: [0, 1, 2, 3, 14, 15]   # System critical

# Environment variables
PI_CONTROLLER_ENVIRONMENT=production  # Enforces HTTPS
JWT_SECRET=<strong-random-secret>     # Set via env or file
JWT_SECRET_FILE=/etc/pi-controller/secrets/jwt.key

GPIO_SAFE_PINS=17,27,22,23,24,25
GPIO_DISABLE_SAFELIST=false  # Keep safelist enabled!
```

---

## Testing Security

### Run Security Test Suite

```bash
# Run all security tests
go test ./test/security/... -v

# Run specific security tests
go test ./test/security -run TestSecurity_NoAuthentication -v
go test ./test/security -run TestSecurity_GPIOSafety -v
```

### Expected Behavior (With Auth Enabled)

```bash
# Unauthenticated requests should fail
curl -X DELETE http://localhost:8080/api/v1/clusters/1
# → 401 Unauthorized

# Authenticated viewer cannot delete
curl -X DELETE -H "Authorization: Bearer <viewer-token>" http://localhost:8080/api/v1/clusters/1
# → 403 Forbidden (requires admin role)

# Admin can delete
curl -X DELETE -H "Authorization: Bearer <admin-token>" http://localhost:8080/api/v1/clusters/1
# → 200 OK

# System critical pin rejected
curl -X POST -H "Authorization: Bearer <operator-token>" \
  -d '{"pin_number": 0, "node_id": 1}' \
  http://localhost:8080/api/v1/gpio
# → 400 Bad Request: "GPIO pin 0 cannot be used: pin 0 is reserved for system use: I2C0 SDA"
```

---

## Security Contact

For security vulnerabilities, please DO NOT open a public issue. Instead:

1. Email: <security@example.com> (configure your email)
2. Provide detailed description
3. Include steps to reproduce
4. Allow 90 days for patch before disclosure

---

## TLS/HTTPS Quick Start

### Development

```bash
# TLS is automatically configured with self-signed certificates
export PI_CONTROLLER_ENVIRONMENT=development
./pi-controller

# Server starts on https://localhost:8080 with auto-generated cert
# Cert stored at ./data/tls/server.crt
```

### Production with Let's Encrypt

```bash
# Use certbot to obtain certificates
certbot certonly --standalone -d your-domain.com

# Configure pi-controller
export PI_CONTROLLER_ENVIRONMENT=production
export TLS_CERT_FILE=/etc/letsencrypt/live/your-domain.com/fullchain.pem
export TLS_KEY_FILE=/etc/letsencrypt/live/your-domain.com/privkey.pem

./pi-controller
```

### Production with Custom CA

```bash
# Use your corporate CA certificates
export PI_CONTROLLER_ENVIRONMENT=production
export TLS_CERT_FILE=/path/to/your/cert.pem
export TLS_KEY_FILE=/path/to/your/key.pem

./pi-controller
```

---

**Last Updated:** 2025-12-07
**Implemented Tasks:** 60, 61, 67, 68, TLS/HTTPS, gRPC Auth, WebSocket Auth, Security Headers
**Security Status:** Production-Ready
