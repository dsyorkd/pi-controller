package defaults

// Database defaults
const (
	// DatabasePath is the default SQLite database file path
	DatabasePath = "pi-controller.db"

	// DatabaseMaxOpenConns is the maximum number of open database connections
	DatabaseMaxOpenConns = 25

	// DatabaseMaxIdleConns is the maximum number of idle database connections
	DatabaseMaxIdleConns = 5

	// DatabaseConnMaxLifetime is the maximum connection lifetime
	DatabaseConnMaxLifetime = "5m"

	// DatabaseLogLevel is the default database log level
	DatabaseLogLevel = "warn"
)
