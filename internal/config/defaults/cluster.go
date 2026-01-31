package defaults

// Cluster defaults
const (
	// ClusterEnabled indicates whether clustering is enabled
	ClusterEnabled = false

	// ClusterPortable indicates whether portable/client-only mode is enabled
	ClusterPortable = false

	// ClusterBootstrap indicates whether to bootstrap as first member
	ClusterBootstrap = false

	// ClusterDataDir is the default Raft data directory
	ClusterDataDir = "./data/raft"

	// ClusterHeartbeatTimeout is the default Raft heartbeat timeout
	ClusterHeartbeatTimeout = "1s"

	// ClusterElectionTimeout is the default Raft election timeout
	ClusterElectionTimeout = "1s"

	// ClusterSnapshotInterval is the default snapshot interval
	ClusterSnapshotInterval = "30m"

	// ClusterSnapshotThreshold is the default snapshot threshold
	ClusterSnapshotThreshold uint64 = 8192

	// ClusterMaxAppendEntries is the default max append entries
	ClusterMaxAppendEntries = 64
)
