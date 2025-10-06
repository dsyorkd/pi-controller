package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/clustering"
	"github.com/dsyorkd/pi-controller/internal/clustering/health"
	"github.com/dsyorkd/pi-controller/internal/logger"
)

// RaftClusterHandler handles Raft cluster management API operations
type RaftClusterHandler struct {
	cluster       *clustering.RaftCluster
	healthChecker *health.HealthChecker
	logger        logger.Interface
}

// NewRaftClusterHandler creates a new Raft cluster handler
func NewRaftClusterHandler(cluster *clustering.RaftCluster, healthChecker *health.HealthChecker, logger logger.Interface) *RaftClusterHandler {
	return &RaftClusterHandler{
		cluster:       cluster,
		healthChecker: healthChecker,
		logger:        logger.WithField("handler", "raft_cluster"),
	}
}

// GetStatus returns the current cluster status
// GET /api/v1/raft/status
func (h *RaftClusterHandler) GetStatus(c *gin.Context) {
	if h.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Clustering not enabled",
			"message": "Controller clustering is not enabled on this instance",
		})
		return
	}

	state := h.cluster.GetState()
	leader := h.cluster.GetLeader()
	isLeader := h.cluster.IsLeader()

	members, err := h.cluster.GetServers()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get cluster members")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve cluster members",
		})
		return
	}

	// Get health status if available
	var healthStatus *health.HealthStatus
	if h.healthChecker != nil {
		healthStatus = h.healthChecker.GetStatus()
	}

	// Get Raft statistics
	stats := h.cluster.Stats()

	c.JSON(http.StatusOK, gin.H{
		"state":         state.String(),
		"leader":        leader,
		"is_leader":     isLeader,
		"members":       members,
		"member_count":  len(members),
		"health":        healthStatus,
		"raft_stats":    stats,
	})
}

// GetMembers returns all cluster members
// GET /api/v1/raft/members
func (h *RaftClusterHandler) GetMembers(c *gin.Context) {
	if h.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Clustering not enabled",
			"message": "Controller clustering is not enabled on this instance",
		})
		return
	}

	members, err := h.cluster.GetServers()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get cluster members")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve cluster members",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
		"total":   len(members),
	})
}

// JoinMemberRequest represents a request to join a node to the cluster
type JoinMemberRequest struct {
	NodeID  string `json:"node_id" binding:"required"`
	Address string `json:"address" binding:"required"`
}

// JoinMember adds a new member to the cluster
// POST /api/v1/raft/members
func (h *RaftClusterHandler) JoinMember(c *gin.Context) {
	if h.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Clustering not enabled",
			"message": "Controller clustering is not enabled on this instance",
		})
		return
	}

	// Only leaders can add members
	if !h.cluster.IsLeader() {
		leader := h.cluster.GetLeader()
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Not the leader",
			"message": "Only the cluster leader can add members",
			"leader":  leader,
		})
		return
	}

	var req JoinMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"node_id": req.NodeID,
		"address": req.Address,
	}).Info("Adding node to cluster")

	if err := h.cluster.Join(req.NodeID, req.Address); err != nil {
		h.logger.WithError(err).Error("Failed to add node to cluster")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to add node to cluster: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Node successfully added to cluster",
		"node_id": req.NodeID,
		"address": req.Address,
	})
}

// RemoveMember removes a member from the cluster
// DELETE /api/v1/raft/members/:id
func (h *RaftClusterHandler) RemoveMember(c *gin.Context) {
	if h.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Clustering not enabled",
			"message": "Controller clustering is not enabled on this instance",
		})
		return
	}

	// Only leaders can remove members
	if !h.cluster.IsLeader() {
		leader := h.cluster.GetLeader()
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Not the leader",
			"message": "Only the cluster leader can remove members",
			"leader":  leader,
		})
		return
	}

	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Node ID is required",
		})
		return
	}

	h.logger.WithField("node_id", nodeID).Info("Removing node from cluster")

	if err := h.cluster.Leave(nodeID); err != nil {
		h.logger.WithError(err).Error("Failed to remove node from cluster")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to remove node from cluster: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Node successfully removed from cluster",
		"node_id": nodeID,
	})
}

// GetHealth returns the cluster health status
// GET /api/v1/raft/health
func (h *RaftClusterHandler) GetHealth(c *gin.Context) {
	if h.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Clustering not enabled",
			"message": "Controller clustering is not enabled on this instance",
		})
		return
	}

	if h.healthChecker == nil {
		c.JSON(http.StatusOK, gin.H{
			"healthy": true,
			"message": "Health checking not configured",
		})
		return
	}

	status := h.healthChecker.GetStatus()

	if status.Healthy {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

// GetLeader returns the current cluster leader
// GET /api/v1/raft/leader
func (h *RaftClusterHandler) GetLeader(c *gin.Context) {
	if h.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Clustering not enabled",
			"message": "Controller clustering is not enabled on this instance",
		})
		return
	}

	leader := h.cluster.GetLeader()
	isLeader := h.cluster.IsLeader()

	c.JSON(http.StatusOK, gin.H{
		"leader":    leader,
		"is_leader": isLeader,
	})
}
