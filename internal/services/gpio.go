package services

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/grpc/client"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// GPIOService handles GPIO device business logic
type GPIOService struct {
	db           *storage.Database
	logger       logger.Interface
	agentManager client.PiAgentClientManagerInterface
}

// NewGPIOService creates a new GPIO service
func NewGPIOService(db *storage.Database, logger logger.Interface) *GPIOService {
	return &GPIOService{
		db:           db,
		logger:       logger.WithField("service", "gpio"),
		agentManager: client.NewPiAgentClientManager(logger),
	}
}

// NewGPIOServiceWithManager creates a new GPIO service with a custom agent manager (for testing)
func NewGPIOServiceWithManager(db *storage.Database, logger logger.Interface, agentManager client.PiAgentClientManagerInterface) *GPIOService {
	return &GPIOService{
		db:           db,
		logger:       logger.WithField("service", "gpio"),
		agentManager: agentManager,
	}
}

// CreateGPIODeviceRequest represents the request to create a GPIO device
type CreateGPIODeviceRequest struct {
	Name        string                `json:"name" validate:"required,min=1,max=100"`
	Description string                `json:"description" validate:"max=500"`
	NodeID      uint                  `json:"node_id" validate:"required"`
	PinNumber   int                   `json:"pin_number" validate:"required,min=0,max=40"`
	Direction   models.GPIODirection  `json:"direction" validate:"required,oneof=input output"`
	PullMode    models.GPIOPullMode   `json:"pull_mode" validate:"oneof=none up down"`
	DeviceType  models.GPIODeviceType `json:"device_type" validate:"oneof=digital analog pwm spi i2c"`
	Config      models.GPIOConfig     `json:"config"`
}

// UpdateGPIODeviceRequest represents the request to update a GPIO device
type UpdateGPIODeviceRequest struct {
	Name        *string               `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string               `json:"description,omitempty" validate:"omitempty,max=500"`
	Direction   *models.GPIODirection `json:"direction,omitempty" validate:"omitempty,oneof=input output"`
	PullMode    *models.GPIOPullMode  `json:"pull_mode,omitempty" validate:"omitempty,oneof=none up down"`
	Status      *models.GPIOStatus    `json:"status,omitempty" validate:"omitempty,oneof=active inactive error"`
	Config      *models.GPIOConfig    `json:"config,omitempty"`
}

// GPIOListOptions represents options for listing GPIO devices
type GPIOListOptions struct {
	NodeID     *uint
	DeviceType *models.GPIODeviceType
	Direction  *models.GPIODirection
	Status     *models.GPIOStatus
	Limit      int
	Offset     int
}

// GPIOReadingFilter represents filtering options for GPIO readings
type GPIOReadingFilter struct {
	DeviceID  uint
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// GPIOReservationRequest represents a request to reserve a GPIO pin
type GPIOReservationRequest struct {
	ClientID string         `json:"client_id" validate:"required,min=1,max=100"`
	TTL      *time.Duration `json:"ttl,omitempty"` // Optional reservation time-to-live
}

// GPIOReleaseRequest represents a request to release a GPIO pin reservation
type GPIOReleaseRequest struct {
	ClientID string `json:"client_id" validate:"required,min=1,max=100"`
}

// GPIOReservationInfo represents information about a GPIO pin reservation
type GPIOReservationInfo struct {
	PinID      uint       `json:"pin_id"`
	NodeID     uint       `json:"node_id"`
	PinNumber  int        `json:"pin_number"`
	ReservedBy string     `json:"reserved_by"`
	ReservedAt time.Time  `json:"reserved_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// List returns a paginated list of GPIO devices
func (s *GPIOService) List(opts GPIOListOptions) ([]models.GPIODevice, int64, error) {
	var devices []models.GPIODevice
	var total int64

	query := s.db.DB().Model(&models.GPIODevice{})

	// Apply filters
	if opts.NodeID != nil {
		query = query.Where("node_id = ?", *opts.NodeID)
	}
	if opts.DeviceType != nil {
		query = query.Where("device_type = ?", *opts.DeviceType)
	}
	if opts.Direction != nil {
		query = query.Where("direction = ?", *opts.Direction)
	}
	if opts.Status != nil {
		query = query.Where("status = ?", *opts.Status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count GPIO devices")
		return nil, 0, errors.Wrapf(err, "failed to count GPIO devices")
	}

	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	// Include relationships
	query = query.Preload("Node")

	// Execute query
	if err := query.Find(&devices).Error; err != nil {
		s.logger.WithError(err).Error("Failed to fetch GPIO devices")
		return nil, 0, errors.Wrapf(err, "failed to fetch GPIO devices")
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(devices),
		"total": total,
	}).Debug("Fetched GPIO devices")

	return devices, total, nil
}

// GetByID returns a GPIO device by ID
func (s *GPIOService) GetByID(id uint) (*models.GPIODevice, error) {
	var device models.GPIODevice

	if err := s.db.DB().Preload("Node").First(&device, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to fetch GPIO device")
		return nil, errors.Wrapf(err, "failed to fetch GPIO device")
	}

	return &device, nil
}

// GetByNodeAndPin returns a GPIO device by node ID and pin number
func (s *GPIOService) GetByNodeAndPin(nodeID uint, pinNumber int) (*models.GPIODevice, error) {
	var device models.GPIODevice

	if err := s.db.DB().Preload("Node").Where("node_id = ? AND pin_number = ?", nodeID, pinNumber).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		s.logger.WithFields(map[string]interface{}{
			"node_id":    nodeID,
			"pin_number": pinNumber,
			"error":      err,
		}).Error("Failed to fetch GPIO device by node and pin")
		return nil, errors.Wrapf(err, "failed to fetch GPIO device")
	}

	return &device, nil
}

// Create creates a new GPIO device
func (s *GPIOService) Create(req CreateGPIODeviceRequest) (*models.GPIODevice, error) {
	// Validate node exists
	var node models.Node
	if err := s.db.DB().First(&node, req.NodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Wrapf(ErrNotFound, "node with ID %d not found", req.NodeID)
		}
		return nil, errors.Wrapf(err, "failed to validate node")
	}

	// Check if pin is already in use on this node
	if _, err := s.GetByNodeAndPin(req.NodeID, req.PinNumber); err != ErrNotFound {
		if err == nil {
			return nil, errors.Wrapf(ErrAlreadyExists, "pin %d is already in use on node %d", req.PinNumber, req.NodeID)
		}
		return nil, err
	}

	device := models.GPIODevice{
		Name:        req.Name,
		Description: req.Description,
		NodeID:      req.NodeID,
		PinNumber:   req.PinNumber,
		Direction:   req.Direction,
		PullMode:    req.PullMode,
		DeviceType:  req.DeviceType,
		Status:      models.GPIOStatusActive,
		Config:      req.Config,
	}

	if err := s.db.DB().Create(&device).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"name":       req.Name,
			"node_id":    req.NodeID,
			"pin_number": req.PinNumber,
			"error":      err,
		}).Error("Failed to create GPIO device")
		return nil, errors.Wrapf(err, "failed to create GPIO device")
	}

	s.logger.WithFields(map[string]interface{}{
		"id":         device.ID,
		"name":       device.Name,
		"node_id":    device.NodeID,
		"pin_number": device.PinNumber,
	}).Info("GPIO device created successfully")

	return &device, nil
}

// Update updates an existing GPIO device
func (s *GPIOService) Update(id uint, req UpdateGPIODeviceRequest) (*models.GPIODevice, error) {
	device, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		device.Name = *req.Name
	}
	if req.Description != nil {
		device.Description = *req.Description
	}
	if req.Direction != nil {
		device.Direction = *req.Direction
	}
	if req.PullMode != nil {
		device.PullMode = *req.PullMode
	}
	if req.Status != nil {
		device.Status = *req.Status
	}
	if req.Config != nil {
		device.Config = *req.Config
	}

	if err := s.db.DB().Save(device).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to update GPIO device")
		return nil, errors.Wrapf(err, "failed to update GPIO device")
	}

	s.logger.WithFields(map[string]interface{}{
		"id":   device.ID,
		"name": device.Name,
	}).Info("GPIO device updated successfully")

	return device, nil
}

// Delete deletes a GPIO device
func (s *GPIOService) Delete(id uint) error {
	device, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Delete associated readings first
	if err := s.db.DB().Where("device_id = ?", id).Delete(&models.GPIOReading{}).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"error":     err,
		}).Error("Failed to delete GPIO readings")
		return errors.Wrapf(err, "failed to delete GPIO readings")
	}

	// Delete the device
	if err := s.db.DB().Delete(&models.GPIODevice{}, id).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to delete GPIO device")
		return errors.Wrapf(err, "failed to delete GPIO device")
	}

	s.logger.WithFields(map[string]interface{}{
		"id":   id,
		"name": device.Name,
	}).Info("GPIO device deleted successfully")

	return nil
}

// Read reads the current value of a GPIO device
func (s *GPIOService) Read(id uint) (*models.GPIODevice, error) {
	device, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !device.IsActive() {
		return nil, errors.Wrapf(ErrValidationFailed, "GPIO device %d is not active", id)
	}

	// Check if pin is reserved - we allow reads from reserved pins as they are generally safe
	// but log for auditing purposes
	if device.IsReserved() {
		s.logger.WithFields(map[string]interface{}{
			"device_id":   id,
			"pin_number":  device.PinNumber,
			"reserved_by": *device.ReservedBy,
		}).Debug("Reading from reserved GPIO pin")
	}

	// Get gRPC client for the node
	agentClient, err := s.agentManager.GetClient(&device.Node)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"node_id":   device.NodeID,
			"error":     err,
		}).Error("Failed to get Pi Agent client")
		return nil, errors.Wrapf(err, "failed to connect to node %d", device.NodeID)
	}

	// Configure the pin if not already configured
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := agentClient.ConfigureGPIOPin(ctx, device); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"pin":       device.PinNumber,
			"error":     err,
		}).Error("Failed to configure GPIO pin")
		return nil, errors.Wrapf(err, "failed to configure GPIO pin %d", device.PinNumber)
	}

	// Read the GPIO pin value from the hardware
	actualValue, err := agentClient.ReadGPIOPin(ctx, device.PinNumber)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"pin":       device.PinNumber,
			"error":     err,
		}).Error("Failed to read GPIO pin from hardware")
		return nil, errors.Wrapf(err, "failed to read GPIO pin %d from hardware", device.PinNumber)
	}

	// Update the device value in database if it changed
	if device.Value != actualValue {
		device.SetValue(actualValue)
		if err := s.db.DB().Save(device).Error; err != nil {
			s.logger.WithFields(map[string]interface{}{
				"device_id": id,
				"value":     actualValue,
				"error":     err,
			}).Error("Failed to update device value after hardware read")
			// Don't return error here, the read was successful
		}
	}

	// Create a reading record with the actual hardware value
	reading := models.GPIOReading{
		DeviceID:  device.ID,
		Value:     float64(actualValue),
		Timestamp: time.Now(),
	}

	if err := s.db.DB().Create(&reading).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"error":     err,
		}).Error("Failed to create GPIO reading")
		return nil, errors.Wrapf(err, "failed to create GPIO reading")
	}

	s.logger.WithFields(map[string]interface{}{
		"device_id": id,
		"value":     device.Value,
	}).Debug("GPIO device read successfully")

	return device, nil
}

// Write writes a value to a GPIO device
func (s *GPIOService) Write(id uint, value int) error {
	return s.WriteWithClient(id, value, "")
}

// WriteWithClient writes a value to a GPIO device with client ID for reservation checking
func (s *GPIOService) WriteWithClient(id uint, value int, clientID string) error {
	device, err := s.GetByID(id)
	if err != nil {
		return err
	}

	if !device.IsActive() {
		return errors.Wrapf(ErrValidationFailed, "GPIO device %d is not active", id)
	}

	if !device.IsOutput() {
		return errors.Wrapf(ErrValidationFailed, "GPIO device %d is not configured as output", id)
	}

	// Check for reservation conflicts on writes (more critical than reads)
	if device.IsReserved() {
		if clientID == "" || !device.IsReservedBy(clientID) {
			reservedBy := "unknown"
			if device.ReservedBy != nil {
				reservedBy = *device.ReservedBy
			}
			return errors.Wrapf(ErrValidationFailed, "GPIO pin %d on node %d is reserved by %s",
				device.PinNumber, device.NodeID, reservedBy)
		}
		// Log authorized write to reserved pin
		s.logger.WithFields(map[string]interface{}{
			"device_id":  id,
			"pin_number": device.PinNumber,
			"client_id":  clientID,
			"value":      value,
		}).Debug("Writing to reserved GPIO pin by authorized client")
	}

	// Get gRPC client for the node
	agentClient, err := s.agentManager.GetClient(&device.Node)
	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"node_id":   device.NodeID,
			"error":     err,
		}).Error("Failed to get Pi Agent client")
		return errors.Wrapf(err, "failed to connect to node %d", device.NodeID)
	}

	// Configure the pin if not already configured
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := agentClient.ConfigureGPIOPin(ctx, device); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"pin":       device.PinNumber,
			"error":     err,
		}).Error("Failed to configure GPIO pin")
		return errors.Wrapf(err, "failed to configure GPIO pin %d", device.PinNumber)
	}

	// Write the value to the hardware GPIO pin
	if err := agentClient.WriteGPIOPin(ctx, device.PinNumber, value); err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"pin":       device.PinNumber,
			"value":     value,
			"error":     err,
		}).Error("Failed to write GPIO pin to hardware")
		return errors.Wrapf(err, "failed to write GPIO pin %d to hardware", device.PinNumber)
	}

	// Update device value in database after successful hardware write
	device.SetValue(value)
	if err := s.db.DB().Save(device).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"value":     value,
			"error":     err,
		}).Error("Failed to update GPIO device value after hardware write")
		return errors.Wrapf(err, "failed to update GPIO device value")
	}

	// Create a reading record
	reading := models.GPIOReading{
		DeviceID:  device.ID,
		Value:     float64(value),
		Timestamp: time.Now(),
	}

	if err := s.db.DB().Create(&reading).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"value":     value,
			"error":     err,
		}).Error("Failed to create GPIO reading after write")
		// Don't return error here as the write was successful
	}

	s.logger.WithFields(map[string]interface{}{
		"device_id": id,
		"value":     value,
	}).Info("GPIO device written successfully")

	return nil
}

// GetReadings returns GPIO readings for a device with optional filtering
func (s *GPIOService) GetReadings(filter GPIOReadingFilter) ([]models.GPIOReading, int64, error) {
	var readings []models.GPIOReading
	var total int64

	query := s.db.DB().Model(&models.GPIOReading{}).Where("device_id = ?", filter.DeviceID)

	// Apply time range filters
	if filter.StartTime != nil {
		query = query.Where("timestamp >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("timestamp <= ?", *filter.EndTime)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count GPIO readings")
		return nil, 0, errors.Wrapf(err, "failed to count GPIO readings")
	}

	// Apply pagination and ordering
	query = query.Order("timestamp DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	// Execute query
	if err := query.Find(&readings).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": filter.DeviceID,
			"error":     err,
		}).Error("Failed to fetch GPIO readings")
		return nil, 0, errors.Wrapf(err, "failed to fetch GPIO readings")
	}

	s.logger.WithFields(map[string]interface{}{
		"device_id": filter.DeviceID,
		"count":     len(readings),
		"total":     total,
	}).Debug("Fetched GPIO readings")

	return readings, total, nil
}

// CleanupOldReadings removes GPIO readings older than the specified duration
func (s *GPIOService) CleanupOldReadings(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result := s.db.DB().Where("timestamp < ?", cutoffTime).Delete(&models.GPIOReading{})
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to cleanup old GPIO readings")
		return 0, errors.Wrapf(result.Error, "failed to cleanup old GPIO readings")
	}

	s.logger.WithFields(map[string]interface{}{
		"deleted_count": result.RowsAffected,
		"cutoff_time":   cutoffTime,
	}).Info("Cleaned up old GPIO readings")

	return result.RowsAffected, nil
}

// ReservePin reserves a GPIO pin for a specific client
func (s *GPIOService) ReservePin(id uint, req GPIOReservationRequest) error {
	device, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Check if pin is already reserved by someone else
	if device.IsReserved() && !device.IsReservedBy(req.ClientID) {
		return errors.Wrapf(ErrAlreadyExists, "GPIO pin %d on node %d is already reserved by %s",
			device.PinNumber, device.NodeID, *device.ReservedBy)
	}

	// If already reserved by the same client, extend/update the reservation
	device.Reserve(req.ClientID, req.TTL)

	if err := s.db.DB().Save(device).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"client_id": req.ClientID,
			"error":     err,
		}).Error("Failed to reserve GPIO pin")
		return errors.Wrapf(err, "failed to reserve GPIO pin")
	}

	s.logger.WithFields(map[string]interface{}{
		"device_id":  id,
		"pin_number": device.PinNumber,
		"node_id":    device.NodeID,
		"client_id":  req.ClientID,
		"expires_at": device.ReservationTTL,
	}).Info("GPIO pin reserved successfully")

	return nil
}

// ReleasePin releases a GPIO pin reservation
func (s *GPIOService) ReleasePin(id uint, req GPIOReleaseRequest) error {
	device, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// Check if pin is reserved
	if !device.IsReserved() {
		return errors.Wrapf(ErrNotFound, "GPIO pin %d on node %d is not reserved",
			device.PinNumber, device.NodeID)
	}

	// Check if client has permission to release this pin
	if !device.IsReservedBy(req.ClientID) {
		return errors.Wrapf(ErrValidationFailed, "GPIO pin %d on node %d is reserved by different client",
			device.PinNumber, device.NodeID)
	}

	device.Release()

	if err := s.db.DB().Save(device).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"device_id": id,
			"client_id": req.ClientID,
			"error":     err,
		}).Error("Failed to release GPIO pin")
		return errors.Wrapf(err, "failed to release GPIO pin")
	}

	s.logger.WithFields(map[string]interface{}{
		"device_id":  id,
		"pin_number": device.PinNumber,
		"node_id":    device.NodeID,
		"client_id":  req.ClientID,
	}).Info("GPIO pin released successfully")

	return nil
}

// GetReservations returns all active GPIO pin reservations
func (s *GPIOService) GetReservations() ([]GPIOReservationInfo, error) {
	var devices []models.GPIODevice

	// Query only reserved pins
	if err := s.db.DB().Where("reserved_by IS NOT NULL").Preload("Node").Find(&devices).Error; err != nil {
		s.logger.WithError(err).Error("Failed to fetch GPIO reservations")
		return nil, errors.Wrapf(err, "failed to fetch GPIO reservations")
	}

	// Filter out expired reservations and convert to info structs
	var reservations []GPIOReservationInfo

	for _, device := range devices {
		// Skip expired reservations
		if device.IsReservationExpired() {
			// Cleanup expired reservation in background (non-blocking)
			go func(d models.GPIODevice) {
				d.Release()
				if err := s.db.DB().Save(&d).Error; err != nil {
					s.logger.WithFields(map[string]interface{}{
						"device_id": d.ID,
						"error":     err,
					}).Error("Failed to cleanup expired reservation")
				}
			}(device)
			continue
		}

		reservation := GPIOReservationInfo{
			PinID:      device.ID,
			NodeID:     device.NodeID,
			PinNumber:  device.PinNumber,
			ReservedBy: *device.ReservedBy,
			ReservedAt: *device.ReservedAt,
			ExpiresAt:  device.ReservationTTL,
		}
		reservations = append(reservations, reservation)
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(reservations),
	}).Debug("Fetched GPIO reservations")

	return reservations, nil
}

// CleanupExpiredReservations removes all expired GPIO pin reservations
func (s *GPIOService) CleanupExpiredReservations() (int64, error) {
	now := time.Now()

	// Find expired reservations
	var expiredDevices []models.GPIODevice
	if err := s.db.DB().Where("reserved_by IS NOT NULL AND reservation_ttl IS NOT NULL AND reservation_ttl < ?", now).Find(&expiredDevices).Error; err != nil {
		s.logger.WithError(err).Error("Failed to find expired reservations")
		return 0, errors.Wrapf(err, "failed to find expired reservations")
	}

	if len(expiredDevices) == 0 {
		return 0, nil
	}

	// Release expired reservations
	for i := range expiredDevices {
		expiredDevices[i].Release()
	}

	// Update all devices in a transaction
	err := s.db.DB().Transaction(func(tx *gorm.DB) error {
		for _, device := range expiredDevices {
			if err := tx.Save(&device).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		s.logger.WithFields(map[string]interface{}{
			"count": len(expiredDevices),
			"error": err,
		}).Error("Failed to cleanup expired reservations")
		return 0, errors.Wrapf(err, "failed to cleanup expired reservations")
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(expiredDevices),
	}).Info("Cleaned up expired GPIO pin reservations")

	return int64(len(expiredDevices)), nil
}

// ConfigurePin configures a GPIO pin for Kubernetes controller integration
func (s *GPIOService) ConfigurePin(ctx context.Context, req *GPIORequest) error {
	s.logger.WithFields(map[string]interface{}{
		"node_id":     req.NodeID,
		"pin_number":  req.PinNumber,
		"mode":        req.Mode,
		"direction":   req.Direction,
		"value":       req.Value,
	}).Info("Configuring GPIO pin for Kubernetes controller")

	// Handle cleanup mode
	if req.Mode == "cleanup" {
		return s.cleanupPin(ctx, req)
	}

	// For now, this is a stub implementation that would integrate with the gRPC client
	// In a full implementation, this would:
	// 1. Find or create a GPIO device entry
	// 2. Get the gRPC client for the target node
	// 3. Configure the pin via gRPC call
	// 4. Update the device status

	// TODO: Implement actual gRPC communication
	s.logger.WithFields(map[string]interface{}{
		"node_id":     req.NodeID,
		"pin_number":  req.PinNumber,
		"mode":        req.Mode,
	}).Info("GPIO pin configuration complete (stub implementation)")

	return nil
}

// cleanupPin handles GPIO pin cleanup
func (s *GPIOService) cleanupPin(ctx context.Context, req *GPIORequest) error {
	s.logger.WithFields(map[string]interface{}{
		"node_id":    req.NodeID,
		"pin_number": req.PinNumber,
	}).Info("Cleaning up GPIO pin")

	// TODO: Implement actual cleanup logic
	// This would typically:
	// 1. Set pin to safe state (input mode, low value)
	// 2. Release any hardware reservations
	// 3. Update database status

	return nil
}

// Close gracefully closes the GPIO service and all agent connections
func (s *GPIOService) Close() error {
	s.logger.Info("Shutting down GPIO service")

	if err := s.agentManager.CloseAll(); err != nil {
		s.logger.WithError(err).Error("Failed to close all agent connections")
		return errors.Wrapf(err, "failed to close agent connections")
	}

	s.logger.Info("GPIO service shut down successfully")
	return nil
}
