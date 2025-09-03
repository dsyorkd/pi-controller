package services

import "context"

// GPIOPinMode defines allowed GPIO pin modes
type GPIOPinMode string

const (
	GPIOPinModeInput   GPIOPinMode = "input"
	GPIOPinModeOutput  GPIOPinMode = "output"
	GPIOPinModePWM     GPIOPinMode = "pwm"
	GPIOPinModeCleanup GPIOPinMode = "cleanup"
)

// GPIORequest represents a request to configure a GPIO pin for Kubernetes controllers
type GPIORequest struct {
	NodeID       string      `json:"node_id"`
	PinNumber    int         `json:"pin_number"`
	Mode         GPIOPinMode `json:"mode"`      // input, output, pwm, cleanup
	Direction    string      `json:"direction"` // in, out
	Value        string      `json:"value"`     // low, high
	PullMode     string      `json:"pull_mode"` // none, up, down
	DebounceMs   int         `json:"debounce_ms"`
	PWMFrequency int         `json:"pwm_frequency"`
	PWMDutyCycle float64     `json:"pwm_duty_cycle"`
}

// PWMRequest represents a request to configure a PWM controller for Kubernetes controllers
type PWMRequest struct {
	NodeID        string              `json:"node_id"`
	Address       int                 `json:"address"`
	BaseFrequency int                 `json:"base_frequency"`
	ChannelCount  int                 `json:"channel_count"`
	OutputEnable  bool                `json:"output_enable"`
	InvertOutput  bool                `json:"invert_output"`
	ExternalClock bool                `json:"external_clock"`
	Channels      []PWMChannelRequest `json:"channels"`
	Cleanup       bool                `json:"cleanup"`
}

// PWMChannelRequest represents configuration for a PWM channel
type PWMChannelRequest struct {
	Channel     int     `json:"channel"`
	DutyCycle   float64 `json:"duty_cycle"`
	PhaseOffset int     `json:"phase_offset"`
	Enabled     bool    `json:"enabled"`
}

// I2CRequest represents a request to configure an I2C device for Kubernetes controllers
type I2CRequest struct {
	NodeID       string               `json:"node_id"`
	Address      string               `json:"address"`
	DeviceType   string               `json:"device_type"`
	BusNumber    int                  `json:"bus_number"`
	DataFormat   string               `json:"data_format"`
	ScanInterval int                  `json:"scan_interval"`
	Registers    []I2CRegisterRequest `json:"registers"`
	Cleanup      bool                 `json:"cleanup"`
}

// I2CRegisterRequest represents configuration for an I2C register
type I2CRegisterRequest struct {
	Address      int    `json:"address"`
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	Size         int    `json:"size"`
	InitialValue *int   `json:"initial_value"`
}

// I2CReadRequest represents a request to read from an I2C device
type I2CReadRequest struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

// GPIOControllerService interface for Kubernetes controller integration
type GPIOControllerService interface {
	ConfigurePin(ctx context.Context, req *GPIORequest) error
}

// PWMControllerService interface for Kubernetes controller integration
type PWMControllerService interface {
	ConfigureController(ctx context.Context, req *PWMRequest) error
}

// I2CControllerService interface for Kubernetes controller integration
type I2CControllerService interface {
	ConfigureDevice(ctx context.Context, req *I2CRequest) error
	ReadDevice(ctx context.Context, req *I2CReadRequest) (map[string]interface{}, error)
}

// NodeControllerService interface for Kubernetes controller integration
type NodeControllerService interface {
	GetByName(ctx context.Context, name string) (interface{}, error)
}

// NodeServiceAdapter adapts the existing NodeService to the controller interface
type NodeServiceAdapter struct {
	nodeService *NodeService
}

// NewNodeServiceAdapter creates a new adapter
func NewNodeServiceAdapter(nodeService *NodeService) *NodeServiceAdapter {
	return &NodeServiceAdapter{nodeService: nodeService}
}

// GetByName adapts the NodeService GetByName method
func (a *NodeServiceAdapter) GetByName(ctx context.Context, name string) (interface{}, error) {
	return a.nodeService.GetByName(name)
}
