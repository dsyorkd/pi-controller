package services

import (
	"context"
	"errors"
	"testing"

	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentService_CreateDeployment(t *testing.T) {
	t.Run("should create a deployment", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		cluster := &models.Cluster{
			ID:   1,
			Name: "test-cluster",
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}

		store.On("GetCluster", uint(1)).Return(cluster, nil)
		k8sClient.On("CreatePod", mock.Anything, "test-cluster", pod).Return(pod, nil)

		createdPod, err := service.CreateDeployment(context.Background(), 1, pod)
		assert.NoError(t, err)
		assert.NotNil(t, createdPod)
		assert.Equal(t, "test-pod", createdPod.Name)

		store.AssertExpectations(t)
		k8sClient.AssertExpectations(t)
	})

	t.Run("should return an error when the store fails", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}

		store.On("GetCluster", uint(1)).Return(nil, errors.New("store error"))

		_, err := service.CreateDeployment(context.Background(), 1, pod)
		assert.Error(t, err)

		store.AssertExpectations(t)
	})

	t.Run("should return an error when the k8s client fails", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		cluster := &models.Cluster{
			ID:   1,
			Name: "test-cluster",
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}

		store.On("GetCluster", uint(1)).Return(cluster, nil)
		k8sClient.On("CreatePod", mock.Anything, "test-cluster", pod).Return(nil, errors.New("k8s error"))

		_, err := service.CreateDeployment(context.Background(), 1, pod)
		assert.Error(t, err)

		store.AssertExpectations(t)
		k8sClient.AssertExpectations(t)
	})
}

func TestDeploymentService_GetDeployment(t *testing.T) {
	t.Run("should get a deployment", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		cluster := &models.Cluster{
			ID:   1,
			Name: "test-cluster",
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}

		store.On("GetCluster", uint(1)).Return(cluster, nil)
		k8sClient.On("GetPod", mock.Anything, "test-cluster", "test-pod").Return(pod, nil)

		retrievedPod, err := service.GetDeployment(context.Background(), 1, "test-pod")
		assert.NoError(t, err)
		assert.NotNil(t, retrievedPod)
		assert.Equal(t, "test-pod", retrievedPod.Name)

		store.AssertExpectations(t)
		k8sClient.AssertExpectations(t)
	})

	t.Run("should return an error when the store fails", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		store.On("GetCluster", uint(1)).Return(nil, errors.New("store error"))

		_, err := service.GetDeployment(context.Background(), 1, "test-pod")
		assert.Error(t, err)

		store.AssertExpectations(t)
	})

	t.Run("should return an error when the k8s client fails", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		cluster := &models.Cluster{
			ID:   1,
			Name: "test-cluster",
		}

		store.On("GetCluster", uint(1)).Return(cluster, nil)
		k8sClient.On("GetPod", mock.Anything, "test-cluster", "test-pod").Return(nil, errors.New("k8s error"))

		_, err := service.GetDeployment(context.Background(), 1, "test-pod")
		assert.Error(t, err)

		store.AssertExpectations(t)
		k8sClient.AssertExpectations(t)
	})
}

func TestDeploymentService_DeleteDeployment(t *testing.T) {
	t.Run("should delete a deployment", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		cluster := &models.Cluster{
			ID:   1,
			Name: "test-cluster",
		}

		store.On("GetCluster", uint(1)).Return(cluster, nil)
		k8sClient.On("DeletePod", mock.Anything, "test-cluster", "test-pod").Return(nil)

		err := service.DeleteDeployment(context.Background(), 1, "test-pod")
		assert.NoError(t, err)

		store.AssertExpectations(t)
		k8sClient.AssertExpectations(t)
	})

	t.Run("should return an error when the store fails", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		store.On("GetCluster", uint(1)).Return(nil, errors.New("store error"))

		err := service.DeleteDeployment(context.Background(), 1, "test-pod")
		assert.Error(t, err)

		store.AssertExpectations(t)
	})

	t.Run("should return an error when the k8s client fails", func(t *testing.T) {
		store := &MockStore{}
		k8sClient := &MockK8sClient{}
		service := NewDeploymentService(store, k8sClient)

		cluster := &models.Cluster{
			ID:   1,
			Name: "test-cluster",
		}

		store.On("GetCluster", uint(1)).Return(cluster, nil)
		k8sClient.On("DeletePod", mock.Anything, "test-cluster", "test-pod").Return(errors.New("k8s error"))

		err := service.DeleteDeployment(context.Background(), 1, "test-pod")
		assert.Error(t, err)

		store.AssertExpectations(t)
		k8sClient.AssertExpectations(t)
	})
}