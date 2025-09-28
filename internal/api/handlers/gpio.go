package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// GPIOHandler handles GPIO-related API operations
type GPIOHandler struct {
	service *services.GPIOService
	logger  logger.Interface
}

// NewGPIOHandler creates a new GPIO handler
func NewGPIOHandler(service *services.GPIOService, logger logger.Interface) *GPIOHandler {
	return &GPIOHandler{
		service: service,
		logger:  logger.WithField("handler", "gpio"),
	}
}

// ReservePin reserves a GPIO pin for exclusive use by a client
func (h *GPIOHandler) ReservePin(c *gin.Context) {
	// Get GPIO device ID from URL parameter
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	// Parse request body
	var req services.GPIOReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid reservation request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Request",
			"message": err.Error(),
		})
		return
	}

	// Reserve the pin
	if err := h.service.ReservePin(uint(id), req); err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"gpio_id":   id,
			"client_id": req.ClientID,
		}).Error("Failed to reserve GPIO pin")

		switch err {
		case services.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "GPIO device not found",
			})
		case services.ErrAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Already Reserved",
				"message": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Failed to reserve GPIO pin",
			})
		}
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"gpio_id":   id,
		"client_id": req.ClientID,
	}).Info("GPIO pin reserved successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":   "GPIO pin reserved successfully",
		"gpio_id":   id,
		"client_id": req.ClientID,
	})
}

// ReleasePin releases a GPIO pin reservation
func (h *GPIOHandler) ReleasePin(c *gin.Context) {
	// Get GPIO device ID from URL parameter
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	// Parse request body
	var req services.GPIOReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid release request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Request",
			"message": err.Error(),
		})
		return
	}

	// Release the pin
	if err := h.service.ReleasePin(uint(id), req); err != nil {
		h.logger.WithError(err).WithFields(map[string]interface{}{
			"gpio_id":   id,
			"client_id": req.ClientID,
		}).Error("Failed to release GPIO pin")

		switch err {
		case services.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "GPIO device not found or not reserved",
			})
		case services.ErrValidationFailed:
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Failed to release GPIO pin",
			})
		}
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"gpio_id":   id,
		"client_id": req.ClientID,
	}).Info("GPIO pin released successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":   "GPIO pin released successfully",
		"gpio_id":   id,
		"client_id": req.ClientID,
	})
}

// GetReservations returns all active GPIO pin reservations
func (h *GPIOHandler) GetReservations(c *gin.Context) {
	reservations, err := h.service.GetReservations()
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch GPIO reservations")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to fetch GPIO reservations",
		})
		return
	}

	h.logger.WithField("count", len(reservations)).Debug("Retrieved GPIO reservations")

	c.JSON(http.StatusOK, gin.H{
		"reservations": reservations,
		"count":        len(reservations),
	})
}

// CleanupExpiredReservations manually triggers cleanup of expired reservations
func (h *GPIOHandler) CleanupExpiredReservations(c *gin.Context) {
	count, err := h.service.CleanupExpiredReservations()
	if err != nil {
		h.logger.WithError(err).Error("Failed to cleanup expired reservations")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to cleanup expired reservations",
		})
		return
	}

	h.logger.WithField("count", count).Info("Cleaned up expired GPIO reservations")

	c.JSON(http.StatusOK, gin.H{
		"message":       "Expired reservations cleaned up successfully",
		"cleaned_count": count,
	})
}

// List returns all GPIO devices
func (h *GPIOHandler) List(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	opts := services.GPIOListOptions{
		Limit:  limit,
		Offset: offset,
	}

	devices, total, err := h.service.List(opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list GPIO devices")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  devices,
		"count": len(devices),
		"total": total,
	})
}

// Create creates a new GPIO device
func (h *GPIOHandler) Create(c *gin.Context) {
	var req services.CreateGPIODeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid GPIO device creation request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Request",
			"message": err.Error(),
		})
		return
	}

	device, err := h.service.Create(req)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create GPIO device")
		return
	}

	h.logger.WithField("device_id", device.ID).Info("GPIO device created successfully")
	c.JSON(http.StatusCreated, gin.H{
		"data": device,
	})
}

// Get returns a specific GPIO device by ID
func (h *GPIOHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	device, err := h.service.GetByID(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to get GPIO device")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": device,
	})
}

// Update updates a GPIO device
func (h *GPIOHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	var req services.UpdateGPIODeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid GPIO device update request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Request",
			"message": err.Error(),
		})
		return
	}

	device, err := h.service.Update(uint(id), req)
	if err != nil {
		h.handleServiceError(c, err, "Failed to update GPIO device")
		return
	}

	h.logger.WithField("device_id", device.ID).Info("GPIO device updated successfully")
	c.JSON(http.StatusOK, gin.H{
		"data": device,
	})
}

// Delete deletes a GPIO device
func (h *GPIOHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	err = h.service.Delete(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to delete GPIO device")
		return
	}

	h.logger.WithField("device_id", id).Info("GPIO device deleted successfully")
	c.JSON(http.StatusNoContent, nil)
}

// Read reads the current value of a GPIO device
func (h *GPIOHandler) Read(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	device, err := h.service.Read(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to read GPIO device")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"device_id": device.ID,
			"value":     device.Value,
			"timestamp": device.UpdatedAt,
		},
	})
}

// Write writes a value to a GPIO device
func (h *GPIOHandler) Write(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	var req struct {
		Value int `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid GPIO write request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid Request",
			"message": err.Error(),
		})
		return
	}

	err = h.service.Write(uint(id), req.Value)
	if err != nil {
		h.handleServiceError(c, err, "Failed to write to GPIO device")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"device_id": id,
		"value":     req.Value,
	}).Info("GPIO device write successful")

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"device_id": id,
			"value":     req.Value,
		},
	})
}

// GetReadings returns GPIO readings for a device
func (h *GPIOHandler) GetReadings(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("id", idStr).Error("Invalid GPIO device ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ID",
			"message": "GPIO device ID must be a valid number",
		})
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := services.GPIOReadingFilter{
		DeviceID: uint(id),
		Limit:    limit,
		Offset:   offset,
	}

	readings, total, err := h.service.GetReadings(filter)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get GPIO readings")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  readings,
		"count": len(readings),
		"total": total,
	})
}

// handleServiceError handles service errors consistently
func (h *GPIOHandler) handleServiceError(c *gin.Context, err error, message string) {
	h.logger.WithError(err).Error(message)

	if services.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": message,
		})
		return
	}

	if services.IsAlreadyExists(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": message,
		})
		return
	}

	if services.IsValidationFailed(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation Failed",
			"message": message,
		})
		return
	}

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Internal Server Error",
		"message": message,
	})
}
