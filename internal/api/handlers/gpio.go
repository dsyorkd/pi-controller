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
