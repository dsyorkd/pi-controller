package agent

import (
	"context"
	"fmt"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/pkg/gpio"
	pb "github.com/dsyorkd/pi-controller/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GPIOService implements the PiAgent GPIO gRPC service
type GPIOService struct {
	pb.UnimplementedPiAgentServiceServer
	controller *gpio.Controller
	logger     logger.Interface
}

// NewGPIOService creates a new GPIO service instance
func NewGPIOService(logger logger.Interface) (*GPIOService, error) {
	// Create GPIO configuration with secure defaults
	gpioConfig := gpio.DefaultConfig()
	securityConfig := gpio.DefaultSecurityConfig()

	// For agent mode, we can be less restrictive since it's running on the node
	securityConfig.Level = gpio.SecurityLevelPermissive
	securityConfig.RequireUserContext = false // Agent operations are system-level

	// Initialize GPIO controller
	controller := gpio.NewController(gpioConfig, securityConfig, logger)

	return &GPIOService{
		controller: controller,
		logger:     logger.WithField("component", "gpio-service"),
	}, nil
}

// Initialize initializes the GPIO service
func (s *GPIOService) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing GPIO service")

	if err := s.controller.Initialize(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to initialize GPIO controller")
		return fmt.Errorf("failed to initialize GPIO controller: %w", err)
	}

	s.logger.Info("GPIO service initialized successfully")
	return nil
}

// Close shuts down the GPIO service
func (s *GPIOService) Close() error {
	s.logger.Info("Shutting down GPIO service")

	if err := s.controller.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close GPIO controller")
		return fmt.Errorf("failed to close GPIO controller: %w", err)
	}

	s.logger.Info("GPIO service shut down successfully")
	return nil
}

// ConfigureGPIOPin configures a GPIO pin
func (s *GPIOService) ConfigureGPIOPin(ctx context.Context, req *pb.ConfigureGPIOPinRequest) (*pb.ConfigureGPIOPinResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"pin":       req.Pin,
		"direction": req.Direction.String(),
		"pull_mode": req.PullMode.String(),
	}).Info("Configuring GPIO pin")

	// Convert protobuf types to internal types
	config := gpio.PinConfig{
		Pin:       int(req.Pin),
		Direction: convertDirection(req.Direction),
		PullMode:  convertPullMode(req.PullMode),
	}

	// Handle PWM configuration if provided
	if req.PwmFrequency > 0 {
		config.PWMFrequency = int(req.PwmFrequency)
		config.PWMDutyCycle = int(req.PwmDutyCycle)
	}

	// Configure the pin (using empty userID for system operations)
	if err := s.controller.ConfigurePin(config, "agent"); err != nil {
		s.logger.WithError(err).WithField("pin", req.Pin).Error("Failed to configure GPIO pin")
		return &pb.ConfigureGPIOPinResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to configure pin: %v", err),
		}, nil
	}

	return &pb.ConfigureGPIOPinResponse{
		Success:      true,
		Message:      "Pin configured successfully",
		ConfiguredAt: timestamppb.Now(),
	}, nil
}

// ReadGPIOPin reads a GPIO pin value
func (s *GPIOService) ReadGPIOPin(ctx context.Context, req *pb.ReadGPIOPinRequest) (*pb.ReadGPIOPinResponse, error) {
	s.logger.WithField("pin", req.Pin).Debug("Reading GPIO pin")

	value, err := s.controller.ReadPin(int(req.Pin), "agent")
	if err != nil {
		s.logger.WithError(err).WithField("pin", req.Pin).Error("Failed to read GPIO pin")
		return nil, fmt.Errorf("failed to read pin %d: %w", req.Pin, err)
	}

	return &pb.ReadGPIOPinResponse{
		Pin:       req.Pin,
		Value:     int32(value), // #nosec G115 -- GPIO values are 0 or 1
		Timestamp: timestamppb.Now(),
	}, nil
}

// WriteGPIOPin writes a value to a GPIO pin
func (s *GPIOService) WriteGPIOPin(ctx context.Context, req *pb.WriteGPIOPinRequest) (*pb.WriteGPIOPinResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"pin":   req.Pin,
		"value": req.Value,
	}).Debug("Writing GPIO pin")

	value := gpio.Low
	if req.Value > 0 {
		value = gpio.High
	}

	if err := s.controller.WritePin(int(req.Pin), value, "agent"); err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"pin":   req.Pin,
			"value": req.Value,
		}).Error("Failed to write GPIO pin")
		return nil, fmt.Errorf("failed to write pin %d: %w", req.Pin, err)
	}

	return &pb.WriteGPIOPinResponse{
		Pin:       req.Pin,
		Value:     req.Value,
		Timestamp: timestamppb.Now(),
	}, nil
}

// SetGPIOPWM configures PWM on a GPIO pin
func (s *GPIOService) SetGPIOPWM(ctx context.Context, req *pb.SetGPIOPWMRequest) (*pb.SetGPIOPWMResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"pin":        req.Pin,
		"frequency":  req.Frequency,
		"duty_cycle": req.DutyCycle,
	}).Info("Setting GPIO PWM")

	if err := s.controller.SetPWM(int(req.Pin), int(req.Frequency), int(req.DutyCycle), "agent"); err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"pin":        req.Pin,
			"frequency":  req.Frequency,
			"duty_cycle": req.DutyCycle,
		}).Error("Failed to set GPIO PWM")
		return &pb.SetGPIOPWMResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to set PWM: %v", err),
			Pin:     req.Pin,
		}, nil
	}

	return &pb.SetGPIOPWMResponse{
		Success:      true,
		Message:      "PWM configured successfully",
		Pin:          req.Pin,
		Frequency:    req.Frequency,
		DutyCycle:    req.DutyCycle,
		ConfiguredAt: timestamppb.Now(),
	}, nil
}

// SPIExchange performs an SPI exchange
func (s *GPIOService) SPIExchange(ctx context.Context, req *pb.SPIExchangeRequest) (*pb.SPIExchangeResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"channel":  req.Channel,
		"data_len": len(req.Data),
	}).Debug("Performing SPI exchange")

	respData, err := s.controller.SPITransfer(int(req.Channel), req.Data, "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform SPI exchange")
		return nil, fmt.Errorf("failed to perform SPI exchange: %w", err)
	}

	return &pb.SPIExchangeResponse{
		Data: respData,
	}, nil
}

// SPIWrite performs an SPI write
func (s *GPIOService) SPIWrite(ctx context.Context, req *pb.SPIWriteRequest) (*pb.SPIWriteResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"channel":  req.Channel,
		"data_len": len(req.Data),
	}).Debug("Performing SPI write")

	err := s.controller.SPIWrite(int(req.Channel), req.Data, "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform SPI write")
		return &pb.SPIWriteResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to perform SPI write: %v", err),
		}, nil
	}

	return &pb.SPIWriteResponse{
		Success: true,
		Message: "SPI write successful",
	}, nil
}

// SPIRead performs an SPI read
func (s *GPIOService) SPIRead(ctx context.Context, req *pb.SPIReadRequest) (*pb.SPIReadResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"channel": req.Channel,
		"length":  req.Length,
	}).Debug("Performing SPI read")

	respData, err := s.controller.SPIRead(int(req.Channel), int(req.Length), "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform SPI read")
		return nil, fmt.Errorf("failed to perform SPI read: %w", err)
	}

	return &pb.SPIReadResponse{
		Data: respData,
	}, nil
}

// I2CWrite performs an I2C write
func (s *GPIOService) I2CWrite(ctx context.Context, req *pb.I2CWriteRequest) (*pb.I2CWriteResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"bus":      req.Bus,
		"address":  req.Address,
		"data_len": len(req.Data),
	}).Debug("Performing I2C write")

	err := s.controller.I2CWrite(int(req.Bus), int(req.Address), req.Data, "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform I2C write")
		return &pb.I2CWriteResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to perform I2C write: %v", err),
		}, nil
	}

	return &pb.I2CWriteResponse{
		Success: true,
		Message: "I2C write successful",
	}, nil
}

// I2CRead performs an I2C read
func (s *GPIOService) I2CRead(ctx context.Context, req *pb.I2CReadRequest) (*pb.I2CReadResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"bus":     req.Bus,
		"address": req.Address,
		"length":  req.Length,
	}).Debug("Performing I2C read")

	respData, err := s.controller.I2CRead(int(req.Bus), int(req.Address), int(req.Length), "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform I2C read")
		return nil, fmt.Errorf("failed to perform I2C read: %w", err)
	}

	return &pb.I2CReadResponse{
		Data: respData,
	}, nil
}

// I2CWriteRegister performs an I2C write to a register
func (s *GPIOService) I2CWriteRegister(ctx context.Context, req *pb.I2CWriteRegisterRequest) (*pb.I2CWriteRegisterResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"bus":      req.Bus,
		"address":  req.Address,
		"register": req.Register,
		"data_len": len(req.Data),
	}).Debug("Performing I2C write register")

	err := s.controller.I2CWriteRegister(int(req.Bus), int(req.Address), int(req.Register), req.Data, "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform I2C write register")
		return &pb.I2CWriteRegisterResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to perform I2C write register: %v", err),
		}, nil
	}

	return &pb.I2CWriteRegisterResponse{
		Success: true,
		Message: "I2C write register successful",
	}, nil
}

// I2CReadRegister performs an I2C read from a register
func (s *GPIOService) I2CReadRegister(ctx context.Context, req *pb.I2CReadRegisterRequest) (*pb.I2CReadRegisterResponse, error) {
	s.logger.WithFields(map[string]interface{}{
		"bus":      req.Bus,
		"address":  req.Address,
		"register": req.Register,
		"length":   req.Length,
	}).Debug("Performing I2C read register")

	respData, err := s.controller.I2CReadRegister(int(req.Bus), int(req.Address), int(req.Register), int(req.Length), "agent")
	if err != nil {
		s.logger.WithError(err).Error("Failed to perform I2C read register")
		return nil, fmt.Errorf("failed to perform I2C read register: %w", err)
	}

	return &pb.I2CReadRegisterResponse{
		Data: respData,
	}, nil
}

// ListConfiguredPins lists configured GPIO pins
func (s *GPIOService) ListConfiguredPins(ctx context.Context, req *pb.ListConfiguredPinsRequest) (*pb.ListConfiguredPinsResponse, error) {
	s.logger.Debug("Listing configured GPIO pins")

	pins, err := s.controller.ListConfiguredPins()
	if err != nil {
		s.logger.WithError(err).Error("Failed to list configured pins")
		return nil, fmt.Errorf("failed to list configured pins: %w", err)
	}

	var pbPins []*pb.GPIOPinState
	for _, pin := range pins {
		pbPins = append(pbPins, &pb.GPIOPinState{
			Pin:         int32(pin.Pin), //nolint:gosec // GPIO pin numbers are always small positive integers
			Direction:   convertDirectionToPB(pin.Direction),
			PullMode:    convertPullModeToPB(pin.PullMode),
			Value:       int32(pin.Value), //nolint:gosec // GPIO values are 0 or 1
			LastUpdated: timestamppb.New(pin.Timestamp),
		})
	}

	return &pb.ListConfiguredPinsResponse{
		Pins: pbPins,
	}, nil
}

// AgentHealth returns the health status of the agent
func (s *GPIOService) AgentHealth(ctx context.Context, req *pb.AgentHealthRequest) (*pb.AgentHealthResponse, error) {
	status := "ok"
	gpioAvailable := s.controller.IsAvailable()

	if !gpioAvailable {
		status = "gpio_unavailable"
	}

	capabilities := []string{"gpio"}
	// TODO: Add more granular capability detection
	if gpioAvailable {
		capabilities = append(capabilities, "pwm", "spi", "i2c")
	}

	return &pb.AgentHealthResponse{
		Status:        status,
		Timestamp:     timestamppb.Now(),
		Version:       "1.0.0", // TODO: Get from build info
		Uptime:        "0s",    // TODO: Calculate uptime
		GpioAvailable: gpioAvailable,
		Capabilities:  capabilities,
	}, nil
}

// GetSystemInfo returns system information
func (s *GPIOService) GetSystemInfo(ctx context.Context, req *pb.GetSystemInfoRequest) (*pb.GetSystemInfoResponse, error) {
	// TODO: Implement system info collection
	// For now, return basic placeholder info
	return &pb.GetSystemInfoResponse{
		Hostname:        "pi-agent",
		Platform:        "linux",
		Architecture:    "arm64",
		CpuCores:        4,
		MemoryTotal:     4 * 1024 * 1024 * 1024, // 4GB
		MemoryAvailable: 2 * 1024 * 1024 * 1024, // 2GB
		KernelVersion:   "6.0.0",
		LoadAverage_1M:  0.1,
		LoadAverage_5M:  0.2,
		LoadAverage_15M: 0.3,
		Timestamp:       timestamppb.Now(),
	}, nil
}

// Helper functions to convert between protobuf and internal types

func convertDirection(dir pb.AgentGPIODirection) gpio.PinDirection {
	switch dir {
	case pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_INPUT:
		return gpio.DirectionInput
	case pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT:
		return gpio.DirectionOutput
	default:
		return gpio.DirectionInput
	}
}

func convertPullMode(pull pb.AgentGPIOPullMode) gpio.PullMode {
	switch pull {
	case pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE:
		return gpio.PullNone
	case pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UP:
		return gpio.PullUp
	case pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_DOWN:
		return gpio.PullDown
	default:
		return gpio.PullNone
	}
}

func convertDirectionToPB(dir gpio.PinDirection) pb.AgentGPIODirection {
	switch dir {
	case gpio.DirectionInput:
		return pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_INPUT
	case gpio.DirectionOutput:
		return pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_OUTPUT
	default:
		return pb.AgentGPIODirection_AGENT_GPIO_DIRECTION_UNSPECIFIED
	}
}

func convertPullModeToPB(pull gpio.PullMode) pb.AgentGPIOPullMode {
	switch pull {
	case gpio.PullNone:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_NONE
	case gpio.PullUp:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UP
	case gpio.PullDown:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_DOWN
	default:
		return pb.AgentGPIOPullMode_AGENT_GPIO_PULL_MODE_UNSPECIFIED
	}
}
