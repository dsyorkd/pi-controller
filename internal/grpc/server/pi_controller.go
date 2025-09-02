package server

import (
	"context"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// PiControllerServer implements the gRPC PiControllerService
type PiControllerServer struct {
	pb.UnimplementedPiControllerServiceServer
	database    *storage.Database
	logger      logger.Interface
	authManager *middleware.AuthManager
}

// NewPiControllerServer creates a new gRPC server instance
func NewPiControllerServer(database *storage.Database, logger logger.Interface, authManager *middleware.AuthManager) *PiControllerServer {
	return &PiControllerServer{
		database:    database,
		logger:      logger.WithField("component", "grpc-server"),
		authManager: authManager,
	}
}

// Health returns the health status of the service
func (s *PiControllerServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:    "ok",
		Timestamp: timestamppb.Now(),
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
		Architecture: req.Architecture,
		Model:        req.Model,
		SerialNumber: req.SerialNumber,
		CPUCores:     int(req.CpuCores),
		Memory:       req.Memory,
		Status:       models.NodeStatusDiscovered,
		Role:         models.NodeRoleWorker,
	}

	if req.ClusterId != nil {
		clusterID := uint(*req.ClusterId)
		node.ClusterID = &clusterID
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
	result := s.database.DB().First(&node, req.Id)
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
	// Validate authentication - GPIO read operations require at least viewer role
	claims, err := s.validateAuthentication(ctx)
	if err != nil {
		return nil, err
	}
	
	// Require at least viewer role for GPIO read operations
	if err := s.requireRole(claims, middleware.RoleViewer); err != nil {
		return nil, err
	}

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

	// Audit log the GPIO read operation
	s.logger.WithFields(map[string]interface{}{
		"event_type": "gpio_read",
		"user_id":    claims.UserID,
		"device_id":  device.ID,
		"pin":        device.PinNumber,
		"value":      device.Value,
	}).Info("GPIO read operation performed")

	return &pb.ReadGPIOResponse{
		DeviceId:  uint32(device.ID),
		Pin:       int32(device.PinNumber),
		Value:     float64(device.Value),
		Timestamp: timestamppb.New(reading.Timestamp),
	}, nil
}

// WriteGPIO writes a value to a GPIO device
func (s *PiControllerServer) WriteGPIO(ctx context.Context, req *pb.WriteGPIORequest) (*pb.WriteGPIOResponse, error) {
	// Validate authentication - GPIO write operations require at least operator role
	claims, err := s.validateAuthentication(ctx)
	if err != nil {
		return nil, err
	}
	
	// Require at least operator role for GPIO write operations (more privileged than read)
	if err := s.requireRole(claims, middleware.RoleOperator); err != nil {
		return nil, err
	}

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

	// Audit log the GPIO write operation
	s.logger.WithFields(map[string]interface{}{
		"event_type": "gpio_write",
		"user_id":    claims.UserID,
		"device_id":  device.ID,
		"pin":        device.PinNumber,
		"value":      req.Value,
	}).Info("GPIO write operation performed")

	return &pb.WriteGPIOResponse{
		DeviceId:  uint32(device.ID),
		Pin:       int32(device.PinNumber),
		Value:     req.Value,
		Timestamp: timestamppb.New(reading.Timestamp),
	}, nil
}

// Helper functions for model conversion

func (s *PiControllerServer) clusterToProto(cluster *models.Cluster) *pb.Cluster {
	pbCluster := &pb.Cluster{
		Id:             uint32(cluster.ID),
		Name:           cluster.Name,
		Description:    cluster.Description,
		Version:        cluster.Version,
		MasterEndpoint: cluster.MasterEndpoint,
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
		Id:         uint32(node.ID),
		Name:       node.Name,
		IpAddress:  node.IPAddress,
		MacAddress: node.MACAddress,
	}

	return pbNode
}

// validateAuthentication validates the authentication token from gRPC metadata
func (s *PiControllerServer) validateAuthentication(ctx context.Context) (*middleware.JWTClaims, error) {
	// Skip authentication if auth manager is not configured
	if s.authManager == nil {
		s.logger.Warn("Authentication manager not configured for gRPC server")
		return nil, status.Error(codes.Unauthenticated, "Authentication not configured")
	}

	// Extract metadata from context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Missing metadata")
	}

	// Get authorization header
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Missing authorization header")
	}

	authHeader := authHeaders[0]
	if authHeader == "" {
		return nil, status.Error(codes.Unauthenticated, "Empty authorization header")
	}

	// Validate Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil, status.Error(codes.Unauthenticated, "Invalid authorization header format")
	}

	// Extract token
	tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
	if tokenString == "" {
		return nil, status.Error(codes.Unauthenticated, "Empty token")
	}

	// Validate token
	claims, err := s.authManager.ValidateToken(tokenString)
	if err != nil {
		s.logger.WithError(err).Warn("Token validation failed in gRPC server")
		return nil, status.Error(codes.Unauthenticated, "Invalid or expired token")
	}

	return claims, nil
}

// requireRole checks if the authenticated user has the required role
func (s *PiControllerServer) requireRole(claims *middleware.JWTClaims, requiredRole string) error {
	if claims == nil {
		return status.Error(codes.Unauthenticated, "Authentication required")
	}

	// Admin role can access everything
	if claims.Role == middleware.RoleAdmin {
		return nil
	}

	// Operator role can access operator and viewer endpoints
	if claims.Role == middleware.RoleOperator && (requiredRole == middleware.RoleOperator || requiredRole == middleware.RoleViewer) {
		return nil
	}

	// Viewer role can only access viewer endpoints
	if claims.Role == middleware.RoleViewer && requiredRole == middleware.RoleViewer {
		return nil
	}

	return status.Error(codes.PermissionDenied, "Insufficient permissions")
}