// Package gpio provides GPIO hardware abstraction layer for Raspberry Pi devices
package gpio

import (
	"context"
	"time"
)

// PinDirection represents the direction of a GPIO pin
type PinDirection string

const (
	DirectionInput  PinDirection = "input"
	DirectionOutput PinDirection = "output"
)

// PullMode represents the pull resistor configuration
type PullMode string

const (
	PullNone PullMode = "none"
	PullUp   PullMode = "up"
	PullDown PullMode = "down"
)

// PinValue represents the logical state of a GPIO pin
type PinValue int

const (
	Low  PinValue = 0
	High PinValue = 1
)

// PinConfig represents the configuration for a GPIO pin
type PinConfig struct {
	Pin       int          `json:"pin"`
	Direction PinDirection `json:"direction"`
	PullMode  PullMode     `json:"pull_mode"`
	// PWM specific
	PWMFrequency int `json:"pwm_frequency,omitempty"`
	PWMDutyCycle int `json:"pwm_duty_cycle,omitempty"`
	// SPI specific
	SPIChannel int `json:"spi_channel,omitempty"`
	SPIMode    int `json:"spi_mode,omitempty"`
	SPISpeed   int `json:"spi_speed,omitempty"`
	SPIBits    int `json:"spi_bits,omitempty"`
	// I2C specific
	I2CAddress int `json:"i2c_address,omitempty"`
	I2CBus     int `json:"i2c_bus,omitempty"`
}

// PinState represents the current state of a GPIO pin
type PinState struct {
	Pin       int          `json:"pin"`
	Direction PinDirection `json:"direction"`
	Value     PinValue     `json:"value"`
	PullMode  PullMode     `json:"pull_mode"`
	Timestamp time.Time    `json:"timestamp"`
}

// Interface defines the GPIO hardware interface
type Interface interface {
	// Initialize initializes the GPIO interface
	Initialize(ctx context.Context) error

	// Close closes the GPIO interface and cleans up resources
	Close() error

	// ConfigurePin configures a GPIO pin with the given configuration
	ConfigurePin(config PinConfig) error

	// ReadPin reads the current value of a GPIO pin
	ReadPin(pin int) (PinValue, error)

	// WritePin writes a value to a GPIO pin
	WritePin(pin int, value PinValue) error

	// GetPinState returns the current state of a GPIO pin
	GetPinState(pin int) (*PinState, error)

	// ListConfiguredPins returns a list of all configured GPIO pins
	ListConfiguredPins() ([]PinState, error)

	// SetPWM configures PWM on a pin
	SetPWM(pin int, frequency int, dutyCycle int) error

	// ReadAnalog reads an analog value (for devices with ADC)
	ReadAnalog(pin int) (float64, error)

	// IsAvailable returns whether GPIO hardware is available
	IsAvailable() bool
}

// SPIInterface defines the SPI interface
type SPIInterface interface {
	// SPITransfer performs a full-duplex SPI transfer
	SPITransfer(channel int, data []byte) ([]byte, error)

	// SPIWrite writes data to SPI
	SPIWrite(channel int, data []byte) error

	// SPIRead reads data from SPI
	SPIRead(channel int, length int) ([]byte, error)
}

// I2CInterface defines the I2C interface
type I2CInterface interface {
	// I2CWrite writes data to an I2C device
	I2CWrite(bus int, address int, data []byte) error

	// I2CRead reads data from an I2C device
	I2CRead(bus int, address int, length int) ([]byte, error)

	// I2CWriteRegister writes data to a specific register on an I2C device
	I2CWriteRegister(bus int, address int, register int, data []byte) error

	// I2CReadRegister reads data from a specific register on an I2C device
	I2CReadRegister(bus int, address int, register int, length int) ([]byte, error)
}

// EventType represents the type of GPIO event
type EventType string

const (
	EventRisingEdge  EventType = "rising"
	EventFallingEdge EventType = "falling"
	EventBothEdges   EventType = "both"
)

// Event represents a GPIO pin event
type Event struct {
	Pin       int       `json:"pin"`
	Type      EventType `json:"type"`
	Value     PinValue  `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// EventHandler is a function type for handling GPIO events
type EventHandler func(event Event)

// EventInterface defines the GPIO event interface
type EventInterface interface {
	// EnableInterrupt enables interrupt-based event detection on a pin
	EnableInterrupt(pin int, eventType EventType, handler EventHandler) error

	// DisableInterrupt disables interrupt-based event detection on a pin
	DisableInterrupt(pin int) error

	// StartEventLoop starts the event processing loop
	StartEventLoop(ctx context.Context) error

	// StopEventLoop stops the event processing loop
	StopEventLoop() error
}

// FullInterface combines all GPIO interfaces
type FullInterface interface {
	Interface
	SPIInterface
	I2CInterface
	EventInterface
}

// Config represents the GPIO controller configuration
type Config struct {
	MockMode        bool     `yaml:"mock_mode" mapstructure:"mock_mode"`
	AllowedPins     []int    `yaml:"allowed_pins" mapstructure:"allowed_pins"`
	RestrictedPins  []int    `yaml:"restricted_pins" mapstructure:"restricted_pins"`
	DefaultPullMode PullMode `yaml:"default_pull_mode" mapstructure:"default_pull_mode"`
}

// DefaultConfig returns a default GPIO configuration
func DefaultConfig() *Config {
	return &Config{
		MockMode: false,
		AllowedPins: []int{
			// Standard GPIO pins on Raspberry Pi
			2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27,
		},
		RestrictedPins: []int{
			// Pins typically reserved for system use
			0, 1, // I2C
		},
		DefaultPullMode: PullNone,
	}
}
