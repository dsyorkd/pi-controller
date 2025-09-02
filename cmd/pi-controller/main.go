package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/dsyorkd/pi-controller/internal/api"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/errors"
	grpcserver "github.com/dsyorkd/pi-controller/internal/grpc/server"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/migrations"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/dsyorkd/pi-controller/internal/websocket"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "pi-controller",
	Short: "Pi Controller - Raspberry Pi Kubernetes cluster management tool",
	Long: `Pi Controller is a comprehensive tool for managing Raspberry Pi Kubernetes clusters
with GPIO-as-a-Service capabilities. It provides REST API, gRPC, and WebSocket interfaces
for cluster management, node provisioning, and real-time GPIO control.`,
	RunE: runServer,
}

var (
	configFile string
	logLevel   string
	logFormat  string
)

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&logFormat, "log-format", "json", "log format (json, text)")
	
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(migrateCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Pi Controller %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
	Long:  `Database migration commands for managing database schema changes`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run pending migrations",
	Long:  `Apply all pending database migrations`,
	RunE:  runMigrateUp,
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the last migration",
	Long:  `Rollback the most recently applied migration`,
	RunE:  runMigrateDown,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  `Display the status of all migrations`,
	RunE:  runMigrateStatus,
}

var migrateResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset database (DANGEROUS)",
	Long:  `Drop all tables and reapply all migrations. WARNING: This destroys all data!`,
	RunE:  runMigrateReset,
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateResetCmd)
	
	// Add confirmation flag for reset command
	migrateResetCmd.Flags().Bool("confirm", false, "Confirm destructive reset operation")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Setup logger
	log, err := setupLogger()
	if err != nil {
		return errors.Wrapf(err, "failed to setup logger")
	}
	
	log.WithFields(map[string]interface{}{
		"version": version,
		"commit":  commit,
		"date":    date,
	}).Info("Starting Pi Controller")

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return errors.Wrapf(err, "failed to load config")
	}

	log.WithField("config", cfg.App.DataDir).Info("Configuration loaded")

	// Initialize database
	db, err := storage.New(&cfg.Database, log)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize database")
	}
	defer db.Close()

	log.Info("Database initialized successfully")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start servers
	var wg sync.WaitGroup
	serverErrors := make(chan error, 3)

	// Start REST API server
	apiServer := api.New(&cfg.API, log, db)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting REST API server")
		if err := apiServer.Start(); err != nil {
			serverErrors <- errors.Wrapf(err, "API server error")
		}
	}()

	// Start gRPC server
	grpcServer, err := grpcserver.New(&cfg.GRPC, log, db)
	if err != nil {
		return errors.Wrapf(err, "failed to create gRPC server")
	}
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting gRPC server")
		if err := grpcServer.Start(); err != nil {
			serverErrors <- errors.Wrapf(err, "gRPC server error")
		}
	}()

	// Start WebSocket server
	wsServer := websocket.New(&cfg.WebSocket, log, db)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting WebSocket server")
		if err := wsServer.Start(); err != nil {
			serverErrors <- errors.Wrapf(err, "WebSocket server error")
		}
	}()

	log.Info("All servers started successfully")

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.WithField("signal", sig.String()).Info("Received shutdown signal")
	case err := <-serverErrors:
		log.WithError(err).Error("Server error occurred")
	}

	// Graceful shutdown
	log.Info("Initiating graceful shutdown...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop servers
	go func() {
		if err := apiServer.Stop(shutdownCtx); err != nil {
			log.WithError(err).Error("Error stopping API server")
		}
	}()

	go func() {
		grpcServer.Stop()
	}()

	go func() {
		if err := wsServer.Stop(shutdownCtx); err != nil {
			log.WithError(err).Error("Error stopping WebSocket server")
		}
	}()

	// Wait for all servers to stop or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("All servers stopped gracefully")
	case <-shutdownCtx.Done():
		log.Warn("Shutdown timeout exceeded")
	}

	log.Info("Pi Controller shutdown complete")
	return nil
}

func setupLogger() (*logger.Logger, error) {
	cfg := logger.Config{
		Level:  logLevel,
		Format: logFormat,
		Output: "stdout",
	}

	log, err := logger.New(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create logger")
	}

	// Set as default logger
	logger.SetDefault(log)
	
	return log, nil
}

// Migration command handlers

func runMigrateUp(cmd *cobra.Command, args []string) error {
	log, db, err := setupMigrationEnvironment()
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrations.NewMigrator(db.DB(), log)
	
	log.Info("Running database migrations...")
	if err := migrator.Up(); err != nil {
		return errors.Wrapf(err, "failed to run migrations")
	}

	log.Info("Migrations completed successfully")
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	log, db, err := setupMigrationEnvironment()
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrations.NewMigrator(db.DB(), log)
	
	log.Info("Rolling back last migration...")
	if err := migrator.Down(); err != nil {
		return errors.Wrapf(err, "failed to rollback migration")
	}

	log.Info("Migration rollback completed successfully")
	return nil
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	log, db, err := setupMigrationEnvironment()
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrations.NewMigrator(db.DB(), log)
	
	statuses, err := migrator.Status()
	if err != nil {
		return errors.Wrapf(err, "failed to get migration status")
	}

	if len(statuses) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	fmt.Println("Migration Status:")
	fmt.Println("=================")
	for _, status := range statuses {
		statusStr := "PENDING"
		appliedAt := ""
		if status.Applied {
			statusStr = "APPLIED"
			if status.AppliedAt != nil {
				appliedAt = fmt.Sprintf(" (applied at %s)", status.AppliedAt.Format("2006-01-02 15:04:05"))
			}
		}
		fmt.Printf("%-15s %s - %s%s\n", status.ID, statusStr, status.Description, appliedAt)
	}

	return nil
}

func runMigrateReset(cmd *cobra.Command, args []string) error {
	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		return fmt.Errorf("reset operation requires --confirm flag due to destructive nature")
	}

	log, db, err := setupMigrationEnvironment()
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrations.NewMigrator(db.DB(), log)
	
	log.Warn("DANGER: Resetting database - all data will be lost!")
	if err := migrator.Reset(); err != nil {
		return errors.Wrapf(err, "failed to reset database")
	}

	log.Info("Database reset completed successfully")
	return nil
}

func setupMigrationEnvironment() (*logger.Logger, *storage.Database, error) {
	// Setup logger
	log, err := setupLogger()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to setup logger")
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load config")
	}

	// Initialize database without auto-migration
	// We'll handle migrations explicitly through the migrator
	db, err := setupDatabaseForMigration(&cfg.Database, log)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to initialize database")
	}

	return log, db, nil
}

func setupDatabaseForMigration(config *storage.Config, logger *logger.Logger) (*storage.Database, error) {
	// This is a modified version of storage.New that doesn't run auto-migrations
	if config == nil {
		config = storage.DefaultConfig()
	}

	// Initialize database without running migrations
	db, err := storage.NewWithoutMigration(config, logger)
	if err != nil {
		return nil, err
	}

	return db, nil
}