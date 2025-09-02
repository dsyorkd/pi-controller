package migrations

import (
	"gorm.io/gorm"
)

// getAllMigrations returns all migration definitions in chronological order
func getAllMigrations() []MigrationDefinition {
	return []MigrationDefinition{
		{
			ID:          "20241201000001",
			Description: "Create clusters table",
			Up:          createClustersTable,
			Down:        dropClustersTable,
		},
		{
			ID:          "20241201000002",
			Description: "Create nodes table",
			Up:          createNodesTable,
			Down:        dropNodesTable,
		},
		{
			ID:          "20241201000003",
			Description: "Create gpio_devices table",
			Up:          createGPIODevicesTable,
			Down:        dropGPIODevicesTable,
		},
		{
			ID:          "20241201000004",
			Description: "Create gpio_readings table",
			Up:          createGPIOReadingsTable,
			Down:        dropGPIOReadingsTable,
		},
		{
			ID:          "20241201000005",
			Description: "Add indexes for performance optimization",
			Up:          addPerformanceIndexes,
			Down:        dropPerformanceIndexes,
		},
	}
}

// createClustersTable creates the clusters table
func createClustersTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS clusters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		status TEXT DEFAULT 'pending' NOT NULL,
		version TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		kube_config TEXT,
		master_endpoint TEXT
	);
	
	CREATE UNIQUE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name);
	CREATE INDEX IF NOT EXISTS idx_clusters_deleted_at ON clusters(deleted_at);
	CREATE INDEX IF NOT EXISTS idx_clusters_status ON clusters(status);
	`
	
	return db.Exec(sql).Error
}

// dropClustersTable drops the clusters table
func dropClustersTable(db *gorm.DB) error {
	sql := `
	DROP INDEX IF EXISTS idx_clusters_status;
	DROP INDEX IF EXISTS idx_clusters_deleted_at;
	DROP INDEX IF EXISTS idx_clusters_name;
	DROP TABLE IF EXISTS clusters;
	`
	
	return db.Exec(sql).Error
}

// createNodesTable creates the nodes table
func createNodesTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS nodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		ip_address TEXT NOT NULL,
		mac_address TEXT UNIQUE,
		status TEXT DEFAULT 'discovered' NOT NULL,
		role TEXT DEFAULT 'worker' NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		architecture TEXT,
		model TEXT,
		serial_number TEXT,
		cpu_cores INTEGER,
		memory INTEGER,
		cluster_id INTEGER,
		kube_version TEXT,
		node_name TEXT,
		os_version TEXT,
		kernel_version TEXT,
		last_seen DATETIME,
		FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE SET NULL
	);
	
	CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_name ON nodes(name);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_mac_address ON nodes(mac_address);
	CREATE INDEX IF NOT EXISTS idx_nodes_deleted_at ON nodes(deleted_at);
	CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);
	CREATE INDEX IF NOT EXISTS idx_nodes_cluster_id ON nodes(cluster_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_ip_address ON nodes(ip_address);
	CREATE INDEX IF NOT EXISTS idx_nodes_last_seen ON nodes(last_seen);
	`
	
	return db.Exec(sql).Error
}

// dropNodesTable drops the nodes table
func dropNodesTable(db *gorm.DB) error {
	sql := `
	DROP INDEX IF EXISTS idx_nodes_last_seen;
	DROP INDEX IF EXISTS idx_nodes_ip_address;
	DROP INDEX IF EXISTS idx_nodes_cluster_id;
	DROP INDEX IF EXISTS idx_nodes_status;
	DROP INDEX IF EXISTS idx_nodes_deleted_at;
	DROP INDEX IF EXISTS idx_nodes_mac_address;
	DROP INDEX IF EXISTS idx_nodes_name;
	DROP TABLE IF EXISTS nodes;
	`
	
	return db.Exec(sql).Error
}

// createGPIODevicesTable creates the gpio_devices table
func createGPIODevicesTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS gpio_devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		pin_number INTEGER NOT NULL,
		direction TEXT DEFAULT 'input' NOT NULL,
		pull_mode TEXT DEFAULT 'none' NOT NULL,
		value INTEGER DEFAULT 0 NOT NULL,
		device_type TEXT DEFAULT 'digital' NOT NULL,
		status TEXT DEFAULT 'active' NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		node_id INTEGER NOT NULL,
		-- GPIO Config embedded fields
		frequency INTEGER,
		duty_cycle INTEGER,
		spi_mode INTEGER,
		spi_bits INTEGER,
		spi_speed INTEGER,
		spi_channel INTEGER,
		i2c_address INTEGER,
		i2c_bus INTEGER,
		sample_rate INTEGER,
		FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_node_id ON gpio_devices(node_id);
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_pin_number ON gpio_devices(pin_number);
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_status ON gpio_devices(status);
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_deleted_at ON gpio_devices(deleted_at);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_gpio_devices_node_pin ON gpio_devices(node_id, pin_number) WHERE deleted_at IS NULL;
	`
	
	return db.Exec(sql).Error
}

// dropGPIODevicesTable drops the gpio_devices table
func dropGPIODevicesTable(db *gorm.DB) error {
	sql := `
	DROP INDEX IF EXISTS idx_gpio_devices_node_pin;
	DROP INDEX IF EXISTS idx_gpio_devices_deleted_at;
	DROP INDEX IF EXISTS idx_gpio_devices_status;
	DROP INDEX IF EXISTS idx_gpio_devices_pin_number;
	DROP INDEX IF EXISTS idx_gpio_devices_node_id;
	DROP TABLE IF EXISTS gpio_devices;
	`
	
	return db.Exec(sql).Error
}

// createGPIOReadingsTable creates the gpio_readings table
func createGPIOReadingsTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS gpio_readings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		value REAL NOT NULL,
		timestamp DATETIME NOT NULL,
		FOREIGN KEY (device_id) REFERENCES gpio_devices(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_gpio_readings_device_id ON gpio_readings(device_id);
	CREATE INDEX IF NOT EXISTS idx_gpio_readings_timestamp ON gpio_readings(timestamp);
	CREATE INDEX IF NOT EXISTS idx_gpio_readings_device_timestamp ON gpio_readings(device_id, timestamp);
	`
	
	return db.Exec(sql).Error
}

// dropGPIOReadingsTable drops the gpio_readings table
func dropGPIOReadingsTable(db *gorm.DB) error {
	sql := `
	DROP INDEX IF EXISTS idx_gpio_readings_device_timestamp;
	DROP INDEX IF EXISTS idx_gpio_readings_timestamp;
	DROP INDEX IF EXISTS idx_gpio_readings_device_id;
	DROP TABLE IF EXISTS gpio_readings;
	`
	
	return db.Exec(sql).Error
}

// addPerformanceIndexes adds additional indexes for performance optimization
func addPerformanceIndexes(db *gorm.DB) error {
	sql := `
	-- Composite indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_nodes_cluster_status ON nodes(cluster_id, status) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_node_status ON gpio_devices(node_id, status) WHERE deleted_at IS NULL;
	
	-- Indexes for time-series queries
	CREATE INDEX IF NOT EXISTS idx_gpio_readings_timestamp_desc ON gpio_readings(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_gpio_readings_device_time_desc ON gpio_readings(device_id, timestamp DESC);
	
	-- Index for active clusters with ready nodes
	CREATE INDEX IF NOT EXISTS idx_clusters_active ON clusters(status) WHERE status = 'active' AND deleted_at IS NULL;
	
	-- Index for recent readings (last 24 hours pattern)
	CREATE INDEX IF NOT EXISTS idx_gpio_readings_recent ON gpio_readings(timestamp) WHERE timestamp > datetime('now', '-1 day');
	`
	
	return db.Exec(sql).Error
}

// dropPerformanceIndexes removes the performance optimization indexes
func dropPerformanceIndexes(db *gorm.DB) error {
	sql := `
	DROP INDEX IF EXISTS idx_gpio_readings_recent;
	DROP INDEX IF EXISTS idx_clusters_active;
	DROP INDEX IF EXISTS idx_gpio_readings_device_time_desc;
	DROP INDEX IF EXISTS idx_gpio_readings_timestamp_desc;
	DROP INDEX IF EXISTS idx_gpio_devices_node_status;
	DROP INDEX IF EXISTS idx_nodes_cluster_status;
	`
	
	return db.Exec(sql).Error
}