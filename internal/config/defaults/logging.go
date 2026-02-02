package defaults

// Logging defaults
const (
	// LogLevel is the default log level
	LogLevel = "info"

	// LogFormat is the default log format
	LogFormat = "json"

	// LogOutput is the default log output
	LogOutput = "stdout"

	// LogMaxSize is the default max log file size in MB
	LogMaxSize = 100

	// LogMaxBackups is the default max number of log file backups
	LogMaxBackups = 3

	// LogMaxAge is the default max age of log files in days
	LogMaxAge = 28

	// LogCompress indicates whether log compression is enabled
	LogCompress = true
)
