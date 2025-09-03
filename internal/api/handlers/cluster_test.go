package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// ClusterServiceInterface defines the interface that ClusterService implements
type ClusterServiceInterface interface {
	Create(req services.CreateClusterRequest) (*models.Cluster, error)
	Update(id uint, req services.UpdateClusterRequest) (*models.Cluster, error)
	Delete(id uint) error
	List(opts services.ClusterListOptions) ([]models.Cluster, int64, error)
	GetByID(id uint) (*models.Cluster, error)
	GetByName(name string) (*models.Cluster, error)
	GetNodes(clusterID uint) ([]models.Node, error)
	GetStatus(id uint) (models.ClusterStatus, error)
}

// TestClusterHandler wraps ClusterHandler with an interface for testing
type TestClusterHandler struct {
	service ClusterServiceInterface
	logger  logger.Interface
}

// NewTestClusterHandler creates a new test cluster handler
func NewTestClusterHandler(service ClusterServiceInterface, logger logger.Interface) *TestClusterHandler {
	return &TestClusterHandler{
		service: service,
		logger:  logger.WithField("handler", "cluster"),
	}
}

// Delegate methods to match ClusterHandler interface
func (h *TestClusterHandler) List(c *gin.Context) {
	// Parse query parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	opts := services.ClusterListOptions{
		Limit:  limit,
		Offset: offset,
	}

	clusters, total, err := h.service.List(opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list clusters")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusters": clusters,
		"count":    len(clusters),
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *TestClusterHandler) Create(c *gin.Context) {
	var req services.CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	cluster, err := h.service.Create(req)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create cluster")
		return
	}

	h.logger.WithField("cluster_id", cluster.ID).Info("Created new cluster")
	c.JSON(http.StatusCreated, cluster)
}

func (h *TestClusterHandler) Get(c *gin.Context) {
	var id uint64
	var err error
	if id, err = parseUintParam(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	cluster, err := h.service.GetByID(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to get cluster")
		return
	}

	c.JSON(http.StatusOK, cluster)
}

func (h *TestClusterHandler) Update(c *gin.Context) {
	var id uint64
	var err error
	if id, err = parseUintParam(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var req services.UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	cluster, err := h.service.Update(uint(id), req)
	if err != nil {
		h.handleServiceError(c, err, "Failed to update cluster")
		return
	}

	h.logger.WithField("cluster_id", cluster.ID).Info("Updated cluster")
	c.JSON(http.StatusOK, cluster)
}

func (h *TestClusterHandler) Delete(c *gin.Context) {
	var id uint64
	var err error
	if id, err = parseUintParam(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		h.handleServiceError(c, err, "Failed to delete cluster")
		return
	}

	h.logger.WithField("cluster_id", id).Info("Deleted cluster")
	c.JSON(http.StatusNoContent, nil)
}

func (h *TestClusterHandler) ListNodes(c *gin.Context) {
	var id uint64
	var err error
	if id, err = parseUintParam(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	nodes, err := h.service.GetNodes(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to list cluster nodes")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes":      nodes,
		"count":      len(nodes),
		"cluster_id": uint(id),
	})
}

func (h *TestClusterHandler) Status(c *gin.Context) {
	var id uint64
	var err error
	if id, err = parseUintParam(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	status, err := h.service.GetStatus(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to get cluster status")
		return
	}

	c.JSON(http.StatusOK, status)
}

// Helper methods
func parseUintParam(param string) (uint64, error) {
	var id uint64
	_, err := fmt.Sscanf(param, "%d", &id)
	return id, err
}

func (h *TestClusterHandler) handleServiceError(c *gin.Context, err error, message string) {
	h.logger.WithError(err).Error(message)

	if services.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "Cluster not found",
		})
		return
	}

	if services.IsAlreadyExists(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "Cluster with that name already exists",
		})
		return
	}

	if err == services.ErrHasAssociatedResources {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "Cannot delete cluster with associated nodes",
		})
		return
	}

	if services.IsValidationFailed(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation Failed",
			"message": err.Error(),
		})
		return
	}

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Internal Server Error",
		"message": message,
	})
}

// MockClusterService is a mock implementation of the ClusterServiceInterface
type MockClusterService struct {
	mock.Mock
}

func (m *MockClusterService) Create(req services.CreateClusterRequest) (*models.Cluster, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) Update(id uint, req services.UpdateClusterRequest) (*models.Cluster, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockClusterService) List(opts services.ClusterListOptions) ([]models.Cluster, int64, error) {
	args := m.Called(opts)
	return args.Get(0).([]models.Cluster), args.Get(1).(int64), args.Error(2)
}

func (m *MockClusterService) GetByID(id uint) (*models.Cluster, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) GetByName(name string) (*models.Cluster, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) GetNodes(clusterID uint) ([]models.Node, error) {
	args := m.Called(clusterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Node), args.Error(1)
}

func (m *MockClusterService) GetStatus(id uint) (models.ClusterStatus, error) {
	args := m.Called(id)
	return args.Get(0).(models.ClusterStatus), args.Error(1)
}

func init() {
	gin.SetMode(gin.TestMode)
}

// setupClusterHandlerTest sets up a test environment for cluster handler tests
func setupClusterHandlerTest() (*TestClusterHandler, *MockClusterService, *gin.Engine) {
	mockService := &MockClusterService{}
	handler := NewTestClusterHandler(mockService, logger.Default())
	
	router := gin.New()
	
	// Setup routes
	v1 := router.Group("/api/v1")
	clusters := v1.Group("/clusters")
	{
		clusters.GET("", handler.List)
		clusters.POST("", handler.Create)
		clusters.GET("/:id", handler.Get)
		clusters.PUT("/:id", handler.Update)
		clusters.DELETE("/:id", handler.Delete)
		clusters.GET("/:id/nodes", handler.ListNodes)
		clusters.GET("/:id/status", handler.Status)
	}
	
	return handler, mockService, router
}

func TestClusterHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func(*MockClusterService)
		expectedStatus int
		expectedCount  int
		expectedTotal  int64
	}{
		{
			name:        "successful list with default parameters",
			queryParams: "",
			mockSetup: func(m *MockClusterService) {
				clusters := []models.Cluster{
					{ID: 1, Name: "cluster1", Status: models.ClusterStatusActive},
					{ID: 2, Name: "cluster2", Status: models.ClusterStatusActive},
				}
				m.On("List", mock.AnythingOfType("services.ClusterListOptions")).Return(clusters, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedTotal:  2,
		},
		{
			name:        "successful list with pagination",
			queryParams: "?limit=1&offset=1",
			mockSetup: func(m *MockClusterService) {
				clusters := []models.Cluster{
					{ID: 2, Name: "cluster2", Status: models.ClusterStatusActive},
				}
				m.On("List", mock.MatchedBy(func(opts services.ClusterListOptions) bool {
					return opts.Limit == 1 && opts.Offset == 1
				})).Return(clusters, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTotal:  2,
		},
		{
			name:        "service error",
			queryParams: "",
			mockSetup: func(m *MockClusterService) {
				m.On("List", mock.AnythingOfType("services.ClusterListOptions")).Return([]models.Cluster{}, int64(0), fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			req, err := http.NewRequest("GET", "/api/v1/clusters"+tt.queryParams, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(tt.expectedCount), response["count"])
				assert.Equal(t, float64(tt.expectedTotal), response["total"])
				assert.Contains(t, response, "clusters")
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestClusterHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockClusterService)
		expectedStatus int
		expectCluster  bool
	}{
		{
			name: "successful creation",
			requestBody: services.CreateClusterRequest{
				Name:        "test-cluster",
				Description: "Test cluster description",
			},
			mockSetup: func(m *MockClusterService) {
				expectedCluster := &models.Cluster{
					ID:          1,
					Name:        "test-cluster",
					Description: "Test cluster description",
					Status:      models.ClusterStatusActive,
				}
				m.On("Create", mock.AnythingOfType("services.CreateClusterRequest")).Return(expectedCluster, nil)
			},
			expectedStatus: http.StatusCreated,
			expectCluster:  true,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
			expectCluster:  false,
		},
		{
			name: "service error - already exists",
			requestBody: services.CreateClusterRequest{
				Name:        "existing-cluster",
				Description: "Test cluster",
			},
			mockSetup: func(m *MockClusterService) {
				m.On("Create", mock.AnythingOfType("services.CreateClusterRequest")).Return(nil, services.ErrAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectCluster:  false,
		},
		{
			name: "service error - validation failed",
			requestBody: services.CreateClusterRequest{
				Name:        "",
				Description: "Test cluster",
			},
			mockSetup: func(m *MockClusterService) {
				m.On("Create", mock.AnythingOfType("services.CreateClusterRequest")).Return(nil, services.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			expectCluster:  false,
		},
		{
			name: "service error - internal error",
			requestBody: services.CreateClusterRequest{
				Name:        "test-cluster",
				Description: "Test cluster",
			},
			mockSetup: func(m *MockClusterService) {
				m.On("Create", mock.AnythingOfType("services.CreateClusterRequest")).Return(nil, fmt.Errorf("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectCluster:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req, err := http.NewRequest("POST", "/api/v1/clusters", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectCluster {
				var cluster models.Cluster
				err = json.Unmarshal(w.Body.Bytes(), &cluster)
				require.NoError(t, err)
				assert.Equal(t, "test-cluster", cluster.Name)
				assert.Equal(t, uint(1), cluster.ID)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestClusterHandler_Get(t *testing.T) {
	tests := []struct {
		name           string
		clusterID      string
		mockSetup      func(*MockClusterService)
		expectedStatus int
		expectCluster  bool
	}{
		{
			name:      "successful get",
			clusterID: "1",
			mockSetup: func(m *MockClusterService) {
				cluster := &models.Cluster{
					ID:     1,
					Name:   "test-cluster",
					Status: models.ClusterStatusActive,
				}
				m.On("GetByID", uint(1)).Return(cluster, nil)
			},
			expectedStatus: http.StatusOK,
			expectCluster:  true,
		},
		{
			name:      "invalid cluster ID",
			clusterID: "invalid",
			mockSetup: func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
			expectCluster:  false,
		},
		{
			name:      "cluster not found",
			clusterID: "999",
			mockSetup: func(m *MockClusterService) {
				m.On("GetByID", uint(999)).Return(nil, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectCluster:  false,
		},
		{
			name:      "service error",
			clusterID: "1",
			mockSetup: func(m *MockClusterService) {
				m.On("GetByID", uint(1)).Return(nil, fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectCluster:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			req, err := http.NewRequest("GET", "/api/v1/clusters/"+tt.clusterID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectCluster {
				var cluster models.Cluster
				err = json.Unmarshal(w.Body.Bytes(), &cluster)
				require.NoError(t, err)
				assert.Equal(t, "test-cluster", cluster.Name)
				assert.Equal(t, uint(1), cluster.ID)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestClusterHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		clusterID      string
		requestBody    interface{}
		mockSetup      func(*MockClusterService)
		expectedStatus int
		expectCluster  bool
	}{
		{
			name:      "successful update",
			clusterID: "1",
			requestBody: services.UpdateClusterRequest{
				Name:        stringPtr("updated-cluster"),
				Description: stringPtr("Updated description"),
			},
			mockSetup: func(m *MockClusterService) {
				updatedCluster := &models.Cluster{
					ID:          1,
					Name:        "updated-cluster",
					Description: "Updated description",
					Status:      models.ClusterStatusActive,
				}
				m.On("Update", uint(1), mock.AnythingOfType("services.UpdateClusterRequest")).Return(updatedCluster, nil)
			},
			expectedStatus: http.StatusOK,
			expectCluster:  true,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			requestBody:    services.UpdateClusterRequest{},
			mockSetup:      func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
			expectCluster:  false,
		},
		{
			name:           "invalid JSON",
			clusterID:      "1",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
			expectCluster:  false,
		},
		{
			name:      "cluster not found",
			clusterID: "999",
			requestBody: services.UpdateClusterRequest{
				Name: stringPtr("updated-cluster"),
			},
			mockSetup: func(m *MockClusterService) {
				m.On("Update", uint(999), mock.AnythingOfType("services.UpdateClusterRequest")).Return(nil, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectCluster:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req, err := http.NewRequest("PUT", "/api/v1/clusters/"+tt.clusterID, bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectCluster {
				var cluster models.Cluster
				err = json.Unmarshal(w.Body.Bytes(), &cluster)
				require.NoError(t, err)
				assert.Equal(t, "updated-cluster", cluster.Name)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestClusterHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		clusterID      string
		mockSetup      func(*MockClusterService)
		expectedStatus int
	}{
		{
			name:      "successful deletion",
			clusterID: "1",
			mockSetup: func(m *MockClusterService) {
				m.On("Delete", uint(1)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			mockSetup:      func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "cluster not found",
			clusterID: "999",
			mockSetup: func(m *MockClusterService) {
				m.On("Delete", uint(999)).Return(services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "cluster has associated resources",
			clusterID: "1",
			mockSetup: func(m *MockClusterService) {
				m.On("Delete", uint(1)).Return(services.ErrHasAssociatedResources)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			req, err := http.NewRequest("DELETE", "/api/v1/clusters/"+tt.clusterID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}

func TestClusterHandler_ListNodes(t *testing.T) {
	tests := []struct {
		name           string
		clusterID      string
		mockSetup      func(*MockClusterService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:      "successful list nodes",
			clusterID: "1",
			mockSetup: func(m *MockClusterService) {
				nodes := []models.Node{
					{ID: 1, Name: "node1", ClusterID: uintPtr(1)},
					{ID: 2, Name: "node2", ClusterID: uintPtr(1)},
				}
				m.On("GetNodes", uint(1)).Return(nodes, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			mockSetup:      func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "cluster not found",
			clusterID: "999",
			mockSetup: func(m *MockClusterService) {
				m.On("GetNodes", uint(999)).Return([]models.Node{}, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			req, err := http.NewRequest("GET", "/api/v1/clusters/"+tt.clusterID+"/nodes", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(tt.expectedCount), response["count"])
				assert.Contains(t, response, "nodes")
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestClusterHandler_Status(t *testing.T) {
	tests := []struct {
		name           string
		clusterID      string
		mockSetup      func(*MockClusterService)
		expectedStatus int
		expectedState  models.ClusterStatus
	}{
		{
			name:      "successful status check",
			clusterID: "1",
			mockSetup: func(m *MockClusterService) {
				m.On("GetStatus", uint(1)).Return(models.ClusterStatusActive, nil)
			},
			expectedStatus: http.StatusOK,
			expectedState:  models.ClusterStatusActive,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			mockSetup:      func(m *MockClusterService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "cluster not found",
			clusterID: "999",
			mockSetup: func(m *MockClusterService) {
				m.On("GetStatus", uint(999)).Return(models.ClusterStatus(""), services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mockService, router := setupClusterHandlerTest()
			tt.mockSetup(mockService)

			req, err := http.NewRequest("GET", "/api/v1/clusters/"+tt.clusterID+"/status", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var status models.ClusterStatus
				err = json.Unmarshal(w.Body.Bytes(), &status)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, status)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func uintPtr(i uint) *uint {
	return &i
}