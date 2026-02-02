package defaults

// GPIO defaults
const (
	// GPIOEnabled indicates whether GPIO is enabled
	GPIOEnabled = true

	// GPIOMockMode indicates whether mock mode is enabled
	GPIOMockMode = false

	// GPIOSampleInterval is the default sampling interval
	GPIOSampleInterval = "1s"

	// GPIORetentionPeriod is the default data retention period
	GPIORetentionPeriod = "24h"

	// GPIODefaultDirection is the default pin direction
	GPIODefaultDirection = "input"

	// GPIODefaultPullMode is the default pull mode
	GPIODefaultPullMode = "none"
)

// GPIOAllowedPins contains safe GPIO pins that can be controlled
var GPIOAllowedPins = []int{2, 3, 4, 17, 27, 22, 10, 9, 11, 5, 6, 13, 19, 26, 18, 23, 24, 25, 8, 7, 12, 16, 20, 21}

// GPIORestrictedPins contains system-critical pins (I2C, UART) that are restricted
var GPIORestrictedPins = []int{0, 1, 14, 15}
