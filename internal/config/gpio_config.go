package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SystemCriticalPins are GPIO pins that should NEVER be exposed for user control
// These are typically used for essential system functions like I2C, UART, etc.
var SystemCriticalPins = map[int]string{
	0:  "I2C0 SDA - System I2C data line",
	1:  "I2C0 SCL - System I2C clock line",
	2:  "I2C1 SDA - Alternate I2C data line",
	3:  "I2C1 SCL - Alternate I2C clock line",
	14: "UART TXD - Serial transmit",
	15: "UART RXD - Serial receive",
}

// DefaultSafeGPIOPins returns the default list of safe GPIO pins for Raspberry Pi
// These are general-purpose pins not reserved for critical functions
func DefaultSafeGPIOPins() []int {
	return []int{
		4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 17, 18,
		19, 20, 21, 22, 23, 24, 25, 26, 27,
	}
}

// GetMaxPinNumber returns the maximum GPIO pin number for Raspberry Pi (BCM numbering)
func GetMaxPinNumber() int {
	return 27
}

// LoadGPIOSafelistConfig loads GPIO safelist from environment or returns defaults
func LoadGPIOSafelistConfig() []int {
	// Start with defaults
	safelist := DefaultSafeGPIOPins()

	// Allow override from environment variable
	if pinsEnv := os.Getenv("GPIO_SAFE_PINS"); pinsEnv != "" {
		pins, err := parsePinList(pinsEnv)
		if err == nil {
			safelist = pins
		}
	}

	return safelist
}

// IsPinSafelistEnabled returns whether GPIO pin safelisting is enabled
func IsPinSafelistEnabled() bool {
	// Allow disabling safelist (NOT RECOMMENDED for production)
	if disableEnv := os.Getenv("GPIO_DISABLE_SAFELIST"); disableEnv == "true" {
		return false
	}
	return true
}

// parsePinList parses a comma-separated list of pin numbers
func parsePinList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	pins := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		pin, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid pin number: %s", part)
		}

		pins = append(pins, pin)
	}

	return pins, nil
}

// IsPinSafe checks if a GPIO pin is safe for user control
func IsPinSafe(pinNumber int, safelist []int) (bool, string) {
	maxPin := GetMaxPinNumber()

	// Validate pin number is within valid range
	if pinNumber < 0 {
		return false, "pin number cannot be negative"
	}

	if pinNumber > maxPin {
		return false, fmt.Sprintf("pin number %d exceeds maximum allowed pin %d", pinNumber, maxPin)
	}

	// Check if pin is in system critical pins
	if reason, isCritical := SystemCriticalPins[pinNumber]; isCritical {
		return false, fmt.Sprintf("pin %d is reserved for system use: %s", pinNumber, reason)
	}

	// If safelist is disabled, allow all non-critical pins (NOT RECOMMENDED)
	if !IsPinSafelistEnabled() {
		return true, ""
	}

	// Check if pin is in the safelist
	for _, safePin := range safelist {
		if pinNumber == safePin {
			return true, ""
		}
	}

	return false, fmt.Sprintf("pin %d is not in the safelist of allowed GPIO pins", pinNumber)
}

// ValidateGPIOPin validates a pin number and returns a user-friendly error
func ValidateGPIOPin(pinNumber int) error {
	safelist := LoadGPIOSafelistConfig()
	safe, reason := IsPinSafe(pinNumber, safelist)
	if !safe {
		return fmt.Errorf("GPIO pin %d cannot be used: %s", pinNumber, reason)
	}
	return nil
}

// GetSystemCriticalPins returns a map of system critical pins and their purposes
func GetSystemCriticalPins() map[int]string {
	// Return a copy to prevent modification
	critical := make(map[int]string, len(SystemCriticalPins))
	for k, v := range SystemCriticalPins {
		critical[k] = v
	}
	return critical
}
