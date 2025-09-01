package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/spenceryork/pi-controller/internal/models"
	"github.com/spenceryork/pi-controller/internal/storage"
)

// NodeHandler handles node-related API operations
type NodeHandler struct {
	database *storage.Database
	logger   *logrus.Logger
}

// NewNodeHandler creates a new node handler
func NewNodeHandler(db *storage.Database, logger *logrus.Logger) *NodeHandler {
	return &NodeHandler{
		database: db,
		logger:   logger,
	}
}

// CreateNodeRequest represents the request to create a node
type CreateNodeRequest struct {
	Name         string          `json:"name" binding:"required"`
	IPAddress    string          `json:"ip_address" binding:"required"`
	MACAddress   string          `json:"mac_address"`
	Role         models.NodeRole `json:"role"`
	Architecture string          `json:"architecture"`
	Model        string          `json:"model"`
	SerialNumber string          `json:"serial_number"`
	CPUCores     int             `json:"cpu_cores"`
	Memory       int64           `json:"memory"`
	ClusterID    *uint           `json:"cluster_id,omitempty"`
}

// UpdateNodeRequest represents the request to update a node
type UpdateNodeRequest struct {
	Name         *string             `json:"name,omitempty"`
	IPAddress    *string             `json:"ip_address,omitempty"`
	MACAddress   *string             `json:"mac_address,omitempty"`
	Status       *models.NodeStatus  `json:"status,omitempty"`
	Role         *models.NodeRole    `json:"role,omitempty"`
	Architecture *string             `json:"architecture,omitempty"`
	Model        *string             `json:"model,omitempty"`
	SerialNumber *string             `json:"serial_number,omitempty"`
	CPUCores     *int                `json:"cpu_cores,omitempty"`
	Memory       *int64              `json:"memory,omitempty"`
	ClusterID    *uint               `json:"cluster_id,omitempty"`
	KubeVersion  *string             `json:"kube_version,omitempty"`
	NodeName     *string             `json:"node_name,omitempty"`
	OSVersion    *string             `json:"os_version,omitempty"`
	KernelVersion *string            `json:"kernel_version,omitempty"`
}

// List returns all nodes
func (h *NodeHandler) List(c *gin.Context) {
	var nodes []models.Node
	
	query := h.database.DB().Preload("Cluster").Preload("GPIODevices")
	
	// Filter by cluster if specified
	if clusterID := c.Query("cluster_id"); clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}
	
	// Filter by status if specified
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	result := query.Find(&nodes)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to list nodes")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve nodes",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// Create creates a new node
func (h *NodeHandler) Create(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	node := models.Node{
		Name:         req.Name,
		IPAddress:    req.IPAddress,
		MACAddress:   req.MACAddress,
		Role:         req.Role,
		Architecture: req.Architecture,
		Model:        req.Model,
		SerialNumber: req.SerialNumber,
		CPUCores:     req.CPUCores,
		Memory:       req.Memory,
		ClusterID:    req.ClusterID,
		Status:       models.NodeStatusDiscovered,
	}

	// Set default role if not provided
	if node.Role == "" {
		node.Role = models.NodeRoleWorker
	}

	result := h.database.DB().Create(&node)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to create node")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create node",
		})
		return
	}

	h.logger.WithField("node_id", node.ID).Info("Created new node")
	c.JSON(http.StatusCreated, node)
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

	var node models.Node
	result := h.database.DB().Preload("Cluster").Preload("GPIODevices").First(&node, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Node not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to get node")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve node",
		})
		return
	}

	c.JSON(http.StatusOK, node)
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

	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	var node models.Node
	result := h.database.DB().First(&node, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Node not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find node for update")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve node",
		})
		return
	}

	// Update fields if provided
	if req.Name != nil {
		node.Name = *req.Name
	}
	if req.IPAddress != nil {
		node.IPAddress = *req.IPAddress
	}
	if req.MACAddress != nil {
		node.MACAddress = *req.MACAddress
	}
	if req.Status != nil {
		node.Status = *req.Status
	}
	if req.Role != nil {
		node.Role = *req.Role
	}
	if req.Architecture != nil {
		node.Architecture = *req.Architecture
	}
	if req.Model != nil {
		node.Model = *req.Model
	}
	if req.SerialNumber != nil {
		node.SerialNumber = *req.SerialNumber
	}
	if req.CPUCores != nil {
		node.CPUCores = *req.CPUCores
	}
	if req.Memory != nil {
		node.Memory = *req.Memory
	}
	if req.ClusterID != nil {
		node.ClusterID = req.ClusterID
	}
	if req.KubeVersion != nil {
		node.KubeVersion = *req.KubeVersion
	}
	if req.NodeName != nil {
		node.NodeName = *req.NodeName
	}
	if req.OSVersion != nil {
		node.OSVersion = *req.OSVersion
	}
	if req.KernelVersion != nil {
		node.KernelVersion = *req.KernelVersion
	}

	result = h.database.DB().Save(&node)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to update node")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update node",
		})
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

	result := h.database.DB().Delete(&models.Node{}, uint(id))
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to delete node")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to delete node",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "Node not found",
		})
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

	var gpioDevices []models.GPIODevice
	result := h.database.DB().Where("node_id = ?", uint(id)).Find(&gpioDevices)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to list node GPIO devices")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve GPIO devices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"gpio_devices": gpioDevices,
		"count":        len(gpioDevices),
		"node_id":      uint(id),
	})
}

// Provision provisions a node for Kubernetes
func (h *NodeHandler) Provision(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	var node models.Node
	result := h.database.DB().First(&node, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Node not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find node for provisioning")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve node",
		})
		return
	}

	// Update node status to provisioning
	node.Status = models.NodeStatusProvisioning
	result = h.database.DB().Save(&node)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to update node status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update node status",
		})
		return
	}

	// TODO: Implement actual provisioning logic here
	// This would typically involve:
	// 1. SSH to the node
	// 2. Install Kubernetes components
	// 3. Join the cluster
	// 4. Update node status based on results

	h.logger.WithField("node_id", node.ID).Info("Started node provisioning")
	
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Node provisioning started",
		"node":    node,
	})
}

// Deprovision removes a node from Kubernetes
func (h *NodeHandler) Deprovision(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	var node models.Node
	result := h.database.DB().First(&node, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Node not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find node for deprovisioning")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve node",
		})
		return
	}

	// Update node status back to discovered
	node.Status = models.NodeStatusDiscovered
	node.ClusterID = nil
	node.KubeVersion = ""
	node.NodeName = ""
	
	result = h.database.DB().Save(&node)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to update node status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update node status",
		})
		return
	}

	// TODO: Implement actual deprovisioning logic here
	// This would typically involve:
	// 1. Drain the node
	// 2. Remove from cluster
	// 3. Clean up node components
	// 4. Update node status based on results

	h.logger.WithField("node_id", node.ID).Info("Started node deprovisioning")
	
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Node deprovisioning started",
		"node":    node,
	})
}