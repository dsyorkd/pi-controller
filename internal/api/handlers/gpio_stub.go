package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// List returns all GPIO devices
func (h *GPIOHandler) List(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO list endpoint not yet implemented",
	})
}

// Create creates a new GPIO device
func (h *GPIOHandler) Create(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO create endpoint not yet implemented",
	})
}

// Get returns a specific GPIO device by ID
func (h *GPIOHandler) Get(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO get endpoint not yet implemented",
	})
}

// Update updates a GPIO device
func (h *GPIOHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO update endpoint not yet implemented",
	})
}

// Delete deletes a GPIO device
func (h *GPIOHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO delete endpoint not yet implemented",
	})
}

// Read reads the current value of a GPIO device
func (h *GPIOHandler) Read(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO read endpoint not yet implemented",
	})
}

// Write writes a value to a GPIO device
func (h *GPIOHandler) Write(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO write endpoint not yet implemented",
	})
}

// GetReadings returns GPIO readings for a device
func (h *GPIOHandler) GetReadings(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "GPIO readings endpoint not yet implemented",
	})
}