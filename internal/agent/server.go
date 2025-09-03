package agent

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// Server represents the Pi Agent gRPC server
type Server struct {
	address      string
	logger       logger.Interface
	server       *grpc.Server
	agentService *AgentService
}

// Config contains server configuration
type Config struct {
	Address string `yaml:"address" mapstructure:"address"`
	Port    int    `yaml:"port" mapstructure:"port"`
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Address: "0.0.0.0",
		Port:    9091,
	}
}

// GetAddress returns the formatted address string
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

// NewServer creates a new Pi Agent gRPC server
func NewServer(config *Config, logger logger.Interface) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create agent service (includes GPIO and metrics)
	agentService, err := NewAgentService(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent service: %w", err)
	}

	// Create gRPC server with logging interceptors
	grpcServer := grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
		grpc.StreamInterceptor(streamLoggingInterceptor(logger)),
	)

	// Register the agent service
	pb.RegisterPiAgentServiceServer(grpcServer, agentService)

	server := &Server{
		address:      config.GetAddress(),
		logger:       logger.WithField("component", "agent-server"),
		server:       grpcServer,
		agentService: agentService,
	}

	return server, nil
}

// Initialize initializes the server and its services
func (s *Server) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing Pi Agent server")

	// Initialize agent service
	if err := s.agentService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize agent service: %w", err)
	}

	s.logger.Info("Pi Agent server initialized successfully")
	return nil
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	s.logger.WithField("address", s.address).Info("Starting Pi Agent gRPC server")

	// Start server in a goroutine so it doesn't block
	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.logger.WithError(err).Error("gRPC server error")
		}
	}()

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.logger.Info("Shutting down Pi Agent server")

	// Stop gRPC server
	s.server.GracefulStop()

	// Close agent service
	if err := s.agentService.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close agent service")
		return err
	}

	s.logger.Info("Pi Agent server stopped successfully")
	return nil
}

// IsReady returns true if the server is ready to accept requests
func (s *Server) IsReady() bool {
	// Check if agent service is available
	return s.agentService != nil && s.agentService.IsReady()
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	return s.address
}

// loggingInterceptor provides request logging for unary RPCs
func loggingInterceptor(logger logger.Interface) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.WithField("method", info.FullMethod).Debug("Agent gRPC request started")

		resp, err := handler(ctx, req)

		if err != nil {
			logger.WithFields(map[string]interface{}{
				"method": info.FullMethod,
				"error":  err,
			}).Error("Agent gRPC request failed")
		} else {
			logger.WithField("method", info.FullMethod).Debug("Agent gRPC request completed")
		}

		return resp, err
	}
}

// streamLoggingInterceptor provides request logging for streaming RPCs
func streamLoggingInterceptor(logger logger.Interface) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		logger.WithField("method", info.FullMethod).Debug("Agent gRPC stream started")

		err := handler(srv, ss)

		if err != nil {
			logger.WithFields(map[string]interface{}{
				"method": info.FullMethod,
				"error":  err,
			}).Error("Agent gRPC stream failed")
		} else {
			logger.WithField("method", info.FullMethod).Debug("Agent gRPC stream completed")
		}

		return err
	}
}
