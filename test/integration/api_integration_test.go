package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/api/handlers"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	testutils "github.com/dsyorkd/pi-controller/internal/testing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// APIIntegrationTestSuite defines the test suite for API integration tests
type APIIntegrationTestSuite struct {
	suite.Suite
	router         *gin.Engine
	db             *storage.Database
	cleanup        func()
	clusterService *services.ClusterService
	nodeService    *services.NodeService
	gpioService    *services.GPIOService
	mockManager    *MockPiAgentClientManager
}

// SetupSuite sets up the test suite
func (suite *APIIntegrationTestSuite) SetupSuite() {
	// Set environment to development for testing (disables HTTPS requirement)
	os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")

	db, cleanup := testutils.SetupTestDBFile(suite.T())
	suite.db = storage.NewForTestWithDB(db, logger.Default())
	suite.cleanup = cleanup

	testLogger := logger.Default()

	// Initialize mock manager for GPIO testing
	suite.mockManager = NewMockPiAgentClientManager()

	// Initialize services
	suite.clusterService = services.NewClusterService(suite.db, testLogger)
	suite.nodeService = services.NewNodeService(suite.db, testLogger)
	suite.gpioService = services.NewGPIOServiceWithManager(suite.db, testLogger, suite.mockManager)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(suite.db)
	clusterHandler := handlers.NewClusterHandler(suite.clusterService, testLogger)
	nodeHandler := handlers.NewNodeHandler(suite.nodeService, testLogger)
	gpioHandler := handlers.NewGPIOHandler(suite.gpioService, testLogger)

	// Setup router
	suite.router = gin.New()

	// Add NoMethod handler to return 405 for unsupported methods
	suite.router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
	})

	// Health endpoints
	suite.router.GET("/health", healthHandler.Health)
	suite.router.GET("/ready", healthHandler.Ready)
	suite.router.GET("/system/info", handlers.SystemInfo)
	suite.router.GET("/system/metrics", handlers.SystemMetrics)

	// API v1 routes
	v1 := suite.router.Group("/api/v1")
	{
		// Cluster routes
		clusters := v1.Group("/clusters")
		{
			clusters.GET("", clusterHandler.List)
			clusters.POST("", clusterHandler.Create)
			clusters.GET("/:id", clusterHandler.Get)
			clusters.PUT("/:id", clusterHandler.Update)
			clusters.DELETE("/:id", clusterHandler.Delete)
			// Handle unsupported methods explicitly
			clusters.PATCH("/:id", func(c *gin.Context) {
				c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
			})
		}

		// Node routes
		nodes := v1.Group("/nodes")
		{
			nodes.GET("", nodeHandler.List)
			nodes.POST("", nodeHandler.Create)
			nodes.GET("/:id", nodeHandler.Get)
			nodes.PUT("/:id", nodeHandler.Update)
			nodes.DELETE("/:id", nodeHandler.Delete)
		}

		// GPIO routes
		gpio := v1.Group("/gpio")
		{
			gpio.GET("", gpioHandler.List)
			gpio.POST("", gpioHandler.Create)
			gpio.GET("/:id", gpioHandler.Get)
			gpio.PUT("/:id", gpioHandler.Update)
			gpio.DELETE("/:id", gpioHandler.Delete)
			gpio.GET("/:id/read", gpioHandler.Read)
			gpio.POST("/:id/write", gpioHandler.Write)
			gpio.GET("/:id/readings", gpioHandler.GetReadings)
		}
	}
}

// SetupTest runs before each test to ensure clean database state
func (suite *APIIntegrationTestSuite) SetupTest() {
	// Clear all tables to ensure clean state for each test
	suite.db.DB().Exec("DELETE FROM gpio_readings")
	suite.db.DB().Exec("DELETE FROM gpio_devices")
	suite.db.DB().Exec("DELETE FROM nodes")
	suite.db.DB().Exec("DELETE FROM clusters")
	suite.db.DB().Exec("DELETE FROM certificate_requests")
	suite.db.DB().Exec("DELETE FROM certificates")
	suite.db.DB().Exec("DELETE FROM ca_info")
	suite.db.DB().Exec("DELETE FROM users")
}

// TearDownSuite cleans up after the test suite
func (suite *APIIntegrationTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

// TestAPIIntegration_HealthEndpoints tests all health-related endpoints
func (suite *APIIntegrationTestSuite) TestAPIIntegration_HealthEndpoints() {
	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{"Health", "/health", http.StatusOK},
		{"Ready", "/ready", http.StatusOK},
		{"System Info", "/system/info", http.StatusOK},
		{"System Metrics", "/system/metrics", http.StatusOK},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req, err := http.NewRequest("GET", tt.endpoint, nil)
			require.NoError(suite.T(), err)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)
			assert.NotEmpty(suite.T(), w.Body.String())
		})
	}
}

// TestAPIIntegration_ClusterWorkflow tests the complete cluster workflow
func (suite *APIIntegrationTestSuite) TestAPIIntegration_ClusterWorkflow() {
	// 1. List clusters (should be empty initially)
	req, err := http.NewRequest("GET", "/api/v1/clusters", nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var listResponse struct {
		Data  []models.Cluster `json:"data"`
		Total int64            `json:"total"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), listResponse.Data, 0)

	// 2. Create a cluster
	createReq := services.CreateClusterRequest{
		Name:        "integration-test-cluster",
		Description: "Cluster for integration testing",
	}

	body, err := json.Marshal(createReq)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("POST", "/api/v1/clusters", bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var createResponse struct {
		Data models.Cluster `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)

	clusterID := createResponse.Data.ID
	assert.NotZero(suite.T(), clusterID)
	assert.Equal(suite.T(), createReq.Name, createResponse.Data.Name)
	assert.Equal(suite.T(), models.ClusterStatusActive, createResponse.Data.Status)

	// 3. Get the cluster by ID
	req, err = http.NewRequest("GET", fmt.Sprintf("/api/v1/clusters/%d", clusterID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var getResponse struct {
		Data models.Cluster `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), clusterID, getResponse.Data.ID)
	assert.Equal(suite.T(), createReq.Name, getResponse.Data.Name)

	// 4. Update the cluster
	updateReq := services.UpdateClusterRequest{
		Description: stringPtr("Updated description"),
	}

	body, err = json.Marshal(updateReq)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("PUT", fmt.Sprintf("/api/v1/clusters/%d", clusterID), bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var updateResponse struct {
		Data models.Cluster `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &updateResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), *updateReq.Description, updateResponse.Data.Description)

	// 5. List clusters (should now contain 1)
	req, err = http.NewRequest("GET", "/api/v1/clusters", nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), listResponse.Data, 1)
	assert.Equal(suite.T(), int64(1), listResponse.Total)

	// 6. Delete the cluster
	req, err = http.NewRequest("DELETE", fmt.Sprintf("/api/v1/clusters/%d", clusterID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// 7. Verify cluster is deleted
	req, err = http.NewRequest("GET", fmt.Sprintf("/api/v1/clusters/%d", clusterID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

// TestAPIIntegration_NodeWorkflow tests the complete node workflow
func (suite *APIIntegrationTestSuite) TestAPIIntegration_NodeWorkflow() {
	// First create a cluster to associate nodes with
	cluster := testutils.CreateTestCluster(suite.T())
	require.NoError(suite.T(), suite.db.DB().Create(cluster).Error)

	// 1. Create a node
	createReq := services.CreateNodeRequest{
		Name:       "integration-test-node",
		IPAddress:  "192.168.1.100",
		MACAddress: "02:00:00:00:01:00",
		Role:       models.NodeRoleWorker,
		ClusterID:  &cluster.ID,
	}

	body, err := json.Marshal(createReq)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/nodes", bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var createResponse struct {
		Data models.Node `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)

	nodeID := createResponse.Data.ID
	assert.NotZero(suite.T(), nodeID)
	assert.Equal(suite.T(), createReq.Name, createResponse.Data.Name)
	assert.Equal(suite.T(), createReq.IPAddress, createResponse.Data.IPAddress)
	assert.Equal(suite.T(), models.NodeStatusDiscovered, createResponse.Data.Status)

	// 2. Get the node by ID
	req, err = http.NewRequest("GET", fmt.Sprintf("/api/v1/nodes/%d", nodeID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 3. Update the node
	updateReq := services.UpdateNodeRequest{
		Name: stringPtr("updated-integration-test-node"),
	}

	body, err = json.Marshal(updateReq)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("PUT", fmt.Sprintf("/api/v1/nodes/%d", nodeID), bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 4. List nodes
	req, err = http.NewRequest("GET", "/api/v1/nodes", nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var listResponse struct {
		Data  []models.Node `json:"data"`
		Total int64         `json:"total"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), listResponse.Data, 1)

	// 5. Delete the node
	req, err = http.NewRequest("DELETE", fmt.Sprintf("/api/v1/nodes/%d", nodeID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)
}

// TestAPIIntegration_GPIOWorkflow tests the complete GPIO workflow
func (suite *APIIntegrationTestSuite) TestAPIIntegration_GPIOWorkflow() {
	// Setup test data
	cluster := testutils.CreateTestCluster(suite.T())
	require.NoError(suite.T(), suite.db.DB().Create(cluster).Error)

	node := testutils.CreateTestNode(suite.T(), cluster.ID)
	require.NoError(suite.T(), suite.db.DB().Create(node).Error)

	// Setup mock expectations for GPIO operations
	suite.mockManager.On("GetClient", mock.AnythingOfType("*models.Node")).Return(suite.mockManager.mockClient, nil)
	suite.mockManager.mockClient.On("IsConnected").Return(true)
	suite.mockManager.mockClient.On("ConfigureGPIOPin", mock.Anything, mock.AnythingOfType("*models.GPIODevice")).Return(nil)
	suite.mockManager.mockClient.On("WriteGPIOPin", mock.Anything, 18, 1).Return(nil)
	suite.mockManager.mockClient.On("ReadGPIOPin", mock.Anything, 18).Return(1, nil)

	// 1. Create a GPIO device
	createReq := services.CreateGPIODeviceRequest{
		Name:        "integration-test-gpio",
		Description: "GPIO device for integration testing",
		NodeID:      node.ID,
		PinNumber:   18,
		Direction:   models.GPIODirectionOutput,
		PullMode:    models.GPIOPullNone,
		DeviceType:  models.GPIODeviceTypeDigital,
	}

	body, err := json.Marshal(createReq)
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", "/api/v1/gpio", bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var createResponse struct {
		Data models.GPIODevice `json:"data"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(suite.T(), err)

	deviceID := createResponse.Data.ID
	assert.NotZero(suite.T(), deviceID)
	assert.Equal(suite.T(), createReq.Name, createResponse.Data.Name)
	assert.Equal(suite.T(), createReq.PinNumber, createResponse.Data.PinNumber)

	// 2. Write to GPIO device
	writeReq := struct {
		Value int `json:"value"`
	}{Value: 1}

	body, err = json.Marshal(writeReq)
	require.NoError(suite.T(), err)

	req, err = http.NewRequest("POST", fmt.Sprintf("/api/v1/gpio/%d/write", deviceID), bytes.NewBuffer(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 3. Read from GPIO device
	req, err = http.NewRequest("GET", fmt.Sprintf("/api/v1/gpio/%d/read", deviceID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 4. Get GPIO readings
	req, err = http.NewRequest("GET", fmt.Sprintf("/api/v1/gpio/%d/readings", deviceID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var readingsResponse struct {
		Data  []models.GPIOReading `json:"data"`
		Total int64                `json:"total"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &readingsResponse)
	require.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(readingsResponse.Data), 1) // Should have at least the write + read

	// 5. List GPIO devices
	req, err = http.NewRequest("GET", "/api/v1/gpio", nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 6. Delete GPIO device
	req, err = http.NewRequest("DELETE", fmt.Sprintf("/api/v1/gpio/%d", deviceID), nil)
	require.NoError(suite.T(), err)

	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)
}

// TestAPIIntegration_Security_NoAuth tests security without authentication
func (suite *APIIntegrationTestSuite) TestAPIIntegration_Security_NoAuth() {
	securityTests := []struct {
		name           string
		method         string
		endpoint       string
		description    string
		expectedStatus int
	}{
		{
			name:           "Unprotected health endpoint",
			method:         "GET",
			endpoint:       "/health",
			description:    "Health endpoint should be accessible without auth",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unprotected API endpoint",
			method:         "GET",
			endpoint:       "/api/v1/clusters",
			description:    "API endpoints are currently unprotected - SECURITY RISK",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Dangerous DELETE without auth",
			method:         "DELETE",
			endpoint:       "/api/v1/clusters/1",
			description:    "DELETE operations without auth - CRITICAL SECURITY RISK",
			expectedStatus: http.StatusNotFound, // Would be 204 if cluster existed
		},
		{
			name:           "System info disclosure",
			method:         "GET",
			endpoint:       "/system/info",
			description:    "System info exposed without auth - potential info disclosure",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range securityTests {
		suite.Run(tt.name, func() {
			req, err := http.NewRequest(tt.method, tt.endpoint, nil)
			require.NoError(suite.T(), err)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code)

			if tt.description != "" {
				suite.T().Logf("Security Issue: %s", tt.description)
			}
		})
	}
}

// TestAPIIntegration_ErrorHandling tests error handling across endpoints
func (suite *APIIntegrationTestSuite) TestAPIIntegration_ErrorHandling() {
	errorTests := []struct {
		name           string
		method         string
		endpoint       string
		body           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Invalid JSON in request",
			method:         "POST",
			endpoint:       "/api/v1/clusters",
			body:           `{"name": invalid json}`,
			expectedStatus: http.StatusBadRequest,
			description:    "Should handle malformed JSON gracefully",
		},
		{
			name:           "Missing required fields",
			method:         "POST",
			endpoint:       "/api/v1/clusters",
			body:           `{"description": "Missing name field"}`,
			expectedStatus: http.StatusBadRequest,
			description:    "Should validate required fields",
		},
		{
			name:           "Non-existent resource",
			method:         "GET",
			endpoint:       "/api/v1/clusters/99999",
			expectedStatus: http.StatusNotFound,
			description:    "Should return 404 for non-existent resources",
		},
		{
			name:           "Invalid ID format",
			method:         "GET",
			endpoint:       "/api/v1/clusters/invalid-id",
			expectedStatus: http.StatusBadRequest,
			description:    "Should validate ID format",
		},
		{
			name:           "Unsupported method",
			method:         "PATCH",
			endpoint:       "/api/v1/clusters/1",
			expectedStatus: http.StatusMethodNotAllowed,
			description:    "Should return 405 for unsupported methods",
		},
	}

	for _, tt := range errorTests {
		suite.Run(tt.name, func() {
			var req *http.Request
			var err error

			if tt.body != "" {
				req, err = http.NewRequest(tt.method, tt.endpoint, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, tt.endpoint, nil)
			}
			require.NoError(suite.T(), err)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tt.expectedStatus, w.Code, tt.description)
		})
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

// TestAPIIntegration runs the integration test suite
func TestAPIIntegration(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}
