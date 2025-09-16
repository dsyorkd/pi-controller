package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// ServiceHandler handles Kubernetes service-related API operations
type ServiceHandler struct {
	service *services.K8sService
	logger  logger.Interface
}

// NewServiceHandler creates a new service handler
func NewServiceHandler(service *services.K8sService, logger logger.Interface) *ServiceHandler {
	return &ServiceHandler{
		service: service,
		logger:  logger.WithField("handler", "service"),
	}
}

// CreateServiceRequest represents the request to create a service
type CreateServiceRequest struct {
	ClusterID uint             `json:"cluster_id" binding:"required"`
	Namespace string           `json:"namespace,omitempty"`
	Service   *corev1.Service  `json:"service" binding:"required"`
}

// GetServicesResponse represents the response for listing services
type GetServicesResponse struct {
	Services []corev1.Service `json:"services"`
	Meta     MetaResponse     `json:"meta"`
}

// CreateService creates a new Kubernetes service in the specified cluster
// @Summary Create a service
// @Description Create a new Kubernetes service in a cluster
// @Tags services
// @Accept json
// @Produce json
// @Param request body CreateServiceRequest true "Service creation request"
// @Success 201 {object} corev1.Service
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /services [post]
func (h *ServiceHandler) CreateService(c *gin.Context) {
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

// GetService retrieves a specific service by cluster ID, namespace, and service name
// @Summary Get a service
// @Description Get a specific Kubernetes service by cluster ID, namespace, and name
// @Tags services
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param namespace path string false "Namespace (optional)"
// @Param name path string true "Service name"
// @Success 200 {object} corev1.Service
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /services/clusters/{cluster_id}/namespaces/{namespace}/services/{name} [get]
func (h *ServiceHandler) GetService(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
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

// ListServices retrieves all services from a specific cluster and namespace
// @Summary List services
// @Description List all Kubernetes services in a cluster and namespace
// @Tags services
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param namespace path string false "Namespace (optional)"
// @Param limit query int false "Limit number of results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {object} GetServicesResponse
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /services/clusters/{cluster_id}/namespaces/{namespace}/services [get]
func (h *ServiceHandler) ListServices(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	namespace := c.Param("namespace")

	// Parse query parameters for pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

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

// UpdateService updates a Kubernetes service
// @Summary Update a service
// @Description Update a specific Kubernetes service
// @Tags services
// @Accept json
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param namespace path string false "Namespace (optional)"
// @Param name path string true "Service name"
// @Param request body corev1.Service true "Service update request"
// @Success 200 {object} corev1.Service
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /services/clusters/{cluster_id}/namespaces/{namespace}/services/{name} [put]
func (h *ServiceHandler) UpdateService(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
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

	var service corev1.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	updatedService, err := h.service.UpdateService(c.Request.Context(), uint(clusterID), namespace, &service)
	if err != nil {
		h.handleServiceError(c, err, "Failed to update service")
		return
	}

	h.logger.WithField("cluster_id", clusterID).WithField("service_name", serviceName).Info("Updated service")
	c.JSON(http.StatusOK, updatedService)
}

// DeleteService deletes a Kubernetes service
// @Summary Delete a service
// @Description Delete a specific Kubernetes service
// @Tags services
// @Produce json
// @Param cluster_id path uint true "Cluster ID"
// @Param namespace path string false "Namespace (optional)"
// @Param name path string true "Service name"
// @Success 204
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /services/clusters/{cluster_id}/namespaces/{namespace}/services/{name} [delete]
func (h *ServiceHandler) DeleteService(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("cluster_id"), 10, 32)
	if err != nil {
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

	err = h.service.DeleteService(c.Request.Context(), uint(clusterID), namespace, serviceName)
	if err != nil {
		h.handleServiceError(c, err, "Failed to delete service")
		return
	}

	h.logger.WithField("cluster_id", clusterID).WithField("service_name", serviceName).Info("Deleted service")
	c.Status(http.StatusNoContent)
}

// handleServiceError handles service layer errors and maps them to appropriate HTTP responses
func (h *ServiceHandler) handleServiceError(c *gin.Context, err error, message string) {
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