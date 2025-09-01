package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/spenceryork/pi-controller/internal/models"
	"github.com/spenceryork/pi-controller/internal/storage"
)

// GPIOHandler handles GPIO-related API operations
type GPIOHandler struct {
	database *storage.Database
	logger   *logrus.Logger
}

// NewGPIOHandler creates a new GPIO handler
func NewGPIOHandler(db *storage.Database, logger *logrus.Logger) *GPIOHandler {
	return &GPIOHandler{
		database: db,
		logger:   logger,
	}
}

// CreateGPIORequest represents the request to create a GPIO device
type CreateGPIORequest struct {
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	PinNumber   int                     `json:"pin_number" binding:"required"`
	Direction   models.GPIODirection    `json:"direction"`
	PullMode    models.GPIOPullMode     `json:"pull_mode"`
	DeviceType  models.GPIODeviceType   `json:"device_type"`
	NodeID      uint                    `json:"node_id" binding:"required"`
	Config      models.GPIOConfig       `json:"config"`
}

// UpdateGPIORequest represents the request to update a GPIO device
type UpdateGPIORequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Direction   *models.GPIODirection   `json:"direction,omitempty"`
	PullMode    *models.GPIOPullMode    `json:"pull_mode,omitempty"`
	DeviceType  *models.GPIODeviceType  `json:"device_type,omitempty"`
	Status      *models.GPIOStatus      `json:"status,omitempty"`
	Config      *models.GPIOConfig      `json:"config,omitempty"`
}

// WriteGPIORequest represents the request to write a value to GPIO
type WriteGPIORequest struct {
	Value int `json:"value" binding:"required"`
}

// List returns all GPIO devices
func (h *GPIOHandler) List(c *gin.Context) {
	var gpioDevices []models.GPIODevice
	
	query := h.database.DB().Preload("Node")
	
	// Filter by node if specified
	if nodeID := c.Query("node_id"); nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}
	
	// Filter by device type if specified
	if deviceType := c.Query("device_type"); deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}
	
	// Filter by status if specified
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	result := query.Find(&gpioDevices)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to list GPIO devices")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO devices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"gpio_devices": gpioDevices,
		"count":        len(gpioDevices),
	})
}

// Create creates a new GPIO device
func (h *GPIOHandler) Create(c *gin.Context) {
	var req CreateGPIORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Verify node exists
	var node models.Node
	result := h.database.DB().First(&node, req.NodeID)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request",
				"message": "Node not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to verify node exists")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to verify node",
		})
		return
	}

	// Check for duplicate pin on same node
	var existingDevice models.GPIODevice
	result = h.database.DB().Where("node_id = ? AND pin_number = ?", req.NodeID, req.PinNumber).First(&existingDevice)
	if result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "GPIO pin already in use on this node",
		})
		return
	}

	device := models.GPIODevice{
		Name:        req.Name,
		Description: req.Description,
		PinNumber:   req.PinNumber,
		Direction:   req.Direction,
		PullMode:    req.PullMode,
		DeviceType:  req.DeviceType,
		NodeID:      req.NodeID,
		Config:      req.Config,
		Status:      models.GPIOStatusActive,
	}

	// Set defaults if not provided
	if device.Direction == "" {
		device.Direction = models.GPIODirectionInput
	}
	if device.PullMode == "" {
		device.PullMode = models.GPIOPullNone
	}
	if device.DeviceType == "" {
		device.DeviceType = models.GPIODeviceTypeDigital
	}

	result = h.database.DB().Create(&device)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to create GPIO device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create GPIO device",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"device_id": device.ID,
		"node_id":   device.NodeID,
		"pin":       device.PinNumber,
	}).Info("Created new GPIO device")

	c.JSON(http.StatusCreated, device)
}

// Get returns a specific GPIO device by ID
func (h *GPIOHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid GPIO device ID",
		})
		return
	}

	var device models.GPIODevice
	result := h.database.DB().Preload("Node").First(&device, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "GPIO device not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to get GPIO device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO device",
		})
		return
	}

	c.JSON(http.StatusOK, device)
}

// Update updates a GPIO device
func (h *GPIOHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid GPIO device ID",
		})
		return
	}

	var req UpdateGPIORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	var device models.GPIODevice
	result := h.database.DB().First(&device, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "GPIO device not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find GPIO device for update")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO device",
		})
		return
	}

	// Update fields if provided
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
	if req.DeviceType != nil {
		device.DeviceType = *req.DeviceType
	}
	if req.Status != nil {
		device.Status = *req.Status
	}
	if req.Config != nil {
		device.Config = *req.Config
	}

	result = h.database.DB().Save(&device)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to update GPIO device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update GPIO device",
		})
		return
	}

	h.logger.WithField("device_id", device.ID).Info("Updated GPIO device")
	c.JSON(http.StatusOK, device)
}

// Delete deletes a GPIO device
func (h *GPIOHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid GPIO device ID",
		})
		return
	}

	result := h.database.DB().Delete(&models.GPIODevice{}, uint(id))
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to delete GPIO device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to delete GPIO device",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "GPIO device not found",
		})
		return
	}

	h.logger.WithField("device_id", id).Info("Deleted GPIO device")
	c.JSON(http.StatusNoContent, nil)
}

// Read reads the current value from a GPIO device
func (h *GPIOHandler) Read(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid GPIO device ID",
		})
		return
	}

	var device models.GPIODevice
	result := h.database.DB().First(&device, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "GPIO device not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find GPIO device for read")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO device",
		})
		return
	}

	if !device.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "GPIO device is not active",
		})
		return
	}

	// TODO: Implement actual GPIO read operation
	// This would interface with the actual GPIO hardware
	// For now, just return the stored value
	
	// Record the reading
	reading := models.GPIOReading{
		DeviceID:  device.ID,
		Value:     float64(device.Value),
		Timestamp: time.Now(),
	}
	
	h.database.DB().Create(&reading)

	c.JSON(http.StatusOK, gin.H{
		"device_id": device.ID,
		"pin":       device.PinNumber,
		"value":     device.Value,
		"timestamp": reading.Timestamp,
	})
}

// Write writes a value to a GPIO device
func (h *GPIOHandler) Write(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid GPIO device ID",
		})
		return
	}

	var req WriteGPIORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	var device models.GPIODevice
	result := h.database.DB().First(&device, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "GPIO device not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find GPIO device for write")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO device",
		})
		return
	}

	if !device.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "GPIO device is not active",
		})
		return
	}

	if !device.IsOutput() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "GPIO device is not configured for output",
		})
		return
	}

	// TODO: Implement actual GPIO write operation
	// This would interface with the actual GPIO hardware
	
	// Update the device value
	device.SetValue(req.Value)
	result = h.database.DB().Save(&device)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to update GPIO device value")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update GPIO device",
		})
		return
	}

	// Record the write operation
	reading := models.GPIOReading{
		DeviceID:  device.ID,
		Value:     float64(req.Value),
		Timestamp: time.Now(),
	}
	
	h.database.DB().Create(&reading)

	h.logger.WithFields(logrus.Fields{
		"device_id": device.ID,
		"pin":       device.PinNumber,
		"value":     req.Value,
	}).Info("Wrote value to GPIO device")

	c.JSON(http.StatusOK, gin.H{
		"device_id": device.ID,
		"pin":       device.PinNumber,
		"value":     req.Value,
		"timestamp": reading.Timestamp,
	})
}

// GetReadings returns historical readings for a GPIO device
func (h *GPIOHandler) GetReadings(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid GPIO device ID",
		})
		return
	}

	// Parse query parameters
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var since time.Time
	if s := c.Query("since"); s != "" {
		if parsedSince, err := time.Parse(time.RFC3339, s); err == nil {
			since = parsedSince
		}
	}

	var readings []models.GPIOReading
	query := h.database.DB().Where("device_id = ?", uint(id))
	
	if !since.IsZero() {
		query = query.Where("timestamp >= ?", since)
	}
	
	result := query.Order("timestamp DESC").Limit(limit).Find(&readings)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to get GPIO readings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO readings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": uint(id),
		"readings":  readings,
		"count":     len(readings),
	})
}