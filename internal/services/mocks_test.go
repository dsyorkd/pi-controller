package services

import (
	"context"
	"time"

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

// MockClusterService is a mock implementation of the ClusterService
type MockClusterService struct {
	mock.Mock
}

func (m *MockClusterService) Create(req CreateClusterRequest) (*models.Cluster, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) Update(id uint, req UpdateClusterRequest) (*models.Cluster, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockClusterService) List(opts ClusterListOptions) ([]models.Cluster, int64, error) {
	args := m.Called(opts)
	return args.Get(0).([]models.Cluster), args.Get(1).(int64), args.Error(2)
}

func (m *MockClusterService) GetByID(id uint) (*models.Cluster, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) GetByName(name string) (*models.Cluster, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Cluster), args.Error(1)
}

func (m *MockClusterService) GetNodes(clusterID uint) ([]models.Node, error) {
	args := m.Called(clusterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Node), args.Error(1)
}

func (m *MockClusterService) GetStatus(id uint) (models.ClusterStatus, error) {
	args := m.Called(id)
	return args.Get(0).(models.ClusterStatus), args.Error(1)
}

// MockNodeService is a mock implementation of the NodeService
type MockNodeService struct {
	mock.Mock
}

func (m *MockNodeService) List(opts NodeListOptions) ([]models.Node, int64, error) {
	args := m.Called(opts)
	return args.Get(0).([]models.Node), args.Get(1).(int64), args.Error(2)
}

func (m *MockNodeService) GetByID(id uint, includeGPIO bool) (*models.Node, error) {
	args := m.Called(id, includeGPIO)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Node), args.Error(1)
}

func (m *MockNodeService) GetByName(name string) (*models.Node, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Node), args.Error(1)
}

func (m *MockNodeService) GetByIPAddress(ipAddress string) (*models.Node, error) {
	args := m.Called(ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Node), args.Error(1)
}

func (m *MockNodeService) Create(req CreateNodeRequest) (*models.Node, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Node), args.Error(1)
}

func (m *MockNodeService) Update(id uint, req UpdateNodeRequest) (*models.Node, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Node), args.Error(1)
}

func (m *MockNodeService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockNodeService) UpdateLastSeen(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockNodeService) GetGPIODevices(nodeID uint) ([]models.GPIODevice, error) {
	args := m.Called(nodeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.GPIODevice), args.Error(1)
}

func (m *MockNodeService) Provision(id uint, clusterID uint) error {
	args := m.Called(id, clusterID)
	return args.Error(0)
}

func (m *MockNodeService) Deprovision(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockGPIOService is a mock implementation of the GPIOService
type MockGPIOService struct {
	mock.Mock
}

func (m *MockGPIOService) List(opts GPIOListOptions) ([]models.GPIODevice, int64, error) {
	args := m.Called(opts)
	return args.Get(0).([]models.GPIODevice), args.Get(1).(int64), args.Error(2)
}

func (m *MockGPIOService) GetByID(id uint) (*models.GPIODevice, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GPIODevice), args.Error(1)
}

func (m *MockGPIOService) GetByNodeAndPin(nodeID uint, pinNumber int) (*models.GPIODevice, error) {
	args := m.Called(nodeID, pinNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GPIODevice), args.Error(1)
}

func (m *MockGPIOService) Create(req CreateGPIODeviceRequest) (*models.GPIODevice, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GPIODevice), args.Error(1)
}

func (m *MockGPIOService) Update(id uint, req UpdateGPIODeviceRequest) (*models.GPIODevice, error) {
	args := m.Called(id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GPIODevice), args.Error(1)
}

func (m *MockGPIOService) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockGPIOService) Read(id uint) (*models.GPIODevice, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GPIODevice), args.Error(1)
}

func (m *MockGPIOService) Write(id uint, value int) error {
	args := m.Called(id, value)
	return args.Error(0)
}

func (m *MockGPIOService) WriteWithClient(id uint, value int, clientID string) error {
	args := m.Called(id, value, clientID)
	return args.Error(0)
}

func (m *MockGPIOService) GetReadings(filter GPIOReadingFilter) ([]models.GPIOReading, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.GPIOReading), args.Get(1).(int64), args.Error(2)
}

func (m *MockGPIOService) CleanupOldReadings(olderThan time.Duration) (int64, error) {
	args := m.Called(olderThan)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockGPIOService) ReservePin(id uint, req GPIOReservationRequest) error {
	args := m.Called(id, req)
	return args.Error(0)
}

func (m *MockGPIOService) ReleasePin(id uint, req GPIOReleaseRequest) error {
	args := m.Called(id, req)
	return args.Error(0)
}

func (m *MockGPIOService) GetReservations() ([]GPIOReservationInfo, error) {
	args := m.Called()
	return args.Get(0).([]GPIOReservationInfo), args.Error(1)
}

func (m *MockGPIOService) CleanupExpiredReservations() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockGPIOService) ConfigurePin(ctx context.Context, req *GPIORequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockGPIOService) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockDeploymentService is a mock implementation of the DeploymentService
type MockDeploymentService struct {
	mock.Mock
}

func (m *MockDeploymentService) CreateDeployment(ctx context.Context, clusterID uint, pod *corev1.Pod) (*corev1.Pod, error) {
	args := m.Called(ctx, clusterID, pod)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockDeploymentService) GetDeployment(ctx context.Context, clusterID uint, name string) (*corev1.Pod, error) {
	args := m.Called(ctx, clusterID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*corev1.Pod), args.Error(1)
}

func (m *MockDeploymentService) DeleteDeployment(ctx context.Context, clusterID uint, name string) error {
	args := m.Called(ctx, clusterID, name)
	return args.Error(0)
}
