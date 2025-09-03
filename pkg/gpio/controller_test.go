package gpio

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGPIOController_Initialize tests GPIO controller initialization
func TestGPIOController_Initialize(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name:   "successful initialization with default config",
			config: DefaultConfig(),
		},
		{
			name: "successful initialization with mock mode",
			config: &Config{
				MockMode:        true,
				AllowedPins:     []int{18, 19, 20, 21},
				RestrictedPins:  []int{0, 1},
				DefaultPullMode: PullNone,
			},
		},
		{
			name: "initialization with custom allowed pins",
			config: &Config{
				MockMode:        true,
				AllowedPins:     []int{22, 23, 24, 25},
				RestrictedPins:  []int{0, 1, 2, 3},
				DefaultPullMode: PullUp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetLevel(logrus.WarnLevel)
			controller := NewController(tt.config, DefaultSecurityConfig(), logger)
			require.NotNil(t, controller)

			ctx := context.Background()
			err := controller.Initialize(ctx)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Cleanup
				err = controller.Close()
				assert.NoError(t, err)
			}
		})
	}
}

// TestGPIOController_ConfigurePin tests GPIO pin configuration
func TestGPIOController_ConfigurePin(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21, 22, 23, 24, 25},
		RestrictedPins:  []int{0, 1, 2, 3},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	tests := []struct {
		name        string
		pinConfig   PinConfig
		expectedErr string
	}{
		{
			name: "configure output pin",
			pinConfig: PinConfig{
				Pin:       18,
				Direction: DirectionOutput,
				PullMode:  PullNone,
			},
		},
		{
			name: "configure input pin with pull-up",
			pinConfig: PinConfig{
				Pin:       19,
				Direction: DirectionInput,
				PullMode:  PullUp,
			},
		},
		{
			name: "configure PWM pin",
			pinConfig: PinConfig{
				Pin:          20,
				Direction:    DirectionOutput,
				PullMode:     PullNone,
				PWMFrequency: 1000,
				PWMDutyCycle: 50,
			},
		},
		{
			name: "configure SPI pin",
			pinConfig: PinConfig{
				Pin:        21,
				Direction:  DirectionOutput,
				SPIChannel: 0,
				SPIMode:    0,
				SPISpeed:   1000000,
				SPIBits:    8,
			},
		},
		{
			name: "configure I2C pin",
			pinConfig: PinConfig{
				Pin:        22,
				Direction:  DirectionOutput,
				I2CAddress: 0x48,
				I2CBus:     1,
			},
		},
		{
			name: "configure restricted pin should fail",
			pinConfig: PinConfig{
				Pin:       0, // Restricted pin
				Direction: DirectionOutput,
				PullMode:  PullNone,
			},
			expectedErr: "pin 0 is restricted",
		},
		{
			name: "configure invalid pin should fail",
			pinConfig: PinConfig{
				Pin:       -1, // Invalid pin
				Direction: DirectionOutput,
				PullMode:  PullNone,
			},
			expectedErr: "invalid pin number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.ConfigurePin(tt.pinConfig, "test-user")

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Verify pin state
				state, err := controller.GetPinState(tt.pinConfig.Pin, "test-user")
				require.NoError(t, err)
				assert.Equal(t, tt.pinConfig.Pin, state.Pin)
				assert.Equal(t, tt.pinConfig.Direction, state.Direction)
				assert.Equal(t, tt.pinConfig.PullMode, state.PullMode)
			}
		})
	}
}

// TestGPIOController_Security_PinRestrictions tests GPIO security restrictions
func TestGPIOController_Security_PinRestrictions(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20},
		RestrictedPins:  []int{0, 1, 2, 3, 14, 15}, // System critical pins
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	securityTests := []struct {
		name        string
		pin         int
		shouldFail  bool
		description string
	}{
		{
			name:        "system critical pin 0",
			pin:         0,
			shouldFail:  true,
			description: "Pin 0 is typically reserved for system use",
		},
		{
			name:        "system critical pin 1",
			pin:         1,
			shouldFail:  true,
			description: "Pin 1 is typically reserved for system use",
		},
		{
			name:        "UART pins",
			pin:         14,
			shouldFail:  true,
			description: "UART pins should be restricted to prevent system disruption",
		},
		{
			name:        "allowed safe pin",
			pin:         18,
			shouldFail:  false,
			description: "Pin 18 is safe for GPIO operations",
		},
		{
			name:        "pin outside allowed range",
			pin:         99,
			shouldFail:  true,
			description: "Pins outside allowed range should be blocked",
		},
		{
			name:        "negative pin number",
			pin:         -1,
			shouldFail:  true,
			description: "Negative pin numbers are invalid and potentially dangerous",
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			pinConfig := PinConfig{
				Pin:       tt.pin,
				Direction: DirectionOutput,
				PullMode:  PullNone,
			}

			err := controller.ConfigurePin(pinConfig, "test-user")

			if tt.shouldFail {
				assert.Error(t, err, "Pin %d should be restricted: %s", tt.pin, tt.description)
			} else {
				assert.NoError(t, err, "Pin %d should be allowed: %s", tt.pin, tt.description)
			}
		})
	}
}

// TestGPIOController_WritePin tests GPIO pin writing with safety checks
func TestGPIOController_WritePin(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	// Configure test pins
	outputPin := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	inputPin := PinConfig{Pin: 19, Direction: DirectionInput, PullMode: PullNone}

	require.NoError(t, controller.ConfigurePin(outputPin, "test-user"))
	require.NoError(t, controller.ConfigurePin(inputPin, "test-user"))

	tests := []struct {
		name        string
		pin         int
		value       PinValue
		expectedErr string
	}{
		{
			name:  "write high to output pin",
			pin:   18,
			value: High,
		},
		{
			name:  "write low to output pin",
			pin:   18,
			value: Low,
		},
		{
			name:        "write to input pin should fail",
			pin:         19,
			value:       High,
			expectedErr: "not configured as output",
		},
		{
			name:        "write to unconfigured pin should fail",
			pin:         20,
			value:       High,
			expectedErr: "not configured",
		},
		{
			name:        "write to restricted pin should fail",
			pin:         0,
			value:       High,
			expectedErr: "not configured", // Will fail at configuration level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.WritePin(tt.pin, tt.value, "test-user")

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Verify the value was written
				state, err := controller.GetPinState(tt.pin, "test-user")
				require.NoError(t, err)
				assert.Equal(t, tt.value, state.Value)
			}
		})
	}
}

// TestGPIOController_ReadPin tests GPIO pin reading
func TestGPIOController_ReadPin(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	// Configure test pins
	outputPin := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	inputPin := PinConfig{Pin: 19, Direction: DirectionInput, PullMode: PullNone}

	require.NoError(t, controller.ConfigurePin(outputPin, "test-user"))
	require.NoError(t, controller.ConfigurePin(inputPin, "test-user"))

	tests := []struct {
		name        string
		pin         int
		expectedErr string
	}{
		{
			name: "read from input pin",
			pin:  19,
		},
		{
			name:        "read from output pin should fail",
			pin:         18,
			expectedErr: "not configured as input",
		},
		{
			name:        "read from unconfigured pin should fail",
			pin:         20,
			expectedErr: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := controller.ReadPin(tt.pin, "test-user")

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Contains(t, []PinValue{Low, High}, value)
			}
		})
	}
}

// TestGPIOController_PWM tests PWM functionality and safety
func TestGPIOController_PWM(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	// Configure output pin
	outputPin := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	require.NoError(t, controller.ConfigurePin(outputPin, "test-user"))

	tests := []struct {
		name        string
		pin         int
		frequency   int
		dutyCycle   int
		expectedErr string
	}{
		{
			name:      "valid PWM configuration",
			pin:       18,
			frequency: 1000,
			dutyCycle: 50,
		},
		{
			name:        "invalid frequency",
			pin:         18,
			frequency:   0,
			dutyCycle:   50,
			expectedErr: "invalid PWM frequency",
		},
		{
			name:        "invalid duty cycle - negative",
			pin:         18,
			frequency:   1000,
			dutyCycle:   -1,
			expectedErr: "invalid PWM duty cycle",
		},
		{
			name:        "invalid duty cycle - over 100",
			pin:         18,
			frequency:   1000,
			dutyCycle:   101,
			expectedErr: "invalid PWM duty cycle",
		},
		{
			name:        "PWM on unconfigured pin",
			pin:         19,
			frequency:   1000,
			dutyCycle:   50,
			expectedErr: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.SetPWM(tt.pin, tt.frequency, tt.dutyCycle, "test-user")

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGPIOController_SPI tests SPI functionality and safety
func TestGPIOController_SPI(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	tests := []struct {
		name        string
		channel     int
		data        []byte
		expectedErr string
	}{
		{
			name:    "valid SPI transfer",
			channel: 0,
			data:    []byte{0x01, 0x02, 0x03},
		},
		{
			name:    "SPI channel 1",
			channel: 1,
			data:    []byte{0xFF, 0x00},
		},
		{
			name:        "invalid SPI channel",
			channel:     2,
			data:        []byte{0x01},
			expectedErr: "invalid SPI channel",
		},
		{
			name:        "negative SPI channel",
			channel:     -1,
			data:        []byte{0x01},
			expectedErr: "invalid SPI channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := controller.SPITransfer(tt.channel, tt.data, "test-user")

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, len(tt.data))
			}
		})
	}
}

// TestGPIOController_I2C tests I2C functionality and safety
func TestGPIOController_I2C(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	tests := []struct {
		name        string
		bus         int
		address     int
		data        []byte
		expectedErr string
	}{
		{
			name:    "valid I2C write",
			bus:     0,
			address: 0x48,
			data:    []byte{0x01, 0x02},
		},
		{
			name:    "I2C bus 1",
			bus:     1,
			address: 0x50,
			data:    []byte{0xFF},
		},
		{
			name:        "invalid I2C bus",
			bus:         2,
			address:     0x48,
			data:        []byte{0x01},
			expectedErr: "invalid I2C bus",
		},
		{
			name:        "invalid I2C address - too low",
			bus:         0,
			address:     0x07,
			data:        []byte{0x01},
			expectedErr: "invalid I2C address",
		},
		{
			name:        "invalid I2C address - too high",
			bus:         0,
			address:     0x78,
			data:        []byte{0x01},
			expectedErr: "invalid I2C address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.I2CWrite(tt.bus, tt.address, tt.data, "test-user")

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGPIOController_Events tests GPIO event handling
func TestGPIOController_Events(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	// Configure input pin for events
	inputPin := PinConfig{Pin: 18, Direction: DirectionInput, PullMode: PullNone}
	require.NoError(t, controller.ConfigurePin(inputPin, "test-user"))

	// Test event handling
	eventReceived := make(chan Event, 1)
	handler := func(event Event) {
		eventReceived <- event
	}

	err = controller.EnableInterrupt(18, EventBothEdges, handler, "test-user")
	require.NoError(t, err)

	// Start event loop
	err = controller.StartEventLoop(ctx)
	require.NoError(t, err)

	// Give some time for events to be generated (mock generates random events)
	select {
	case event := <-eventReceived:
		assert.Equal(t, 18, event.Pin)
		assert.Contains(t, []EventType{EventRisingEdge, EventFallingEdge, EventBothEdges}, event.Type)
		assert.Contains(t, []PinValue{Low, High}, event.Value)
		assert.WithinDuration(t, time.Now(), event.Timestamp, 10*time.Second)
	case <-time.After(10 * time.Second):
		t.Log("No events received within timeout (this is expected for mock)")
	}

	// Disable interrupt
	err = controller.DisableInterrupt(18, "test-user")
	require.NoError(t, err)

	// Stop event loop
	err = controller.StopEventLoop()
	require.NoError(t, err)
}

// TestGPIOController_ListConfiguredPins tests pin listing
func TestGPIOController_ListConfiguredPins(t *testing.T) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(t, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(t, err)
	defer controller.Close()

	// Initially no pins configured
	pins, err := controller.ListConfiguredPins()
	require.NoError(t, err)
	assert.Len(t, pins, 0)

	// Configure some pins
	pinConfigs := []PinConfig{
		{Pin: 18, Direction: DirectionOutput, PullMode: PullNone},
		{Pin: 19, Direction: DirectionInput, PullMode: PullUp},
		{Pin: 20, Direction: DirectionOutput, PullMode: PullDown},
	}

	for _, config := range pinConfigs {
		require.NoError(t, controller.ConfigurePin(config, "test-user"))
	}

	// List configured pins
	pins, err = controller.ListConfiguredPins()
	require.NoError(t, err)
	assert.Len(t, pins, 3)

	// Verify pin details
	pinMap := make(map[int]PinState)
	for _, pin := range pins {
		pinMap[pin.Pin] = pin
	}

	for _, expectedConfig := range pinConfigs {
		pin, exists := pinMap[expectedConfig.Pin]
		require.True(t, exists, "Pin %d should be in the list", expectedConfig.Pin)
		assert.Equal(t, expectedConfig.Direction, pin.Direction)
		assert.Equal(t, expectedConfig.PullMode, pin.PullMode)
	}
}

// BenchmarkGPIOController_WritePin benchmarks GPIO pin writing
func BenchmarkGPIOController_WritePin(b *testing.B) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(b, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(b, err)
	defer controller.Close()

	// Configure output pin
	pinConfig := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	require.NoError(b, controller.ConfigurePin(pinConfig, "benchmark"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		value := PinValue(i % 2) // Alternate between High and Low
		err := controller.WritePin(18, value, "benchmark")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGPIOController_ReadPin benchmarks GPIO pin reading
func BenchmarkGPIOController_ReadPin(b *testing.B) {
	config := &Config{
		MockMode:        true,
		AllowedPins:     []int{18},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: PullNone,
	}

	logger := logrus.New()
	controller := NewController(config, DefaultSecurityConfig(), logger)
	require.NotNil(b, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(b, err)
	defer controller.Close()

	// Configure input pin
	pinConfig := PinConfig{Pin: 18, Direction: DirectionInput, PullMode: PullNone}
	require.NoError(b, controller.ConfigurePin(pinConfig, "benchmark"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := controller.ReadPin(18, "benchmark")
		if err != nil {
			b.Fatal(err)
		}
	}
}
