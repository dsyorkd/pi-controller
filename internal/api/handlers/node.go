package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/services"
	pb "github.com/dsyorkd/pi-controller/proto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Error message constants
const (
	errMsgBadRequest      = "Bad Request"
	errMsgInvalidNodeID   = "Invalid node ID"
	errMsgFailedToGetNode = "Failed to get node"
)

// NodeHandler handles node-related API operations
type NodeHandler struct {
	service    *services.NodeService
	logger     logger.Interface
	trustToken string // Trust token for node adoption
}

// NewNodeHandler creates a new node handler
func NewNodeHandler(service *services.NodeService, logger logger.Interface, trustToken string) *NodeHandler {
	return &NodeHandler{
		service:    service,
		logger:     logger.WithField("handler", "node"),
		trustToken: trustToken,
	}
}

// List godoc
// @Summary      List all nodes
// @Description  Get a list of all discovered nodes with optional filtering
// @Tags         nodes
// @Accept       json
// @Produce      json
// @Param        limit             query     int     false  "Limit number of results (default 50)"
// @Param        offset            query     int     false  "Offset for pagination (default 0)"
// @Param        include_gpio      query     boolean false  "Include GPIO pins in response"
// @Param        cluster_id        query     int     false  "Filter by cluster ID"
// @Param        status            query     string  false  "Filter by node status"
// @Param        role              query     string  false  "Filter by node role"
// @Param        discovery_method  query     string  false  "Filter by discovery method"
// @Param        node_type         query     string  false  "Filter by node type"
// @Success      200  {object}  object{data=[]object,total=int,limit=int,offset=int}
// @Failure      400  {object}  object{error=string,message=string}
// @Failure      401  {object}  object{error=string,message=string}
// @Failure      500  {object}  object{error=string,message=string}
// @Security     BearerAuth
// @Router       /nodes [get]
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

	if discoveryMethod := c.Query("discovery_method"); discoveryMethod != "" {
		method := models.DiscoveryMethod(discoveryMethod)
		opts.DiscoveryMethod = &method
	}

	if nodeType := c.Query("node_type"); nodeType != "" {
		nType := models.NodeType(nodeType)
		opts.NodeType = &nType
	}

	nodes, total, err := h.service.List(opts)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list nodes")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   nodes,
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
		h.handleServiceError(c, err, errMsgFailedToGetNode)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": node,
	})
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
	c.JSON(http.StatusCreated, gin.H{
		"data": node,
	})
}

// Update updates a node
func (h *NodeHandler) Update(c *gin.Context) { // nolint:dupl // similar pattern to cluster.Update but operates on different types
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
	c.JSON(http.StatusOK, gin.H{
		"data": node,
	})
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
func (h *NodeHandler) handleServiceError(c *gin.Context, err error, message string) { // nolint:dupl // Common error handling pattern duplicated in test helpers
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

	if services.IsInvalidInput(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
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

// AdoptNodeRequest represents the request to adopt a discovered node
type AdoptNodeRequest struct {
	TrustToken string `json:"trust_token"` // Optional - only required if trust token is configured
	ClusterID  *uint  `json:"cluster_id,omitempty"`
}

// Adopt adopts a discovered node by verifying the trust token
func (h *NodeHandler) Adopt(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	var req AdoptNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Verify trust token if configured
	if h.trustToken != "" && req.TrustToken != h.trustToken {
		h.logger.WithField("node_id", id).Warn("Node adoption attempted with invalid trust token")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid trust token",
		})
		return
	}

	// Get the node
	node, err := h.service.GetByID(uint(id), false)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get node")
		return
	}

	// Verify node is discovered (not manually added)
	if !node.IsDiscoveredAutomatically() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Node must be auto-discovered to adopt",
		})
		return
	}

	// Update node with cluster ID if provided and set status to ready
	updateReq := services.UpdateNodeRequest{}
	if req.ClusterID != nil {
		updateReq.ClusterID = req.ClusterID
	}

	// Set node status to ready when adopted
	readyStatus := models.NodeStatusReady
	updateReq.Status = &readyStatus

	updatedNode, err := h.service.Update(uint(id), updateReq)
	if err != nil {
		h.handleServiceError(c, err, "Failed to adopt node")
		return
	}

	h.logger.WithField("node_id", id).Info("Node adopted successfully")
	c.JSON(http.StatusOK, gin.H{
		"data":    updatedNode,
		"message": "Node adopted successfully",
	})
}

// GetSystemMetrics returns system metrics for a specific node
func (h *NodeHandler) GetSystemMetrics(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid node ID",
		})
		return
	}

	// Get the node to verify it exists
	node, err := h.service.GetByID(uint(id), false)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get node")
		return
	}

	// Call the node's agent gRPC endpoint to get real metrics
	agentAddr := fmt.Sprintf("%s:9091", node.IPAddress)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		h.logger.WithError(err).WithField("node_id", id).Error("Failed to create gRPC client for metrics")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Service Unavailable",
			"message": "Failed to connect to node agent",
		})
		return
	}
	defer conn.Close()

	client := pb.NewPiAgentServiceClient(conn)

	resp, err := client.GetSystemMetrics(ctx, &pb.GetSystemMetricsRequest{})
	if err != nil {
		h.logger.WithError(err).WithField("node_id", id).Error("Failed to get metrics from node agent")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Service Unavailable",
			"message": "Failed to retrieve metrics from node",
		})
		return
	}

	// Convert protobuf metrics to JSON response
	metrics := convertMetricsToJSON(node.ID, node.Name, resp.Metrics)

	c.JSON(http.StatusOK, gin.H{
		"data": metrics,
	})
}

// convertMetricsToJSON converts protobuf metrics to a JSON-friendly format
func convertMetricsToJSON(nodeID uint, hostname string, m *pb.SystemMetrics) gin.H {
	response := gin.H{
		"node_id":   nodeID,
		"hostname":  hostname,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// CPU metrics
	if m.Cpu != nil {
		response["cpu"] = gin.H{
			"usage_percent":  m.Cpu.UsagePercent,
			"per_core_usage": m.Cpu.PerCoreUsage,
			"user_percent":   m.Cpu.UserPercent,
			"system_percent": m.Cpu.SystemPercent,
			"idle_percent":   m.Cpu.IdlePercent,
			"iowait_percent": m.Cpu.IowaitPercent,
		}
	}

	// Memory metrics
	if m.Memory != nil {
		response["memory"] = gin.H{
			"total_bytes":        m.Memory.TotalBytes,
			"used_bytes":         m.Memory.UsedBytes,
			"available_bytes":    m.Memory.AvailableBytes,
			"free_bytes":         m.Memory.FreeBytes,
			"cached_bytes":       m.Memory.CachedBytes,
			"buffers_bytes":      m.Memory.BuffersBytes,
			"usage_percent":      m.Memory.UsagePercent,
			"swap_total_bytes":   m.Memory.SwapTotalBytes,
			"swap_used_bytes":    m.Memory.SwapUsedBytes,
			"swap_usage_percent": m.Memory.SwapUsagePercent,
		}
	}

	// Disk metrics
	if len(m.Disks) > 0 {
		var disks []gin.H
		for _, disk := range m.Disks {
			disks = append(disks, gin.H{
				"device":               disk.Device,
				"mountpoint":           disk.Mountpoint,
				"filesystem":           disk.Filesystem,
				"total_bytes":          disk.TotalBytes,
				"used_bytes":           disk.UsedBytes,
				"free_bytes":           disk.FreeBytes,
				"usage_percent":        disk.UsagePercent,
				"inodes_total":         disk.InodesTotal,
				"inodes_used":          disk.InodesUsed,
				"inodes_free":          disk.InodesFree,
				"inodes_usage_percent": disk.InodesUsagePercent,
			})
		}
		response["disks"] = disks
	}

	// Network metrics
	if len(m.Network) > 0 {
		var networks []gin.H
		for _, net := range m.Network {
			networks = append(networks, gin.H{
				"interface":    net.Interface,
				"bytes_sent":   net.BytesSent,
				"bytes_recv":   net.BytesRecv,
				"packets_sent": net.PacketsSent,
				"packets_recv": net.PacketsRecv,
				"err_in":       net.ErrIn,
				"err_out":      net.ErrOut,
				"drop_in":      net.DropIn,
				"drop_out":     net.DropOut,
			})
		}
		response["network"] = networks
	}

	// Thermal metrics
	if m.Thermal != nil && len(m.Thermal.Zones) > 0 {
		var zones []gin.H
		for _, zone := range m.Thermal.Zones {
			zones = append(zones, gin.H{
				"name":                zone.Name,
				"temperature_celsius": zone.TemperatureCelsius,
				"critical_temp":       zone.CriticalTemp,
				"status":              zone.Status,
			})
		}
		response["thermal"] = gin.H{
			"zones": zones,
		}
	}

	// Load metrics
	if m.Load != nil {
		response["load"] = gin.H{
			"load1":  m.Load.Load1,
			"load5":  m.Load.Load5,
			"load15": m.Load.Load15,
		}
	}

	// Process metrics
	if m.Processes != nil {
		response["processes"] = gin.H{
			"total":    m.Processes.Total,
			"running":  m.Processes.Running,
			"sleeping": m.Processes.Sleeping,
			"stopped":  m.Processes.Stopped,
			"zombie":   m.Processes.Zombie,
		}
	}

	return response
}
