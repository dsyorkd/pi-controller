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
	Enabled           bool     `yaml:"enabled"`
	MockMode          bool     `yaml:"mock_mode"`
	SampleInterval    string   `yaml:"sample_interval"`
	RetentionPeriod   string   `yaml:"retention_period"`
	AllowedPins       []int    `yaml:"allowed_pins"`
	RestrictedPins    []int    `yaml:"restricted_pins"`
	DefaultDirection  string   `yaml:"default_direction"`
	DefaultPullMode   string   `yaml:"default_pull_mode"`
}

// DiscoveryConfig contains node discovery settings
type DiscoveryConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Method          string   `yaml:"method"` // mdns, scan, static
	Interface       string   `yaml:"interface"`
	Port            int      `yaml:"port"`
	Interval        string   `yaml:"interval"`
	Timeout         string   `yaml:"timeout"`
	StaticNodes     []string `yaml:"static_nodes"`
	ServiceName     string   `yaml:"service_name"`
	ServiceType     string   `yaml:"service_type"`
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
	MaxRetries        int    `yaml:"max_retries"`
	InitialRetryDelay string `yaml:"initial_retry_delay"`
	MaxRetryDelay     string `yaml:"max_retry_delay"`
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
			AuthEnabled:  true,  // Enable authentication by default for security
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
			RestrictedPins:   []int{0, 1, 14, 15}, // System critical pins (I2C, UART)
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