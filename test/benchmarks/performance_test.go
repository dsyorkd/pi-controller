package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/api/handlers"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	testutils "github.com/dsyorkd/pi-controller/internal/testing"
	"github.com/dsyorkd/pi-controller/pkg/gpio"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupBenchmarkEnvironment sets up a test environment for benchmarks
func setupBenchmarkEnvironment(b *testing.B) (*gin.Engine, *storage.Database, func()) {
	t := &testing.T{}
	db, cleanup := testutils.SetupTestDBFile(t)
	database := storage.NewForTestWithDB(db, logger.Default())

	testLogger := logger.Default()

	// Initialize services
	clusterService := services.NewClusterService(database, testLogger)
	nodeService := services.NewNodeService(database, testLogger)
	gpioService := services.NewGPIOService(database, testLogger)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(database)
	clusterHandler := handlers.NewClusterHandler(clusterService, testLogger)
	nodeHandler := handlers.NewNodeHandler(nodeService, testLogger)
	gpioHandler := handlers.NewGPIOHandler(gpioService, testLogger)

	// Setup router
	router := gin.New()

	// Health endpoints
	router.GET("/health", healthHandler.Health)
	router.GET("/system/info", handlers.SystemInfo)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		clusters := v1.Group("/clusters")
		{
			clusters.GET("", clusterHandler.List)
			clusters.POST("", clusterHandler.Create)
			clusters.GET("/:id", clusterHandler.Get)
		}

		nodes := v1.Group("/nodes")
		{
			nodes.GET("", nodeHandler.List)
			nodes.POST("", nodeHandler.Create)
		}

		gpio := v1.Group("/gpio")
		{
			gpio.GET("", gpioHandler.List)
			gpio.POST("", gpioHandler.Create)
			gpio.POST("/:id/write", gpioHandler.Write)
		}
	}

	return router, database, cleanup
}

// BenchmarkHealth_SimpleEndpoint benchmarks the health endpoint
func BenchmarkHealth_SimpleEndpoint(b *testing.B) {
	router, _, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkSystemInfo_ComplexEndpoint benchmarks the system info endpoint
func BenchmarkSystemInfo_ComplexEndpoint(b *testing.B) {
	router, _, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/system/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkCluster_Create benchmarks cluster creation
func BenchmarkCluster_Create(b *testing.B) {
	router, _, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		createReq := services.CreateClusterRequest{
			Name:        fmt.Sprintf("benchmark-cluster-%d", i),
			Description: "Benchmark test cluster",
		}

		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/v1/clusters", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status 201, got %d", w.Code)
		}
	}
}

// BenchmarkCluster_List benchmarks cluster listing with varying data sizes
func BenchmarkCluster_List(b *testing.B) {
	router, database, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	// Pre-populate database with test clusters
	testSizes := []int{10, 100, 1000}

	for _, size := range testSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			// Create test clusters
			for i := 0; i < size; i++ {
				cluster := &models.Cluster{
					Name:        fmt.Sprintf("bench-cluster-%d", i),
					Description: "Benchmark cluster",
					Status:      models.ClusterStatusActive,
				}
				database.DB().Create(cluster)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, _ := http.NewRequest("GET", "/api/v1/clusters", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					b.Fatalf("Expected status 200, got %d", w.Code)
				}
			}

			// Cleanup for next iteration
			database.DB().Where("name LIKE ?", "bench-cluster-%").Delete(&models.Cluster{})
		})
	}
}

// BenchmarkNode_Create benchmarks node creation with database operations
func BenchmarkNode_Create(b *testing.B) {
	router, database, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	// Setup test cluster
	t := &testing.T{}
	cluster := testutils.CreateTestCluster(t)
	require.NoError(b, database.DB().Create(cluster).Error)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		createReq := services.CreateNodeRequest{
			Name:       fmt.Sprintf("benchmark-node-%d", i),
			IPAddress:  fmt.Sprintf("192.168.1.%d", (i%254)+1),
			MACAddress: fmt.Sprintf("02:00:00:00:00:%02x", i%256),
			Role:       models.NodeRoleWorker,
			ClusterID:  &cluster.ID,
		}

		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/v1/nodes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status 201, got %d", w.Code)
		}
	}
}

// BenchmarkGPIO_Create benchmarks GPIO device creation
func BenchmarkGPIO_Create(b *testing.B) {
	router, database, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	// Setup test data
	t := &testing.T{}
	cluster := testutils.CreateTestCluster(t)
	require.NoError(b, database.DB().Create(cluster).Error)
	node := testutils.CreateTestNode(t, cluster.ID)
	require.NoError(b, database.DB().Create(node).Error)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		createReq := services.CreateGPIODeviceRequest{
			Name:        fmt.Sprintf("benchmark-gpio-%d", i),
			Description: "Benchmark GPIO device",
			NodeID:      node.ID,
			PinNumber:   18 + (i % 10), // Cycle through pins 18-27
			Direction:   models.GPIODirectionOutput,
			PullMode:    models.GPIOPullNone,
			DeviceType:  models.GPIODeviceTypeDigital,
		}

		body, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/v1/gpio", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Some may fail due to duplicate pins, that's OK for benchmarking
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			b.Fatalf("Unexpected status code: %d", w.Code)
		}
	}
}

// BenchmarkGPIO_Write benchmarks GPIO write operations
func BenchmarkGPIO_Write(b *testing.B) {
	router, database, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	// Setup test data
	t := &testing.T{}
	cluster := testutils.CreateTestCluster(t)
	require.NoError(b, database.DB().Create(cluster).Error)
	node := testutils.CreateTestNode(t, cluster.ID)
	require.NoError(b, database.DB().Create(node).Error)
	device := testutils.CreateTestGPIODevice(t, node.ID)
	device.Direction = models.GPIODirectionOutput
	device.Status = models.GPIOStatusActive
	require.NoError(b, database.DB().Create(device).Error)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		writeReq := struct {
			Value int `json:"value"`
		}{Value: i % 2} // Alternate between 0 and 1

		body, _ := json.Marshal(writeReq)
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/gpio/%d/write", device.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkGPIO_Controller_Operations benchmarks low-level GPIO operations
func BenchmarkGPIO_Controller_Operations(b *testing.B) {
	config := &gpio.Config{
		MockMode:        true,
		AllowedPins:     []int{18, 19, 20, 21, 22, 23, 24, 25},
		RestrictedPins:  []int{0, 1},
		DefaultPullMode: gpio.PullNone,
	}

	gpioLogger := logrus.New()
	controller := gpio.NewController(config, gpio.DefaultSecurityConfig(), gpioLogger)
	require.NotNil(b, controller)

	ctx := context.Background()
	err := controller.Initialize(ctx)
	require.NoError(b, err)
	defer controller.Close()

	// Configure test pins
	outputConfig := gpio.PinConfig{Pin: 18, Direction: gpio.DirectionOutput, PullMode: gpio.PullNone}
	require.NoError(b, controller.ConfigurePin(outputConfig, "benchmark"))

	inputConfig := gpio.PinConfig{Pin: 19, Direction: gpio.DirectionInput, PullMode: gpio.PullNone}
	require.NoError(b, controller.ConfigurePin(inputConfig, "benchmark"))

	b.Run("WritePin", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			value := gpio.PinValue(i % 2)
			err := controller.WritePin(18, value, "benchmark")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ReadPin", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := controller.ReadPin(19, "benchmark")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetPinState", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := controller.GetPinState(18, "benchmark")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SetPWM", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			frequency := 1000 + (i % 1000)
			dutyCycle := i % 101
			err := controller.SetPWM(18, frequency, dutyCycle, "benchmark")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListConfiguredPins", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := controller.ListConfiguredPins()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDatabase_Operations benchmarks database operations
func BenchmarkDatabase_Operations(b *testing.B) {
	t := &testing.T{}
	db, cleanup := testutils.SetupTestDBFile(t)
	defer cleanup()

	b.Run("ClusterInsert", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			cluster := &models.Cluster{
				Name:        fmt.Sprintf("bench-cluster-%d", i),
				Description: "Benchmark cluster",
				Status:      models.ClusterStatusActive,
			}

			result := db.Create(cluster)
			if result.Error != nil {
				b.Fatal(result.Error)
			}
		}
	})

	// Pre-populate for read tests
	for i := 0; i < 1000; i++ {
		cluster := &models.Cluster{
			Name:        fmt.Sprintf("read-test-cluster-%d", i),
			Description: "Read test cluster",
			Status:      models.ClusterStatusActive,
		}
		db.Create(cluster)
	}

	b.Run("ClusterSelect", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var cluster models.Cluster
			result := db.Where("name = ?", fmt.Sprintf("read-test-cluster-%d", i%1000)).First(&cluster)
			if result.Error != nil {
				b.Fatal(result.Error)
			}
		}
	})

	b.Run("ClusterList", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var clusters []models.Cluster
			result := db.Limit(10).Find(&clusters)
			if result.Error != nil {
				b.Fatal(result.Error)
			}
		}
	})
}

// BenchmarkConcurrency_HealthEndpoint benchmarks concurrent access to health endpoint
func BenchmarkConcurrency_HealthEndpoint(b *testing.B) {
	router, _, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req, _ := http.NewRequest("GET", "/health", nil)
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)

					if w.Code != http.StatusOK {
						b.Fatalf("Expected status 200, got %d", w.Code)
					}
				}
			})
		})
	}
}

// BenchmarkMemory_AllocationPatterns benchmarks memory allocation patterns
func BenchmarkMemory_AllocationPatterns(b *testing.B) {
	router, _, cleanup := setupBenchmarkEnvironment(b)
	defer cleanup()

	b.Run("HealthEndpoint_Allocations", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("SystemInfo_Allocations", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", "/system/info", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("JSON_Marshal_Unmarshal", func(b *testing.B) {
		createReq := services.CreateClusterRequest{
			Name:        "benchmark-cluster",
			Description: "Benchmark test cluster",
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Marshal
			data, err := json.Marshal(createReq)
			if err != nil {
				b.Fatal(err)
			}

			// Unmarshal
			var unmarshaled services.CreateClusterRequest
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
