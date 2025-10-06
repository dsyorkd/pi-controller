package election

import (
	"context"
	"time"
)

// LeaderElector manages leader election for controller clustering
type LeaderElector interface {
	// Run starts the leader election loop
	// This should be called in a goroutine
	Run(ctx context.Context) error

	// IsLeader returns true if this controller is currently the leader
	IsLeader() bool

	// GetLeader returns the current leader ID, or empty string if no leader
	GetLeader() (string, error)

	// GetTerm returns the current election term
	GetTerm() uint64

	// OnBecomeLeader sets callback for when this controller becomes leader
	OnBecomeLeader(callback func())

	// OnLoseLeadership sets callback for when this controller loses leadership
	OnLoseLeadership(callback func())

	// Stop gracefully stops the leader election
	Stop() error
}

// Config holds leader election configuration
type Config struct {
	// ControllerID is the unique identifier for this controller
	ControllerID string

	// LeaseTTL is the lease duration in seconds (default: 30)
	LeaseTTL int

	// RenewalInterval is how often to renew the lease in seconds (default: 10)
	RenewalInterval int

	// RetryInterval is the interval between election attempts in seconds (default: 5)
	RetryInterval int

	// Backend specifies the election backend: "etcd" or "raft"
	Backend string

	// EtcdEndpoints are the etcd cluster endpoints (for etcd backend)
	EtcdEndpoints []string

	// EtcdTLS holds TLS configuration for etcd
	EtcdTLS *TLSConfig

	// LeadershipKey is the etcd key used for leadership (default: "/pi-controller/leader")
	LeadershipKey string
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// ElectionEvent represents a leadership change event
type ElectionEvent struct {
	Type         EventType
	LeaderID     string
	Term         uint64
	Timestamp    time.Time
	PreviousLeader string
}

// EventType represents the type of election event
type EventType string

const (
	// EventBecameLeader is emitted when this controller becomes leader
	EventBecameLeader EventType = "became_leader"

	// EventLostLeadership is emitted when this controller loses leadership
	EventLostLeadership EventType = "lost_leadership"

	// EventLeaderChanged is emitted when a different controller becomes leader
	EventLeaderChanged EventType = "leader_changed"

	// EventElectionFailed is emitted when leader election fails
	EventElectionFailed EventType = "election_failed"
)

// DefaultConfig returns default leader election configuration
func DefaultConfig() *Config {
	return &Config{
		LeaseTTL:        30,
		RenewalInterval: 10,
		RetryInterval:   5,
		Backend:         "etcd",
		LeadershipKey:   "/pi-controller/leader",
	}
}
