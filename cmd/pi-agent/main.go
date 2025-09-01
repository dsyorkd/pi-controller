package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/spenceryork/pi-controller/internal/config"
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
	Use:   "pi-agent",
	Short: "Pi Agent - Node-level agent for Pi Controller",
	Long: `Pi Agent runs on individual Raspberry Pi nodes and provides local GPIO access,
system monitoring, and communication with the Pi Controller master server.`,
	RunE: runAgent,
}

var (
	configFile    string
	logLevel      string
	logFormat     string
	serverAddress string
	nodeID        string
)

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&logFormat, "log-format", "json", "log format (json, text)")
	rootCmd.Flags().StringVar(&serverAddress, "server", "", "Pi Controller server address")
	rootCmd.Flags().StringVar(&nodeID, "node-id", "", "unique node identifier")
	
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Pi Agent %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
	},
}

func runAgent(cmd *cobra.Command, args []string) error {
	// Setup logger
	logger := setupLogger()
	
	logger.WithFields(logrus.Fields{
		"version": version,
		"commit":  commit,
		"date":    date,
	}).Info("Starting Pi Agent")

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.Info("Configuration loaded")

	// TODO: Implement agent functionality
	// This would typically include:
	// 1. GPIO hardware interface initialization
	// 2. System monitoring setup
	// 3. Connection to Pi Controller server
	// 4. Registration with cluster
	// 5. Periodic health reporting
	// 6. GPIO command execution

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Pi Agent started successfully")

	// Main agent loop (placeholder)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Periodic tasks like health reporting
			logger.Debug("Agent heartbeat")
			
			// TODO: 
			// - Report system metrics to server
			// - Update GPIO device states
			// - Check for pending commands
			// - Send status updates

		case sig := <-sigChan:
			logger.WithField("signal", sig).Info("Received shutdown signal")
			cancel()
			
		case <-ctx.Done():
			logger.Info("Pi Agent shutdown complete")
			return nil
		}
	}
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