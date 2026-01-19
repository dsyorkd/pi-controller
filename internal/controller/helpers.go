package controller

import (
	"context"
	"fmt"

	"github.com/dsyorkd/pi-controller/internal/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeFinder interface for listing nodes
type NodeFinder interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

// FindNodeBySelector finds a node that matches the given label selector
// This is a common helper to reduce duplication across GPIO, I2C, and PWM controllers
func FindNodeBySelector(
	ctx context.Context,
	finder NodeFinder,
	nodeSelector map[string]string,
	resourceType string,
	logger logger.Interface,
) (*corev1.Node, error) {
	// List all nodes
	var nodeList corev1.NodeList
	if err := finder.List(ctx, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Convert nodeSelector to label selector
	selector := labels.SelectorFromSet(nodeSelector)

	// Find matching nodes
	var matchingNodes []corev1.Node
	for _, node := range nodeList.Items {
		if selector.Matches(labels.Set(node.Labels)) {
			matchingNodes = append(matchingNodes, node)
		}
	}

	if len(matchingNodes) == 0 {
		return nil, nil // No matching nodes found
	}

	// Return the first matching node (could be enhanced with more sophisticated selection)
	logger.WithFields(map[string]interface{}{
		"selected_node":  matchingNodes[0].Name,
		"matching_count": len(matchingNodes),
		"resource_type":  resourceType,
	}).Info("Found matching node")

	return &matchingNodes[0], nil
}

// CheckNodeReady checks if a node is in Ready state
// This is a common helper to reduce duplication across GPIO, I2C, and PWM controllers
func CheckNodeReady(node *corev1.Node, logger logger.Interface) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				logger.WithField("node", node.Name).Info("Node is ready")
				return true
			}
			logger.WithField("node", node.Name).Warn("Node is not ready")
			return false
		}
	}

	logger.WithField("node", node.Name).Warn("Node ready status unknown")
	return false
}
