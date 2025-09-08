package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.nodeId"
// +kubebuilder:printcolumn:name="Pin",type="integer",JSONPath=".spec.pinNumber"
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Current Value",type="string",JSONPath=".status.currentValue"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GPIOPin is the Schema for the gpiopins API
type GPIOPin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GPIOPinSpec   `json:"spec,omitempty"`
	Status GPIOPinStatus `json:"status,omitempty"`
}

// GPIOPinSpec defines the desired state of GPIOPin
type GPIOPinSpec struct {
	// NodeSelector specifies which node this GPIO pin should be managed on
	NodeSelector map[string]string `json:"nodeSelector"`

	// PinNumber is the GPIO pin number on the target node
	PinNumber int `json:"pinNumber"`

	// Mode specifies the pin mode (input, output, pwm)
	Mode GPIOMode `json:"mode"`

	// Direction specifies the pin direction for input/output modes
	Direction *GPIODirection `json:"direction,omitempty"`

	// InitialValue sets the initial pin value for output mode
	InitialValue *GPIOValue `json:"initialValue,omitempty"`

	// PullMode configures internal pull resistors for input mode
	PullMode *GPIOPullMode `json:"pullMode,omitempty"`

	// DebounceMs sets debounce time in milliseconds for input mode
	DebounceMs *int `json:"debounceMs,omitempty"`

	// PWMFrequency sets the PWM frequency in Hz (PWM mode only)
	PWMFrequency *int `json:"pwmFrequency,omitempty"`

	// PWMDutyCycle sets the PWM duty cycle as percentage (0-100)
	PWMDutyCycle *float64 `json:"pwmDutyCycle,omitempty"`
}

// GPIOPinStatus defines the observed state of GPIOPin
type GPIOPinStatus struct {
	// Phase represents the current lifecycle phase
	Phase GPIOPhase `json:"phase,omitempty"`

	// NodeID is the ID of the node managing this pin
	NodeID string `json:"nodeId,omitempty"`

	// CurrentValue is the current pin state (high/low)
	CurrentValue GPIOValue `json:"currentValue,omitempty"`

	// ActualMode is the currently configured pin mode
	ActualMode GPIOMode `json:"actualMode,omitempty"`

	// ActualFrequency is the currently configured PWM frequency
	ActualFrequency *int `json:"actualFrequency,omitempty"`

	// ActualDutyCycle is the currently configured PWM duty cycle
	ActualDutyCycle *float64 `json:"actualDutyCycle,omitempty"`

	// LastUpdated is the timestamp of the last status update
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Conditions represent the current service state
	Conditions []GPIOCondition `json:"conditions,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`
}

// GPIOCondition represents a condition of the GPIO pin
type GPIOCondition struct {
	// Type of condition
	Type GPIOConditionType `json:"type"`

	// Status of the condition (True, False, Unknown)
	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason is the reason for the condition's last transition
	Reason string `json:"reason,omitempty"`

	// Message is a human readable message indicating details
	Message string `json:"message,omitempty"`
}

// GPIOMode defines the possible GPIO pin modes
// +kubebuilder:validation:Enum=input;output;pwm
type GPIOMode string

const (
	GPIOModeInput  GPIOMode = "input"
	GPIOModeOutput GPIOMode = "output"
	GPIOModePWM    GPIOMode = "pwm"
)

// GPIODirection defines the pin direction
// +kubebuilder:validation:Enum=in;out
type GPIODirection string

const (
	GPIODirectionIn  GPIODirection = "in"
	GPIODirectionOut GPIODirection = "out"
)

// GPIOValue defines the pin value
// +kubebuilder:validation:Enum=low;high
type GPIOValue string

const (
	GPIOValueLow  GPIOValue = "low"
	GPIOValueHigh GPIOValue = "high"
)

// GPIOPullMode defines the pull resistor mode
// +kubebuilder:validation:Enum=none;up;down
type GPIOPullMode string

const (
	GPIOPullModeNone GPIOPullMode = "none"
	GPIOPullModeUp   GPIOPullMode = "up"
	GPIOPullModeDown GPIOPullMode = "down"
)

// GPIOPhase defines the lifecycle phase
// +kubebuilder:validation:Enum=Pending;Configuring;Ready;Failed
type GPIOPhase string

const (
	GPIOPhasePending     GPIOPhase = "Pending"
	GPIOPhaseConfiguring GPIOPhase = "Configuring"
	GPIOPhaseReady       GPIOPhase = "Ready"
	GPIOPhaseFailed      GPIOPhase = "Failed"
)

// GPIOConditionType defines the condition types
type GPIOConditionType string

const (
	// GPIOConditionConfigured indicates whether the GPIO pin is properly configured
	GPIOConditionConfigured GPIOConditionType = "Configured"

	// GPIOConditionReady indicates whether the GPIO pin is ready for use
	GPIOConditionReady GPIOConditionType = "Ready"

	// GPIOConditionNodeReachable indicates whether the target node is reachable
	GPIOConditionNodeReachable GPIOConditionType = "NodeReachable"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GPIOPinList contains a list of GPIOPin
type GPIOPinList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GPIOPin `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.nodeId"
// +kubebuilder:printcolumn:name="Channels",type="integer",JSONPath=".spec.channelCount"
// +kubebuilder:printcolumn:name="Frequency",type="integer",JSONPath=".spec.baseFrequency"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PWMController is the Schema for the pwmcontrollers API
type PWMController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PWMControllerSpec   `json:"spec,omitempty"`
	Status PWMControllerStatus `json:"status,omitempty"`
}

// PWMControllerSpec defines the desired state of PWMController
type PWMControllerSpec struct {
	// NodeSelector specifies which node this PWM controller should be managed on
	NodeSelector map[string]string `json:"nodeSelector"`

	// Address is the I2C address of the PWM controller (e.g., 0x40)
	Address int `json:"address"`

	// BaseFrequency sets the base PWM frequency in Hz for all channels
	BaseFrequency int `json:"baseFrequency"`

	// ChannelCount is the number of PWM channels supported by this controller
	ChannelCount int `json:"channelCount"`

	// Channels defines the configuration for individual PWM channels
	Channels []PWMChannel `json:"channels,omitempty"`

	// OutputEnable controls whether outputs are enabled globally
	OutputEnable bool `json:"outputEnable"`

	// InvertOutput inverts all PWM outputs if true
	InvertOutput bool `json:"invertOutput,omitempty"`

	// ExternalClock uses external clock source if true
	ExternalClock bool `json:"externalClock,omitempty"`
}

// PWMChannel defines configuration for an individual PWM channel
type PWMChannel struct {
	// Channel is the channel number (0-15 for PCA9685)
	Channel int `json:"channel"`

	// DutyCycle sets the PWM duty cycle as percentage (0.0-100.0)
	DutyCycle float64 `json:"dutyCycle"`

	// PhaseOffset sets the phase offset for this channel in degrees (0-360)
	PhaseOffset int `json:"phaseOffset,omitempty"`

	// Enabled controls whether this channel is active
	Enabled bool `json:"enabled"`
}

// PWMControllerStatus defines the observed state of PWMController
type PWMControllerStatus struct {
	// Phase represents the current lifecycle phase
	Phase PWMPhase `json:"phase,omitempty"`

	// NodeID is the ID of the node managing this controller
	NodeID string `json:"nodeId,omitempty"`

	// ActualFrequency is the currently configured base frequency
	ActualFrequency int `json:"actualFrequency,omitempty"`

	// ChannelStatus provides status for each channel
	ChannelStatus []PWMChannelStatus `json:"channelStatus,omitempty"`

	// LastUpdated is the timestamp of the last status update
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Conditions represent the current service state
	Conditions []PWMCondition `json:"conditions,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`
}

// PWMChannelStatus defines the status of an individual PWM channel
type PWMChannelStatus struct {
	// Channel is the channel number
	Channel int `json:"channel"`

	// ActualDutyCycle is the currently configured duty cycle
	ActualDutyCycle float64 `json:"actualDutyCycle"`

	// ActualPhaseOffset is the currently configured phase offset
	ActualPhaseOffset int `json:"actualPhaseOffset"`

	// Enabled indicates if this channel is currently active
	Enabled bool `json:"enabled"`
}

// PWMCondition represents a condition of the PWM controller
type PWMCondition struct {
	// Type of condition
	Type PWMConditionType `json:"type"`

	// Status of the condition (True, False, Unknown)
	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason is the reason for the condition's last transition
	Reason string `json:"reason,omitempty"`

	// Message is a human readable message indicating details
	Message string `json:"message,omitempty"`
}

// PWMPhase defines the lifecycle phase for PWM controllers
// +kubebuilder:validation:Enum=Pending;Configuring;Ready;Failed
type PWMPhase string

const (
	PWMPhasePending     PWMPhase = "Pending"
	PWMPhaseConfiguring PWMPhase = "Configuring"
	PWMPhaseReady       PWMPhase = "Ready"
	PWMPhaseFailed      PWMPhase = "Failed"
)

// PWMConditionType defines the condition types for PWM controllers
type PWMConditionType string

const (
	// PWMConditionConfigured indicates whether the PWM controller is properly configured
	PWMConditionConfigured PWMConditionType = "Configured"

	// PWMConditionReady indicates whether the PWM controller is ready for use
	PWMConditionReady PWMConditionType = "Ready"

	// PWMConditionNodeReachable indicates whether the target node is reachable
	PWMConditionNodeReachable PWMConditionType = "NodeReachable"

	// PWMConditionI2CConnected indicates whether the I2C device is accessible
	PWMConditionI2CConnected PWMConditionType = "I2CConnected"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PWMControllerList contains a list of PWMController
type PWMControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PWMController `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.nodeId"
// +kubebuilder:printcolumn:name="Address",type="string",JSONPath=".spec.address"
// +kubebuilder:printcolumn:name="Device Type",type="string",JSONPath=".spec.deviceType"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// I2CDevice is the Schema for the i2cdevices API
type I2CDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   I2CDeviceSpec   `json:"spec,omitempty"`
	Status I2CDeviceStatus `json:"status,omitempty"`
}

// I2CDeviceSpec defines the desired state of I2CDevice
type I2CDeviceSpec struct {
	// NodeSelector specifies which node this I2C device should be managed on
	NodeSelector map[string]string `json:"nodeSelector"`

	// Address is the I2C address of the device (e.g., "0x48")
	Address string `json:"address"`

	// DeviceType identifies the type of I2C device
	DeviceType I2CDeviceType `json:"deviceType"`

	// BusNumber specifies the I2C bus number (default: 1)
	BusNumber int `json:"busNumber,omitempty"`

	// Registers defines the register configuration for the device
	Registers []I2CRegister `json:"registers,omitempty"`

	// DataFormat specifies how to interpret device data
	DataFormat I2CDataFormat `json:"dataFormat,omitempty"`

	// ScanInterval sets how often to read from the device (in seconds)
	ScanInterval int `json:"scanInterval,omitempty"`
}

// I2CRegister defines configuration for reading/writing I2C device registers
type I2CRegister struct {
	// Address is the register address
	Address int `json:"address"`

	// Name is a human-readable name for this register
	Name string `json:"name"`

	// Mode defines whether this register is readable, writable, or both
	Mode I2CRegisterMode `json:"mode"`

	// Size is the register size in bytes (1, 2, or 4)
	Size int `json:"size"`

	// InitialValue sets the initial value to write (for writable registers)
	InitialValue *int `json:"initialValue,omitempty"`
}

// I2CDeviceStatus defines the observed state of I2CDevice
type I2CDeviceStatus struct {
	// Phase represents the current lifecycle phase
	Phase I2CPhase `json:"phase,omitempty"`

	// NodeID is the ID of the node managing this device
	NodeID string `json:"nodeId,omitempty"`

	// RegisterData contains the latest data read from device registers
	RegisterData map[string]interface{} `json:"registerData,omitempty"`

	// LastScan is the timestamp of the last successful device scan
	LastScan *metav1.Time `json:"lastScan,omitempty"`

	// LastUpdated is the timestamp of the last status update
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// Conditions represent the current service state
	Conditions []I2CCondition `json:"conditions,omitempty"`

	// Message provides additional information about the current state
	Message string `json:"message,omitempty"`
}

// I2CCondition represents a condition of the I2C device
type I2CCondition struct {
	// Type of condition
	Type I2CConditionType `json:"type"`

	// Status of the condition (True, False, Unknown)
	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason is the reason for the condition's last transition
	Reason string `json:"reason,omitempty"`

	// Message is a human readable message indicating details
	Message string `json:"message,omitempty"`
}

// I2CDeviceType defines the supported I2C device types
// +kubebuilder:validation:Enum=sensor;display;pwm;gpio;adc;dac;rtc;eeprom;other
type I2CDeviceType string

const (
	I2CDeviceTypeSensor  I2CDeviceType = "sensor"
	I2CDeviceTypeDisplay I2CDeviceType = "display"
	I2CDeviceTypePWM     I2CDeviceType = "pwm"
	I2CDeviceTypeGPIO    I2CDeviceType = "gpio"
	I2CDeviceTypeADC     I2CDeviceType = "adc"
	I2CDeviceTypeDAC     I2CDeviceType = "dac"
	I2CDeviceTypeRTC     I2CDeviceType = "rtc"
	I2CDeviceTypeEEPROM  I2CDeviceType = "eeprom"
	I2CDeviceTypeOther   I2CDeviceType = "other"
)

// I2CDataFormat defines how device data should be interpreted
// +kubebuilder:validation:Enum=raw;integer;float;string;json
type I2CDataFormat string

const (
	I2CDataFormatRaw     I2CDataFormat = "raw"
	I2CDataFormatInteger I2CDataFormat = "integer"
	I2CDataFormatFloat   I2CDataFormat = "float"
	I2CDataFormatString  I2CDataFormat = "string"
	I2CDataFormatJSON    I2CDataFormat = "json"
)

// I2CRegisterMode defines the access mode for device registers
// +kubebuilder:validation:Enum=read;write;readwrite
type I2CRegisterMode string

const (
	I2CRegisterModeRead      I2CRegisterMode = "read"
	I2CRegisterModeWrite     I2CRegisterMode = "write"
	I2CRegisterModeReadWrite I2CRegisterMode = "readwrite"
)

// I2CPhase defines the lifecycle phase for I2C devices
// +kubebuilder:validation:Enum=Pending;Configuring;Ready;Failed
type I2CPhase string

const (
	I2CPhasePending     I2CPhase = "Pending"
	I2CPhaseConfiguring I2CPhase = "Configuring"
	I2CPhaseReady       I2CPhase = "Ready"
	I2CPhaseFailed      I2CPhase = "Failed"
)

// I2CConditionType defines the condition types for I2C devices
type I2CConditionType string

const (
	// I2CConditionConfigured indicates whether the I2C device is properly configured
	I2CConditionConfigured I2CConditionType = "Configured"

	// I2CConditionReady indicates whether the I2C device is ready for use
	I2CConditionReady I2CConditionType = "Ready"

	// I2CConditionNodeReachable indicates whether the target node is reachable
	I2CConditionNodeReachable I2CConditionType = "NodeReachable"

	// I2CConditionDeviceConnected indicates whether the I2C device is accessible
	I2CConditionDeviceConnected I2CConditionType = "DeviceConnected"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// I2CDeviceList contains a list of I2CDevice
type I2CDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []I2CDevice `json:"items"`
}
