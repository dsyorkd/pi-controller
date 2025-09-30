package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/provisioner"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// ClusterHandler handles cluster-related API operations
type ClusterHandler struct {
	service *services.ClusterService
	logger  logger.Interface
}

// hasPermission checks if user role has permission for cluster write operations
func (h *ClusterHandler) hasPermission(userRole, requiredRole string) bool {
	// Admin can access everything
	if userRole == middleware.RoleAdmin {
		return true
	}

	// Operator can access operator and viewer endpoints
	if userRole == middleware.RoleOperator && (requiredRole == middleware.RoleOperator || requiredRole == middleware.RoleViewer) {
		return true
	}

	// Viewer can only access viewer endpoints
	if userRole == middleware.RoleViewer && requiredRole == middleware.RoleViewer {
		return true
	}

	return false
}

// writeError writes an error response to the client
func (h *ClusterHandler) writeError(w *gin.Context, status int, message string) {
	w.JSON(status, gin.H{
		"error":   http.StatusText(status),
		"message": message,
	})
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
		Limit:  limit,
		Offset: offset,
	}

	clusters, total, err := h.service.List(opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list clusters")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   clusters,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Create creates a new cluster
func (h *ClusterHandler) Create(c *gin.Context) {
	// Check permissions for cluster write operations
	userRole := middleware.GetUserRole(c)
	if !h.hasPermission(userRole, middleware.RoleOperator) {
		h.writeError(c, http.StatusForbidden, "insufficient permissions")
		return
	}

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
		logger := h.logger.WithField("handler", "ClusterHandler").WithField("method", "Create")
		logger.WithError(err).Error("failed to create cluster")
		h.handleServiceError(c, err, "Failed to create cluster")
		return
	}

	h.logger.WithField("cluster_id", cluster.ID).Info("Created new cluster")
	c.JSON(http.StatusCreated, gin.H{
		"data": cluster,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"data": cluster,
	})
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
	c.JSON(http.StatusOK, gin.H{
		"data": cluster,
	})
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

	// Use the enhanced GetClusterStatus method for detailed information
	detailedStatus, err := h.service.GetClusterStatus(c.Request.Context(), uint(id))
	if err != nil {
		h.handleServiceError(c, err, "Failed to get cluster status")
		return
	}

	c.JSON(http.StatusOK, detailedStatus)
}

// HTTP request types for cluster lifecycle operations

// ProvisionClusterHTTPRequest represents HTTP request for cluster provisioning
type ProvisionClusterHTTPRequest struct {
	MasterNodeID  uint               `json:"master_node_id" binding:"required"`
	WorkerNodeIDs []uint             `json:"worker_node_ids,omitempty"`
	K3sConfig     services.K3sConfig `json:"k3s_config,omitempty"`
	SSHConfig     ClusterSSHConfig   `json:"ssh_config" binding:"required"`
}

// DeprovisionClusterHTTPRequest represents HTTP request for cluster deprovisioning
type DeprovisionClusterHTTPRequest struct {
	SSHConfig ClusterSSHConfig `json:"ssh_config" binding:"required"`
}

// ScaleClusterHTTPRequest represents HTTP request for cluster scaling
type ScaleClusterHTTPRequest struct {
	NodeCount uint             `json:"node_count" binding:"required,min=1"`
	SSHConfig ClusterSSHConfig `json:"ssh_config" binding:"required"`
}

// ClusterSSHConfig represents SSH configuration for cluster operations
type ClusterSSHConfig struct {
	Port           int    `json:"port,omitempty"`
	Username       string `json:"username,omitempty"`
	PrivateKeyPath string `json:"private_key_path,omitempty"`
	Password       string `json:"password,omitempty"`
	UseAgent       bool   `json:"use_agent,omitempty"`
	Timeout        int    `json:"timeout_seconds,omitempty"` // In seconds
}

// ProvisionCluster provisions a K3s cluster
// @Summary Provision cluster
// @Description Provision a K3s cluster with master and worker nodes
// @Tags clusters
// @Accept json
// @Produce json
// @Param id path int true "Cluster ID"
// @Param request body ProvisionClusterHTTPRequest true "Provisioning request"
// @Success 202 {object} services.ProvisioningResult
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /clusters/{id}/provision [post]
func (h *ClusterHandler) ProvisionCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var req ProvisionClusterHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Convert HTTP request to service request
	serviceReq := services.ClusterProvisionRequest{
		MasterNodeID:  req.MasterNodeID,
		WorkerNodeIDs: req.WorkerNodeIDs,
		K3sConfig:     req.K3sConfig,
		SSHConfig:     h.convertClusterSSHConfig(req.SSHConfig),
	}

	h.logger.WithFields(map[string]interface{}{
		"cluster_id":      id,
		"master_node_id":  req.MasterNodeID,
		"worker_node_ids": req.WorkerNodeIDs,
	}).Info("Received cluster provisioning request")

	// This is an async operation, return 202 Accepted
	result, err := h.service.ProvisionCluster(c.Request.Context(), uint(id), serviceReq)
	if err != nil {
		h.handleServiceError(c, err, "Failed to provision cluster")
		return
	}

	h.logger.WithField("cluster_id", id).Info("Cluster provisioning initiated")
	c.JSON(http.StatusAccepted, result)
}

// DeprovisionCluster tears down a K3s cluster
// @Summary Deprovision cluster
// @Description Tear down a K3s cluster and return nodes to discovered state
// @Tags clusters
// @Accept json
// @Produce json
// @Param id path int true "Cluster ID"
// @Param request body DeprovisionClusterHTTPRequest true "Deprovisioning request"
// @Success 202 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /clusters/{id}/deprovision [post]
func (h *ClusterHandler) DeprovisionCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var req DeprovisionClusterHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Convert HTTP request to service request
	serviceReq := services.ClusterDeprovisionRequest{
		SSHConfig: h.convertClusterSSHConfig(req.SSHConfig),
	}

	h.logger.WithField("cluster_id", id).Info("Received cluster deprovisioning request")

	// This is an async operation, return 202 Accepted
	err = h.service.DeprovisionCluster(c.Request.Context(), uint(id), serviceReq)
	if err != nil {
		h.handleServiceError(c, err, "Failed to deprovision cluster")
		return
	}

	h.logger.WithField("cluster_id", id).Info("Cluster deprovisioning initiated")
	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Cluster deprovisioning initiated",
		"cluster_id": id,
	})
}

// ScaleCluster scales a cluster to the specified number of nodes
// @Summary Scale cluster
// @Description Scale a cluster to the specified number of nodes
// @Tags clusters
// @Accept json
// @Produce json
// @Param id path int true "Cluster ID"
// @Param request body ScaleClusterHTTPRequest true "Scaling request"
// @Success 202 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /clusters/{id}/scale [post]
func (h *ClusterHandler) ScaleCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid cluster ID",
		})
		return
	}

	var req ScaleClusterHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Convert HTTP request to service request
	serviceReq := services.ClusterScaleRequest{
		NodeCount: req.NodeCount,
		SSHConfig: h.convertClusterSSHConfig(req.SSHConfig),
	}

	h.logger.WithFields(map[string]interface{}{
		"cluster_id": id,
		"node_count": req.NodeCount,
	}).Info("Received cluster scaling request")

	// This is an async operation, return 202 Accepted
	err = h.service.ScaleCluster(c.Request.Context(), uint(id), serviceReq)
	if err != nil {
		h.handleServiceError(c, err, "Failed to scale cluster")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"cluster_id": id,
		"node_count": req.NodeCount,
	}).Info("Cluster scaling initiated")

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Cluster scaling initiated",
		"cluster_id": id,
		"node_count": req.NodeCount,
	})
}

// convertClusterSSHConfig converts HTTP SSH config to provisioner SSH config
func (h *ClusterHandler) convertClusterSSHConfig(httpConfig ClusterSSHConfig) provisioner.SSHClientConfig {
	config := provisioner.DefaultSSHClientConfig()

	if httpConfig.Port != 0 {
		config.Port = httpConfig.Port
	}
	if httpConfig.Username != "" {
		config.Username = httpConfig.Username
	}
	if httpConfig.PrivateKeyPath != "" {
		config.PrivateKeyPath = httpConfig.PrivateKeyPath
	}
	if httpConfig.Password != "" {
		config.Password = httpConfig.Password
	}
	if httpConfig.UseAgent {
		config.UseAgent = true
	}
	if httpConfig.Timeout > 0 {
		config.Timeout = time.Duration(httpConfig.Timeout) * time.Second
	}

	return config
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

	if services.IsValidationFailed(err) || services.IsInvalidInput(err) {
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
