package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// PiAgentClientInterface defines the interface for Pi Agent client operations
type PiAgentClientInterface interface {
	IsConnected() bool
	ConfigureGPIOPin(ctx context.Context, device *models.GPIODevice) error
	ReadGPIOPin(ctx context.Context, pinNumber int) (int, error)
	WriteGPIOPin(ctx context.Context, pinNumber int, value int) error
	Close() error
}

// PiAgentClientManagerInterface defines the interface for managing Pi Agent clients
type PiAgentClientManagerInterface interface {
	GetClient(node *models.Node) (PiAgentClientInterface, error)
	CloseClient(nodeID uint) error
	CloseAll() error
}

// PiAgentClient provides a gRPC client interface for communicating with Pi Agents
type PiAgentClient struct {
	conn    *grpc.ClientConn
	client  pb.PiAgentServiceClient
	logger  logger.Interface
	nodeID  uint
	address string
}

// PiAgentClientManager manages connections to multiple Pi Agent nodes
type PiAgentClientManager struct {
	clients map[uint]*PiAgentClient // nodeID -> client
	mu      sync.RWMutex
	logger  logger.Interface
	timeout time.Duration
}

// NewPiAgentClientManager creates a new Pi Agent client manager
func NewPiAgentClientManager(logger logger.Interface) *PiAgentClientManager {
	return &PiAgentClientManager{
		clients: make(map[uint]*PiAgentClient),
		logger:  logger.WithField("component", "pi_agent_client_manager"),
		timeout: 30 * time.Second,
	}
}

// GetClient returns a gRPC client for the specified node, creating one if necessary
func (m *PiAgentClientManager) GetClient(node *models.Node) (PiAgentClientInterface, error) {
	m.mu.RLock()
	if client, exists := m.clients[node.ID]; exists {
		m.mu.RUnlock()
		// Check if connection is still healthy
		if client.IsConnected() {
			return client, nil
		}
		// Connection is not healthy, remove it and create a new one
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.clients, node.ID)
		m.mu.Unlock()
	} else {
		m.mu.RUnlock()
	}

	// Create new client
	client, err := m.createClient(node)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for node %d: %w", node.ID, err)
	}

	m.mu.Lock()
	m.clients[node.ID] = client
	m.mu.Unlock()

	return client, nil
}

// createClient creates a new Pi Agent client for the specified node
func (m *PiAgentClientManager) createClient(node *models.Node) (*PiAgentClient, error) {
	// Pi Agent typically runs on port 9091
	address := fmt.Sprintf("%s:9091", node.IPAddress)

	m.logger.WithFields(map[string]interface{}{
		"node_id": node.ID,
		"address": address,
	}).Debug("Creating new Pi Agent client")

	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Pi Agent at %s: %w", address, err)
	}

	client := &PiAgentClient{
		conn:    conn,
		client:  pb.NewPiAgentServiceClient(conn),
		logger:  m.logger.WithField("node_id", node.ID),
		nodeID:  node.ID,
		address: address,
	}

	m.logger.WithFields(map[string]interface{}{
		"node_id": node.ID,
		"address": address,
	}).Info("Successfully connected to Pi Agent")

	return client, nil
}

// CloseClient closes the connection to a specific node
func (m *PiAgentClientManager) CloseClient(nodeID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[nodeID]
	if !exists {
		return nil // Already closed or never existed
	}

	delete(m.clients, nodeID)
	return client.Close()
}

// CloseAll closes all client connections
func (m *PiAgentClientManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for nodeID, client := range m.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close client for node %d: %w", nodeID, err))
		}
	}

	m.clients = make(map[uint]*PiAgentClient)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}

	return nil
}

// IsConnected checks if the client connection is healthy
func (c *PiAgentClient) IsConnected() bool {
	if c.conn == nil {
		return false
	}

	// Test connection with a quick health check (with short timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.client.AgentHealth(ctx, &pb.AgentHealthRequest{})
	return err == nil
}

// ConfigureGPIOPin configures a GPIO pin on the agent
func (c *PiAgentClient) ConfigureGPIOPin(ctx context.Context, device *models.GPIODevice) error {
	req := &pb.ConfigureGPIOPinRequest{
		Pin:       int32(device.PinNumber),
		Direction: modelDirectionToProto(device.Direction),
		PullMode:  modelPullModeToProto(device.PullMode),
	}

	// Add PWM configuration if applicable
	if device.DeviceType == models.GPIODeviceTypePWM {
		req.PwmFrequency = int32(device.Config.Frequency)
		req.PwmDutyCycle = int32(device.Config.DutyCycle)
	}

	resp, err := c.client.ConfigureGPIOPin(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("pin configuration failed: %s", resp.Message)
	}

	c.logger.WithFields(map[string]interface{}{
		"pin":       device.PinNumber,
		"direction": device.Direction,
		"pull_mode": device.PullMode,
	}).Debug("GPIO pin configured successfully")

	return nil
}

// ReadGPIOPin reads the current value of a GPIO pin
func (c *PiAgentClient) ReadGPIOPin(ctx context.Context, pinNumber int) (int, error) {
	req := &pb.ReadGPIOPinRequest{
		Pin: int32(pinNumber),
	}

	resp, err := c.client.ReadGPIOPin(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("gRPC call failed: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"pin":   pinNumber,
		"value": resp.Value,
	}).Debug("GPIO pin read successfully")

	return int(resp.Value), nil
}

// WriteGPIOPin writes a value to a GPIO pin
func (c *PiAgentClient) WriteGPIOPin(ctx context.Context, pinNumber int, value int) error {
	req := &pb.WriteGPIOPinRequest{
		Pin:   int32(pinNumber),
		Value: int32(value),
	}

	resp, err := c.client.WriteGPIOPin(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	c.logger.WithFields(map[string]interface{}{
		"pin":          pinNumber,
		"value":        value,
		"actual_value": resp.Value,
	}).Debug("GPIO pin written successfully")

	return nil
}

// Close closes the client connection
func (c *PiAgentClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

// Helper functions to convert between model types and protobuf types

func modelDirectionToProto(direction models.GPIODirection) pb.AgentGPIODirection {
	switch direction {
	case models.GPIODirectionInput:
		return pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_INPUT
	case models.GPIODirectionOutput:
		return pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT
	default:
		return pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_UNSPECIFIED
	}
}

func modelPullModeToProto(pullMode models.GPIOPullMode) pb.AgentGPIOPullMode {
	switch pullMode {
	case models.GPIOPullNone:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE
	case models.GPIOPullUp:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UP
	case models.GPIOPullDown:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_DOWN
	default:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UNSPECIFIED
	}
}
