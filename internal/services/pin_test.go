package services

import (
	"errors"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestPinService_GetPin(t *testing.T) {
	t.Run("should get a pin successfully", func(t *testing.T) {
		store := &MockStore{}
		service := NewPinService(store)

		pin := &models.GPIODevice{
			ID:   1,
			Name: "test-pin",
		}

		store.On("GetPin", uint(1)).Return(pin, nil)

		retrievedPin, err := service.GetPin(1)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPin)
		assert.Equal(t, "test-pin", retrievedPin.Name)

		store.AssertExpectations(t)
	})

	t.Run("should return an error when the store fails", func(t *testing.T) {
		store := &MockStore{}
		service := NewPinService(store)

		store.On("GetPin", uint(1)).Return(nil, errors.New("store error"))

		_, err := service.GetPin(1)
		assert.Error(t, err)

		store.AssertExpectations(t)
	})
}

func TestPinService_GetPins(t *testing.T) {
	t.Run("should get all pins successfully", func(t *testing.T) {
		store := &MockStore{}
		service := NewPinService(store)

		pins := []models.GPIODevice{
			{ID: 1, Name: "test-pin-1"},
			{ID: 2, Name: "test-pin-2"},
		}

		store.On("GetPins").Return(pins, nil)

		retrievedPins, err := service.GetPins()
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPins)
		assert.Len(t, retrievedPins, 2)

		store.AssertExpectations(t)
	})

	t.Run("should return an error when the store fails", func(t *testing.T) {
		store := &MockStore{}
		service := NewPinService(store)

		store.On("GetPins").Return(nil, errors.New("store error"))

		_, err := service.GetPins()
		assert.Error(t, err)

		store.AssertExpectations(t)
	})
}

func TestPinService_UpdatePin(t *testing.T) {
	t.Run("should update a pin successfully", func(t *testing.T) {
		store := &MockStore{}
		service := NewPinService(store)

		pin := &models.GPIODevice{
			ID:   1,
			Name: "test-pin",
		}

		store.On("UpdatePin", pin).Return(nil)

		err := service.UpdatePin(pin)
		assert.NoError(t, err)

		store.AssertExpectations(t)
	})

	t.Run("should return an error when the store fails", func(t *testing.T) {
		store := &MockStore{}
		service := NewPinService(store)

		pin := &models.GPIODevice{
			ID:   1,
			Name: "test-pin",
		}

		store.On("UpdatePin", pin).Return(errors.New("store error"))

		err := service.UpdatePin(pin)
		assert.Error(t, err)

		store.AssertExpectations(t)
	})
}
