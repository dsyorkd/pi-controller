package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	pb "github.com/dsyorkd/pi-controller/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Server represents the Pi Agent gRPC server
type Server struct {
	address      string
	logger       logger.Interface
	server       *grpc.Server
	agentService *AgentService
	config       *Config // Store config to access metrics interval
	nodeID       string
}

// Config contains server configuration
type Config struct {
	Address         string `yaml:"address" mapstructure:"address"`
	Port            int    `yaml:"port" mapstructure:"port"`
	MetricsInterval int    `yaml:"metrics_interval" mapstructure:"metrics_interval"`
	TLSCertFile     string `yaml:"tls_cert_file" mapstructure:"tls_cert_file"`
	TLSKeyFile      string `yaml:"tls_key_file" mapstructure:"tls_key_file"`
	TLSCAFile       string `yaml:"tls_ca_file" mapstructure:"tls_ca_file"`
}

// IsTLSEnabled returns true if TLS is configured
func (c *Config) IsTLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != "" && c.TLSCAFile != ""
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Address:         "0.0.0.0",
		Port:            9091,
		MetricsInterval: 5, // Default to 5 seconds
	}
}

// GetAddress returns the formatted address string
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

// NewServer creates a new Pi Agent gRPC server
func NewServer(config *Config, logger logger.Interface, controllerClient pb.PiControllerServiceClient, nodeID string) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create agent service (includes GPIO and metrics)
	agentService, err := NewAgentService(logger, controllerClient, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent service: %w", err)
	}

	// Create gRPC server options
	var opts []grpc.ServerOption

	// Configure TLS if enabled
	if config.IsTLSEnabled() {
		// Load server certificate and private key
		serverCert, err := tls.LoadX509KeyPair(config.TLSCertFile, config.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}

		// Load CA certificate for client authentication
		caCert, err := os.ReadFile(config.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate to pool")
		}

		// Configure TLS with client authentication
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
		}

		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		logger.Info("Pi Agent gRPC server configured with mTLS")
	} else {
		opts = append(opts, grpc.Creds(insecure.NewCredentials()))
		logger.Warn("Pi Agent gRPC server configured with INSECURE credentials")
	}

	opts = append(opts, grpc.UnaryInterceptor(loggingInterceptor(logger)))
	opts = append(opts, grpc.StreamInterceptor(streamLoggingInterceptor(logger)))

	// Create gRPC server with logging interceptors
	grpcServer := grpc.NewServer(opts...)

	// Register the agent service
	pb.RegisterPiAgentServiceServer(grpcServer, agentService)

	server := &Server{
		address:      config.GetAddress(),
		logger:       logger.WithField("component", "agent-server"),
		server:       grpcServer,
		agentService: agentService,
		config:       config,
		nodeID:       nodeID,
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

	// Start streaming metrics to controller in background
	s.agentService.StartMetricsStreaming(ctx, time.Duration(s.config.MetricsInterval)*time.Second)

	// Start server - this blocks until stopped
	return s.server.Serve(listener)
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
