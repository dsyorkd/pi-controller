package clustering

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftCluster implements a self-contained Raft-based clustering system
// All consensus and replication is embedded in the binary with minimal external dependencies
type RaftCluster struct {
	config    *ClusterConfig
	logger    logger.Interface
	raft      *raft.Raft
	fsm       *clusterFSM
	transport *raft.NetworkTransport
	isLeader  bool
	leaderMu  sync.RWMutex

	// Callbacks
	becomeLeaderCallback func()
	loseLeaderCallback   func()
	callbackMu           sync.RWMutex
}

// ClusterConfig holds clustering configuration
type ClusterConfig struct {
	// ControllerID is the unique identifier for this controller
	ControllerID string

	// BindAddr is the address to bind the Raft transport (e.g., "192.168.1.10:9091")
	BindAddr string

	// DataDir is where Raft stores its state (logs, snapshots)
	DataDir string

	// Bootstrap indicates if this is the first node in the cluster
	Bootstrap bool

	// InitialPeers are the addresses of other cluster members
	// Only used during bootstrap
	InitialPeers []string

	// HeartbeatTimeout is the time in follower state without contact before attempting election
	// Default: 1s (optimized for Raspberry Pi)
	HeartbeatTimeout time.Duration

	// ElectionTimeout is the time in candidate state without becoming leader before restarting election
	// Default: 1s
	ElectionTimeout time.Duration

	// SnapshotInterval is how often to snapshot state
	// Default: 30 minutes
	SnapshotInterval time.Duration

	// SnapshotThreshold is how many logs before triggering snapshot
	// Default: 8192
	SnapshotThreshold uint64

	// MaxAppendEntries controls how many append entries to send at once
	// Lower for Raspberry Pi to reduce memory (Default: 64)
	MaxAppendEntries int
}

// DefaultClusterConfig returns a configuration optimized for Raspberry Pi
func DefaultClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		HeartbeatTimeout:  1 * time.Second,
		ElectionTimeout:   1 * time.Second,
		SnapshotInterval:  30 * time.Minute,
		SnapshotThreshold: 8192,
		MaxAppendEntries:  64,
	}
}

// NewRaftCluster creates a new self-contained Raft cluster
func NewRaftCluster(config *ClusterConfig, log logger.Interface) (*RaftCluster, error) {
	if config == nil {
		config = DefaultClusterConfig()
	}

	if config.ControllerID == "" {
		return nil, fmt.Errorf("ControllerID is required")
	}

	if config.BindAddr == "" {
		return nil, fmt.Errorf("BindAddr is required")
	}

	if config.DataDir == "" {
		config.DataDir = "./data/raft"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	cluster := &RaftCluster{
		config: config,
		logger: log,
	}

	// Initialize Raft
	if err := cluster.setupRaft(); err != nil {
		return nil, fmt.Errorf("failed to setup Raft: %w", err)
	}

	return cluster, nil
}

// setupRaft initializes the Raft consensus engine
func (c *RaftCluster) setupRaft() error {
	// Create Raft configuration optimized for Raspberry Pi
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(c.config.ControllerID)
	raftConfig.HeartbeatTimeout = c.config.HeartbeatTimeout
	raftConfig.ElectionTimeout = c.config.ElectionTimeout
	raftConfig.SnapshotInterval = c.config.SnapshotInterval
	raftConfig.SnapshotThreshold = c.config.SnapshotThreshold
	raftConfig.MaxAppendEntries = c.config.MaxAppendEntries

	// Create FSM (Finite State Machine) for applying log entries
	c.fsm = newClusterFSM(c.logger)

	// Setup Raft transport (TCP)
	addr, err := net.ResolveTCPAddr("tcp", c.config.BindAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve bind address: %w", err)
	}

	transport, err := raft.NewTCPTransport(c.config.BindAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}
	c.transport = transport

	// Create log store using BoltDB (embedded database)
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(c.config.DataDir, "raft-log.db"))
	if err != nil {
		return fmt.Errorf("failed to create log store: %w", err)
	}

	// Create stable store (also using BoltDB)
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(c.config.DataDir, "raft-stable.db"))
	if err != nil {
		return fmt.Errorf("failed to create stable store: %w", err)
	}

	// Create snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(c.config.DataDir, 2, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Create Raft instance
	ra, err := raft.NewRaft(raftConfig, c.fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("failed to create Raft: %w", err)
	}
	c.raft = ra

	// Bootstrap cluster if this is the first node
	if c.config.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(c.config.ControllerID),
					Address: transport.LocalAddr(),
				},
			},
		}

		// Bootstrap the cluster
		future := c.raft.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			return fmt.Errorf("failed to bootstrap cluster: %w", err)
		}

		c.logger.WithField("controller_id", c.config.ControllerID).Info("Cluster bootstrapped successfully")
	}

	// Start observing leadership changes
	go c.observeLeadership()

	return nil
}

// observeLeadership watches for leadership changes
func (c *RaftCluster) observeLeadership() {
	for isLeader := range c.raft.LeaderCh() {
		c.leaderMu.Lock()
		previousState := c.isLeader
		c.isLeader = isLeader
		c.leaderMu.Unlock()

		if isLeader && !previousState {
			// Became leader
			c.logger.WithField("controller_id", c.config.ControllerID).Info("Became cluster leader")

			c.callbackMu.RLock()
			if c.becomeLeaderCallback != nil {
				go c.becomeLeaderCallback()
			}
			c.callbackMu.RUnlock()
		} else if !isLeader && previousState {
			// Lost leadership
			c.logger.WithField("controller_id", c.config.ControllerID).Warn("Lost cluster leadership")

			c.callbackMu.RLock()
			if c.loseLeaderCallback != nil {
				go c.loseLeaderCallback()
			}
			c.callbackMu.RUnlock()
		}
	}
}

// IsLeader returns true if this controller is the cluster leader
func (c *RaftCluster) IsLeader() bool {
	c.leaderMu.RLock()
	defer c.leaderMu.RUnlock()
	return c.isLeader
}

// GetLeader returns the current leader address
func (c *RaftCluster) GetLeader() string {
	leaderAddr, _ := c.raft.LeaderWithID()
	return string(leaderAddr)
}

// GetState returns the current Raft state
func (c *RaftCluster) GetState() raft.RaftState {
	return c.raft.State()
}

// Join adds a new node to the cluster (called on leader)
func (c *RaftCluster) Join(nodeID, addr string) error {
	if !c.IsLeader() {
		return fmt.Errorf("not the leader, cannot add nodes")
	}

	c.logger.WithFields(map[string]interface{}{
		"node_id": nodeID,
		"address": addr,
	}).Info("Adding node to cluster")

	// Add as voter (full Raft participant)
	future := c.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to add voter: %w", err)
	}

	return nil
}

// Leave removes a node from the cluster (called on leader)
func (c *RaftCluster) Leave(nodeID string) error {
	if !c.IsLeader() {
		return fmt.Errorf("not the leader, cannot remove nodes")
	}

	c.logger.WithField("node_id", nodeID).Info("Removing node from cluster")

	future := c.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	return nil
}

// Apply applies a command to the Raft log (replicates to all nodes)
// This is how we synchronize state across the cluster
func (c *RaftCluster) Apply(cmd []byte, timeout time.Duration) error {
	if !c.IsLeader() {
		return fmt.Errorf("not the leader, cannot apply commands")
	}

	future := c.raft.Apply(cmd, timeout)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply command: %w", err)
	}

	return nil
}

// GetServers returns all servers in the cluster
func (c *RaftCluster) GetServers() ([]ClusterMember, error) {
	future := c.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	config := future.Configuration()
	members := make([]ClusterMember, 0, len(config.Servers))

	for _, server := range config.Servers {
		members = append(members, ClusterMember{
			ID:      string(server.ID),
			Address: string(server.Address),
			Voter:   server.Suffrage == raft.Voter,
		})
	}

	return members, nil
}

// OnBecomeLeader sets callback for when this controller becomes leader
func (c *RaftCluster) OnBecomeLeader(callback func()) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.becomeLeaderCallback = callback
}

// OnLoseLeadership sets callback for when this controller loses leadership
func (c *RaftCluster) OnLoseLeadership(callback func()) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.loseLeaderCallback = callback
}

// Shutdown gracefully shuts down the Raft cluster
func (c *RaftCluster) Shutdown() error {
	c.logger.Info("Shutting down Raft cluster")

	// Shutdown Raft
	future := c.raft.Shutdown()
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to shutdown Raft: %w", err)
	}

	// Close transport
	if c.transport != nil {
		if err := c.transport.Close(); err != nil {
			return fmt.Errorf("failed to close transport: %w", err)
		}
	}

	return nil
}

// Stats returns Raft statistics
func (c *RaftCluster) Stats() map[string]string {
	return c.raft.Stats()
}

// ClusterMember represents a member of the cluster
type ClusterMember struct {
	ID      string
	Address string
	Voter   bool
}

// clusterFSM implements raft.FSM for the cluster state machine
type clusterFSM struct {
	logger logger.Interface
	mu     sync.RWMutex
}

func newClusterFSM(logger logger.Interface) *clusterFSM {
	return &clusterFSM{
		logger: logger,
	}
}

// Apply applies a Raft log entry to the FSM
func (f *clusterFSM) Apply(log *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	// TODO: Parse command and apply to local state
	// For now, just log it
	f.logger.WithFields(map[string]interface{}{
		"index": log.Index,
		"term":  log.Term,
		"type":  log.Type,
	}).Debug("Applying Raft log entry")

	return nil
}

// Snapshot creates a snapshot of the FSM state
func (f *clusterFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// TODO: Create snapshot of current state
	return &clusterSnapshot{}, nil
}

// Restore restores the FSM from a snapshot
func (f *clusterFSM) Restore(snapshot io.ReadCloser) error {
	defer snapshot.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	// TODO: Restore state from snapshot
	return nil
}

// clusterSnapshot implements raft.FSMSnapshot
type clusterSnapshot struct {
}

func (s *clusterSnapshot) Persist(sink raft.SnapshotSink) error {
	// TODO: Write snapshot data
	return sink.Close()
}

func (s *clusterSnapshot) Release() {
	// Nothing to release for now
}

// WaitForLeader blocks until a leader is elected or context is cancelled
func (c *RaftCluster) WaitForLeader(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if c.GetLeader() != "" {
				return nil
			}
		}
	}
}
