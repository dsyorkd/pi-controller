package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	proto "github.com/dsyorkd/pi-controller/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// PiAgentClientInterface defines the interface for Pi Agent client operations
type PiAgentClientInterface interface {
	IsConnected() bool
	ConfigureGPIOPin(ctx context.Context, device *models.GPIODevice) error
	ReadGPIOPin(ctx context.Context, pinNumber int) (int, error)
	WriteGPIOPin(ctx context.Context, pinNumber int, value int) error
	SPIExchange(ctx context.Context, channel int, data []byte) ([]byte, error)
	SPIWrite(ctx context.Context, channel int, data []byte) error
	SPIRead(ctx context.Context, channel int, length int) ([]byte, error)
	I2CWrite(ctx context.Context, bus int, address int, data []byte) error
	I2CRead(ctx context.Context, bus int, address int, length int) ([]byte, error)
	I2CWriteRegister(ctx context.Context, bus int, address int, register int, data []byte) error
	I2CReadRegister(ctx context.Context, bus int, address int, register int, length int) ([]byte, error)
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
	client  proto.PiAgentServiceClient
	logger  logger.Interface
	nodeID  uint
	address string
}

// PiAgentClientManager manages connections to multiple Pi Agent nodes
type PiAgentClientManager struct {
	clients   map[uint]*PiAgentClient // nodeID -> client
	mu        sync.RWMutex
	logger    logger.Interface
	timeout   time.Duration
	tlsConfig *tls.Config // TLS configuration for mTLS
}

// NewPiAgentClientManager creates a new Pi Agent client manager
func NewPiAgentClientManager(logger logger.Interface, tlsConfig *tls.Config) *PiAgentClientManager {
	return &PiAgentClientManager{
		clients:   make(map[uint]*PiAgentClient),
		logger:    logger.WithField("component", "pi_agent_client_manager"),
		timeout:   30 * time.Second,
		tlsConfig: tlsConfig,
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

	// Create connection (NewClient creates lazy connection)
	var dialOpts []grpc.DialOption
	if m.tlsConfig != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(m.tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Pi Agent at %s: %w", address, err)
	}

	client := &PiAgentClient{
		conn:    conn,
		client:  proto.NewPiAgentServiceClient(conn),
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

	_, err := c.client.AgentHealth(ctx, &proto.AgentHealthRequest{})
	return err == nil
}

// ConfigureGPIOPin configures a GPIO pin on the agent
func (c *PiAgentClient) ConfigureGPIOPin(ctx context.Context, device *models.GPIODevice) error {
	req := &proto.ConfigureGPIOPinRequest{
		Pin:       mustSafeIntToInt32(device.PinNumber), // #nosec G115 - safe conversion with overflow check
		Direction: modelDirectionToProto(device.Direction),
		PullMode:  modelPullModeToProto(device.PullMode),
	}

	// Add PWM configuration if applicable
	if device.DeviceType == models.GPIODeviceTypePWM {
		req.PwmFrequency = mustSafeIntToInt32(device.Config.Frequency) // #nosec G115 - safe conversion with overflow check
		req.PwmDutyCycle = mustSafeIntToInt32(device.Config.DutyCycle) // #nosec G115 - safe conversion with overflow check
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
	req := &proto.ReadGPIOPinRequest{
		Pin: mustSafeIntToInt32(pinNumber), // #nosec G115 - safe conversion with overflow check
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
	req := &proto.WriteGPIOPinRequest{
		Pin:   mustSafeIntToInt32(pinNumber), // #nosec G115 - safe conversion with overflow check
		Value: mustSafeIntToInt32(value),     // #nosec G115 - safe conversion with overflow check
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

// SPIExchange performs an SPI exchange on the agent
func (c *PiAgentClient) SPIExchange(ctx context.Context, channel int, data []byte) ([]byte, error) {
	req := &proto.SPIExchangeRequest{
		Channel: mustSafeIntToInt32(channel),
		Data:    data,
	}
	resp, err := c.client.SPIExchange(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}
	return resp.Data, nil
}

// SPIWrite performs an SPI write on the agent
func (c *PiAgentClient) SPIWrite(ctx context.Context, channel int, data []byte) error {
	req := &proto.SPIWriteRequest{
		Channel: mustSafeIntToInt32(channel),
		Data:    data,
	}
	resp, err := c.client.SPIWrite(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("SPI write failed: %s", resp.Message)
	}
	return nil
}

// SPIRead performs an SPI read from the agent
func (c *PiAgentClient) SPIRead(ctx context.Context, channel int, length int) ([]byte, error) {
	req := &proto.SPIReadRequest{
		Channel: mustSafeIntToInt32(channel),
		Length:  mustSafeIntToInt32(length),
	}
	resp, err := c.client.SPIRead(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}
	return resp.Data, nil
}

// I2CWrite performs an I2C write on the agent
func (c *PiAgentClient) I2CWrite(ctx context.Context, bus int, address int, data []byte) error {
	req := &proto.I2CWriteRequest{
		Bus:     mustSafeIntToInt32(bus),
		Address: mustSafeIntToInt32(address),
		Data:    data,
	}
	resp, err := c.client.I2CWrite(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("I2C write failed: %s", resp.Message)
	}
	return nil
}

// I2CRead performs an I2C read from the agent
func (c *PiAgentClient) I2CRead(ctx context.Context, bus int, address int, length int) ([]byte, error) {
	req := &proto.I2CReadRequest{
		Bus:     mustSafeIntToInt32(bus),
		Address: mustSafeIntToInt32(address),
		Length:  mustSafeIntToInt32(length),
	}
	resp, err := c.client.I2CRead(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}
	return resp.Data, nil
}

// I2CWriteRegister performs an I2C write to a register on the agent
func (c *PiAgentClient) I2CWriteRegister(ctx context.Context, bus int, address int, register int, data []byte) error {
	req := &proto.I2CWriteRegisterRequest{
		Bus:      mustSafeIntToInt32(bus),
		Address:  mustSafeIntToInt32(address),
		Register: mustSafeIntToInt32(register),
		Data:     data,
	}
	resp, err := c.client.I2CWriteRegister(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("I2C write register failed: %s", resp.Message)
	}
	return nil
}

// I2CReadRegister performs an I2C read from a register on the agent
func (c *PiAgentClient) I2CReadRegister(ctx context.Context, bus int, address int, register int, length int) ([]byte, error) {
	req := &proto.I2CReadRegisterRequest{
		Bus:      mustSafeIntToInt32(bus),
		Address:  mustSafeIntToInt32(address),
		Register: mustSafeIntToInt32(register),
		Length:   mustSafeIntToInt32(length),
	}
	resp, err := c.client.I2CReadRegister(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}
	return resp.Data, nil
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

func modelDirectionToProto(direction models.GPIODirection) proto.AgentGPIODirection {
	switch direction {
	case models.GPIODirectionInput:
		return proto.AgentGPIODirection_AGENT_GPIO_DIRECTION_INPUT
	case models.GPIODirectionOutput:
		return proto.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT
	default:
		return proto.AgentGPIODirection_AGENT_GPIO_DIRECTION_UNSPECIFIED
	}
}

func modelPullModeToProto(pullMode models.GPIOPullMode) proto.AgentGPIOPullMode {
	switch pullMode {
	case models.GPIOPullNone:
		return proto.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE
	case models.GPIOPullUp:
		return proto.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UP
	case models.GPIOPullDown:
		return proto.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_DOWN
	default:
		return proto.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UNSPECIFIED
	}
}
