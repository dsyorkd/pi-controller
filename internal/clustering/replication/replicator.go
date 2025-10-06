package replication

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// Replicator handles SQLite database replication across cluster nodes
type Replicator struct {
	db         *storage.Database
	logger     logger.Interface
	applier    LogApplier
	isLeader   bool
	leaderMu   sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// LogApplier is the interface for applying replicated commands
// This will be implemented by the Raft cluster
type LogApplier interface {
	// Apply applies a command to the distributed log
	Apply(cmd []byte, timeout time.Duration) error

	// IsLeader returns true if this node is the leader
	IsLeader() bool
}

// NewReplicator creates a new database replicator
func NewReplicator(db *storage.Database, applier LogApplier, logger logger.Interface) *Replicator {
	return &Replicator{
		db:      db,
		logger:  logger,
		applier: applier,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the replication process
func (r *Replicator) Start(ctx context.Context) error {
	r.logger.Info("Starting database replicator")

	// Start watching for leadership changes
	r.wg.Add(1)
	go r.watchLeadership(ctx)

	return nil
}

// Stop gracefully stops the replicator
func (r *Replicator) Stop() error {
	r.logger.Info("Stopping database replicator")
	close(r.stopCh)
	r.wg.Wait()
	return nil
}

// watchLeadership monitors leadership status and adjusts replication mode
func (r *Replicator) watchLeadership(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			isLeader := r.applier.IsLeader()

			r.leaderMu.Lock()
			previousState := r.isLeader
			r.isLeader = isLeader
			r.leaderMu.Unlock()

			if isLeader && !previousState {
				// Became leader - enable read-write mode
				r.logger.Info("Became leader - enabling read-write database mode")
				r.promoteToReadWrite()
			} else if !isLeader && previousState {
				// Lost leadership - switch to read-only mode
				r.logger.Info("Lost leadership - switching to read-only database mode")
				r.demoteToReadOnly()
			}
		}
	}
}

// promoteToReadWrite enables read-write mode on the database
func (r *Replicator) promoteToReadWrite() {
	// Database is already read-write by default
	// Just log the transition
	r.logger.Info("Database in read-write mode (leader)")
}

// demoteToReadOnly switches database to read-only mode
func (r *Replicator) demoteToReadOnly() {
	// For now, we don't enforce read-only at the database level
	// Instead, we reject write operations at the service layer
	r.logger.Info("Database in read-only mode (follower)")
}

// ReplicateCommand replicates a database command to all nodes
func (r *Replicator) ReplicateCommand(cmd *DatabaseCommand) error {
	r.leaderMu.RLock()
	isLeader := r.isLeader
	r.leaderMu.RUnlock()

	if !isLeader {
		return fmt.Errorf("not the leader, cannot replicate commands")
	}

	// Serialize command
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Apply to Raft log (will replicate to all nodes)
	if err := r.applier.Apply(data, 5*time.Second); err != nil {
		return fmt.Errorf("failed to apply command: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"operation": cmd.Operation,
		"table":     cmd.Table,
	}).Debug("Replicated database command")

	return nil
}

// ApplyCommand applies a replicated command to the local database
func (r *Replicator) ApplyCommand(data []byte) error {
	var cmd DatabaseCommand
	if err := json.Unmarshal(data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"operation": cmd.Operation,
		"table":     cmd.Table,
	}).Debug("Applying replicated command")

	// Execute the SQL command
	return r.executeCommand(&cmd)
}

// executeCommand executes a database command
func (r *Replicator) executeCommand(cmd *DatabaseCommand) error {
	tx := r.db.DB().Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Execute the raw SQL
	if err := tx.Exec(cmd.SQL, cmd.Args...).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute command: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DatabaseCommand represents a replicated database operation
type DatabaseCommand struct {
	Operation string        `json:"operation"` // INSERT, UPDATE, DELETE
	Table     string        `json:"table"`
	SQL       string        `json:"sql"`
	Args      []interface{} `json:"args"`
	Timestamp time.Time     `json:"timestamp"`
}

// IsLeader returns whether this node is the leader
func (r *Replicator) IsLeader() bool {
	r.leaderMu.RLock()
	defer r.leaderMu.RUnlock()
	return r.isLeader
}
