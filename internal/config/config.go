package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/dsyorkd/pi-controller/internal/storage"
)

// Config holds the entire application configuration
type Config struct {
	// Application settings
	App AppConfig `yaml:"app"`

	// Database configuration
	Database storage.Config `yaml:"database"`

	// API server configuration
	API APIConfig `yaml:"api"`

	// gRPC server configuration
	GRPC GRPCConfig `yaml:"grpc"`

	// WebSocket configuration
	WebSocket WebSocketConfig `yaml:"websocket"`

	// Logging configuration
	Log LogConfig `yaml:"log"`

	// Kubernetes configuration
	Kubernetes KubernetesConfig `yaml:"kubernetes"`

	// GPIO configuration
	GPIO GPIOConfig `yaml:"gpio"`

	// Discovery configuration
	Discovery DiscoveryConfig `yaml:"discovery"`

	// gRPC client configuration (for Pi Agent)
	GRPCClient GRPCClientConfig `yaml:"grpc_client"`

	// Pi Agent gRPC server configuration
	AgentServer AgentServerConfig `yaml:"agent_server"`

	// Certificate Authority configuration
	CA CAConfig `yaml:"ca"`

	// Sentry configuration
	Sentry SentryConfig `yaml:"sentry"`

	// Web UI configuration
	WebUI WebUIConfig `yaml:"webui"`
}

// AppConfig contains general application settings
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
	DataDir     string `yaml:"data_dir"`
	Debug       bool   `yaml:"debug"`
}

// APIConfig contains REST API server settings
type APIConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	ReadTimeout  string `yaml:"read_timeout"`
	WriteTimeout string `yaml:"write_timeout"`
	TLSCertFile  string `yaml:"tls_cert_file"`
	TLSKeyFile   string `yaml:"tls_key_file"`
	CORSEnabled  bool   `yaml:"cors_enabled"`
	AuthEnabled  bool   `yaml:"auth_enabled"`
}

// GRPCConfig contains gRPC server settings
type GRPCConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`
}

// WebSocketConfig contains WebSocket server settings
type WebSocketConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Path            string `yaml:"path"`
	ReadBufferSize  int    `yaml:"read_buffer_size"`
	WriteBufferSize int    `yaml:"write_buffer_size"`
	CheckOrigin     bool   `yaml:"check_origin"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
}

// KubernetesConfig contains Kubernetes client settings
type KubernetesConfig struct {
	ConfigPath     string `yaml:"config_path"`
	InCluster      bool   `yaml:"in_cluster"`
	Namespace      string `yaml:"namespace"`
	ResyncInterval string `yaml:"resync_interval"`
}

// GPIOConfig contains GPIO service settings
type GPIOConfig struct {
	Enabled          bool   `yaml:"enabled"`
	MockMode         bool   `yaml:"mock_mode"`
	SampleInterval   string `yaml:"sample_interval"`
	RetentionPeriod  string `yaml:"retention_period"`
	AllowedPins      []int  `yaml:"allowed_pins"`
	RestrictedPins   []int  `yaml:"restricted_pins"`
	DefaultDirection string `yaml:"default_direction"`
	DefaultPullMode  string `yaml:"default_pull_mode"`
}

// DiscoveryConfig contains node discovery settings
type DiscoveryConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Method      string   `yaml:"method"` // mdns, scan, static
	Interface   string   `yaml:"interface"`
	Port        int      `yaml:"port"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	StaticNodes []string `yaml:"static_nodes"`
	ServiceName string   `yaml:"service_name"`
	ServiceType string   `yaml:"service_type"`
}

// GRPCClientConfig contains gRPC client settings for Pi Agent
type GRPCClientConfig struct {
	// Server connection
	ServerAddress string `yaml:"server_address"`
	ServerPort    int    `yaml:"server_port"`

	// Connection settings
	ConnectionTimeout string `yaml:"connection_timeout"`
	RequestTimeout    string `yaml:"request_timeout"`
	MaxMessageSize    int    `yaml:"max_message_size"`

	// Retry configuration
	MaxRetries        int     `yaml:"max_retries"`
	InitialRetryDelay string  `yaml:"initial_retry_delay"`
	MaxRetryDelay     string  `yaml:"max_retry_delay"`
	RetryMultiplier   float64 `yaml:"retry_multiplier"`

	// Heartbeat settings
	HeartbeatInterval string `yaml:"heartbeat_interval"`
	HeartbeatTimeout  string `yaml:"heartbeat_timeout"`

	// Keep-alive settings
	KeepAliveTime    string `yaml:"keepalive_time"`
	KeepAliveTimeout string `yaml:"keepalive_timeout"`

	// Security
	Insecure bool   `yaml:"insecure"`
	TLSCert  string `yaml:"tls_cert"`
	TLSKey   string `yaml:"tls_key"`

	// Node information
	NodeID   string `yaml:"node_id"`
	NodeName string `yaml:"node_name"`
}

// CAConfig contains Certificate Authority settings
type CAConfig struct {
	// CA backend type: "local" or "vault"
	Backend string `yaml:"backend"`

	// Local CA configuration
	Local LocalCAConfig `yaml:"local"`

	// Vault CA configuration
	Vault VaultCAConfig `yaml:"vault"`

	// SSH configuration for remote certificate operations
	SSH SSHConfig `yaml:"ssh"`

	// Certificate settings
	CertificateConfig CertificateConfig `yaml:"certificate"`
}

// LocalCAConfig contains local CA backend settings
type LocalCAConfig struct {
	// Directory to store CA certificates and keys (on server nodes)
	DataDir string `yaml:"data_dir"`

	// CA certificate validity period
	CAValidityPeriod string `yaml:"ca_validity_period"`

	// Default certificate validity period for issued certificates
	CertValidityPeriod string `yaml:"cert_validity_period"`

	// Key size for RSA keys
	KeySize int `yaml:"key_size"`

	// Organization information for CA certificate
	Organization       string `yaml:"organization"`
	OrganizationalUnit string `yaml:"organizational_unit"`
	Country            string `yaml:"country"`
	Province           string `yaml:"province"`
	Locality           string `yaml:"locality"`
}

// VaultCAConfig contains Vault PKI backend settings
type VaultCAConfig struct {
	// Vault server address
	Address string `yaml:"address"`

	// PKI mount path
	MountPath string `yaml:"mount_path"`

	// AppRole authentication settings
	AppRoleID   string `yaml:"app_role_id"`
	SecretID    string `yaml:"secret_id"`
	SecretIDEnv string `yaml:"secret_id_env"` // Environment variable for secret_id

	// Admin token for initial setup (dev only)
	AdminToken    string `yaml:"admin_token"`
	AdminTokenEnv string `yaml:"admin_token_env"` // Environment variable for admin_token

	// Connection settings
	Timeout   string         `yaml:"timeout"`
	TLSConfig VaultTLSConfig `yaml:"tls"`

	// Certificate role name in Vault
	CertRole string `yaml:"cert_role"`

	// Allow insecure connections (dev only)
	AllowInsecure bool `yaml:"allow_insecure"`
}

// VaultTLSConfig contains TLS settings for Vault connection
type VaultTLSConfig struct {
	// Skip TLS verification (dev only)
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	CACert             string `yaml:"ca_cert"`
	ClientCert         string `yaml:"client_cert"`
	ClientKey          string `yaml:"client_key"`
}

// SSHConfig contains SSH settings for remote CA operations
type SSHConfig struct {
	// SSH key for authenticating to server nodes
	PrivateKeyPath string `yaml:"private_key_path"`
	PrivateKeyEnv  string `yaml:"private_key_env"` // Environment variable for private key

	// Default SSH user for server nodes
	User string `yaml:"user"`

	// Default SSH port
	Port int `yaml:"port"`

	// Connection timeout
	Timeout string `yaml:"timeout"`

	// Host key checking
	StrictHostKeyChecking bool `yaml:"strict_host_key_checking"`

	// Known hosts file path
	KnownHostsFile string `yaml:"known_hosts_file"`
}

// CertificateConfig contains general certificate settings
type CertificateConfig struct {
	// Default certificate validity period
	DefaultValidityPeriod string `yaml:"default_validity_period"`

	// Certificate renewal threshold (renew when this much time is left)
	RenewalThreshold string `yaml:"renewal_threshold"`

	// Key usage settings
	DefaultKeyUsage    []string `yaml:"default_key_usage"`
	DefaultExtKeyUsage []string `yaml:"default_ext_key_usage"`

	// Subject Alternative Name settings
	AllowWildcardDNS bool     `yaml:"allow_wildcard_dns"`
	AllowedDomains   []string `yaml:"allowed_domains"`

	// Certificate storage and cleanup
	StoragePath     string `yaml:"storage_path"`     // Path to store certificates on control machine
	CleanupInterval string `yaml:"cleanup_interval"` // How often to clean up expired certificates
	RetentionPeriod string `yaml:"retention_period"` // How long to keep expired certificates
}

// AgentServerConfig contains Pi Agent gRPC server settings
type AgentServerConfig struct {
	// Server settings
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`

	// Service settings
	EnableGPIO bool `yaml:"enable_gpio"`

	// Security
	TLSCertFile string `yaml:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file"`
}

// SentryConfig contains Sentry error tracking and performance monitoring settings
type SentryConfig struct {
	// Sentry DSN (Data Source Name)
	DSN string `yaml:"dsn"`

	// Environment name (e.g., development, staging, production)
	Environment string `yaml:"environment"`

	// Release version for tracking
	Release string `yaml:"release"`

	// Enable debug mode for Sentry SDK
	Debug bool `yaml:"debug"`

	// Sample rate for performance monitoring (0.0 to 1.0)
	TracesSampleRate float64 `yaml:"traces_sample_rate"`

	// Sample rate for errors (0.0 to 1.0)
	SampleRate float64 `yaml:"sample_rate"`

	// Enable performance monitoring
	EnableTracing bool `yaml:"enable_tracing"`

	// Send PII (Personally Identifiable Information)
	SendDefaultPII bool `yaml:"send_default_pii"`

	// Maximum number of breadcrumbs
	MaxBreadcrumbs int `yaml:"max_breadcrumbs"`

	// Attach stack traces to messages
	AttachStacktrace bool `yaml:"attach_stacktrace"`
}

// WebUIConfig contains web UI server settings
type WebUIConfig struct {
	// Server settings
	Enabled   bool   `yaml:"enabled"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	StaticDir string `yaml:"static_dir"`
	IndexFile string `yaml:"index_file"`
	SPAMode   bool   `yaml:"spa_mode"`

	// Backend connection
	Backend BackendConfig `yaml:"backend"`

	// Runtime config injection
	RuntimeConfig RuntimeConfigSettings `yaml:"runtime_config"`

	// Authentication
	Auth WebAuthConfig `yaml:"auth"`

	// CORS configuration
	CORS CORSConfig `yaml:"cors"`

	// Feature flags
	Features FeatureFlags `yaml:"features"`

	// UI branding
	Branding BrandingConfig `yaml:"branding"`

	// Performance & caching
	Cache CacheConfig `yaml:"cache"`

	// Compression
	Compression CompressionConfig `yaml:"compression"`

	// Security headers
	Security SecurityConfig `yaml:"security"`

	// Rate limiting
	RateLimit RateLimitConfig `yaml:"rate_limit"`

	// Observability
	Observability ObservabilityConfig `yaml:"observability"`

	// Resource limits (for K8s)
	Resources ResourcesConfig `yaml:"resources"`

	// Health checks
	Health HealthConfig `yaml:"health"`
}

// BackendConfig contains backend service URLs
type BackendConfig struct {
	API       BackendServiceConfig `yaml:"api"`
	GRPC      BackendServiceConfig `yaml:"grpc"`
	WebSocket BackendServiceConfig `yaml:"websocket"`
	TLS       TLSConfig            `yaml:"tls"`
}

// BackendServiceConfig contains configuration for a backend service
type BackendServiceConfig struct {
	URL         string `yaml:"url"`
	InternalURL string `yaml:"internal_url"`
	Prefix      string `yaml:"prefix"`
	Path        string `yaml:"path"`
	WebEnabled  bool   `yaml:"web_enabled"`
}

// TLSConfig contains TLS settings
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// RuntimeConfigSettings controls runtime config injection
type RuntimeConfigSettings struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// WebAuthConfig contains web UI authentication settings
type WebAuthConfig struct {
	Enabled          bool   `yaml:"enabled"`
	SessionSecretEnv string `yaml:"session_secret_env"`
	JWTSecretEnv     string `yaml:"jwt_secret_env"`
	SessionTimeout   string `yaml:"session_timeout"`
	CookieSecure     bool   `yaml:"cookie_secure"`
	CookieSameSite   string `yaml:"cookie_same_site"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	ExposedHeaders []string `yaml:"exposed_headers"`
	Credentials    bool     `yaml:"credentials"`
	MaxAge         string   `yaml:"max_age"`
}

// FeatureFlags contains UI feature toggles
type FeatureFlags struct {
	GPIOControl           bool `yaml:"gpio_control"`
	ClusterManagement     bool `yaml:"cluster_management"`
	CertificateManagement bool `yaml:"certificate_management"`
	RealTimeMetrics       bool `yaml:"real_time_metrics"`
	NodeDiscovery         bool `yaml:"node_discovery"`
	AdvancedNetworking    bool `yaml:"advanced_networking"`
	Experimental          bool `yaml:"experimental"`
}

// BrandingConfig contains UI customization settings
type BrandingConfig struct {
	Title        string `yaml:"title"`
	LogoURL      string `yaml:"logo_url"`
	FaviconURL   string `yaml:"favicon_url"`
	PrimaryColor string `yaml:"primary_color"`
	Theme        string `yaml:"theme"`
}

// CacheConfig contains caching settings
type CacheConfig struct {
	Enabled        bool   `yaml:"enabled"`
	StaticMaxAge   string `yaml:"static_max_age"`
	HTMLMaxAge     string `yaml:"html_max_age"`
	APICacheMaxAge string `yaml:"api_cache_max_age"`
}

// CompressionConfig contains compression settings
type CompressionConfig struct {
	Enabled bool `yaml:"enabled"`
	Level   int  `yaml:"level"`
	MinSize int  `yaml:"min_size"`
}

// SecurityConfig contains security header settings
type SecurityConfig struct {
	HSTSEnabled         bool   `yaml:"hsts_enabled"`
	HSTSMaxAge          string `yaml:"hsts_max_age"`
	FrameDeny           bool   `yaml:"frame_deny"`
	ContentTypeNoSniff  bool   `yaml:"content_type_nosniff"`
	XSSProtection       bool   `yaml:"xss_protection"`
	CSPEnabled          bool   `yaml:"csp_enabled"`
	CSPDirectives       string `yaml:"csp_directives"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled             bool `yaml:"enabled"`
	RequestsPerMinute   int  `yaml:"requests_per_minute"`
	BurstSize           int  `yaml:"burst_size"`
}

// ObservabilityConfig contains observability settings
type ObservabilityConfig struct {
	AccessLog         bool   `yaml:"access_log"`
	AccessLogFormat   string `yaml:"access_log_format"`
	MetricsEnabled    bool   `yaml:"metrics_enabled"`
	MetricsPath       string `yaml:"metrics_path"`
	TracingEnabled    bool   `yaml:"tracing_enabled"`
}

// ResourcesConfig contains resource limit settings
type ResourcesConfig struct {
	CPULimit      string `yaml:"cpu_limit"`
	MemoryLimit   string `yaml:"memory_limit"`
	CPURequest    string `yaml:"cpu_request"`
	MemoryRequest string `yaml:"memory_request"`
}

// HealthConfig contains health check settings
type HealthConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Path          string `yaml:"path"`
	LivenessPath  string `yaml:"liveness_path"`
	ReadinessPath string `yaml:"readiness_path"`
}

// Load loads configuration from YAML file with defaults
func Load(configPath string) (*Config, error) {
	// Start with defaults
	config := getDefaults()

	// Load config file if provided or found
	var configFile string
	if configPath != "" {
		configFile = configPath
	} else {
		// Search for config file in standard locations
		searchPaths := []string{
			"./pi-controller.yaml",
			"./config/pi-controller.yaml",
			"/etc/pi-controller/pi-controller.yaml",
			filepath.Join(os.Getenv("HOME"), ".pi-controller", "pi-controller.yaml"),
		}

		for _, path := range searchPaths {
			if _, err := os.Stat(path); err == nil {
				configFile = path
				break
			}
		}
	}

	// Read and parse config file if found
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configFile, err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(&config)

	// Validate and set derived values
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validate validates the configuration and sets derived values
func (c *Config) validate() error {
	// Ensure data directory exists
	if c.App.DataDir != "" {
		if err := os.MkdirAll(c.App.DataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}

		// Set database path relative to data directory if not absolute
		if !filepath.IsAbs(c.Database.Path) {
			c.Database.Path = filepath.Join(c.App.DataDir, c.Database.Path)
		}
	}

	// Validate log level
	if _, err := logrus.ParseLevel(c.Log.Level); err != nil {
		return fmt.Errorf("invalid log level '%s': %w", c.Log.Level, err)
	}

	// Validate port ranges
	if c.API.Port < 1 || c.API.Port > 65535 {
		return fmt.Errorf("invalid API port: %d", c.API.Port)
	}
	if c.GRPC.Port < 1 || c.GRPC.Port > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPC.Port)
	}
	if c.WebSocket.Port < 1 || c.WebSocket.Port > 65535 {
		return fmt.Errorf("invalid WebSocket port: %d", c.WebSocket.Port)
	}

	// Set default Sentry environment and release if not specified
	if c.Sentry.Environment == "" {
		c.Sentry.Environment = c.App.Environment
	}
	if c.Sentry.Release == "" {
		c.Sentry.Release = c.App.Version
	}

	// Validate Sentry sample rates
	if c.Sentry.TracesSampleRate < 0.0 || c.Sentry.TracesSampleRate > 1.0 {
		return fmt.Errorf("invalid Sentry traces sample rate: %f (must be between 0.0 and 1.0)", c.Sentry.TracesSampleRate)
	}
	if c.Sentry.SampleRate < 0.0 || c.Sentry.SampleRate > 1.0 {
		return fmt.Errorf("invalid Sentry sample rate: %f (must be between 0.0 and 1.0)", c.Sentry.SampleRate)
	}

	return nil
}

// getDefaults returns a Config struct with default values based on environment
func getDefaults() Config {
	env := os.Getenv("PI_CONTROLLER_ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}

	// Use secure production defaults unless explicitly set to development
	if env == "development" || env == "dev" {
		return getDevelopmentDefaults()
	}
	return getProductionDefaults()
}

// getDevelopmentDefaults returns development-friendly defaults (less secure, easier setup)
func getDevelopmentDefaults() Config {
	config := getProductionDefaults()

	// Disable TLS for development ease
	config.API.TLSCertFile = ""
	config.API.TLSKeyFile = ""
	config.GRPC.TLSCertFile = ""
	config.GRPC.TLSKeyFile = ""

	// Development-specific CA settings
	config.CA.Local.DataDir = "./data/ca"
	config.CA.SSH.StrictHostKeyChecking = false
	config.CA.SSH.KnownHostsFile = ""
	config.CA.Vault.AllowInsecure = true
	config.CA.Vault.TLSConfig.InsecureSkipVerify = true

	// Development-specific WebUI settings
	config.WebUI.Auth.CookieSecure = false
	config.WebUI.Backend.TLS.Enabled = false
	config.WebUI.Security.HSTSEnabled = false

	return config
}

// getProductionDefaults returns secure production defaults
func getProductionDefaults() Config {
	return Config{
		App: AppConfig{
			Name:        "pi-controller",
			Version:     "dev",
			Environment: "development",
			DataDir:     "./data",
			Debug:       false,
		},
		Database: storage.Config{
			Path:            "pi-controller.db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: "5m",
			LogLevel:        "warn",
		},
		API: APIConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  "30s",
			WriteTimeout: "30s",
			TLSCertFile:  "/etc/pi-controller/tls/server.crt", // Default TLS cert path for production
			TLSKeyFile:   "/etc/pi-controller/tls/server.key", // Default TLS key path for production
			CORSEnabled:  true,
			AuthEnabled:  true, // Enable authentication by default for security
		},
		GRPC: GRPCConfig{
			Host:        "0.0.0.0",
			Port:        9090,
			TLSCertFile: "/etc/pi-controller/tls/server.crt", // Default TLS cert path for production
			TLSKeyFile:  "/etc/pi-controller/tls/server.key", // Default TLS key path for production
		},
		WebSocket: WebSocketConfig{
			Host:            "0.0.0.0",
			Port:            8081,
			Path:            "/ws",
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     false,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
		Kubernetes: KubernetesConfig{
			InCluster:      false,
			Namespace:      "default",
			ResyncInterval: "30s",
		},
		GPIO: GPIOConfig{
			Enabled:          true,
			MockMode:         false,
			SampleInterval:   "1s",
			RetentionPeriod:  "24h",
			AllowedPins:      []int{2, 3, 4, 17, 27, 22, 10, 9, 11, 5, 6, 13, 19, 26, 18, 23, 24, 25, 8, 7, 12, 16, 20, 21}, // Safe GPIO pins
			RestrictedPins:   []int{0, 1, 14, 15},                                                                           // System critical pins (I2C, UART)
			DefaultDirection: "input",
			DefaultPullMode:  "none",
		},
		Discovery: DiscoveryConfig{
			Enabled:     true,
			Method:      "mdns",
			Port:        9091,
			Interval:    "30s",
			Timeout:     "5s",
			ServiceName: "pi-controller",
			ServiceType: "_pi-controller._tcp",
		},
		GRPCClient: GRPCClientConfig{
			ServerAddress:     "localhost",
			ServerPort:        9090,
			ConnectionTimeout: "10s",
			RequestTimeout:    "30s",
			MaxMessageSize:    4 * 1024 * 1024, // 4MB
			MaxRetries:        5,
			InitialRetryDelay: "1s",
			MaxRetryDelay:     "60s",
			RetryMultiplier:   2.0,
			HeartbeatInterval: "30s",
			HeartbeatTimeout:  "5s",
			KeepAliveTime:     "30s",
			KeepAliveTimeout:  "5s",
			Insecure:          true,
			NodeID:            "",
			NodeName:          "",
		},
		AgentServer: AgentServerConfig{
			Address:    "0.0.0.0",
			Port:       9091,
			EnableGPIO: true,
		},
		CA: CAConfig{
			Backend: "local", // Default to local CA for development
			Local: LocalCAConfig{
				DataDir:            "/etc/pi-controller/ca",
				CAValidityPeriod:   "87600h", // 10 years
				CertValidityPeriod: "8760h",  // 1 year
				KeySize:            2048,
				Organization:       "Pi Controller",
				OrganizationalUnit: "Infrastructure",
				Country:            "US",
				Province:           "CA",
				Locality:           "San Francisco",
			},
			Vault: VaultCAConfig{
				Address:       "https://vault.example.com:8200",
				MountPath:     "pki",
				Timeout:       "30s",
				CertRole:      "pi-controller",
				AllowInsecure: false,
				TLSConfig: VaultTLSConfig{
					InsecureSkipVerify: false,
				},
			},
			SSH: SSHConfig{
				User:                  "pi",
				Port:                  22,
				Timeout:               "30s",
				StrictHostKeyChecking: true,
				KnownHostsFile:        "/etc/pi-controller/known_hosts",
			},
			CertificateConfig: CertificateConfig{
				DefaultValidityPeriod: "8760h", // 1 year
				RenewalThreshold:      "720h",  // 30 days
				DefaultKeyUsage: []string{
					"digital_signature",
					"key_encipherment",
				},
				DefaultExtKeyUsage: []string{
					"server_auth",
					"client_auth",
				},
				AllowWildcardDNS: false,
				AllowedDomains:   []string{"*.pi-controller.local", "*.cluster.local"},
				StoragePath:      "./data/certificates",
				CleanupInterval:  "24h",
				RetentionPeriod:  "2160h", // 90 days
			},
		},
		Sentry: SentryConfig{
			DSN:              "", // Must be set via environment variable
			Environment:      "", // Will be set from App.Environment if not specified
			Release:          "", // Will be set from App.Version if not specified
			Debug:            false,
			TracesSampleRate: 0.1, // 10% sampling for performance
			SampleRate:       1.0, // 100% error sampling
			EnableTracing:    true,
			SendDefaultPII:   false, // Security: don't send PII by default
			MaxBreadcrumbs:   100,
			AttachStacktrace: true,
		},
		WebUI: WebUIConfig{
			Enabled:   true,
			Host:      "0.0.0.0",
			Port:      3000,
			StaticDir: "./web/dist",
			IndexFile: "index.html",
			SPAMode:   true,
			Backend: BackendConfig{
				API: BackendServiceConfig{
					URL:         "http://localhost:8080",
					InternalURL: "http://pi-controller-api:8080",
					Prefix:      "/api",
				},
				GRPC: BackendServiceConfig{
					URL:         "localhost:9090",
					InternalURL: "pi-controller-grpc:9090",
					WebEnabled:  false,
				},
				WebSocket: BackendServiceConfig{
					URL:         "ws://localhost:8081",
					InternalURL: "ws://pi-controller-ws:8081",
					Path:        "/socket.io",
				},
				TLS: TLSConfig{
					Enabled: false,
				},
			},
			RuntimeConfig: RuntimeConfigSettings{
				Enabled: true,
				Path:    "/config.js",
			},
			Auth: WebAuthConfig{
				Enabled:          true,
				SessionSecretEnv: "WEBUI_SESSION_SECRET",
				JWTSecretEnv:     "WEBUI_JWT_SECRET",
				SessionTimeout:   "24h",
				CookieSecure:     true,
				CookieSameSite:   "strict",
			},
			CORS: CORSConfig{
				Enabled: true,
				AllowedOrigins: []string{
					"http://localhost:3000",
					"http://localhost:8080",
				},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
				AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With"},
				ExposedHeaders: []string{"X-Total-Count", "X-Page"},
				Credentials:    true,
				MaxAge:         "12h",
			},
			Features: FeatureFlags{
				GPIOControl:           true,
				ClusterManagement:     true,
				CertificateManagement: true,
				RealTimeMetrics:       true,
				NodeDiscovery:         true,
				AdvancedNetworking:    false,
				Experimental:          false,
			},
			Branding: BrandingConfig{
				Title:        "Pi Controller",
				LogoURL:      "",
				FaviconURL:   "",
				PrimaryColor: "#10b981",
				Theme:        "dark",
			},
			Cache: CacheConfig{
				Enabled:        true,
				StaticMaxAge:   "31536000", // 1 year
				HTMLMaxAge:     "0",
				APICacheMaxAge: "300", // 5 minutes
			},
			Compression: CompressionConfig{
				Enabled: true,
				Level:   6,
				MinSize: 1024,
			},
			Security: SecurityConfig{
				HSTSEnabled:        true,
				HSTSMaxAge:         "31536000",
				FrameDeny:          true,
				ContentTypeNoSniff: true,
				XSSProtection:      true,
				CSPEnabled:         false,
				CSPDirectives:      "",
			},
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
				BurstSize:         100,
			},
			Observability: ObservabilityConfig{
				AccessLog:       true,
				AccessLogFormat: "json",
				MetricsEnabled:  true,
				MetricsPath:     "/metrics",
				TracingEnabled:  false,
			},
			Resources: ResourcesConfig{
				CPULimit:      "500m",
				MemoryLimit:   "512Mi",
				CPURequest:    "100m",
				MemoryRequest: "128Mi",
			},
			Health: HealthConfig{
				Enabled:       true,
				Path:          "/health",
				LivenessPath:  "/health/live",
				ReadinessPath: "/health/ready",
			},
		},
	}
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(config *Config) {
	// Simple environment variable overrides for key settings
	if env := os.Getenv("PI_CONTROLLER_API_PORT"); env != "" {
		if port := parseIntEnv(env); port > 0 {
			config.API.Port = port
		}
	}
	if env := os.Getenv("PI_CONTROLLER_API_HOST"); env != "" {
		config.API.Host = env
	}
	if env := os.Getenv("PI_CONTROLLER_LOG_LEVEL"); env != "" {
		config.Log.Level = env
	}
	if env := os.Getenv("PI_CONTROLLER_DEBUG"); env == "true" {
		config.App.Debug = true
	}
	if env := os.Getenv("PI_CONTROLLER_DATA_DIR"); env != "" {
		config.App.DataDir = env
	}

	// Sentry configuration overrides
	if env := os.Getenv("SENTRY_DSN"); env != "" {
		config.Sentry.DSN = env
	}
	if env := os.Getenv("SENTRY_ENVIRONMENT"); env != "" {
		config.Sentry.Environment = env
	}
	if env := os.Getenv("SENTRY_RELEASE"); env != "" {
		config.Sentry.Release = env
	}
	if env := os.Getenv("SENTRY_DEBUG"); env == "true" {
		config.Sentry.Debug = true
	}
}

// parseIntEnv safely parses an integer from environment variable
func parseIntEnv(env string) int {
	var i int
	if _, err := fmt.Sscanf(env, "%d", &i); err == nil {
		return i
	}
	return 0
}

// GetAddress returns the formatted address for a service
func (c *APIConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetAddress returns the formatted address for gRPC service
func (c *GRPCConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetAddress returns the formatted address for WebSocket service
func (c *WebSocketConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsTLSEnabled returns true if TLS is configured for API
func (c *APIConfig) IsTLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// IsTLSEnabled returns true if TLS is configured for gRPC
func (c *GRPCConfig) IsTLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// GetAddress returns the formatted address for WebUI service
func (c *WebUIConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetBackendAPIURL returns the appropriate backend API URL based on environment
func (c *WebUIConfig) GetBackendAPIURL(k8sMode bool) string {
	if k8sMode {
		return c.Backend.API.InternalURL
	}
	return c.Backend.API.URL
}

// GetBackendWebSocketURL returns the appropriate backend WebSocket URL based on environment
func (c *WebUIConfig) GetBackendWebSocketURL(k8sMode bool) string {
	if k8sMode {
		return c.Backend.WebSocket.InternalURL
	}
	return c.Backend.WebSocket.URL
}

// GetBackendGRPCURL returns the appropriate backend gRPC URL based on environment
func (c *WebUIConfig) GetBackendGRPCURL(k8sMode bool) string {
	if k8sMode {
		return c.Backend.GRPC.InternalURL
	}
	return c.Backend.GRPC.URL
}
