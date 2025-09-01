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

// ClusterHandler handles cluster-related API operations
type ClusterHandler struct {
	database *storage.Database
	logger   *logrus.Logger
}

// NewClusterHandler creates a new cluster handler
func NewClusterHandler(db *storage.Database, logger *logrus.Logger) *ClusterHandler {
	return &ClusterHandler{
		database: db,
		logger:   logger,
	}
}

// CreateClusterRequest represents the request to create a cluster
type CreateClusterRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	Version        string `json:"version"`
	MasterEndpoint string `json:"master_endpoint"`
}

// UpdateClusterRequest represents the request to update a cluster
type UpdateClusterRequest struct {
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Status         *models.ClusterStatus  `json:"status,omitempty"`
	Version        *string                `json:"version,omitempty"`
	MasterEndpoint *string                `json:"master_endpoint,omitempty"`
}

// List returns all clusters
func (h *ClusterHandler) List(c *gin.Context) {
	var clusters []models.Cluster
	
	result := h.database.DB().Preload("Nodes").Find(&clusters)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to list clusters")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve clusters",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusters": clusters,
		"count":    len(clusters),
	})
}

// Create creates a new cluster
func (h *ClusterHandler) Create(c *gin.Context) {
	var req CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	cluster := models.Cluster{
		Name:           req.Name,
		Description:    req.Description,
		Version:        req.Version,
		MasterEndpoint: req.MasterEndpoint,
		Status:         models.ClusterStatusPending,
	}

	result := h.database.DB().Create(&cluster)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to create cluster")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create cluster",
		})
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

	var cluster models.Cluster
	result := h.database.DB().Preload("Nodes").First(&cluster, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Cluster not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to get cluster")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error", 
			"message": "Failed to retrieve cluster",
		})
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

	var req UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	var cluster models.Cluster
	result := h.database.DB().First(&cluster, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Cluster not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to find cluster for update")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve cluster",
		})
		return
	}

	// Update fields if provided
	if req.Name != nil {
		cluster.Name = *req.Name
	}
	if req.Description != nil {
		cluster.Description = *req.Description
	}
	if req.Status != nil {
		cluster.Status = *req.Status
	}
	if req.Version != nil {
		cluster.Version = *req.Version
	}
	if req.MasterEndpoint != nil {
		cluster.MasterEndpoint = *req.MasterEndpoint
	}

	result = h.database.DB().Save(&cluster)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to update cluster")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update cluster",
		})
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

	result := h.database.DB().Delete(&models.Cluster{}, uint(id))
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to delete cluster")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to delete cluster",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "Cluster not found",
		})
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

	var nodes []models.Node
	result := h.database.DB().Where("cluster_id = ?", uint(id)).Find(&nodes)
	if result.Error != nil {
		h.logger.WithError(result.Error).Error("Failed to list cluster nodes")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve cluster nodes",
		})
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

	var cluster models.Cluster
	result := h.database.DB().Preload("Nodes").First(&cluster, uint(id))
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Cluster not found",
			})
			return
		}
		
		h.logger.WithError(result.Error).Error("Failed to get cluster status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve cluster status",
		})
		return
	}

	// Calculate node statistics
	var readyNodes, totalNodes int
	for _, node := range cluster.Nodes {
		totalNodes++
		if node.IsReady() {
			readyNodes++
		}
	}

	status := gin.H{
		"cluster_id":     cluster.ID,
		"name":           cluster.Name,
		"status":         cluster.Status,
		"version":        cluster.Version,
		"master_endpoint": cluster.MasterEndpoint,
		"nodes": gin.H{
			"total": totalNodes,
			"ready": readyNodes,
		},
		"healthy":    cluster.IsHealthy(),
		"created_at": cluster.CreatedAt,
		"updated_at": cluster.UpdatedAt,
	}

	c.JSON(http.StatusOK, status)
}