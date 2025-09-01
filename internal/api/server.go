package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/spenceryork/pi-controller/internal/api/handlers"
	"github.com/spenceryork/pi-controller/internal/api/middleware"
	"github.com/spenceryork/pi-controller/internal/config"
	"github.com/spenceryork/pi-controller/internal/storage"
)

// Server represents the REST API server
type Server struct {
	config   *config.APIConfig
	logger   *logrus.Logger
	database *storage.Database
	router   *gin.Engine
	server   *http.Server
}

// New creates a new API server instance
func New(cfg *config.APIConfig, logger *logrus.Logger, db *storage.Database) *Server {
	// Set Gin mode based on environment
	if logger.Level == logrus.DebugLevel {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		config:   cfg,
		logger:   logger,
		database: db,
		router:   router,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes and middleware
func (s *Server) setupRoutes() {
	// Global middleware
	s.router.Use(middleware.Logger(s.logger))
	s.router.Use(middleware.Recovery(s.logger))
	s.router.Use(middleware.RequestID())
	
	if s.config.CORSEnabled {
		s.router.Use(middleware.CORS())
	}

	// Health check endpoints (no auth required)
	s.router.GET("/health", handlers.NewHealthHandler(s.database).Health)
	s.router.GET("/ready", handlers.NewHealthHandler(s.database).Ready)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Authentication middleware for protected routes
		if s.config.AuthEnabled {
			v1.Use(middleware.Auth())
		}

		// Cluster management
		clusterHandler := handlers.NewClusterHandler(s.database, s.logger)
		clusters := v1.Group("/clusters")
		{
			clusters.GET("", clusterHandler.List)
			clusters.POST("", clusterHandler.Create)
			clusters.GET("/:id", clusterHandler.Get)
			clusters.PUT("/:id", clusterHandler.Update)
			clusters.DELETE("/:id", clusterHandler.Delete)
			clusters.GET("/:id/nodes", clusterHandler.ListNodes)
			clusters.GET("/:id/status", clusterHandler.Status)
		}

		// Node management
		nodeHandler := handlers.NewNodeHandler(s.database, s.logger)
		nodes := v1.Group("/nodes")
		{
			nodes.GET("", nodeHandler.List)
			nodes.POST("", nodeHandler.Create)
			nodes.GET("/:id", nodeHandler.Get)
			nodes.PUT("/:id", nodeHandler.Update)
			nodes.DELETE("/:id", nodeHandler.Delete)
			nodes.GET("/:id/gpio", nodeHandler.ListGPIO)
			nodes.POST("/:id/provision", nodeHandler.Provision)
			nodes.POST("/:id/deprovision", nodeHandler.Deprovision)
		}

		// GPIO management
		gpioHandler := handlers.NewGPIOHandler(s.database, s.logger)
		gpio := v1.Group("/gpio")
		{
			gpio.GET("", gpioHandler.List)
			gpio.POST("", gpioHandler.Create)
			gpio.GET("/:id", gpioHandler.Get)
			gpio.PUT("/:id", gpioHandler.Update)
			gpio.DELETE("/:id", gpioHandler.Delete)
			gpio.POST("/:id/read", gpioHandler.Read)
			gpio.POST("/:id/write", gpioHandler.Write)
			gpio.GET("/:id/readings", gpioHandler.GetReadings)
		}

		// System information
		system := v1.Group("/system")
		{
			system.GET("/info", handlers.SystemInfo)
			system.GET("/metrics", handlers.SystemMetrics)
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	readTimeout, err := time.ParseDuration(s.config.ReadTimeout)
	if err != nil {
		readTimeout = 30 * time.Second
	}

	writeTimeout, err := time.ParseDuration(s.config.WriteTimeout)
	if err != nil {
		writeTimeout = 30 * time.Second
	}

	s.server = &http.Server{
		Addr:         s.config.GetAddress(),
		Handler:      s.router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.WithField("address", s.config.GetAddress()).Info("Starting API server")

	if s.config.IsTLSEnabled() {
		return s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
	}
	
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down API server")
	return s.server.Shutdown(ctx)
}

// Router returns the underlying Gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}