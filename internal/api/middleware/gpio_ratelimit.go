package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	applogger "github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// GPIORateLimitConfig holds rate limiting configuration specific to GPIO operations
type GPIORateLimitConfig struct {
	Enabled           bool          `yaml:"enabled"`
	RequestsPerSecond int           `yaml:"requests_per_second"`
	BurstSize         int           `yaml:"burst_size"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
	EnableByUser      bool          `yaml:"enable_by_user"`
	EnableByIP        bool          `yaml:"enable_by_ip"`
	WhitelistedIPs    []string      `yaml:"whitelisted_ips"`
}

// DefaultGPIORateLimitConfig returns strict default rate limiting for GPIO endpoints
// GPIO operations can cause hardware damage if performed too rapidly
func DefaultGPIORateLimitConfig() *GPIORateLimitConfig {
	return &GPIORateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10, // Conservative limit to protect hardware
		BurstSize:         20, // Allow small bursts for multi-pin operations
		CleanupInterval:   5 * time.Minute,
		EnableByUser:      true,
		EnableByIP:        true,
		WhitelistedIPs:    []string{"127.0.0.1", "::1"},
	}
}

// GPIORateLimiter manages rate limiting specifically for GPIO endpoints
type GPIORateLimiter struct {
	config    *GPIORateLimitConfig
	logger    applogger.Interface
	limiters  map[string]*rate.Limiter
	mutex     sync.RWMutex
	lastClean time.Time
}

// NewGPIORateLimiter creates a new GPIO-specific rate limiter
func NewGPIORateLimiter(config *GPIORateLimitConfig, logger applogger.Interface) *GPIORateLimiter {
	if config == nil {
		config = DefaultGPIORateLimitConfig()
	}

	rl := &GPIORateLimiter{
		config:    config,
		logger:    logger.WithField("component", "gpio_ratelimit"),
		limiters:  make(map[string]*rate.Limiter),
		lastClean: time.Now(),
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	rl.logger.WithFields(map[string]interface{}{
		"requests_per_second": config.RequestsPerSecond,
		"burst_size":          config.BurstSize,
		"enable_by_user":      config.EnableByUser,
		"enable_by_ip":        config.EnableByIP,
	}).Info("GPIO rate limiter initialized with strict limits")

	return rl
}

// RateLimit returns a rate limiting middleware for GPIO operations
func (rl *GPIORateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting if disabled
		if !rl.config.Enabled {
			c.Next()
			return
		}

		// Get client identifier
		clientID := rl.getClientID(c)
		if clientID == "" {
			c.Next()
			return
		}

		// Check if IP is whitelisted
		if rl.isWhitelisted(c.ClientIP()) {
			rl.logger.WithField("client_ip", sanitizeLogValue(c.ClientIP())).Debug("Whitelisted IP, skipping rate limit")
			c.Next()
			return
		}

		// Get or create limiter for client
		limiter := rl.getLimiter(clientID)

		// Check if request is allowed
		if !limiter.Allow() {
			rl.logger.WithFields(map[string]interface{}{
				"client_id":  sanitizeLogValue(clientID),
				"client_ip":  sanitizeLogValue(c.ClientIP()),
				"method":     sanitizeLogValue(c.Request.Method),
				"path":       sanitizeLogValue(c.Request.URL.Path),
				"user_agent": sanitizeLogValue(c.GetHeader("User-Agent")),
			}).Warn("GPIO rate limit exceeded - potential hardware abuse attempt")

			// Calculate retry after time
			retryAfter := int(time.Second.Seconds())

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerSecond))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate Limit Exceeded",
				"message":     "GPIO operation rate limit exceeded. Rapid GPIO switching can damage hardware.",
				"retry_after": retryAfter,
				"limit":       rl.config.RequestsPerSecond,
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		remaining := int(limiter.Tokens())
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

		c.Next()
	}
}

// getClientID returns a client identifier for rate limiting
func (rl *GPIORateLimiter) getClientID(c *gin.Context) string {
	// Prioritize user-based rate limiting if enabled and user is authenticated
	if rl.config.EnableByUser {
		if userID := GetUserID(c); userID != "" {
			return "user:" + userID
		}
	}

	// Fall back to IP-based rate limiting if enabled
	if rl.config.EnableByIP {
		return "ip:" + c.ClientIP()
	}

	return ""
}

// getLimiter gets or creates a rate limiter for a client
func (rl *GPIORateLimiter) getLimiter(clientID string) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Check if limiter already exists
	if limiter, exists := rl.limiters[clientID]; exists {
		return limiter
	}

	// Create new limiter with per-second rate
	limiter := rate.NewLimiter(
		rate.Limit(rl.config.RequestsPerSecond),
		rl.config.BurstSize,
	)
	rl.limiters[clientID] = limiter

	rl.logger.WithField("client_id", sanitizeLogValue(clientID)).Debug("Created new GPIO rate limiter for client")

	return limiter
}

// isWhitelisted checks if an IP is whitelisted
func (rl *GPIORateLimiter) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range rl.config.WhitelistedIPs {
		if ip == whitelistedIP {
			return true
		}
	}
	return false
}

// cleanupRoutine periodically cleans up old limiters
func (rl *GPIORateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes inactive limiters to prevent memory leaks
func (rl *GPIORateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	removed := 0

	for clientID, limiter := range rl.limiters {
		// Remove limiter if it's at full capacity (hasn't been used recently)
		if limiter.Tokens() >= float64(rl.config.BurstSize) {
			delete(rl.limiters, clientID)
			removed++
		}
	}

	rl.lastClean = now

	if removed > 0 {
		rl.logger.WithField("removed_count", removed).Debug("GPIO rate limiter cleanup completed")
	}
}

// GetStats returns GPIO rate limiting statistics
func (rl *GPIORateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	return map[string]interface{}{
		"active_limiters":     len(rl.limiters),
		"requests_per_second": rl.config.RequestsPerSecond,
		"burst_size":          rl.config.BurstSize,
		"cleanup_interval":    rl.config.CleanupInterval.String(),
		"enable_by_user":      rl.config.EnableByUser,
		"enable_by_ip":        rl.config.EnableByIP,
	}
}
