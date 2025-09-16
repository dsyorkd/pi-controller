package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// K8sClient is the interface for the Kubernetes client
type K8sClient interface {
	// Pod operations
	CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error

	// Service operations
	CreateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error)
	GetService(ctx context.Context, namespace, name string) (*corev1.Service, error)
	ListServices(ctx context.Context, namespace string) ([]corev1.Service, error)
	UpdateService(ctx context.Context, namespace string, service *corev1.Service) (*corev1.Service, error)
	DeleteService(ctx context.Context, namespace, name string) error
}
