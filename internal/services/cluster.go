package services

import (
	"gorm.io/gorm"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/storage"
)

// ClusterService is the service for managing clusters
type ClusterService struct {
	store *storage.Database
	log   logger.Interface
}

// NewClusterService creates a new ClusterService
func NewClusterService(store *storage.Database, log logger.Interface) *ClusterService {
	return &ClusterService{
		store: store,
		log:   log,
	}
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

	// Check for duplicate name
	existing, err := s.GetByName(req.Name)
	if err != nil && err.Error() != "not found" {
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
		cluster.Name = *req.Name
	}
	if req.Description != nil {
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
