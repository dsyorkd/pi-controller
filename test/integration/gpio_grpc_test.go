package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/grpc/client"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// MockPiAgentClient implements PiAgentClientInterface for testing
type MockPiAgentClient struct {
	mock.Mock
}

func (m *MockPiAgentClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPiAgentClient) ConfigureGPIOPin(ctx context.Context, device *models.GPIODevice) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockPiAgentClient) ReadGPIOPin(ctx context.Context, pinNumber int) (int, error) {
	args := m.Called(ctx, pinNumber)
	return args.Int(0), args.Error(1)
}

func (m *MockPiAgentClient) WriteGPIOPin(ctx context.Context, pinNumber int, value int) error {
	args := m.Called(ctx, pinNumber, value)
	return args.Error(0)
}

func (m *MockPiAgentClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockPiAgentClientManager implements PiAgentClientManagerInterface for testing
type MockPiAgentClientManager struct {
	mock.Mock
	mockClient *MockPiAgentClient
}

func NewMockPiAgentClientManager() *MockPiAgentClientManager {
	return &MockPiAgentClientManager{
		mockClient: &MockPiAgentClient{},
	}
}

func (m *MockPiAgentClientManager) GetClient(node *models.Node) (client.PiAgentClientInterface, error) {
	args := m.Called(node)
	return m.mockClient, args.Error(1)
}

func (m *MockPiAgentClientManager) CloseClient(nodeID uint) error {
	args := m.Called(nodeID)
	return args.Error(0)
}

func (m *MockPiAgentClientManager) CloseAll() error {
	args := m.Called()
	return args.Error(0)
}

func TestGPIOService_ReadWithGRPCClient(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-input",
		NodeID:     node.ID,
		PinNumber:  18,
		Direction:  models.GPIODirectionInput,
		PullMode:   models.GPIOPullUp,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock expectations
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, device).Return(nil)
	mockManager.mockClient.On("ReadGPIOPin", mock.Anything, 18).Return(1, nil)

	// Test reading GPIO device
	result, err := gpioService.Read(device.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Value)

	// Verify all expectations were met
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}

func TestGPIOService_WriteWithGRPCClient(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-output",
		NodeID:     node.ID,
		PinNumber:  25,
		Direction:  models.GPIODirectionOutput,
		PullMode:   models.GPIOPullNone,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock expectations
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, device).Return(nil)
	mockManager.mockClient.On("WriteGPIOPin", mock.Anything, 25, 1).Return(nil)

	// Test writing to GPIO device
	err = gpioService.Write(device.ID, 1)
	require.NoError(t, err)

	// Verify device value was updated in database
	var updatedDevice models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice, device.ID).Error)
	assert.Equal(t, 1, updatedDevice.Value)

	// Verify all expectations were met
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}

func TestGPIOService_ClientConnectionError(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-input",
		NodeID:     node.ID,
		PinNumber:  18,
		Direction:  models.GPIODirectionInput,
		PullMode:   models.GPIOPullUp,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock to return connection error
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(nil, assert.AnError)

	// Test reading GPIO device with connection error
	result, err := gpioService.Read(device.ID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to connect to node")

	// Verify expectations
	mockManager.AssertExpectations(t)
}

func TestGPIOService_ConfigurationError(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-input",
		NodeID:     node.ID,
		PinNumber:  18,
		Direction:  models.GPIODirectionInput,
		PullMode:   models.GPIOPullUp,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock expectations - configuration fails
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, device).Return(assert.AnError)

	// Test reading GPIO device with configuration error
	result, err := gpioService.Read(device.ID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to configure GPIO pin")

	// Verify expectations
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}

func TestGPIOService_HardwareReadError(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-input",
		NodeID:     node.ID,
		PinNumber:  18,
		Direction:  models.GPIODirectionInput,
		PullMode:   models.GPIOPullUp,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock expectations - hardware read fails
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, device).Return(nil)
	mockManager.mockClient.On("ReadGPIOPin", mock.Anything, 18).Return(0, assert.AnError)

	// Test reading GPIO device with hardware error
	result, err := gpioService.Read(device.ID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to read GPIO pin")

	// Verify expectations
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}

func TestGPIOService_HardwareWriteError(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-output",
		NodeID:     node.ID,
		PinNumber:  25,
		Direction:  models.GPIODirectionOutput,
		PullMode:   models.GPIOPullNone,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock expectations - hardware write fails
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, device).Return(nil)
	mockManager.mockClient.On("WriteGPIOPin", mock.Anything, 25, 1).Return(assert.AnError)

	// Test writing to GPIO device with hardware error
	err = gpioService.Write(device.ID, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write GPIO pin")

	// Verify expectations
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}

func TestGPIOService_Close(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Setup mock expectations
	mockManager.On("CloseAll").Return(nil)

	// Test closing the service
	err = gpioService.Close()
	require.NoError(t, err)

	// Verify expectations
	mockManager.AssertExpectations(t)
}

func TestGPIOService_CloseError(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Setup mock expectations - close fails
	mockManager.On("CloseAll").Return(assert.AnError)

	// Test closing the service with error
	err = gpioService.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close agent connections")

	// Verify expectations
	mockManager.AssertExpectations(t)
}

func TestGPIOService_ReadingCreation(t *testing.T) {
	// Setup test database
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Setup mock client manager
	mockManager := NewMockPiAgentClientManager()
	
	// Create GPIO service with mock manager
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	// Create test data
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)

	device := &models.GPIODevice{
		Name:       "test-input",
		NodeID:     node.ID,
		PinNumber:  18,
		Direction:  models.GPIODirectionInput,
		PullMode:   models.GPIOPullUp,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)

	// Setup mock expectations
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, device).Return(nil)
	mockManager.mockClient.On("ReadGPIOPin", mock.Anything, 18).Return(1, nil)

	// Test reading GPIO device
	result, err := gpioService.Read(device.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify reading was created in database
	var readings []models.GPIOReading
	require.NoError(t, db.DB().Where("device_id = ?", device.ID).Find(&readings).Error)
	assert.Len(t, readings, 1)
	assert.Equal(t, float64(1), readings[0].Value)
	assert.Equal(t, device.ID, readings[0].DeviceID)
	assert.WithinDuration(t, time.Now(), readings[0].Timestamp, 5*time.Second)

	// Verify expectations
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}