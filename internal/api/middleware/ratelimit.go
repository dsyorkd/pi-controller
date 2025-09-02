package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int           `yaml:"requests_per_minute"`
	BurstSize         int           `yaml:"burst_size"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
	EnableByUser      bool          `yaml:"enable_by_user"`
	EnableByIP        bool          `yaml:"enable_by_ip"`
	WhitelistedIPs    []string      `yaml:"whitelisted_ips"`
}

// DefaultRateLimitConfig returns secure default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		CleanupInterval:   5 * time.Minute,
		EnableByUser:      true,
		EnableByIP:        true,
		WhitelistedIPs:    []string{"127.0.0.1", "::1"},
	}
}

// RateLimiter manages rate limiting for clients
type RateLimiter struct {
	config    *RateLimitConfig
	logger    *logrus.Entry
	limiters  map[string]*rate.Limiter
	mutex     sync.RWMutex
	lastClean time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig, logger *logrus.Logger) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimiter{
		config:    config,
		logger:    logger.WithField("component", "ratelimit"),
		limiters:  make(map[string]*rate.Limiter),
		lastClean: time.Now(),
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	rl.logger.WithFields(logrus.Fields{
		"requests_per_minute": config.RequestsPerMinute,
		"burst_size":          config.BurstSize,
		"enable_by_user":      config.EnableByUser,
		"enable_by_ip":        config.EnableByIP,
	}).Info("Rate limiter initialized")

	return rl
}

// RateLimit returns a rate limiting middleware
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier
		clientID := rl.getClientID(c)
		if clientID == "" {
			c.Next()
			return
		}

		// Check if IP is whitelisted
		if rl.isWhitelisted(c.ClientIP()) {
			c.Next()
			return
		}

		// Get or create limiter for client
		limiter := rl.getLimiter(clientID)

		// Check if request is allowed
		if !limiter.Allow() {
			rl.logger.WithFields(logrus.Fields{
				"client_id":  clientID,
				"client_ip":  c.ClientIP(),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"user_agent": c.GetHeader("User-Agent"),
			}).Warn("Rate limit exceeded")

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate Limit Exceeded",
				"message": "Too many requests, please slow down",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		remaining := int(limiter.Tokens())
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

		c.Next()
	}
}

// getClientID returns a client identifier for rate limiting
func (rl *RateLimiter) getClientID(c *gin.Context) string {
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
func (rl *RateLimiter) getLimiter(clientID string) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Check if limiter already exists
	if limiter, exists := rl.limiters[clientID]; exists {
		return limiter
	}

	// Create new limiter
	limiter := rate.NewLimiter(
		rate.Every(time.Minute/time.Duration(rl.config.RequestsPerMinute)),
		rl.config.BurstSize,
	)
	rl.limiters[clientID] = limiter

	return limiter
}

// isWhitelisted checks if an IP is whitelisted
func (rl *RateLimiter) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range rl.config.WhitelistedIPs {
		if ip == whitelistedIP {
			return true
		}
	}
	return false
}

// cleanupRoutine periodically cleans up old limiters
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes inactive limiters to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for clientID, limiter := range rl.limiters {
		// Remove limiter if it hasn't been used recently
		if limiter.Tokens() >= float64(rl.config.BurstSize) && now.Sub(rl.lastClean) > rl.config.CleanupInterval {
			delete(rl.limiters, clientID)
		}
	}

	rl.lastClean = now
	rl.logger.Debug("Rate limiter cleanup completed")
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	stats := map[string]interface{}{
		"active_limiters":     len(rl.limiters),
		"requests_per_minute": rl.config.RequestsPerMinute,
		"burst_size":          rl.config.BurstSize,
		"cleanup_interval":    rl.config.CleanupInterval.String(),
		"enable_by_user":      rl.config.EnableByUser,
		"enable_by_ip":        rl.config.EnableByIP,
	}

	return stats
}
