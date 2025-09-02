package agent

import (
	"context"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// AgentService implements the complete PiAgent gRPC service
// It combines GPIO and metrics functionality
type AgentService struct {
	pb.UnimplementedPiAgentServiceServer
	gpio    *GPIOService
	metrics *MetricsService
	logger  logger.Interface
}

// NewAgentService creates a new complete agent service
func NewAgentService(logger logger.Interface) (*AgentService, error) {
	gpio, err := NewGPIOService(logger)
	if err != nil {
		return nil, err
	}

	metrics := NewMetricsService(logger)

	return &AgentService{
		gpio:    gpio,
		metrics: metrics,
		logger:  logger.WithField("component", "agent-service"),
	}, nil
}

// Initialize initializes all services
func (s *AgentService) Initialize(ctx context.Context) error {
	return s.gpio.Initialize(ctx)
}

// Close shuts down all services
func (s *AgentService) Close() error {
	return s.gpio.Close()
}

// IsReady returns true if the service is ready to accept requests
func (s *AgentService) IsReady() bool {
	return s.gpio.controller.IsAvailable()
}

// GPIO-related methods - delegate to GPIO service

func (s *AgentService) ConfigureGPIOPin(ctx context.Context, req *pb.ConfigureGPIOPinRequest) (*pb.ConfigureGPIOPinResponse, error) {
	return s.gpio.ConfigureGPIOPin(ctx, req)
}

func (s *AgentService) ReadGPIOPin(ctx context.Context, req *pb.ReadGPIOPinRequest) (*pb.ReadGPIOPinResponse, error) {
	return s.gpio.ReadGPIOPin(ctx, req)
}

func (s *AgentService) WriteGPIOPin(ctx context.Context, req *pb.WriteGPIOPinRequest) (*pb.WriteGPIOPinResponse, error) {
	return s.gpio.WriteGPIOPin(ctx, req)
}

func (s *AgentService) SetGPIOPWM(ctx context.Context, req *pb.SetGPIOPWMRequest) (*pb.SetGPIOPWMResponse, error) {
	return s.gpio.SetGPIOPWM(ctx, req)
}

func (s *AgentService) ListConfiguredPins(ctx context.Context, req *pb.ListConfiguredPinsRequest) (*pb.ListConfiguredPinsResponse, error) {
	return s.gpio.ListConfiguredPins(ctx, req)
}

func (s *AgentService) AgentHealth(ctx context.Context, req *pb.AgentHealthRequest) (*pb.AgentHealthResponse, error) {
	return s.gpio.AgentHealth(ctx, req)
}

func (s *AgentService) GetSystemInfo(ctx context.Context, req *pb.GetSystemInfoRequest) (*pb.GetSystemInfoResponse, error) {
	return s.gpio.GetSystemInfo(ctx, req)
}

// Metrics-related methods - delegate to metrics service

func (s *AgentService) GetSystemMetrics(ctx context.Context, req *pb.GetSystemMetricsRequest) (*pb.GetSystemMetricsResponse, error) {
	return s.metrics.GetSystemMetrics(ctx, req)
}

func (s *AgentService) StreamSystemMetrics(req *pb.StreamSystemMetricsRequest, stream pb.PiAgentService_StreamSystemMetricsServer) error {
	return s.metrics.StreamSystemMetrics(req, stream)
}