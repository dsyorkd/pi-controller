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

	"github.com/spenceryork/pi-controller/internal/api"
	"github.com/spenceryork/pi-controller/internal/config"
	grpcserver "github.com/spenceryork/pi-controller/internal/grpc/server"
	"github.com/spenceryork/pi-controller/internal/storage"
	"github.com/spenceryork/pi-controller/internal/websocket"
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

func runServer(cmd *cobra.Command, args []string) error {
	// Setup logger
	logger := setupLogger()
	
	logger.WithFields(logrus.Fields{
		"version": version,
		"commit":  commit,
		"date":    date,
	}).Info("Starting Pi Controller")

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.WithField("config", cfg.App.DataDir).Info("Configuration loaded")

	// Initialize database
	db, err := storage.New(&cfg.Database, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	logger.Info("Database initialized successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start servers
	var wg sync.WaitGroup
	serverErrors := make(chan error, 3)

	// Start REST API server
	apiServer := api.New(&cfg.API, logger, db)
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting REST API server")
		if err := apiServer.Start(); err != nil {
			serverErrors <- fmt.Errorf("API server error: %w", err)
		}
	}()

	// Start gRPC server
	grpcServer, err := grpcserver.New(&cfg.GRPC, logger, db)
	if err != nil {
		return fmt.Errorf("failed to create gRPC server: %w", err)
	}
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting gRPC server")
		if err := grpcServer.Start(); err != nil {
			serverErrors <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Start WebSocket server
	wsServer := websocket.New(&cfg.WebSocket, logger, db)
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting WebSocket server")
		if err := wsServer.Start(); err != nil {
			serverErrors <- fmt.Errorf("WebSocket server error: %w", err)
		}
	}()

	logger.Info("All servers started successfully")

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig).Info("Received shutdown signal")
	case err := <-serverErrors:
		logger.WithError(err).Error("Server error occurred")
		cancel()
	}

	// Graceful shutdown
	logger.Info("Initiating graceful shutdown...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop servers
	go func() {
		if err := apiServer.Stop(shutdownCtx); err != nil {
			logger.WithError(err).Error("Error stopping API server")
		}
	}()

	go func() {
		grpcServer.Stop()
	}()

	go func() {
		if err := wsServer.Stop(shutdownCtx); err != nil {
			logger.WithError(err).Error("Error stopping WebSocket server")
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
		logger.Info("All servers stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded")
	}

	logger.Info("Pi Controller shutdown complete")
	return nil
}

func setupLogger() *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logger.Warnf("Invalid log level '%s', using info", logLevel)
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	switch logFormat {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	default:
		logger.Warnf("Invalid log format '%s', using json", logFormat)
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}

	return logger
}