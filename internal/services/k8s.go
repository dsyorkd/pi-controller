package services

import (
	"context"

	"github.com/dsyorkd/pi-controller/internal/storage"
	"github.com/dsyorkd/pi-controller/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

// K8sService is the service for managing Kubernetes Services
type K8sService struct {
	store storage.Store
	k8s   k8s.K8sClient
}

// NewK8sService creates a new K8sService
func NewK8sService(store storage.Store, k8s k8s.K8sClient) *K8sService {
	return &K8sService{
		store: store,
		k8s:   k8s,
	}
}

// CreateService creates a new Kubernetes service
func (s *K8sService) CreateService(ctx context.Context, clusterID uint, service *corev1.Service) (*corev1.Service, error) {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	return s.k8s.CreateService(ctx, cluster.Name, service)
}

// GetService retrieves a Kubernetes service
func (s *K8sService) GetService(ctx context.Context, clusterID uint, namespace, name string) (*corev1.Service, error) {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	// Use cluster name as namespace if namespace not provided
	if namespace == "" {
		namespace = cluster.Name
	}

	return s.k8s.GetService(ctx, namespace, name)
}

// ListServices retrieves all Kubernetes services in a namespace
func (s *K8sService) ListServices(ctx context.Context, clusterID uint, namespace string) ([]corev1.Service, error) {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	// Use cluster name as namespace if namespace not provided
	if namespace == "" {
		namespace = cluster.Name
	}

	return s.k8s.ListServices(ctx, namespace)
}

// UpdateService updates a Kubernetes service
func (s *K8sService) UpdateService(ctx context.Context, clusterID uint, namespace string, service *corev1.Service) (*corev1.Service, error) {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	// Use cluster name as namespace if namespace not provided
	if namespace == "" {
		namespace = cluster.Name
	}

	return s.k8s.UpdateService(ctx, namespace, service)
}

// DeleteService deletes a Kubernetes service
func (s *K8sService) DeleteService(ctx context.Context, clusterID uint, namespace, name string) error {
	cluster, err := s.store.GetCluster(clusterID)
	if err != nil {
		return err
	}

	// Use cluster name as namespace if namespace not provided
	if namespace == "" {
		namespace = cluster.Name
	}

	return s.k8s.DeleteService(ctx, namespace, name)
}
