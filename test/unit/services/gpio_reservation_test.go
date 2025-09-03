package services

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

func setupGPIOServiceTest(t *testing.T) (*services.GPIOService, *storage.Database, *MockPiAgentClientManager) {
	db, err := storage.NewForTest(logger.Default())
	require.NoError(t, err)

	mockManager := NewMockPiAgentClientManager()
	gpioService := services.NewGPIOServiceWithManager(db, logger.Default(), mockManager)

	return gpioService, db, mockManager
}

func createTestNode(t *testing.T, db *storage.Database) *models.Node {
	node := &models.Node{
		Name:      "test-pi",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
	}
	require.NoError(t, db.DB().Create(node).Error)
	return node
}

func createTestGPIODevice(t *testing.T, db *storage.Database, nodeID uint) *models.GPIODevice {
	device := &models.GPIODevice{
		Name:       "test-gpio",
		NodeID:     nodeID,
		PinNumber:  18,
		Direction:  models.GPIODirectionOutput,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device).Error)
	return device
}

func TestGPIOService_ReservePin_Success(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	// Test basic reservation
	req := services.GPIOReservationRequest{
		ClientID: "client1",
	}

	err := gpioService.ReservePin(device.ID, req)
	require.NoError(t, err)

	// Verify reservation in database
	var updatedDevice models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice, device.ID).Error)

	assert.True(t, updatedDevice.IsReserved())
	assert.True(t, updatedDevice.IsReservedBy("client1"))
	assert.NotNil(t, updatedDevice.ReservedBy)
	assert.Equal(t, "client1", *updatedDevice.ReservedBy)
	assert.NotNil(t, updatedDevice.ReservedAt)
	assert.Nil(t, updatedDevice.ReservationTTL) // No TTL specified
}

func TestGPIOService_ReservePin_WithTTL(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	// Test reservation with TTL
	ttl := 30 * time.Minute
	req := services.GPIOReservationRequest{
		ClientID: "client1",
		TTL:      &ttl,
	}

	err := gpioService.ReservePin(device.ID, req)
	require.NoError(t, err)

	// Verify reservation in database
	var updatedDevice models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice, device.ID).Error)

	assert.True(t, updatedDevice.IsReserved())
	assert.True(t, updatedDevice.IsReservedBy("client1"))
	assert.NotNil(t, updatedDevice.ReservationTTL)
	expectedExpiry := time.Now().Add(ttl)
	assert.WithinDuration(t, expectedExpiry, *updatedDevice.ReservationTTL, 5*time.Second)
}

func TestGPIOService_ReservePin_AlreadyReservedByDifferentClient(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	// Reserve pin for client1
	req1 := services.GPIOReservationRequest{
		ClientID: "client1",
	}
	err := gpioService.ReservePin(device.ID, req1)
	require.NoError(t, err)

	// Try to reserve same pin for client2 - should fail
	req2 := services.GPIOReservationRequest{
		ClientID: "client2",
	}
	err = gpioService.ReservePin(device.ID, req2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already reserved")
	assert.Contains(t, err.Error(), "client1")
}

func TestGPIOService_ReservePin_ExtendReservationSameClient(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	// Reserve pin for client1 without TTL
	req1 := services.GPIOReservationRequest{
		ClientID: "client1",
	}
	err := gpioService.ReservePin(device.ID, req1)
	require.NoError(t, err)

	// Extend/update reservation with TTL - should succeed
	ttl := 1 * time.Hour
	req2 := services.GPIOReservationRequest{
		ClientID: "client1",
		TTL:      &ttl,
	}
	err = gpioService.ReservePin(device.ID, req2)
	require.NoError(t, err)

	// Verify TTL was added
	var updatedDevice models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice, device.ID).Error)
	assert.NotNil(t, updatedDevice.ReservationTTL)
}

func TestGPIOService_ReservePin_DeviceNotFound(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	req := services.GPIOReservationRequest{
		ClientID: "client1",
	}

	err := gpioService.ReservePin(999, req) // Non-existent device ID
	assert.Error(t, err)
	assert.Equal(t, services.ErrNotFound, err)
}

func TestGPIOService_ReleasePin_Success(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data and reserve pin
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	reserveReq := services.GPIOReservationRequest{
		ClientID: "client1",
	}
	err := gpioService.ReservePin(device.ID, reserveReq)
	require.NoError(t, err)

	// Release the pin
	releaseReq := services.GPIOReleaseRequest{
		ClientID: "client1",
	}
	err = gpioService.ReleasePin(device.ID, releaseReq)
	require.NoError(t, err)

	// Verify reservation is cleared
	var updatedDevice models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice, device.ID).Error)
	assert.False(t, updatedDevice.IsReserved())
	assert.Nil(t, updatedDevice.ReservedBy)
	assert.Nil(t, updatedDevice.ReservedAt)
	assert.Nil(t, updatedDevice.ReservationTTL)
}

func TestGPIOService_ReleasePin_NotReserved(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data (not reserved)
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	// Try to release unreserved pin
	req := services.GPIOReleaseRequest{
		ClientID: "client1",
	}
	err := gpioService.ReleasePin(device.ID, req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not reserved")
}

func TestGPIOService_ReleasePin_WrongClient(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data and reserve pin
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	reserveReq := services.GPIOReservationRequest{
		ClientID: "client1",
	}
	err := gpioService.ReservePin(device.ID, reserveReq)
	require.NoError(t, err)

	// Try to release with different client
	releaseReq := services.GPIOReleaseRequest{
		ClientID: "client2",
	}
	err = gpioService.ReleasePin(device.ID, releaseReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reserved by different client")
}

func TestGPIOService_GetReservations(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data with multiple pins
	node := createTestNode(t, db)
	device1 := createTestGPIODevice(t, db, node.ID)
	device2 := &models.GPIODevice{
		Name:       "test-gpio-2",
		NodeID:     node.ID,
		PinNumber:  19,
		Direction:  models.GPIODirectionInput,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device2).Error)

	// Reserve first pin
	req1 := services.GPIOReservationRequest{
		ClientID: "client1",
	}
	err := gpioService.ReservePin(device1.ID, req1)
	require.NoError(t, err)

	// Reserve second pin with TTL
	ttl := 1 * time.Hour
	req2 := services.GPIOReservationRequest{
		ClientID: "client2",
		TTL:      &ttl,
	}
	err = gpioService.ReservePin(device2.ID, req2)
	require.NoError(t, err)

	// Get reservations
	reservations, err := gpioService.GetReservations()
	require.NoError(t, err)
	assert.Len(t, reservations, 2)

	// Sort by pin number for consistent testing
	if reservations[0].PinNumber > reservations[1].PinNumber {
		reservations[0], reservations[1] = reservations[1], reservations[0]
	}

	// Check first reservation
	assert.Equal(t, device1.ID, reservations[0].PinID)
	assert.Equal(t, node.ID, reservations[0].NodeID)
	assert.Equal(t, 18, reservations[0].PinNumber)
	assert.Equal(t, "client1", reservations[0].ReservedBy)
	assert.Nil(t, reservations[0].ExpiresAt)

	// Check second reservation
	assert.Equal(t, device2.ID, reservations[1].PinID)
	assert.Equal(t, node.ID, reservations[1].NodeID)
	assert.Equal(t, 19, reservations[1].PinNumber)
	assert.Equal(t, "client2", reservations[1].ReservedBy)
	assert.NotNil(t, reservations[1].ExpiresAt)
}

func TestGPIOService_CleanupExpiredReservations(t *testing.T) {
	gpioService, db, _ := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data
	node := createTestNode(t, db)
	device1 := createTestGPIODevice(t, db, node.ID)
	device2 := &models.GPIODevice{
		Name:       "test-gpio-2",
		NodeID:     node.ID,
		PinNumber:  19,
		Direction:  models.GPIODirectionInput,
		DeviceType: models.GPIODeviceTypeDigital,
		Status:     models.GPIOStatusActive,
	}
	require.NoError(t, db.DB().Create(device2).Error)

	// Manually create expired reservation
	expiredTime := time.Now().Add(-1 * time.Hour)
	clientID1 := "expired-client"
	device1.ReservedBy = &clientID1
	device1.ReservedAt = &expiredTime
	device1.ReservationTTL = &expiredTime
	require.NoError(t, db.DB().Save(device1).Error)

	// Create active reservation (no expiry)
	clientID2 := "active-client"
	device2.ReservedBy = &clientID2
	device2.ReservedAt = &time.Time{}
	require.NoError(t, db.DB().Save(device2).Error)

	// Run cleanup
	count, err := gpioService.CleanupExpiredReservations()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify expired reservation was cleaned up
	var updatedDevice1 models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice1, device1.ID).Error)
	assert.False(t, updatedDevice1.IsReserved())

	// Verify active reservation remains
	var updatedDevice2 models.GPIODevice
	require.NoError(t, db.DB().First(&updatedDevice2, device2.ID).Error)
	assert.True(t, updatedDevice2.IsReserved())
}

func TestGPIOService_WriteWithClient_ReservationConflict(t *testing.T) {
	gpioService, db, mockManager := setupGPIOServiceTest(t)
	defer func() {
		assert.NoError(t, db.Close())
	}()

	// Create test data
	node := createTestNode(t, db)
	device := createTestGPIODevice(t, db, node.ID)

	// Reserve pin for client1
	req := services.GPIOReservationRequest{
		ClientID: "client1",
	}
	err := gpioService.ReservePin(device.ID, req)
	require.NoError(t, err)

	// Try to write without providing client ID - should fail
	err = gpioService.Write(device.ID, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reserved by")

	// Try to write with wrong client ID - should fail
	err = gpioService.WriteWithClient(device.ID, 1, "client2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reserved by")

	// Write with correct client ID - should succeed
	mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(mockManager.mockClient, nil)
	mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, mock.Anything).Return(nil)
	mockManager.mockClient.On("WriteGPIOPin", mock.Anything, 18, 1).Return(nil)

	err = gpioService.WriteWithClient(device.ID, 1, "client1")
	assert.NoError(t, err)

	// Verify expectations
	mockManager.AssertExpectations(t)
	mockManager.mockClient.AssertExpectations(t)
}
