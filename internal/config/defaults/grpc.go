package defaults

// gRPC server defaults
const (
	// GRPCHost is the default gRPC server bind address
	GRPCHost = "0.0.0.0"

	// GRPCPort is the default gRPC server port
	GRPCPort = 9090

	// GRPCTLSCertFile is the default TLS certificate path (production)
	GRPCTLSCertFile = "/etc/pi-controller/tls/server.crt"

	// GRPCTLSKeyFile is the default TLS key path (production)
	GRPCTLSKeyFile = "/etc/pi-controller/tls/server.key"

	// GRPCTLSCAFile is the default CA certificate path for mTLS
	GRPCTLSCAFile = "/etc/pi-controller/tls/ca.crt"
)

// gRPC client defaults
const (
	// GRPCClientServerAddress is the default server address for gRPC client
	GRPCClientServerAddress = "localhost"

	// GRPCClientServerPort is the default server port for gRPC client
	GRPCClientServerPort = 9090

	// GRPCClientConnectionTimeout is the default connection timeout
	GRPCClientConnectionTimeout = "10s"

	// GRPCClientRequestTimeout is the default request timeout
	GRPCClientRequestTimeout = "30s"

	// GRPCClientMaxMessageSize is the default max message size (4MB)
	GRPCClientMaxMessageSize = 4 * 1024 * 1024

	// GRPCClientMaxRetries is the default max retry count
	GRPCClientMaxRetries = 5

	// GRPCClientInitialRetryDelay is the default initial retry delay
	GRPCClientInitialRetryDelay = "1s"

	// GRPCClientMaxRetryDelay is the default max retry delay
	GRPCClientMaxRetryDelay = "60s"

	// GRPCClientRetryMultiplier is the default retry backoff multiplier
	GRPCClientRetryMultiplier = 2.0

	// GRPCClientHeartbeatInterval is the default heartbeat interval
	GRPCClientHeartbeatInterval = "30s"

	// GRPCClientHeartbeatTimeout is the default heartbeat timeout
	GRPCClientHeartbeatTimeout = "5s"

	// GRPCClientKeepAliveTime is the default keep-alive time
	GRPCClientKeepAliveTime = "30s"

	// GRPCClientKeepAliveTimeout is the default keep-alive timeout
	GRPCClientKeepAliveTimeout = "5s"

	// GRPCClientInsecure indicates whether insecure mode is enabled by default
	GRPCClientInsecure = true
)
