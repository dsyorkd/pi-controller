package replication

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/dsyorkd/pi-controller/internal/storage"
)

// ReplicatedStore wraps a storage.Database and automatically replicates writes
type ReplicatedStore struct {
	*storage.Database
	replicator *Replicator
}

// NewReplicatedStore creates a new replicated store
func NewReplicatedStore(db *storage.Database, replicator *Replicator) *ReplicatedStore {
	return &ReplicatedStore{
		Database:   db,
		replicator: replicator,
	}
}

// Create intercepts create operations and replicates them
func (s *ReplicatedStore) Create(value interface{}) error {
	if !s.replicator.IsLeader() {
		return fmt.Errorf("not the leader, write operations not allowed")
	}

	// Execute locally first
	if err := s.Database.DB().Create(value).Error; err != nil {
		return err
	}

	// TODO: Extract SQL from GORM operation and replicate
	// For now, replication happens at the service layer

	return nil
}

// Update intercepts update operations and replicates them
func (s *ReplicatedStore) Update(value interface{}) error {
	if !s.replicator.IsLeader() {
		return fmt.Errorf("not the leader, write operations not allowed")
	}

	if err := s.Database.DB().Save(value).Error; err != nil {
		return err
	}

	return nil
}

// Delete intercepts delete operations and replicates them
func (s *ReplicatedStore) Delete(value interface{}) error {
	if !s.replicator.IsLeader() {
		return fmt.Errorf("not the leader, write operations not allowed")
	}

	if err := s.Database.DB().Delete(value).Error; err != nil {
		return err
	}

	return nil
}

// Transaction wraps a transaction and ensures it only runs on the leader
func (s *ReplicatedStore) Transaction(fc func(tx *gorm.DB) error) error {
	if !s.replicator.IsLeader() {
		return fmt.Errorf("not the leader, transactions not allowed")
	}

	return s.Database.DB().Transaction(fc)
}

// IsLeader returns whether this node can accept writes
func (s *ReplicatedStore) IsLeader() bool {
	return s.replicator.IsLeader()
}
