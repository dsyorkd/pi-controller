package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// Client provides a gRPC client interface for the Pi Agent to communicate with the Pi Controller
type Client struct {
	// Configuration
	config Config
	logger logger.Interface

	// Connection management
	conn   *grpc.ClientConn
	client pb.PiControllerServiceClient

	// State management
	mu              sync.RWMutex
	connected       bool
	reconnecting    bool
	lastConnectTime time.Time
	heartbeatStop   chan struct{}

	// Node information
	nodeInfo *NodeInfo

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
}

// Config contains gRPC client configuration
type Config struct {
	// Server connection
	ServerAddress string `yaml:"server_address"`
	ServerPort    int    `yaml:"server_port"`

	// Connection settings
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	RequestTimeout    time.Duration `yaml:"request_timeout"`
	MaxMessageSize    int           `yaml:"max_message_size"`

	// Retry configuration
	MaxRetries        int           `yaml:"max_retries"`
	InitialRetryDelay time.Duration `yaml:"initial_retry_delay"`
	MaxRetryDelay     time.Duration `yaml:"max_retry_delay"`
	RetryMultiplier   float64       `yaml:"retry_multiplier"`

	// Heartbeat settings
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	HeartbeatTimeout  time.Duration `yaml:"heartbeat_timeout"`

	// Keep-alive settings
	KeepAliveTime    time.Duration `yaml:"keepalive_time"`
	KeepAliveTimeout time.Duration `yaml:"keepalive_timeout"`

	// Security
	Insecure bool   `yaml:"insecure"`
	TLSCert  string `yaml:"tls_cert"`
	TLSKey   string `yaml:"tls_key"`
}

// NodeInfo contains information about the current node
type NodeInfo struct {
	ID            string
	Name          string
	IPAddress     string
	MACAddress    string
	Architecture  string
	Model         string
	SerialNumber  string
	CPUCores      int32
	Memory        int64
	OSVersion     string
	KernelVersion string
}

// ClientInterface defines the interface for the gRPC client
type ClientInterface interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	WaitForConnection(ctx context.Context) error

	// Node operations
	RegisterNode(ctx context.Context, nodeInfo *NodeInfo) (*pb.Node, error)
	UpdateNodeStatus(ctx context.Context, nodeID uint32, status pb.NodeStatus) error
	SendHeartbeat(ctx context.Context) error

	// Health check
	HealthCheck(ctx context.Context) (*pb.HealthResponse, error)

	// GPIO operations (for future use)
	ReadGPIO(ctx context.Context, deviceID uint32) (*pb.ReadGPIOResponse, error)
	WriteGPIO(ctx context.Context, deviceID uint32, value int32) (*pb.WriteGPIOResponse, error)

	// Start/Stop lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// NewClient creates a new gRPC client instance
func NewClient(config Config, logger logger.Interface) (*Client, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if config.ServerAddress == "" {
		return nil, fmt.Errorf("server address is required")
	}

	// Apply defaults
	config = applyDefaults(config)

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:        config,
		logger:        logger,
		heartbeatStop: make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}

	return client, nil
}

// applyDefaults sets default values for unset configuration fields
func applyDefaults(config Config) Config {
	if config.ServerPort == 0 {
		config.ServerPort = 9090
	}
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 10 * time.Second
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.MaxMessageSize == 0 {
		config.MaxMessageSize = 4 * 1024 * 1024 // 4MB
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 5
	}
	if config.InitialRetryDelay == 0 {
		config.InitialRetryDelay = 1 * time.Second
	}
	if config.MaxRetryDelay == 0 {
		config.MaxRetryDelay = 60 * time.Second
	}
	if config.RetryMultiplier == 0 {
		config.RetryMultiplier = 2.0
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.HeartbeatTimeout == 0 {
		config.HeartbeatTimeout = 5 * time.Second
	}
	if config.KeepAliveTime == 0 {
		config.KeepAliveTime = 30 * time.Second
	}
	if config.KeepAliveTimeout == 0 {
		config.KeepAliveTimeout = 5 * time.Second
	}

	return config
}

// Connect establishes a connection to the gRPC server
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	c.logger.Info("Attempting to connect to gRPC server",
		"address", c.getServerAddress())

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.ConnectionTimeout)
	defer cancel()

	// Configure dial options
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(c.config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(c.config.MaxMessageSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.config.KeepAliveTime,
			Timeout:             c.config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
	}

	// Add security options
	if c.config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	// TODO: Add TLS credentials when TLS is configured

	// Establish connection
	conn, err := grpc.DialContext(ctx, c.getServerAddress(), opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.conn = conn
	c.client = pb.NewPiControllerServiceClient(conn)
	c.connected = true
	c.lastConnectTime = time.Now()

	c.logger.Info("Successfully connected to gRPC server")

	return nil
}

// Disconnect closes the gRPC connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.logger.Info("Disconnecting from gRPC server")

	// Stop heartbeat
	select {
	case c.heartbeatStop <- struct{}{}:
	default:
	}

	c.connected = false

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil

		if err != nil {
			c.logger.WithError(err).Error("Error closing gRPC connection")
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	c.logger.Info("Disconnected from gRPC server")
	return nil
}

// IsConnected returns true if the client is connected to the server
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return false
	}

	// Check connection state
	state := c.conn.GetState()
	return state == connectivity.Ready
}

// WaitForConnection waits for the connection to be established
func (c *Client) WaitForConnection(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if c.IsConnected() {
				return nil
			}
		}
	}
}

// getServerAddress returns the formatted server address
func (c *Client) getServerAddress() string {
	return fmt.Sprintf("%s:%d", c.config.ServerAddress, c.config.ServerPort)
}

// ensureConnected checks the connection and attempts to reconnect if necessary
func (c *Client) ensureConnected(ctx context.Context) error {
	if c.IsConnected() {
		return nil
	}

	return c.connectWithRetry(ctx)
}

// connectWithRetry attempts to connect with exponential backoff retry logic
func (c *Client) connectWithRetry(ctx context.Context) error {
	c.mu.Lock()
	if c.reconnecting {
		c.mu.Unlock()
		return c.WaitForConnection(ctx)
	}
	c.reconnecting = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
	}()

	var lastErr error
	delay := c.config.InitialRetryDelay

	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		// Try to connect
		if err := c.Connect(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		c.logger.WithError(lastErr).WithField("attempt", attempt+1).
			Warn("Connection attempt failed, retrying")

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * c.config.RetryMultiplier)
		if delay > c.config.MaxRetryDelay {
			delay = c.config.MaxRetryDelay
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w",
		c.config.MaxRetries, lastErr)
}

// SetNodeInfo sets the node information for registration and heartbeat
func (c *Client) SetNodeInfo(nodeInfo *NodeInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodeInfo = nodeInfo
}

// GetNodeInfo returns the current node information
func (c *Client) GetNodeInfo() *NodeInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.nodeInfo == nil {
		return nil
	}
	// Return a copy to avoid race conditions
	info := *c.nodeInfo
	return &info
}

// createCallContext creates a context with timeout for gRPC calls
func (c *Client) createCallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.config.RequestTimeout)
}

// Start begins the client lifecycle, including connection management and heartbeat
func (c *Client) Start(ctx context.Context) error {
	c.logger.Info("Starting gRPC client")

	// Initial connection
	if err := c.connectWithRetry(ctx); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start heartbeat goroutine
	go c.heartbeatLoop(ctx)

	c.logger.Info("gRPC client started successfully")
	return nil
}

// Stop stops the client and cleans up resources
func (c *Client) Stop() error {
	c.logger.Info("Stopping gRPC client")

	// Cancel context to stop all goroutines
	c.cancel()

	// Disconnect
	if err := c.Disconnect(); err != nil {
		c.logger.WithError(err).Error("Error during disconnect")
		return err
	}

	c.logger.Info("gRPC client stopped")
	return nil
}
