package testing

import (
	"database/sql/driver"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dsyorkd/pi-controller/internal/models"
)

// TestDB holds test database connection and mock
type TestDB struct {
	DB   *gorm.DB
	Mock sqlmock.Sqlmock
}

// SetupTestDB creates a test database instance with mocking capabilities
func SetupTestDB(t *testing.T) *TestDB {
	_, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return &TestDB{
		DB:   gormDB,
		Mock: mock,
	}
}

// SetupTestDBFile creates a real SQLite test database file
func SetupTestDBFile(t *testing.T) (*gorm.DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(
		&models.Cluster{},
		&models.Node{},
		&models.GPIODevice{},
		&models.GPIOReading{},
	)
	require.NoError(t, err)

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// CloseTestDB closes the test database and verifies mock expectations
func (tdb *TestDB) Close(t *testing.T) {
	sqlDB, err := tdb.DB.DB()
	require.NoError(t, err)
	sqlDB.Close()
	require.NoError(t, tdb.Mock.ExpectationsWereMet())
}

// AnyTime is a mock argument matcher for any time value
type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// CreateTestCluster creates a test cluster for testing purposes
func CreateTestCluster(t *testing.T) *models.Cluster {
	return &models.Cluster{
		ID:          1,
		Name:        "test-cluster",
		Description: "Test cluster for unit tests",
		Status:      models.ClusterStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestNode creates a test node for testing purposes
func CreateTestNode(t *testing.T, clusterID uint) *models.Node {
	return &models.Node{
		ID:        1,
		Name:      "test-node",
		IPAddress: "192.168.1.100",
		Status:    models.NodeStatusReady,
		ClusterID: &clusterID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestGPIODevice creates a test GPIO device for testing purposes
func CreateTestGPIODevice(t *testing.T, nodeID uint) *models.GPIODevice {
	return &models.GPIODevice{
		ID:          1,
		Name:        "test-gpio",
		Description: "Test GPIO device",
		PinNumber:   18,
		Direction:   models.GPIODirectionOutput,
		PullMode:    models.GPIOPullNone,
		Value:       0,
		DeviceType:  models.GPIODeviceTypeDigital,
		Status:      models.GPIOStatusActive,
		NodeID:      nodeID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestGPIOReading creates a test GPIO reading for testing purposes
func CreateTestGPIOReading(t *testing.T, deviceID uint) *models.GPIOReading {
	return &models.GPIOReading{
		ID:        1,
		DeviceID:  deviceID,
		Value:     1.0,
		Timestamp: time.Now(),
	}
}

// TestConfig returns a test configuration
type TestConfig struct {
	DatabasePath string
	APIPort      int
	GRPCPort     int
	WSPort       int
	MockGPIO     bool
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabasePath: ":memory:",
		APIPort:      0, // Random available port
		GRPCPort:     0, // Random available port
		WSPort:       0, // Random available port
		MockGPIO:     true,
	}
}

// AssertStatusCode is a helper for asserting HTTP status codes
func AssertStatusCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected status code %d, got %d", expected, actual)
	}
}

// AssertNoError is a helper for asserting no error occurred
func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError is a helper for asserting an error occurred
func AssertError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

// GPIOTestPins defines safe test pins that can be used in tests
var GPIOTestPins = []int{18, 19, 20, 21, 22, 23, 24, 25}

// IsTestEnvironment returns true if running in test environment
func IsTestEnvironment() bool {
	return os.Getenv("GO_TEST") != "" || os.Getenv("CI") != ""
}