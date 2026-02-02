package gpio

import (
	"context"
	"fmt"
	"sync"
	"time"

	applogger "github.com/dsyorkd/pi-controller/internal/logger"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

// PeriphGPIO implements the GPIO interface using periph.io
type PeriphGPIO struct {
	config       *Config
	logger       applogger.Interface
	initialized  bool
	pins         map[int]*pinState
	pwmPins      map[int]*pwmState
	spiDevices   map[int]spi.Port // Channel to SPI port
	i2cDevices   map[int]i2c.Bus  // Bus to I2C bus
	mutex        sync.RWMutex
	eventLoopCtx context.Context
	eventCancel  context.CancelFunc
	eventWG      sync.WaitGroup
}

// pinState tracks the state of a configured GPIO pin
type pinState struct {
	pin       gpio.PinIO
	config    PinConfig
	lastRead  time.Time
	lastValue PinValue
}

// pwmState tracks PWM configuration
type pwmState struct {
	pin       gpio.PinOut
	frequency int
	dutyCycle int
	active    bool
}

// NewPeriphGPIO creates a new periph.io-based GPIO implementation
func NewPeriphGPIO(config *Config) *PeriphGPIO {
	if config == nil {
		config = DefaultConfig()
	}

	logger := applogger.Default().WithField("component", "periph-gpio")

	return &PeriphGPIO{
		config:     config,
		logger:     logger,
		pins:       make(map[int]*pinState),
		pwmPins:    make(map[int]*pwmState),
		spiDevices: make(map[int]spi.Port),
		i2cDevices: make(map[int]i2c.Bus),
	}
}

// Initialize initializes the periph.io GPIO system
func (p *PeriphGPIO) Initialize(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.initialized {
		return nil
	}

	p.logger.Info("Initializing periph.io GPIO system")

	// Initialize periph.io host
	if _, err := host.Init(); err != nil {
		p.logger.Errorf("Failed to initialize periph.io host: %v", err)
		return fmt.Errorf("failed to initialize periph.io host: %w", err)
	}

	p.initialized = true
	p.logger.Info("periph.io GPIO system initialized successfully")

	return nil
}

// Close shuts down the GPIO system and releases resources
func (p *PeriphGPIO) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return nil
	}

	p.logger.Info("Shutting down periph.io GPIO system")

	// Stop event loop if running
	if p.eventCancel != nil {
		p.eventCancel()
		p.eventWG.Wait()
	}

	// Reset all configured pins to safe states
	for pinNum, state := range p.pins {
		if state.config.Direction == DirectionOutput {
			if err := state.pin.Out(gpio.Low); err != nil {
				p.logger.WithField("pin", pinNum).Warnf("Failed to reset pin to low: %v", err)
			}
		}
	}

	// Stop PWM on all pins
	for pinNum, pwm := range p.pwmPins {
		if pwm.active {
			if err := pwm.pin.Out(gpio.Low); err != nil {
				p.logger.WithField("pin", pinNum).Warnf("Failed to stop PWM: %v", err)
			}
		}
	}

	p.pins = make(map[int]*pinState)
	p.pwmPins = make(map[int]*pwmState)
	p.initialized = false

	p.logger.Info("periph.io GPIO system shut down successfully")
	return nil
}

// ConfigurePin configures a GPIO pin with the specified settings
func (p *PeriphGPIO) ConfigurePin(config PinConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	p.logger.WithFields(map[string]interface{}{
		"pin":       config.Pin,
		"direction": config.Direction,
		"pull_mode": config.PullMode,
	}).Debug("Configuring GPIO pin")

	// Get the pin by BCM number
	pin := gpioreg.ByName(fmt.Sprintf("GPIO%d", config.Pin))
	if pin == nil {
		return fmt.Errorf("pin GPIO%d not found", config.Pin)
	}

	// Convert pull mode
	pull := gpio.PullNoChange
	switch config.PullMode {
	case PullNone:
		pull = gpio.Float
	case PullUp:
		pull = gpio.PullUp
	case PullDown:
		pull = gpio.PullDown
	}

	// Configure pin direction and pull
	var err error
	switch config.Direction {
	case DirectionInput:
		err = pin.In(pull, gpio.NoEdge)
	case DirectionOutput:
		// Start with low output
		err = pin.Out(gpio.Low)
	default:
		return fmt.Errorf("invalid pin direction: %s", config.Direction)
	}

	if err != nil {
		return fmt.Errorf("failed to configure pin %d: %w", config.Pin, err)
	}

	// Store pin state
	p.pins[config.Pin] = &pinState{
		pin:    pin,
		config: config,
	}

	// Handle PWM configuration if specified
	if config.PWMFrequency > 0 {
		if err := p.configurePWM(config.Pin, config.PWMFrequency, config.PWMDutyCycle); err != nil {
			return fmt.Errorf("failed to configure PWM: %w", err)
		}
	}

	p.logger.WithField("pin", config.Pin).Info("GPIO pin configured successfully")
	return nil
}

// configurePWM sets up PWM on a pin
func (p *PeriphGPIO) configurePWM(pinNum, frequency, dutyCycle int) error {
	// PWM is only available on certain pins
	validPWMPins := map[int]bool{
		12: true, 13: true, 18: true, 19: true,
	}

	if !validPWMPins[pinNum] {
		return fmt.Errorf("pin %d does not support PWM", pinNum)
	}

	state, exists := p.pins[pinNum]
	if !exists {
		return fmt.Errorf("pin %d not configured", pinNum)
	}

	// For now, we'll simulate PWM using rapid on/off switching
	// In a production implementation, you'd use hardware PWM where available
	p.pwmPins[pinNum] = &pwmState{
		pin:       state.pin,
		frequency: frequency,
		dutyCycle: dutyCycle,
		active:    false,
	}

	return nil
}

// ReadPin reads the current value of a GPIO pin
func (p *PeriphGPIO) ReadPin(pin int) (PinValue, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if !p.initialized {
		return Low, fmt.Errorf("GPIO system not initialized")
	}

	state, exists := p.pins[pin]
	if !exists {
		return Low, fmt.Errorf("pin %d not configured", pin)
	}

	level := state.pin.Read()
	value := Low
	if level == gpio.High {
		value = High
	}

	// Update state
	state.lastRead = time.Now()
	state.lastValue = value

	p.logger.WithFields(map[string]interface{}{
		"pin":   pin,
		"value": value,
	}).Debug("GPIO pin read")

	return value, nil
}

// WritePin writes a value to a GPIO pin
func (p *PeriphGPIO) WritePin(pin int, value PinValue) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	state, exists := p.pins[pin]
	if !exists {
		return fmt.Errorf("pin %d not configured", pin)
	}

	if state.config.Direction != DirectionOutput {
		return fmt.Errorf("pin %d is not configured as output", pin)
	}

	// Convert value to gpio level
	level := gpio.Low
	if value == High {
		level = gpio.High
	}

	if err := state.pin.Out(level); err != nil {
		return fmt.Errorf("failed to write pin %d: %w", pin, err)
	}

	// Update state
	state.lastValue = value

	p.logger.WithFields(map[string]interface{}{
		"pin":   pin,
		"value": value,
	}).Debug("GPIO pin written")

	return nil
}

// GetPinState returns the current state of a GPIO pin
func (p *PeriphGPIO) GetPinState(pin int) (*PinState, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	state, exists := p.pins[pin]
	if !exists {
		return nil, fmt.Errorf("pin %d not configured", pin)
	}

	// Read current value to ensure fresh state
	currentValue, err := p.ReadPin(pin)
	if err != nil {
		return nil, err
	}

	return &PinState{
		Pin:       pin,
		Direction: state.config.Direction,
		Value:     currentValue,
		PullMode:  state.config.PullMode,
		Timestamp: time.Now(),
	}, nil
}

// ListConfiguredPins returns a list of all configured GPIO pins
func (p *PeriphGPIO) ListConfiguredPins() ([]PinState, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	states := make([]PinState, 0, len(p.pins))
	for pinNum := range p.pins {
		state, err := p.GetPinState(pinNum)
		if err != nil {
			p.logger.WithField("pin", pinNum).Warnf("Failed to get pin state: %v", err)
			continue
		}
		states = append(states, *state)
	}

	return states, nil
}

// SetPWM configures PWM on a pin
func (p *PeriphGPIO) SetPWM(pin int, frequency int, dutyCycle int) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	// Validate parameters
	if frequency <= 0 || frequency > 40000 {
		return fmt.Errorf("frequency %d Hz out of range (1-40000)", frequency)
	}
	if dutyCycle < 0 || dutyCycle > 100 {
		return fmt.Errorf("duty cycle %d%% out of range (0-100)", dutyCycle)
	}

	pwmState, exists := p.pwmPins[pin]
	if !exists {
		return fmt.Errorf("pin %d not configured for PWM", pin)
	}

	// Update PWM parameters
	pwmState.frequency = frequency
	pwmState.dutyCycle = dutyCycle

	// Start PWM if not already active
	if !pwmState.active {
		go p.runSoftwarePWM(pin)
		pwmState.active = true
	}

	p.logger.WithFields(map[string]interface{}{
		"pin":        pin,
		"frequency":  frequency,
		"duty_cycle": dutyCycle,
	}).Info("PWM configured")

	return nil
}

// runSoftwarePWM implements software PWM for pins that don't have hardware PWM
func (p *PeriphGPIO) runSoftwarePWM(pinNum int) {
	p.mutex.RLock()
	pwmState := p.pwmPins[pinNum]
	p.mutex.RUnlock()

	if pwmState == nil {
		return
	}

	ticker := time.NewTicker(time.Second / time.Duration(pwmState.frequency))
	defer ticker.Stop()

	for pwmState.active {
		// Calculate on/off durations
		period := time.Second / time.Duration(pwmState.frequency)
		onTime := time.Duration(float64(period) * float64(pwmState.dutyCycle) / 100.0)
		offTime := period - onTime

		// PWM cycle: ON phase
		if onTime > 0 {
			if err := pwmState.pin.Out(gpio.High); err != nil {
				p.logger.WithField("pin", pinNum).Errorf("Failed to set pin high during PWM: %v", err)
				break
			}
			time.Sleep(onTime)
		}

		// PWM cycle: OFF phase
		if offTime > 0 {
			if err := pwmState.pin.Out(gpio.Low); err != nil {
				p.logger.WithField("pin", pinNum).Errorf("Failed to set pin low during PWM: %v", err)
				break
			}
			time.Sleep(offTime)
		}

		// Check if PWM is still active (could be changed by another goroutine)
		p.mutex.RLock()
		active := pwmState.active
		p.mutex.RUnlock()

		if !active {
			break
		}
	}
}

// ReadAnalog reads an analog value (not supported on Raspberry Pi GPIO)
func (p *PeriphGPIO) ReadAnalog(pin int) (float64, error) {
	return 0, fmt.Errorf("analog reading not supported on Raspberry Pi GPIO pins")
}

// IsAvailable returns whether GPIO hardware is available
func (p *PeriphGPIO) IsAvailable() bool {
	return p.initialized
}

// SPITransfer performs a full-duplex SPI transfer
func (p *PeriphGPIO) SPITransfer(channel int, data []byte) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return nil, fmt.Errorf("GPIO system not initialized")
	}

	// Get or open SPI port
	port, err := p.getSPIPort(channel)
	if err != nil {
		return nil, err
	}

	// Connect to the SPI device
	conn, err := port.Connect(physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SPI device: %w", err)
	}

	// Perform the transfer
	readData := make([]byte, len(data))
	if err := conn.Tx(data, readData); err != nil {
		return nil, fmt.Errorf("failed to perform SPI transfer on channel %d: %w", channel, err)
	}

	return readData, nil
}

// getSPIPort gets an SPI port by channel, opening it if necessary
func (p *PeriphGPIO) getSPIPort(channel int) (spi.Port, error) {
	if port, exists := p.spiDevices[channel]; exists {
		return port, nil
	}

	// Open the SPI bus
	// Periph.io uses /dev/spidevX.Y where X is bus and Y is chip select
	// We'll assume default chip select 0 for now
	port, err := spireg.Open(fmt.Sprintf("SPI%d.0", channel))
	if err != nil {
		return nil, fmt.Errorf("failed to open SPI port %d: %w", channel, err)
	}

	p.spiDevices[channel] = port
	return port, nil
}

// SPIWrite writes data to SPI
func (p *PeriphGPIO) SPIWrite(channel int, data []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	// Get or open SPI port
	port, err := p.getSPIPort(channel)
	if err != nil {
		return err
	}

	// Connect to the SPI device
	conn, err := port.Connect(physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return fmt.Errorf("failed to connect to SPI device: %w", err)
	}

	// Perform the write
	if err := conn.Tx(data, nil); err != nil { // Tx with nil read data performs write only
		return fmt.Errorf("failed to perform SPI write on channel %d: %w", channel, err)
	}

	return nil
}

// SPIRead reads data from SPI
func (p *PeriphGPIO) SPIRead(channel int, length int) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return nil, fmt.Errorf("GPIO system not initialized")
	}

	// Get or open SPI port
	port, err := p.getSPIPort(channel)
	if err != nil {
		return nil, err
	}

	// Connect to the SPI device
	conn, err := port.Connect(physic.MegaHertz, spi.Mode0, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SPI device: %w", err)
	}

	// Perform the read
	readData := make([]byte, length)
	if err := conn.Tx(nil, readData); err != nil { // Tx with nil write data performs read only
		return nil, fmt.Errorf("failed to perform SPI read on channel %d: %w", channel, err)
	}

	return readData, nil
}

// I2CWrite writes data to an I2C device
func (p *PeriphGPIO) I2CWrite(bus int, address int, data []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	// Validate I2C address is within valid range (0-1023 for 10-bit addressing)
	if address < 0 || address > 1023 {
		return fmt.Errorf("I2C address 0x%02x out of valid range (0-1023)", address)
	}

	// Get or open I2C bus
	i2cBus, err := p.getI2CBus(bus)
	if err != nil {
		return err
	}

	// Get I2C device handle
	dev := &i2c.Dev{Bus: i2cBus, Addr: uint16(address)} //nolint:gosec // address validated above

	// Perform the write
	if _, err := dev.Write(data); err != nil {
		return fmt.Errorf("failed to write to I2C bus %d address 0x%02x: %w", bus, address, err)
	}

	return nil
}

// getI2CBus gets an I2C bus by number, opening it if necessary
func (p *PeriphGPIO) getI2CBus(bus int) (i2c.Bus, error) {
	if i2cBus, exists := p.i2cDevices[bus]; exists {
		return i2cBus, nil
	}

	// Open the I2C bus
	i2cBus, err := i2creg.Open(fmt.Sprintf("I2C%d", bus))
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus %d: %w", bus, err)
	}

	p.i2cDevices[bus] = i2cBus
	return i2cBus, nil
}

// I2CRead reads data from an I2C device
func (p *PeriphGPIO) I2CRead(bus int, address int, length int) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return nil, fmt.Errorf("GPIO system not initialized")
	}

	// Validate I2C address is within valid range (0-1023 for 10-bit addressing)
	if address < 0 || address > 1023 {
		return nil, fmt.Errorf("I2C address 0x%02x out of valid range (0-1023)", address)
	}

	// Get or open I2C bus
	i2cBus, err := p.getI2CBus(bus)
	if err != nil {
		return nil, err
	}

	// Get I2C device handle
	dev := &i2c.Dev{Bus: i2cBus, Addr: uint16(address)} //nolint:gosec // address validated above

	// Perform the read
	readData := make([]byte, length)
	if err := dev.Tx(nil, readData); err != nil {
		return nil, fmt.Errorf("failed to read from I2C bus %d address 0x%02x: %w", bus, address, err)
	}

	return readData, nil
}

// I2CWriteRegister writes data to a specific register on an I2C device
func (p *PeriphGPIO) I2CWriteRegister(bus int, address int, register int, data []byte) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	// Validate I2C address is within valid range (0-1023 for 10-bit addressing)
	if address < 0 || address > 1023 {
		return fmt.Errorf("I2C address 0x%02x out of valid range (0-1023)", address)
	}

	// Get or open I2C bus
	i2cBus, err := p.getI2CBus(bus)
	if err != nil {
		return err
	}

	// Get I2C device handle
	dev := &i2c.Dev{Bus: i2cBus, Addr: uint16(address)} //nolint:gosec // address validated above

	// Prepare data: register address followed by actual data
	writeData := append([]byte{byte(register)}, data...)

	// Perform the write
	if _, err := dev.Write(writeData); err != nil {
		return fmt.Errorf("failed to write to I2C bus %d address 0x%02x register 0x%02x: %w", bus, address, register, err)
	}

	return nil
}

// I2CReadRegister reads data from a specific register on an I2C device
func (p *PeriphGPIO) I2CReadRegister(bus int, address int, register int, length int) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return nil, fmt.Errorf("GPIO system not initialized")
	}

	// Validate I2C address is within valid range (0-1023 for 10-bit addressing)
	if address < 0 || address > 1023 {
		return nil, fmt.Errorf("I2C address 0x%02x out of valid range (0-1023)", address)
	}

	// Get or open I2C bus
	i2cBus, err := p.getI2CBus(bus)
	if err != nil {
		return nil, err
	}

	// Get I2C device handle
	dev := &i2c.Dev{Bus: i2cBus, Addr: uint16(address)} //nolint:gosec // address validated above

	// Write register address, then read data
	writeData := []byte{byte(register)}
	readData := make([]byte, length)

	if err := dev.Tx(writeData, readData); err != nil {
		return nil, fmt.Errorf("failed to read from I2C bus %d address 0x%02x register 0x%02x: %w", bus, address, register, err)
	}

	return readData, nil
}

// Event Interface methods
func (p *PeriphGPIO) EnableInterrupt(pin int, eventType EventType, handler EventHandler) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized {
		return fmt.Errorf("GPIO system not initialized")
	}

	state, exists := p.pins[pin]
	if !exists {
		return fmt.Errorf("pin %d not configured", pin)
	}

	if state.config.Direction != DirectionInput {
		return fmt.Errorf("pin %d must be configured as input for interrupts", pin)
	}

	// Configure edge detection
	var edge gpio.Edge
	switch eventType {
	case EventRisingEdge:
		edge = gpio.RisingEdge
	case EventFallingEdge:
		edge = gpio.FallingEdge
	case EventBothEdges:
		edge = gpio.BothEdges
	default:
		return fmt.Errorf("invalid event type: %s", eventType)
	}

	if err := state.pin.In(gpio.PullNoChange, edge); err != nil {
		return fmt.Errorf("failed to configure interrupt on pin %d: %w", pin, err)
	}

	// Start monitoring in a goroutine
	go p.monitorPin(pin, eventType, handler)

	p.logger.WithFields(map[string]interface{}{
		"pin":        pin,
		"event_type": eventType,
	}).Info("Interrupt enabled on pin")

	return nil
}

func (p *PeriphGPIO) DisableInterrupt(pin int) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	state, exists := p.pins[pin]
	if !exists {
		return fmt.Errorf("pin %d not configured", pin)
	}

	// Disable edge detection
	if err := state.pin.In(gpio.PullNoChange, gpio.NoEdge); err != nil {
		return fmt.Errorf("failed to disable interrupt on pin %d: %w", pin, err)
	}

	p.logger.WithField("pin", pin).Info("Interrupt disabled on pin")
	return nil
}

// monitorPin monitors a pin for events
func (p *PeriphGPIO) monitorPin(pinNum int, eventType EventType, handler EventHandler) {
	state := p.pins[pinNum]
	if state == nil {
		return
	}

	for {
		// Wait for edge
		if state.pin.WaitForEdge(-1) {
			value := Low
			if state.pin.Read() == gpio.High {
				value = High
			}

			event := Event{
				Pin:       pinNum,
				Type:      eventType,
				Value:     value,
				Timestamp: time.Now(),
			}

			// Call the handler
			handler(event)
		}

		// Check if we should continue monitoring
		p.mutex.RLock()
		_, exists := p.pins[pinNum]
		p.mutex.RUnlock()

		if !exists {
			break
		}
	}
}

func (p *PeriphGPIO) StartEventLoop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.eventLoopCtx != nil {
		return fmt.Errorf("event loop already running")
	}

	p.eventLoopCtx, p.eventCancel = context.WithCancel(ctx)
	p.logger.Info("Started GPIO event loop")

	return nil
}

func (p *PeriphGPIO) StopEventLoop() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.eventCancel != nil {
		p.eventCancel()
		p.eventWG.Wait()
		p.eventLoopCtx = nil
		p.eventCancel = nil
	}

	p.logger.Info("Stopped GPIO event loop")
	return nil
}
