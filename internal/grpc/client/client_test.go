package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		logger      logger.Interface
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				ServerAddress: "localhost",
				ServerPort:    9090,
			},
			logger:      &mockLogger{},
			expectError: false,
		},
		{
			name:        "missing logger",
			config:      Config{ServerAddress: "localhost"},
			logger:      nil,
			expectError: true,
			errorMsg:    "logger is required",
		},
		{
			name:        "missing server address",
			config:      Config{},
			logger:      &mockLogger{},
			expectError: true,
			errorMsg:    "server address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config, tt.logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Errorf("client is nil")
				return
			}

			// Verify defaults are applied
			if client.config.ServerPort == 0 {
				t.Errorf("expected default port to be set")
			}
			if client.config.ConnectionTimeout == 0 {
				t.Errorf("expected default connection timeout to be set")
			}
		})
	}
}

func TestClientConnectionLifecycle(t *testing.T) {
	// Start mock server
	mockServer, err := NewMockServer(MockServerConfig{
		Address: "localhost:0",
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	// Extract host and port from server address
	serverAddr := mockServer.GetAddress()
	parts := strings.Split(serverAddr, ":")
	if len(parts) != 2 {
		t.Fatalf("invalid server address format: %s", serverAddr)
	}

	// Create client
	config := Config{
		ServerAddress:     parts[0],
		ServerPort:        parsePort(parts[1]),
		ConnectionTimeout: 5 * time.Second,
		RequestTimeout:    2 * time.Second,
		Insecure:          true,
	}

	client, err := NewClient(config, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()

	// Test initial disconnected state
	if client.IsConnected() {
		t.Errorf("client should not be connected initially")
	}

	// Test connection
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	if !client.IsConnected() {
		t.Errorf("client should be connected after Connect()")
	}

	// Test disconnect
	if err := client.Disconnect(); err != nil {
		t.Errorf("failed to disconnect: %v", err)
	}

	if client.IsConnected() {
		t.Errorf("client should not be connected after Disconnect()")
	}
}

func TestClientNodeRegistration(t *testing.T) {
	// Start mock server
	mockServer, err := NewMockServer(MockServerConfig{
		Address: "localhost:0",
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	// Create and connect client
	client := createTestClient(t, mockServer.GetAddress())
	defer client.Stop()

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Test node registration
	nodeInfo := &NodeInfo{
		ID:           "test-node-1",
		Name:         "test-node",
		IPAddress:    "192.168.1.100",
		MACAddress:   "aa:bb:cc:dd:ee:ff",
		Architecture: "arm64",
		Model:        "raspberry-pi",
		SerialNumber: "123456789",
		CPUCores:     4,
		Memory:       8 * 1024 * 1024 * 1024, // 8GB
	}

	node, err := client.RegisterNode(ctx, nodeInfo)
	if err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	if node == nil {
		t.Fatalf("registered node is nil")
	}

	if node.Name != nodeInfo.Name {
		t.Errorf("expected node name %q, got %q", nodeInfo.Name, node.Name)
	}

	if node.IpAddress != nodeInfo.IPAddress {
		t.Errorf("expected node IP %q, got %q", nodeInfo.IPAddress, node.IpAddress)
	}

	// Verify server received the registration
	stats := mockServer.GetStats()
	if stats.NodeRegistrations != 1 {
		t.Errorf("expected 1 node registration, got %d", stats.NodeRegistrations)
	}
}

func TestClientHealthCheck(t *testing.T) {
	mockServer, err := NewMockServer(MockServerConfig{
		Address: "localhost:0",
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	client := createTestClient(t, mockServer.GetAddress())
	defer client.Stop()

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Test health check
	resp, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected health status 'ok', got %q", resp.Status)
	}

	// Verify server received the health check
	stats := mockServer.GetStats()
	if stats.HealthChecks != 1 {
		t.Errorf("expected 1 health check, got %d", stats.HealthChecks)
	}
}

func TestClientRetryLogic(t *testing.T) {
	// Create client with non-existent server to test retry logic
	config := Config{
		ServerAddress:     "localhost",
		ServerPort:        99999, // Non-existent port
		ConnectionTimeout: 100 * time.Millisecond,
		RequestTimeout:    100 * time.Millisecond,
		MaxRetries:        2,
		InitialRetryDelay: 10 * time.Millisecond,
		MaxRetryDelay:     50 * time.Millisecond,
		RetryMultiplier:   2.0,
		Insecure:          true,
	}

	client, err := NewClient(config, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This should fail after retries
	err = client.connectWithRetry(ctx)
	if err == nil {
		t.Errorf("expected connection to fail but it succeeded")
	}

	if !strings.Contains(err.Error(), "failed to connect after") {
		t.Errorf("expected retry error message, got: %v", err)
	}
}

func TestClientGPIOOperations(t *testing.T) {
	mockServer, err := NewMockServer(MockServerConfig{
		Address: "localhost:0",
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	client := createTestClient(t, mockServer.GetAddress())
	defer client.Stop()

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Test GPIO read
	readResp, err := client.ReadGPIO(ctx, 1)
	if err != nil {
		t.Fatalf("GPIO read failed: %v", err)
	}

	if readResp.DeviceId != 1 {
		t.Errorf("expected device ID 1, got %d", readResp.DeviceId)
	}

	// Test GPIO write
	writeResp, err := client.WriteGPIO(ctx, 1, 255)
	if err != nil {
		t.Fatalf("GPIO write failed: %v", err)
	}

	if writeResp.DeviceId != 1 {
		t.Errorf("expected device ID 1, got %d", writeResp.DeviceId)
	}

	if writeResp.Value != 255 {
		t.Errorf("expected value 255, got %d", writeResp.Value)
	}

	// Verify server received the operations
	stats := mockServer.GetStats()
	if stats.GPIOReads != 1 {
		t.Errorf("expected 1 GPIO read, got %d", stats.GPIOReads)
	}
	if stats.GPIOWrites != 1 {
		t.Errorf("expected 1 GPIO write, got %d", stats.GPIOWrites)
	}
}

func TestClientHeartbeat(t *testing.T) {
	mockServer, err := NewMockServer(MockServerConfig{
		Address: "localhost:0",
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	// Create client with short heartbeat interval
	config := Config{
		ServerAddress:     extractHost(mockServer.GetAddress()),
		ServerPort:        extractPort(mockServer.GetAddress()),
		ConnectionTimeout: 5 * time.Second,
		RequestTimeout:    2 * time.Second,
		HeartbeatInterval: 100 * time.Millisecond,
		HeartbeatTimeout:  1 * time.Second,
		Insecure:          true,
	}

	client, err := NewClient(config, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Stop()

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("failed to start client: %v", err)
	}

	// Wait for multiple heartbeats
	time.Sleep(350 * time.Millisecond)

	stats := mockServer.GetStats()
	if stats.HealthChecks < 2 {
		t.Errorf("expected at least 2 heartbeats, got %d", stats.HealthChecks)
	}
}

func TestServerFailureHandling(t *testing.T) {
	mockServer, err := NewMockServer(MockServerConfig{
		Address:          "localhost:0",
		ShouldFailHealth: true,
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	client := createTestClient(t, mockServer.GetAddress())
	defer client.Stop()

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Health check should fail
	_, err = client.HealthCheck(ctx)
	if err == nil {
		t.Errorf("expected health check to fail but it succeeded")
	}

	// Heartbeat should also fail
	err = client.SendHeartbeat(ctx)
	if err == nil {
		t.Errorf("expected heartbeat to fail but it succeeded")
	}
}

func TestNodeInfoCollection(t *testing.T) {
	nodeInfo, err := CollectNodeInfo("test-node-1", "test-node")
	if err != nil {
		t.Fatalf("failed to collect node info: %v", err)
	}

	if nodeInfo.ID != "test-node-1" {
		t.Errorf("expected ID 'test-node-1', got %q", nodeInfo.ID)
	}

	if nodeInfo.Name != "test-node" {
		t.Errorf("expected name 'test-node', got %q", nodeInfo.Name)
	}

	if nodeInfo.Architecture == "" {
		t.Errorf("expected architecture to be set")
	}

	if nodeInfo.CPUCores <= 0 {
		t.Errorf("expected positive CPU cores, got %d", nodeInfo.CPUCores)
	}
}

func TestClientNodeUpdate(t *testing.T) {
	mockServer, err := NewMockServer(MockServerConfig{
		Address: "localhost:0",
	}, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create mock server: %v", err)
	}

	if err := mockServer.Start(); err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer mockServer.Stop()

	client := createTestClient(t, mockServer.GetAddress())
	defer client.Stop()

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	// Register a node first
	nodeInfo := &NodeInfo{
		Name:       "test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "aa:bb:cc:dd:ee:ff",
	}

	node, err := client.RegisterNode(ctx, nodeInfo)
	if err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	// Test status update
	err = client.UpdateNodeStatus(ctx, node.Id, pb.NodeStatus_NODE_STATUS_MAINTENANCE)
	if err != nil {
		t.Fatalf("failed to update node status: %v", err)
	}

	// Verify server received the update
	stats := mockServer.GetStats()
	if stats.NodeUpdates != 1 {
		t.Errorf("expected 1 node update, got %d", stats.NodeUpdates)
	}
}

// Helper functions

func createTestClient(t *testing.T, serverAddr string) *Client {
	config := Config{
		ServerAddress:     extractHost(serverAddr),
		ServerPort:        extractPort(serverAddr),
		ConnectionTimeout: 5 * time.Second,
		RequestTimeout:    2 * time.Second,
		Insecure:          true,
	}

	client, err := NewClient(config, &mockLogger{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return client
}

func extractHost(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) >= 1 {
		return parts[0]
	}
	return "localhost"
}

func extractPort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) >= 2 {
		return parsePort(parts[1])
	}
	return 9090
}

func parsePort(portStr string) int {
	// Simple port parsing for test purposes
	switch portStr {
	case "9090":
		return 9090
	case "9091":
		return 9091
	default:
		// Try to extract numeric part
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil && port > 0 && port < 65536 {
			return port
		}
		return 9090 // default
	}
}
