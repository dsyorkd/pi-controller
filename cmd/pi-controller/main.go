package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/dsyorkd/pi-controller/internal/api"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/errors"
	grpcserver "github.com/dsyorkd/pi-controller/internal/grpc/server"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/migrations"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/dsyorkd/pi-controller/internal/websocket"
	"github.com/dsyorkd/pi-controller/pkg/discovery"
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
	serverErrors := make(chan error, 4)

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

	// Start Discovery Service
	var discoveryService *discovery.Service
	if cfg.Discovery.Enabled {
		discoveryConfig := &discovery.Config{
			Enabled:     cfg.Discovery.Enabled,
			Method:      cfg.Discovery.Method,
			Interface:   cfg.Discovery.Interface,
			Port:        cfg.Discovery.Port,
			Interval:    cfg.Discovery.Interval,
			Timeout:     cfg.Discovery.Timeout,
			StaticNodes: cfg.Discovery.StaticNodes,
			ServiceName: cfg.Discovery.ServiceName,
			ServiceType: cfg.Discovery.ServiceType,
		}

		// Create a logrus logger for discovery service
		logrusLogger := logrus.New()
		logrusLogger.SetLevel(logrus.InfoLevel)
		logrusLogger.SetFormatter(&logrus.JSONFormatter{})

		discoveryService, err = discovery.NewService(discoveryConfig, logrusLogger)
		if err != nil {
			return errors.Wrapf(err, "failed to create discovery service")
		}

		// Add event handler to process discovered nodes
		discoveryService.AddEventHandler(func(event discovery.NodeEvent) {
			log.WithFields(map[string]interface{}{
				"type":       event.Type,
				"node_id":    event.Node.ID,
				"node_name":  event.Node.Name,
				"ip_address": event.Node.IPAddress,
				"port":       event.Node.Port,
			}).Info("Node discovery event")

			// Handle automatic node registration
			if err := handleNodeDiscoveryEvent(event, db, log); err != nil {
				log.WithError(err).WithFields(map[string]interface{}{
					"type":       event.Type,
					"node_id":    event.Node.ID,
					"ip_address": event.Node.IPAddress,
				}).Error("Failed to handle node discovery event")
			}
		})

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("Starting discovery service")
			if err := discoveryService.Start(context.Background()); err != nil {
				serverErrors <- errors.Wrapf(err, "Discovery service error")
			}
		}()
	}

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

	// Stop discovery service
	if discoveryService != nil {
		go func() {
			if err := discoveryService.Stop(); err != nil {
				log.WithError(err).Error("Error stopping discovery service")
			}
		}()
	}

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

// handleNodeDiscoveryEvent processes node discovery events and automatically registers new nodes
func handleNodeDiscoveryEvent(event discovery.NodeEvent, db *storage.Database, log *logger.Logger) error {
	switch event.Type {
	case discovery.NodeDiscovered:
		return handleNodeRegistration(event, db, log)
	case discovery.NodeLost:
		return handleNodeLost(event, db, log)
	case discovery.NodeUpdated:
		return handleNodeUpdate(event, db, log)
	default:
		log.WithField("event_type", event.Type).Debug("Ignoring unhandled discovery event type")
		return nil
	}
}

// handleNodeRegistration processes new node discovery and registration
func handleNodeRegistration(event discovery.NodeEvent, db *storage.Database, log *logger.Logger) error {

	node := event.Node
	log.WithFields(map[string]interface{}{
		"node_id":    node.ID,
		"node_name":  node.Name,
		"ip_address": node.IPAddress,
		"port":       node.Port,
	}).Info("Processing automatic node registration")

	// Create node service
	nodeService := services.NewNodeService(db, log)

	// Check if node already exists by IP address
	existingNode, err := nodeService.GetByIPAddress(node.IPAddress)
	if err != nil && err != services.ErrNotFound {
		return errors.Wrapf(err, "failed to check for existing node")
	}

	if existingNode != nil {
		// Node already exists, update last seen timestamp
		if err := nodeService.UpdateLastSeen(existingNode.ID); err != nil {
			return errors.Wrapf(err, "failed to update last seen for existing node")
		}
		
		log.WithFields(map[string]interface{}{
			"node_id":      existingNode.ID,
			"node_name":    existingNode.Name,
			"ip_address":   existingNode.IPAddress,
			"discovery_id": node.ID,
		}).Info("Updated existing node last seen timestamp")
		
		return nil
	}

	// Extract node information from TXT records
	architecture := node.TXTRecords["arch"]
	model := node.TXTRecords["model"]
	version := node.TXTRecords["version"]
	nodeIdFromTXT := node.TXTRecords["node_id"]
	
	// Use discovery node_id if available from TXT records, otherwise use discovery ID
	nodeName := node.Name
	if nodeIdFromTXT != "" {
		nodeName = nodeIdFromTXT
	}

	// Default role to worker - could be enhanced to detect master nodes
	role := models.NodeRoleWorker

	// Create new node registration request
	createReq := services.CreateNodeRequest{
		Name:         nodeName,
		IPAddress:    node.IPAddress,
		MACAddress:   "", // Not available from mDNS discovery
		Role:         role,
		Architecture: architecture,
		Model:        model,
		CPUCores:     1, // Default, will be updated when node connects
		Memory:       1024 * 1024 * 1024, // Default 1GB, will be updated when node connects
	}

	// Create the new node
	newNode, err := nodeService.Create(createReq)
	if err != nil {
		return errors.Wrapf(err, "failed to create new node")
	}

	log.WithFields(map[string]interface{}{
		"node_id":      newNode.ID,
		"node_name":    newNode.Name,
		"ip_address":   newNode.IPAddress,
		"discovery_id": node.ID,
		"architecture": architecture,
		"model":        model,
		"version":      version,
	}).Info("Successfully registered new node from discovery")

	// TODO: Generate and distribute client certificates for secure gRPC communication
	// This will be implemented in a future task (Task 28: Certificate Authority)
	
	return nil
}

// handleNodeLost processes node lost events and updates node status
func handleNodeLost(event discovery.NodeEvent, db *storage.Database, log *logger.Logger) error {
	node := event.Node
	
	log.WithFields(map[string]interface{}{
		"node_id":    node.ID,
		"ip_address": node.IPAddress,
	}).Info("Processing node lost event")

	// Create node service
	nodeService := services.NewNodeService(db, log)

	// Find existing node by IP address
	existingNode, err := nodeService.GetByIPAddress(node.IPAddress)
	if err != nil {
		if err == services.ErrNotFound {
			log.WithField("ip_address", node.IPAddress).Debug("Node lost event for unknown node, ignoring")
			return nil
		}
		return errors.Wrapf(err, "failed to find node for lost event")
	}

	// Update node status to unknown since it's no longer responding
	updateReq := services.UpdateNodeRequest{
		Status: &[]models.NodeStatus{models.NodeStatusUnknown}[0],
	}

	_, err = nodeService.Update(existingNode.ID, updateReq)
	if err != nil {
		return errors.Wrapf(err, "failed to update node status on lost event")
	}

	log.WithFields(map[string]interface{}{
		"node_id":      existingNode.ID,
		"node_name":    existingNode.Name,
		"ip_address":   existingNode.IPAddress,
		"new_status":   models.NodeStatusUnknown,
	}).Info("Updated node status due to lost event")

	return nil
}

// handleNodeUpdate processes node update events and refreshes node information
func handleNodeUpdate(event discovery.NodeEvent, db *storage.Database, log *logger.Logger) error {
	node := event.Node
	
	log.WithFields(map[string]interface{}{
		"node_id":    node.ID,
		"ip_address": node.IPAddress,
	}).Debug("Processing node update event")

	// Create node service
	nodeService := services.NewNodeService(db, log)

	// Find existing node by IP address
	existingNode, err := nodeService.GetByIPAddress(node.IPAddress)
	if err != nil {
		if err == services.ErrNotFound {
			// Node doesn't exist yet, treat as discovery
			log.WithField("ip_address", node.IPAddress).Info("Node update event for unknown node, treating as discovery")
			return handleNodeRegistration(event, db, log)
		}
		return errors.Wrapf(err, "failed to find node for update event")
	}

	// Update last seen timestamp and any changed TXT record information
	if err := nodeService.UpdateLastSeen(existingNode.ID); err != nil {
		return errors.Wrapf(err, "failed to update last seen for updated node")
	}

	// Extract updated node information from TXT records
	architecture := node.TXTRecords["arch"]
	model := node.TXTRecords["model"]
	
	updateReq := services.UpdateNodeRequest{}
	hasUpdates := false

	// Update architecture if it has changed
	if architecture != "" && architecture != existingNode.Architecture {
		updateReq.Architecture = &architecture
		hasUpdates = true
	}

	// Update model if it has changed
	if model != "" && model != existingNode.Model {
		updateReq.Model = &model
		hasUpdates = true
	}

	// If the node was marked as unknown, update it back to discovered status
	if existingNode.Status == models.NodeStatusUnknown {
		status := models.NodeStatusDiscovered
		updateReq.Status = &status
		hasUpdates = true
	}

	// Apply updates if any
	if hasUpdates {
		_, err = nodeService.Update(existingNode.ID, updateReq)
		if err != nil {
			return errors.Wrapf(err, "failed to update node information")
		}

		log.WithFields(map[string]interface{}{
			"node_id":      existingNode.ID,
			"node_name":    existingNode.Name,
			"ip_address":   existingNode.IPAddress,
			"architecture": architecture,
			"model":        model,
		}).Info("Updated node information from discovery update")
	} else {
		log.WithFields(map[string]interface{}{
			"node_id":      existingNode.ID,
			"ip_address":   existingNode.IPAddress,
		}).Debug("Node update event processed, no changes detected")
	}

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
