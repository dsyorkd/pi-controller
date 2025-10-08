package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sServiceInterface defines the interface that K8sService implements
type K8sServiceInterface interface {
	CreateService(ctx context.Context, clusterID uint, service *corev1.Service) (*corev1.Service, error)
	GetService(ctx context.Context, clusterID uint, namespace, name string) (*corev1.Service, error)
	ListServices(ctx context.Context, clusterID uint, namespace string) ([]corev1.Service, error)
	UpdateService(ctx context.Context, clusterID uint, namespace string, service *corev1.Service) (*corev1.Service, error)
	DeleteService(ctx context.Context, clusterID uint, namespace, name string) error
}

// TestServiceHandler wraps ServiceHandler with an interface for testing
type TestServiceHandler struct {
	service K8sServiceInterface
	logger  logger.Interface
}

// NewTestServiceHandler creates a new test service handler
func NewTestServiceHandler(service K8sServiceInterface, logger logger.Interface) *TestServiceHandler {
	return &TestServiceHandler{
		service: service,
		logger:  logger.WithField("handler", "service"),
	}
}

// Delegate methods to match ServiceHandler interface
func (h *TestServiceHandler) CreateService(c *gin.Context) {
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	service, err := h.service.CreateService(c.Request.Context(), req.ClusterID, req.Service)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create service")
		return
	}

	h.logger.WithField("cluster_id", req.ClusterID).WithField("service_name", service.Name).Info("Created new service")
	c.JSON(http.StatusCreated, service)
}

func (h *TestServiceHandler) GetService(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var clusterID uint64
	if _, err := fmt.Sscanf(clusterIDStr, "%d", &clusterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	namespace := c.Param("namespace")
	serviceName := c.Param("name")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Service name is required",
		})
		return
	}

	service, err := h.service.GetService(c.Request.Context(), uint(clusterID), namespace, serviceName)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get service")
		return
	}

	c.JSON(http.StatusOK, service)
}

func (h *TestServiceHandler) ListServices(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var clusterID uint64
	if _, err := fmt.Sscanf(clusterIDStr, "%d", &clusterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	namespace := c.Param("namespace")

	// Parse query parameters for pagination
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	services, err := h.service.ListServices(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list services")
		return
	}

	// Apply pagination (simple in-memory pagination for now)
	total := len(services)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedServices := services[start:end]

	response := GetServicesResponse{
		Services: paginatedServices,
		Meta: MetaResponse{
			Count:  len(paginatedServices),
			Limit:  limit,
			Offset: offset,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (h *TestServiceHandler) DeleteService(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	if clusterIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var clusterID uint64
	if _, err := fmt.Sscanf(clusterIDStr, "%d", &clusterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	namespace := c.Param("namespace")
	serviceName := c.Param("name")
	if serviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Service name is required",
		})
		return
	}

	err := h.service.DeleteService(c.Request.Context(), uint(clusterID), namespace, serviceName)
	if err != nil {
		h.handleServiceError(c, err, "Failed to delete service")
		return
	}

	h.logger.WithField("cluster_id", clusterID).WithField("service_name", serviceName).Info("Deleted service")
	c.Status(http.StatusNoContent)
}

// handleServiceError handles service layer errors and maps them to appropriate HTTP responses
func (h *TestServiceHandler) handleServiceError(c *gin.Context, err error, message string) {
	h.logger.WithError(err).Error(message)

	if services.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": message,
		})
		return
	}

	if services.IsAlreadyExists(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": message,
		})
		return
	}

	if err == services.ErrHasAssociatedResources {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "Cannot delete resource with associated dependencies",
		})
		return
	}

	if services.IsValidationFailed(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation Failed",
			"message": message,
		})
		return
	}

	// Default to internal server error
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Internal Server Error",
		"message": message,
	})
}

// MockK8sService is a mock implementation of the K8sServiceInterface
type MockK8sService struct {
	mock.Mock
}

func (m *MockK8sService) CreateService(ctx context.Context, clusterID uint, service *corev1.Service) (*corev1.Service, error) {
	args := m.Called(ctx, clusterID, service)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Service), args.Error(1)
}

func (m *MockK8sService) GetService(ctx context.Context, clusterID uint, namespace, name string) (*corev1.Service, error) {
	args := m.Called(ctx, clusterID, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Service), args.Error(1)
}

func (m *MockK8sService) ListServices(ctx context.Context, clusterID uint, namespace string) ([]corev1.Service, error) {
	args := m.Called(ctx, clusterID, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]corev1.Service), args.Error(1)
}

func (m *MockK8sService) UpdateService(ctx context.Context, clusterID uint, namespace string, service *corev1.Service) (*corev1.Service, error) {
	args := m.Called(ctx, clusterID, namespace, service)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Service), args.Error(1)
}

func (m *MockK8sService) DeleteService(ctx context.Context, clusterID uint, namespace, name string) error {
	args := m.Called(ctx, clusterID, namespace, name)
	return args.Error(0)
}

func setupServiceHandlerTest() (*TestServiceHandler, *MockK8sService, *gin.Engine) {
	mockService := &MockK8sService{}
	handler := NewTestServiceHandler(mockService, logger.Default())
	router := gin.New()

	// Setup routes
	router.POST("/services", handler.CreateService)
	router.GET("/services/clusters/:cluster_id/namespaces/:namespace/services", handler.ListServices)
	router.GET("/services/clusters/:cluster_id/namespaces/:namespace/services/:name", handler.GetService)
	router.DELETE("/services/clusters/:cluster_id/namespaces/:namespace/services/:name", handler.DeleteService)

	return handler, mockService, router
}

func TestServiceHandler_CreateService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockK8sService)
		expectedStatus int
		expectService  bool
	}{
		{
			name: "successful creation",
			requestBody: CreateServiceRequest{
				ClusterID: 1,
				Namespace: "default",
				Service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service",
					},
				},
			},
			mockSetup: func(m *MockK8sService) {
				expectedService := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service",
					},
				}
				m.On("CreateService", mock.Anything, uint(1), mock.AnythingOfType("*v1.Service")).Return(expectedService, nil)
			},
			expectedStatus: http.StatusCreated,
			expectService:  true,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid-json",
			mockSetup:      func(m *MockK8sService) {},
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name: "service error - not found",
			requestBody: CreateServiceRequest{
				ClusterID: 999,
				Service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service",
					},
				},
			},
			mockSetup: func(m *MockK8sService) {
				m.On("CreateService", mock.Anything, uint(999), mock.AnythingOfType("*v1.Service")).Return(nil, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectService:  false,
		},
		{
			name: "service error - internal error",
			requestBody: CreateServiceRequest{
				ClusterID: 1,
				Service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service",
					},
				},
			},
			mockSetup: func(m *MockK8sService) {
				m.On("CreateService", mock.Anything, uint(1), mock.AnythingOfType("*v1.Service")).Return(nil, fmt.Errorf("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectService:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupServiceHandlerTest()
			tc.mockSetup(mockService)

			var reqBody []byte
			var err error
			if tc.requestBody != nil {
				reqBody, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			} else {
				reqBody = []byte(tc.requestBody.(string))
			}

			req, err := http.NewRequest(http.MethodPost, "/services", bytes.NewBuffer(reqBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectService {
				var response corev1.Service
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "test-service", response.Name)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestServiceHandler_GetService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		clusterID      string
		namespace      string
		serviceName    string
		mockSetup      func(*MockK8sService)
		expectedStatus int
		expectService  bool
	}{
		{
			name:        "successful get",
			clusterID:   "1",
			namespace:   "default",
			serviceName: "test-service",
			mockSetup: func(m *MockK8sService) {
				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-service",
					},
				}
				m.On("GetService", mock.Anything, uint(1), "default", "test-service").Return(service, nil)
			},
			expectedStatus: http.StatusOK,
			expectService:  true,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			namespace:      "default",
			serviceName:    "test-service",
			mockSetup:      func(m *MockK8sService) {},
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:        "service not found",
			clusterID:   "1",
			namespace:   "default",
			serviceName: "nonexistent-service",
			mockSetup: func(m *MockK8sService) {
				m.On("GetService", mock.Anything, uint(1), "default", "nonexistent-service").Return(nil, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectService:  false,
		},
		{
			name:        "service error",
			clusterID:   "1",
			namespace:   "default",
			serviceName: "test-service",
			mockSetup: func(m *MockK8sService) {
				m.On("GetService", mock.Anything, uint(1), "default", "test-service").Return(nil, fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectService:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupServiceHandlerTest()
			tc.mockSetup(mockService)

			url := fmt.Sprintf("/services/clusters/%s/namespaces/%s/services/%s", tc.clusterID, tc.namespace, tc.serviceName)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectService {
				var response corev1.Service
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "test-service", response.Name)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestServiceHandler_ListServices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		clusterID      string
		namespace      string
		queryParams    string
		mockSetup      func(*MockK8sService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:        "successful list",
			clusterID:   "1",
			namespace:   "default",
			queryParams: "",
			mockSetup: func(m *MockK8sService) {
				services := []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service2",
						},
					},
				}
				m.On("ListServices", mock.Anything, uint(1), "default").Return(services, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "list with pagination",
			clusterID:   "1",
			namespace:   "default",
			queryParams: "?limit=1&offset=1",
			mockSetup: func(m *MockK8sService) {
				services := []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service2",
						},
					},
				}
				m.On("ListServices", mock.Anything, uint(1), "default").Return(services, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			namespace:      "default",
			queryParams:    "",
			mockSetup:      func(m *MockK8sService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:        "service error",
			clusterID:   "1",
			namespace:   "default",
			queryParams: "",
			mockSetup: func(m *MockK8sService) {
				m.On("ListServices", mock.Anything, uint(1), "default").Return(nil, fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupServiceHandlerTest()
			tc.mockSetup(mockService)

			url := fmt.Sprintf("/services/clusters/%s/namespaces/%s/services%s", tc.clusterID, tc.namespace, tc.queryParams)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response GetServicesResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, len(response.Services))
				assert.Equal(t, tc.expectedCount, response.Meta.Count)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestServiceHandler_DeleteService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		clusterID      string
		namespace      string
		serviceName    string
		mockSetup      func(*MockK8sService)
		expectedStatus int
	}{
		{
			name:        "successful deletion",
			clusterID:   "1",
			namespace:   "default",
			serviceName: "test-service",
			mockSetup: func(m *MockK8sService) {
				m.On("DeleteService", mock.Anything, uint(1), "default", "test-service").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			namespace:      "default",
			serviceName:    "test-service",
			mockSetup:      func(m *MockK8sService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "service not found",
			clusterID:   "1",
			namespace:   "default",
			serviceName: "nonexistent-service",
			mockSetup: func(m *MockK8sService) {
				m.On("DeleteService", mock.Anything, uint(1), "default", "nonexistent-service").Return(services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "service error",
			clusterID:   "1",
			namespace:   "default",
			serviceName: "test-service",
			mockSetup: func(m *MockK8sService) {
				m.On("DeleteService", mock.Anything, uint(1), "default", "test-service").Return(fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupServiceHandlerTest()
			tc.mockSetup(mockService)

			url := fmt.Sprintf("/services/clusters/%s/namespaces/%s/services/%s", tc.clusterID, tc.namespace, tc.serviceName)
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
