package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/api/handlers"
	"github.com/dsyorkd/pi-controller/internal/api/middleware"
	"github.com/dsyorkd/pi-controller/internal/config"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/sirupsen/logrus"
)

// Server represents the REST API server
type Server struct {
	config         *config.APIConfig
	logger         logger.Interface
	database       *storage.Database
	clusterService *services.ClusterService
	nodeService    *services.NodeService
	gpioService    *services.GPIOService
	authManager    *middleware.AuthManager
	validator      *middleware.Validator
	rateLimiter    *middleware.RateLimiter
	router         *gin.Engine
	server         *http.Server
}

// New creates a new API server instance
func New(cfg *config.APIConfig, log logger.Interface, db *storage.Database) *Server {
	// Set Gin mode based on environment  
	gin.SetMode(gin.ReleaseMode) // Default to release mode for structured logging

	router := gin.New()

	// Initialize services
	clusterService := services.NewClusterService(db, log)
	nodeService := services.NewNodeService(db, log)
	gpioService := services.NewGPIOService(db, log)

	// Initialize authentication manager if auth is enabled
	var authManager *middleware.AuthManager
	if cfg.AuthEnabled {
		authConfig := middleware.DefaultAuthConfig()
		var err error
		authManager, err = middleware.NewAuthManager(authConfig, log)
		if err != nil {
			log.WithError(err).Fatalf("Failed to initialize authentication manager")
		}
	}

	// Initialize validator for input validation
	validator := middleware.NewValidator(middleware.DefaultValidationConfig(), log)

	// Initialize rate limiter with default secure configuration
	logrusLogger := logrus.New()
	rateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimitConfig(), logrusLogger)

	s := &Server{
		config:         cfg,
		logger:         log,
		database:       db,
		clusterService: clusterService,
		nodeService:    nodeService,
		gpioService:    gpioService,
		authManager:    authManager,
		validator:      validator,
		rateLimiter:    rateLimiter,
		router:         router,
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
	s.router.Use(s.validator.ValidateRequest()) // Add input validation
	s.router.Use(s.rateLimiter.RateLimit())     // Add rate limiting
	
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
		if s.config.AuthEnabled && s.authManager != nil {
			v1.Use(s.authManager.Auth())
		}

		// Cluster management
		clusterHandler := handlers.NewClusterHandler(s.clusterService, s.logger)
		clusters := v1.Group("/clusters")
		{
			// Read operations - require viewer role
			clusters.GET("", s.requireRole("viewer"), clusterHandler.List)
			clusters.GET("/:id", s.requireRole("viewer"), clusterHandler.Get)
			clusters.GET("/:id/nodes", s.requireRole("viewer"), clusterHandler.ListNodes)
			clusters.GET("/:id/status", s.requireRole("viewer"), clusterHandler.Status)
			
			// Write operations - require operator role
			clusters.POST("", s.requireRole("operator"), clusterHandler.Create)
			clusters.PUT("/:id", s.requireRole("operator"), clusterHandler.Update)
			
			// Delete operations - require admin role
			clusters.DELETE("/:id", s.requireRole("admin"), clusterHandler.Delete)
		}

		// Node management
		nodeHandler := handlers.NewNodeHandler(s.nodeService, s.logger)
		nodes := v1.Group("/nodes")
		{
			// Read operations - require viewer role
			nodes.GET("", s.requireRole("viewer"), nodeHandler.List)
			nodes.GET("/:id", s.requireRole("viewer"), nodeHandler.Get)
			nodes.GET("/:id/gpio", s.requireRole("viewer"), nodeHandler.ListGPIO)
			
			// Write operations - require operator role
			nodes.POST("", s.requireRole("operator"), nodeHandler.Create)
			nodes.PUT("/:id", s.requireRole("operator"), nodeHandler.Update)
			nodes.POST("/:id/provision", s.requireRole("operator"), nodeHandler.Provision)
			nodes.POST("/:id/deprovision", s.requireRole("operator"), nodeHandler.Deprovision)
			
			// Delete operations - require admin role
			nodes.DELETE("/:id", s.requireRole("admin"), nodeHandler.Delete)
		}

		// GPIO management
		gpioHandler := handlers.NewGPIOHandler(s.gpioService, s.logger)
		gpio := v1.Group("/gpio")
		{
			// Read operations - require viewer role
			gpio.GET("", s.requireRole("viewer"), gpioHandler.List)
			gpio.GET("/:id", s.requireRole("viewer"), gpioHandler.Get)
			gpio.GET("/:id/readings", s.requireRole("viewer"), gpioHandler.GetReadings)
			gpio.POST("/:id/read", s.requireRole("viewer"), gpioHandler.Read)
			
			// Write operations - require operator role (GPIO control is sensitive)
			gpio.POST("", s.requireRole("operator"), gpioHandler.Create)
			gpio.PUT("/:id", s.requireRole("operator"), gpioHandler.Update)
			gpio.POST("/:id/write", s.requireRole("operator"), gpioHandler.Write)
			
			// Delete operations - require admin role
			gpio.DELETE("/:id", s.requireRole("admin"), gpioHandler.Delete)
		}

		// System information - require viewer role
		system := v1.Group("/system")
		{
			system.GET("/info", s.requireRole("viewer"), handlers.SystemInfo)
			system.GET("/metrics", s.requireRole("viewer"), handlers.SystemMetrics)
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

// requireRole creates a middleware that requires a specific role, only if auth is enabled
func (s *Server) requireRole(role string) gin.HandlerFunc {
	// Return a no-op middleware if auth is disabled or authManager is nil
	if !s.config.AuthEnabled || s.authManager == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	
	return s.authManager.RequireRole(role)
}