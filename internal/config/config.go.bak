package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/spenceryork/pi-controller/internal/storage"
)

// Config holds the entire application configuration
type Config struct {
	// Application settings
	App AppConfig `yaml:"app" mapstructure:"app"`
	
	// Database configuration
	Database storage.Config `yaml:"database" mapstructure:"database"`
	
	// API server configuration
	API APIConfig `yaml:"api" mapstructure:"api"`
	
	// gRPC server configuration  
	GRPC GRPCConfig `yaml:"grpc" mapstructure:"grpc"`
	
	// WebSocket configuration
	WebSocket WebSocketConfig `yaml:"websocket" mapstructure:"websocket"`
	
	// Logging configuration
	Log LogConfig `yaml:"log" mapstructure:"log"`
	
	// Kubernetes configuration
	Kubernetes KubernetesConfig `yaml:"kubernetes" mapstructure:"kubernetes"`
	
	// GPIO configuration
	GPIO GPIOConfig `yaml:"gpio" mapstructure:"gpio"`
	
	// Discovery configuration
	Discovery DiscoveryConfig `yaml:"discovery" mapstructure:"discovery"`
}

// AppConfig contains general application settings
type AppConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`
	Version     string `yaml:"version" mapstructure:"version"`
	Environment string `yaml:"environment" mapstructure:"environment"`
	DataDir     string `yaml:"data_dir" mapstructure:"data_dir"`
	Debug       bool   `yaml:"debug" mapstructure:"debug"`
}

// APIConfig contains REST API server settings
type APIConfig struct {
	Host         string `yaml:"host" mapstructure:"host"`
	Port         int    `yaml:"port" mapstructure:"port"`
	ReadTimeout  string `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout string `yaml:"write_timeout" mapstructure:"write_timeout"`
	TLSCertFile  string `yaml:"tls_cert_file" mapstructure:"tls_cert_file"`
	TLSKeyFile   string `yaml:"tls_key_file" mapstructure:"tls_key_file"`
	CORSEnabled  bool   `yaml:"cors_enabled" mapstructure:"cors_enabled"`
	AuthEnabled  bool   `yaml:"auth_enabled" mapstructure:"auth_enabled"`
}

// GRPCConfig contains gRPC server settings
type GRPCConfig struct {
	Host        string `yaml:"host" mapstructure:"host"`
	Port        int    `yaml:"port" mapstructure:"port"`
	TLSCertFile string `yaml:"tls_cert_file" mapstructure:"tls_cert_file"`
	TLSKeyFile  string `yaml:"tls_key_file" mapstructure:"tls_key_file"`
}

// WebSocketConfig contains WebSocket server settings
type WebSocketConfig struct {
	Host            string `yaml:"host" mapstructure:"host"`
	Port            int    `yaml:"port" mapstructure:"port"`
	Path            string `yaml:"path" mapstructure:"path"`
	ReadBufferSize  int    `yaml:"read_buffer_size" mapstructure:"read_buffer_size"`
	WriteBufferSize int    `yaml:"write_buffer_size" mapstructure:"write_buffer_size"`
	CheckOrigin     bool   `yaml:"check_origin" mapstructure:"check_origin"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`
	Format     string `yaml:"format" mapstructure:"format"`
	Output     string `yaml:"output" mapstructure:"output"`
	File       string `yaml:"file" mapstructure:"file"`
	MaxSize    int    `yaml:"max_size" mapstructure:"max_size"`
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"`
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`
	Compress   bool   `yaml:"compress" mapstructure:"compress"`
}

// KubernetesConfig contains Kubernetes client settings
type KubernetesConfig struct {
	ConfigPath     string `yaml:"config_path" mapstructure:"config_path"`
	InCluster      bool   `yaml:"in_cluster" mapstructure:"in_cluster"`
	Namespace      string `yaml:"namespace" mapstructure:"namespace"`
	ResyncInterval string `yaml:"resync_interval" mapstructure:"resync_interval"`
}

// GPIOConfig contains GPIO service settings
type GPIOConfig struct {
	Enabled           bool     `yaml:"enabled" mapstructure:"enabled"`
	MockMode          bool     `yaml:"mock_mode" mapstructure:"mock_mode"`
	SampleInterval    string   `yaml:"sample_interval" mapstructure:"sample_interval"`
	RetentionPeriod   string   `yaml:"retention_period" mapstructure:"retention_period"`
	AllowedPins       []int    `yaml:"allowed_pins" mapstructure:"allowed_pins"`
	RestrictedPins    []int    `yaml:"restricted_pins" mapstructure:"restricted_pins"`
	DefaultDirection  string   `yaml:"default_direction" mapstructure:"default_direction"`
	DefaultPullMode   string   `yaml:"default_pull_mode" mapstructure:"default_pull_mode"`
}

// DiscoveryConfig contains node discovery settings
type DiscoveryConfig struct {
	Enabled         bool     `yaml:"enabled" mapstructure:"enabled"`
	Method          string   `yaml:"method" mapstructure:"method"` // mdns, scan, static
	Interface       string   `yaml:"interface" mapstructure:"interface"`
	Port            int      `yaml:"port" mapstructure:"port"`
	Interval        string   `yaml:"interval" mapstructure:"interval"`
	Timeout         string   `yaml:"timeout" mapstructure:"timeout"`
	StaticNodes     []string `yaml:"static_nodes" mapstructure:"static_nodes"`
	ServiceName     string   `yaml:"service_name" mapstructure:"service_name"`
	ServiceType     string   `yaml:"service_type" mapstructure:"service_type"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Set defaults
	setDefaults()
	
	// Configure viper
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("PI_CONTROLLER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	
	// Load config file if provided
	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	} else {
		// Search for config file in standard locations
		viper.SetConfigName("pi-controller")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/pi-controller")
		viper.AddConfigPath("$HOME/.pi-controller")
		
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}
	
	// Unmarshal into config struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
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
	
	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "pi-controller")
	viper.SetDefault("app.version", "dev")
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("app.data_dir", "./data")
	viper.SetDefault("app.debug", false)
	
	// Database defaults
	viper.SetDefault("database.path", "pi-controller.db")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("database.log_level", "warn")
	
	// API defaults
	viper.SetDefault("api.host", "0.0.0.0")
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.read_timeout", "30s")
	viper.SetDefault("api.write_timeout", "30s")
	viper.SetDefault("api.cors_enabled", true)
	viper.SetDefault("api.auth_enabled", false)
	
	// gRPC defaults
	viper.SetDefault("grpc.host", "0.0.0.0")
	viper.SetDefault("grpc.port", 9090)
	
	// WebSocket defaults
	viper.SetDefault("websocket.host", "0.0.0.0")
	viper.SetDefault("websocket.port", 8081)
	viper.SetDefault("websocket.path", "/ws")
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)
	viper.SetDefault("websocket.check_origin", false)
	
	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)
	
	// Kubernetes defaults
	viper.SetDefault("kubernetes.config_path", "")
	viper.SetDefault("kubernetes.in_cluster", false)
	viper.SetDefault("kubernetes.namespace", "default")
	viper.SetDefault("kubernetes.resync_interval", "30s")
	
	// GPIO defaults
	viper.SetDefault("gpio.enabled", true)
	viper.SetDefault("gpio.mock_mode", false)
	viper.SetDefault("gpio.sample_interval", "1s")
	viper.SetDefault("gpio.retention_period", "24h")
	viper.SetDefault("gpio.default_direction", "input")
	viper.SetDefault("gpio.default_pull_mode", "none")
	
	// Discovery defaults
	viper.SetDefault("discovery.enabled", true)
	viper.SetDefault("discovery.method", "mdns")
	viper.SetDefault("discovery.port", 9091)
	viper.SetDefault("discovery.interval", "30s")
	viper.SetDefault("discovery.timeout", "5s")
	viper.SetDefault("discovery.service_name", "pi-controller")
	viper.SetDefault("discovery.service_type", "_pi-controller._tcp")
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