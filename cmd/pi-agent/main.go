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

	"github.com/dsyorkd/pi-controller/internal/agent"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/grpc/client"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/pkg/discovery"
	pb "github.com/dsyorkd/pi-controller/proto"
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
	// Setup structured logger
	structuredLogger, err := setupStructuredLogger()
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}

	structuredLogger.WithFields(map[string]interface{}{
		"version": version,
		"commit":  commit,
		"date":    date,
	}).Info("Starting Pi Agent")

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	structuredLogger.Info("Configuration loaded")

	// Override configuration with command-line flags
	if serverAddress != "" {
		cfg.GRPCClient.ServerAddress = serverAddress
	}
	if nodeID != "" {
		cfg.GRPCClient.NodeID = nodeID
	}

	// Create gRPC client configuration
	grpcConfig, err := client.ConfigFromYAML(cfg.GRPCClient)
	if err != nil {
		return fmt.Errorf("invalid gRPC client config: %w", err)
	}

	// Validate the configuration
	if err := client.ValidateConfig(grpcConfig); err != nil {
		return fmt.Errorf("gRPC client config validation failed: %w", err)
	}

	// Create gRPC client
	grpcClient, err := client.NewClient(grpcConfig, structuredLogger)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}

	// Collect node information
	nodeInfo, err := client.CollectNodeInfo(cfg.GRPCClient.NodeID, cfg.GRPCClient.NodeName)
	if err != nil {
		structuredLogger.WithError(err).Warn("Failed to collect node info, using defaults")
		nodeInfo = &client.NodeInfo{
			ID:   cfg.GRPCClient.NodeID,
			Name: cfg.GRPCClient.NodeName,
		}
	}

	// Set node info in client
	grpcClient.SetNodeInfo(nodeInfo)

	structuredLogger.WithFields(map[string]interface{}{
		"node_id":      nodeInfo.ID,
		"node_name":    nodeInfo.Name,
		"ip_address":   nodeInfo.IPAddress,
		"mac_address":  nodeInfo.MACAddress,
		"architecture": nodeInfo.Architecture,
		"model":        nodeInfo.Model,
		"cpu_cores":    nodeInfo.CPUCores,
	}).Info("Node information collected")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start gRPC client (this will connect and begin heartbeat)
	if err := grpcClient.Start(ctx); err != nil {
		structuredLogger.WithError(err).Error("Failed to start gRPC client")
		return fmt.Errorf("failed to start gRPC client: %w", err)
	}

	// Register node with controller
	registeredNode, err := grpcClient.RegisterNode(ctx, nodeInfo)
	if err != nil {
		structuredLogger.WithError(err).Error("Failed to register node")
		return fmt.Errorf("failed to register node: %w", err)
	}

	structuredLogger.WithFields(map[string]interface{}{
		"registered_node_id":   registeredNode.Id,
		"registered_node_name": registeredNode.Name,
		"node_status":          registeredNode.Status,
	}).Info("Node registered successfully with controller")

	// Start GPIO gRPC server if enabled
	var agentServer *agent.Server
	if cfg.AgentServer.EnableGPIO {
		agentConfig := &agent.Config{
			Address: cfg.AgentServer.Address,
			Port:    cfg.AgentServer.Port,
		}

		agentServer, err = agent.NewServer(agentConfig, structuredLogger)
		if err != nil {
			structuredLogger.WithError(err).Error("Failed to create agent server")
			return fmt.Errorf("failed to create agent server: %w", err)
		}

		if err := agentServer.Initialize(ctx); err != nil {
			structuredLogger.WithError(err).Error("Failed to initialize agent server")
			return fmt.Errorf("failed to initialize agent server: %w", err)
		}

		if err := agentServer.Start(ctx); err != nil {
			structuredLogger.WithError(err).Error("Failed to start agent server")
			return fmt.Errorf("failed to start agent server: %w", err)
		}

		structuredLogger.WithField("address", agentServer.GetAddress()).Info("Agent GPIO gRPC server started")
	}

	// Start mDNS advertiser to announce this agent
	var advertiser *discovery.Advertiser
	if cfg.Discovery.Enabled && cfg.AgentServer.EnableGPIO {
		advertiserConfig := discovery.DefaultAdvertiserConfig()
		advertiserConfig.ServiceName = nodeInfo.Name
		advertiserConfig.ServiceType = cfg.Discovery.ServiceType
		advertiserConfig.Port = cfg.AgentServer.Port
		advertiserConfig.TXTRecords = map[string]string{
			"version":      version,
			"capabilities": "gpio,monitoring",
			"model":        nodeInfo.Model,
			"arch":         nodeInfo.Architecture,
			"node_id":      nodeInfo.ID,
		}

		// Create a logrus logger for advertiser
		logrusLogger := logrus.New()
		logrusLogger.SetLevel(logrus.InfoLevel)
		logrusLogger.SetFormatter(&logrus.JSONFormatter{})

		advertiser = discovery.NewAdvertiser(advertiserConfig, logrusLogger)
		if err := advertiser.Start(ctx); err != nil {
			structuredLogger.WithError(err).Error("Failed to start mDNS advertiser")
			// Don't fail the agent if advertiser fails - it's not critical
		} else {
			structuredLogger.WithFields(map[string]interface{}{
				"service_name": advertiserConfig.ServiceName,
				"service_type": advertiserConfig.ServiceType,
				"port":         advertiserConfig.Port,
			}).Info("mDNS advertiser started")
		}
	}

	structuredLogger.Info("Pi Agent started successfully")

	// Main agent loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Periodic tasks - the heartbeat is handled automatically by the gRPC client
			structuredLogger.Debug("Agent periodic maintenance")

			// Update node status to indicate we're still alive
			if err := grpcClient.UpdateNodeStatus(ctx, registeredNode.Id, registeredNode.Status); err != nil {
				structuredLogger.WithError(err).Warn("Failed to update node status")
			}

			// TODO: Future enhancements:
			// - Collect and report system metrics
			// - Update GPIO device states
			// - Check for pending commands from controller
			// - Monitor local system health

		case sig := <-sigChan:
			structuredLogger.WithField("signal", sig).Info("Received shutdown signal")

			// Update node status to maintenance before shutting down
			if err := grpcClient.UpdateNodeStatus(ctx, registeredNode.Id,
				pb.NodeStatus_NODE_STATUS_MAINTENANCE); err != nil {
				structuredLogger.WithError(err).Warn("Failed to update node status to maintenance")
			}

			// Stop the mDNS advertiser if it's running
			if advertiser != nil {
				if err := advertiser.Stop(); err != nil {
					structuredLogger.WithError(err).Error("Error stopping mDNS advertiser")
				}
			}

			// Stop the agent server if it's running
			if agentServer != nil {
				if err := agentServer.Stop(); err != nil {
					structuredLogger.WithError(err).Error("Error stopping agent server")
				}
			}

			// Stop the gRPC client
			if err := grpcClient.Stop(); err != nil {
				structuredLogger.WithError(err).Error("Error stopping gRPC client")
			}

			cancel()

		case <-ctx.Done():
			structuredLogger.Info("Pi Agent shutdown complete")
			return nil
		}
	}
}

// setupStructuredLogger creates a structured logger compatible with our logger interface
func setupStructuredLogger() (logger.Interface, error) {
	loggerConfig := logger.Config{
		Level:  logLevel,
		Format: logFormat,
		Output: "stdout",
	}

	structuredLogger, err := logger.New(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create structured logger: %w", err)
	}

	return structuredLogger, nil
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
