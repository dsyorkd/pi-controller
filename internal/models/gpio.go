package models

import (
	"time"

	"gorm.io/gorm"
)

// GPIODevice represents a GPIO device/pin configuration on a node
type GPIODevice struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	PinNumber   int            `json:"pin_number" gorm:"not null"`
	Direction   GPIODirection  `json:"direction" gorm:"default:'input'"`
	PullMode    GPIOPullMode   `json:"pull_mode" gorm:"default:'none'"`
	Value       int            `json:"value" gorm:"default:0"`
	DeviceType  GPIODeviceType `json:"device_type" gorm:"default:'digital'"`
	Status      GPIOStatus     `json:"status" gorm:"default:'active'"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Node relationship
	NodeID uint `json:"node_id" gorm:"not null"`
	Node   Node `json:"node,omitempty" gorm:"foreignKey:NodeID"`

	// Configuration
	Config GPIOConfig `json:"config" gorm:"embedded"`

	// Reservation tracking
	ReservedBy     *string    `json:"reserved_by,omitempty" gorm:"index"` // Optional client/user ID that reserved this pin
	ReservedAt     *time.Time `json:"reserved_at,omitempty"`              // When the pin was reserved
	ReservationTTL *time.Time `json:"reservation_ttl,omitempty"`          // When the reservation expires
}

// GPIODirection defines the direction of a GPIO pin
type GPIODirection string

const (
	GPIODirectionInput  GPIODirection = "input"
	GPIODirectionOutput GPIODirection = "output"
)

// GPIOPullMode defines the pull resistor configuration
type GPIOPullMode string

const (
	GPIOPullNone GPIOPullMode = "none"
	GPIOPullUp   GPIOPullMode = "up"
	GPIOPullDown GPIOPullMode = "down"
)

// GPIODeviceType defines the type of GPIO device
type GPIODeviceType string

const (
	GPIODeviceTypeDigital GPIODeviceType = "digital"
	GPIODeviceTypeAnalog  GPIODeviceType = "analog"
	GPIODeviceTypePWM     GPIODeviceType = "pwm"
	GPIODeviceTypeSPI     GPIODeviceType = "spi"
	GPIODeviceTypeI2C     GPIODeviceType = "i2c"
)

// GPIOStatus defines the status of a GPIO device
type GPIOStatus string

const (
	GPIOStatusActive   GPIOStatus = "active"
	GPIOStatusInactive GPIOStatus = "inactive"
	GPIOStatusError    GPIOStatus = "error"
)

// GPIOConfig holds device-specific configuration
type GPIOConfig struct {
	// PWM specific
	Frequency int `json:"frequency,omitempty"`  // Hz
	DutyCycle int `json:"duty_cycle,omitempty"` // 0-100

	// SPI specific
	SPIMode    int `json:"spi_mode,omitempty"`    // 0-3
	SPIBits    int `json:"spi_bits,omitempty"`    // bits per word
	SPISpeed   int `json:"spi_speed,omitempty"`   // Hz
	SPIChannel int `json:"spi_channel,omitempty"` // 0 or 1

	// I2C specific
	I2CAddress int `json:"i2c_address,omitempty"` // 7-bit address
	I2CBus     int `json:"i2c_bus,omitempty"`     // bus number

	// Sampling configuration
	SampleRate int `json:"sample_rate,omitempty"` // samples per second
}

// IsOutput returns true if the GPIO is configured as output
func (g *GPIODevice) IsOutput() bool {
	return g.Direction == GPIODirectionOutput
}

// IsInput returns true if the GPIO is configured as input
func (g *GPIODevice) IsInput() bool {
	return g.Direction == GPIODirectionInput
}

// IsActive returns true if the GPIO device is active
func (g *GPIODevice) IsActive() bool {
	return g.Status == GPIOStatusActive
}

// SetValue sets the GPIO value and updates timestamp
func (g *GPIODevice) SetValue(value int) {
	g.Value = value
	g.UpdatedAt = time.Now()
}

// IsReserved returns true if the GPIO pin is currently reserved
func (g *GPIODevice) IsReserved() bool {
	return g.ReservedBy != nil && g.ReservedAt != nil &&
		(g.ReservationTTL == nil || g.ReservationTTL.After(time.Now()))
}

// IsReservedBy returns true if the GPIO pin is reserved by the specified client
func (g *GPIODevice) IsReservedBy(clientID string) bool {
	return g.IsReserved() && g.ReservedBy != nil && *g.ReservedBy == clientID
}

// Reserve reserves the GPIO pin for a specific client with optional TTL
func (g *GPIODevice) Reserve(clientID string, ttl *time.Duration) {
	now := time.Now()
	g.ReservedBy = &clientID
	g.ReservedAt = &now
	g.UpdatedAt = now

	if ttl != nil {
		expiryTime := now.Add(*ttl)
		g.ReservationTTL = &expiryTime
	} else {
		g.ReservationTTL = nil // No expiry
	}
}

// Release removes the reservation from the GPIO pin
func (g *GPIODevice) Release() {
	g.ReservedBy = nil
	g.ReservedAt = nil
	g.ReservationTTL = nil
	g.UpdatedAt = time.Now()
}

// IsReservationExpired returns true if the reservation has expired
func (g *GPIODevice) IsReservationExpired() bool {
	return g.ReservationTTL != nil && g.ReservationTTL.Before(time.Now())
}

// TableName returns the table name for the GPIODevice model
func (GPIODevice) TableName() string {
	return "gpio_devices"
}

// GPIOReading represents a time-series reading from a GPIO device
type GPIOReading struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	DeviceID  uint      `json:"device_id" gorm:"not null"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`

	// Relationships
	Device GPIODevice `json:"device,omitempty" gorm:"foreignKey:DeviceID"`
}

// TableName returns the table name for the GPIOReading model
func (GPIOReading) TableName() string {
	return "gpio_readings"
}
