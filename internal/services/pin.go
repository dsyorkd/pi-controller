package services

import (
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// PinService is the service for managing pins
type PinService struct {
	store storage.Store
}

// NewPinService creates a new PinService
func NewPinService(store storage.Store) *PinService {
	return &PinService{
		store: store,
	}
}

// GetPin retrieves a pin by ID
func (s *PinService) GetPin(id uint) (*models.GPIODevice, error) {
	return s.store.GetPin(id)
}

// GetPins retrieves all pins
func (s *PinService) GetPins() ([]models.GPIODevice, error) {
	return s.store.GetPins()
}

// UpdatePin updates a pin
func (s *PinService) UpdatePin(pin *models.GPIODevice) error {
	return s.store.UpdatePin(pin)
}
