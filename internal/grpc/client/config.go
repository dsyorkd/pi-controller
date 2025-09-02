package client

import (
	"fmt"
	"time"

	"github.com/dsyorkd/pi-controller/internal/config"
)

// ConfigFromYAML converts the YAML-based config to the gRPC client Config
func ConfigFromYAML(yamlConfig config.GRPCClientConfig) (Config, error) {
	clientConfig := Config{
		ServerAddress:   yamlConfig.ServerAddress,
		ServerPort:      yamlConfig.ServerPort,
		MaxMessageSize:  yamlConfig.MaxMessageSize,
		MaxRetries:      yamlConfig.MaxRetries,
		RetryMultiplier: yamlConfig.RetryMultiplier,
		Insecure:        yamlConfig.Insecure,
		TLSCert:         yamlConfig.TLSCert,
		TLSKey:          yamlConfig.TLSKey,
	}

	// Parse duration strings
	var err error
	
	if yamlConfig.ConnectionTimeout != "" {
		clientConfig.ConnectionTimeout, err = time.ParseDuration(yamlConfig.ConnectionTimeout)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid connection_timeout: %w", err)
		}
	}
	
	if yamlConfig.RequestTimeout != "" {
		clientConfig.RequestTimeout, err = time.ParseDuration(yamlConfig.RequestTimeout)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid request_timeout: %w", err)
		}
	}
	
	if yamlConfig.InitialRetryDelay != "" {
		clientConfig.InitialRetryDelay, err = time.ParseDuration(yamlConfig.InitialRetryDelay)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid initial_retry_delay: %w", err)
		}
	}
	
	if yamlConfig.MaxRetryDelay != "" {
		clientConfig.MaxRetryDelay, err = time.ParseDuration(yamlConfig.MaxRetryDelay)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid max_retry_delay: %w", err)
		}
	}
	
	if yamlConfig.HeartbeatInterval != "" {
		clientConfig.HeartbeatInterval, err = time.ParseDuration(yamlConfig.HeartbeatInterval)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid heartbeat_interval: %w", err)
		}
	}
	
	if yamlConfig.HeartbeatTimeout != "" {
		clientConfig.HeartbeatTimeout, err = time.ParseDuration(yamlConfig.HeartbeatTimeout)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid heartbeat_timeout: %w", err)
		}
	}
	
	if yamlConfig.KeepAliveTime != "" {
		clientConfig.KeepAliveTime, err = time.ParseDuration(yamlConfig.KeepAliveTime)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid keepalive_time: %w", err)
		}
	}
	
	if yamlConfig.KeepAliveTimeout != "" {
		clientConfig.KeepAliveTimeout, err = time.ParseDuration(yamlConfig.KeepAliveTimeout)
		if err != nil {
			return clientConfig, fmt.Errorf("invalid keepalive_timeout: %w", err)
		}
	}

	return clientConfig, nil
}

// ValidateConfig validates the gRPC client configuration
func ValidateConfig(config Config) error {
	if config.ServerAddress == "" {
		return fmt.Errorf("server_address is required")
	}
	
	if config.ServerPort <= 0 || config.ServerPort > 65535 {
		return fmt.Errorf("invalid server_port: %d", config.ServerPort)
	}
	
	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	
	if config.ConnectionTimeout <= 0 {
		return fmt.Errorf("connection_timeout must be positive")
	}
	
	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}
	
	if config.RetryMultiplier <= 1.0 {
		return fmt.Errorf("retry_multiplier must be greater than 1.0")
	}
	
	if config.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat_interval must be positive")
	}
	
	if config.HeartbeatTimeout <= 0 {
		return fmt.Errorf("heartbeat_timeout must be positive")
	}

	return nil
}