package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dsyorkd/pi-controller/internal/errors"
	applogger "github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/migrations"
	"github.com/dsyorkd/pi-controller/internal/models"
)

// Database wraps GORM database connection with additional functionality
type Database struct {
	db     *gorm.DB
	logger applogger.Interface
}

// Config holds database configuration
type Config struct {
	Path            string `yaml:"path" mapstructure:"path"`
	MaxOpenConns    int    `yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	LogLevel        string `yaml:"log_level" mapstructure:"log_level"`
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	return &Config{
		Path:            "data/pi-controller.db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: "5m",
		LogLevel:        "warn",
	}
}

// New creates a new database connection
func New(config *Config, logger applogger.Interface) (*Database, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directory exists
	if err := ensureDirExists(filepath.Dir(config.Path)); err != nil {
		return nil, errors.Wrapf(err, "failed to create database directory")
	}

	// Configure GORM logger
	gormLogger := logger.WithField("component", "database")
	var logLevel func(msg string, args ...interface{})
	switch config.LogLevel {
	case "error":
		logLevel = gormLogger.Error
	case "warn":
		logLevel = gormLogger.Warn
	case "info":
		logLevel = gormLogger.Info
	default:
		logLevel = gormLogger.Info
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(config.Path), &gorm.Config{
		Logger: &gormSlogAdapter{
			logger: gormLogger,
			level:  logLevel,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to database")
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get underlying sql.DB")
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	
	if config.ConnMaxLifetime != "" {
		duration, err := time.ParseDuration(config.ConnMaxLifetime)
		if err != nil {
			logger.Warnf("Invalid conn_max_lifetime '%s', using default 5m", config.ConnMaxLifetime)
			duration = 5 * time.Minute
		}
		sqlDB.SetConnMaxLifetime(duration)
	}

	database := &Database{
		db:     db,
		logger: logger,
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		return nil, errors.Wrapf(err, "failed to migrate database")
	}

	logger.WithField("path", config.Path).Info("Database connection established")
	return database, nil
}

// NewWithoutMigration creates a new database connection without running migrations
// This is useful for migration commands that need to manage migrations explicitly
func NewWithoutMigration(config *Config, logger applogger.Interface) (*Database, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directory exists
	if err := ensureDirExists(filepath.Dir(config.Path)); err != nil {
		return nil, errors.Wrapf(err, "failed to create database directory")
	}

	// Configure GORM logger
	gormLogger := logger.WithField("component", "database")
	var logLevel func(msg string, args ...interface{})
	switch config.LogLevel {
	case "error":
		logLevel = gormLogger.Error
	case "warn":
		logLevel = gormLogger.Warn
	case "info":
		logLevel = gormLogger.Info
	default:
		logLevel = gormLogger.Info
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(config.Path), &gorm.Config{
		Logger: &gormSlogAdapter{
			logger: gormLogger,
			level:  logLevel,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to database")
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get underlying sql.DB")
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	
	if config.ConnMaxLifetime != "" {
		duration, err := time.ParseDuration(config.ConnMaxLifetime)
		if err != nil {
			logger.Warnf("Invalid conn_max_lifetime '%s', using default 5m", config.ConnMaxLifetime)
			duration = 5 * time.Minute
		}
		sqlDB.SetConnMaxLifetime(duration)
	}

	database := &Database{
		db:     db,
		logger: logger,
	}

	// NOTE: We deliberately do NOT run migrations here
	// This function is for migration commands that manage migrations explicitly

	logger.WithField("path", config.Path).Info("Database connection established (without migrations)")
	return database, nil
}

// DB returns the underlying GORM database instance
func (d *Database) DB() *gorm.DB {
	return d.db
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks database connectivity
func (d *Database) Health() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// migrate runs database migrations
func (d *Database) migrate() error {
	d.logger.Info("Running database migrations")
	
	// Use the new migration system
	migrator := migrations.NewMigrator(d.db, d.logger)
	
	// Validate migration order first
	if err := migrator.ValidateMigrationOrder(); err != nil {
		return errors.Wrapf(err, "migration validation failed")
	}
	
	// Run migrations
	if err := migrator.Up(); err != nil {
		return errors.Wrapf(err, "failed to run migrations")
	}
	
	d.logger.Info("Database migrations completed successfully")
	return nil
}

// BeginTx starts a new transaction
func (d *Database) BeginTx() *gorm.DB {
	return d.db.Begin()
}

// WithTx executes a function within a transaction
func (d *Database) WithTx(fn func(tx *gorm.DB) error) error {
	tx := d.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}

// ensureDirExists creates directory if it doesn't exist
func ensureDirExists(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path %s exists but is not a directory", dir)
		}
		return nil
	}
	
	if !os.IsNotExist(err) {
		return err
	}
	
	return os.MkdirAll(dir, 0755)
}

// gormSlogAdapter adapts our structured logger to GORM logger interface
type gormSlogAdapter struct {
	logger applogger.Interface
	level  func(msg string, args ...interface{})
}

func (g *gormSlogAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *gormSlogAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	g.logger.Infof(msg, data...)
}

func (g *gormSlogAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	g.logger.Warnf(msg, data...)
}

func (g *gormSlogAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	g.logger.Errorf(msg, data...)
}

func (g *gormSlogAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	
	fields := map[string]interface{}{
		"duration": elapsed.String(),
		"rows":     rows,
		"sql":      sql,
	}
	
	if err != nil {
		g.logger.WithFields(fields).WithError(err).Error("Database query failed")
	} else {
		g.logger.WithFields(fields).Debug("Database query executed")
	}
}

// NewForTest creates a new database connection for testing without running migrations
func NewForTest(logger applogger.Interface) (*Database, error) {
	config := &Config{
		Path:            ":memory:",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "5m",
		LogLevel:        "error", // Reduce noise in tests
	}

	db, err := gorm.Open(sqlite.Open(config.Path), &gorm.Config{
		Logger: &gormSlogAdapter{
			logger: logger.WithField("component", "database"),
			level:  logger.Error,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to test database")
	}

	database := &Database{
		db:     db,
		logger: logger,
	}

	return database, nil
}

// NewForTestWithDB creates a new database instance using an existing gorm.DB for testing
func NewForTestWithDB(db *gorm.DB, logger applogger.Interface) *Database {
	return &Database{
		db:     db,
		logger: logger,
	}
}

// Cluster CRUD operations

// CreateCluster creates a new cluster in the database
func (d *Database) CreateCluster(cluster *models.Cluster) error {
	return d.db.Create(cluster).Error
}

// GetCluster retrieves a cluster by ID
func (d *Database) GetCluster(id uint) (*models.Cluster, error) {
	var cluster models.Cluster
	err := d.db.First(&cluster, id).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

// GetClusters retrieves all clusters
func (d *Database) GetClusters() ([]*models.Cluster, error) {
	var clusters []*models.Cluster
	err := d.db.Find(&clusters).Error
	return clusters, err
}

// UpdateCluster updates a cluster in the database
func (d *Database) UpdateCluster(cluster *models.Cluster) error {
	return d.db.Save(cluster).Error
}

// DeleteCluster deletes a cluster by ID
func (d *Database) DeleteCluster(id uint) error {
	return d.db.Delete(&models.Cluster{}, id).Error
}