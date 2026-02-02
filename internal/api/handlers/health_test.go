package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/storage"
	testutils "github.com/dsyorkd/pi-controller/internal/testing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestHealthHandler_Health tests the basic health endpoint
func TestHealthHandler_Health(t *testing.T) {
	db, cleanup := testutils.SetupTestDBFile(t)
	defer cleanup()

	database := storage.NewForTestWithDB(db, logger.Default())
	handler := NewHealthHandler(database)

	// Setup Gin router
	router := gin.New()
	router.GET("/health", handler.Health)

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.NotZero(t, response.Timestamp)
	assert.NotEmpty(t, response.Uptime)
}

// TestHealthHandler_Ready tests the readiness endpoint
func TestHealthHandler_Ready(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func() (*storage.Database, func())
		expectedStatus int
		expectedState  string
	}{
		{
			name: "healthy database",
			setupDB: func() (*storage.Database, func()) {
				db, cleanup := testutils.SetupTestDBFile(t)
				return storage.NewForTestWithDB(db, logger.Default()), cleanup
			},
			expectedStatus: http.StatusOK,
			expectedState:  "ready",
		},
		{
			name: "unhealthy database",
			setupDB: func() (*storage.Database, func()) {
				// Create database and close it immediately to simulate unhealthy state
				db, cleanup := testutils.SetupTestDBFile(t)
				sqlDB, _ := db.DB()
				sqlDB.Close() // Close the connection to make it unhealthy
				return storage.NewForTestWithDB(db, logger.Default()), cleanup
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedState:  "not_ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, cleanup := tt.setupDB()
			defer cleanup()

			handler := NewHealthHandler(database)

			// Setup Gin router
			router := gin.New()
			router.GET("/ready", handler.Ready)

			// Create test request
			req, err := http.NewRequest("GET", "/ready", nil)
			require.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ReadinessResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedState, response.Status)
			assert.NotZero(t, response.Timestamp)
			assert.Contains(t, response.Services, "database")

			if tt.expectedState == "ready" {
				assert.Equal(t, "healthy", response.Services["database"])
			} else {
				assert.Contains(t, response.Services["database"], "unhealthy")
			}
		})
	}
}

// TestSystemInfo tests the system info endpoint
func TestSystemInfo(t *testing.T) {
	// Setup Gin router
	router := gin.New()
	router.GET("/system/info", SystemInfo)

	// Create test request
	req, err := http.NewRequest("GET", "/system/info", nil)
	require.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify expected fields are present
	expectedFields := []string{
		"go_version", "go_os", "go_arch", "cpu_count",
		"goroutines", "memory", "gc", "timestamp", "uptime",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field, "Response should contain field %s", field)
	}

	// Verify memory structure
	memory, ok := response["memory"].(map[string]interface{})
	require.True(t, ok, "Memory should be a map")

	memoryFields := []string{
		"alloc", "total_alloc", "sys", "heap_alloc",
		"heap_sys", "heap_inuse", "heap_idle", "heap_objects",
	}

	for _, field := range memoryFields {
		assert.Contains(t, memory, field, "Memory should contain field %s", field)
	}

	// Verify GC structure
	gc, ok := response["gc"].(map[string]interface{})
	require.True(t, ok, "GC should be a map")

	gcFields := []string{"num_gc", "pause_total", "last_gc"}
	for _, field := range gcFields {
		assert.Contains(t, gc, field, "GC should contain field %s", field)
	}
}

// TestSystemMetrics tests the system metrics endpoint
func TestSystemMetrics(t *testing.T) {
	// Setup Gin router
	router := gin.New()
	router.GET("/system/metrics", SystemMetrics)

	// Create test request
	req, err := http.NewRequest("GET", "/system/metrics", nil)
	require.NoError(t, err)

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify expected fields are present
	expectedFields := []string{
		"uptime_seconds", "goroutines_count", "memory_alloc",
		"memory_sys", "gc_count", "gc_pause_total_ns", "timestamp",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field, "Response should contain field %s", field)
	}

	// Verify numeric types
	uptimeSeconds, ok := response["uptime_seconds"].(float64)
	require.True(t, ok, "uptime_seconds should be numeric")
	assert.GreaterOrEqual(t, uptimeSeconds, 0.0)

	goroutinesCount, ok := response["goroutines_count"].(float64)
	require.True(t, ok, "goroutines_count should be numeric")
	assert.Greater(t, goroutinesCount, 0.0)
}

// TestHealthEndpoints_Security tests security aspects of health endpoints
func TestHealthEndpoints_Security(t *testing.T) {
	db, cleanup := testutils.SetupTestDBFile(t)
	defer cleanup()

	database := storage.NewForTestWithDB(db, logger.Default())
	handler := NewHealthHandler(database)

	// Setup Gin router
	router := gin.New()
	router.GET("/health", handler.Health)
	router.GET("/ready", handler.Ready)
	router.GET("/system/info", SystemInfo)
	router.GET("/system/metrics", SystemMetrics)

	securityTests := []struct {
		name             string
		endpoint         string
		method           string
		expectedStatus   int
		shouldNotContain []string
		description      string
	}{
		{
			name:           "health endpoint doesn't leak sensitive info",
			endpoint:       "/health",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"password", "secret", "key", "token",
				"database", "connection", "user",
			},
			description: "Health endpoint should not expose sensitive information",
		},
		{
			name:           "ready endpoint minimal info exposure",
			endpoint:       "/ready",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"password", "secret", "key", "token",
				"connection_string", "user",
			},
			description: "Ready endpoint should minimize information exposure",
		},
		{
			name:           "system info potential info disclosure",
			endpoint:       "/system/info",
			method:         "GET",
			expectedStatus: http.StatusOK,
			shouldNotContain: []string{
				"password", "secret", "key", "token",
			},
			description: "System info should not contain secrets but may expose system details",
		},
		{
			name:           "POST not allowed on health endpoints",
			endpoint:       "/health",
			method:         "POST",
			expectedStatus: http.StatusNotFound,
			description:    "Health endpoints should not accept POST requests",
		},
		{
			name:           "PUT not allowed on health endpoints",
			endpoint:       "/ready",
			method:         "PUT",
			expectedStatus: http.StatusNotFound,
			description:    "Health endpoints should not accept PUT requests",
		},
		{
			name:           "DELETE not allowed on health endpoints",
			endpoint:       "/system/info",
			method:         "DELETE",
			expectedStatus: http.StatusNotFound,
			description:    "Health endpoints should not accept DELETE requests",
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.endpoint, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)

			if tt.expectedStatus == http.StatusOK {
				responseBody := w.Body.String()
				for _, sensitiveInfo := range tt.shouldNotContain {
					assert.NotContains(t, responseBody, sensitiveInfo,
						"Response should not contain sensitive information: %s", sensitiveInfo)
				}
			}
		})
	}
}

// TestHealthHandler_ConcurrentRequests tests concurrent access to health endpoints
func TestHealthHandler_ConcurrentRequests(t *testing.T) {
	db, cleanup := testutils.SetupTestDBFile(t)
	defer cleanup()

	database := storage.NewForTestWithDB(db, logger.Default())
	handler := NewHealthHandler(database)

	router := gin.New()
	router.GET("/health", handler.Health)
	router.GET("/ready", handler.Ready)

	// Test concurrent access
	const numRequests = 50
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(endpoint string) {
			defer func() { done <- true }()

			req, err := http.NewRequest("GET", endpoint, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		}([]string{"/health", "/ready"}[i%2])
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// BenchmarkHealthHandler_Health benchmarks the health endpoint
func BenchmarkHealthHandler_Health(b *testing.B) {
	t := &testing.T{}
	db, cleanup := testutils.SetupTestDBFile(t)
	defer cleanup()

	database := storage.NewForTestWithDB(db, logger.Default())
	handler := NewHealthHandler(database)

	router := gin.New()
	router.GET("/health", handler.Health)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatal("Unexpected status code:", w.Code)
		}
	}
}

// BenchmarkHealthHandler_Ready benchmarks the readiness endpoint
func BenchmarkHealthHandler_Ready(b *testing.B) {
	t := &testing.T{}
	db, cleanup := testutils.SetupTestDBFile(t)
	defer cleanup()

	database := storage.NewForTestWithDB(db, logger.Default())
	handler := NewHealthHandler(database)

	router := gin.New()
	router.GET("/ready", handler.Ready)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			b.Fatal("Unexpected status code:", w.Code)
		}
	}
}
