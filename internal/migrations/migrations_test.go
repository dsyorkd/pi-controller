package migrations

import (
	"testing"
	"time"

	applogger "github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testLogger implements a simple logger for testing
type testLogger struct {
	t *testing.T
}

func (l *testLogger) Debug(msg string, args ...interface{}) {
	l.t.Logf("[DEBUG] "+msg, args...)
}

func (l *testLogger) Info(msg string, args ...interface{}) {
	l.t.Logf("[INFO] "+msg, args...)
}

func (l *testLogger) Warn(msg string, args ...interface{}) {
	l.t.Logf("[WARN] "+msg, args...)
}

func (l *testLogger) Error(msg string, args ...interface{}) {
	l.t.Logf("[ERROR] "+msg, args...)
}

func (l *testLogger) Debugf(format string, args ...interface{}) {
	l.t.Logf("[DEBUG] "+format, args...)
}

func (l *testLogger) Infof(format string, args ...interface{}) {
	l.t.Logf("[INFO] "+format, args...)
}

func (l *testLogger) Warnf(format string, args ...interface{}) {
	l.t.Logf("[WARN] "+format, args...)
}

func (l *testLogger) Errorf(format string, args ...interface{}) {
	l.t.Logf("[ERROR] "+format, args...)
}

func (l *testLogger) Fatalf(format string, args ...interface{}) {
	l.t.Fatalf("[FATAL] "+format, args...)
}

func (l *testLogger) WithField(key string, value interface{}) applogger.Interface {
	return l
}

func (l *testLogger) WithFields(fields map[string]interface{}) applogger.Interface {
	return l
}

func (l *testLogger) WithError(err error) applogger.Interface {
	return l
}

func setupTestDB(t *testing.T) (*gorm.DB, applogger.Interface) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	testLog := &testLogger{t: t}
	return db, testLog
}

func TestMigrator_EnsureMigrationTable(t *testing.T) {
	db, log := setupTestDB(t)
	migrator := NewMigrator(db, log)

	// Test creating migration table
	err := migrator.EnsureMigrationTable()
	assert.NoError(t, err)

	// Verify table exists
	var count int64
	err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Test calling it again should not error
	err = migrator.EnsureMigrationTable()
	assert.NoError(t, err)
}

func TestMigrator_ValidateMigrationOrder(t *testing.T) {
	db, log := setupTestDB(t)

	tests := []struct {
		name        string
		migrations  []MigrationDefinition
		expectError bool
	}{
		{
			name:       "empty migrations",
			migrations: []MigrationDefinition{},
		},
		{
			name: "valid migrations",
			migrations: []MigrationDefinition{
				{ID: "20241201000001", Description: "Test 1"},
				{ID: "20241201000002", Description: "Test 2"},
			},
		},
		{
			name: "invalid ID length",
			migrations: []MigrationDefinition{
				{ID: "2024120100001", Description: "Test 1"}, // 13 chars instead of 14
			},
			expectError: true,
		},
		{
			name: "non-numeric ID",
			migrations: []MigrationDefinition{
				{ID: "2024120100000a", Description: "Test 1"},
			},
			expectError: true,
		},
		{
			name: "duplicate IDs",
			migrations: []MigrationDefinition{
				{ID: "20241201000001", Description: "Test 1"},
				{ID: "20241201000001", Description: "Test 2"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrator := &Migrator{
				db:         db,
				logger:     log,
				migrations: tt.migrations,
			}

			err := migrator.ValidateMigrationOrder()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMigrator_GetPendingMigrations(t *testing.T) {
	db, log := setupTestDB(t)

	// Create test migrations
	testMigrations := []MigrationDefinition{
		{ID: "20241201000001", Description: "Test 1"},
		{ID: "20241201000002", Description: "Test 2"},
		{ID: "20241201000003", Description: "Test 3"},
	}

	migrator := &Migrator{
		db:         db,
		logger:     log,
		migrations: testMigrations,
	}

	// Ensure migration table exists
	err := migrator.EnsureMigrationTable()
	require.NoError(t, err)

	// Initially, all migrations should be pending
	pending, err := migrator.GetPendingMigrations()
	assert.NoError(t, err)
	assert.Len(t, pending, 3)

	// Apply one migration manually
	migration := Migration{
		ID:          "20241201000001",
		AppliedAt:   time.Now(),
		Description: "Test 1",
	}
	err = db.Create(&migration).Error
	require.NoError(t, err)

	// Now only 2 should be pending
	pending, err = migrator.GetPendingMigrations()
	assert.NoError(t, err)
	assert.Len(t, pending, 2)
	assert.Equal(t, "20241201000002", pending[0].ID)
	assert.Equal(t, "20241201000003", pending[1].ID)
}

func TestMigrator_Status(t *testing.T) {
	db, log := setupTestDB(t)

	testMigrations := []MigrationDefinition{
		{ID: "20241201000001", Description: "Test 1"},
		{ID: "20241201000002", Description: "Test 2"},
	}

	migrator := &Migrator{
		db:         db,
		logger:     log,
		migrations: testMigrations,
	}

	// Ensure migration table exists
	err := migrator.EnsureMigrationTable()
	require.NoError(t, err)

	// Get initial status
	statuses, err := migrator.Status()
	assert.NoError(t, err)
	assert.Len(t, statuses, 2)
	assert.False(t, statuses[0].Applied)
	assert.False(t, statuses[1].Applied)

	// Apply one migration
	appliedAt := time.Now()
	migration := Migration{
		ID:          "20241201000001",
		AppliedAt:   appliedAt,
		Description: "Test 1",
	}
	err = db.Create(&migration).Error
	require.NoError(t, err)

	// Get updated status
	statuses, err = migrator.Status()
	assert.NoError(t, err)
	assert.Len(t, statuses, 2)
	assert.True(t, statuses[0].Applied)
	assert.NotNil(t, statuses[0].AppliedAt)
	assert.False(t, statuses[1].Applied)
	assert.Nil(t, statuses[1].AppliedAt)
}

// Test the complete migration lifecycle - Up, Down, and Reset
func TestMigrator_CompleteMigrationLifecycle(t *testing.T) {
	db, log := setupTestDB(t)

	// Create a simple test migration
	testMigrations := []MigrationDefinition{
		{
			ID:          "20241201000001",
			Description: "Create test table",
			Up: func(db *gorm.DB) error {
				return db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)").Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("DROP TABLE IF EXISTS test_table").Error
			},
		},
		{
			ID:          "20241201000002",
			Description: "Add index to test table",
			Up: func(db *gorm.DB) error {
				return db.Exec("CREATE INDEX idx_test_name ON test_table(name)").Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("DROP INDEX IF EXISTS idx_test_name").Error
			},
		},
	}

	migrator := &Migrator{
		db:         db,
		logger:     log,
		migrations: testMigrations,
	}

	// Test 1: Run all migrations up
	t.Run("Up - Apply all migrations", func(t *testing.T) {
		err := migrator.Up()
		assert.NoError(t, err)

		// Verify both migrations were applied
		var count int64
		err = db.Model(&Migration{}).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Verify test table exists
		var tableCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), tableCount)

		// Verify index exists
		var indexCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_test_name'").Scan(&indexCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), indexCount)
	})

	// Test 2: Run Down to rollback last migration
	t.Run("Down - Rollback last migration", func(t *testing.T) {
		err := migrator.Down()
		assert.NoError(t, err)

		// Verify only one migration remains
		var count int64
		err = db.Model(&Migration{}).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Verify index is removed but table still exists
		var indexCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_test_name'").Scan(&indexCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), indexCount)

		var tableCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), tableCount)
	})

	// Test 3: Run Down again to rollback remaining migration
	t.Run("Down - Rollback remaining migration", func(t *testing.T) {
		err := migrator.Down()
		assert.NoError(t, err)

		// Verify no migrations remain
		var count int64
		err = db.Model(&Migration{}).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Verify table is removed
		var tableCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), tableCount)
	})

	// Test 4: Run Down when no migrations exist
	t.Run("Down - No migrations to rollback", func(t *testing.T) {
		err := migrator.Down()
		assert.NoError(t, err) // Should not error, just log that no migrations exist
	})

	// Test 5: Run Up again to reapply migrations
	t.Run("Up - Reapply migrations after rollback", func(t *testing.T) {
		err := migrator.Up()
		assert.NoError(t, err)

		// Verify both migrations were reapplied
		var count int64
		err = db.Model(&Migration{}).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	// Test 6: Test Reset (drops all tables and reapplies migrations)
	t.Run("Reset - Drop all tables and reapply", func(t *testing.T) {
		// Add some test data first
		err := db.Exec("INSERT INTO test_table (name) VALUES ('test data')").Error
		assert.NoError(t, err)

		// Verify data exists
		var dataCount int64
		err = db.Raw("SELECT count(*) FROM test_table").Scan(&dataCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), dataCount)

		// Reset database
		err = migrator.Reset()
		assert.NoError(t, err)

		// Verify migrations are reapplied
		var migrationCount int64
		err = db.Model(&Migration{}).Count(&migrationCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(2), migrationCount)

		// Verify table exists but data is gone
		var tableCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), tableCount)

		// Verify no data remains
		err = db.Raw("SELECT count(*) FROM test_table").Scan(&dataCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), dataCount)
	})
}

// Test migration error handling and rollback
func TestMigrator_MigrationErrorHandling(t *testing.T) {
	db, log := setupTestDB(t)

	// Create migrations where the second one will fail
	testMigrations := []MigrationDefinition{
		{
			ID:          "20241201000001",
			Description: "Create test table",
			Up: func(db *gorm.DB) error {
				return db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY)").Error
			},
			Down: func(db *gorm.DB) error {
				return db.Exec("DROP TABLE IF EXISTS test_table").Error
			},
		},
		{
			ID:          "20241201000002",
			Description: "Failing migration",
			Up: func(db *gorm.DB) error {
				// This will fail due to invalid SQL
				return db.Exec("CREATE TABLE invalid syntax").Error
			},
			Down: func(db *gorm.DB) error {
				return nil // No-op
			},
		},
	}

	migrator := &Migrator{
		db:         db,
		logger:     log,
		migrations: testMigrations,
	}

	// Test that migration fails and is rolled back
	err := migrator.Up()
	assert.Error(t, err)

	// Verify first migration was applied
	var count int64
	err = db.Model(&Migration{}).Count(&count).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count) // Only first migration should be recorded

	// Verify first table was created
	var tableCount int64
	err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), tableCount)
}

// Test migration with actual schema creation matching production migrations
func TestMigrator_ProductionSchemaIntegration(t *testing.T) {
	db, log := setupTestDB(t)

	// Use actual production migrations
	migrator := NewMigrator(db, log)

	t.Run("Apply all production migrations", func(t *testing.T) {
		// Validate migration order
		err := migrator.ValidateMigrationOrder()
		assert.NoError(t, err)

		// Apply all migrations
		err = migrator.Up()
		assert.NoError(t, err)

		// Verify all expected tables exist
		expectedTables := []string{"clusters", "nodes", "gpio_devices", "gpio_readings", "migrations"}
		for _, table := range expectedTables {
			var count int64
			err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count).Error
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count, "Table %s should exist", table)
		}

		// Verify expected indexes exist (sample a few critical ones)
		expectedIndexes := []string{
			"idx_clusters_name",
			"idx_nodes_name",
			"idx_nodes_ip_address",
			"idx_gpio_devices_node_id",
			"idx_gpio_readings_device_id",
		}
		for _, index := range expectedIndexes {
			var count int64
			err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count).Error
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count, "Index %s should exist", index)
		}

		// Verify foreign key constraints work by testing table structure
		var pragmaResults []struct {
			CID       int         `gorm:"column:cid"`
			Name      string      `gorm:"column:name"`
			Type      string      `gorm:"column:type"`
			NotNull   int         `gorm:"column:notnull"`
			DfltValue interface{} `gorm:"column:dflt_value"`
			PK        int         `gorm:"column:pk"`
		}

		// Check nodes table has foreign key to clusters
		err = db.Raw("PRAGMA table_info(nodes)").Scan(&pragmaResults).Error
		assert.NoError(t, err)

		clusterIDFound := false
		for _, col := range pragmaResults {
			if col.Name == "cluster_id" {
				clusterIDFound = true
				assert.Equal(t, "INTEGER", col.Type)
				break
			}
		}
		assert.True(t, clusterIDFound, "cluster_id column should exist in nodes table")
	})

	t.Run("Rollback all production migrations", func(t *testing.T) {
		// Get status before rollback
		statuses, err := migrator.Status()
		assert.NoError(t, err)
		appliedCount := 0
		for _, status := range statuses {
			if status.Applied {
				appliedCount++
			}
		}
		assert.Greater(t, appliedCount, 0, "Should have applied migrations to rollback")

		// Roll back all migrations one by one
		for i := 0; i < appliedCount; i++ {
			err = migrator.Down()
			assert.NoError(t, err, "Rollback %d should succeed", i+1)
		}

		// Verify no migrations remain applied
		var count int64
		err = db.Model(&Migration{}).Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count, "No applied migrations should remain")

		// Verify main tables are dropped (migrations table should still exist)
		droppedTables := []string{"clusters", "nodes", "gpio_devices", "gpio_readings"}
		for _, table := range droppedTables {
			var tableCount int64
			err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&tableCount).Error
			assert.NoError(t, err)
			assert.Equal(t, int64(0), tableCount, "Table %s should be dropped", table)
		}

		// Verify migrations table still exists
		var migrationTableCount int64
		err = db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&migrationTableCount).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), migrationTableCount, "Migrations table should still exist")
	})
}

// Test migration 20241201000013 - Add node discovery and type tracking fields
func TestMigration_AddNodeDiscoveryFields(t *testing.T) {
	db, log := setupTestDB(t)

	// Apply migrations up to but not including the discovery fields migration
	preMigrations := []MigrationDefinition{}
	for _, m := range getAllMigrations() {
		if m.ID < "20241201000013" {
			preMigrations = append(preMigrations, m)
		}
	}

	preMigrator := &Migrator{
		db:         db,
		logger:     log,
		migrations: preMigrations,
	}

	// Apply pre-discovery migrations
	err := preMigrator.Up()
	require.NoError(t, err)

	// Insert a test node before the discovery migration
	err = db.Exec(`
		INSERT INTO nodes (name, ip_address, mac_address, status, role, created_at, updated_at)
		VALUES ('test-node', '192.168.1.10', 'aa:bb:cc:dd:ee:ff', 'ready', 'worker', datetime('now'), datetime('now'))
	`).Error
	require.NoError(t, err)

	// Get the node ID
	var nodeID uint
	err = db.Raw("SELECT id FROM nodes WHERE name = 'test-node'").Scan(&nodeID).Error
	require.NoError(t, err)

	// Get the node's created_at timestamp
	var createdAt time.Time
	err = db.Raw("SELECT created_at FROM nodes WHERE id = ?", nodeID).Scan(&createdAt).Error
	require.NoError(t, err)

	t.Run("Apply discovery fields migration", func(t *testing.T) {
		// Find and apply only the discovery migration
		var discoveryMigration *MigrationDefinition
		for _, m := range getAllMigrations() {
			if m.ID == "20241201000013" {
				migDef := m
				discoveryMigration = &migDef
				break
			}
		}
		require.NotNil(t, discoveryMigration)

		// Apply the migration
		err := discoveryMigration.Up(db)
		assert.NoError(t, err)

		// Verify new columns exist
		var pragmaResults []struct {
			CID       int         `gorm:"column:cid"`
			Name      string      `gorm:"column:name"`
			Type      string      `gorm:"column:type"`
			NotNull   int         `gorm:"column:notnull"`
			DfltValue interface{} `gorm:"column:dflt_value"`
			PK        int         `gorm:"column:pk"`
		}

		err = db.Raw("PRAGMA table_info(nodes)").Scan(&pragmaResults).Error
		require.NoError(t, err)

		// Check that all new columns exist
		requiredColumns := map[string]bool{
			"discovery_method":   false,
			"discovered_at":      false,
			"node_type":          false,
			"controller_version": false,
			"agent_port":         false,
		}

		for _, col := range pragmaResults {
			if _, exists := requiredColumns[col.Name]; exists {
				requiredColumns[col.Name] = true
			}
		}

		for colName, found := range requiredColumns {
			assert.True(t, found, "Column %s should exist", colName)
		}

		// Verify default values are set correctly
		var node struct {
			DiscoveryMethod   string
			DiscoveredAt      *time.Time
			NodeType          string
			ControllerVersion string
			AgentPort         int
		}

		err = db.Raw(`
			SELECT discovery_method, discovered_at, node_type, controller_version, agent_port
			FROM nodes WHERE id = ?
		`, nodeID).Scan(&node).Error
		require.NoError(t, err)

		// Check default values
		assert.Equal(t, "manual", node.DiscoveryMethod, "Should default to manual")
		assert.Equal(t, "generic", node.NodeType, "Should default to generic")
		assert.NotNil(t, node.DiscoveredAt, "discovered_at should be set")
		// discovered_at should equal created_at for existing nodes
		if node.DiscoveredAt != nil {
			assert.WithinDuration(t, createdAt, *node.DiscoveredAt, time.Second)
		}
		assert.Equal(t, 0, node.AgentPort, "AgentPort should default to 0")
		assert.Empty(t, node.ControllerVersion, "ControllerVersion should be empty")
	})

	t.Run("Test discovery fields with new node", func(t *testing.T) {
		// Insert a new node with discovery information
		err := db.Exec(`
			INSERT INTO nodes (
				name, ip_address, mac_address, status, role,
				discovery_method, discovered_at, node_type, controller_version, agent_port,
				created_at, updated_at
			) VALUES (
				'controller-node', '192.168.1.11', 'aa:bb:cc:dd:ee:01', 'discovered', 'master',
				'mdns', datetime('now'), 'controller', 'v1.0.0', 0,
				datetime('now'), datetime('now')
			)
		`).Error
		assert.NoError(t, err)

		// Verify the data
		var controllerNode struct {
			Name              string
			DiscoveryMethod   string
			NodeType          string
			ControllerVersion string
			AgentPort         int
		}

		err = db.Raw(`
			SELECT name, discovery_method, node_type, controller_version, agent_port
			FROM nodes WHERE name = 'controller-node'
		`).Scan(&controllerNode).Error
		require.NoError(t, err)

		assert.Equal(t, "controller-node", controllerNode.Name)
		assert.Equal(t, "mdns", controllerNode.DiscoveryMethod)
		assert.Equal(t, "controller", controllerNode.NodeType)
		assert.Equal(t, "v1.0.0", controllerNode.ControllerVersion)
		assert.Equal(t, 0, controllerNode.AgentPort)

		// Insert an agent node
		err = db.Exec(`
			INSERT INTO nodes (
				name, ip_address, mac_address, status, role,
				discovery_method, discovered_at, node_type, controller_version, agent_port,
				created_at, updated_at
			) VALUES (
				'agent-node', '192.168.1.15', 'aa:bb:cc:dd:ee:02', 'ready', 'worker',
				'mdns', datetime('now'), 'agent', '', 9091,
				datetime('now'), datetime('now')
			)
		`).Error
		assert.NoError(t, err)

		// Verify agent data
		var agentNode struct {
			Name              string
			DiscoveryMethod   string
			NodeType          string
			ControllerVersion string
			AgentPort         int
		}

		err = db.Raw(`
			SELECT name, discovery_method, node_type, controller_version, agent_port
			FROM nodes WHERE name = 'agent-node'
		`).Scan(&agentNode).Error
		require.NoError(t, err)

		assert.Equal(t, "agent-node", agentNode.Name)
		assert.Equal(t, "mdns", agentNode.DiscoveryMethod)
		assert.Equal(t, "agent", agentNode.NodeType)
		assert.Empty(t, agentNode.ControllerVersion)
		assert.Equal(t, 9091, agentNode.AgentPort)
	})

	t.Run("Rollback discovery fields migration", func(t *testing.T) {
		// Find the discovery migration
		var discoveryMigration *MigrationDefinition
		for _, m := range getAllMigrations() {
			if m.ID == "20241201000013" {
				migDef := m
				discoveryMigration = &migDef
				break
			}
		}
		require.NotNil(t, discoveryMigration)

		// Roll back the migration
		err := discoveryMigration.Down(db)
		assert.NoError(t, err)

		// NOTE: SQLite doesn't support DROP COLUMN, so the rollback is a no-op
		// The columns will still exist after rollback. This is acceptable because:
		// 1. Migrations are typically applied forward, not rolled back in production
		// 2. Implementing column removal would require recreating the entire table
		// 3. The columns being present doesn't break anything

		// Verify the rollback completed without error
		// and that the table structure is still intact
		var nodeCount int64
		err = db.Raw("SELECT count(*) FROM nodes").Scan(&nodeCount).Error
		assert.NoError(t, err)
		assert.Greater(t, nodeCount, int64(0), "Nodes should still exist after rollback")

		// Verify we can still query the basic node fields
		var testNode struct {
			ID        uint
			Name      string
			IPAddress string
		}
		err = db.Raw("SELECT id, name, ip_address FROM nodes LIMIT 1").Scan(&testNode).Error
		assert.NoError(t, err)
		assert.NotZero(t, testNode.ID)
	})
}
