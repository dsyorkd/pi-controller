package services

import (
	"time"

	"gorm.io/gorm"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// NodeService handles node business logic
type NodeService struct {
	db     *storage.Database
	logger logger.Interface
}

// NewNodeService creates a new node service
func NewNodeService(db *storage.Database, logger logger.Interface) *NodeService {
	return &NodeService{
		db:     db,
		logger: logger.WithField("service", "node"),
	}
}

// CreateNodeRequest represents the request to create a node
type CreateNodeRequest struct {
	Name         string          `json:"name" validate:"required,min=1,max=100"`
	IPAddress    string          `json:"ip_address" validate:"required,ip"`
	MACAddress   string          `json:"mac_address" validate:"required,mac"`
	Role         models.NodeRole `json:"role" validate:"required,oneof=master worker"`
	ClusterID    *uint           `json:"cluster_id,omitempty"`
	Architecture string          `json:"architecture" validate:"max=50"`
	Model        string          `json:"model" validate:"max=100"`
	SerialNumber string          `json:"serial_number" validate:"max=100"`
	CPUCores     int             `json:"cpu_cores" validate:"min=1"`
	Memory       int64           `json:"memory" validate:"min=1"`
}

// UpdateNodeRequest represents the request to update a node
type UpdateNodeRequest struct {
	Name          *string            `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	IPAddress     *string            `json:"ip_address,omitempty" validate:"omitempty,ip"`
	MACAddress    *string            `json:"mac_address,omitempty" validate:"omitempty,mac"`
	Status        *models.NodeStatus `json:"status,omitempty"`
	Role          *models.NodeRole   `json:"role,omitempty" validate:"omitempty,oneof=master worker"`
	ClusterID     *uint              `json:"cluster_id,omitempty"`
	Architecture  *string            `json:"architecture,omitempty" validate:"omitempty,max=50"`
	Model         *string            `json:"model,omitempty" validate:"omitempty,max=100"`
	SerialNumber  *string            `json:"serial_number,omitempty" validate:"omitempty,max=100"`
	CPUCores      *int               `json:"cpu_cores,omitempty" validate:"omitempty,min=1"`
	Memory        *int64             `json:"memory,omitempty" validate:"omitempty,min=1"`
	OSVersion     *string            `json:"os_version,omitempty" validate:"omitempty,max=100"`
	KernelVersion *string            `json:"kernel_version,omitempty" validate:"omitempty,max=100"`
	KubeVersion   *string            `json:"kube_version,omitempty" validate:"omitempty,max=50"`
	NodeName      *string            `json:"node_name,omitempty" validate:"omitempty,max=100"`
}

// NodeListOptions represents options for listing nodes
type NodeListOptions struct {
	ClusterID   *uint
	Status      *models.NodeStatus
	Role        *models.NodeRole
	IncludeGPIO bool
	Limit       int
	Offset      int
}

// List returns a paginated list of nodes
func (s *NodeService) List(opts NodeListOptions) ([]models.Node, int64, error) {
	var nodes []models.Node
	var total int64

	query := s.db.DB().Model(&models.Node{})

	// Apply filters
	if opts.ClusterID != nil {
		query = query.Where("cluster_id = ?", *opts.ClusterID)
	}
	if opts.Status != nil {
		query = query.Where("status = ?", *opts.Status)
	}
	if opts.Role != nil {
		query = query.Where("role = ?", *opts.Role)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		s.logger.WithError(err).Error("Failed to count nodes")
		return nil, 0, errors.Wrapf(err, "failed to count nodes")
	}

	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	// Include relationships
	if opts.IncludeGPIO {
		query = query.Preload("GPIODevices")
	}
	query = query.Preload("Cluster")

	// Execute query
	if err := query.Find(&nodes).Error; err != nil {
		s.logger.WithError(err).Error("Failed to fetch nodes")
		return nil, 0, errors.Wrapf(err, "failed to fetch nodes")
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(nodes),
		"total": total,
	}).Debug("Fetched nodes")

	return nodes, total, nil
}

// GetByID returns a node by ID
func (s *NodeService) GetByID(id uint, includeGPIO bool) (*models.Node, error) {
	var node models.Node

	query := s.db.DB().Preload("Cluster")
	if includeGPIO {
		query = query.Preload("GPIODevices")
	}

	if err := query.First(&node, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to fetch node")
		return nil, errors.Wrapf(err, "failed to fetch node")
	}

	return &node, nil
}

// GetByName returns a node by name
func (s *NodeService) GetByName(name string) (*models.Node, error) {
	var node models.Node

	if err := s.db.DB().Preload("Cluster").Where("name = ?", name).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		s.logger.WithFields(map[string]interface{}{
			"name":  name,
			"error": err,
		}).Error("Failed to fetch node by name")
		return nil, errors.Wrapf(err, "failed to fetch node")
	}

	return &node, nil
}

// GetByIPAddress returns a node by IP address
func (s *NodeService) GetByIPAddress(ipAddress string) (*models.Node, error) {
	var node models.Node

	if err := s.db.DB().Preload("Cluster").Where("ip_address = ?", ipAddress).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		s.logger.WithFields(map[string]interface{}{
			"ip_address": ipAddress,
			"error":      err,
		}).Error("Failed to fetch node by IP address")
		return nil, errors.Wrapf(err, "failed to fetch node")
	}

	return &node, nil
}

// Create creates a new node
func (s *NodeService) Create(req CreateNodeRequest) (*models.Node, error) {
	// Check if node with same name already exists
	if _, err := s.GetByName(req.Name); err != ErrNotFound {
		if err == nil {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	// Check if node with same IP address already exists
	if _, err := s.GetByIPAddress(req.IPAddress); err != ErrNotFound {
		if err == nil {
			return nil, errors.Wrapf(ErrAlreadyExists, "node with IP address %s already exists", req.IPAddress)
		}
		return nil, err
	}

	// Validate cluster exists if provided
	if req.ClusterID != nil {
		var cluster models.Cluster
		if err := s.db.DB().First(&cluster, *req.ClusterID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.Wrapf(ErrNotFound, "cluster with ID %d not found", *req.ClusterID)
			}
			return nil, errors.Wrapf(err, "failed to validate cluster")
		}
	}

	node := models.Node{
		Name:         req.Name,
		IPAddress:    req.IPAddress,
		MACAddress:   req.MACAddress,
		Status:       models.NodeStatusDiscovered,
		Role:         req.Role,
		ClusterID:    req.ClusterID,
		Architecture: req.Architecture,
		Model:        req.Model,
		SerialNumber: req.SerialNumber,
		CPUCores:     req.CPUCores,
		Memory:       req.Memory,
		LastSeen:     time.Now(),
	}

	if err := s.db.DB().Create(&node).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"name":  req.Name,
			"error": err,
		}).Error("Failed to create node")
		return nil, errors.Wrapf(err, "failed to create node")
	}

	s.logger.WithFields(map[string]interface{}{
		"id":   node.ID,
		"name": node.Name,
		"ip":   node.IPAddress,
	}).Info("Node created successfully")

	return &node, nil
}

// Update updates an existing node
func (s *NodeService) Update(id uint, req UpdateNodeRequest) (*models.Node, error) {
	node, err := s.GetByID(id, false)
	if err != nil {
		return nil, err
	}

	// Check if name is being changed and if new name already exists
	if req.Name != nil && *req.Name != node.Name {
		if _, err := s.GetByName(*req.Name); err != ErrNotFound {
			if err == nil {
				return nil, ErrAlreadyExists
			}
			return nil, err
		}
		node.Name = *req.Name
	}

	// Check if IP address is being changed and if new IP already exists
	if req.IPAddress != nil && *req.IPAddress != node.IPAddress {
		if _, err := s.GetByIPAddress(*req.IPAddress); err != ErrNotFound {
			if err == nil {
				return nil, errors.Wrapf(ErrAlreadyExists, "node with IP address %s already exists", *req.IPAddress)
			}
			return nil, err
		}
		node.IPAddress = *req.IPAddress
	}

	// Validate cluster exists if being changed
	if req.ClusterID != nil && req.ClusterID != node.ClusterID {
		var cluster models.Cluster
		if err := s.db.DB().First(&cluster, *req.ClusterID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, errors.Wrapf(ErrNotFound, "cluster with ID %d not found", *req.ClusterID)
			}
			return nil, errors.Wrapf(err, "failed to validate cluster")
		}
		node.ClusterID = req.ClusterID
	}

	// Update fields
	if req.MACAddress != nil {
		node.MACAddress = *req.MACAddress
	}
	if req.Status != nil {
		node.Status = *req.Status
	}
	if req.Role != nil {
		node.Role = *req.Role
	}
	if req.Architecture != nil {
		node.Architecture = *req.Architecture
	}
	if req.Model != nil {
		node.Model = *req.Model
	}
	if req.SerialNumber != nil {
		node.SerialNumber = *req.SerialNumber
	}
	if req.CPUCores != nil {
		node.CPUCores = *req.CPUCores
	}
	if req.Memory != nil {
		node.Memory = *req.Memory
	}
	if req.OSVersion != nil {
		node.OSVersion = *req.OSVersion
	}
	if req.KernelVersion != nil {
		node.KernelVersion = *req.KernelVersion
	}
	if req.KubeVersion != nil {
		node.KubeVersion = *req.KubeVersion
	}
	if req.NodeName != nil {
		node.NodeName = *req.NodeName
	}

	if err := s.db.DB().Save(node).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to update node")
		return nil, errors.Wrapf(err, "failed to update node")
	}

	s.logger.WithFields(map[string]interface{}{
		"id":   node.ID,
		"name": node.Name,
	}).Info("Node updated successfully")

	return node, nil
}

// Delete deletes a node
func (s *NodeService) Delete(id uint) error {
	node, err := s.GetByID(id, true)
	if err != nil {
		return err
	}

	// Check if node has GPIO devices
	if len(node.GPIODevices) > 0 {
		return ErrHasAssociatedResources
	}

	if err := s.db.DB().Delete(&models.Node{}, id).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to delete node")
		return errors.Wrapf(err, "failed to delete node")
	}

	s.logger.WithFields(map[string]interface{}{
		"id":   id,
		"name": node.Name,
	}).Info("Node deleted successfully")

	return nil
}

// UpdateLastSeen updates the last seen timestamp for a node
func (s *NodeService) UpdateLastSeen(id uint) error {
	if err := s.db.DB().Model(&models.Node{}).Where("id = ?", id).Update("last_seen", time.Now()).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"id":    id,
			"error": err,
		}).Error("Failed to update node last seen")
		return errors.Wrapf(err, "failed to update node last seen")
	}

	return nil
}

// GetGPIODevices returns all GPIO devices for a node
func (s *NodeService) GetGPIODevices(nodeID uint) ([]models.GPIODevice, error) {
	node, err := s.GetByID(nodeID, false)
	if err != nil {
		return nil, err
	}

	var devices []models.GPIODevice
	if err := s.db.DB().Where("node_id = ?", node.ID).Find(&devices).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"node_id": nodeID,
			"error":   err,
		}).Error("Failed to fetch node GPIO devices")
		return nil, errors.Wrapf(err, "failed to fetch node GPIO devices")
	}

	return devices, nil
}

// Provision marks a node as being provisioned and updates its cluster assignment
func (s *NodeService) Provision(id uint, clusterID uint) error {
	node, err := s.GetByID(id, false)
	if err != nil {
		return err
	}

	// Validate cluster exists
	var cluster models.Cluster
	if err := s.db.DB().First(&cluster, clusterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.Wrapf(ErrNotFound, "cluster with ID %d not found", clusterID)
		}
		return errors.Wrapf(err, "failed to validate cluster")
	}

	// Update node status and cluster assignment
	node.Status = models.NodeStatusProvisioning
	node.ClusterID = &clusterID

	if err := s.db.DB().Save(node).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"node_id":    id,
			"cluster_id": clusterID,
			"error":      err,
		}).Error("Failed to provision node")
		return errors.Wrapf(err, "failed to provision node")
	}

	s.logger.WithFields(map[string]interface{}{
		"node_id":      id,
		"node_name":    node.Name,
		"cluster_id":   clusterID,
		"cluster_name": cluster.Name,
	}).Info("Node provisioning started")

	return nil
}

// Deprovision removes a node from its cluster and marks it as discovered
func (s *NodeService) Deprovision(id uint) error {
	node, err := s.GetByID(id, false)
	if err != nil {
		return err
	}

	oldClusterID := node.ClusterID

	// Update node status and remove cluster assignment
	node.Status = models.NodeStatusDiscovered
	node.ClusterID = nil
	node.NodeName = ""
	node.KubeVersion = ""

	if err := s.db.DB().Save(node).Error; err != nil {
		s.logger.WithFields(map[string]interface{}{
			"node_id": id,
			"error":   err,
		}).Error("Failed to deprovision node")
		return errors.Wrapf(err, "failed to deprovision node")
	}

	s.logger.WithFields(map[string]interface{}{
		"node_id":    id,
		"node_name":  node.Name,
		"cluster_id": oldClusterID,
	}).Info("Node deprovisioned successfully")

	return nil
}
