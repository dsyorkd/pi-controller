package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// ClusterHandler handles cluster-related API operations
type ClusterHandler struct {
	service *services.ClusterService
	logger  logger.Interface
}

// NewClusterHandler creates a new cluster handler
func NewClusterHandler(service *services.ClusterService, logger logger.Interface) *ClusterHandler {
	return &ClusterHandler{
		service: service,
		logger:  logger.WithField("handler", "cluster"),
	}
}

// Request and response types are now defined in the services package

// List returns all clusters
func (h *ClusterHandler) List(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	opts := services.ClusterListOptions{
		Limit:        limit,
		Offset:       offset,
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

// Create creates a new cluster
func (h *ClusterHandler) Create(c *gin.Context) {
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

// Get returns a specific cluster by ID
func (h *ClusterHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
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

// Update updates a cluster
func (h *ClusterHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
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

// Delete deletes a cluster
func (h *ClusterHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
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

// ListNodes returns all nodes for a specific cluster
func (h *ClusterHandler) ListNodes(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
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

// Status returns the status of a cluster
func (h *ClusterHandler) Status(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
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

// handleServiceError handles service layer errors and maps them to appropriate HTTP responses
func (h *ClusterHandler) handleServiceError(c *gin.Context, err error, message string) {
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