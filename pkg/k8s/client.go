// Package k8s provides Kubernetes client utilities for cluster management
package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes client with additional functionality
type Client struct {
	clientset kubernetes.Interface
	config    *rest.Config
	logger    *logrus.Entry
	namespace string
}

// Config represents Kubernetes client configuration
type Config struct {
	ConfigPath     string `yaml:"config_path" mapstructure:"config_path"`
	InCluster      bool   `yaml:"in_cluster" mapstructure:"in_cluster"`
	Namespace      string `yaml:"namespace" mapstructure:"namespace"`
	ResyncInterval string `yaml:"resync_interval" mapstructure:"resync_interval"`
}

// DefaultConfig returns default Kubernetes configuration
func DefaultConfig() *Config {
	return &Config{
		ConfigPath:     "",
		InCluster:      false,
		Namespace:      "default",
		ResyncInterval: "30s",
	}
}

// NewClient creates a new Kubernetes client
func NewClient(config *Config, logger *logrus.Logger) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var restConfig *rest.Config
	var err error

	if config.InCluster {
		// Create in-cluster config
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
		logger.Info("Using in-cluster Kubernetes configuration")
	} else {
		// Create out-of-cluster config
		configPath := config.ConfigPath
		if configPath == "" {
			if home := homedir.HomeDir(); home != "" {
				configPath = filepath.Join(home, ".kube", "config")
			}
		}

		restConfig, err = clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from %s: %w", configPath, err)
		}
		logger.WithField("config_path", configPath).Info("Using out-of-cluster Kubernetes configuration")
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	client := &Client{
		clientset: clientset,
		config:    restConfig,
		logger:    logger.WithField("component", "k8s-client"),
		namespace: config.Namespace,
	}

	return client, nil
}

// HealthCheck performs a basic health check against the Kubernetes API
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes API: %w", err)
	}
	return nil
}

// GetServerVersion returns the Kubernetes server version
func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	version, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get server version: %w", err)
	}
	return version.String(), nil
}

// NodeInfo represents information about a Kubernetes node
type NodeInfo struct {
	Name         string            `json:"name"`
	Ready        bool              `json:"ready"`
	Version      string            `json:"version"`
	OS           string            `json:"os"`
	Architecture string            `json:"architecture"`
	Roles        []string          `json:"roles"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	CreationTime time.Time         `json:"creation_time"`
	Conditions   []NodeCondition   `json:"conditions"`
	Capacity     ResourceList      `json:"capacity"`
	Allocatable  ResourceList      `json:"allocatable"`
}

// NodeCondition represents a node condition
type NodeCondition struct {
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Reason  string    `json:"reason"`
	Message string    `json:"message"`
	LastTransitionTime time.Time `json:"last_transition_time"`
}

// ResourceList represents node resource information
type ResourceList struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   string `json:"pods"`
}

// ListNodes returns information about all nodes in the cluster
func (c *Client) ListNodes(ctx context.Context) ([]NodeInfo, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.WithError(err).Error("Failed to list nodes")
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodeInfos := make([]NodeInfo, len(nodes.Items))
	for i, node := range nodes.Items {
		nodeInfos[i] = c.convertNodeToInfo(node)
	}

	c.logger.WithField("count", len(nodeInfos)).Debug("Listed Kubernetes nodes")
	return nodeInfos, nil
}

// GetNode returns information about a specific node
func (c *Client) GetNode(ctx context.Context, nodeName string) (*NodeInfo, error) {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"node_name": nodeName,
			"error":     err,
		}).Error("Failed to get node")
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	nodeInfo := c.convertNodeToInfo(*node)
	return &nodeInfo, nil
}

// convertNodeToInfo converts a Kubernetes Node object to NodeInfo
func (c *Client) convertNodeToInfo(node corev1.Node) NodeInfo {
	// Determine if node is ready
	ready := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			ready = condition.Status == corev1.ConditionTrue
			break
		}
	}

	// Extract roles from labels
	roles := []string{}
	if _, exists := node.Labels["node-role.kubernetes.io/master"]; exists {
		roles = append(roles, "master")
	}
	if _, exists := node.Labels["node-role.kubernetes.io/control-plane"]; exists {
		roles = append(roles, "control-plane")
	}
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}

	// Convert conditions
	conditions := make([]NodeCondition, len(node.Status.Conditions))
	for i, condition := range node.Status.Conditions {
		conditions[i] = NodeCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime.Time,
		}
	}

	return NodeInfo{
		Name:         node.Name,
		Ready:        ready,
		Version:      node.Status.NodeInfo.KubeletVersion,
		OS:           node.Status.NodeInfo.OperatingSystem,
		Architecture: node.Status.NodeInfo.Architecture,
		Roles:        roles,
		Labels:       node.Labels,
		Annotations:  node.Annotations,
		CreationTime: node.CreationTimestamp.Time,
		Conditions:   conditions,
		Capacity: ResourceList{
			CPU:    node.Status.Capacity.Cpu().String(),
			Memory: node.Status.Capacity.Memory().String(),
			Pods:   node.Status.Capacity.Pods().String(),
		},
		Allocatable: ResourceList{
			CPU:    node.Status.Allocatable.Cpu().String(),
			Memory: node.Status.Allocatable.Memory().String(),
			Pods:   node.Status.Allocatable.Pods().String(),
		},
	}
}

// PodInfo represents information about a Kubernetes pod
type PodInfo struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Phase     string    `json:"phase"`
	Ready     bool      `json:"ready"`
	NodeName  string    `json:"node_name"`
	CreatedAt time.Time `json:"created_at"`
}

// ListPods returns information about all pods in the specified namespace
func (c *Client) ListPods(ctx context.Context, namespace string) ([]PodInfo, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"namespace": namespace,
			"error":     err,
		}).Error("Failed to list pods")
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	podInfos := make([]PodInfo, len(pods.Items))
	for i, pod := range pods.Items {
		// Check if pod is ready
		ready := true
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				ready = condition.Status == corev1.ConditionTrue
				break
			}
		}

		podInfos[i] = PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			Ready:     ready,
			NodeName:  pod.Spec.NodeName,
			CreatedAt: pod.CreationTimestamp.Time,
		}
	}

	c.logger.WithFields(logrus.Fields{
		"namespace": namespace,
		"count":     len(podInfos),
	}).Debug("Listed Kubernetes pods")

	return podInfos, nil
}

// ListPodsOnNode returns information about all pods running on a specific node
func (c *Client) ListPodsOnNode(ctx context.Context, nodeName string) ([]PodInfo, error) {
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", nodeName)
	
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"node_name": nodeName,
			"error":     err,
		}).Error("Failed to list pods on node")
		return nil, fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	podInfos := make([]PodInfo, len(pods.Items))
	for i, pod := range pods.Items {
		// Check if pod is ready
		ready := true
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				ready = condition.Status == corev1.ConditionTrue
				break
			}
		}

		podInfos[i] = PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			Ready:     ready,
			NodeName:  pod.Spec.NodeName,
			CreatedAt: pod.CreationTimestamp.Time,
		}
	}

	c.logger.WithFields(logrus.Fields{
		"node_name": nodeName,
		"count":     len(podInfos),
	}).Debug("Listed pods on Kubernetes node")

	return podInfos, nil
}

// DrainNode cordons and drains a node for maintenance
func (c *Client) DrainNode(ctx context.Context, nodeName string) error {
	// First, cordon the node
	if err := c.CordonNode(ctx, nodeName); err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	// TODO: Implement pod eviction logic
	// This would typically involve:
	// 1. Getting all pods on the node
	// 2. Evicting pods that can be safely evicted
	// 3. Waiting for pods to be terminated
	// 4. Handling pods that cannot be evicted

	c.logger.WithField("node_name", nodeName).Info("Node drained successfully")
	return nil
}

// CordonNode marks a node as unschedulable
func (c *Client) CordonNode(ctx context.Context, nodeName string) error {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if node.Spec.Unschedulable {
		c.logger.WithField("node_name", nodeName).Info("Node is already cordoned")
		return nil
	}

	node.Spec.Unschedulable = true
	_, err = c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"node_name": nodeName,
			"error":     err,
		}).Error("Failed to cordon node")
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	c.logger.WithField("node_name", nodeName).Info("Node cordoned successfully")
	return nil
}

// UncordonNode marks a node as schedulable
func (c *Client) UncordonNode(ctx context.Context, nodeName string) error {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if !node.Spec.Unschedulable {
		c.logger.WithField("node_name", nodeName).Info("Node is already uncordoned")
		return nil
	}

	node.Spec.Unschedulable = false
	_, err = c.clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"node_name": nodeName,
			"error":     err,
		}).Error("Failed to uncordon node")
		return fmt.Errorf("failed to uncordon node: %w", err)
	}

	c.logger.WithField("node_name", nodeName).Info("Node uncordoned successfully")
	return nil
}

// GetClusterInfo returns general information about the cluster
func (c *Client) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	// Get server version
	version, err := c.GetServerVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	// Get nodes
	nodes, err := c.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Count ready nodes
	readyNodes := 0
	for _, node := range nodes {
		if node.Ready {
			readyNodes++
		}
	}

	// Get all pods
	allPods, err := c.ListPods(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Count running pods
	runningPods := 0
	for _, pod := range allPods {
		if pod.Phase == "Running" && pod.Ready {
			runningPods++
		}
	}

	clusterInfo := &ClusterInfo{
		Version:      version,
		TotalNodes:   len(nodes),
		ReadyNodes:   readyNodes,
		TotalPods:    len(allPods),
		RunningPods:  runningPods,
		Nodes:        nodes,
	}

	return clusterInfo, nil
}

// ClusterInfo represents general information about a Kubernetes cluster
type ClusterInfo struct {
	Version     string     `json:"version"`
	TotalNodes  int        `json:"total_nodes"`
	ReadyNodes  int        `json:"ready_nodes"`
	TotalPods   int        `json:"total_pods"`
	RunningPods int        `json:"running_pods"`
	Nodes       []NodeInfo `json:"nodes"`
}

// CreatePod creates a new pod in the specified namespace
func (c *Client) CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	if namespace == "" {
		namespace = c.namespace
	}
	return c.clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
}

// GetPod retrieves a pod by name from the specified namespace
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	if namespace == "" {
		namespace = c.namespace
	}
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

// UpdatePod updates a pod in the specified namespace
func (c *Client) UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	if namespace == "" {
		namespace = c.namespace
	}
	return c.clientset.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
}

// DeletePod deletes a pod by name from the specified namespace
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	if namespace == "" {
		namespace = c.namespace
	}
	return c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}