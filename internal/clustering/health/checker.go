package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
)

// HealthChecker monitors the health of cluster components
type HealthChecker struct {
	logger        logger.Interface
	checks        map[string]HealthCheck
	checksMu      sync.RWMutex
	status        *HealthStatus
	statusMu      sync.RWMutex
	checkInterval time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// HealthCheck is a function that checks a component's health
type HealthCheck func() error

// HealthStatus represents the overall health status
type HealthStatus struct {
	Healthy    bool                       `json:"healthy"`
	Components map[string]ComponentHealth `json:"components"`
	LastCheck  time.Time                  `json:"last_check"`
	Message    string                     `json:"message,omitempty"`
}

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message,omitempty"`
	LastCheck time.Time `json:"last_check"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger logger.Interface, checkInterval time.Duration) *HealthChecker {
	if checkInterval == 0 {
		checkInterval = 5 * time.Second
	}

	return &HealthChecker{
		logger:        logger,
		checks:        make(map[string]HealthCheck),
		status:        &HealthStatus{Components: make(map[string]ComponentHealth)},
		checkInterval: checkInterval,
		stopCh:        make(chan struct{}),
	}
}

// RegisterCheck adds a health check for a component
func (h *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	h.checksMu.Lock()
	defer h.checksMu.Unlock()

	h.checks[name] = check
	h.logger.WithField("component", name).Debug("Registered health check")
}

// Start begins periodic health checking
func (h *HealthChecker) Start(ctx context.Context) error {
	h.logger.Info("Starting health checker")

	h.wg.Add(1)
	go h.runChecks(ctx)

	return nil
}

// Stop gracefully stops the health checker
func (h *HealthChecker) Stop() error {
	h.logger.Info("Stopping health checker")
	close(h.stopCh)
	h.wg.Wait()
	return nil
}

// runChecks executes health checks periodically
func (h *HealthChecker) runChecks(ctx context.Context) {
	defer h.wg.Done()

	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	// Run checks immediately
	h.executeChecks()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.executeChecks()
		}
	}
}

// executeChecks runs all registered health checks
func (h *HealthChecker) executeChecks() {
	h.checksMu.RLock()
	checks := make(map[string]HealthCheck, len(h.checks))
	for name, check := range h.checks {
		checks[name] = check
	}
	h.checksMu.RUnlock()

	components := make(map[string]ComponentHealth)
	overallHealthy := true

	for name, check := range checks {
		err := check()
		componentHealthy := err == nil

		message := ""
		if err != nil {
			message = err.Error()
			overallHealthy = false
		}

		components[name] = ComponentHealth{
			Healthy:   componentHealthy,
			Message:   message,
			LastCheck: time.Now(),
		}
	}

	// Update status
	h.statusMu.Lock()
	h.status = &HealthStatus{
		Healthy:    overallHealthy,
		Components: components,
		LastCheck:  time.Now(),
	}

	if !overallHealthy {
		h.status.Message = "One or more components are unhealthy"
	}
	h.statusMu.Unlock()

	if !overallHealthy {
		h.logger.Warn("Health check failed for one or more components")
	}
}

// GetStatus returns the current health status
func (h *HealthChecker) GetStatus() *HealthStatus {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()

	// Return a copy
	status := &HealthStatus{
		Healthy:    h.status.Healthy,
		Components: make(map[string]ComponentHealth, len(h.status.Components)),
		LastCheck:  h.status.LastCheck,
		Message:    h.status.Message,
	}

	for name, comp := range h.status.Components {
		status.Components[name] = comp
	}

	return status
}

// IsHealthy returns whether the system is healthy
func (h *HealthChecker) IsHealthy() bool {
	h.statusMu.RLock()
	defer h.statusMu.RUnlock()
	return h.status.Healthy
}

// Common health check functions

// DatabaseHealthCheck checks if the database is accessible
func DatabaseHealthCheck(db interface{ Health() error }) HealthCheck {
	return func() error {
		if err := db.Health(); err != nil {
			return fmt.Errorf("database health check failed: %w", err)
		}
		return nil
	}
}

// RaftHealthCheck checks if Raft is operational
func RaftHealthCheck(raft interface{ Stats() map[string]string }) HealthCheck {
	return func() error {
		stats := raft.Stats()
		state, ok := stats["state"]
		if !ok {
			return fmt.Errorf("cannot determine Raft state")
		}

		// Valid states: Follower, Candidate, Leader
		if state != "Follower" && state != "Candidate" && state != "Leader" {
			return fmt.Errorf("invalid Raft state: %s", state)
		}

		return nil
	}
}

// APIServerHealthCheck checks if the API server is responding
func APIServerHealthCheck(url string) HealthCheck {
	return func() error {
		// TODO: Implement HTTP health check
		// For now, always return healthy
		return nil
	}
}
