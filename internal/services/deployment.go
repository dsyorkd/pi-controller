package services

import (
	"context"

	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/dsyorkd/pi-controller/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

// DeploymentService is the service for managing deployments
type DeploymentService struct {
	store storage.Store
	k8s   k8s.K8sClient
}

// NewDeploymentService creates a new DeploymentService
func NewDeploymentService(store storage.Store, k8s k8s.K8sClient) *DeploymentService {
	return &DeploymentService{
		store: store,
		k8s:   k8s,
	}
}

// CreateDeployment creates a new deployment
func (s *DeploymentService) CreateDeployment(ctx context.Context, clusterID uint, pod *corev1.Pod) (*corev1.Pod, error) {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	return s.k8s.CreatePod(ctx, cluster.Name, pod)
}

// GetDeployment retrieves a deployment
func (s *DeploymentService) GetDeployment(ctx context.Context, clusterID uint, name string) (*corev1.Pod, error) {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	return s.k8s.GetPod(ctx, cluster.Name, name)
}

// DeleteDeployment deletes a deployment
func (s *DeploymentService) DeleteDeployment(ctx context.Context, clusterID uint, name string) error {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return err
	}

	return s.k8s.DeletePod(ctx, cluster.Name, name)
}
