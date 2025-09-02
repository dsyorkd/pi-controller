package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// K8sClient is the interface for the Kubernetes client
type K8sClient interface {
	CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string) error
}
