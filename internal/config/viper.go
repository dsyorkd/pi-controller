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

	// gRPC defaults
	v.SetDefault("grpc.host", defaults.GRPC.Host)
	v.SetDefault("grpc.port", defaults.GRPC.Port)

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
	v.SetDefault("gpio.mock_mode", defaults.GPIO.MockMode)

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