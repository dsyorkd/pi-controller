package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/dsyorkd/pi-controller/internal/config/defaults"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// LoadWithViper loads configuration using Viper with support for:
// - YAML configuration files
// - Environment variable overrides
// - Command-line flag binding
// - Hot-reload capability (when enabled)
func LoadWithViper(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults programmatically
	setViperDefaults(v)

	// Configure config file location
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("pi-controller")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/pi-controller")
		v.AddConfigPath("$HOME/.pi-controller")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// Environment variable support with prefix
	v.SetEnvPrefix("PI_CONTROLLER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind nested environment variables
	// This is needed because AutomaticEnv() doesn't automatically map nested struct fields
	bindNestedEnvVars(v)

	// Read config file (not required if all config from env/flags)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable - we can use env vars and defaults
	}

	// Unmarshal into Config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// LoadWithViperAndWatch loads configuration with hot-reload support
func LoadWithViperAndWatch(configPath string, onChange func(*Config)) (*Config, error) {
	v := viper.New()

	// Set defaults programmatically
	setViperDefaults(v)

	// Configure config file location
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("pi-controller")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/pi-controller")
		v.AddConfigPath("$HOME/.pi-controller")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// Environment variable support
	v.SetEnvPrefix("PI_CONTROLLER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind nested environment variables
	bindNestedEnvVars(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal initial config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate initial configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Set up file watcher for hot-reload
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		var newConfig Config
		if err := v.Unmarshal(&newConfig); err != nil {
			// Log error but don't crash
			fmt.Fprintf(os.Stderr, "Failed to reload config: %v\n", err)
			return
		}

		if err := newConfig.validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Reloaded config validation failed: %v\n", err)
			return
		}

		if onChange != nil {
			onChange(&newConfig)
		}
	})

	return &config, nil
}

// setViperDefaults sets default values in Viper using the defaults package
func setViperDefaults(v *viper.Viper) {
	// Determine environment
	env := os.Getenv("PI_CONTROLLER_ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
		if env == "" {
			env = defaults.AppEnvironment
		}
	}

	// App defaults
	v.SetDefault("app.name", defaults.AppName)
	v.SetDefault("app.version", defaults.AppVersion)
	v.SetDefault("app.environment", env)
	v.SetDefault("app.data_dir", defaults.AppDataDir)
	v.SetDefault("app.debug", defaults.AppDebug)

	// Database defaults
	v.SetDefault("database.path", defaults.DatabasePath)
	v.SetDefault("database.max_open_conns", defaults.DatabaseMaxOpenConns)
	v.SetDefault("database.max_idle_conns", defaults.DatabaseMaxIdleConns)
	v.SetDefault("database.conn_max_lifetime", defaults.DatabaseConnMaxLifetime)
	v.SetDefault("database.log_level", defaults.DatabaseLogLevel)

	// API defaults
	v.SetDefault("api.host", defaults.APIHost)
	v.SetDefault("api.port", defaults.APIPort)
	v.SetDefault("api.read_timeout", defaults.APIReadTimeout)
	v.SetDefault("api.write_timeout", defaults.APIWriteTimeout)
	v.SetDefault("api.cors_enabled", defaults.APICORSEnabled)
	v.SetDefault("api.auth_enabled", defaults.APIAuthEnabled)
	v.SetDefault("api.tls_cert_file", defaults.APITLSCertFile)
	v.SetDefault("api.tls_key_file", defaults.APITLSKeyFile)
	v.SetDefault("api.tls_ca_file", defaults.APITLSCAFile)

	// gRPC defaults
	v.SetDefault("grpc.host", defaults.GRPCHost)
	v.SetDefault("grpc.port", defaults.GRPCPort)
	v.SetDefault("grpc.tls_cert_file", defaults.GRPCTLSCertFile)
	v.SetDefault("grpc.tls_key_file", defaults.GRPCTLSKeyFile)
	v.SetDefault("grpc.tls_ca_file", defaults.GRPCTLSCAFile)

	// WebSocket defaults
	v.SetDefault("websocket.host", defaults.WebSocketHost)
	v.SetDefault("websocket.port", defaults.WebSocketPort)
	v.SetDefault("websocket.path", defaults.WebSocketPath)

	// Log defaults
	v.SetDefault("log.level", defaults.LogLevel)
	v.SetDefault("log.format", defaults.LogFormat)
	v.SetDefault("log.output", defaults.LogOutput)

	// GPIO defaults
	v.SetDefault("gpio.enabled", defaults.GPIOEnabled)
	v.SetDefault("agent_server.port", defaults.AgentServerPort)
	v.SetDefault("agent_server.enable_gpio", defaults.AgentServerEnableGPIO)
	v.SetDefault("agent_server.tls_cert_file", "")
	v.SetDefault("agent_server.tls_key_file", "")
	v.SetDefault("agent_server.tls_ca_file", "")

	// GRPC Client defaults
	v.SetDefault("grpc_client.server_address", defaults.GRPCClientServerAddress)
	v.SetDefault("grpc_client.server_port", defaults.GRPCClientServerPort)
	v.SetDefault("grpc_client.insecure", defaults.GRPCClientInsecure)
	v.SetDefault("grpc_client.tls_cert_file", "")
	v.SetDefault("grpc_client.tls_key_file", "")
	v.SetDefault("grpc_client.tls_ca_file", "")

	// Discovery defaults
	v.SetDefault("discovery.enabled", defaults.DiscoveryEnabled)
	v.SetDefault("discovery.method", defaults.DiscoveryMethod)
	v.SetDefault("discovery.port", defaults.DiscoveryPort)
	v.SetDefault("discovery.interval", defaults.DiscoveryInterval)
	v.SetDefault("discovery.timeout", defaults.DiscoveryTimeout)
	v.SetDefault("discovery.service_name", defaults.DiscoveryServiceName)
	v.SetDefault("discovery.service_type", defaults.DiscoveryServiceType)
	v.SetDefault("discovery.scan_ranges", []string{})
	v.SetDefault("discovery.scan_ports", defaults.DiscoveryScanPorts)
	v.SetDefault("discovery.scan_timeout", defaults.DiscoveryScanTimeout)
	v.SetDefault("discovery.scan_concurrency", defaults.DiscoveryScanConcurrency)
	v.SetDefault("discovery.scan_rate_limit", defaults.DiscoveryScanRateLimit)

	// Cluster defaults
	v.SetDefault("cluster.enabled", defaults.ClusterEnabled)
	v.SetDefault("cluster.portable", defaults.ClusterPortable)
	v.SetDefault("cluster.bootstrap", defaults.ClusterBootstrap)
	v.SetDefault("cluster.data_dir", defaults.ClusterDataDir)
	v.SetDefault("cluster.heartbeat_timeout", defaults.ClusterHeartbeatTimeout)
	v.SetDefault("cluster.election_timeout", defaults.ClusterElectionTimeout)
	v.SetDefault("cluster.snapshot_interval", defaults.ClusterSnapshotInterval)
	v.SetDefault("cluster.snapshot_threshold", defaults.ClusterSnapshotThreshold)
	v.SetDefault("cluster.max_append_entries", defaults.ClusterMaxAppendEntries)

	// WebUI defaults
	v.SetDefault("webui.enabled", defaults.WebUIEnabled)
	v.SetDefault("webui.host", defaults.WebUIHost)
	v.SetDefault("webui.port", defaults.WebUIPort)
	v.SetDefault("webui.static_dir", defaults.WebUIStaticDir)
	v.SetDefault("webui.index_file", defaults.WebUIIndexFile)
	v.SetDefault("webui.spa_mode", defaults.WebUISPAMode)

	// WebUI Backend defaults
	v.SetDefault("webui.backend.api.url", defaults.WebUIBackendAPIURL)
	v.SetDefault("webui.backend.api.internal_url", defaults.WebUIBackendAPIInternalURL)
	v.SetDefault("webui.backend.api.prefix", defaults.WebUIBackendAPIPrefix)
	v.SetDefault("webui.backend.grpc.url", defaults.WebUIBackendGRPCURL)
	v.SetDefault("webui.backend.grpc.internal_url", defaults.WebUIBackendGRPCInternalURL)
	v.SetDefault("webui.backend.websocket.url", defaults.WebUIBackendWebSocketURL)
	v.SetDefault("webui.backend.websocket.internal_url", defaults.WebUIBackendWebSocketInternalURL)
	v.SetDefault("webui.backend.websocket.path", defaults.WebUIBackendWebSocketPath)

	// WebUI Runtime Config defaults
	v.SetDefault("webui.runtime_config.enabled", defaults.WebUIRuntimeConfigEnabled)
	v.SetDefault("webui.runtime_config.path", defaults.WebUIRuntimeConfigPath)

	// WebUI Auth defaults
	v.SetDefault("webui.auth.enabled", defaults.WebUIAuthEnabled)
	v.SetDefault("webui.auth.session_secret_env", defaults.WebUIAuthSessionSecretEnv)
	v.SetDefault("webui.auth.jwt_secret_env", defaults.WebUIAuthJWTSecretEnv)
	v.SetDefault("webui.auth.session_timeout", defaults.WebUIAuthSessionTimeout)
	v.SetDefault("webui.auth.cookie_secure", defaults.WebUIAuthCookieSecure)
	v.SetDefault("webui.auth.cookie_same_site", defaults.WebUIAuthCookieSameSite)

	// WebUI CORS defaults
	v.SetDefault("webui.cors.enabled", defaults.WebUICORSEnabled)
	v.SetDefault("webui.cors.allowed_origins", defaults.WebUICORSAllowedOrigins)
	v.SetDefault("webui.cors.allowed_methods", defaults.WebUICORSAllowedMethods)
	v.SetDefault("webui.cors.allowed_headers", defaults.WebUICORSAllowedHeaders)
	v.SetDefault("webui.cors.credentials", defaults.WebUICORSCredentials)

	// WebUI Features defaults
	v.SetDefault("webui.features.gpio_control", defaults.WebUIFeatureGPIOControl)
	v.SetDefault("webui.features.cluster_management", defaults.WebUIFeatureClusterManagement)
	v.SetDefault("webui.features.certificate_management", defaults.WebUIFeatureCertificateManagement)
	v.SetDefault("webui.features.real_time_metrics", defaults.WebUIFeatureRealTimeMetrics)
	v.SetDefault("webui.features.node_discovery", defaults.WebUIFeatureNodeDiscovery)
	v.SetDefault("webui.features.advanced_networking", defaults.WebUIFeatureAdvancedNetworking)
	v.SetDefault("webui.features.experimental", defaults.WebUIFeatureExperimental)

	// WebUI Branding defaults
	v.SetDefault("webui.branding.title", defaults.WebUIBrandingTitle)
	v.SetDefault("webui.branding.primary_color", defaults.WebUIBrandingPrimaryColor)
	v.SetDefault("webui.branding.theme", defaults.WebUIBrandingTheme)

	// WebUI Cache defaults
	v.SetDefault("webui.cache.enabled", defaults.WebUICacheEnabled)
	v.SetDefault("webui.cache.static_max_age", defaults.WebUICacheStaticMaxAge)
	v.SetDefault("webui.cache.html_max_age", defaults.WebUICacheHTMLMaxAge)

	// WebUI Compression defaults
	v.SetDefault("webui.compression.enabled", defaults.WebUICompressionEnabled)
	v.SetDefault("webui.compression.level", defaults.WebUICompressionLevel)
	v.SetDefault("webui.compression.min_size", defaults.WebUICompressionMinSize)

	// WebUI Security defaults
	v.SetDefault("webui.security.hsts_enabled", defaults.WebUISecurityHSTSEnabled)
	v.SetDefault("webui.security.hsts_max_age", defaults.WebUISecurityHSTSMaxAge)
	v.SetDefault("webui.security.frame_deny", defaults.WebUISecurityFrameDeny)
	v.SetDefault("webui.security.content_type_nosniff", defaults.WebUISecurityContentTypeNoSniff)
	v.SetDefault("webui.security.xss_protection", defaults.WebUISecurityXSSProtection)
	v.SetDefault("webui.security.csp_enabled", defaults.WebUISecurityCSPEnabled)
	v.SetDefault("webui.security.csp_directives", "")

	// WebUI Rate Limit defaults
	v.SetDefault("webui.rate_limit.enabled", defaults.WebUIRateLimitEnabled)
	v.SetDefault("webui.rate_limit.requests_per_minute", defaults.WebUIRateLimitRequestsPerMinute)
	v.SetDefault("webui.rate_limit.burst_size", defaults.WebUIRateLimitBurstSize)

	// WebUI Health defaults
	v.SetDefault("webui.health.enabled", defaults.WebUIHealthEnabled)
	v.SetDefault("webui.health.path", defaults.WebUIHealthPath)
	v.SetDefault("webui.health.liveness_path", defaults.WebUIHealthLivenessPath)
	v.SetDefault("webui.health.readiness_path", defaults.WebUIHealthReadinessPath)

	// Development environment overrides - less secure settings for easier local development
	if env == "development" || env == "dev" {
		v.SetDefault("webui.auth.cookie_secure", false)
		v.SetDefault("webui.security.hsts_enabled", false)
		v.SetDefault("webui.backend.tls.enabled", false)
		v.SetDefault("api.tls_cert_file", "")
		v.SetDefault("api.tls_key_file", "")
		v.SetDefault("grpc.tls_cert_file", "")
		v.SetDefault("grpc.tls_key_file", "")
		v.SetDefault("ca.local.data_dir", "./data/ca")
		v.SetDefault("ca.ssh.strict_host_key_checking", false)
		v.SetDefault("ca.ssh.known_hosts_file", "")
		v.SetDefault("ca.vault.allow_insecure", true)
		v.SetDefault("ca.vault.tls_config.insecure_skip_verify", true)
	}
}

// bindEnv wraps viper.BindEnv with error handling
func bindEnv(v *viper.Viper, key string) {
	_ = v.BindEnv(key) // Errors are non-critical for env bindings
}

// bindNestedEnvVars explicitly binds all nested configuration fields to environment variables
// This is necessary because Viper's AutomaticEnv() doesn't automatically map nested struct fields
func bindNestedEnvVars(v *viper.Viper) {
	// App
	bindEnv(v, "app.name")
	bindEnv(v, "app.version")
	bindEnv(v, "app.environment")
	bindEnv(v, "app.data_dir")
	bindEnv(v, "app.debug")

	// Database
	bindEnv(v, "database.path")
	bindEnv(v, "database.max_open_conns")
	bindEnv(v, "database.max_idle_conns")
	bindEnv(v, "database.conn_max_lifetime")
	bindEnv(v, "database.log_level")

	// API
	bindEnv(v, "api.host")
	bindEnv(v, "api.port")
	bindEnv(v, "api.read_timeout")
	bindEnv(v, "api.write_timeout")
	bindEnv(v, "api.tls_cert_file")
	bindEnv(v, "api.tls_key_file")
	bindEnv(v, "api.tls_ca_file")
	bindEnv(v, "api.cors_enabled")
	bindEnv(v, "api.auth_enabled")

	// gRPC
	bindEnv(v, "grpc.host")
	bindEnv(v, "grpc.tls_key_file")
	bindEnv(v, "grpc.tls_ca_file")

	// WebSocket
	bindEnv(v, "websocket.host")
	bindEnv(v, "websocket.port")
	bindEnv(v, "websocket.path")

	// Log
	bindEnv(v, "log.level")
	bindEnv(v, "log.format")
	bindEnv(v, "log.output")

	// GPIO
	bindEnv(v, "gpio.enabled")
	bindEnv(v, "gpio.mock_mode")
	bindEnv(v, "gpio.sample_interval")
	bindEnv(v, "gpio.retention_period")
	bindEnv(v, "gpio.allowed_pins")
	bindEnv(v, "gpio.restricted_pins")
	bindEnv(v, "gpio.default_direction")
	bindEnv(v, "gpio.default_pull_mode")

	// Pi Agent gRPC server
	bindEnv(v, "agent_server.address")
	bindEnv(v, "agent_server.port")
	bindEnv(v, "agent_server.enable_gpio")
	bindEnv(v, "agent_server.tls_cert_file")
	bindEnv(v, "agent_server.tls_key_file")
	bindEnv(v, "agent_server.tls_ca_file")

	// gRPC Client
	bindEnv(v, "grpc_client.server_address")
	bindEnv(v, "grpc_client.server_port")
	bindEnv(v, "grpc_client.insecure")
	bindEnv(v, "grpc_client.tls_cert_file")
	bindEnv(v, "grpc_client.tls_key_file")
	bindEnv(v, "grpc_client.tls_ca_file")

	// Discovery
	bindEnv(v, "discovery.enabled")
	bindEnv(v, "discovery.method")
	bindEnv(v, "discovery.port")
	bindEnv(v, "discovery.interval")
	bindEnv(v, "discovery.timeout")
	bindEnv(v, "discovery.static_nodes")
	bindEnv(v, "discovery.service_name")
	bindEnv(v, "discovery.service_type")
	bindEnv(v, "discovery.scan_ranges")
	bindEnv(v, "discovery.scan_ports")
	bindEnv(v, "discovery.scan_timeout")
	bindEnv(v, "discovery.scan_concurrency")
	bindEnv(v, "discovery.scan_rate_limit")

	// Cluster
	bindEnv(v, "cluster.enabled")
	bindEnv(v, "cluster.portable")
	bindEnv(v, "cluster.controller_id")
	bindEnv(v, "cluster.bind_addr")
	bindEnv(v, "cluster.bootstrap")
	bindEnv(v, "cluster.initial_peers")
	bindEnv(v, "cluster.data_dir")
	bindEnv(v, "cluster.heartbeat_timeout")
	bindEnv(v, "cluster.election_timeout")
	bindEnv(v, "cluster.snapshot_interval")
	bindEnv(v, "cluster.snapshot_threshold")
	bindEnv(v, "cluster.max_append_entries")

	// WebUI
	bindEnv(v, "webui.enabled")
	bindEnv(v, "webui.host")
	bindEnv(v, "webui.port")
	bindEnv(v, "webui.static_dir")
	bindEnv(v, "webui.index_file")
	bindEnv(v, "webui.spa_mode")

	// WebUI Backend API
	bindEnv(v, "webui.backend.api.url")
	bindEnv(v, "webui.backend.api.internal_url")
	bindEnv(v, "webui.backend.api.prefix")

	// WebUI Backend gRPC
	bindEnv(v, "webui.backend.grpc.url")
	bindEnv(v, "webui.backend.grpc.internal_url")

	// WebUI Backend WebSocket
	bindEnv(v, "webui.backend.websocket.url")
	bindEnv(v, "webui.backend.websocket.internal_url")
	bindEnv(v, "webui.backend.websocket.path")

	// WebUI Runtime Config
	bindEnv(v, "webui.runtime_config.enabled")
	bindEnv(v, "webui.runtime_config.path")

	// WebUI Auth
	bindEnv(v, "webui.auth.enabled")
	bindEnv(v, "webui.auth.session_secret_env")
	bindEnv(v, "webui.auth.jwt_secret_env")
	bindEnv(v, "webui.auth.session_timeout")
	bindEnv(v, "webui.auth.cookie_secure")
	bindEnv(v, "webui.auth.cookie_same_site")

	// WebUI CORS
	bindEnv(v, "webui.cors.enabled")
	bindEnv(v, "webui.cors.allowed_origins")
	bindEnv(v, "webui.cors.allowed_methods")
	bindEnv(v, "webui.cors.allowed_headers")
	bindEnv(v, "webui.cors.credentials")

	// WebUI Features
	bindEnv(v, "webui.features.gpio_control")
	bindEnv(v, "webui.features.cluster_management")
	bindEnv(v, "webui.features.certificate_management")
	bindEnv(v, "webui.features.real_time_metrics")
	bindEnv(v, "webui.features.node_discovery")
	bindEnv(v, "webui.features.advanced_networking")
	bindEnv(v, "webui.features.experimental")

	// WebUI Branding
	bindEnv(v, "webui.branding.title")
	bindEnv(v, "webui.branding.primary_color")
	bindEnv(v, "webui.branding.theme")

	// WebUI Cache
	bindEnv(v, "webui.cache.enabled")
	bindEnv(v, "webui.cache.static_max_age")
	bindEnv(v, "webui.cache.html_max_age")

	// WebUI Compression
	bindEnv(v, "webui.compression.enabled")
	bindEnv(v, "webui.compression.level")
	bindEnv(v, "webui.compression.min_size")

	// WebUI Security
	bindEnv(v, "webui.security.hsts_enabled")
	bindEnv(v, "webui.security.hsts_max_age")
	bindEnv(v, "webui.security.frame_deny")
	bindEnv(v, "webui.security.content_type_nosniff")
	bindEnv(v, "webui.security.xss_protection")
	bindEnv(v, "webui.security.csp_enabled")
	bindEnv(v, "webui.security.csp_directives")

	// WebUI Rate Limit
	bindEnv(v, "webui.rate_limit.enabled")
	bindEnv(v, "webui.rate_limit.requests_per_minute")
	bindEnv(v, "webui.rate_limit.burst_size")

	// WebUI Health
	bindEnv(v, "webui.health.enabled")
	bindEnv(v, "webui.health.path")
	bindEnv(v, "webui.health.liveness_path")
	bindEnv(v, "webui.health.readiness_path")
}
