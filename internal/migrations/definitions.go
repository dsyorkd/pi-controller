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
		{
			ID:          "20241201000006",
			Description: "Add GPIO pin reservation fields",
			Up:          addGPIOReservationFields,
			Down:        dropGPIOReservationFields,
		},
		{
			ID:          "20241201000007",
			Description: "Create certificates table",
			Up:          createCertificatesTable,
			Down:        dropCertificatesTable,
		},
		{
			ID:          "20241201000008",
			Description: "Create certificate_requests table",
			Up:          createCertificateRequestsTable,
			Down:        dropCertificateRequestsTable,
		},
		{
			ID:          "20241201000009",
			Description: "Create ca_info table",
			Up:          createCAInfoTable,
			Down:        dropCAInfoTable,
		},
		{
			ID:          "20241201000010",
			Description: "Add certificate indexes for performance",
			Up:          addCertificateIndexes,
			Down:        dropCertificateIndexes,
		},
		{
			ID:          "20241201000011",
			Description: "Create users table for authentication",
			Up:          createUsersTable,
			Down:        dropUsersTable,
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

// addGPIOReservationFields adds reservation tracking fields to gpio_devices table
func addGPIOReservationFields(db *gorm.DB) error {
	sql := `
	-- Add reservation fields to gpio_devices table
	ALTER TABLE gpio_devices ADD COLUMN reserved_by TEXT;
	ALTER TABLE gpio_devices ADD COLUMN reserved_at DATETIME;
	ALTER TABLE gpio_devices ADD COLUMN reservation_ttl DATETIME;
	
	-- Add index for reserved_by field for efficient queries
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_reserved_by ON gpio_devices(reserved_by);
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_reserved_at ON gpio_devices(reserved_at);
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_reservation_ttl ON gpio_devices(reservation_ttl);
	
	-- Composite index for reservation queries
	CREATE INDEX IF NOT EXISTS idx_gpio_devices_reservation_status ON gpio_devices(reserved_by, reservation_ttl) WHERE reserved_by IS NOT NULL;
	`

	return db.Exec(sql).Error
}

// dropGPIOReservationFields removes reservation tracking fields from gpio_devices table
func dropGPIOReservationFields(db *gorm.DB) error {
	sql := `
	-- Drop indexes first
	DROP INDEX IF EXISTS idx_gpio_devices_reservation_status;
	DROP INDEX IF EXISTS idx_gpio_devices_reservation_ttl;
	DROP INDEX IF EXISTS idx_gpio_devices_reserved_at;
	DROP INDEX IF EXISTS idx_gpio_devices_reserved_by;
	
	-- Remove reservation fields (Note: SQLite doesn't support DROP COLUMN directly)
	-- For SQLite, we would need to recreate the table, but this is a rollback scenario
	-- so we'll use a pragmatic approach and just set them to NULL
	UPDATE gpio_devices SET reserved_by = NULL, reserved_at = NULL, reservation_ttl = NULL;
	`

	return db.Exec(sql).Error
}

// Certificate Authority migrations

// createCertificatesTable creates the certificates table
func createCertificatesTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS certificates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		serial_number TEXT NOT NULL UNIQUE,
		common_name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		certificate_pem TEXT NOT NULL,
		subject TEXT NOT NULL,
		issuer TEXT NOT NULL,
		not_before DATETIME NOT NULL,
		not_after DATETIME NOT NULL,
		key_usage TEXT,
		ext_key_usage TEXT,
		sans TEXT,
		backend TEXT NOT NULL,
		vault_path TEXT,
		local_path TEXT,
		node_id INTEGER,
		cluster_id INTEGER,
		renewed_from_id INTEGER,
		auto_renew BOOLEAN NOT NULL DEFAULT 1,
		renewed_at DATETIME,
		revoked_at DATETIME,
		revoked_reason TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		FOREIGN KEY (node_id) REFERENCES nodes(id),
		FOREIGN KEY (cluster_id) REFERENCES clusters(id),
		FOREIGN KEY (renewed_from_id) REFERENCES certificates(id)
	);
	`

	return db.Exec(sql).Error
}

// dropCertificatesTable drops the certificates table
func dropCertificatesTable(db *gorm.DB) error {
	return db.Exec("DROP TABLE IF EXISTS certificates").Error
}

// createCertificateRequestsTable creates the certificate_requests table
func createCertificateRequestsTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS certificate_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		common_name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		csr_pem TEXT NOT NULL,
		sans TEXT,
		validity_period TEXT,
		key_usage TEXT,
		ext_key_usage TEXT,
		node_id INTEGER,
		cluster_id INTEGER,
		processed_at DATETIME,
		certificate_id INTEGER,
		failure_reason TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		FOREIGN KEY (node_id) REFERENCES nodes(id),
		FOREIGN KEY (cluster_id) REFERENCES clusters(id),
		FOREIGN KEY (certificate_id) REFERENCES certificates(id)
	);
	`

	return db.Exec(sql).Error
}

// dropCertificateRequestsTable drops the certificate_requests table
func dropCertificateRequestsTable(db *gorm.DB) error {
	return db.Exec("DROP TABLE IF EXISTS certificate_requests").Error
}

// createCAInfoTable creates the ca_info table
func createCAInfoTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS ca_info (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		type TEXT NOT NULL,
		backend TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		certificate_id INTEGER,
		local_path TEXT,
		vault_path TEXT,
		subject TEXT NOT NULL,
		not_before DATETIME NOT NULL,
		not_after DATETIME NOT NULL,
		serial_number TEXT NOT NULL,
		certificates_issued INTEGER NOT NULL DEFAULT 0,
		certificates_active INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		FOREIGN KEY (certificate_id) REFERENCES certificates(id)
	);
	`

	return db.Exec(sql).Error
}

// dropCAInfoTable drops the ca_info table
func dropCAInfoTable(db *gorm.DB) error {
	return db.Exec("DROP TABLE IF EXISTS ca_info").Error
}

// addCertificateIndexes adds indexes for certificate-related tables
func addCertificateIndexes(db *gorm.DB) error {
	sql := `
	-- Certificates table indexes
	CREATE INDEX IF NOT EXISTS idx_certificates_serial_number ON certificates(serial_number);
	CREATE INDEX IF NOT EXISTS idx_certificates_common_name ON certificates(common_name);
	CREATE INDEX IF NOT EXISTS idx_certificates_status ON certificates(status) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_type ON certificates(type) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_backend ON certificates(backend) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_node_id ON certificates(node_id) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_cluster_id ON certificates(cluster_id) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_not_after ON certificates(not_after);
	CREATE INDEX IF NOT EXISTS idx_certificates_auto_renew ON certificates(auto_renew, not_after) WHERE auto_renew = 1 AND status = 'active';
	
	-- Certificate requests table indexes
	CREATE INDEX IF NOT EXISTS idx_certificate_requests_status ON certificate_requests(status) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificate_requests_type ON certificate_requests(type) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificate_requests_node_id ON certificate_requests(node_id) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificate_requests_cluster_id ON certificate_requests(cluster_id) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificate_requests_created_at ON certificate_requests(created_at DESC);
	
	-- CA info table indexes
	CREATE INDEX IF NOT EXISTS idx_ca_info_name ON ca_info(name);
	CREATE INDEX IF NOT EXISTS idx_ca_info_type ON ca_info(type) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_ca_info_backend ON ca_info(backend) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_ca_info_status ON ca_info(status) WHERE deleted_at IS NULL;
	
	-- Composite indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_certificates_node_status ON certificates(node_id, status) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_cluster_status ON certificates(cluster_id, status) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_certificates_expiring_soon ON certificates(not_after, auto_renew) WHERE status = 'active' AND not_after < datetime('now', '+30 days');
	`

	return db.Exec(sql).Error
}

// dropCertificateIndexes drops certificate-related indexes
func dropCertificateIndexes(db *gorm.DB) error {
	sql := `
	-- Drop composite indexes
	DROP INDEX IF EXISTS idx_certificates_expiring_soon;
	DROP INDEX IF EXISTS idx_certificates_cluster_status;
	DROP INDEX IF EXISTS idx_certificates_node_status;
	
	-- Drop CA info indexes
	DROP INDEX IF EXISTS idx_ca_info_status;
	DROP INDEX IF EXISTS idx_ca_info_backend;
	DROP INDEX IF EXISTS idx_ca_info_type;
	DROP INDEX IF EXISTS idx_ca_info_name;
	
	-- Drop certificate requests indexes
	DROP INDEX IF EXISTS idx_certificate_requests_created_at;
	DROP INDEX IF EXISTS idx_certificate_requests_cluster_id;
	DROP INDEX IF EXISTS idx_certificate_requests_node_id;
	DROP INDEX IF EXISTS idx_certificate_requests_type;
	DROP INDEX IF EXISTS idx_certificate_requests_status;
	
	-- Drop certificates indexes
	DROP INDEX IF EXISTS idx_certificates_auto_renew;
	DROP INDEX IF EXISTS idx_certificates_not_after;
	DROP INDEX IF EXISTS idx_certificates_cluster_id;
	DROP INDEX IF EXISTS idx_certificates_node_id;
	DROP INDEX IF EXISTS idx_certificates_backend;
	DROP INDEX IF EXISTS idx_certificates_type;
	DROP INDEX IF EXISTS idx_certificates_status;
	DROP INDEX IF EXISTS idx_certificates_common_name;
	DROP INDEX IF EXISTS idx_certificates_serial_number;
	`

	return db.Exec(sql).Error
}

// createUsersTable creates the users table for authentication
func createUsersTable(db *gorm.DB) error {
	sql := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		first_name TEXT,
		last_name TEXT,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		last_login DATETIME,
		api_key TEXT,
		api_key_expiry DATETIME,
		failed_logins INTEGER NOT NULL DEFAULT 0,
		locked_until DATETIME,
		password_reset TEXT,
		reset_expiry DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	);
	
	-- Create indexes for performance and security
	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role) WHERE deleted_at IS NULL;
	CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active) WHERE deleted_at IS NULL;
	CREATE UNIQUE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key) WHERE api_key IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users(locked_until) WHERE locked_until IS NOT NULL;
	CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
	
	-- Composite indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_users_active_role ON users(is_active, role) WHERE deleted_at IS NULL;
	
	-- Insert default admin user with password "admin123" (bcrypt hash)
	-- $2a$10$zgl1xM6oW1wR6YaxKa/m/.Cuhvd.kNNdZX1yDqMj7BwymVWwDHtdW
	INSERT OR IGNORE INTO users (username, email, password_hash, role, first_name, last_name, is_active) 
	VALUES ('admin', 'admin@pi-controller.local', '$2a$10$zgl1xM6oW1wR6YaxKa/m/.Cuhvd.kNNdZX1yDqMj7BwymVWwDHtdW', 'admin', 'System', 'Administrator', 1);
	`

	return db.Exec(sql).Error
}

// dropUsersTable drops the users table
func dropUsersTable(db *gorm.DB) error {
	sql := `
	-- Drop indexes first
	DROP INDEX IF EXISTS idx_users_active_role;
	DROP INDEX IF EXISTS idx_users_deleted_at;
	DROP INDEX IF EXISTS idx_users_locked_until;
	DROP INDEX IF EXISTS idx_users_api_key;
	DROP INDEX IF EXISTS idx_users_is_active;
	DROP INDEX IF EXISTS idx_users_role;
	DROP INDEX IF EXISTS idx_users_email;
	DROP INDEX IF EXISTS idx_users_username;
	
	-- Drop table
	DROP TABLE IF EXISTS users;
	`

	return db.Exec(sql).Error
}
