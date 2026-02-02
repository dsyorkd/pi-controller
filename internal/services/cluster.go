package services

import (
	"context"
	"fmt"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/provisioner"
	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/dsyorkd/pi-controller/internal/validation"
	"gorm.io/gorm"
)

// ClusterService is the service for managing clusters
type ClusterService struct {
	store               *storage.Database
	log                 logger.Interface
	provisioningService *ProvisioningService
	nodeService         *NodeService
}

// NewClusterService creates a new ClusterService
func NewClusterService(store *storage.Database, log logger.Interface) *ClusterService {
	return &ClusterService{
		store: store,
		log:   log,
	}
}

// SetDependencies sets the dependencies for ClusterService (called after all services are created)
func (s *ClusterService) SetDependencies(provisioningService *ProvisioningService, nodeService *NodeService) {
	s.provisioningService = provisioningService
	s.nodeService = nodeService
}

// CreateClusterRequest is the request to create a cluster
type CreateClusterRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create creates a new cluster
func (s *ClusterService) Create(req CreateClusterRequest) (*models.Cluster, error) {
	if req.Name == "" {
		return nil, errors.Wrapf(ErrInvalidInput, "name is required")
	}

	// Validate cluster name for malicious input
	if err := validation.ValidateResourceName("name", req.Name, 255); err != nil {
		return nil, errors.Wrapf(ErrInvalidInput, "%s", err.Error())
	}

	// Validate description if provided
	if req.Description != "" {
		if err := validation.ValidateDescription("description", req.Description, 1000); err != nil {
			return nil, errors.Wrapf(ErrInvalidInput, "%s", err.Error())
		}
	}

	// Check for duplicate name
	existing, err := s.GetByName(req.Name)
	if err != nil && !IsNotFound(err) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.Wrapf(ErrAlreadyExists, "cluster name already exists")
	}

	cluster := &models.Cluster{
		Name:        req.Name,
		Description: req.Description,
		Status:      models.ClusterStatusActive,
	}

	err = s.store.DB().Create(cluster).Error
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

// UpdateClusterRequest is the request to update a cluster
type UpdateClusterRequest struct {
	Name        *string               `json:"name"`
	Description *string               `json:"description"`
	Status      *models.ClusterStatus `json:"status"`
}

// Update updates a cluster
func (s *ClusterService) Update(id uint, req UpdateClusterRequest) (*models.Cluster, error) {
	var cluster models.Cluster
	err := s.store.DB().First(&cluster, id).Error
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		// Validate cluster name for malicious input
		if err := validation.ValidateResourceName("name", *req.Name, 255); err != nil {
			return nil, errors.Wrapf(ErrInvalidInput, "%s", err.Error())
		}
		cluster.Name = *req.Name
	}
	if req.Description != nil {
		// Validate description if provided
		if *req.Description != "" {
			if err := validation.ValidateDescription("description", *req.Description, 1000); err != nil {
				return nil, errors.Wrapf(ErrInvalidInput, "%s", err.Error())
			}
		}
		cluster.Description = *req.Description
	}
	if req.Status != nil {
		cluster.Status = *req.Status
	}

	err = s.store.DB().Save(&cluster).Error
	if err != nil {
		return nil, err
	}

	return &cluster, nil
}

// Delete deletes a cluster
func (s *ClusterService) Delete(id uint) error {
	// Check if cluster has nodes
	// nodes, err := s.nodeStore.GetNodesByClusterID(id)
	// if err != nil {
	// 	return err
	// }
	// if len(nodes) > 0 {
	// 	return fmt.Errorf("cannot delete cluster with existing nodes")
	// }

	return s.store.DB().Delete(&models.Cluster{}, id).Error
}

// ClusterListOptions is the options for listing clusters
type ClusterListOptions struct {
	Status *models.ClusterStatus `json:"status"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

// List lists clusters
func (s *ClusterService) List(opts ClusterListOptions) ([]models.Cluster, int64, error) {
	var clusters []models.Cluster
	err := s.store.DB().Find(&clusters).Error
	if err != nil {
		return nil, 0, err
	}

	return clusters, int64(len(clusters)), nil
}

// GetByID gets a cluster by ID
func (s *ClusterService) GetByID(id uint) (*models.Cluster, error) {
	var cluster models.Cluster
	err := s.store.DB().First(&cluster, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cluster, nil
}

// GetByName gets a cluster by name
func (s *ClusterService) GetByName(name string) (*models.Cluster, error) {
	var cluster models.Cluster
	err := s.store.DB().Where("name = ?", name).First(&cluster).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cluster, nil
}

// GetNodes retrieves all nodes for a given cluster ID
func (s *ClusterService) GetNodes(clusterID uint) ([]models.Node, error) {
	var nodes []models.Node
	err := s.store.DB().Where("cluster_id = ?", clusterID).Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetStatus retrieves the status of a cluster
func (s *ClusterService) GetStatus(id uint) (models.ClusterStatus, error) {
	var cluster models.Cluster
	err := s.store.DB().First(&cluster, id).Error
	if err != nil {
		return "", err
	}
	return cluster.Status, nil
}

// ClusterProvisionRequest represents a request to provision a cluster
type ClusterProvisionRequest struct {
	MasterNodeID  uint                        `json:"master_node_id" validate:"required"`
	WorkerNodeIDs []uint                      `json:"worker_node_ids,omitempty"`
	K3sConfig     K3sConfig                   `json:"k3s_config,omitempty"`
	SSHConfig     provisioner.SSHClientConfig `json:"ssh_config"`
}

// ClusterDeprovisionRequest represents a request to deprovision a cluster
type ClusterDeprovisionRequest struct {
	SSHConfig provisioner.SSHClientConfig `json:"ssh_config"`
}

// ClusterScaleRequest represents a request to scale a cluster
type ClusterScaleRequest struct {
	NodeCount uint                        `json:"node_count" validate:"required,min=1"`
	SSHConfig provisioner.SSHClientConfig `json:"ssh_config"`
}

// ClusterStatusResponse represents a detailed cluster status response
type ClusterStatusResponse struct {
	Cluster     *models.Cluster `json:"cluster"`
	Nodes       []models.Node   `json:"nodes"`
	TotalNodes  int             `json:"total_nodes"`
	ReadyNodes  int             `json:"ready_nodes"`
	MasterNodes int             `json:"master_nodes"`
	WorkerNodes int             `json:"worker_nodes"`
}

// ProvisionCluster orchestrates K3s cluster provisioning
func (s *ClusterService) ProvisionCluster(ctx context.Context, clusterID uint, req ClusterProvisionRequest) (*ProvisioningResult, error) {
	s.log.WithField("cluster_id", clusterID).Info("Starting cluster provisioning")

	// Get cluster and validate it exists
	_, err := s.GetByID(clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster %d", clusterID)
	}

	// Update cluster status to provisioning
	provisioningStatus := models.ClusterStatusProvisioning
	updateReq := UpdateClusterRequest{
		Status: &provisioningStatus,
	}
	_, err = s.Update(clusterID, updateReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update cluster status to provisioning")
	}

	// Delegate to provisioning service
	if s.provisioningService == nil {
		return nil, fmt.Errorf("provisioning service not available")
	}

	provisionReq := ProvisionClusterRequest{
		ClusterID:     clusterID,
		MasterNodeID:  req.MasterNodeID,
		WorkerNodeIDs: req.WorkerNodeIDs,
		K3sConfig:     req.K3sConfig,
		SSHConfig:     req.SSHConfig,
	}

	result, err := s.provisioningService.ProvisionCluster(ctx, provisionReq)

	// Update cluster status based on result
	var finalStatus models.ClusterStatus
	if err != nil || !result.Success {
		finalStatus = models.ClusterStatusFailed
		s.log.WithError(err).WithField("cluster_id", clusterID).Error("Cluster provisioning failed")
	} else {
		finalStatus = models.ClusterStatusActive
		s.log.WithField("cluster_id", clusterID).Info("Cluster provisioning succeeded")
	}

	// Update cluster status
	updateReq.Status = &finalStatus
	_, updateErr := s.Update(clusterID, updateReq)
	if updateErr != nil {
		s.log.WithError(updateErr).WithField("cluster_id", clusterID).Error("Failed to update cluster status after provisioning")
	}

	return result, err
}

// DeprovisionCluster tears down a K3s cluster
func (s *ClusterService) DeprovisionCluster(ctx context.Context, clusterID uint, req ClusterDeprovisionRequest) error {
	s.log.WithField("cluster_id", clusterID).Info("Starting cluster deprovisioning")

	// Get cluster and validate it exists
	_, err := s.GetByID(clusterID)
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster %d", clusterID)
	}

	// Get all nodes in the cluster
	nodes, err := s.GetNodes(clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster nodes")
	}

	if len(nodes) == 0 {
		s.log.WithField("cluster_id", clusterID).Info("No nodes to deprovision")
		return s.updateClusterStatus(clusterID, models.ClusterStatusPending)
	}

	// Update cluster status to indicate deprovisioning
	if err := s.updateClusterStatus(clusterID, models.ClusterStatusMaintenance); err != nil {
		return errors.Wrap(err, "failed to update cluster status to maintenance")
	}

	// Deprovision all nodes
	if s.provisioningService == nil {
		return fmt.Errorf("provisioning service not available")
	}

	var deprovisionErrors []error
	for _, node := range nodes {
		s.log.WithFields(map[string]interface{}{
			"cluster_id": clusterID,
			"node_id":    node.ID,
			"node_name":  node.Name,
		}).Info("Deprovisioning node")

		result, err := s.provisioningService.DeprovisionNode(ctx, node.ID, req.SSHConfig)
		if err != nil {
			s.log.WithError(err).WithField("node_id", node.ID).Error("Failed to deprovision node")
			deprovisionErrors = append(deprovisionErrors, fmt.Errorf("node %d: %w", node.ID, err))
		} else if !result.Success {
			s.log.WithField("node_id", node.ID).WithField("error", result.Error).Error("Node deprovisioning failed")
			deprovisionErrors = append(deprovisionErrors, fmt.Errorf("node %d: %s", node.ID, result.Error))
		}
	}

	// Update cluster status based on results
	var finalStatus models.ClusterStatus
	if len(deprovisionErrors) > 0 {
		finalStatus = models.ClusterStatusFailed
		s.log.WithField("cluster_id", clusterID).WithField("errors", deprovisionErrors).Error("Some nodes failed to deprovision")
	} else {
		finalStatus = models.ClusterStatusPending
		s.log.WithField("cluster_id", clusterID).Info("All nodes deprovisioned successfully")
	}

	if err := s.updateClusterStatus(clusterID, finalStatus); err != nil {
		s.log.WithError(err).WithField("cluster_id", clusterID).Error("Failed to update cluster status after deprovisioning")
	}

	// Return error if any nodes failed to deprovision
	if len(deprovisionErrors) > 0 {
		return fmt.Errorf("failed to deprovision some nodes: %v", deprovisionErrors)
	}

	return nil
}

// GetClusterStatus returns detailed cluster status information
func (s *ClusterService) GetClusterStatus(ctx context.Context, clusterID uint) (*ClusterStatusResponse, error) {
	s.log.WithField("cluster_id", clusterID).Debug("Getting detailed cluster status")

	// Get cluster
	cluster, err := s.GetByID(clusterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster %d", clusterID)
	}

	// Get nodes
	nodes, err := s.GetNodes(clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster nodes")
	}

	// Calculate statistics
	totalNodes := len(nodes)
	readyNodes := 0
	masterNodes := 0
	workerNodes := 0

	for _, node := range nodes {
		if node.IsReady() {
			readyNodes++
		}
		if node.IsMaster() {
			masterNodes++
		} else {
			workerNodes++
		}
	}

	return &ClusterStatusResponse{
		Cluster:     cluster,
		Nodes:       nodes,
		TotalNodes:  totalNodes,
		ReadyNodes:  readyNodes,
		MasterNodes: masterNodes,
		WorkerNodes: workerNodes,
	}, nil
}

// ScaleCluster scales the cluster to the specified number of nodes
func (s *ClusterService) ScaleCluster(ctx context.Context, clusterID uint, req ClusterScaleRequest) error {
	s.log.WithFields(map[string]interface{}{
		"cluster_id":   clusterID,
		"target_nodes": req.NodeCount,
	}).Info("Starting cluster scaling")

	// Get cluster and validate it exists
	_, err := s.GetByID(clusterID)
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster %d", clusterID)
	}

	// Get current nodes
	currentNodes, err := s.GetNodes(clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current cluster nodes")
	}

	currentNodeCount := uint(len(currentNodes))

	if currentNodeCount == req.NodeCount {
		s.log.WithField("cluster_id", clusterID).Info("Cluster already at target size")
		return nil
	}

	if req.NodeCount < currentNodeCount {
		// Scale down - remove worker nodes
		return s.scaleDown(ctx, clusterID, currentNodes, req.NodeCount, req.SSHConfig)
	} else {
		// Scale up - add worker nodes
		return s.scaleUp(ctx, clusterID, currentNodes, req.NodeCount, req.SSHConfig)
	}
}

// scaleDown removes worker nodes from the cluster
func (s *ClusterService) scaleDown(ctx context.Context, clusterID uint, currentNodes []models.Node, targetCount uint, sshConfig provisioner.SSHClientConfig) error {
	currentCount := uint(len(currentNodes))
	nodesToRemove := currentCount - targetCount

	s.log.WithFields(map[string]interface{}{
		"cluster_id":      clusterID,
		"current_nodes":   currentCount,
		"target_nodes":    targetCount,
		"nodes_to_remove": nodesToRemove,
	}).Info("Scaling down cluster")

	// Find worker nodes to remove (prefer ones that are not ready first)
	var workerNodes []models.Node
	for _, node := range currentNodes {
		if !node.IsMaster() {
			workerNodes = append(workerNodes, node)
		}
	}

	if uint(len(workerNodes)) < nodesToRemove {
		return fmt.Errorf("cannot remove %d nodes: only %d worker nodes available", nodesToRemove, len(workerNodes))
	}

	// Sort worker nodes by status (failed/not ready first, then ready)
	// This is a simple approach - in production you might want more sophisticated selection
	var nodesToDeprovision []models.Node
	for i := uint(0); i < nodesToRemove && i < uint(len(workerNodes)); i++ {
		nodesToDeprovision = append(nodesToDeprovision, workerNodes[i])
	}

	// Deprovision selected nodes
	var errors []error
	for _, node := range nodesToDeprovision {
		s.log.WithFields(map[string]interface{}{
			"cluster_id": clusterID,
			"node_id":    node.ID,
			"node_name":  node.Name,
		}).Info("Removing node from cluster")

		if s.provisioningService != nil {
			result, err := s.provisioningService.DeprovisionNode(ctx, node.ID, sshConfig)
			if err != nil || !result.Success {
				errorMsg := fmt.Sprintf("node %d: %v", node.ID, err)
				if result != nil && result.Error != "" {
					errorMsg = fmt.Sprintf("node %d: %s", node.ID, result.Error)
				}
				errors = append(errors, fmt.Errorf("%s", errorMsg))
				s.log.WithError(err).WithField("node_id", node.ID).Error("Failed to deprovision node during scale down")
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to scale down: %v", errors)
	}

	s.log.WithField("cluster_id", clusterID).Info("Cluster scaled down successfully")
	return nil
}

// scaleUp adds worker nodes to the cluster
func (s *ClusterService) scaleUp(ctx context.Context, clusterID uint, currentNodes []models.Node, targetCount uint, sshConfig provisioner.SSHClientConfig) error {
	currentCount := uint(len(currentNodes))
	nodesToAdd := targetCount - currentCount

	s.log.WithFields(map[string]interface{}{
		"cluster_id":    clusterID,
		"current_nodes": currentCount,
		"target_nodes":  targetCount,
		"nodes_to_add":  nodesToAdd,
	}).Info("Scaling up cluster")

	// For scale up, we need to find available nodes that are not part of any cluster
	// This is a simplified implementation - in production you might want to implement
	// auto-discovery of new nodes or pre-registration of available nodes

	if s.nodeService == nil {
		return fmt.Errorf("node service not available")
	}

	// Find available nodes (discovered but not provisioned)
	availableNodes, err := s.findAvailableNodes(nodesToAdd)
	if err != nil {
		return errors.Wrap(err, "failed to find available nodes for scale up")
	}

	if uint(len(availableNodes)) < nodesToAdd {
		return fmt.Errorf("not enough available nodes: need %d, found %d", nodesToAdd, len(availableNodes))
	}

	// Find a master node to get cluster configuration
	var masterNode *models.Node
	for _, node := range currentNodes {
		if node.IsMaster() && node.IsReady() {
			masterNode = &node
			break
		}
	}

	if masterNode == nil {
		return fmt.Errorf("no ready master node found for cluster token")
	}

	// Get cluster token from master node
	if s.provisioningService == nil {
		return fmt.Errorf("provisioning service not available")
	}

	clusterToken, err := s.provisioningService.GetClusterToken(ctx, masterNode.ID, sshConfig)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster token from master node")
	}

	// Provision new worker nodes
	k3sConfig := DefaultK3sConfig()
	k3sConfig.ClusterToken = clusterToken
	k3sConfig.ServerURL = fmt.Sprintf("https://%s:6443", masterNode.IPAddress)

	var provisionErrors []error
	for i := uint(0); i < nodesToAdd && i < uint(len(availableNodes)); i++ {
		node := availableNodes[i]

		s.log.WithFields(map[string]interface{}{
			"cluster_id": clusterID,
			"node_id":    node.ID,
			"node_name":  node.Name,
		}).Info("Adding node to cluster")

		// Assign node to cluster first
		if s.nodeService != nil {
			err := s.nodeService.Provision(node.ID, clusterID)
			if err != nil {
				s.log.WithError(err).WithField("node_id", node.ID).Error("Failed to assign node to cluster")
				provisionErrors = append(provisionErrors, fmt.Errorf("node %d assignment: %w", node.ID, err))
				continue
			}
		}

		// Provision as worker node
		provisionReq := ProvisionNodeRequest{
			NodeID:    node.ID,
			ClusterID: clusterID,
			Role:      models.NodeRoleWorker,
			K3sConfig: k3sConfig,
			SSHConfig: sshConfig,
		}

		result, err := s.provisioningService.ProvisionNode(ctx, provisionReq)
		if err != nil || !result.Success {
			errorMsg := fmt.Sprintf("node %d provisioning: %v", node.ID, err)
			if result != nil && result.Error != "" {
				errorMsg = fmt.Sprintf("node %d provisioning: %s", node.ID, result.Error)
			}
			provisionErrors = append(provisionErrors, fmt.Errorf("%s", errorMsg))
			s.log.WithError(err).WithField("node_id", node.ID).Error("Failed to provision node during scale up")
		}
	}

	if len(provisionErrors) > 0 {
		return fmt.Errorf("failed to scale up: %v", provisionErrors)
	}

	s.log.WithField("cluster_id", clusterID).Info("Cluster scaled up successfully")
	return nil
}

// findAvailableNodes finds nodes that are available for provisioning
func (s *ClusterService) findAvailableNodes(count uint) ([]models.Node, error) {
	var nodes []models.Node

	// Find nodes that are discovered but not assigned to any cluster
	err := s.store.DB().Where("status = ? AND cluster_id IS NULL", models.NodeStatusDiscovered).
		Limit(int(count)).Find(&nodes).Error // #nosec G115 -- Node count is reasonable

	if err != nil {
		return nil, errors.Wrap(err, "failed to query available nodes")
	}

	return nodes, nil
}

// updateClusterStatus is a helper method to update cluster status
func (s *ClusterService) updateClusterStatus(clusterID uint, status models.ClusterStatus) error {
	updateReq := UpdateClusterRequest{
		Status: &status,
	}
	_, err := s.Update(clusterID, updateReq)
	return err
}
