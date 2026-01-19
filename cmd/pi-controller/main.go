package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dsyorkd/pi-controller/internal/agent"
	"github.com/dsyorkd/pi-controller/internal/api"
	"github.com/dsyorkd/pi-controller/internal/clustering"
	"github.com/dsyorkd/pi-controller/internal/clustering/health"
	"github.com/dsyorkd/pi-controller/internal/clustering/replication"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/controller"
	"github.com/dsyorkd/pi-controller/internal/errors"
	grpcclient "github.com/dsyorkd/pi-controller/internal/grpc/client"
	grpcserver "github.com/dsyorkd/pi-controller/internal/grpc/server"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/migrations"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	internaltls "github.com/dsyorkd/pi-controller/internal/tls"
	"github.com/dsyorkd/pi-controller/internal/websocket"
	"github.com/dsyorkd/pi-controller/pkg/discovery"
	pb "github.com/dsyorkd/pi-controller/proto"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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

	// Initialize Sentry SDK
	if err := initializeSentry(cfg, log); err != nil {
		log.WithError(err).Warn("Failed to initialize Sentry - continuing without error tracking")
	}

	// Initialize database
	db, err := storage.New(&cfg.Database, log)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize database")
	}
	defer db.Close()

	log.Info("Database initialized successfully")

	// Register local node in database
	if err := registerLocalNode(db, cfg, log); err != nil {
		log.WithError(err).Warn("Failed to register local node - continuing without self-registration")
	}

	// Initialize clustering if enabled
	var cluster *clustering.RaftCluster
	var healthChecker *health.HealthChecker
	var replicator *replication.Replicator

	if cfg.Cluster.Enabled {
		log.Info("Initializing controller clustering...")

		// Parse duration configs
		heartbeatTimeout, err := time.ParseDuration(cfg.Cluster.HeartbeatTimeout)
		if err != nil {
			return errors.Wrapf(err, "invalid heartbeat timeout")
		}
		electionTimeout, err := time.ParseDuration(cfg.Cluster.ElectionTimeout)
		if err != nil {
			return errors.Wrapf(err, "invalid election timeout")
		}
		snapshotInterval, err := time.ParseDuration(cfg.Cluster.SnapshotInterval)
		if err != nil {
			return errors.Wrapf(err, "invalid snapshot interval")
		}

		// Create cluster configuration
		clusterConfig := &clustering.ClusterConfig{
			ControllerID:      cfg.Cluster.ControllerID,
			BindAddr:          cfg.Cluster.BindAddr,
			DataDir:           cfg.Cluster.DataDir,
			Bootstrap:         cfg.Cluster.Bootstrap,
			InitialPeers:      cfg.Cluster.InitialPeers,
			HeartbeatTimeout:  heartbeatTimeout,
			ElectionTimeout:   electionTimeout,
			SnapshotInterval:  snapshotInterval,
			SnapshotThreshold: cfg.Cluster.SnapshotThreshold,
			MaxAppendEntries:  cfg.Cluster.MaxAppendEntries,
		}

		// Create Raft cluster
		cluster, err = clustering.NewRaftCluster(clusterConfig, log)
		if err != nil {
			return errors.Wrapf(err, "failed to create Raft cluster")
		}

		log.WithFields(map[string]interface{}{
			"controller_id": clusterConfig.ControllerID,
			"bind_addr":     clusterConfig.BindAddr,
			"bootstrap":     clusterConfig.Bootstrap,
		}).Info("Raft cluster initialized")

		// Wait for leader election
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := cluster.WaitForLeader(ctx); err != nil {
			log.WithError(err).Warn("No leader elected within timeout - continuing anyway")
		} else {
			log.WithField("leader", cluster.GetLeader()).Info("Cluster leader elected")
		}

		// Initialize database replication
		replicator = replication.NewReplicator(db, cluster, log)
		if err := replicator.Start(context.Background()); err != nil {
			return errors.Wrapf(err, "failed to start replicator")
		}

		log.Info("Database replication initialized")

		// Initialize health checker
		healthChecker = health.NewHealthChecker(log, 5*time.Second)

		// Register health checks
		healthChecker.RegisterCheck("database", health.DatabaseHealthCheck(db))
		healthChecker.RegisterCheck("raft", health.RaftHealthCheck(cluster))

		if err := healthChecker.Start(context.Background()); err != nil {
			return errors.Wrapf(err, "failed to start health checker")
		}

		log.Info("Health checking initialized")

		// Set up leadership callbacks
		cluster.OnBecomeLeader(func() {
			log.Info("🎖️  This controller became the cluster leader")
		})

		cluster.OnLoseLeadership(func() {
			log.Warn("Lost cluster leadership - switching to follower mode")
		})
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start servers
	var wg sync.WaitGroup
	serverErrors := make(chan error, 4)

	// Initialize CA service
	sshExecutor := services.NewSSHExecutor(&cfg.CA.SSH, log)
	caService, err := services.NewCAService(&cfg.CA, db, log, sshExecutor)
	if err != nil {
		log.WithError(err).Warn("Failed to initialize CA service - running without CA functionality")
	} else {
		log.Info("CA service initialized successfully")

		// Initialize CA if it hasn't been initialized yet (this is idempotent)
		if err := caService.InitializeCA(context.Background()); err != nil {
			log.WithError(err).Warn("CA initialization failed - manual initialization may be required")
		}
	}

	// Setup client TLS config for mTLS (Controller -> Remote Agent)
	var clientTLSConfig *tls.Config
	if cfg.GRPCClient.TLSCert != "" && cfg.GRPCClient.TLSKey != "" && cfg.GRPCClient.TLSCAFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.GRPCClient.TLSCert, cfg.GRPCClient.TLSKey)
		if err != nil {
			log.WithError(err).Warn("Failed to load client TLS key pair")
		} else {
			caCert, err := os.ReadFile(cfg.GRPCClient.TLSCAFile)
			if err != nil {
				log.WithError(err).Warn("Failed to read client CA certificate")
			} else {
				caCertPool := x509.NewCertPool()
				if caCertPool.AppendCertsFromPEM(caCert) {
					clientTLSConfig = &tls.Config{
						Certificates: []tls.Certificate{cert},
						RootCAs:      caCertPool,
						MinVersion:   tls.VersionTLS12,
					}
					log.Info("Client TLS configuration loaded")
				}
			}
		}
	}

	// Start Controller Manager (Kubernetes Operator) - only if enabled
	var controllerManager *controller.ControllerManager
	if cfg.Kubernetes.Enabled {
		cmConfig := &controller.ControllerManagerConfig{
			MetricsAddr:      ":8082", // Use different port than API
			HealthAddr:       ":8083",
			LeaderElection:   cfg.Cluster.Enabled,
			LeaderElectionID: "pi-controller-leader-election",
			LeaderElectionNS: "kube-system",
			LogLevel:         logLevel,
		}

		var err error
		controllerManager, err = controller.NewControllerManager(cmConfig, log, db, clientTLSConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to create controller manager")
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("Starting Controller Manager")
			if err := controllerManager.Start(context.Background()); err != nil {
				log.WithError(err).Error("Controller Manager failed")
				serverErrors <- err
			}
		}()
	} else {
		log.Info("Kubernetes controller manager disabled in configuration")
	}

	// Start Agent Server (Hardware Control)
	// Create a client connection to the local Controller gRPC server for metrics streaming
	var controllerClient pb.PiControllerServiceClient
	controllerAddr := fmt.Sprintf("localhost:%d", cfg.GRPC.Port)
	var dialOpts []grpc.DialOption

	if cfg.GRPC.IsTLSEnabled() {
		if clientTLSConfig != nil {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)))
		} else {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(controllerAddr, dialOpts...)
	if err == nil {
		controllerClient = pb.NewPiControllerServiceClient(conn)
	} else {
		log.WithError(err).Warn("Failed to create local controller client")
	}

	agentConfig := &agent.Config{
		Address:         cfg.AgentServer.Address,
		Port:            cfg.AgentServer.Port,
		MetricsInterval: 5,
		TLSCertFile:     cfg.AgentServer.TLSCertFile,
		TLSKeyFile:      cfg.AgentServer.TLSKeyFile,
		TLSCAFile:       cfg.AgentServer.TLSCAFile, // Use configured CA file
	}

	// We need a node ID. Use controller ID or hostname.
	nodeID := cfg.Cluster.ControllerID
	if nodeID == "" {
		nodeID, _ = os.Hostname()
	}

	agentServer, err := agent.NewServer(agentConfig, log, controllerClient, nodeID)
	if err != nil {
		return errors.Wrapf(err, "failed to create agent server")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("Starting Agent Server")
		if err := agentServer.Start(context.Background()); err != nil {
			serverErrors <- err
		}
	}()

	// Setup TLS/HTTPS if enabled or in production
	if cfg.API.IsTLSEnabled() || cfg.App.Environment == "production" {
		tlsConfig := internaltls.Config{
			CertFile: cfg.API.TLSCertFile,
			KeyFile:  cfg.API.TLSKeyFile,
			// Auto-generate cert for development if not provided
			AutoCert: cfg.App.Environment == "development" && !cfg.API.IsTLSEnabled(),
			CertDir:  cfg.App.DataDir + "/tls",
			// Default hostnames for development
			Hostnames: []string{"localhost", "127.0.0.1", "::1"},
		}

		_, err := internaltls.Setup(tlsConfig, log)
		if err != nil {
			return errors.Wrapf(err, "failed to setup TLS")
		}

		log.WithFields(map[string]interface{}{
			"cert_file": cfg.API.TLSCertFile,
			"key_file":  cfg.API.TLSKeyFile,
			"auto_cert": tlsConfig.AutoCert,
		}).Info("TLS/HTTPS configured successfully")
	} else if cfg.App.Environment == "production" {
		log.Warn("⚠️  Production environment detected but TLS is not enabled!")
		log.Warn("⚠️  It is STRONGLY recommended to enable TLS in production")
	}

	// Start REST API server
	apiServer := api.New(&cfg.API, &cfg.Discovery, log, db, caService)

	// Set clustering components on API server if clustering is enabled
	if cfg.Cluster.Enabled && cluster != nil {
		apiServer.SetClusteringComponents(cluster, healthChecker)
		log.Info("Clustering API endpoints enabled")
	}

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

	// Start mDNS Advertiser to broadcast this node's presence
	var advertiser *discovery.Advertiser
	if cfg.Discovery.Enabled {
		hostname, _ := os.Hostname()
		advertiserConfig := &discovery.AdvertiserConfig{
			ServiceName: hostname,
			ServiceType: cfg.Discovery.ServiceType,
			Domain:      "local.",
			Port:        cfg.API.Port,
			HostName:    hostname + ".local.",
			TXTRecords: map[string]string{
				"version":    version,
				"node_id":    cfg.Cluster.ControllerID,
				"arch":       "arm64",
				"agent_port": strconv.Itoa(cfg.GRPC.Port),
			},
			TTL: 3600 * time.Second,
		}

		advertiser = discovery.NewAdvertiser(advertiserConfig, log)
		if err := advertiser.Start(context.Background()); err != nil {
			log.WithError(err).Warn("Failed to start mDNS advertiser - nodes may not be discoverable")
		} else {
			log.WithFields(map[string]interface{}{
				"service_name": advertiserConfig.ServiceName,
				"service_type": advertiserConfig.ServiceType,
				"port":         advertiserConfig.Port,
			}).Info("Started mDNS advertiser")
		}
	}

	// Start Discovery Service to find other nodes
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

		discoveryService, err = discovery.NewService(discoveryConfig, log)
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
			if err := handleNodeDiscoveryEvent(event, db, caService, sshExecutor, log); err != nil {
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

	go func() {
		if controllerManager != nil {
			if err := controllerManager.Stop(); err != nil {
				log.WithError(err).Error("Error stopping Controller Manager")
			}
		}
	}()

	go func() {
		if err := agentServer.Stop(); err != nil {
			log.WithError(err).Error("Error stopping Agent Server")
		}
	}()

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

	// Stop mDNS advertiser
	if advertiser != nil {
		go func() {
			if err := advertiser.Stop(); err != nil {
				log.WithError(err).Error("Error stopping mDNS advertiser")
			}
		}()
	}

	// Stop clustering components
	if cfg.Cluster.Enabled {
		go func() {
			if healthChecker != nil {
				if err := healthChecker.Stop(); err != nil {
					log.WithError(err).Error("Error stopping health checker")
				}
			}

			if replicator != nil {
				if err := replicator.Stop(); err != nil {
					log.WithError(err).Error("Error stopping replicator")
				}
			}

			if cluster != nil {
				if err := cluster.Shutdown(); err != nil {
					log.WithError(err).Error("Error shutting down cluster")
				}
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

	// Flush any pending Sentry events
	sentry.Flush(2 * time.Second)

	log.Info("Pi Controller shutdown complete")
	return nil
}

// handleNodeDiscoveryEvent processes node discovery events and automatically registers new nodes
func handleNodeDiscoveryEvent(event discovery.NodeEvent, db *storage.Database, caService services.CAService, sshExecutor services.SSHExecutor, log *logger.Logger) error {
	switch event.Type {
	case discovery.NodeDiscovered:
		return handleNodeRegistration(event, db, caService, sshExecutor, log)
	case discovery.NodeLost:
		return handleNodeLost(event, db, log)
	case discovery.NodeUpdated:
		return handleNodeUpdate(event, db, caService, sshExecutor, log)
	default:
		log.WithField("event_type", event.Type).Debug("Ignoring unhandled discovery event type")
		return nil
	}
}

// handleNodeRegistration processes new node discovery and registration
func handleNodeRegistration(event discovery.NodeEvent, db *storage.Database, caService services.CAService, sshExecutor services.SSHExecutor, log *logger.Logger) error {
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
	agentPort := node.TXTRecords["agent_port"]

	// Use discovery node_id if available from TXT records, otherwise use discovery ID
	nodeName := node.Name
	if nodeIdFromTXT != "" {
		nodeName = nodeIdFromTXT
	}

	// All nodes are now Generic type in the single-binary architecture
	discoveredNodeType := models.NodeTypeGeneric
	controllerVersion := version
	discoveredAgentPort := 0

	// Extract agent port if provided in TXT records
	if port := agentPort; port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			discoveredAgentPort = p
		}
	}

	// Default role to worker - could be enhanced to detect master nodes
	role := models.NodeRoleWorker

	// Create new node registration request
	createReq := services.CreateNodeRequest{
		Name:              nodeName,
		IPAddress:         node.IPAddress,
		MACAddress:        "", // Not available from mDNS discovery
		Role:              role,
		DiscoveryMethod:   models.DiscoveryMethodMDNS,
		NodeType:          discoveredNodeType,
		ControllerVersion: controllerVersion,
		AgentPort:         discoveredAgentPort,
		Architecture:      architecture,
		Model:             model,
		CPUCores:          1,                  // Default, will be updated when node connects
		Memory:            1024 * 1024 * 1024, // Default 1GB, will be updated when node connects
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

	// Generate and distribute client certificates
	log.WithField("node_name", newNode.Name).Info("Generating client certificate for new node")

	// Construct common name and SANs for the certificate
	commonName := newNode.Name
	sans := []string{newNode.IPAddress}
	if newNode.NodeName != "" {
		sans = append(sans, newNode.NodeName)
	}

	certReq := &services.IssueCertificateRequest{
		CommonName: commonName,
		Type:       models.CertificateTypeClient,
		SANs:       sans,
		NodeID:     &newNode.ID,
		AutoRenew:  true,
	}

	clientCertRecord, clientKeyPEM, err := caService.IssueCertificate(context.Background(), certReq)
	if err != nil {
		return errors.Wrapf(err, "failed to issue client certificate for node %s", newNode.Name)
	}

	// Get CA certificate
	caCert, err := caService.GetCACertificate(context.Background())
	if err != nil {
		return errors.Wrapf(err, "failed to get CA certificate")
	}
	caCertPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw}))

	// Securely transmit certificates to the agent via SSH
	remoteTLSDir := "/etc/pi-controller/tls" // TODO: Make configurable

	// Create TLS directory on the remote agent
	createDirCmd := fmt.Sprintf("sudo mkdir -p %s && sudo chmod 700 %s", remoteTLSDir, remoteTLSDir)
	if _, err := sshExecutor.Execute(context.Background(), newNode.IPAddress, createDirCmd); err != nil {
		return errors.Wrapf(err, "failed to create TLS directory on node %s", newNode.IPAddress)
	}

	// Copy client certificate
	certPath := filepath.Join(remoteTLSDir, "client.crt")
	if err := sshExecutor.CopyContent(context.Background(), newNode.IPAddress, clientCertRecord.CertificatePEM, certPath); err != nil {
		return errors.Wrapf(err, "failed to copy client certificate to node %s", newNode.IPAddress)
	}

	// Copy client private key
	keyPath := filepath.Join(remoteTLSDir, "client.key")
	if err := sshExecutor.CopyContent(context.Background(), newNode.IPAddress, clientKeyPEM, keyPath); err != nil {
		return errors.Wrapf(err, "failed to copy client private key to node %s", newNode.IPAddress)
	}

	// Copy CA certificate
	caPath := filepath.Join(remoteTLSDir, "ca.crt")
	if err := sshExecutor.CopyContent(context.Background(), newNode.IPAddress, caCertPEM, caPath); err != nil {
		return errors.Wrapf(err, "failed to copy CA certificate to node %s", newNode.IPAddress)
	}

	log.WithField("node_name", newNode.Name).Info("Client certificates distributed successfully")

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
		"node_id":    existingNode.ID,
		"node_name":  existingNode.Name,
		"ip_address": existingNode.IPAddress,
		"new_status": models.NodeStatusUnknown,
	}).Info("Updated node status due to lost event")

	return nil
}

// handleNodeUpdate processes node update events and refreshes node information
func handleNodeUpdate(event discovery.NodeEvent, db *storage.Database, caService services.CAService, sshExecutor services.SSHExecutor, log *logger.Logger) error {
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
			return handleNodeRegistration(event, db, caService, sshExecutor, log)
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
			"node_id":    existingNode.ID,
			"ip_address": existingNode.IPAddress,
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

// initializeSentry initializes the Sentry SDK with the provided configuration
func initializeSentry(cfg *config.Config, log *logger.Logger) error {
	// Skip initialization if DSN is not provided
	if cfg.Sentry.DSN == "" {
		log.Debug("Sentry DSN not configured, skipping initialization")
		return nil
	}

	// Set release to application version if not configured
	release := cfg.Sentry.Release
	if release == "" {
		release = version
	}

	// Initialize Sentry SDK
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.Sentry.DSN,
		Environment:      cfg.Sentry.Environment,
		Release:          release,
		Debug:            cfg.Sentry.Debug,
		TracesSampleRate: cfg.Sentry.TracesSampleRate,
		SampleRate:       cfg.Sentry.SampleRate,
		EnableTracing:    cfg.Sentry.EnableTracing,
		SendDefaultPII:   cfg.Sentry.SendDefaultPII,
		MaxBreadcrumbs:   cfg.Sentry.MaxBreadcrumbs,
		AttachStacktrace: cfg.Sentry.AttachStacktrace,
		ServerName:       "", // Don't send server name for privacy
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Filter out sensitive information from error events
			if event.Request != nil {
				// Remove authorization headers
				if event.Request.Headers != nil {
					delete(event.Request.Headers, "authorization")
					delete(event.Request.Headers, "Authorization")
				}
			}
			return event
		},
	})

	if err != nil {
		return errors.Wrapf(err, "failed to initialize Sentry SDK")
	}

	// Set up a flush function to ensure events are sent on shutdown
	defer func() {
		if r := recover(); r != nil {
			sentry.Flush(2 * time.Second)
			panic(r)
		}
	}()

	log.WithFields(map[string]interface{}{
		"environment": cfg.Sentry.Environment,
		"release":     release,
		"debug":       cfg.Sentry.Debug,
	}).Info("Sentry SDK initialized successfully")

	return nil
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

// registerLocalNode registers the local pi-controller instance as a node in the database
func registerLocalNode(db *storage.Database, _ *config.Config, log *logger.Logger) error {
	log.Info("Attempting to register local node...")

	// Gather local system information using the client's collection function
	nodeInfo, err := grpcclient.CollectNodeInfo("", "")
	if err != nil {
		return errors.Wrapf(err, "failed to collect local system info")
	}

	// Use hostname if available, otherwise generate a name
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "pi-controller"
	}
	if nodeInfo.Name == "" {
		nodeInfo.Name = hostname
	}

	log.WithFields(map[string]interface{}{
		"name":         nodeInfo.Name,
		"ip_address":   nodeInfo.IPAddress,
		"mac_address":  nodeInfo.MACAddress,
		"architecture": nodeInfo.Architecture,
	}).Info("Collected local system information")

	// Initialize node service
	nodeService := services.NewNodeService(db, log)

	// Check if node already exists by MAC address (unique identifier)
	existingNode, err := nodeService.GetByMAC(context.Background(), nodeInfo.MACAddress)
	if err == nil && existingNode != nil {
		// Node exists, update it
		log.WithField("node_id", existingNode.ID).Info("Local node already registered, updating...")

		updateReq := services.UpdateNodeRequest{
			Name:         &nodeInfo.Name,
			IPAddress:    &nodeInfo.IPAddress,
			Architecture: &nodeInfo.Architecture,
			Model:        &nodeInfo.Model,
			Status:       func() *models.NodeStatus { s := models.NodeStatusReady; return &s }(),
		}

		if nodeInfo.CPUCores > 0 {
			cores := int(nodeInfo.CPUCores)
			updateReq.CPUCores = &cores
		}

		updatedNode, err := nodeService.Update(existingNode.ID, updateReq)
		if err != nil {
			return errors.Wrapf(err, "failed to update existing local node")
		}

		log.WithField("node_id", updatedNode.ID).Info("Local node updated successfully")
		return nil
	}

	// Node doesn't exist, create new one
	log.Info("Creating new local node record...")

	createReq := services.CreateNodeRequest{
		Name:            nodeInfo.Name,
		IPAddress:       nodeInfo.IPAddress,
		MACAddress:      nodeInfo.MACAddress,
		Role:            models.NodeRoleWorker, // Default to worker role
		Architecture:    nodeInfo.Architecture,
		Model:           nodeInfo.Model,
		CPUCores:        int(nodeInfo.CPUCores),
		DiscoveryMethod: models.DiscoveryMethodManual, // Self-registration is considered manual
		NodeType:        models.NodeTypeController,    // This is a controller node
	}

	// Set hostname if available
	if hostname != "" {
		createReq.Hostname = hostname
	}

	createdNode, err := nodeService.Create(createReq)
	if err != nil {
		// Check if error is due to duplicate (race condition)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique constraint") {
			log.Info("Node was created by another process, attempting to retrieve...")
			existingNode, err := nodeService.GetByMAC(context.Background(), nodeInfo.MACAddress)
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve node after duplicate error")
			}
			log.WithField("node_id", existingNode.ID).Info("Retrieved existing local node")
			return nil
		}
		return errors.Wrapf(err, "failed to create local node")
	}

	log.WithField("node_id", createdNode.ID).Info("✅ Local node registered successfully")
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
