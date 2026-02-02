// Package defaults contains default configuration values for the pi-controller application.
// These constants provide a single source of truth for all configuration defaults,
// improving maintainability, discoverability, and testability.
package defaults

// App defaults
const (
	// AppName is the default application name
	AppName = "pi-controller"

	// AppVersion is the default application version
	AppVersion = "dev"

	// AppEnvironment is the default environment (production for security)
	AppEnvironment = "production"

	// AppDataDir is the default data directory
	AppDataDir = "./data"

	// AppDebug indicates whether debug mode is enabled by default
	AppDebug = false
)
