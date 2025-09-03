package server

import (
	"context"
	"net"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/storage"
	pb "github.com/dsyorkd/pi-controller/proto"
)

// Server represents the gRPC server
type Server struct {
	config   *config.GRPCConfig
	logger   logger.Interface
	database *storage.Database
	server   *grpc.Server
}

// New creates a new gRPC server instance
func New(cfg *config.GRPCConfig, logger logger.Interface, db *storage.Database) (*Server, error) {
	var opts []grpc.ServerOption

	// Add TLS credentials if configured
	if cfg.IsTLSEnabled() {
		creds, err := credentials.NewServerTLSFromFile(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Add logging interceptor
	opts = append(opts, grpc.UnaryInterceptor(loggingInterceptor(logger)))
	opts = append(opts, grpc.StreamInterceptor(streamLoggingInterceptor(logger)))

	grpcServer := grpc.NewServer(opts...)

	s := &Server{
		config:   cfg,
		logger:   logger,
		database: db,
		server:   grpcServer,
	}

	// Register service implementation
	piControllerServer := &PiControllerServer{
		database: db,
		logger:   logger,
	}
	pb.RegisterPiControllerServiceServer(grpcServer, piControllerServer)

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.GetAddress())
	if err != nil {
		return err
	}

	s.logger.WithField("address", s.config.GetAddress()).Info("Starting gRPC server")
	return s.server.Serve(listener)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("Shutting down gRPC server")
	s.server.GracefulStop()
}

// loggingInterceptor provides request logging for unary RPCs
func loggingInterceptor(logger logger.Interface) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.WithField("method", info.FullMethod).Debug("gRPC request started")

		resp, err := handler(ctx, req)

		if err != nil {
			logger.WithFields(map[string]interface{}{
				"method": info.FullMethod,
				"error":  err,
			}).Error("gRPC request failed")
		} else {
			logger.WithField("method", info.FullMethod).Debug("gRPC request completed")
		}

		return resp, err
	}
}

// streamLoggingInterceptor provides request logging for streaming RPCs
func streamLoggingInterceptor(logger logger.Interface) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		logger.WithField("method", info.FullMethod).Debug("gRPC stream started")

		err := handler(srv, ss)

		if err != nil {
			logger.WithFields(map[string]interface{}{
				"method": info.FullMethod,
				"error":  err,
			}).Error("gRPC stream failed")
		} else {
			logger.WithField("method", info.FullMethod).Debug("gRPC stream completed")
		}

		return err
	}
}
