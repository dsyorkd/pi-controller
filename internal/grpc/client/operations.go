package client

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/dsyorkd/pi-controller/proto"
)

// RegisterNode registers the node with the Pi Controller server
func (c *Client) RegisterNode(ctx context.Context, nodeInfo *NodeInfo) (*pb.Node, error) {
	if nodeInfo == nil {
		return nil, fmt.Errorf("nodeInfo is required")
	}
	
	c.logger.Info("Registering node with controller", 
		"node_name", nodeInfo.Name,
		"ip_address", nodeInfo.IPAddress)
	
	if err := c.ensureConnected(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	// Create the registration request
	req := &pb.CreateNodeRequest{
		Name:         nodeInfo.Name,
		IpAddress:    nodeInfo.IPAddress,
		MacAddress:   nodeInfo.MACAddress,
		Role:         pb.NodeRole_NODE_ROLE_WORKER, // Default to worker role
		Architecture: nodeInfo.Architecture,
		Model:        nodeInfo.Model,
		SerialNumber: nodeInfo.SerialNumber,
		CpuCores:     nodeInfo.CPUCores,
		Memory:       nodeInfo.Memory,
	}
	
	// Make the gRPC call
	node, err := c.client.CreateNode(callCtx, req)
	if err != nil {
		// Check if node already exists
		if grpcStatus, ok := status.FromError(err); ok {
			switch grpcStatus.Code() {
			case codes.AlreadyExists:
				c.logger.Info("Node already registered, attempting to update")
				return c.updateExistingNode(ctx, nodeInfo)
			case codes.InvalidArgument:
				return nil, fmt.Errorf("invalid node information: %w", err)
			case codes.Unavailable:
				return nil, fmt.Errorf("server unavailable: %w", err)
			}
		}
		return nil, fmt.Errorf("failed to register node: %w", err)
	}
	
	c.logger.Info("Node registered successfully", 
		"node_id", node.Id,
		"node_name", node.Name)
	
	// Store node info for future use
	c.SetNodeInfo(nodeInfo)
	
	return node, nil
}

// updateExistingNode attempts to find and update an existing node
func (c *Client) updateExistingNode(ctx context.Context, nodeInfo *NodeInfo) (*pb.Node, error) {
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	// List nodes to find the matching one
	listReq := &pb.ListNodesRequest{
		Page:     1,
		PageSize: 100,
	}
	
	listResp, err := c.client.ListNodes(callCtx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	
	// Find node by MAC address or IP address
	var targetNode *pb.Node
	for _, node := range listResp.Nodes {
		if node.MacAddress == nodeInfo.MACAddress || 
		   node.IpAddress == nodeInfo.IPAddress {
			targetNode = node
			break
		}
	}
	
	if targetNode == nil {
		return nil, fmt.Errorf("existing node not found")
	}
	
	// Update the node
	updateReq := &pb.UpdateNodeRequest{
		Id:            targetNode.Id,
		Name:          &nodeInfo.Name,
		IpAddress:     &nodeInfo.IPAddress,
		MacAddress:    &nodeInfo.MACAddress,
		Architecture:  &nodeInfo.Architecture,
		Model:         &nodeInfo.Model,
		SerialNumber:  &nodeInfo.SerialNumber,
		CpuCores:      &nodeInfo.CPUCores,
		Memory:        &nodeInfo.Memory,
		OsVersion:     &nodeInfo.OSVersion,
		KernelVersion: &nodeInfo.KernelVersion,
		Status:        pb.NodeStatus_NODE_STATUS_READY.Enum(),
	}
	
	updatedNode, err := c.client.UpdateNode(callCtx, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}
	
	c.logger.Info("Node updated successfully", 
		"node_id", updatedNode.Id,
		"node_name", updatedNode.Name)
	
	return updatedNode, nil
}

// UpdateNodeStatus updates the status of the registered node
func (c *Client) UpdateNodeStatus(ctx context.Context, nodeID uint32, status pb.NodeStatus) error {
	c.logger.Debug("Updating node status", 
		"node_id", nodeID,
		"status", status)
	
	if err := c.ensureConnected(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	req := &pb.UpdateNodeRequest{
		Id:     nodeID,
		Status: &status,
	}
	
	_, err := c.client.UpdateNode(callCtx, req)
	if err != nil {
		return fmt.Errorf("failed to update node status: %w", err)
	}
	
	c.logger.Debug("Node status updated successfully")
	return nil
}

// SendHeartbeat sends a heartbeat to indicate the node is alive
func (c *Client) SendHeartbeat(ctx context.Context) error {
	if err := c.ensureConnected(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	// Use health check as heartbeat mechanism
	req := &pb.HealthRequest{}
	
	_, err := c.client.Health(callCtx, req)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	
	c.logger.Debug("Heartbeat sent successfully")
	return nil
}

// HealthCheck performs a health check against the server
func (c *Client) HealthCheck(ctx context.Context) (*pb.HealthResponse, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	req := &pb.HealthRequest{}
	
	resp, err := c.client.Health(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	
	return resp, nil
}

// ReadGPIO reads the value from a GPIO device
func (c *Client) ReadGPIO(ctx context.Context, deviceID uint32) (*pb.ReadGPIOResponse, error) {
	c.logger.Debug("Reading GPIO device", "device_id", deviceID)
	
	if err := c.ensureConnected(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	req := &pb.ReadGPIORequest{
		Id: deviceID,
	}
	
	resp, err := c.client.ReadGPIO(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read GPIO: %w", err)
	}
	
	c.logger.Debug("GPIO read successful", 
		"device_id", resp.DeviceId,
		"pin", resp.Pin,
		"value", resp.Value)
	
	return resp, nil
}

// WriteGPIO writes a value to a GPIO device
func (c *Client) WriteGPIO(ctx context.Context, deviceID uint32, value int32) (*pb.WriteGPIOResponse, error) {
	c.logger.Debug("Writing GPIO device", 
		"device_id", deviceID,
		"value", value)
	
	if err := c.ensureConnected(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	req := &pb.WriteGPIORequest{
		Id:    deviceID,
		Value: value,
	}
	
	resp, err := c.client.WriteGPIO(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to write GPIO: %w", err)
	}
	
	c.logger.Debug("GPIO write successful", 
		"device_id", resp.DeviceId,
		"pin", resp.Pin,
		"value", resp.Value)
	
	return resp, nil
}

// heartbeatLoop runs the periodic heartbeat in a separate goroutine
func (c *Client) heartbeatLoop(ctx context.Context) {
	c.logger.Debug("Starting heartbeat loop", 
		"interval", c.config.HeartbeatInterval)
	
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			c.logger.Debug("Heartbeat loop stopped due to context cancellation")
			return
		case <-c.heartbeatStop:
			c.logger.Debug("Heartbeat loop stopped")
			return
		case <-ticker.C:
			if err := c.sendPeriodicHeartbeat(ctx); err != nil {
				c.logger.WithError(err).Warn("Heartbeat failed")
			}
		}
	}
}

// sendPeriodicHeartbeat sends a heartbeat with timeout
func (c *Client) sendPeriodicHeartbeat(ctx context.Context) error {
	heartbeatCtx, cancel := context.WithTimeout(ctx, c.config.HeartbeatTimeout)
	defer cancel()
	
	return c.SendHeartbeat(heartbeatCtx)
}

// CollectNodeInfo collects information about the current node
func CollectNodeInfo(nodeID, nodeName string) (*NodeInfo, error) {
	// Get network interfaces for IP and MAC
	var ipAddress, macAddress string
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue // Skip down or loopback interfaces
			}
			
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
					if ipNet.IP.To4() != nil {
						ipAddress = ipNet.IP.String()
						macAddress = iface.HardwareAddr.String()
						break
					}
				}
			}
			
			if ipAddress != "" {
				break
			}
		}
	}
	
	// Get system information
	nodeInfo := &NodeInfo{
		ID:           nodeID,
		Name:         nodeName,
		IPAddress:    ipAddress,
		MACAddress:   macAddress,
		Architecture: runtime.GOARCH,
		OSVersion:    runtime.GOOS,
		CPUCores:     int32(runtime.NumCPU()),
	}
	
	// Try to detect Raspberry Pi model
	if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
		nodeInfo.Model = detectRaspberryPiModel()
	} else {
		nodeInfo.Model = "generic-" + runtime.GOARCH
	}
	
	// Set default name if not provided
	if nodeInfo.Name == "" {
		if nodeInfo.Model != "" {
			nodeInfo.Name = fmt.Sprintf("%s-%s", nodeInfo.Model, 
				strings.ReplaceAll(nodeInfo.MACAddress, ":", ""))
		} else {
			nodeInfo.Name = fmt.Sprintf("node-%s", nodeInfo.IPAddress)
		}
	}
	
	return nodeInfo, nil
}

// detectRaspberryPiModel attempts to detect the Raspberry Pi model
func detectRaspberryPiModel() string {
	// This is a simple implementation - in a real system you might
	// read from /proc/cpuinfo or /sys/firmware/devicetree/base/model
	return "raspberry-pi"
}

// GetSystemInfo returns system information for the GetSystemInfo gRPC call
func (c *Client) GetSystemInfo(ctx context.Context) (*pb.SystemInfoResponse, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	
	callCtx, cancel := c.createCallContext(ctx)
	defer cancel()
	
	req := &pb.SystemInfoRequest{}
	
	resp, err := c.client.GetSystemInfo(callCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}
	
	return resp, nil
}