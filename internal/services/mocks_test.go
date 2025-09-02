package services

import (
	"context"

	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
)

// MockStore is a mock implementation of the Store interface
type MockStore struct {
	mock.Mock
}

func (m *MockStore) CreateCluster(cluster *models.Cluster) error {
	args := m.Called(cluster)
	return args.Error(0)
}

func (m *MockStore) GetCluster(id uint) (*models.Cluster, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockStore) GetClusters() ([]models.Cluster, error) {
	args := m.Called()
	return args.Get(0).([]models.Cluster), args.Error(1)
}

func (m *MockStore) UpdateCluster(cluster *models.Cluster) error {
	args := m.Called(cluster)
	return args.Error(0)
}

func (m *MockStore) DeleteCluster(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStore) GetNodesByClusterID(clusterID uint) ([]models.Node, error) {
	args := m.Called(clusterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Node), args.Error(1)
}

func (m *MockStore) CreatePin(pin *models.GPIODevice) error {
	args := m.Called(pin)
	return args.Error(0)
}

func (m *MockStore) GetPin(id uint) (*models.GPIODevice, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GPIODevice), args.Error(1)
}

func (m *MockStore) GetPins() ([]models.GPIODevice, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.GPIODevice), args.Error(1)
}

func (m *MockStore) UpdatePin(pin *models.GPIODevice) error {
	args := m.Called(pin)
	return args.Error(0)
}

func (m *MockStore) DeletePin(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockK8sClient is a mock implementation of the K8sClient interface
type MockK8sClient struct {
	mock.Mock
}

func (m *MockK8sClient) CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	args := m.Called(ctx, namespace, pod)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockK8sClient) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	args := m.Called(ctx, namespace, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockK8sClient) UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	args := m.Called(ctx, namespace, pod)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockK8sClient) DeletePod(ctx context.Context, namespace, name string) error {
	args := m.Called(ctx, namespace, name)
	return args.Error(0)
}
