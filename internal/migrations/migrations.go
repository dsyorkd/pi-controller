package migrations

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID          string    `gorm:"primaryKey"`
	AppliedAt   time.Time `gorm:"not null"`
	Description string    `gorm:"not null"`
}

// MigrationFunc represents a migration function
type MigrationFunc func(*gorm.DB) error

// MigrationDefinition represents a single migration with up and down functions
type MigrationDefinition struct {
	ID          string
	Description string
	Up          MigrationFunc
	Down        MigrationFunc
}

// Migrator handles database migrations
type Migrator struct {
	db         *gorm.DB
	logger     logger.Interface
	migrations []MigrationDefinition
}

// NewMigrator creates a new migration manager
func NewMigrator(db *gorm.DB, logger logger.Interface) *Migrator {
	return &Migrator{
		db:         db,
		logger:     logger,
		migrations: getAllMigrations(),
	}
}

// EnsureMigrationTable creates the migrations table if it doesn't exist
func (m *Migrator) EnsureMigrationTable() error {
	if err := m.db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}
	return nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if err := m.EnsureMigrationTable(); err != nil {
		return err
	}

	// Get applied migrations
	var applied []Migration
	if err := m.db.Find(&applied).Error; err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, migration := range applied {
		appliedMap[migration.ID] = true
	}

	// Sort migrations by ID to ensure order
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].ID < m.migrations[j].ID
	})

	// Apply pending migrations
	for _, migration := range m.migrations {
		if appliedMap[migration.ID] {
			m.logger.Debug("Migration already applied", "id", migration.ID)
			continue
		}

		m.logger.Info("Applying migration", "id", migration.ID, "description", migration.Description)
		
		// Run migration in transaction
		err := m.db.Transaction(func(tx *gorm.DB) error {
			if err := migration.Up(tx); err != nil {
				return fmt.Errorf("migration %s failed: %w", migration.ID, err)
			}

			// Record migration as applied
			migrationRecord := Migration{
				ID:          migration.ID,
				AppliedAt:   time.Now(),
				Description: migration.Description,
			}
			if err := tx.Create(&migrationRecord).Error; err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migration.ID, err)
			}

			return nil
		})

		if err != nil {
			return err
		}

		m.logger.Info("Migration applied successfully", "id", migration.ID)
	}

	m.logger.Info("All migrations applied successfully")
	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down() error {
	if err := m.EnsureMigrationTable(); err != nil {
		return err
	}

	// Get the most recent migration
	var lastMigration Migration
	if err := m.db.Order("applied_at DESC").First(&lastMigration).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			m.logger.Info("No migrations to roll back")
			return nil
		}
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	// Find the migration definition
	var migrationDef *MigrationDefinition
	for _, migration := range m.migrations {
		if migration.ID == lastMigration.ID {
			migrationDef = &migration
			break
		}
	}

	if migrationDef == nil {
		return fmt.Errorf("migration definition not found for ID: %s", lastMigration.ID)
	}

	m.logger.Info("Rolling back migration", "id", migrationDef.ID, "description", migrationDef.Description)

	// Run rollback in transaction
	err := m.db.Transaction(func(tx *gorm.DB) error {
		if err := migrationDef.Down(tx); err != nil {
			return fmt.Errorf("rollback for migration %s failed: %w", migrationDef.ID, err)
		}

		// Remove migration record
		if err := tx.Delete(&Migration{}, "id = ?", migrationDef.ID).Error; err != nil {
			return fmt.Errorf("failed to remove migration record %s: %w", migrationDef.ID, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	m.logger.Info("Migration rolled back successfully", "id", migrationDef.ID)
	return nil
}

// Status shows the current migration status
func (m *Migrator) Status() ([]MigrationStatus, error) {
	if err := m.EnsureMigrationTable(); err != nil {
		return nil, err
	}

	// Get applied migrations
	var applied []Migration
	if err := m.db.Find(&applied).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]Migration)
	for _, migration := range applied {
		appliedMap[migration.ID] = migration
	}

	// Create status for all migrations
	var statuses []MigrationStatus
	for _, migration := range m.migrations {
		status := MigrationStatus{
			ID:          migration.ID,
			Description: migration.Description,
			Applied:     false,
		}

		if applied, exists := appliedMap[migration.ID]; exists {
			status.Applied = true
			status.AppliedAt = &applied.AppliedAt
		}

		statuses = append(statuses, status)
	}

	// Sort by ID
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].ID < statuses[j].ID
	})

	return statuses, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	ID          string
	Description string
	Applied     bool
	AppliedAt   *time.Time
}

// Reset drops all tables and reapplies all migrations (DANGEROUS - only for development)
func (m *Migrator) Reset() error {
	m.logger.Warn("Resetting database - this will drop all tables!")

	// Get all table names
	var tables []string
	rows, err := m.db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Rows()
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	// Drop all tables
	for _, table := range tables {
		if err := m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)).Error; err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
		m.logger.Debug("Dropped table", "table", table)
	}

	// Run all migrations
	return m.Up()
}

// ValidateMigrationOrder validates that migration IDs are properly ordered
func (m *Migrator) ValidateMigrationOrder() error {
	if len(m.migrations) == 0 {
		return nil
	}

	// Sort migrations by ID
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].ID < m.migrations[j].ID
	})

	// Validate that all IDs are properly formatted timestamps
	for _, migration := range m.migrations {
		if len(migration.ID) != 14 {
			return fmt.Errorf("migration ID %s must be 14 characters (YYYYMMDDHHMMSS)", migration.ID)
		}

		if _, err := strconv.ParseInt(migration.ID, 10, 64); err != nil {
			return fmt.Errorf("migration ID %s must be numeric timestamp (YYYYMMDDHHMMSS)", migration.ID)
		}
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, migration := range m.migrations {
		if seen[migration.ID] {
			return fmt.Errorf("duplicate migration ID: %s", migration.ID)
		}
		seen[migration.ID] = true
	}

	return nil
}

// GetPendingMigrations returns a list of migrations that haven't been applied
func (m *Migrator) GetPendingMigrations() ([]MigrationDefinition, error) {
	if err := m.EnsureMigrationTable(); err != nil {
		return nil, err
	}

	// Get applied migrations
	var applied []Migration
	if err := m.db.Find(&applied).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, migration := range applied {
		appliedMap[migration.ID] = true
	}

	// Filter pending migrations
	var pending []MigrationDefinition
	for _, migration := range m.migrations {
		if !appliedMap[migration.ID] {
			pending = append(pending, migration)
		}
	}

	// Sort by ID
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].ID < pending[j].ID
	})

	return pending, nil
}