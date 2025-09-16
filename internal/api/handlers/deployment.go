package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// DeploymentHandler handles pod deployment-related API operations
type DeploymentHandler struct {
	service *services.DeploymentService
	logger  logger.Interface
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(service *services.DeploymentService, logger logger.Interface) *DeploymentHandler {
	return &DeploymentHandler{
		service: service,
		logger:  logger.WithField("handler", "deployment"),
	}
}

// CreatePodRequest represents the request to create a pod
type CreatePodRequest struct {
	ClusterID uint        `json:"cluster_id" binding:"required"`
	Pod       *corev1.Pod `json:"pod" binding:"required"`
}

// GetPodsResponse represents the response for listing pods
type GetPodsResponse struct {
	Pods []corev1.Pod `json:"pods"`
	Meta MetaResponse `json:"meta"`
}

// MetaResponse represents metadata for list responses
type MetaResponse struct {
	Count  int `json:"count"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// CreatePod creates a new pod in the specified cluster
// @Summary Create a pod
// @Description Create a new pod in a Kubernetes cluster
// @Tags deployments
// @Accept json
// @Produce json
// @Param request body CreatePodRequest true "Pod creation request"
// @Success 201 {object} corev1.Pod
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /deployments/pods [post]
func (h *DeploymentHandler) CreatePod(c *gin.Context) {
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

// GetPod retrieves a specific pod by cluster ID and pod name
// @Summary Get a pod
// @Description Get a specific pod by cluster ID and name
// @Tags deployments
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param name path string true "Pod name"
// @Success 200 {object} corev1.Pod
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /deployments/clusters/{cluster_id}/pods/{name} [get]
func (h *DeploymentHandler) GetPod(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
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

// DeletePod deletes a pod by cluster ID and pod name
// @Summary Delete a pod
// @Description Delete a specific pod by cluster ID and name
// @Tags deployments
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param name path string true "Pod name"
// @Success 204
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /deployments/clusters/{cluster_id}/pods/{name} [delete]
func (h *DeploymentHandler) DeletePod(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
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

	err = h.service.DeleteDeployment(c.Request.Context(), uint(clusterID), podName)
	if err != nil {
		h.handleServiceError(c, err, "Failed to delete pod")
		return
	}

	h.logger.WithField("cluster_id", clusterID).WithField("pod_name", podName).Info("Deleted pod")
	c.Status(http.StatusNoContent)
}

// GetPodLogs retrieves logs for a specific pod
// Note: This is a placeholder as the DeploymentService doesn't currently implement log retrieval
// @Summary Get pod logs
// @Description Get logs for a specific pod
// @Tags deployments
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param name path string true "Pod name"
// @Param follow query bool false "Follow logs"
// @Param tail query int false "Number of lines to tail"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /deployments/clusters/{cluster_id}/pods/{name}/logs [get]
func (h *DeploymentHandler) GetPodLogs(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
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
	tail, _ := strconv.Atoi(c.DefaultQuery("tail", "100"))

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
func (h *DeploymentHandler) handleServiceError(c *gin.Context, err error, message string) {
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