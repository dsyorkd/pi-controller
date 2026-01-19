package config

import (
	"fmt"
	"os"
	"strings"

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

// setViperDefaults sets default values in Viper
func setViperDefaults(v *viper.Viper) {
	// Determine environment
	env := os.Getenv("PI_CONTROLLER_ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
		if env == "" {
			env = "development"
		}
	}

	// Get defaults based on environment
	defaults := getDefaults()

	// App defaults
	v.SetDefault("app.name", defaults.App.Name)
	v.SetDefault("app.version", defaults.App.Version)
	v.SetDefault("app.environment", env)
	v.SetDefault("app.data_dir", defaults.App.DataDir)
	v.SetDefault("app.debug", defaults.App.Debug)

	// Database defaults
	v.SetDefault("database.path", defaults.Database.Path)
	v.SetDefault("database.max_open_conns", defaults.Database.MaxOpenConns)
	v.SetDefault("database.max_idle_conns", defaults.Database.MaxIdleConns)
	v.SetDefault("database.conn_max_lifetime", defaults.Database.ConnMaxLifetime)
	v.SetDefault("database.log_level", defaults.Database.LogLevel)

	// API defaults
	v.SetDefault("api.host", defaults.API.Host)
	v.SetDefault("api.port", defaults.API.Port)
	v.SetDefault("api.read_timeout", defaults.API.ReadTimeout)
	v.SetDefault("api.write_timeout", defaults.API.WriteTimeout)
	v.SetDefault("api.cors_enabled", defaults.API.CORSEnabled)
	v.SetDefault("api.auth_enabled", defaults.API.AuthEnabled)
	v.SetDefault("api.tls_cert_file", defaults.API.TLSCertFile)
	v.SetDefault("api.tls_key_file", defaults.API.TLSKeyFile)
	v.SetDefault("api.tls_ca_file", defaults.API.TLSCAFile)

	// gRPC defaults
	v.SetDefault("grpc.host", defaults.GRPC.Host)
	v.SetDefault("grpc.port", defaults.GRPC.Port)
	v.SetDefault("grpc.tls_cert_file", defaults.GRPC.TLSCertFile)
	v.SetDefault("grpc.tls_key_file", defaults.GRPC.TLSKeyFile)
	v.SetDefault("grpc.tls_ca_file", defaults.GRPC.TLSCAFile)

	// WebSocket defaults
	v.SetDefault("websocket.host", defaults.WebSocket.Host)
	v.SetDefault("websocket.port", defaults.WebSocket.Port)
	v.SetDefault("websocket.path", defaults.WebSocket.Path)

	// Log defaults
	v.SetDefault("log.level", defaults.Log.Level)
	v.SetDefault("log.format", defaults.Log.Format)
	v.SetDefault("log.output", defaults.Log.Output)

	// GPIO defaults
	v.SetDefault("gpio.enabled", defaults.GPIO.Enabled)
	v.SetDefault("agent_server.port", defaults.AgentServer.Port)
	v.SetDefault("agent_server.enable_gpio", defaults.AgentServer.EnableGPIO)
	v.SetDefault("agent_server.tls_cert_file", defaults.AgentServer.TLSCertFile)
	v.SetDefault("agent_server.tls_key_file", defaults.AgentServer.TLSKeyFile)
	v.SetDefault("agent_server.tls_ca_file", defaults.AgentServer.TLSCAFile)

	// GRPC Client defaults
	v.SetDefault("grpc_client.server_address", defaults.GRPCClient.ServerAddress)
	v.SetDefault("grpc_client.server_port", defaults.GRPCClient.ServerPort)
	v.SetDefault("grpc_client.insecure", defaults.GRPCClient.Insecure)
	v.SetDefault("grpc_client.tls_cert_file", defaults.GRPCClient.TLSCert)
	v.SetDefault("grpc_client.tls_key_file", defaults.GRPCClient.TLSKey)
	v.SetDefault("grpc_client.tls_ca_file", defaults.GRPCClient.TLSCAFile)

	// Discovery defaults
	v.SetDefault("discovery.enabled", defaults.Discovery.Enabled)
	v.SetDefault("discovery.method", defaults.Discovery.Method)
	v.SetDefault("discovery.port", defaults.Discovery.Port)

	// WebUI defaults
	v.SetDefault("webui.enabled", defaults.WebUI.Enabled)
	v.SetDefault("webui.host", defaults.WebUI.Host)
	v.SetDefault("webui.port", defaults.WebUI.Port)
	v.SetDefault("webui.static_dir", defaults.WebUI.StaticDir)
	v.SetDefault("webui.index_file", defaults.WebUI.IndexFile)
	v.SetDefault("webui.spa_mode", defaults.WebUI.SPAMode)

	// WebUI Backend defaults
	v.SetDefault("webui.backend.api.url", defaults.WebUI.Backend.API.URL)
	v.SetDefault("webui.backend.api.internal_url", defaults.WebUI.Backend.API.InternalURL)
	v.SetDefault("webui.backend.api.prefix", defaults.WebUI.Backend.API.Prefix)
	v.SetDefault("webui.backend.grpc.url", defaults.WebUI.Backend.GRPC.URL)
	v.SetDefault("webui.backend.grpc.internal_url", defaults.WebUI.Backend.GRPC.InternalURL)
	v.SetDefault("webui.backend.websocket.url", defaults.WebUI.Backend.WebSocket.URL)
	v.SetDefault("webui.backend.websocket.internal_url", defaults.WebUI.Backend.WebSocket.InternalURL)
	v.SetDefault("webui.backend.websocket.path", defaults.WebUI.Backend.WebSocket.Path)

	// WebUI Runtime Config defaults
	v.SetDefault("webui.runtime_config.enabled", defaults.WebUI.RuntimeConfig.Enabled)
	v.SetDefault("webui.runtime_config.path", defaults.WebUI.RuntimeConfig.Path)

	// WebUI Auth defaults
	v.SetDefault("webui.auth.enabled", defaults.WebUI.Auth.Enabled)
	v.SetDefault("webui.auth.session_secret_env", defaults.WebUI.Auth.SessionSecretEnv)
	v.SetDefault("webui.auth.jwt_secret_env", defaults.WebUI.Auth.JWTSecretEnv)
	v.SetDefault("webui.auth.session_timeout", defaults.WebUI.Auth.SessionTimeout)
	v.SetDefault("webui.auth.cookie_secure", defaults.WebUI.Auth.CookieSecure)
	v.SetDefault("webui.auth.cookie_same_site", defaults.WebUI.Auth.CookieSameSite)

	// WebUI CORS defaults
	v.SetDefault("webui.cors.enabled", defaults.WebUI.CORS.Enabled)
	v.SetDefault("webui.cors.allowed_origins", defaults.WebUI.CORS.AllowedOrigins)
	v.SetDefault("webui.cors.allowed_methods", defaults.WebUI.CORS.AllowedMethods)
	v.SetDefault("webui.cors.allowed_headers", defaults.WebUI.CORS.AllowedHeaders)
	v.SetDefault("webui.cors.credentials", defaults.WebUI.CORS.Credentials)

	// WebUI Features defaults
	v.SetDefault("webui.features.gpio_control", defaults.WebUI.Features.GPIOControl)
	v.SetDefault("webui.features.cluster_management", defaults.WebUI.Features.ClusterManagement)
	v.SetDefault("webui.features.certificate_management", defaults.WebUI.Features.CertificateManagement)
	v.SetDefault("webui.features.real_time_metrics", defaults.WebUI.Features.RealTimeMetrics)
	v.SetDefault("webui.features.node_discovery", defaults.WebUI.Features.NodeDiscovery)
	v.SetDefault("webui.features.advanced_networking", defaults.WebUI.Features.AdvancedNetworking)
	v.SetDefault("webui.features.experimental", defaults.WebUI.Features.Experimental)

	// WebUI Branding defaults
	v.SetDefault("webui.branding.title", defaults.WebUI.Branding.Title)
	v.SetDefault("webui.branding.primary_color", defaults.WebUI.Branding.PrimaryColor)
	v.SetDefault("webui.branding.theme", defaults.WebUI.Branding.Theme)

	// WebUI Cache defaults
	v.SetDefault("webui.cache.enabled", defaults.WebUI.Cache.Enabled)
	v.SetDefault("webui.cache.static_max_age", defaults.WebUI.Cache.StaticMaxAge)
	v.SetDefault("webui.cache.html_max_age", defaults.WebUI.Cache.HTMLMaxAge)

	// WebUI Compression defaults
	v.SetDefault("webui.compression.enabled", defaults.WebUI.Compression.Enabled)
	v.SetDefault("webui.compression.level", defaults.WebUI.Compression.Level)
	v.SetDefault("webui.compression.min_size", defaults.WebUI.Compression.MinSize)

	// WebUI Security defaults
	v.SetDefault("webui.security.hsts_enabled", defaults.WebUI.Security.HSTSEnabled)
	v.SetDefault("webui.security.hsts_max_age", defaults.WebUI.Security.HSTSMaxAge)
	v.SetDefault("webui.security.frame_deny", defaults.WebUI.Security.FrameDeny)
	v.SetDefault("webui.security.content_type_nosniff", defaults.WebUI.Security.ContentTypeNoSniff)
	v.SetDefault("webui.security.xss_protection", defaults.WebUI.Security.XSSProtection)
	v.SetDefault("webui.security.csp_enabled", defaults.WebUI.Security.CSPEnabled)
	v.SetDefault("webui.security.csp_directives", defaults.WebUI.Security.CSPDirectives)

	// WebUI Rate Limit defaults
	v.SetDefault("webui.rate_limit.enabled", defaults.WebUI.RateLimit.Enabled)
	v.SetDefault("webui.rate_limit.requests_per_minute", defaults.WebUI.RateLimit.RequestsPerMinute)
	v.SetDefault("webui.rate_limit.burst_size", defaults.WebUI.RateLimit.BurstSize)

	// WebUI Health defaults
	v.SetDefault("webui.health.enabled", defaults.WebUI.Health.Enabled)
	v.SetDefault("webui.health.path", defaults.WebUI.Health.Path)
	v.SetDefault("webui.health.liveness_path", defaults.WebUI.Health.LivenessPath)
	v.SetDefault("webui.health.readiness_path", defaults.WebUI.Health.ReadinessPath)
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
