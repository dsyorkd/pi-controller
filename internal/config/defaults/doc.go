// Package defaults provides default configuration values for the pi-controller application.
//
// This package centralizes all configuration defaults in domain-specific files,
// making them easy to find, modify, and test. The constants are used by the
// main config package to initialize configuration structures.
//
// Files:
//   - app.go: Application-level defaults (name, version, environment)
//   - database.go: SQLite database configuration
//   - api.go: REST API server settings
//   - grpc.go: gRPC server and client settings
//   - websocket.go: WebSocket server settings
//   - logging.go: Logging configuration
//   - kubernetes.go: Kubernetes integration settings
//   - gpio.go: GPIO pin control settings
//   - discovery.go: Node discovery settings (mDNS, network scanning)
//   - agent.go: Pi Agent server settings
//   - ca.go: Certificate Authority settings (local and Vault)
//   - sentry.go: Sentry error tracking settings
//   - cluster.go: Raft clustering settings
//   - webui.go: Web UI server and feature settings
//
// Usage:
//
//	import "github.com/dsyorkd/pi-controller/internal/config/defaults"
//
//	config := Config{
//	    API: APIConfig{
//	        Host: defaults.APIHost,
//	        Port: defaults.APIPort,
//	    },
//	}
package defaults
