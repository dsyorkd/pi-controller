package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dsyorkd/pi-controller/internal/logger"
)

// Recovery creates a gin middleware for panic recovery with logrus
func Recovery(logger logger.Interface) gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, recovered interface{}) {
		logger.WithFields(map[string]interface{}{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
			"panic":  recovered,
		}).Error("Panic recovered in HTTP handler")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "An unexpected error occurred",
		})
	})
}