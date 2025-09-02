package client

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// MockServer provides a mock gRPC server for testing the client
type MockServer struct {
	pb.UnimplementedPiControllerServiceServer
	
	// Server management
	server   *grpc.Server
	listener net.Listener
	address  string
	logger   logger.Interface
	
	// State tracking
	mu              sync.RWMutex
	nodes           map[uint32]*pb.Node
	gpioDevices     map[uint32]*pb.GPIODevice
	nextNodeID      uint32
	nextGPIOID      uint32
	healthCheckCount int
	
	// Behavior configuration
	shouldFailHealth      bool
	shouldFailRegistration bool
	registrationDelay     time.Duration
	
	// Statistics
	stats MockServerStats
}

// MockServerStats tracks server statistics for testing
type MockServerStats struct {
	HealthChecks      int
	NodeRegistrations int
	NodeUpdates       int
	GPIOReads         int
	GPIOWrites        int
	ConnectionCount   int
}

// MockServerConfig configures the mock server behavior
type MockServerConfig struct {
	Address               string
	ShouldFailHealth      bool
	ShouldFailRegistration bool
	RegistrationDelay     time.Duration
}

// NewMockServer creates a new mock gRPC server for testing
func NewMockServer(config MockServerConfig, logger logger.Interface) (*MockServer, error) {
	if logger == nil {
		logger = &mockLogger{}
	}
	
	if config.Address == "" {
		config.Address = "localhost:0" // Use random available port
	}
	
	server := &MockServer{
		logger:                logger,
		address:               config.Address,
		nodes:                 make(map[uint32]*pb.Node),
		gpioDevices:          make(map[uint32]*pb.GPIODevice),
		nextNodeID:           1,
		nextGPIOID:           1,
		shouldFailHealth:      config.ShouldFailHealth,
		shouldFailRegistration: config.ShouldFailRegistration,
		registrationDelay:     config.RegistrationDelay,
	}
	
	return server, nil
}

// Start starts the mock server
func (m *MockServer) Start() error {
	listener, err := net.Listen("tcp", m.address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	
	m.listener = listener
	m.address = listener.Addr().String()
	
	m.server = grpc.NewServer()
	pb.RegisterPiControllerServiceServer(m.server, m)
	
	go func() {
		m.logger.Info("Mock server starting", "address", m.address)
		if err := m.server.Serve(listener); err != nil {
			m.logger.WithError(err).Error("Mock server error")
		}
	}()
	
	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

// Stop stops the mock server
func (m *MockServer) Stop() {
	if m.server != nil {
		m.server.GracefulStop()
		m.server = nil
	}
	if m.listener != nil {
		m.listener.Close()
		m.listener = nil
	}
	m.logger.Info("Mock server stopped")
}

// GetAddress returns the server address
func (m *MockServer) GetAddress() string {
	return m.address
}

// GetStats returns server statistics
func (m *MockServer) GetStats() MockServerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// SetHealthFailure configures health check failures
func (m *MockServer) SetHealthFailure(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailHealth = shouldFail
}

// SetRegistrationFailure configures registration failures
func (m *MockServer) SetRegistrationFailure(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailRegistration = shouldFail
}

// Health implements the Health gRPC method
func (m *MockServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	m.mu.Lock()
	m.stats.HealthChecks++
	m.healthCheckCount++
	shouldFail := m.shouldFailHealth
	m.mu.Unlock()
	
	if shouldFail {
		return nil, status.Error(codes.Unavailable, "mock health check failure")
	}
	
	return &pb.HealthResponse{
		Status:    "ok",
		Timestamp: timestamppb.Now(),
		Version:   "mock-1.0.0",
		Uptime:    "1h30m",
	}, nil
}

// CreateNode implements the CreateNode gRPC method
func (m *MockServer) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stats.NodeRegistrations++
	
	if m.shouldFailRegistration {
		return nil, status.Error(codes.Internal, "mock registration failure")
	}
	
	// Add registration delay if configured
	if m.registrationDelay > 0 {
		m.mu.Unlock()
		time.Sleep(m.registrationDelay)
		m.mu.Lock()
	}
	
	// Check for existing node by MAC address
	for _, node := range m.nodes {
		if node.MacAddress == req.MacAddress {
			return nil, status.Error(codes.AlreadyExists, "node already exists")
		}
	}
	
	nodeID := m.nextNodeID
	m.nextNodeID++
	
	node := &pb.Node{
		Id:            nodeID,
		Name:          req.Name,
		IpAddress:     req.IpAddress,
		MacAddress:    req.MacAddress,
		Status:        pb.NodeStatus_NODE_STATUS_READY,
		Role:          req.Role,
		Architecture:  req.Architecture,
		Model:         req.Model,
		SerialNumber:  req.SerialNumber,
		CpuCores:      req.CpuCores,
		Memory:        req.Memory,
		LastSeen:      timestamppb.Now(),
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
	}
	
	if req.ClusterId != nil {
		node.ClusterId = req.ClusterId
	}
	
	m.nodes[nodeID] = node
	
	m.logger.Info("Mock node registered", 
		"node_id", nodeID,
		"name", req.Name)
	
	return node, nil
}

// GetNode implements the GetNode gRPC method
func (m *MockServer) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	node, exists := m.nodes[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	
	return node, nil
}

// ListNodes implements the ListNodes gRPC method
func (m *MockServer) ListNodes(ctx context.Context, req *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	m.mu.RLock()
	defer m.mu.Unlock()
	
	var nodes []*pb.Node
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	
	return &pb.ListNodesResponse{
		Nodes:      nodes,
		TotalCount: int32(len(nodes)),
	}, nil
}

// UpdateNode implements the UpdateNode gRPC method
func (m *MockServer) UpdateNode(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stats.NodeUpdates++
	
	node, exists := m.nodes[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "node not found")
	}
	
	// Update fields if provided
	if req.Name != nil {
		node.Name = *req.Name
	}
	if req.IpAddress != nil {
		node.IpAddress = *req.IpAddress
	}
	if req.MacAddress != nil {
		node.MacAddress = *req.MacAddress
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
	if req.CpuCores != nil {
		node.CpuCores = *req.CpuCores
	}
	if req.Memory != nil {
		node.Memory = *req.Memory
	}
	if req.ClusterId != nil {
		node.ClusterId = req.ClusterId
	}
	if req.KubeVersion != nil {
		node.KubeVersion = *req.KubeVersion
	}
	if req.NodeName != nil {
		node.NodeName = *req.NodeName
	}
	if req.OsVersion != nil {
		node.OsVersion = *req.OsVersion
	}
	if req.KernelVersion != nil {
		node.KernelVersion = *req.KernelVersion
	}
	
	node.UpdatedAt = timestamppb.Now()
	node.LastSeen = timestamppb.Now()
	
	return node, nil
}

// ReadGPIO implements the ReadGPIO gRPC method
func (m *MockServer) ReadGPIO(ctx context.Context, req *pb.ReadGPIORequest) (*pb.ReadGPIOResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stats.GPIOReads++
	
	// For mock, return a simulated reading
	return &pb.ReadGPIOResponse{
		DeviceId:  req.Id,
		Pin:       18, // Mock pin number
		Value:     1.0, // Mock value
		Timestamp: timestamppb.Now(),
	}, nil
}

// WriteGPIO implements the WriteGPIO gRPC method
func (m *MockServer) WriteGPIO(ctx context.Context, req *pb.WriteGPIORequest) (*pb.WriteGPIOResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stats.GPIOWrites++
	
	return &pb.WriteGPIOResponse{
		DeviceId:  req.Id,
		Pin:       18, // Mock pin number
		Value:     req.Value,
		Timestamp: timestamppb.Now(),
	}, nil
}

// GetSystemInfo implements the GetSystemInfo gRPC method
func (m *MockServer) GetSystemInfo(ctx context.Context, req *pb.SystemInfoRequest) (*pb.SystemInfoResponse, error) {
	return &pb.SystemInfoResponse{
		GoVersion:   "go1.21.0",
		GoOs:        "linux",
		GoArch:      "arm64",
		CpuCount:    4,
		Goroutines:  10,
		Memory:      &pb.MemoryInfo{},
		Gc:          &pb.GCInfo{},
		Timestamp:   timestamppb.Now(),
		Uptime:      "1h30m",
	}, nil
}

// Unimplemented methods return NotImplemented errors
func (m *MockServer) CreateCluster(ctx context.Context, req *pb.CreateClusterRequest) (*pb.Cluster, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) GetCluster(ctx context.Context, req *pb.GetClusterRequest) (*pb.Cluster, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) ListClusters(ctx context.Context, req *pb.ListClustersRequest) (*pb.ListClustersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) UpdateCluster(ctx context.Context, req *pb.UpdateClusterRequest) (*pb.Cluster, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) DeleteCluster(ctx context.Context, req *pb.DeleteClusterRequest) (*pb.DeleteClusterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) DeleteNode(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.DeleteNodeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) ProvisionNode(ctx context.Context, req *pb.ProvisionNodeRequest) (*pb.ProvisionNodeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) DeprovisionNode(ctx context.Context, req *pb.DeprovisionNodeRequest) (*pb.DeprovisionNodeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) CreateGPIODevice(ctx context.Context, req *pb.CreateGPIODeviceRequest) (*pb.GPIODevice, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) GetGPIODevice(ctx context.Context, req *pb.GetGPIODeviceRequest) (*pb.GPIODevice, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) ListGPIODevices(ctx context.Context, req *pb.ListGPIODevicesRequest) (*pb.ListGPIODevicesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) UpdateGPIODevice(ctx context.Context, req *pb.UpdateGPIODeviceRequest) (*pb.GPIODevice, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) DeleteGPIODevice(ctx context.Context, req *pb.DeleteGPIODeviceRequest) (*pb.DeleteGPIODeviceResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in mock")
}

func (m *MockServer) StreamGPIOReadings(req *pb.StreamGPIOReadingsRequest, stream grpc.ServerStreamingServer[pb.GPIOReading]) error {
	return status.Error(codes.Unimplemented, "not implemented in mock")
}

// mockLogger is a simple logger implementation for testing
type mockLogger struct{}

func (l *mockLogger) Debug(msg string, args ...interface{}) {}
func (l *mockLogger) Info(msg string, args ...interface{}) {}
func (l *mockLogger) Warn(msg string, args ...interface{}) {}
func (l *mockLogger) Error(msg string, args ...interface{}) {}
func (l *mockLogger) Debugf(format string, args ...interface{}) {}
func (l *mockLogger) Infof(format string, args ...interface{}) {}
func (l *mockLogger) Warnf(format string, args ...interface{}) {}
func (l *mockLogger) Errorf(format string, args ...interface{}) {}
func (l *mockLogger) Fatalf(format string, args ...interface{}) {}
func (l *mockLogger) WithField(key string, value interface{}) logger.Interface { return l }
func (l *mockLogger) WithFields(fields map[string]interface{}) logger.Interface { return l }
func (l *mockLogger) WithError(err error) logger.Interface { return l }