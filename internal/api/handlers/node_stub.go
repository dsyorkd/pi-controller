package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// List returns all nodes
func (h *NodeHandler) List(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node list endpoint not yet implemented",
	})
}

// Create creates a new node
func (h *NodeHandler) Create(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node create endpoint not yet implemented",
	})
}

// Get returns a specific node by ID
func (h *NodeHandler) Get(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node get endpoint not yet implemented",
	})
}

// Update updates a node
func (h *NodeHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node update endpoint not yet implemented",
	})
}

// Delete deletes a node
func (h *NodeHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node delete endpoint not yet implemented",
	})
}

// ListGPIO returns all GPIO devices for a node
func (h *NodeHandler) ListGPIO(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node GPIO list endpoint not yet implemented",
	})
}

// Provision provisions a node to a cluster
func (h *NodeHandler) Provision(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node provision endpoint not yet implemented",
	})
}

// Deprovision removes a node from its cluster
func (h *NodeHandler) Deprovision(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not Implemented",
		"message": "Node deprovision endpoint not yet implemented",
	})
}