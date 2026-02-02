package defaults

// WebUI server defaults
const (
	// WebUIEnabled indicates whether WebUI is enabled
	WebUIEnabled = true

	// WebUIHost is the default WebUI bind address
	WebUIHost = "0.0.0.0"

	// WebUIPort is the default WebUI port
	WebUIPort = 3000

	// WebUIStaticDir is the default static files directory
	WebUIStaticDir = "./web/dist"

	// WebUIIndexFile is the default index file
	WebUIIndexFile = "index.html"

	// WebUISPAMode indicates whether SPA mode is enabled
	WebUISPAMode = true
)

// WebUI Backend defaults
const (
	// WebUIBackendAPIURL is the default API URL
	WebUIBackendAPIURL = "http://localhost:8080"

	// WebUIBackendAPIInternalURL is the default internal API URL (for K8s)
	WebUIBackendAPIInternalURL = "http://pi-controller-api:8080"

	// WebUIBackendAPIPrefix is the default API prefix
	WebUIBackendAPIPrefix = "/api"

	// WebUIBackendGRPCURL is the default gRPC URL
	WebUIBackendGRPCURL = "localhost:9090"

	// WebUIBackendGRPCInternalURL is the default internal gRPC URL
	WebUIBackendGRPCInternalURL = "pi-controller-grpc:9090"

	// WebUIBackendGRPCWebEnabled indicates whether gRPC-Web is enabled
	WebUIBackendGRPCWebEnabled = false

	// WebUIBackendWebSocketURL is the default WebSocket URL
	WebUIBackendWebSocketURL = "ws://localhost:8081"

	// WebUIBackendWebSocketInternalURL is the default internal WebSocket URL
	WebUIBackendWebSocketInternalURL = "ws://pi-controller-ws:8081"

	// WebUIBackendWebSocketPath is the default WebSocket path
	WebUIBackendWebSocketPath = "/socket.io"

	// WebUIBackendTLSEnabled indicates whether TLS is enabled for backend
	WebUIBackendTLSEnabled = false
)

// WebUI Runtime Config defaults
const (
	// WebUIRuntimeConfigEnabled indicates whether runtime config is enabled
	WebUIRuntimeConfigEnabled = true

	// WebUIRuntimeConfigPath is the default runtime config path
	WebUIRuntimeConfigPath = "/config.js"
)

// WebUI Auth defaults
const (
	// WebUIAuthEnabled indicates whether auth is enabled
	WebUIAuthEnabled = true

	// WebUIAuthSessionSecretEnv is the env var for session secret
	WebUIAuthSessionSecretEnv = "WEBUI_SESSION_SECRET" //nolint:gosec // Not a hardcoded credential, just an env var name

	// WebUIAuthJWTSecretEnv is the env var for JWT secret
	WebUIAuthJWTSecretEnv = "WEBUI_JWT_SECRET" //nolint:gosec // Not a hardcoded credential, just an env var name

	// WebUIAuthSessionTimeout is the default session timeout
	WebUIAuthSessionTimeout = "24h"

	// WebUIAuthCookieSecure indicates whether secure cookies are used (production)
	WebUIAuthCookieSecure = true

	// WebUIAuthCookieSameSite is the default SameSite cookie setting
	WebUIAuthCookieSameSite = "strict"
)

// WebUI CORS defaults
const (
	// WebUICORSEnabled indicates whether CORS is enabled
	WebUICORSEnabled = true

	// WebUICORSCredentials indicates whether credentials are allowed
	WebUICORSCredentials = true

	// WebUICORSMaxAge is the default CORS max age
	WebUICORSMaxAge = "12h"
)

// WebUICORSAllowedOrigins contains default allowed origins
var WebUICORSAllowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:8080",
}

// WebUICORSAllowedMethods contains default allowed methods
var WebUICORSAllowedMethods = []string{
	"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS",
}

// WebUICORSAllowedHeaders contains default allowed headers
var WebUICORSAllowedHeaders = []string{
	"Authorization", "Content-Type", "X-Requested-With",
}

// WebUICORSExposedHeaders contains default exposed headers
var WebUICORSExposedHeaders = []string{
	"X-Total-Count", "X-Page",
}

// WebUI Features defaults
const (
	// WebUIFeatureGPIOControl indicates whether GPIO control is enabled
	WebUIFeatureGPIOControl = true

	// WebUIFeatureClusterManagement indicates whether cluster management is enabled
	WebUIFeatureClusterManagement = true

	// WebUIFeatureCertificateManagement indicates whether cert management is enabled
	WebUIFeatureCertificateManagement = true

	// WebUIFeatureRealTimeMetrics indicates whether real-time metrics are enabled
	WebUIFeatureRealTimeMetrics = true

	// WebUIFeatureNodeDiscovery indicates whether node discovery is enabled
	WebUIFeatureNodeDiscovery = true

	// WebUIFeatureAdvancedNetworking indicates whether advanced networking is enabled
	WebUIFeatureAdvancedNetworking = false

	// WebUIFeatureExperimental indicates whether experimental features are enabled
	WebUIFeatureExperimental = false
)

// WebUI Branding defaults
const (
	// WebUIBrandingTitle is the default UI title
	WebUIBrandingTitle = "Pi Controller"

	// WebUIBrandingPrimaryColor is the default primary color
	WebUIBrandingPrimaryColor = "#10b981"

	// WebUIBrandingTheme is the default theme
	WebUIBrandingTheme = "dark"
)

// WebUI Cache defaults
const (
	// WebUICacheEnabled indicates whether caching is enabled
	WebUICacheEnabled = true

	// WebUICacheStaticMaxAge is the max age for static assets (1 year)
	WebUICacheStaticMaxAge = "31536000"

	// WebUICacheHTMLMaxAge is the max age for HTML (no cache)
	WebUICacheHTMLMaxAge = "0"

	// WebUICacheAPIMaxAge is the max age for API responses (5 minutes)
	WebUICacheAPIMaxAge = "300"
)

// WebUI Compression defaults
const (
	// WebUICompressionEnabled indicates whether compression is enabled
	WebUICompressionEnabled = true

	// WebUICompressionLevel is the default compression level
	WebUICompressionLevel = 6

	// WebUICompressionMinSize is the minimum size for compression
	WebUICompressionMinSize = 1024
)

// WebUI Security defaults
const (
	// WebUISecurityHSTSEnabled indicates whether HSTS is enabled
	WebUISecurityHSTSEnabled = true

	// WebUISecurityHSTSMaxAge is the HSTS max age (1 year)
	WebUISecurityHSTSMaxAge = "31536000"

	// WebUISecurityFrameDeny indicates whether to deny framing
	WebUISecurityFrameDeny = true

	// WebUISecurityContentTypeNoSniff indicates whether to set nosniff
	WebUISecurityContentTypeNoSniff = true

	// WebUISecurityXSSProtection indicates whether XSS protection is enabled
	WebUISecurityXSSProtection = true

	// WebUISecurityCSPEnabled indicates whether CSP is enabled
	WebUISecurityCSPEnabled = false
)

// WebUI Rate Limit defaults
const (
	// WebUIRateLimitEnabled indicates whether rate limiting is enabled
	WebUIRateLimitEnabled = true

	// WebUIRateLimitRequestsPerMinute is the default requests per minute
	WebUIRateLimitRequestsPerMinute = 60

	// WebUIRateLimitBurstSize is the default burst size
	WebUIRateLimitBurstSize = 100
)

// WebUI Observability defaults
const (
	// WebUIObservabilityAccessLog indicates whether access logging is enabled
	WebUIObservabilityAccessLog = true

	// WebUIObservabilityAccessLogFormat is the default access log format
	WebUIObservabilityAccessLogFormat = "json"

	// WebUIObservabilityMetricsEnabled indicates whether metrics are enabled
	WebUIObservabilityMetricsEnabled = true

	// WebUIObservabilityMetricsPath is the default metrics path
	WebUIObservabilityMetricsPath = "/metrics"

	// WebUIObservabilityTracingEnabled indicates whether tracing is enabled
	WebUIObservabilityTracingEnabled = false
)

// WebUI Resources defaults (for K8s)
const (
	// WebUIResourcesCPULimit is the default CPU limit
	WebUIResourcesCPULimit = "500m"

	// WebUIResourcesMemoryLimit is the default memory limit
	WebUIResourcesMemoryLimit = "512Mi"

	// WebUIResourcesCPURequest is the default CPU request
	WebUIResourcesCPURequest = "100m"

	// WebUIResourcesMemoryRequest is the default memory request
	WebUIResourcesMemoryRequest = "128Mi"
)

// WebUI Health defaults
const (
	// WebUIHealthEnabled indicates whether health checks are enabled
	WebUIHealthEnabled = true

	// WebUIHealthPath is the default health check path
	WebUIHealthPath = "/health"

	// WebUIHealthLivenessPath is the default liveness path
	WebUIHealthLivenessPath = "/health/live"

	// WebUIHealthReadinessPath is the default readiness path
	WebUIHealthReadinessPath = "/health/ready"
)
