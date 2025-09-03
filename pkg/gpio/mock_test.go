package gpio

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockGPIO_Initialize tests mock GPIO initialization
func TestMockGPIO_Initialize(t *testing.T) {
	config := DefaultConfig()
	config.MockMode = true

	mock := NewMockGPIO(config)
	require.NotNil(t, mock)

	ctx := context.Background()
	err := mock.Initialize(ctx)
	require.NoError(t, err)

	// Verify mock is available
	assert.True(t, mock.IsAvailable())

	// Cleanup
	err = mock.Close()
	require.NoError(t, err)
}

// TestMockGPIO_ConfigurePin tests mock pin configuration
func TestMockGPIO_ConfigurePin(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	tests := []struct {
		name        string
		config      PinConfig
		expectedErr string
	}{
		{
			name: "configure output pin",
			config: PinConfig{
				Pin:       18,
				Direction: DirectionOutput,
				PullMode:  PullNone,
			},
		},
		{
			name: "configure input pin with pull-up",
			config: PinConfig{
				Pin:       19,
				Direction: DirectionInput,
				PullMode:  PullUp,
			},
		},
		{
			name: "configure PWM pin",
			config: PinConfig{
				Pin:          20,
				Direction:    DirectionOutput,
				PullMode:     PullNone,
				PWMFrequency: 1000,
				PWMDutyCycle: 75,
			},
		},
		{
			name: "invalid pin number - negative",
			config: PinConfig{
				Pin:       -1,
				Direction: DirectionOutput,
				PullMode:  PullNone,
			},
			expectedErr: "invalid pin number",
		},
		{
			name: "invalid pin number - too high",
			config: PinConfig{
				Pin:       41,
				Direction: DirectionOutput,
				PullMode:  PullNone,
			},
			expectedErr: "invalid pin number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mock.ConfigurePin(tt.config)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Verify pin state
				state, err := mock.GetPinState(tt.config.Pin)
				require.NoError(t, err)
				assert.Equal(t, tt.config.Pin, state.Pin)
				assert.Equal(t, tt.config.Direction, state.Direction)
				assert.Equal(t, tt.config.PullMode, state.PullMode)
			}
		})
	}
}

// TestMockGPIO_WritePin tests mock pin writing
func TestMockGPIO_WritePin(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Configure pins
	outputConfig := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	inputConfig := PinConfig{Pin: 19, Direction: DirectionInput, PullMode: PullNone}

	require.NoError(t, mock.ConfigurePin(outputConfig))
	require.NoError(t, mock.ConfigurePin(inputConfig))

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mock.WritePin(tt.pin, tt.value)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Verify the value was written
				state, err := mock.GetPinState(tt.pin)
				require.NoError(t, err)
				assert.Equal(t, tt.value, state.Value)
			}
		})
	}
}

// TestMockGPIO_ReadPin tests mock pin reading
func TestMockGPIO_ReadPin(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Configure pins
	outputConfig := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	inputConfig := PinConfig{Pin: 19, Direction: DirectionInput, PullMode: PullNone}

	require.NoError(t, mock.ConfigurePin(outputConfig))
	require.NoError(t, mock.ConfigurePin(inputConfig))

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
			value, err := mock.ReadPin(tt.pin)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Equal(t, Low, value) // Should return Low on error
			} else {
				require.NoError(t, err)
				assert.Contains(t, []PinValue{Low, High}, value)
			}
		})
	}
}

// TestMockGPIO_PWM tests mock PWM functionality
func TestMockGPIO_PWM(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Configure output pin
	outputConfig := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	require.NoError(t, mock.ConfigurePin(outputConfig))

	tests := []struct {
		name        string
		pin         int
		frequency   int
		dutyCycle   int
		expectedErr string
	}{
		{
			name:      "valid PWM settings",
			pin:       18,
			frequency: 1000,
			dutyCycle: 50,
		},
		{
			name:      "minimum frequency",
			pin:       18,
			frequency: 1,
			dutyCycle: 25,
		},
		{
			name:      "maximum frequency",
			pin:       18,
			frequency: 10000,
			dutyCycle: 75,
		},
		{
			name:      "minimum duty cycle",
			pin:       18,
			frequency: 1000,
			dutyCycle: 0,
		},
		{
			name:      "maximum duty cycle",
			pin:       18,
			frequency: 1000,
			dutyCycle: 100,
		},
		{
			name:        "invalid frequency - too low",
			pin:         18,
			frequency:   0,
			dutyCycle:   50,
			expectedErr: "invalid PWM frequency",
		},
		{
			name:        "invalid frequency - too high",
			pin:         18,
			frequency:   10001,
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
			err := mock.SetPWM(tt.pin, tt.frequency, tt.dutyCycle)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMockGPIO_AnalogRead tests mock analog reading
func TestMockGPIO_AnalogRead(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Configure input pin
	inputConfig := PinConfig{Pin: 18, Direction: DirectionInput, PullMode: PullNone}
	require.NoError(t, mock.ConfigurePin(inputConfig))

	tests := []struct {
		name        string
		pin         int
		expectedErr string
	}{
		{
			name: "read analog from input pin",
			pin:  18,
		},
		{
			name:        "read analog from unconfigured pin",
			pin:         19,
			expectedErr: "not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := mock.ReadAnalog(tt.pin)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Equal(t, 0.0, value)
			} else {
				require.NoError(t, err)
				assert.GreaterOrEqual(t, value, 0.0)
				assert.LessOrEqual(t, value, 1.0)
			}
		})
	}

	// Configure output pin and test that it should fail
	outputConfig := PinConfig{Pin: 19, Direction: DirectionOutput, PullMode: PullNone}
	require.NoError(t, mock.ConfigurePin(outputConfig))

	_, err := mock.ReadAnalog(19)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured as input")
}

// TestMockGPIO_SPI tests mock SPI functionality
func TestMockGPIO_SPI(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	tests := []struct {
		name        string
		channel     int
		data        []byte
		expectedErr string
	}{
		{
			name:    "SPI transfer on channel 0",
			channel: 0,
			data:    []byte{0x01, 0x02, 0x03},
		},
		{
			name:    "SPI transfer on channel 1",
			channel: 1,
			data:    []byte{0xFF, 0x00, 0xAA, 0x55},
		},
		{
			name:    "empty data transfer",
			channel: 0,
			data:    []byte{},
		},
		{
			name:        "invalid channel - negative",
			channel:     -1,
			data:        []byte{0x01},
			expectedErr: "invalid SPI channel",
		},
		{
			name:        "invalid channel - too high",
			channel:     2,
			data:        []byte{0x01},
			expectedErr: "invalid SPI channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test SPI transfer
			result, err := mock.SPITransfer(tt.channel, tt.data)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, len(tt.data))

				// Mock implementation should invert bits
				for i, expected := range tt.data {
					assert.Equal(t, ^expected, result[i])
				}
			}

			// Test SPI write
			err = mock.SPIWrite(tt.channel, tt.data)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}

			// Test SPI read
			readData, err := mock.SPIRead(tt.channel, len(tt.data))
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, readData)
			} else if len(tt.data) > 0 {
				require.NoError(t, err)
				require.NotNil(t, readData)
				assert.Len(t, readData, len(tt.data))
			}
		})
	}

	// Test SPI read with invalid length
	_, err := mock.SPIRead(0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid read length")

	_, err = mock.SPIRead(0, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid read length")
}

// TestMockGPIO_I2C tests mock I2C functionality
func TestMockGPIO_I2C(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	tests := []struct {
		name        string
		bus         int
		address     int
		register    int
		data        []byte
		expectedErr string
	}{
		{
			name:     "I2C on bus 0",
			bus:      0,
			address:  0x48,
			register: 0x00,
			data:     []byte{0x01, 0x02},
		},
		{
			name:     "I2C on bus 1",
			bus:      1,
			address:  0x50,
			register: 0x10,
			data:     []byte{0xFF},
		},
		{
			name:     "minimum valid address",
			bus:      0,
			address:  0x08,
			register: 0x00,
			data:     []byte{0x00},
		},
		{
			name:     "maximum valid address",
			bus:      0,
			address:  0x77,
			register: 0xFF,
			data:     []byte{0xFF},
		},
		{
			name:        "invalid bus - negative",
			bus:         -1,
			address:     0x48,
			register:    0x00,
			data:        []byte{0x01},
			expectedErr: "invalid I2C bus",
		},
		{
			name:        "invalid bus - too high",
			bus:         2,
			address:     0x48,
			register:    0x00,
			data:        []byte{0x01},
			expectedErr: "invalid I2C bus",
		},
		{
			name:        "invalid address - too low",
			bus:         0,
			address:     0x07,
			register:    0x00,
			data:        []byte{0x01},
			expectedErr: "invalid I2C address",
		},
		{
			name:        "invalid address - too high",
			bus:         0,
			address:     0x78,
			register:    0x00,
			data:        []byte{0x01},
			expectedErr: "invalid I2C address",
		},
		{
			name:        "invalid register - negative",
			bus:         0,
			address:     0x48,
			register:    -1,
			data:        []byte{0x01},
			expectedErr: "invalid I2C register",
		},
		{
			name:        "invalid register - too high",
			bus:         0,
			address:     0x48,
			register:    256,
			data:        []byte{0x01},
			expectedErr: "invalid I2C register",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test I2C write
			err := mock.I2CWrite(tt.bus, tt.address, tt.data)
			if tt.expectedErr != "" && !contains(tt.expectedErr, "register") {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}

			// Test I2C read
			readData, err := mock.I2CRead(tt.bus, tt.address, len(tt.data))
			if tt.expectedErr != "" && !contains(tt.expectedErr, "register") {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, readData)
			} else if len(tt.data) > 0 {
				require.NoError(t, err)
				require.NotNil(t, readData)
				assert.Len(t, readData, len(tt.data))
			}

			// Test I2C register write
			err = mock.I2CWriteRegister(tt.bus, tt.address, tt.register, tt.data)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}

			// Test I2C register read
			readData, err = mock.I2CReadRegister(tt.bus, tt.address, tt.register, len(tt.data))
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, readData)
			} else if len(tt.data) > 0 {
				require.NoError(t, err)
				require.NotNil(t, readData)
				assert.Len(t, readData, len(tt.data))
			}
		})
	}

	// Test I2C read with invalid length
	_, err := mock.I2CRead(0, 0x48, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid read length")

	_, err = mock.I2CReadRegister(0, 0x48, 0x00, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid read length")
}

// TestMockGPIO_Events tests mock event handling
func TestMockGPIO_Events(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Configure input pin for events
	inputConfig := PinConfig{Pin: 18, Direction: DirectionInput, PullMode: PullNone}
	require.NoError(t, mock.ConfigurePin(inputConfig))

	// Test enabling interrupt on unconfigured pin
	handler := func(event Event) {}
	err := mock.EnableInterrupt(19, EventBothEdges, handler)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")

	// Test enabling interrupt on configured pin
	err = mock.EnableInterrupt(18, EventBothEdges, handler)
	require.NoError(t, err)

	// Test starting event loop
	err = mock.StartEventLoop(ctx)
	require.NoError(t, err)

	// Test starting event loop again should fail
	err = mock.StartEventLoop(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test disabling interrupt
	err = mock.DisableInterrupt(18)
	require.NoError(t, err)

	// Test stopping event loop
	err = mock.StopEventLoop()
	require.NoError(t, err)

	// Test stopping event loop again should fail
	err = mock.StopEventLoop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestMockGPIO_EventTriggering tests event triggering through WritePin
func TestMockGPIO_EventTriggering(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Configure output pin (for writing to trigger events)
	outputConfig := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	require.NoError(t, mock.ConfigurePin(outputConfig))

	// Set up event handler
	eventReceived := make(chan Event, 10)
	handler := func(event Event) {
		eventReceived <- event
	}

	// Enable interrupt
	err := mock.EnableInterrupt(18, EventBothEdges, handler)
	require.NoError(t, err)

	// Write values to trigger events
	require.NoError(t, mock.WritePin(18, High))
	require.NoError(t, mock.WritePin(18, Low))
	require.NoError(t, mock.WritePin(18, High))

	// Check that events were triggered
	timeout := time.After(1 * time.Second)
	eventCount := 0

	for eventCount < 3 {
		select {
		case event := <-eventReceived:
			assert.Equal(t, 18, event.Pin)
			assert.Equal(t, EventBothEdges, event.Type)
			assert.Contains(t, []PinValue{Low, High}, event.Value)
			eventCount++
		case <-timeout:
			break
		}
	}

	assert.Equal(t, 3, eventCount, "Expected 3 events to be triggered")
}

// TestMockGPIO_ListConfiguredPins tests listing configured pins
func TestMockGPIO_ListConfiguredPins(t *testing.T) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(t, mock.Initialize(ctx))
	defer mock.Close()

	// Initially no pins configured
	pins, err := mock.ListConfiguredPins()
	require.NoError(t, err)
	assert.Len(t, pins, 0)

	// Configure some pins
	pinConfigs := []PinConfig{
		{Pin: 18, Direction: DirectionOutput, PullMode: PullNone},
		{Pin: 19, Direction: DirectionInput, PullMode: PullUp},
		{Pin: 20, Direction: DirectionOutput, PullMode: PullDown},
	}

	for _, config := range pinConfigs {
		require.NoError(t, mock.ConfigurePin(config))
	}

	// List configured pins
	pins, err = mock.ListConfiguredPins()
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
		assert.Equal(t, Low, pin.Value) // Default value
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && s[1:len(substr)+1] == substr)))
}

// BenchmarkMockGPIO_WritePin benchmarks mock GPIO writing
func BenchmarkMockGPIO_WritePin(b *testing.B) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(b, mock.Initialize(ctx))
	defer mock.Close()

	// Configure output pin
	outputConfig := PinConfig{Pin: 18, Direction: DirectionOutput, PullMode: PullNone}
	require.NoError(b, mock.ConfigurePin(outputConfig))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		value := PinValue(i % 2)
		err := mock.WritePin(18, value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMockGPIO_ReadPin benchmarks mock GPIO reading
func BenchmarkMockGPIO_ReadPin(b *testing.B) {
	mock := NewMockGPIO(DefaultConfig())
	ctx := context.Background()
	require.NoError(b, mock.Initialize(ctx))
	defer mock.Close()

	// Configure input pin
	inputConfig := PinConfig{Pin: 18, Direction: DirectionInput, PullMode: PullNone}
	require.NoError(b, mock.ConfigurePin(inputConfig))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := mock.ReadPin(18)
		if err != nil {
			b.Fatal(err)
		}
	}
}
