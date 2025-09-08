package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
	"github.com/dsyorkd/pi-controller/internal/storage"
	gpiov1 "github.com/dsyorkd/pi-controller/pkg/apis/gpio/v1"
)

// ControllerManagerConfig holds configuration for the controller manager
type ControllerManagerConfig struct {
	MetricsAddr      string
	HealthAddr       string
	LeaderElection   bool
	LeaderElectionID string
	LeaderElectionNS string
	ReconcileTimeout time.Duration
	LogLevel         string
}

// DefaultControllerManagerConfig returns default configuration
func DefaultControllerManagerConfig() *ControllerManagerConfig {
	return &ControllerManagerConfig{
		MetricsAddr:      ":8080",
		HealthAddr:       ":8081",
		LeaderElection:   true,
		LeaderElectionID: "pi-controller-leader-election",
		LeaderElectionNS: "kube-system",
		ReconcileTimeout: 60 * time.Second,
		LogLevel:         "info",
	}
}

// ControllerManager manages all Kubernetes controllers
type ControllerManager struct {
	config      *ControllerManagerConfig
	logger      *logrus.Entry
	db          *storage.Database
	gpioService *services.GPIOService
	pwmService  *services.PWMService
	i2cService  *services.I2CService
	nodeService *services.NodeServiceAdapter
}

// NewControllerManager creates a new controller manager
func NewControllerManager(config *ControllerManagerConfig, logrusLogger *logrus.Logger, db *storage.Database) (*ControllerManager, error) {
	if config == nil {
		config = DefaultControllerManagerConfig()
	}

	// Create a logger instance that implements the logger.Interface
	internalLogger := logger.NewLogrusAdapter(logrusLogger)

	// Create services
	gpioService := services.NewGPIOService(db, internalLogger)
	pwmService := services.NewPWMService(db, internalLogger)
	i2cService := services.NewI2CService(db, internalLogger)
	nodeService := services.NewNodeService(db, internalLogger)
	nodeServiceAdapter := services.NewNodeServiceAdapter(nodeService)

	return &ControllerManager{
		config:      config,
		logger:      logrusLogger.WithField("component", "controller-manager"),
		db:          db,
		gpioService: gpioService,
		pwmService:  pwmService,
		i2cService:  i2cService,
		nodeService: nodeServiceAdapter,
	}, nil
}

// Start starts the controller manager
func (cm *ControllerManager) Start(ctx context.Context) error {
	cm.logger.Info("Starting Kubernetes controller manager")

	// Create runtime scheme
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gpiov1.AddToScheme(scheme))

	// Create controller manager configuration
	opts := ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      cm.config.MetricsAddr,
		HealthProbeBindAddress:  cm.config.HealthAddr,
		LeaderElection:          cm.config.LeaderElection,
		LeaderElectionID:        cm.config.LeaderElectionID,
		LeaderElectionNamespace: cm.config.LeaderElectionNS,
	}

	// Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		cm.logger.WithError(err).Error("Failed to create controller manager")
		return fmt.Errorf("failed to create controller manager: %w", err)
	}

	// Setup controllers
	if err := cm.setupControllers(mgr); err != nil {
		cm.logger.WithError(err).Error("Failed to setup controllers")
		return fmt.Errorf("failed to setup controllers: %w", err)
	}

	// Setup health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		cm.logger.WithError(err).Error("Failed to setup health check")
		return fmt.Errorf("failed to setup health check: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		cm.logger.WithError(err).Error("Failed to setup readiness check")
		return fmt.Errorf("failed to setup readiness check: %w", err)
	}

	cm.logger.WithFields(logrus.Fields{
		"metrics_addr":    cm.config.MetricsAddr,
		"health_addr":     cm.config.HealthAddr,
		"leader_election": cm.config.LeaderElection,
	}).Info("Controller manager configured successfully")

	// Start manager
	cm.logger.Info("Starting controller manager...")
	if err := mgr.Start(ctx); err != nil {
		cm.logger.WithError(err).Error("Controller manager failed")
		return fmt.Errorf("controller manager failed: %w", err)
	}

	cm.logger.Info("Controller manager stopped")
	return nil
}

// setupControllers sets up all the controllers with the manager
func (cm *ControllerManager) setupControllers(mgr ctrl.Manager) error {
	cm.logger.Info("Setting up controllers")

	// Create controller logger
	controllerLogger := cm.logger.WithField("component", "controller")

	// Setup GPIOPin controller
	gpioPinReconciler := &GPIOPinReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Logger:      controllerLogger.WithField("controller", "gpiopin"),
		GPIOService: cm.gpioService,
		NodeService: cm.nodeService,
	}

	if err := gpioPinReconciler.SetupWithManager(mgr); err != nil {
		cm.logger.WithError(err).Error("Failed to setup GPIOPin controller")
		return fmt.Errorf("failed to setup GPIOPin controller: %w", err)
	}

	// Setup PWMController controller
	pwmControllerReconciler := &PWMControllerReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Logger:      controllerLogger.WithField("controller", "pwmcontroller"),
		PWMService:  cm.pwmService,
		NodeService: cm.nodeService,
	}

	if err := pwmControllerReconciler.SetupWithManager(mgr); err != nil {
		cm.logger.WithError(err).Error("Failed to setup PWMController controller")
		return fmt.Errorf("failed to setup PWMController controller: %w", err)
	}

	// Setup I2CDevice controller
	i2cDeviceReconciler := &I2CDeviceReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Logger:      controllerLogger.WithField("controller", "i2cdevice"),
		I2CService:  cm.i2cService,
		NodeService: cm.nodeService,
	}

	if err := i2cDeviceReconciler.SetupWithManager(mgr); err != nil {
		cm.logger.WithError(err).Error("Failed to setup I2CDevice controller")
		return fmt.Errorf("failed to setup I2CDevice controller: %w", err)
	}

	cm.logger.Info("All controllers setup successfully")
	return nil
}

// Stop gracefully stops the controller manager
func (cm *ControllerManager) Stop() error {
	cm.logger.Info("Shutting down controller manager")

	// Close services
	if err := cm.gpioService.Close(); err != nil {
		cm.logger.WithError(err).Error("Failed to close GPIO service")
	}

	cm.logger.Info("Controller manager shutdown complete")
	return nil
}
