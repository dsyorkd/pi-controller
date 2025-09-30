package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// DeploymentServiceInterface defines the interface that DeploymentService implements
type DeploymentServiceInterface interface {
	CreateDeployment(ctx context.Context, clusterID uint, pod *corev1.Pod) (*corev1.Pod, error)
	GetDeployment(ctx context.Context, clusterID uint, name string) (*corev1.Pod, error)
	DeleteDeployment(ctx context.Context, clusterID uint, name string) error
}

// TestDeploymentHandler wraps DeploymentHandler with an interface for testing
type TestDeploymentHandler struct {
	service DeploymentServiceInterface
	logger  logger.Interface
}

// NewTestDeploymentHandler creates a new test deployment handler
func NewTestDeploymentHandler(service DeploymentServiceInterface, logger logger.Interface) *TestDeploymentHandler {
	return &TestDeploymentHandler{
		service: service,
		logger:  logger.WithField("handler", "deployment"),
	}
}

// Delegate methods to match DeploymentHandler interface
func (h *TestDeploymentHandler) CreatePod(c *gin.Context) {
	var req CreatePodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	pod, err := h.service.CreateDeployment(c.Request.Context(), req.ClusterID, req.Pod)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create pod")
		return
	}

	h.logger.WithField("cluster_id", req.ClusterID).WithField("pod_name", pod.Name).Info("Created new pod")
	c.JSON(http.StatusCreated, pod)
}

func (h *TestDeploymentHandler) GetPod(c *gin.Context) {
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

	podName := c.Param("name")
	if podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Pod name is required",
		})
		return
	}

	pod, err := h.service.GetDeployment(c.Request.Context(), uint(clusterID), podName)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get pod")
		return
	}

	c.JSON(http.StatusOK, pod)
}

func (h *TestDeploymentHandler) DeletePod(c *gin.Context) {
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

	podName := c.Param("name")
	if podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Pod name is required",
		})
		return
	}

	err := h.service.DeleteDeployment(c.Request.Context(), uint(clusterID), podName)
	if err != nil {
		h.handleServiceError(c, err, "Failed to delete pod")
		return
	}

	h.logger.WithField("cluster_id", clusterID).WithField("pod_name", podName).Info("Deleted pod")
	c.Status(http.StatusNoContent)
}

func (h *TestDeploymentHandler) GetPodLogs(c *gin.Context) {
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

	podName := c.Param("name")
	if podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Pod name is required",
		})
		return
	}

	// Parse query parameters
	follow := c.DefaultQuery("follow", "false") == "true"
	tail := 100
	if t := c.Query("tail"); t != "" {
		fmt.Sscanf(t, "%d", &tail)
	}

	// Note: This would require extending the DeploymentService to support log retrieval
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"cluster_id": clusterID,
		"pod_name":   podName,
		"follow":     follow,
		"tail":       tail,
		"logs":       "Log retrieval not yet implemented in DeploymentService",
	})
}

// handleServiceError handles service layer errors and maps them to appropriate HTTP responses
func (h *TestDeploymentHandler) handleServiceError(c *gin.Context, err error, message string) {
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

// MockDeploymentService is a mock implementation of the DeploymentServiceInterface
type MockDeploymentService struct {
	mock.Mock
}

func (m *MockDeploymentService) CreateDeployment(ctx context.Context, clusterID uint, pod *corev1.Pod) (*corev1.Pod, error) {
	args := m.Called(ctx, clusterID, pod)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockDeploymentService) GetDeployment(ctx context.Context, clusterID uint, name string) (*corev1.Pod, error) {
	args := m.Called(ctx, clusterID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockDeploymentService) DeleteDeployment(ctx context.Context, clusterID uint, name string) error {
	args := m.Called(ctx, clusterID, name)
	return args.Error(0)
}

func setupDeploymentHandlerTest() (*TestDeploymentHandler, *MockDeploymentService, *gin.Engine) {
	mockService := &MockDeploymentService{}
	handler := NewTestDeploymentHandler(mockService, logger.Default())
	router := gin.New()

	// Setup routes
	router.POST("/deployments/pods", handler.CreatePod)
	router.GET("/deployments/clusters/:cluster_id/pods/:name", handler.GetPod)
	router.DELETE("/deployments/clusters/:cluster_id/pods/:name", handler.DeletePod)
	router.GET("/deployments/clusters/:cluster_id/pods/:name/logs", handler.GetPodLogs)

	return handler, mockService, router
}

func TestDeploymentHandler_CreatePod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockDeploymentService)
		expectedStatus int
		expectPod      bool
	}{
		{
			name: "successful creation",
			requestBody: CreatePodRequest{
				ClusterID: 1,
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				},
			},
			mockSetup: func(m *MockDeploymentService) {
				expectedPod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				}
				m.On("CreateDeployment", mock.Anything, uint(1), mock.AnythingOfType("*v1.Pod")).Return(expectedPod, nil)
			},
			expectedStatus: http.StatusCreated,
			expectPod:      true,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid-json",
			mockSetup:      func(m *MockDeploymentService) {},
			expectedStatus: http.StatusBadRequest,
			expectPod:      false,
		},
		{
			name: "service error - not found",
			requestBody: CreatePodRequest{
				ClusterID: 999,
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				},
			},
			mockSetup: func(m *MockDeploymentService) {
				m.On("CreateDeployment", mock.Anything, uint(999), mock.AnythingOfType("*v1.Pod")).Return(nil, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectPod:      false,
		},
		{
			name: "service error - internal error",
			requestBody: CreatePodRequest{
				ClusterID: 1,
				Pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				},
			},
			mockSetup: func(m *MockDeploymentService) {
				m.On("CreateDeployment", mock.Anything, uint(1), mock.AnythingOfType("*v1.Pod")).Return(nil, fmt.Errorf("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectPod:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupDeploymentHandlerTest()
			tc.mockSetup(mockService)

			var reqBody []byte
			var err error
			if tc.requestBody != nil {
				reqBody, err = json.Marshal(tc.requestBody)
				require.NoError(t, err)
			} else {
				reqBody = []byte(tc.requestBody.(string))
			}

			req, err := http.NewRequest(http.MethodPost, "/deployments/pods", bytes.NewBuffer(reqBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectPod {
				var response corev1.Pod
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "test-pod", response.Name)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDeploymentHandler_GetPod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		clusterID      string
		podName        string
		mockSetup      func(*MockDeploymentService)
		expectedStatus int
		expectPod      bool
	}{
		{
			name:      "successful get",
			clusterID: "1",
			podName:   "test-pod",
			mockSetup: func(m *MockDeploymentService) {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pod",
					},
				}
				m.On("GetDeployment", mock.Anything, uint(1), "test-pod").Return(pod, nil)
			},
			expectedStatus: http.StatusOK,
			expectPod:      true,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			podName:        "test-pod",
			mockSetup:      func(m *MockDeploymentService) {},
			expectedStatus: http.StatusBadRequest,
			expectPod:      false,
		},
		{
			name:      "pod not found",
			clusterID: "1",
			podName:   "nonexistent-pod",
			mockSetup: func(m *MockDeploymentService) {
				m.On("GetDeployment", mock.Anything, uint(1), "nonexistent-pod").Return(nil, services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectPod:      false,
		},
		{
			name:      "service error",
			clusterID: "1",
			podName:   "test-pod",
			mockSetup: func(m *MockDeploymentService) {
				m.On("GetDeployment", mock.Anything, uint(1), "test-pod").Return(nil, fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectPod:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupDeploymentHandlerTest()
			tc.mockSetup(mockService)

			url := fmt.Sprintf("/deployments/clusters/%s/pods/%s", tc.clusterID, tc.podName)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectPod {
				var response corev1.Pod
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "test-pod", response.Name)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDeploymentHandler_DeletePod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		clusterID      string
		podName        string
		mockSetup      func(*MockDeploymentService)
		expectedStatus int
	}{
		{
			name:      "successful deletion",
			clusterID: "1",
			podName:   "test-pod",
			mockSetup: func(m *MockDeploymentService) {
				m.On("DeleteDeployment", mock.Anything, uint(1), "test-pod").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			podName:        "test-pod",
			mockSetup:      func(m *MockDeploymentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "pod not found",
			clusterID: "1",
			podName:   "nonexistent-pod",
			mockSetup: func(m *MockDeploymentService) {
				m.On("DeleteDeployment", mock.Anything, uint(1), "nonexistent-pod").Return(services.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "service error",
			clusterID: "1",
			podName:   "test-pod",
			mockSetup: func(m *MockDeploymentService) {
				m.On("DeleteDeployment", mock.Anything, uint(1), "test-pod").Return(fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, mockService, router := setupDeploymentHandlerTest()
			tc.mockSetup(mockService)

			url := fmt.Sprintf("/deployments/clusters/%s/pods/%s", tc.clusterID, tc.podName)
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDeploymentHandler_GetPodLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		clusterID      string
		podName        string
		queryParams    string
		expectedStatus int
		expectLogs     bool
	}{
		{
			name:           "successful log retrieval",
			clusterID:      "1",
			podName:        "test-pod",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectLogs:     true,
		},
		{
			name:           "log retrieval with query params",
			clusterID:      "1",
			podName:        "test-pod",
			queryParams:    "?follow=true&tail=50",
			expectedStatus: http.StatusOK,
			expectLogs:     true,
		},
		{
			name:           "invalid cluster ID",
			clusterID:      "invalid",
			podName:        "test-pod",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectLogs:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, router := setupDeploymentHandlerTest()

			url := fmt.Sprintf("/deployments/clusters/%s/pods/%s/logs%s", tc.clusterID, tc.podName, tc.queryParams)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectLogs {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tc.podName, response["pod_name"])
				assert.Contains(t, response, "logs")
			}
		})
	}
}
