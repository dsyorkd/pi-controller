package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/provisioner"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// ProvisionerHandler handles provisioning-related HTTP requests
type ProvisionerHandler struct {
	provisionerService *services.ProvisioningService
	logger             logger.Interface
}

// NewProvisionerHandler creates a new provisioner handler
func NewProvisionerHandler(provisionerService *services.ProvisioningService, logger logger.Interface) *ProvisionerHandler {
	return &ProvisionerHandler{
		provisionerService: provisionerService,
		logger: logger.WithFields(map[string]interface{}{
			"handler": "provisioner",
		}),
	}
}

// ProvisionClusterRequest represents the HTTP request body for cluster provisioning
type ProvisionClusterRequest struct {
	MasterNodeID  uint                 `json:"master_node_id" binding:"required"`
	WorkerNodeIDs []uint               `json:"worker_node_ids,omitempty"`
	K3sConfig     services.K3sConfig   `json:"k3s_config,omitempty"`
	SSHConfig     ProvisionerSSHConfig `json:"ssh_config" binding:"required"`
}

// ProvisionNodeRequest represents the HTTP request body for single node provisioning
type ProvisionNodeRequest struct {
	Role      string               `json:"role" binding:"required,oneof=master worker"`
	K3sConfig services.K3sConfig   `json:"k3s_config,omitempty"`
	SSHConfig ProvisionerSSHConfig `json:"ssh_config" binding:"required"`
}

// ProvisionerSSHConfig represents SSH configuration for provisioning
type ProvisionerSSHConfig struct {
	Port           int    `json:"port,omitempty"`
	Username       string `json:"username,omitempty"`
	PrivateKeyPath string `json:"private_key_path,omitempty"`
	Password       string `json:"password,omitempty"`
	UseAgent       bool   `json:"use_agent,omitempty"`
	Timeout        int    `json:"timeout_seconds,omitempty"` // In seconds
}

// DeprovisionNodeRequest represents the HTTP request body for node deprovisioning
type DeprovisionNodeRequest struct {
	SSHConfig ProvisionerSSHConfig `json:"ssh_config" binding:"required"`
}

// ProvisionCluster provisions a complete K3s cluster
// @Summary Provision K3s cluster
// @Description Provision a complete K3s cluster with master and worker nodes
// @Tags provisioner
// @Accept json
// @Produce json
// @Param id path int true "Cluster ID"
// @Param request body ProvisionClusterRequest true "Provisioning request"
// @Success 200 {object} services.ProvisioningResult
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /clusters/{id}/provision [post]
func (h *ProvisionerHandler) ProvisionCluster(c *gin.Context) {
	clusterIDStr := c.Param("id")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("cluster_id", clusterIDStr).Error("Invalid cluster ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid cluster ID",
			"code":  "INVALID_ID",
		})
		return
	}

	var req ProvisionClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Convert HTTP request to service request
	serviceReq := services.ProvisionClusterRequest{
		ClusterID:     uint(clusterID),
		MasterNodeID:  req.MasterNodeID,
		WorkerNodeIDs: req.WorkerNodeIDs,
		K3sConfig:     req.K3sConfig,
		SSHConfig:     h.convertSSHConfig(req.SSHConfig),
	}

	h.logger.WithFields(map[string]interface{}{
		"cluster_id":      clusterID,
		"master_node_id":  req.MasterNodeID,
		"worker_node_ids": req.WorkerNodeIDs,
	}).Info("Received cluster provisioning request")

	result, err := h.provisionerService.ProvisionCluster(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.WithError(err).Error("Cluster provisioning failed")

		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "provisioning failed: " + err.Error(),
			"code":  "PROVISIONING_FAILED",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"cluster_id": clusterID,
		"success":    result.Success,
		"duration":   result.Duration,
	}).Info("Cluster provisioning completed")

	c.JSON(http.StatusOK, result)
}

// ProvisionNode provisions K3s on a single node
// @Summary Provision K3s node
// @Description Provision K3s on a single node (master or worker)
// @Tags provisioner
// @Accept json
// @Produce json
// @Param cluster_id path int true "Cluster ID"
// @Param node_id path int true "Node ID"
// @Param request body ProvisionNodeRequest true "Node provisioning request"
// @Success 200 {object} services.ProvisioningResult
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /clusters/{cluster_id}/nodes/{node_id}/provision [post]
func (h *ProvisionerHandler) ProvisionNode(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("cluster_id", clusterIDStr).Error("Invalid cluster ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid cluster ID",
			"code":  "INVALID_ID",
		})
		return
	}

	nodeIDStr := c.Param("node_id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("node_id", nodeIDStr).Error("Invalid node ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid node ID",
			"code":  "INVALID_ID",
		})
		return
	}

	var req ProvisionNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Convert role string to enum
	var role models.NodeRole
	switch req.Role {
	case "master":
		role = models.NodeRoleMaster
	case "worker":
		role = models.NodeRoleWorker
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid role: must be 'master' or 'worker'",
			"code":  "INVALID_ROLE",
		})
		return
	}

	// Convert HTTP request to service request
	serviceReq := services.ProvisionNodeRequest{
		NodeID:    uint(nodeID),
		ClusterID: uint(clusterID),
		Role:      role,
		K3sConfig: req.K3sConfig,
		SSHConfig: h.convertSSHConfig(req.SSHConfig),
	}

	h.logger.WithFields(map[string]interface{}{
		"cluster_id": clusterID,
		"node_id":    nodeID,
		"role":       req.Role,
	}).Info("Received node provisioning request")

	result, err := h.provisionerService.ProvisionNode(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.WithError(err).Error("Node provisioning failed")

		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "provisioning failed: " + err.Error(),
			"code":  "PROVISIONING_FAILED",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"node_id":  nodeID,
		"success":  result.Success,
		"duration": result.Duration,
	}).Info("Node provisioning completed")

	c.JSON(http.StatusOK, result)
}

// DeprovisionNode removes K3s from a node
// @Summary Deprovision K3s node
// @Description Remove K3s from a node and return it to discovered state
// @Tags provisioner
// @Accept json
// @Produce json
// @Param id path int true "Node ID"
// @Param request body DeprovisionNodeRequest true "Deprovisioning request"
// @Success 200 {object} services.ProvisioningResult
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /nodes/{id}/deprovision [post]
func (h *ProvisionerHandler) DeprovisionNode(c *gin.Context) {
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("node_id", nodeIDStr).Error("Invalid node ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid node ID",
			"code":  "INVALID_ID",
		})
		return
	}

	var req DeprovisionNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
			"code":  "INVALID_REQUEST",
		})
		return
	}

	h.logger.WithField("node_id", nodeID).Info("Received node deprovisioning request")

	result, err := h.provisionerService.DeprovisionNode(c.Request.Context(), uint(nodeID), h.convertSSHConfig(req.SSHConfig))
	if err != nil {
		h.logger.WithError(err).Error("Node deprovisioning failed")

		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "deprovisioning failed: " + err.Error(),
			"code":  "DEPROVISIONING_FAILED",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"node_id":  nodeID,
		"success":  result.Success,
		"duration": result.Duration,
	}).Info("Node deprovisioning completed")

	c.JSON(http.StatusOK, result)
}

// GetClusterToken retrieves the cluster join token from a master node
// @Summary Get cluster token
// @Description Retrieve the cluster join token from a master node
// @Tags provisioner
// @Accept json
// @Produce json
// @Param id path int true "Master Node ID"
// @Param request body ProvisionerSSHConfig true "SSH configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /nodes/{id}/cluster-token [post]
func (h *ProvisionerHandler) GetClusterToken(c *gin.Context) {
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("node_id", nodeIDStr).Error("Invalid node ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid node ID",
			"code":  "INVALID_ID",
		})
		return
	}

	var sshConfig ProvisionerSSHConfig
	if err := c.ShouldBindJSON(&sshConfig); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
			"code":  "INVALID_REQUEST",
		})
		return
	}

	h.logger.WithField("master_node_id", nodeID).Info("Received cluster token request")

	token, err := h.provisionerService.GetClusterToken(c.Request.Context(), uint(nodeID), h.convertSSHConfig(sshConfig))
	if err != nil {
		h.logger.WithError(err).Error("Failed to retrieve cluster token")

		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve cluster token: " + err.Error(),
			"code":  "TOKEN_RETRIEVAL_FAILED",
		})
		return
	}

	h.logger.WithField("master_node_id", nodeID).Info("Cluster token retrieved successfully")

	c.JSON(http.StatusOK, map[string]string{
		"cluster_token": token,
	})
}

// GetKubeConfig retrieves the kubeconfig from a master node
// @Summary Get kubeconfig
// @Description Retrieve the kubeconfig file from a master node
// @Tags provisioner
// @Accept json
// @Produce json
// @Param id path int true "Master Node ID"
// @Param request body ProvisionerSSHConfig true "SSH configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /nodes/{id}/kubeconfig [post]
func (h *ProvisionerHandler) GetKubeConfig(c *gin.Context) {
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.ParseUint(nodeIDStr, 10, 32)
	if err != nil {
		h.logger.WithError(err).WithField("node_id", nodeIDStr).Error("Invalid node ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid node ID",
			"code":  "INVALID_ID",
		})
		return
	}

	var sshConfig ProvisionerSSHConfig
	if err := c.ShouldBindJSON(&sshConfig); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
			"code":  "INVALID_REQUEST",
		})
		return
	}

	h.logger.WithField("master_node_id", nodeID).Info("Received kubeconfig request")

	kubeConfig, err := h.provisionerService.GetKubeConfig(c.Request.Context(), uint(nodeID), h.convertSSHConfig(sshConfig))
	if err != nil {
		h.logger.WithError(err).Error("Failed to retrieve kubeconfig")

		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
				"code":  "NOT_FOUND",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve kubeconfig: " + err.Error(),
			"code":  "KUBECONFIG_RETRIEVAL_FAILED",
		})
		return
	}

	h.logger.WithField("master_node_id", nodeID).Info("Kubeconfig retrieved successfully")

	c.JSON(http.StatusOK, map[string]string{
		"kubeconfig": kubeConfig,
	})
}

// convertSSHConfig converts HTTP SSH config to provisioner SSH config
func (h *ProvisionerHandler) convertSSHConfig(httpConfig ProvisionerSSHConfig) provisioner.SSHClientConfig {
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
