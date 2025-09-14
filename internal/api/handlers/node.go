package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// NodeHandler handles node-related API operations
type NodeHandler struct {
	service *services.NodeService
	logger  logger.Interface
}

// NewNodeHandler creates a new node handler
func NewNodeHandler(service *services.NodeService, logger logger.Interface) *NodeHandler {
	return &NodeHandler{
		service: service,
		logger:  logger.WithField("handler", "node"),
	}
}

// List returns all nodes with optional filtering
func (h *NodeHandler) List(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	includeGPIO := c.DefaultQuery("include_gpio", "false") == "true"

	opts := services.NodeListOptions{
		Limit:       limit,
		Offset:      offset,
		IncludeGPIO: includeGPIO,
	}

	// Parse optional filters
	if clusterID := c.Query("cluster_id"); clusterID != "" {
		if id, err := strconv.ParseUint(clusterID, 10, 32); err == nil {
			cid := uint(id)
			opts.ClusterID = &cid
		}
	}

	if status := c.Query("status"); status != "" {
		nodeStatus := models.NodeStatus(status)
		opts.Status = &nodeStatus
	}

	if role := c.Query("role"); role != "" {
		nodeRole := models.NodeRole(role)
		opts.Role = &nodeRole
	}

	nodes, total, err := h.service.List(opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list nodes")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes":  nodes,
		"count":  len(nodes),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Get returns a specific node by ID
func (h *NodeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	includeGPIO := c.DefaultQuery("include_gpio", "false") == "true"

	node, err := h.service.GetByID(uint(id), includeGPIO)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get node")
		return
	}

	c.JSON(http.StatusOK, node)
}

// Create creates a new node
func (h *NodeHandler) Create(c *gin.Context) {
	var req services.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	node, err := h.service.Create(req)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create node")
		return
	}

	h.logger.WithField("node_id", node.ID).Info("Created new node")
	c.JSON(http.StatusCreated, node)
}

// Update updates a node
func (h *NodeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	var req services.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	node, err := h.service.Update(uint(id), req)
	if err != nil {
		h.handleServiceError(c, err, "Failed to update node")
		return
	}

	h.logger.WithField("node_id", node.ID).Info("Updated node")
	c.JSON(http.StatusOK, node)
}

// Delete deletes a node
func (h *NodeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		h.handleServiceError(c, err, "Failed to delete node")
		return
	}

	h.logger.WithField("node_id", id).Info("Deleted node")
	c.JSON(http.StatusNoContent, nil)
}

// ListGPIO returns all GPIO devices for a specific node
func (h *NodeHandler) ListGPIO(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	devices, err := h.service.GetGPIODevices(uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to list node GPIO devices")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"gpio_devices": devices,
		"count":        len(devices),
		"node_id":      uint(id),
	})
}


// handleServiceError handles service layer errors and maps them to appropriate HTTP responses
func (h *NodeHandler) handleServiceError(c *gin.Context, err error, message string) {
	h.logger.WithError(err).Error(message)

	if services.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "Node not found",
		})
		return
	}

	if services.IsAlreadyExists(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "Node with that name or IP address already exists",
		})
		return
	}

	if err == services.ErrHasAssociatedResources {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Conflict",
			"message": "Cannot delete node with associated GPIO devices",
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
