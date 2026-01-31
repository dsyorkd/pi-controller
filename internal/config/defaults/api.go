package defaults

// API server defaults
const (
	// APIHost is the default API server bind address
	APIHost = "0.0.0.0"

	// APIPort is the default API server port
	APIPort = 8080

	// APIReadTimeout is the default read timeout
	APIReadTimeout = "30s"

	// APIWriteTimeout is the default write timeout
	APIWriteTimeout = "30s"

	// APITLSCertFile is the default TLS certificate path (production)
	APITLSCertFile = "/etc/pi-controller/tls/server.crt"

	// APITLSKeyFile is the default TLS key path (production)
	APITLSKeyFile = "/etc/pi-controller/tls/server.key"

	// APITLSCAFile is the default CA certificate path for mTLS
	APITLSCAFile = "/etc/pi-controller/tls/ca.crt"

	// APICORSEnabled indicates whether CORS is enabled by default
	APICORSEnabled = true

	// APIAuthEnabled indicates whether authentication is enabled by default
	APIAuthEnabled = true
)
