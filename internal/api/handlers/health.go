package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dsyorkd/pi-controller/internal/storage"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	database *storage.Database
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *storage.Database) *HealthHandler {
	return &HealthHandler{
		database: db,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

var startTime = time.Now()

// Health returns the basic health status
func (h *HealthHandler) Health(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime).String(),
	}

	c.JSON(http.StatusOK, response)
}

// Ready returns the readiness status including service dependencies
func (h *HealthHandler) Ready(c *gin.Context) {
	services := make(map[string]string)
	status := "ready"
	statusCode := http.StatusOK

	// Check database connectivity
	if err := h.database.Health(); err != nil {
		services["database"] = "unhealthy: " + err.Error()
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	} else {
		services["database"] = "healthy"
	}

	response := ReadinessResponse{
		Status:    status,
		Timestamp: time.Now(),
		Services:  services,
	}

	c.JSON(statusCode, response)
}

// SystemInfo returns system information
func SystemInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := gin.H{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"cpu_count":  runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"memory": gin.H{
			"alloc":        m.Alloc,
			"total_alloc":  m.TotalAlloc,
			"sys":          m.Sys,
			"heap_alloc":   m.HeapAlloc,
			"heap_sys":     m.HeapSys,
			"heap_inuse":   m.HeapInuse,
			"heap_idle":    m.HeapIdle,
			"heap_objects": m.HeapObjects,
		},
		"gc": gin.H{
			"num_gc":      m.NumGC,
			"pause_total": m.PauseTotalNs,
			"last_gc":     time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
		},
		"timestamp": time.Now(),
		"uptime":    time.Since(startTime).String(),
	}

	c.JSON(http.StatusOK, info)
}

// SystemMetrics returns basic system metrics
func SystemMetrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := gin.H{
		"uptime_seconds":    time.Since(startTime).Seconds(),
		"goroutines_count":  runtime.NumGoroutine(),
		"memory_alloc":      m.Alloc,
		"memory_sys":        m.Sys,
		"gc_count":          m.NumGC,
		"gc_pause_total_ns": m.PauseTotalNs,
		"timestamp":         time.Now().Unix(),
	}

	c.JSON(http.StatusOK, metrics)
}
