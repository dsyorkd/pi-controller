package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/spenceryork/pi-controller/internal/models"
)

// Database wraps GORM database connection with additional functionality
type Database struct {
	db     *gorm.DB
	logger *logrus.Logger
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
func New(config *Config, logger *logrus.Logger) (*Database, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directory exists
	if err := ensureDirExists(filepath.Dir(config.Path)); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM logger
	gormLogger := logger.WithField("component", "database")
	logLevel := logger.Info
	switch config.LogLevel {
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(config.Path), &gorm.Config{
		Logger: &gormLogrusAdapter{
			logger: gormLogger,
			level:  logLevel,
		},
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
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
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.WithField("path", config.Path).Info("Database connection established")
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
	
	err := d.db.AutoMigrate(
		&models.Cluster{},
		&models.Node{},
		&models.GPIODevice{},
		&models.GPIOReading{},
	)
	if err != nil {
		return fmt.Errorf("failed to run auto-migration: %w", err)
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

// gormLogrusAdapter adapts logrus logger to GORM logger interface
type gormLogrusAdapter struct {
	logger *logrus.Entry
	level  func(args ...interface{})
}

func (g *gormLogrusAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *gormLogrusAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	g.logger.Infof(msg, data...)
}

func (g *gormLogrusAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	g.logger.Warnf(msg, data...)
}

func (g *gormLogrusAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	g.logger.Errorf(msg, data...)
}

func (g *gormLogrusAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	
	fields := logrus.Fields{
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