package defaults

// Kubernetes defaults
const (
	// KubernetesEnabled indicates whether Kubernetes integration is enabled
	KubernetesEnabled = false

	// KubernetesInCluster indicates whether running in-cluster
	KubernetesInCluster = false

	// KubernetesNamespace is the default namespace
	KubernetesNamespace = "default"

	// KubernetesResyncInterval is the default resync interval
	KubernetesResyncInterval = "30s"
)
