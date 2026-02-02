package middleware

import (
	"time"

	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/gin-gonic/gin"
)

// Logger creates a gin middleware for request logging using logrus
func Logger(logger logger.Interface) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Use logrus for structured logging instead of default formatter
			entry := logger.WithFields(map[string]interface{}{
				"status":     param.StatusCode,
				"method":     param.Method,
				"path":       param.Path,
				"ip":         param.ClientIP,
				"user_agent": param.Request.UserAgent(),
				"latency":    param.Latency.String(),
				"time":       param.TimeStamp.Format(time.RFC3339),
			})

			if param.ErrorMessage != "" {
				entry = entry.WithField("error", param.ErrorMessage)
			}

			if param.StatusCode >= 400 {
				entry.Error("HTTP request completed with error")
			} else {
				entry.Info("HTTP request completed")
			}

			return ""
		},
		Output:    gin.DefaultWriter,
		SkipPaths: []string{"/health", "/ready"},
	})
}
