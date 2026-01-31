package defaults

// Discovery defaults
const (
	// DiscoveryEnabled indicates whether discovery is enabled
	DiscoveryEnabled = true

	// DiscoveryMethod is the default discovery method
	DiscoveryMethod = "mdns"

	// DiscoveryPort is the default discovery port
	DiscoveryPort = 9091

	// DiscoveryInterval is the default discovery interval
	DiscoveryInterval = "30s"

	// DiscoveryTimeout is the default discovery timeout
	DiscoveryTimeout = "5s"

	// DiscoveryServiceName is the default service name
	DiscoveryServiceName = "pi-controller"

	// DiscoveryServiceType is the default service type
	DiscoveryServiceType = "_pi-controller._tcp"

	// DiscoveryScanTimeout is the default timeout for individual host scans
	DiscoveryScanTimeout = "2s"

	// DiscoveryScanConcurrency is the default number of concurrent scan workers
	DiscoveryScanConcurrency = 10

	// DiscoveryScanRateLimit is the default max scans per second
	DiscoveryScanRateLimit = 100
)

// DiscoveryScanPorts contains default ports to scan
var DiscoveryScanPorts = []int{9091}
