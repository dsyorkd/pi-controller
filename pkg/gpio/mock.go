package gpio

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockGPIO provides a mock implementation of the GPIO interface for testing and development
type MockGPIO struct {
	mu               sync.RWMutex
	pins             map[int]*mockPin
	eventHandlers    map[int]EventHandler
	eventTypes       map[int]EventType
	eventLoopRunning bool
	eventLoopCancel  context.CancelFunc
}

type mockPin struct {
	config    PinConfig
	value     PinValue
	timestamp time.Time
}

// NewMockGPIO creates a new mock GPIO interface
func NewMockGPIO(config *Config) *MockGPIO {
	return &MockGPIO{
		pins:          make(map[int]*mockPin),
		eventHandlers: make(map[int]EventHandler),
		eventTypes:    make(map[int]EventType),
	}
}

// Initialize initializes the mock GPIO interface
func (m *MockGPIO) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear any existing state
	m.pins = make(map[int]*mockPin)
	m.eventHandlers = make(map[int]EventHandler)
	m.eventTypes = make(map[int]EventType)

	return nil
}

// Close closes the mock GPIO interface
func (m *MockGPIO) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop event loop if running
	if m.eventLoopRunning && m.eventLoopCancel != nil {
		m.eventLoopCancel()
	}

	// Clear state
	m.pins = make(map[int]*mockPin)
	m.eventHandlers = make(map[int]EventHandler)
	m.eventTypes = make(map[int]EventType)

	return nil
}

// ConfigurePin configures a GPIO pin
func (m *MockGPIO) ConfigurePin(config PinConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Pin < 0 || config.Pin > 40 {
		return fmt.Errorf("invalid pin number: %d", config.Pin)
	}

	m.pins[config.Pin] = &mockPin{
		config:    config,
		value:     Low,
		timestamp: time.Now(),
	}

	return nil
}

// ReadPin reads the current value of a GPIO pin
func (m *MockGPIO) ReadPin(pin int) (PinValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mockPin, exists := m.pins[pin]
	if !exists {
		return Low, fmt.Errorf("pin %d not configured", pin)
	}

	if mockPin.config.Direction != DirectionInput {
		return Low, fmt.Errorf("pin %d is not configured as input", pin)
	}

	// Update timestamp
	mockPin.timestamp = time.Now()

	return mockPin.value, nil
}

// WritePin writes a value to a GPIO pin
func (m *MockGPIO) WritePin(pin int, value PinValue) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mockPin, exists := m.pins[pin]
	if !exists {
		return fmt.Errorf("pin %d not configured", pin)
	}

	if mockPin.config.Direction != DirectionOutput {
		return fmt.Errorf("pin %d is not configured as output", pin)
	}

	oldValue := mockPin.value
	mockPin.value = value
	mockPin.timestamp = time.Now()

	// Trigger event if there's a handler and value changed
	if handler, hasHandler := m.eventHandlers[pin]; hasHandler && oldValue != value {
		eventType := m.eventTypes[pin]
		shouldTrigger := false

		switch eventType {
		case EventRisingEdge:
			shouldTrigger = oldValue == Low && value == High
		case EventFallingEdge:
			shouldTrigger = oldValue == High && value == Low
		case EventBothEdges:
			shouldTrigger = true
		}

		if shouldTrigger {
			go handler(Event{
				Pin:       pin,
				Type:      eventType,
				Value:     value,
				Timestamp: mockPin.timestamp,
			})
		}
	}

	return nil
}

// GetPinState returns the current state of a GPIO pin
func (m *MockGPIO) GetPinState(pin int) (*PinState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mockPin, exists := m.pins[pin]
	if !exists {
		return nil, fmt.Errorf("pin %d not configured", pin)
	}

	return &PinState{
		Pin:       pin,
		Direction: mockPin.config.Direction,
		Value:     mockPin.value,
		PullMode:  mockPin.config.PullMode,
		Timestamp: mockPin.timestamp,
	}, nil
}

// ListConfiguredPins returns a list of all configured GPIO pins
func (m *MockGPIO) ListConfiguredPins() ([]PinState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]PinState, 0, len(m.pins))
	for pin, mockPin := range m.pins {
		states = append(states, PinState{
			Pin:       pin,
			Direction: mockPin.config.Direction,
			Value:     mockPin.value,
			PullMode:  mockPin.config.PullMode,
			Timestamp: mockPin.timestamp,
		})
	}

	return states, nil
}

// SetPWM configures PWM on a pin
func (m *MockGPIO) SetPWM(pin int, frequency int, dutyCycle int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mockPin, exists := m.pins[pin]
	if !exists {
		return fmt.Errorf("pin %d not configured", pin)
	}

	if mockPin.config.Direction != DirectionOutput {
		return fmt.Errorf("pin %d is not configured as output", pin)
	}

	if frequency < 1 || frequency > 10000 {
		return fmt.Errorf("invalid PWM frequency: %d", frequency)
	}

	if dutyCycle < 0 || dutyCycle > 100 {
		return fmt.Errorf("invalid PWM duty cycle: %d", dutyCycle)
	}

	// Update config
	mockPin.config.PWMFrequency = frequency
	mockPin.config.PWMDutyCycle = dutyCycle
	mockPin.timestamp = time.Now()

	return nil
}

// ReadAnalog reads an analog value (mock implementation returns a random value)
func (m *MockGPIO) ReadAnalog(pin int) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mockPin, exists := m.pins[pin]
	if !exists {
		return 0, fmt.Errorf("pin %d not configured", pin)
	}

	if mockPin.config.Direction != DirectionInput {
		return 0, fmt.Errorf("pin %d is not configured as input", pin)
	}

	// Return a mock analog value based on current timestamp
	// This provides predictable but varying values for testing
	now := time.Now()
	mockValue := float64(now.Second()%100) / 100.0 // 0.0 to 0.99

	mockPin.timestamp = now

	return mockValue, nil
}

// IsAvailable returns whether GPIO hardware is available (always true for mock)
func (m *MockGPIO) IsAvailable() bool {
	return true
}

// SPITransfer performs a mock SPI transfer
func (m *MockGPIO) SPITransfer(channel int, data []byte) ([]byte, error) {
	if channel < 0 || channel > 1 {
		return nil, fmt.Errorf("invalid SPI channel: %d", channel)
	}

	// Mock implementation echoes the input data with bit inversion
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = ^b // Invert bits for mock response
	}

	return result, nil
}

// SPIWrite writes data to SPI
func (m *MockGPIO) SPIWrite(channel int, data []byte) error {
	if channel < 0 || channel > 1 {
		return fmt.Errorf("invalid SPI channel: %d", channel)
	}

	// Mock implementation does nothing
	return nil
}

// SPIRead reads data from SPI
func (m *MockGPIO) SPIRead(channel int, length int) ([]byte, error) {
	if channel < 0 || channel > 1 {
		return nil, fmt.Errorf("invalid SPI channel: %d", channel)
	}

	if length <= 0 {
		return nil, fmt.Errorf("invalid read length: %d", length)
	}

	// Mock implementation returns incrementing bytes
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = byte(i % 256)
	}

	return result, nil
}

// I2CWrite writes data to an I2C device
func (m *MockGPIO) I2CWrite(bus int, address int, data []byte) error {
	if bus < 0 || bus > 1 {
		return fmt.Errorf("invalid I2C bus: %d", bus)
	}

	if address < 0x08 || address > 0x77 {
		return fmt.Errorf("invalid I2C address: 0x%02X", address)
	}

	// Mock implementation does nothing
	return nil
}

// I2CRead reads data from an I2C device
func (m *MockGPIO) I2CRead(bus int, address int, length int) ([]byte, error) {
	if bus < 0 || bus > 1 {
		return nil, fmt.Errorf("invalid I2C bus: %d", bus)
	}

	if address < 0x08 || address > 0x77 {
		return nil, fmt.Errorf("invalid I2C address: 0x%02X", address)
	}

	if length <= 0 {
		return nil, fmt.Errorf("invalid read length: %d", length)
	}

	// Mock implementation returns address-based pattern
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = byte((address + i) % 256)
	}

	return result, nil
}

// I2CWriteRegister writes data to a specific register on an I2C device
func (m *MockGPIO) I2CWriteRegister(bus int, address int, register int, data []byte) error {
	if bus < 0 || bus > 1 {
		return fmt.Errorf("invalid I2C bus: %d", bus)
	}

	if address < 0x08 || address > 0x77 {
		return fmt.Errorf("invalid I2C address: 0x%02X", address)
	}

	if register < 0 || register > 255 {
		return fmt.Errorf("invalid I2C register: 0x%02X", register)
	}

	// Mock implementation does nothing
	return nil
}

// I2CReadRegister reads data from a specific register on an I2C device
func (m *MockGPIO) I2CReadRegister(bus int, address int, register int, length int) ([]byte, error) {
	if bus < 0 || bus > 1 {
		return nil, fmt.Errorf("invalid I2C bus: %d", bus)
	}

	if address < 0x08 || address > 0x77 {
		return nil, fmt.Errorf("invalid I2C address: 0x%02X", address)
	}

	if register < 0 || register > 255 {
		return nil, fmt.Errorf("invalid I2C register: 0x%02X", register)
	}

	if length <= 0 {
		return nil, fmt.Errorf("invalid read length: %d", length)
	}

	// Mock implementation returns register-based pattern
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = byte((register + i) % 256)
	}

	return result, nil
}

// EnableInterrupt enables interrupt-based event detection on a pin
func (m *MockGPIO) EnableInterrupt(pin int, eventType EventType, handler EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pins[pin]; !exists {
		return fmt.Errorf("pin %d not configured", pin)
	}

	m.eventHandlers[pin] = handler
	m.eventTypes[pin] = eventType

	return nil
}

// DisableInterrupt disables interrupt-based event detection on a pin
func (m *MockGPIO) DisableInterrupt(pin int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.eventHandlers, pin)
	delete(m.eventTypes, pin)

	return nil
}

// StartEventLoop starts the event processing loop
func (m *MockGPIO) StartEventLoop(ctx context.Context) error {
	m.mu.Lock()
	if m.eventLoopRunning {
		m.mu.Unlock()
		return fmt.Errorf("event loop is already running")
	}

	ctx, cancel := context.WithCancel(ctx)
	m.eventLoopCancel = cancel
	m.eventLoopRunning = true
	m.mu.Unlock()

	// Mock event loop that generates random events for testing
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.mu.Lock()
				m.eventLoopRunning = false
				m.eventLoopCancel = nil
				m.mu.Unlock()
				return
			case <-ticker.C:
				m.generateMockEvents()
			}
		}
	}()

	return nil
}

// StopEventLoop stops the event processing loop
func (m *MockGPIO) StopEventLoop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.eventLoopRunning {
		return fmt.Errorf("event loop is not running")
	}

	if m.eventLoopCancel != nil {
		m.eventLoopCancel()
	}

	// Set running to false immediately to prevent race conditions
	m.eventLoopRunning = false
	m.eventLoopCancel = nil

	return nil
}

// generateMockEvents generates random events for testing
func (m *MockGPIO) generateMockEvents() {
	m.mu.RLock()
	handlers := make(map[int]EventHandler)
	eventTypes := make(map[int]EventType)
	for pin, handler := range m.eventHandlers {
		handlers[pin] = handler
		eventTypes[pin] = m.eventTypes[pin]
	}
	m.mu.RUnlock()

	// Generate mock events for pins with handlers
	for pin, handler := range handlers {
		eventType := eventTypes[pin]

		// Simulate random events (50% chance per cycle)
		if time.Now().UnixNano()%2 == 0 {
			value := PinValue(time.Now().UnixNano() % 2)

			event := Event{
				Pin:       pin,
				Type:      eventType,
				Value:     value,
				Timestamp: time.Now(),
			}

			go handler(event)
		}
	}
}
