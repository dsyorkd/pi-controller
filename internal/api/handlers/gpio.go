package handlers

import (
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