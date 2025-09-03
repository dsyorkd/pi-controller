package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGPIODevice_IsReserved(t *testing.T) {
	tests := []struct {
		name           string
		device         GPIODevice
		expectedResult bool
	}{
		{
			name:           "not reserved - all fields nil",
			device:         GPIODevice{},
			expectedResult: false,
		},
		{
			name: "not reserved - reserved_by nil",
			device: GPIODevice{
				ReservedAt: &time.Time{},
			},
			expectedResult: false,
		},
		{
			name: "not reserved - reserved_at nil",
			device: GPIODevice{
				ReservedBy: stringPtr("client1"),
			},
			expectedResult: false,
		},
		{
			name: "reserved - no expiry",
			device: GPIODevice{
				ReservedBy: stringPtr("client1"),
				ReservedAt: timePtr(time.Now()),
			},
			expectedResult: true,
		},
		{
			name: "reserved - not expired",
			device: GPIODevice{
				ReservedBy:     stringPtr("client1"),
				ReservedAt:     timePtr(time.Now()),
				ReservationTTL: timePtr(time.Now().Add(1 * time.Hour)),
			},
			expectedResult: true,
		},
		{
			name: "not reserved - expired",
			device: GPIODevice{
				ReservedBy:     stringPtr("client1"),
				ReservedAt:     timePtr(time.Now()),
				ReservationTTL: timePtr(time.Now().Add(-1 * time.Hour)),
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsReserved()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGPIODevice_IsReservedBy(t *testing.T) {
	clientID := "test-client"
	otherClientID := "other-client"

	tests := []struct {
		name           string
		device         GPIODevice
		clientID       string
		expectedResult bool
	}{
		{
			name:           "not reserved by anyone",
			device:         GPIODevice{},
			clientID:       clientID,
			expectedResult: false,
		},
		{
			name: "reserved by same client",
			device: GPIODevice{
				ReservedBy: &clientID,
				ReservedAt: timePtr(time.Now()),
			},
			clientID:       clientID,
			expectedResult: true,
		},
		{
			name: "reserved by different client",
			device: GPIODevice{
				ReservedBy: &otherClientID,
				ReservedAt: timePtr(time.Now()),
			},
			clientID:       clientID,
			expectedResult: false,
		},
		{
			name: "reserved by same client but expired",
			device: GPIODevice{
				ReservedBy:     &clientID,
				ReservedAt:     timePtr(time.Now()),
				ReservationTTL: timePtr(time.Now().Add(-1 * time.Hour)),
			},
			clientID:       clientID,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsReservedBy(tt.clientID)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGPIODevice_Reserve(t *testing.T) {
	device := GPIODevice{}
	clientID := "test-client"
	ttl := 1 * time.Hour

	// Test reservation without TTL
	device.Reserve(clientID, nil)

	assert.NotNil(t, device.ReservedBy)
	assert.Equal(t, clientID, *device.ReservedBy)
	assert.NotNil(t, device.ReservedAt)
	assert.Nil(t, device.ReservationTTL)
	assert.WithinDuration(t, time.Now(), device.UpdatedAt, 1*time.Second)

	// Test reservation with TTL
	device.Reserve(clientID, &ttl)

	assert.NotNil(t, device.ReservedBy)
	assert.Equal(t, clientID, *device.ReservedBy)
	assert.NotNil(t, device.ReservedAt)
	assert.NotNil(t, device.ReservationTTL)
	expectedExpiry := time.Now().Add(ttl)
	assert.WithinDuration(t, expectedExpiry, *device.ReservationTTL, 1*time.Second)
}

func TestGPIODevice_Release(t *testing.T) {
	clientID := "test-client"
	device := GPIODevice{
		ReservedBy:     &clientID,
		ReservedAt:     timePtr(time.Now()),
		ReservationTTL: timePtr(time.Now().Add(1 * time.Hour)),
	}

	// Ensure device is initially reserved
	assert.True(t, device.IsReserved())

	// Release the reservation
	device.Release()

	// Verify all reservation fields are cleared
	assert.Nil(t, device.ReservedBy)
	assert.Nil(t, device.ReservedAt)
	assert.Nil(t, device.ReservationTTL)
	assert.WithinDuration(t, time.Now(), device.UpdatedAt, 1*time.Second)

	// Verify device is no longer reserved
	assert.False(t, device.IsReserved())
}

func TestGPIODevice_IsReservationExpired(t *testing.T) {
	tests := []struct {
		name           string
		device         GPIODevice
		expectedResult bool
	}{
		{
			name:           "no expiry set",
			device:         GPIODevice{},
			expectedResult: false,
		},
		{
			name: "not expired",
			device: GPIODevice{
				ReservationTTL: timePtr(time.Now().Add(1 * time.Hour)),
			},
			expectedResult: false,
		},
		{
			name: "expired",
			device: GPIODevice{
				ReservationTTL: timePtr(time.Now().Add(-1 * time.Hour)),
			},
			expectedResult: true,
		},
		{
			name: "exactly expired",
			device: GPIODevice{
				ReservationTTL: timePtr(time.Now().Add(-1 * time.Millisecond)),
			},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.IsReservationExpired()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
