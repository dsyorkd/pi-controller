package k8s

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestNewClient(t *testing.T) {
	t.Run("should create a new out-of-cluster client", func(t *testing.T) {
		// Create a dummy kubeconfig file
		kubeconfig, err := os.CreateTemp("", "kubeconfig")
		assert.NoError(t, err)
		defer os.Remove(kubeconfig.Name())

		// Create a dummy client config
		config := api.Config{
			Clusters: map[string]*api.Cluster{
				"default": {
					Server: "http://localhost:8080",
				},
			},
			Contexts: map[string]*api.Context{
				"default": {
					Cluster: "default",
				},
			},
			CurrentContext: "default",
		}

		// Write the dummy client config to the kubeconfig file
		err = clientcmd.WriteToFile(config, kubeconfig.Name())
		assert.NoError(t, err)

		// Create a new client
		logger := logrus.New()
		client, err := NewClient(&Config{ConfigPath: kubeconfig.Name()}, logger)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("should create a pod", func(t *testing.T) {
		// Create a fake clientset
		clientset := fake.NewSimpleClientset()

		// Create a new client
		logger := logrus.New()
		client := &Client{
			clientset: clientset,
			logger:    logger.WithField("component", "k8s-client"),
			namespace: "default",
		}

		// Create a pod
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}
		createdPod, err := client.CreatePod(context.Background(), "default", pod)
		assert.NoError(t, err)
		assert.NotNil(t, createdPod)
		assert.Equal(t, "test-pod", createdPod.Name)
	})

	t.Run("should get a pod", func(t *testing.T) {
		// Create a fake clientset
		clientset := fake.NewSimpleClientset(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		})

		// Create a new client
		logger := logrus.New()
		client := &Client{
			clientset: clientset,
			logger:    logger.WithField("component", "k8s-client"),
			namespace: "default",
		}

		// Get a pod that exists
		pod, err := client.GetPod(context.Background(), "default", "test-pod")
		assert.NoError(t, err)
		assert.NotNil(t, pod)
		assert.Equal(t, "test-pod", pod.Name)

		// Get a pod that does not exist
		_, err = client.GetPod(context.Background(), "default", "non-existent-pod")
		assert.Error(t, err)
	})

	t.Run("should update a pod", func(t *testing.T) {
		// Create a fake clientset
		clientset := fake.NewSimpleClientset(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
		})

		// Create a new client
		logger := logrus.New()
		client := &Client{
			clientset: clientset,
			logger:    logger.WithField("component", "k8s-client"),
			namespace: "default",
		}

		// Get the pod
		pod, err := client.GetPod(context.Background(), "default", "test-pod")
		assert.NoError(t, err)
		assert.NotNil(t, pod)

		// Update the pod
		pod.Labels["foo"] = "baz"
		updatedPod, err := client.UpdatePod(context.Background(), "default", pod)
		assert.NoError(t, err)
		assert.NotNil(t, updatedPod)
		assert.Equal(t, "baz", updatedPod.Labels["foo"])
	})

	t.Run("should delete a pod", func(t *testing.T) {
		// Create a fake clientset
		clientset := fake.NewSimpleClientset(&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		})

		// Create a new client
		logger := logrus.New()
		client := &Client{
			clientset: clientset,
			logger:    logger.WithField("component", "k8s-client"),
			namespace: "default",
		}

		// Delete the pod
		err := client.DeletePod(context.Background(), "default", "test-pod")
		assert.NoError(t, err)

		// Verify the pod was deleted
		_, err = client.GetPod(context.Background(), "default", "test-pod")
		assert.Error(t, err)
	})
}
