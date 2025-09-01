package server

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/spenceryork/pi-controller/internal/models"
	"github.com/spenceryork/pi-controller/internal/storage"
	pb "github.com/spenceryork/pi-controller/proto"
)

// PiControllerServer implements the gRPC PiControllerService
type PiControllerServer struct {
	pb.UnimplementedPiControllerServiceServer
	database *storage.Database
	logger   *logrus.Logger
}

// Health returns the health status of the service
func (s *PiControllerServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:    "ok",
		Timestamp: timestamppb.New(time.Now()),
		Version:   "dev",
		Uptime:    "unknown", // TODO: Calculate actual uptime
	}, nil
}

// CreateCluster creates a new cluster
func (s *PiControllerServer) CreateCluster(ctx context.Context, req *pb.CreateClusterRequest) (*pb.Cluster, error) {
	cluster := models.Cluster{
		Name:           req.Name,
		Description:    req.Description,
		Version:        req.Version,
		MasterEndpoint: req.MasterEndpoint,
		Status:         models.ClusterStatusPending,
	}

	result := s.database.DB().Create(&cluster)
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to create cluster")
		return nil, status.Error(codes.Internal, "Failed to create cluster")
	}

	return s.clusterToProto(&cluster), nil
}

// GetCluster retrieves a cluster by ID
func (s *PiControllerServer) GetCluster(ctx context.Context, req *pb.GetClusterRequest) (*pb.Cluster, error) {
	var cluster models.Cluster
	result := s.database.DB().Preload("Nodes").First(&cluster, req.Id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "Cluster not found")
		}
		s.logger.WithError(result.Error).Error("Failed to get cluster")
		return nil, status.Error(codes.Internal, "Failed to retrieve cluster")
	}

	return s.clusterToProto(&cluster), nil
}

// ListClusters retrieves all clusters
func (s *PiControllerServer) ListClusters(ctx context.Context, req *pb.ListClustersRequest) (*pb.ListClustersResponse, error) {
	var clusters []models.Cluster
	
	query := s.database.DB().Preload("Nodes")
	
	// Apply pagination if specified
	if req.PageSize > 0 {
		offset := 0
		if req.Page > 0 {
			offset = int(req.Page-1) * int(req.PageSize)
		}
		query = query.Offset(offset).Limit(int(req.PageSize))
	}

	result := query.Find(&clusters)
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to list clusters")
		return nil, status.Error(codes.Internal, "Failed to retrieve clusters")
	}

	// Convert to protobuf
	pbClusters := make([]*pb.Cluster, len(clusters))
	for i, cluster := range clusters {
		pbClusters[i] = s.clusterToProto(&cluster)
	}

	// Get total count for pagination
	var totalCount int64
	s.database.DB().Model(&models.Cluster{}).Count(&totalCount)

	return &pb.ListClustersResponse{
		Clusters:   pbClusters,
		TotalCount: int32(totalCount),
	}, nil
}

// CreateNode creates a new node
func (s *PiControllerServer) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.Node, error) {
	node := models.Node{
		Name:         req.Name,
		IPAddress:    req.IpAddress,
		MACAddress:   req.MacAddress,
		Role:         s.nodeRoleFromProto(req.Role),
		Architecture: req.Architecture,
		Model:        req.Model,
		SerialNumber: req.SerialNumber,
		CPUCores:     int(req.CpuCores),
		Memory:       req.Memory,
		Status:       models.NodeStatusDiscovered,
	}

	if req.ClusterId != nil {
		node.ClusterID = req.ClusterId
	}

	result := s.database.DB().Create(&node)
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to create node")
		return nil, status.Error(codes.Internal, "Failed to create node")
	}

	return s.nodeToProto(&node), nil
}

// GetNode retrieves a node by ID
func (s *PiControllerServer) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.Node, error) {
	var node models.Node
	result := s.database.DB().Preload("GPIODevices").First(&node, req.Id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "Node not found")
		}
		s.logger.WithError(result.Error).Error("Failed to get node")
		return nil, status.Error(codes.Internal, "Failed to retrieve node")
	}

	return s.nodeToProto(&node), nil
}

// ReadGPIO reads the current value from a GPIO device
func (s *PiControllerServer) ReadGPIO(ctx context.Context, req *pb.ReadGPIORequest) (*pb.ReadGPIOResponse, error) {
	var device models.GPIODevice
	result := s.database.DB().First(&device, req.Id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "GPIO device not found")
		}
		s.logger.WithError(result.Error).Error("Failed to get GPIO device")
		return nil, status.Error(codes.Internal, "Failed to retrieve GPIO device")
	}

	if !device.IsActive() {
		return nil, status.Error(codes.FailedPrecondition, "GPIO device is not active")
	}

	// TODO: Implement actual GPIO read operation
	// For now, return the stored value
	
	// Record the reading
	reading := models.GPIOReading{
		DeviceID:  device.ID,
		Value:     float64(device.Value),
		Timestamp: time.Now(),
	}
	s.database.DB().Create(&reading)

	return &pb.ReadGPIOResponse{
		DeviceId:  device.ID,
		Pin:       int32(device.PinNumber),
		Value:     float64(device.Value),
		Timestamp: timestamppb.New(reading.Timestamp),
	}, nil
}

// WriteGPIO writes a value to a GPIO device
func (s *PiControllerServer) WriteGPIO(ctx context.Context, req *pb.WriteGPIORequest) (*pb.WriteGPIOResponse, error) {
	var device models.GPIODevice
	result := s.database.DB().First(&device, req.Id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "GPIO device not found")
		}
		s.logger.WithError(result.Error).Error("Failed to get GPIO device")
		return nil, status.Error(codes.Internal, "Failed to retrieve GPIO device")
	}

	if !device.IsActive() {
		return nil, status.Error(codes.FailedPrecondition, "GPIO device is not active")
	}

	if !device.IsOutput() {
		return nil, status.Error(codes.FailedPrecondition, "GPIO device is not configured for output")
	}

	// TODO: Implement actual GPIO write operation
	
	// Update the device value
	device.SetValue(int(req.Value))
	result = s.database.DB().Save(&device)
	if result.Error != nil {
		s.logger.WithError(result.Error).Error("Failed to update GPIO device value")
		return nil, status.Error(codes.Internal, "Failed to update GPIO device")
	}

	// Record the write operation
	reading := models.GPIOReading{
		DeviceID:  device.ID,
		Value:     float64(req.Value),
		Timestamp: time.Now(),
	}
	s.database.DB().Create(&reading)

	return &pb.WriteGPIOResponse{
		DeviceId:  device.ID,
		Pin:       int32(device.PinNumber),
		Value:     req.Value,
		Timestamp: timestamppb.New(reading.Timestamp),
	}, nil
}

// Helper functions for model conversion

func (s *PiControllerServer) clusterToProto(cluster *models.Cluster) *pb.Cluster {
	pbCluster := &pb.Cluster{
		Id:             cluster.ID,
		Name:           cluster.Name,
		Description:    cluster.Description,
		Status:         s.clusterStatusToProto(cluster.Status),
		Version:        cluster.Version,
		MasterEndpoint: cluster.MasterEndpoint,
		CreatedAt:      timestamppb.New(cluster.CreatedAt),
		UpdatedAt:      timestamppb.New(cluster.UpdatedAt),
	}

	// Convert nodes if loaded
	if cluster.Nodes != nil {
		pbCluster.Nodes = make([]*pb.Node, len(cluster.Nodes))
		for i, node := range cluster.Nodes {
			pbCluster.Nodes[i] = s.nodeToProto(&node)
		}
	}

	return pbCluster
}

func (s *PiControllerServer) nodeToProto(node *models.Node) *pb.Node {
	pbNode := &pb.Node{
		Id:            node.ID,
		Name:          node.Name,
		IpAddress:     node.IPAddress,
		MacAddress:    node.MACAddress,
		Status:        s.nodeStatusToProto(node.Status),
		Role:          s.nodeRoleToProto(node.Role),
		Architecture:  node.Architecture,
		Model:         node.Model,
		SerialNumber:  node.SerialNumber,
		CpuCores:      int32(node.CPUCores),
		Memory:        node.Memory,
		KubeVersion:   node.KubeVersion,
		NodeName:      node.NodeName,
		OsVersion:     node.OSVersion,
		KernelVersion: node.KernelVersion,
		LastSeen:      timestamppb.New(node.LastSeen),
		CreatedAt:     timestamppb.New(node.CreatedAt),
		UpdatedAt:     timestamppb.New(node.UpdatedAt),
	}

	if node.ClusterID != nil {
		pbNode.ClusterId = node.ClusterID
	}

	// Convert GPIO devices if loaded
	if node.GPIODevices != nil {
		pbNode.GpioDevices = make([]*pb.GPIODevice, len(node.GPIODevices))
		for i, device := range node.GPIODevices {
			pbNode.GpioDevices[i] = s.gpioDeviceToProto(&device)
		}
	}

	return pbNode
}

func (s *PiControllerServer) gpioDeviceToProto(device *models.GPIODevice) *pb.GPIODevice {
	return &pb.GPIODevice{
		Id:          device.ID,
		Name:        device.Name,
		Description: device.Description,
		PinNumber:   int32(device.PinNumber),
		Direction:   s.gpioDirectionToProto(device.Direction),
		PullMode:    s.gpioPullModeToProto(device.PullMode),
		Value:       int32(device.Value),
		DeviceType:  s.gpioDeviceTypeToProto(device.DeviceType),
		Status:      s.gpioStatusToProto(device.Status),
		NodeId:      device.NodeID,
		Config:      s.gpioConfigToProto(&device.Config),
		CreatedAt:   timestamppb.New(device.CreatedAt),
		UpdatedAt:   timestamppb.New(device.UpdatedAt),
	}
}

func (s *PiControllerServer) gpioConfigToProto(config *models.GPIOConfig) *pb.GPIOConfig {
	return &pb.GPIOConfig{
		Frequency:   int32(config.Frequency),
		DutyCycle:   int32(config.DutyCycle),
		SpiMode:     int32(config.SPIMode),
		SpiBits:     int32(config.SPIBits),
		SpiSpeed:    int32(config.SPISpeed),
		SpiChannel:  int32(config.SPIChannel),
		I2CAddress:  int32(config.I2CAddress),
		I2CBus:      int32(config.I2CBus),
		SampleRate:  int32(config.SampleRate),
	}
}

// Status conversion helpers
func (s *PiControllerServer) clusterStatusToProto(status models.ClusterStatus) pb.ClusterStatus {
	switch status {
	case models.ClusterStatusPending:
		return pb.ClusterStatus_CLUSTER_STATUS_PENDING
	case models.ClusterStatusProvisioning:
		return pb.ClusterStatus_CLUSTER_STATUS_PROVISIONING
	case models.ClusterStatusActive:
		return pb.ClusterStatus_CLUSTER_STATUS_ACTIVE
	case models.ClusterStatusDegraded:
		return pb.ClusterStatus_CLUSTER_STATUS_DEGRADED
	case models.ClusterStatusMaintenance:
		return pb.ClusterStatus_CLUSTER_STATUS_MAINTENANCE
	case models.ClusterStatusFailed:
		return pb.ClusterStatus_CLUSTER_STATUS_FAILED
	default:
		return pb.ClusterStatus_CLUSTER_STATUS_UNSPECIFIED
	}
}

func (s *PiControllerServer) nodeStatusToProto(status models.NodeStatus) pb.NodeStatus {
	switch status {
	case models.NodeStatusDiscovered:
		return pb.NodeStatus_NODE_STATUS_DISCOVERED
	case models.NodeStatusProvisioning:
		return pb.NodeStatus_NODE_STATUS_PROVISIONING
	case models.NodeStatusReady:
		return pb.NodeStatus_NODE_STATUS_READY
	case models.NodeStatusNotReady:
		return pb.NodeStatus_NODE_STATUS_NOT_READY
	case models.NodeStatusMaintenance:
		return pb.NodeStatus_NODE_STATUS_MAINTENANCE
	case models.NodeStatusFailed:
		return pb.NodeStatus_NODE_STATUS_FAILED
	case models.NodeStatusUnknown:
		return pb.NodeStatus_NODE_STATUS_UNKNOWN
	default:
		return pb.NodeStatus_NODE_STATUS_UNSPECIFIED
	}
}

func (s *PiControllerServer) nodeRoleToProto(role models.NodeRole) pb.NodeRole {
	switch role {
	case models.NodeRoleMaster:
		return pb.NodeRole_NODE_ROLE_MASTER
	case models.NodeRoleWorker:
		return pb.NodeRole_NODE_ROLE_WORKER
	default:
		return pb.NodeRole_NODE_ROLE_UNSPECIFIED
	}
}

func (s *PiControllerServer) nodeRoleFromProto(role pb.NodeRole) models.NodeRole {
	switch role {
	case pb.NodeRole_NODE_ROLE_MASTER:
		return models.NodeRoleMaster
	case pb.NodeRole_NODE_ROLE_WORKER:
		return models.NodeRoleWorker
	default:
		return models.NodeRoleWorker
	}
}

func (s *PiControllerServer) gpioDirectionToProto(direction models.GPIODirection) pb.GPIODirection {
	switch direction {
	case models.GPIODirectionInput:
		return pb.GPIODirection_GPIO_DIRECTION_INPUT
	case models.GPIODirectionOutput:
		return pb.GPIODirection_GPIO_DIRECTION_OUTPUT
	default:
		return pb.GPIODirection_GPIO_DIRECTION_UNSPECIFIED
	}
}

func (s *PiControllerServer) gpioPullModeToProto(pullMode models.GPIOPullMode) pb.GPIOPullMode {
	switch pullMode {
	case models.GPIOPullNone:
		return pb.GPIOPullMode_GPIO_PULL_MODE_NONE
	case models.GPIOPullUp:
		return pb.GPIOPullMode_GPIO_PULL_MODE_UP
	case models.GPIOPullDown:
		return pb.GPIOPullMode_GPIO_PULL_MODE_DOWN
	default:
		return pb.GPIOPullMode_GPIO_PULL_MODE_UNSPECIFIED
	}
}

func (s *PiControllerServer) gpioDeviceTypeToProto(deviceType models.GPIODeviceType) pb.GPIODeviceType {
	switch deviceType {
	case models.GPIODeviceTypeDigital:
		return pb.GPIODeviceType_GPIO_DEVICE_TYPE_DIGITAL
	case models.GPIODeviceTypeAnalog:
		return pb.GPIODeviceType_GPIO_DEVICE_TYPE_ANALOG
	case models.GPIODeviceTypePWM:
		return pb.GPIODeviceType_GPIO_DEVICE_TYPE_PWM
	case models.GPIODeviceTypeSPI:
		return pb.GPIODeviceType_GPIO_DEVICE_TYPE_SPI
	case models.GPIODeviceTypeI2C:
		return pb.GPIODeviceType_GPIO_DEVICE_TYPE_I2C
	default:
		return pb.GPIODeviceType_GPIO_DEVICE_TYPE_UNSPECIFIED
	}
}

func (s *PiControllerServer) gpioStatusToProto(status models.GPIOStatus) pb.GPIOStatus {
	switch status {
	case models.GPIOStatusActive:
		return pb.GPIOStatus_GPIO_STATUS_ACTIVE
	case models.GPIOStatusInactive:
		return pb.GPIOStatus_GPIO_STATUS_INACTIVE
	case models.GPIOStatusError:
		return pb.GPIOStatus_GPIO_STATUS_ERROR
	default:
		return pb.GPIOStatus_GPIO_STATUS_UNSPECIFIED
	}
}