package handlers

import (
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// NodeHandler handles node-related API operations
type NodeHandler struct {
	service *services.NodeService
	logger  logger.Interface
}

// NewNodeHandler creates a new node handler
func NewNodeHandler(service *services.NodeService, logger logger.Interface) *NodeHandler {
	return &NodeHandler{
		service: service,
		logger:  logger.WithField("handler", "node"),
	}
}