package gpio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Critical system pins that must be protected
var CriticalSystemPins = []int{
	0,  // I2C SDA (system reserved)
	1,  // I2C SCL (system reserved)
	14, // UART TXD (console/system)
	15, // UART RXD (console/system)
}

// SecurityLevel represents the security enforcement level
type SecurityLevel int

const (
	SecurityLevelPermissive SecurityLevel = iota // Allow most operations with warnings
	SecurityLevelStrict                          // Enforce strict pin restrictions
	SecurityLevelParanoid                        // Maximum security, minimal pin access
)

// SecurityConfig holds GPIO security configuration
type SecurityConfig struct {
	Level              SecurityLevel `yaml:"level"`
	AllowCriticalPins  bool          `yaml:"allow_critical_pins"`
	MaxConcurrentOps   int           `yaml:"max_concurrent_ops"`
	OperationTimeout   time.Duration `yaml:"operation_timeout"`
	EnableAuditLog     bool          `yaml:"enable_audit_log"`
	RequireUserContext bool          `yaml:"require_user_context"`
	AllowedOperations  []string      `yaml:"allowed_operations"`
}

// DefaultSecurityConfig returns secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Level:              SecurityLevelStrict,
		AllowCriticalPins:  false,
		MaxConcurrentOps:   10,
		OperationTimeout:   30 * time.Second,
		EnableAuditLog:     true,
		RequireUserContext: true,
		AllowedOperations:  []string{"read", "write", "configure"},
	}
}

// Controller manages GPIO operations and provides a unified interface
type Controller struct {
	config         *Config
	securityConfig *SecurityConfig
	impl           FullInterface
	logger         *logrus.Entry
	available      bool
	activePins     map[int]*PinState
	activeOps      int
	mutex          sync.RWMutex
	opMutex        sync.Mutex
}

// NewController creates a new GPIO controller with enhanced security
func NewController(config *Config, securityConfig *SecurityConfig, logger *logrus.Logger) *Controller {
	if config == nil {
		config = DefaultConfig()
	}
	if securityConfig == nil {
		securityConfig = DefaultSecurityConfig()
	}

	controller := &Controller{
		config:         config,
		securityConfig: securityConfig,
		logger:         logger.WithField("component", "gpio"),
		activePins:     make(map[int]*PinState),
	}

	// Initialize with enhanced security defaults
	controller.initializeSecureDefaults()

	// Initialize the appropriate implementation based on configuration
	if config.MockMode {
		controller.impl = NewMockGPIO(config)
		controller.available = true
		controller.logger.Info("Initialized GPIO controller in mock mode")
	} else {
		// Initialize real GPIO implementation using periph.io
		controller.impl = NewPeriphGPIO(config)
		controller.available = true
		controller.logger.Info("Initialized GPIO controller with periph.io implementation")
	}

	controller.logger.WithFields(logrus.Fields{
		"security_level":      securityConfig.Level,
		"allow_critical_pins": securityConfig.AllowCriticalPins,
		"max_concurrent_ops":  securityConfig.MaxConcurrentOps,
		"audit_enabled":       securityConfig.EnableAuditLog,
	}).Info("GPIO controller initialized with security configuration")

	return controller
}

// initializeSecureDefaults sets up secure default configurations
func (c *Controller) initializeSecureDefaults() {
	// Add critical system pins to restricted list if not already present
	for _, criticalPin := range CriticalSystemPins {
		found := false
		for _, restrictedPin := range c.config.RestrictedPins {
			if restrictedPin == criticalPin {
				found = true
				break
			}
		}
		if !found {
			c.config.RestrictedPins = append(c.config.RestrictedPins, criticalPin)
		}
	}

	// Apply security level specific restrictions
	switch c.securityConfig.Level {
	case SecurityLevelParanoid:
		// Only allow known safe pins
		c.config.AllowedPins = []int{18, 19, 20, 21} // PWM and safe GPIO pins
		c.config.RestrictedPins = append(c.config.RestrictedPins, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 17, 22, 23, 24, 25, 26, 27)

	case SecurityLevelStrict:
		// Default secure configuration - allow most GPIO but protect system pins
		// RestrictedPins already includes critical system pins

	case SecurityLevelPermissive:
		// Allow more pins but still protect critical ones
		if !c.securityConfig.AllowCriticalPins {
			// Keep critical pins restricted even in permissive mode
		}
	}

	c.logger.WithFields(logrus.Fields{
		"allowed_pins":    c.config.AllowedPins,
		"restricted_pins": c.config.RestrictedPins,
		"security_level":  c.securityConfig.Level,
	}).Debug("Security defaults initialized")
}

// Initialize initializes the GPIO controller
func (c *Controller) Initialize(ctx context.Context) error {
	if err := c.impl.Initialize(ctx); err != nil {
		c.logger.WithError(err).Error("Failed to initialize GPIO implementation")
		return fmt.Errorf("failed to initialize GPIO: %w", err)
	}

	c.logger.Info("GPIO controller initialized successfully")
	return nil
}

// Close closes the GPIO controller
func (c *Controller) Close() error {
	if err := c.impl.Close(); err != nil {
		c.logger.WithError(err).Error("Failed to close GPIO implementation")
		return fmt.Errorf("failed to close GPIO: %w", err)
	}

	c.logger.Info("GPIO controller closed successfully")
	return nil
}

// IsAvailable returns whether GPIO hardware is available
func (c *Controller) IsAvailable() bool {
	return c.available && c.impl.IsAvailable()
}

// IsPinAllowed checks if a pin is allowed to be used with detailed security checks
func (c *Controller) IsPinAllowed(pin int, operation string, userID string) error {
	// Check operation limits
	if err := c.checkOperationLimits(); err != nil {
		return err
	}

	// Check if operation is allowed
	if !c.isOperationAllowed(operation) {
		c.auditLog("operation_denied", fmt.Sprintf("Operation %s not allowed", operation), userID, pin)
		return fmt.Errorf("operation %s is not allowed", operation)
	}

	// Check if pin is in critical system pins list
	if c.isCriticalSystemPin(pin) && !c.securityConfig.AllowCriticalPins {
		c.auditLog("critical_pin_access_denied", fmt.Sprintf("Access to critical system pin %d denied", pin), userID, pin)
		return fmt.Errorf("pin %d is a critical system pin and cannot be accessed", pin)
	}

	// Check if pin is in restricted list
	for _, restricted := range c.config.RestrictedPins {
		if pin == restricted {
			c.auditLog("restricted_pin_access", fmt.Sprintf("Access to restricted pin %d denied", pin), userID, pin)
			return fmt.Errorf("pin %d is restricted", pin)
		}
	}

	// If allowed pins list is specified, pin must be in it
	if len(c.config.AllowedPins) > 0 {
		allowed := false
		for _, allowedPin := range c.config.AllowedPins {
			if pin == allowedPin {
				allowed = true
				break
			}
		}
		if !allowed {
			c.auditLog("pin_not_allowed", fmt.Sprintf("Pin %d not in allowed list", pin), userID, pin)
			return fmt.Errorf("pin %d is not in allowed pins list", pin)
		}
	}

	// Validate pin range (Raspberry Pi GPIO pins are 0-27)
	if pin < 0 || pin > 27 {
		return fmt.Errorf("invalid pin number %d: must be 0-27", pin)
	}

	c.auditLog("pin_access_granted", fmt.Sprintf("Access to pin %d granted for operation %s", pin, operation), userID, pin)
	return nil
}

// isCriticalSystemPin checks if a pin is in the critical system pins list
func (c *Controller) isCriticalSystemPin(pin int) bool {
	for _, criticalPin := range CriticalSystemPins {
		if pin == criticalPin {
			return true
		}
	}
	return false
}

// checkOperationLimits checks if we're within operation limits
func (c *Controller) checkOperationLimits() error {
	c.opMutex.Lock()
	defer c.opMutex.Unlock()

	if c.activeOps >= c.securityConfig.MaxConcurrentOps {
		return fmt.Errorf("maximum concurrent operations (%d) reached", c.securityConfig.MaxConcurrentOps)
	}

	c.activeOps++

	// Set up cleanup after timeout
	go func() {
		time.Sleep(c.securityConfig.OperationTimeout)
		c.opMutex.Lock()
		if c.activeOps > 0 {
			c.activeOps--
		}
		c.opMutex.Unlock()
	}()

	return nil
}

// isOperationAllowed checks if an operation is allowed
func (c *Controller) isOperationAllowed(operation string) bool {
	if len(c.securityConfig.AllowedOperations) == 0 {
		return true // If no restrictions, allow all
	}

	for _, allowedOp := range c.securityConfig.AllowedOperations {
		if operation == allowedOp {
			return true
		}
	}
	return false
}

// ConfigurePin configures a GPIO pin with security checks
func (c *Controller) ConfigurePin(config PinConfig, userID string) error {
	if err := c.IsPinAllowed(config.Pin, "configure", userID); err != nil {
		return err
	}

	// Apply default pull mode if not specified
	if config.PullMode == "" {
		config.PullMode = c.config.DefaultPullMode
	}

	// Additional security validation for PWM
	if config.PWMFrequency > 0 {
		if config.PWMFrequency > 40000 { // Limit PWM frequency for safety
			return fmt.Errorf("PWM frequency %d Hz exceeds maximum allowed (40kHz)", config.PWMFrequency)
		}
		if config.PWMDutyCycle > 100 {
			return fmt.Errorf("PWM duty cycle %d%% exceeds maximum allowed (100%%)", config.PWMDutyCycle)
		}
	}

	if err := c.impl.ConfigurePin(config); err != nil {
		c.logger.WithFields(logrus.Fields{
			"pin":       config.Pin,
			"direction": config.Direction,
			"user_id":   userID,
			"error":     err,
		}).Error("Failed to configure GPIO pin")
		c.auditLog("pin_configure_failed", fmt.Sprintf("Failed to configure pin %d: %v", config.Pin, err), userID, config.Pin)
		return fmt.Errorf("failed to configure pin %d: %w", config.Pin, err)
	}

	// Track active pin
	c.mutex.Lock()
	c.activePins[config.Pin] = &PinState{
		Pin:       config.Pin,
		Direction: config.Direction,
		PullMode:  config.PullMode,
		Timestamp: time.Now().UTC(),
	}
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"pin":       config.Pin,
		"direction": config.Direction,
		"pull_mode": config.PullMode,
		"user_id":   userID,
	}).Info("GPIO pin configured successfully")

	c.auditLog("pin_configured", fmt.Sprintf("Pin %d configured successfully", config.Pin), userID, config.Pin)
	return nil
}

// ReadPin reads the current value of a GPIO pin with security checks
func (c *Controller) ReadPin(pin int, userID string) (PinValue, error) {
	if err := c.IsPinAllowed(pin, "read", userID); err != nil {
		return Low, err
	}

	value, err := c.impl.ReadPin(pin)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"pin":     pin,
			"user_id": userID,
			"error":   err,
		}).Error("Failed to read GPIO pin")
		c.auditLog("pin_read_failed", fmt.Sprintf("Failed to read pin %d: %v", pin, err), userID, pin)
		return Low, fmt.Errorf("failed to read pin %d: %w", pin, err)
	}

	// Update active pin state
	c.mutex.Lock()
	if state, exists := c.activePins[pin]; exists {
		state.Value = value
		state.Timestamp = time.Now().UTC()
	}
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"pin":     pin,
		"value":   value,
		"user_id": userID,
	}).Debug("GPIO pin read successfully")

	c.auditLog("pin_read", fmt.Sprintf("Pin %d read, value: %d", pin, value), userID, pin)
	return value, nil
}

// WritePin writes a value to a GPIO pin with security checks
func (c *Controller) WritePin(pin int, value PinValue, userID string) error {
	if err := c.IsPinAllowed(pin, "write", userID); err != nil {
		return err
	}

	// Additional safety check for output pins
	c.mutex.RLock()
	if state, exists := c.activePins[pin]; exists {
		if state.Direction != DirectionOutput {
			c.mutex.RUnlock()
			c.auditLog("write_to_input_pin", fmt.Sprintf("Attempted to write to input pin %d", pin), userID, pin)
			return fmt.Errorf("cannot write to pin %d: configured as input", pin)
		}
	}
	c.mutex.RUnlock()

	if err := c.impl.WritePin(pin, value); err != nil {
		c.logger.WithFields(logrus.Fields{
			"pin":     pin,
			"value":   value,
			"user_id": userID,
			"error":   err,
		}).Error("Failed to write GPIO pin")
		c.auditLog("pin_write_failed", fmt.Sprintf("Failed to write pin %d: %v", pin, err), userID, pin)
		return fmt.Errorf("failed to write pin %d: %w", pin, err)
	}

	// Update active pin state
	c.mutex.Lock()
	if state, exists := c.activePins[pin]; exists {
		state.Value = value
		state.Timestamp = time.Now().UTC()
	}
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"pin":     pin,
		"value":   value,
		"user_id": userID,
	}).Debug("GPIO pin written successfully")

	c.auditLog("pin_write", fmt.Sprintf("Pin %d written, value: %d", pin, value), userID, pin)
	return nil
}

// GetPinState returns the current state of a GPIO pin with security checks
func (c *Controller) GetPinState(pin int, userID string) (*PinState, error) {
	if err := c.IsPinAllowed(pin, "read", userID); err != nil {
		return nil, err
	}

	return c.impl.GetPinState(pin)
}

// ListConfiguredPins returns a list of all configured GPIO pins
func (c *Controller) ListConfiguredPins() ([]PinState, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	states := make([]PinState, 0, len(c.activePins))
	for _, state := range c.activePins {
		states = append(states, *state)
	}

	return states, nil
}

// SetPWM configures PWM on a pin with security checks
func (c *Controller) SetPWM(pin int, frequency int, dutyCycle int, userID string) error {
	if err := c.IsPinAllowed(pin, "pwm", userID); err != nil {
		return err
	}

	// Additional PWM safety checks
	if frequency <= 0 || frequency > 40000 {
		return fmt.Errorf("PWM frequency %d Hz is outside safe range (1-40000 Hz)", frequency)
	}
	if dutyCycle < 0 || dutyCycle > 100 {
		return fmt.Errorf("PWM duty cycle %d%% is outside valid range (0-100%%)", dutyCycle)
	}

	if err := c.impl.SetPWM(pin, frequency, dutyCycle); err != nil {
		c.logger.WithFields(logrus.Fields{
			"pin":        pin,
			"frequency":  frequency,
			"duty_cycle": dutyCycle,
			"user_id":    userID,
			"error":      err,
		}).Error("Failed to set PWM")
		c.auditLog("pwm_set_failed", fmt.Sprintf("Failed to set PWM on pin %d: %v", pin, err), userID, pin)
		return fmt.Errorf("failed to set PWM on pin %d: %w", pin, err)
	}

	c.logger.WithFields(logrus.Fields{
		"pin":        pin,
		"frequency":  frequency,
		"duty_cycle": dutyCycle,
		"user_id":    userID,
	}).Info("PWM configured successfully")

	c.auditLog("pwm_configured", fmt.Sprintf("PWM on pin %d configured: %dHz, %d%%", pin, frequency, dutyCycle), userID, pin)
	return nil
}

// ReadAnalog reads an analog value with security checks
func (c *Controller) ReadAnalog(pin int, userID string) (float64, error) {
	if err := c.IsPinAllowed(pin, "read", userID); err != nil {
		return 0, err
	}

	value, err := c.impl.ReadAnalog(pin)
	if err != nil {
		c.auditLog("analog_read_failed", fmt.Sprintf("Failed to read analog pin %d: %v", pin, err), userID, pin)
		return 0, err
	}

	c.auditLog("analog_read", fmt.Sprintf("Analog pin %d read, value: %f", pin, value), userID, pin)
	return value, nil
}

// SPI methods with security checks
func (c *Controller) SPITransfer(channel int, data []byte, userID string) ([]byte, error) {
	if len(data) > 4096 { // Limit SPI transfer size for safety
		return nil, fmt.Errorf("SPI transfer size %d bytes exceeds maximum allowed (4096)", len(data))
	}

	c.auditLog("spi_transfer", fmt.Sprintf("SPI transfer on channel %d, %d bytes", channel, len(data)), userID, -1)
	return c.impl.SPITransfer(channel, data)
}

func (c *Controller) SPIWrite(channel int, data []byte, userID string) error {
	if len(data) > 4096 {
		return fmt.Errorf("SPI write size %d bytes exceeds maximum allowed (4096)", len(data))
	}

	c.auditLog("spi_write", fmt.Sprintf("SPI write on channel %d, %d bytes", channel, len(data)), userID, -1)
	return c.impl.SPIWrite(channel, data)
}

func (c *Controller) SPIRead(channel int, length int, userID string) ([]byte, error) {
	if length > 4096 {
		return nil, fmt.Errorf("SPI read length %d bytes exceeds maximum allowed (4096)", length)
	}

	c.auditLog("spi_read", fmt.Sprintf("SPI read on channel %d, %d bytes", channel, length), userID, -1)
	return c.impl.SPIRead(channel, length)
}

// I2C methods with security checks
func (c *Controller) I2CWrite(bus int, address int, data []byte, userID string) error {
	if len(data) > 256 { // Limit I2C transfer size
		return fmt.Errorf("I2C write size %d bytes exceeds maximum allowed (256)", len(data))
	}

	c.auditLog("i2c_write", fmt.Sprintf("I2C write to bus %d, address 0x%02x, %d bytes", bus, address, len(data)), userID, -1)
	return c.impl.I2CWrite(bus, address, data)
}

func (c *Controller) I2CRead(bus int, address int, length int, userID string) ([]byte, error) {
	if length > 256 {
		return nil, fmt.Errorf("I2C read length %d bytes exceeds maximum allowed (256)", length)
	}

	c.auditLog("i2c_read", fmt.Sprintf("I2C read from bus %d, address 0x%02x, %d bytes", bus, address, length), userID, -1)
	return c.impl.I2CRead(bus, address, length)
}

func (c *Controller) I2CWriteRegister(bus int, address int, register int, data []byte, userID string) error {
	if len(data) > 256 {
		return fmt.Errorf("I2C register write size %d bytes exceeds maximum allowed (256)", len(data))
	}

	c.auditLog("i2c_write_register", fmt.Sprintf("I2C write to bus %d, address 0x%02x, register 0x%02x, %d bytes", bus, address, register, len(data)), userID, -1)
	return c.impl.I2CWriteRegister(bus, address, register, data)
}

func (c *Controller) I2CReadRegister(bus int, address int, register int, length int, userID string) ([]byte, error) {
	if length > 256 {
		return nil, fmt.Errorf("I2C register read length %d bytes exceeds maximum allowed (256)", length)
	}

	c.auditLog("i2c_read_register", fmt.Sprintf("I2C read from bus %d, address 0x%02x, register 0x%02x, %d bytes", bus, address, register, length), userID, -1)
	return c.impl.I2CReadRegister(bus, address, register, length)
}

// Event methods with security checks
func (c *Controller) EnableInterrupt(pin int, eventType EventType, handler EventHandler, userID string) error {
	if err := c.IsPinAllowed(pin, "interrupt", userID); err != nil {
		return err
	}

	c.auditLog("interrupt_enabled", fmt.Sprintf("Interrupt enabled on pin %d, type: %s", pin, eventType), userID, pin)
	return c.impl.EnableInterrupt(pin, eventType, handler)
}

func (c *Controller) DisableInterrupt(pin int, userID string) error {
	if err := c.IsPinAllowed(pin, "interrupt", userID); err != nil {
		return err
	}

	c.auditLog("interrupt_disabled", fmt.Sprintf("Interrupt disabled on pin %d", pin), userID, pin)
	return c.impl.DisableInterrupt(pin)
}

func (c *Controller) StartEventLoop(ctx context.Context) error {
	return c.impl.StartEventLoop(ctx)
}

func (c *Controller) StopEventLoop() error {
	return c.impl.StopEventLoop()
}

// auditLog logs security-related events
func (c *Controller) auditLog(eventType, message, userID string, pin int) {
	if !c.securityConfig.EnableAuditLog {
		return
	}

	fields := logrus.Fields{
		"event_type": eventType,
		"message":    message,
		"user_id":    userID,
		"timestamp":  time.Now().UTC(),
	}

	if pin >= 0 {
		fields["pin"] = pin
	}

	c.logger.WithFields(fields).Info("GPIO security event")
}

// GetSecurityStats returns security-related statistics
func (c *Controller) GetSecurityStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"security_level":      c.securityConfig.Level,
		"allow_critical_pins": c.securityConfig.AllowCriticalPins,
		"active_pins":         len(c.activePins),
		"active_operations":   c.activeOps,
		"max_concurrent_ops":  c.securityConfig.MaxConcurrentOps,
		"restricted_pins":     c.config.RestrictedPins,
		"allowed_pins":        c.config.AllowedPins,
		"critical_pins":       CriticalSystemPins,
		"audit_enabled":       c.securityConfig.EnableAuditLog,
	}
}
