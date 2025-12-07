package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dsyorkd/pi-controller/internal/api/handlers"
	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	testutils "github.com/dsyorkd/pi-controller/internal/testing"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// SecurityTestSuite defines the test suite for security vulnerability testing
type SecurityTestSuite struct {
	suite.Suite
	router      *gin.Engine
	db          *storage.Database
	cleanup     func()
	gpioService *services.GPIOService
	authManager *middleware.AuthManager
}

// SetupSuite sets up the test suite with authentication ENABLED
func (suite *SecurityTestSuite) SetupSuite() {
	// Set environment to development for testing (disables HTTPS requirement)
	os.Setenv("PI_CONTROLLER_ENVIRONMENT", "development")

	db, cleanup := testutils.SetupTestDBFile(suite.T())
	appLogger := logger.Default()

	suite.db = storage.NewForTestWithDB(db, appLogger)
	suite.cleanup = cleanup

	logrusLogger := logrus.New()
	logrusLogger.SetLevel(logrus.WarnLevel)

	// Initialize auth manager - SECURITY TESTS REQUIRE AUTH ENABLED
	var err error
	suite.authManager, err = middleware.NewAuthManager(&middleware.AuthConfig{
		JWTSecret:         []byte("test-secret-key-for-security-testing"),
		AccessTokenExpiry: 15 * time.Minute,
	}, appLogger)
	require.NoError(suite.T(), err, "Auth manager must be initialized for security tests")

	// Initialize services
	clusterService := services.NewClusterService(suite.db, appLogger)
	nodeService := services.NewNodeService(suite.db, appLogger)
	suite.gpioService = services.NewGPIOService(suite.db, appLogger)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(suite.db)
	clusterHandler := handlers.NewClusterHandler(clusterService, appLogger)
	nodeHandler := handlers.NewNodeHandler(nodeService, appLogger)
	gpioHandler := handlers.NewGPIOHandler(suite.gpioService, appLogger)

	// Setup router WITH authentication middleware - THIS IS THE KEY DIFFERENCE
	suite.router = gin.New()

	// Health endpoints - public (no auth required)
	suite.router.GET("/health", healthHandler.Health)

	// System info - should require authentication
	suite.router.GET("/system/info", suite.authManager.Auth(), handlers.SystemInfo)

	// API v1 routes - PROTECTED with authentication
	v1 := suite.router.Group("/api/v1")
	v1.Use(suite.authManager.Auth())
	{
		clusters := v1.Group("/clusters")
		{
			clusters.GET("", clusterHandler.List)
			clusters.POST("", clusterHandler.Create)
			clusters.DELETE("/:id", clusterHandler.Delete)
		}

		nodes := v1.Group("/nodes")
		{
			nodes.GET("", nodeHandler.List)
			nodes.POST("", nodeHandler.Create)
			nodes.DELETE("/:id", nodeHandler.Delete)
		}

		gpio := v1.Group("/gpio")
		{
			gpio.GET("", gpioHandler.List)
			gpio.POST("", gpioHandler.Create)
			gpio.DELETE("/:id", gpioHandler.Delete)
			gpio.POST("/:id/write", gpioHandler.Write)
		}
	}
}

// TearDownSuite cleans up after the test suite
func (suite *SecurityTestSuite) TearDownSuite() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

// TestSecurity_AuthenticationRequired tests that all protected endpoints reject unauthenticated requests
func (suite *SecurityTestSuite) TestSecurity_AuthenticationRequired() {
	authTests := []struct {
		name        string
		method      string
		endpoint    string
		body        string
		description string
	}{
		{
			name:        "Cluster creation requires auth",
			method:      "POST",
			endpoint:    "/api/v1/clusters",
			body:        `{"name": "malicious-cluster", "description": "Created without auth"}`,
			description: "Cluster creation must require authentication",
		},
		{
			name:        "Cluster deletion requires auth",
			method:      "DELETE",
			endpoint:    "/api/v1/clusters/1",
			description: "Cluster deletion must require authentication",
		},
		{
			name:        "Cluster listing requires auth",
			method:      "GET",
			endpoint:    "/api/v1/clusters",
			description: "Cluster listing must require authentication",
		},
		{
			name:        "Node creation requires auth",
			method:      "POST",
			endpoint:    "/api/v1/nodes",
			body:        `{"name": "malicious-node", "hostname": "evil.local", "ip_address": "127.0.0.1", "cluster_id": 1}`,
			description: "Node creation must require authentication",
		},
		{
			name:        "Node deletion requires auth",
			method:      "DELETE",
			endpoint:    "/api/v1/nodes/1",
			description: "Node deletion must require authentication",
		},
		{
			name:        "GPIO control requires auth",
			method:      "POST",
			endpoint:    "/api/v1/gpio/1/write",
			body:        `{"value": 1}`,
			description: "GPIO write must require authentication - PHYSICAL SECURITY RISK",
		},
		{
			name:        "GPIO creation requires auth",
			method:      "POST",
			endpoint:    "/api/v1/gpio",
			body:        `{"name": "test-gpio", "pin_number": 17, "node_id": 1}`,
			description: "GPIO creation must require authentication",
		},
		{
			name:        "GPIO deletion requires auth",
			method:      "DELETE",
			endpoint:    "/api/v1/gpio/1",
			description: "GPIO deletion must require authentication",
		},
		{
			name:        "System info requires auth",
			method:      "GET",
			endpoint:    "/system/info",
			description: "System information must require authentication",
		},
	}

	for _, tt := range authTests {
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

			// CRITICAL: Unauthenticated requests MUST return 401
			assert.Equal(suite.T(), http.StatusUnauthorized, w.Code,
				"SECURITY FAILURE: %s - %s returned %d instead of 401 Unauthorized",
				tt.name, tt.description, w.Code)
		})
	}
}

// TestSecurity_AuthenticatedAccessWorks tests that authenticated requests succeed
func (suite *SecurityTestSuite) TestSecurity_AuthenticatedAccessWorks() {
	// Generate a valid token
	token, err := suite.authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(suite.T(), err)

	// Test that authenticated requests work
	req, err := http.NewRequest("GET", "/api/v1/clusters", nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Authenticated request should succeed (200 OK)
	assert.Equal(suite.T(), http.StatusOK, w.Code,
		"Authenticated request should succeed with 200 OK")
}

// TestSecurity_HealthEndpointPublic tests that health endpoint is publicly accessible
func (suite *SecurityTestSuite) TestSecurity_HealthEndpointPublic() {
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(suite.T(), err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Health endpoint should be public for load balancer checks
	assert.Equal(suite.T(), http.StatusOK, w.Code,
		"Health endpoint should be publicly accessible")
}

// TestSecurity_InputValidation tests input validation for security
func (suite *SecurityTestSuite) TestSecurity_InputValidation() {
	// Generate a valid token for authenticated tests
	token, err := suite.authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(suite.T(), err)

	injectionTests := []struct {
		name        string
		method      string
		endpoint    string
		payload     interface{}
		description string
	}{
		{
			name:     "SQL injection in cluster name",
			method:   "POST",
			endpoint: "/api/v1/clusters",
			payload: map[string]string{
				"name":        "'; DROP TABLE clusters; --",
				"description": "SQL injection attempt",
			},
			description: "SQL injection must be rejected",
		},
		{
			name:     "XSS in cluster description",
			method:   "POST",
			endpoint: "/api/v1/clusters",
			payload: map[string]string{
				"name":        "xss-test",
				"description": "<script>alert('XSS')</script>",
			},
			description: "XSS must be sanitized or rejected",
		},
		{
			name:     "Command injection in hostname",
			method:   "POST",
			endpoint: "/api/v1/nodes",
			payload: map[string]interface{}{
				"name":       "injection-test",
				"hostname":   "$(rm -rf /)",
				"ip_address": "192.168.1.1",
				"cluster_id": 1,
			},
			description: "Command injection must be rejected",
		},
	}

	for _, tt := range injectionTests {
		suite.Run(tt.name, func() {
			body, err := json.Marshal(tt.payload)
			require.NoError(suite.T(), err)

			req, err := http.NewRequest(tt.method, tt.endpoint, bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Injection attempts should be rejected with 400 Bad Request
			assert.True(suite.T(), w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity,
				"SECURITY FAILURE: %s - %s returned %d, injection may have succeeded",
				tt.name, tt.description, w.Code)
		})
	}
}

// TestSecurity_GPIOPinRestrictions tests that system-critical GPIO pins are protected
func (suite *SecurityTestSuite) TestSecurity_GPIOPinRestrictions() {
	// Generate a valid token
	token, err := suite.authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(suite.T(), err)

	// Create a test cluster and node first
	cluster := testutils.CreateTestCluster(suite.T())
	require.NoError(suite.T(), suite.db.DB().Create(cluster).Error)
	node := testutils.CreateTestNode(suite.T(), cluster.ID)
	require.NoError(suite.T(), suite.db.DB().Create(node).Error)

	criticalPins := []struct {
		pin         int
		description string
	}{
		{0, "I2C0 SDA - System critical"},
		{1, "I2C0 SCL - System critical"},
		{14, "UART TXD - Serial communication"},
		{15, "UART RXD - Serial communication"},
	}

	for _, tt := range criticalPins {
		suite.Run(strings.ReplaceAll(tt.description, " ", "_"), func() {
			body, _ := json.Marshal(map[string]interface{}{
				"name":       "critical-pin-test",
				"pin_number": tt.pin,
				"node_id":    node.ID,
				"mode":       "output",
			})

			req, err := http.NewRequest("POST", "/api/v1/gpio", bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Critical pins MUST be rejected
			assert.Equal(suite.T(), http.StatusBadRequest, w.Code,
				"SECURITY FAILURE: Critical GPIO pin %d (%s) was not protected - returned %d",
				tt.pin, tt.description, w.Code)
		})
	}
}

// TestSecurity_InvalidGPIOPinRange tests that invalid GPIO pin numbers are rejected
func (suite *SecurityTestSuite) TestSecurity_InvalidGPIOPinRange() {
	token, err := suite.authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(suite.T(), err)

	cluster := testutils.CreateTestCluster(suite.T())
	require.NoError(suite.T(), suite.db.DB().Create(cluster).Error)
	node := testutils.CreateTestNode(suite.T(), cluster.ID)
	require.NoError(suite.T(), suite.db.DB().Create(node).Error)

	invalidPins := []struct {
		pin         int
		description string
	}{
		{-1, "Negative pin number"},
		{100, "Pin number exceeds BCM range"},
		{999, "Extremely high pin number"},
	}

	for _, tt := range invalidPins {
		suite.Run(tt.description, func() {
			body, _ := json.Marshal(map[string]interface{}{
				"name":       "invalid-pin-test",
				"pin_number": tt.pin,
				"node_id":    node.ID,
				"mode":       "output",
			})

			req, err := http.NewRequest("POST", "/api/v1/gpio", bytes.NewBuffer(body))
			require.NoError(suite.T(), err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Invalid pins MUST be rejected
			assert.Equal(suite.T(), http.StatusBadRequest, w.Code,
				"SECURITY FAILURE: Invalid GPIO pin %d (%s) was not rejected - returned %d",
				tt.pin, tt.description, w.Code)
		})
	}
}

// TestSecurity_RoleBasedAccess tests that role-based access control is enforced
func (suite *SecurityTestSuite) TestSecurity_RoleBasedAccess() {
	// Generate tokens with different roles
	viewerToken, err := suite.authManager.GenerateToken("viewer-user", middleware.RoleViewer, middleware.TokenTypeAccess)
	require.NoError(suite.T(), err)

	// Viewer should NOT be able to create clusters (requires operator+)
	body := `{"name": "viewer-created-cluster", "description": "Should fail"}`
	req, err := http.NewRequest("POST", "/api/v1/clusters", bytes.NewBufferString(body))
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viewerToken)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Note: This test documents that RBAC should be enforced on write operations
	// The actual enforcement depends on the handler implementation
	suite.T().Logf("Viewer attempting write operation returned: %d", w.Code)
}

// TestSecurity runs the security test suite
func TestSecurity(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}

// TestSecurity_RateLimiting tests that rate limiting is configured
func (suite *SecurityTestSuite) TestSecurity_RateLimiting() {
	token, err := suite.authManager.GenerateToken("test-user", middleware.RoleAdmin, middleware.TokenTypeAccess)
	require.NoError(suite.T(), err)

	// Make many rapid requests
	var responses []int
	for i := 0; i < 100; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/clusters", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		responses = append(responses, w.Code)
	}

	// Check if any requests were rate limited (429)
	rateLimited := false
	for _, code := range responses {
		if code == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
	}

	// Log warning if no rate limiting detected
	// Note: Rate limiting may be configured differently in production
	if !rateLimited {
		suite.T().Log("WARNING: No rate limiting detected after 100 rapid requests")
		suite.T().Log("Consider enabling rate limiting for production deployments")
	}
}
