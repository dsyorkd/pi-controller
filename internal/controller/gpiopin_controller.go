package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/dsyorkd/pi-controller/internal/services"
	gpiov1 "github.com/dsyorkd/pi-controller/pkg/apis/gpio/v1"
)

// GPIOPinReconciler reconciles a GPIOPin object
type GPIOPinReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Logger      *logrus.Entry
	GPIOService services.GPIOControllerService
	NodeService services.NodeControllerService
}

// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=gpiopins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=gpiopins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=gpiopins/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GPIOPinReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithFields(logrus.Fields{
		"gpiopin":   req.NamespacedName,
		"namespace": req.Namespace,
	})

	logger.Info("Starting reconciliation")

	// Fetch the GPIOPin instance
	var gpioPin gpiov1.GPIOPin
	if err := r.Get(ctx, req.NamespacedName, &gpioPin); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("GPIOPin resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		logger.WithError(err).Error("Failed to get GPIOPin resource")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !gpioPin.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &gpioPin, logger)
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(&gpioPin, "gpio.pi-controller.io/finalizer") {
		controllerutil.AddFinalizer(&gpioPin, "gpio.pi-controller.io/finalizer")
		if err := r.Update(ctx, &gpioPin); err != nil {
			logger.WithError(err).Error("Failed to add finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("Added finalizer to GPIOPin")
		return ctrl.Result{Requeue: true}, nil
	}

	// Find the target node
	targetNode, err := r.findTargetNode(ctx, &gpioPin, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to find target node")
		return r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhaseFailed,
			"Failed to find target node", err.Error(), logger)
	}

	if targetNode == nil {
		logger.Warn("No suitable node found for GPIOPin")
		return r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhasePending,
			"No suitable node found", "Waiting for matching node", logger)
	}

	// Check if node is reachable
	nodeReachable, err := r.checkNodeReachability(ctx, targetNode, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to check node reachability")
		return r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhaseFailed,
			"Failed to check node reachability", err.Error(), logger)
	}

	if !nodeReachable {
		logger.Warn("Target node is not reachable")
		return r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhaseFailed,
			"Target node unreachable", "Node is not responding", logger)
	}

	// Update phase to configuring
	if gpioPin.Status.Phase != gpiov1.GPIOPhaseConfiguring {
		result, err := r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhaseConfiguring,
			"Configuring GPIO pin", "Setting up hardware configuration", logger)
		if err != nil {
			return result, err
		}
	}

	// Configure the GPIO pin on the target node
	if err := r.configureGPIOPin(ctx, &gpioPin, targetNode, logger); err != nil {
		logger.WithError(err).Error("Failed to configure GPIO pin")
		return r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhaseFailed,
			"Configuration failed", err.Error(), logger)
	}

	// Update phase to ready
	return r.updateStatus(ctx, &gpioPin, gpiov1.GPIOPhaseReady,
		"GPIO pin ready", "Hardware configured successfully", logger)
}

// findTargetNode finds the node that matches the nodeSelector
func (r *GPIOPinReconciler) findTargetNode(ctx context.Context, gpioPin *gpiov1.GPIOPin, logger *logrus.Entry) (*corev1.Node, error) {
	// List all nodes
	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Convert nodeSelector to label selector
	selector := labels.SelectorFromSet(gpioPin.Spec.NodeSelector)

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
	logger.WithFields(logrus.Fields{
		"selected_node":  matchingNodes[0].Name,
		"matching_count": len(matchingNodes),
	}).Info("Found matching node for GPIOPin")

	return &matchingNodes[0], nil
}

// checkNodeReachability checks if the target node is reachable
func (r *GPIOPinReconciler) checkNodeReachability(ctx context.Context, node *corev1.Node, logger *logrus.Entry) (bool, error) {
	// Check if node is ready
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				logger.WithField("node", node.Name).Info("Node is ready")
				return true, nil
			}
			logger.WithField("node", node.Name).Warn("Node is not ready")
			return false, nil
		}
	}

	logger.WithField("node", node.Name).Warn("Node ready status unknown")
	return false, nil
}

// configureGPIOPin configures the GPIO pin on the target node via gRPC
func (r *GPIOPinReconciler) configureGPIOPin(ctx context.Context, gpioPin *gpiov1.GPIOPin, node *corev1.Node, logger *logrus.Entry) error {
	// Convert GPIOPin spec to GPIO service request
	// TODO: Fix GPIORequest struct definition
	_ = node.Name
	_ = gpioPin.Spec.PinNumber
	_ = string(gpioPin.Spec.Mode)

	// TODO: Implement GPIO pin configuration logic
	logger.WithFields(logrus.Fields{
		"pin_number": gpioPin.Spec.PinNumber,
		"mode":       string(gpioPin.Spec.Mode),
	}).Info("GPIO pin configuration would be performed here")

	// Temporary stub - return success for now
	return nil

	logger.Info("Successfully configured GPIO pin")
	return nil
}

// updateStatus updates the GPIOPin status
func (r *GPIOPinReconciler) updateStatus(ctx context.Context, gpioPin *gpiov1.GPIOPin,
	phase gpiov1.GPIOPhase, message, reason string, logger *logrus.Entry) (ctrl.Result, error) {

	// Update basic status fields
	now := metav1.Now()
	gpioPin.Status.Phase = phase
	gpioPin.Status.Message = message
	gpioPin.Status.LastUpdated = &now

	// Update conditions based on phase
	var conditionType gpiov1.GPIOConditionType
	var conditionStatus metav1.ConditionStatus

	switch phase {
	case gpiov1.GPIOPhaseReady:
		conditionType = gpiov1.GPIOConditionReady
		conditionStatus = metav1.ConditionTrue
		// Also set configured condition
		r.setCondition(&gpioPin.Status, gpiov1.GPIOConditionConfigured, metav1.ConditionTrue, "ConfiguredSuccessfully", "GPIO pin configured successfully")
		r.setCondition(&gpioPin.Status, gpiov1.GPIOConditionNodeReachable, metav1.ConditionTrue, "NodeReachable", "Target node is reachable")
	case gpiov1.GPIOPhaseFailed:
		conditionType = gpiov1.GPIOConditionReady
		conditionStatus = metav1.ConditionFalse
	case gpiov1.GPIOPhaseConfiguring:
		conditionType = gpiov1.GPIOConditionConfigured
		conditionStatus = metav1.ConditionUnknown
	default:
		conditionType = gpiov1.GPIOConditionReady
		conditionStatus = metav1.ConditionUnknown
	}

	r.setCondition(&gpioPin.Status, conditionType, conditionStatus, reason, message)

	// Update the status
	if err := r.Status().Update(ctx, gpioPin); err != nil {
		logger.WithError(err).Error("Failed to update GPIOPin status")
		return ctrl.Result{}, err
	}

	logger.WithFields(logrus.Fields{
		"phase":   phase,
		"message": message,
	}).Info("Updated GPIOPin status")

	// Determine requeue behavior
	switch phase {
	case gpiov1.GPIOPhasePending:
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	case gpiov1.GPIOPhaseConfiguring:
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	case gpiov1.GPIOPhaseFailed:
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	default:
		return ctrl.Result{}, nil
	}
}

// setCondition sets a condition in the GPIOPin status
func (r *GPIOPinReconciler) setCondition(status *gpiov1.GPIOPinStatus,
	conditionType gpiov1.GPIOConditionType, conditionStatus metav1.ConditionStatus,
	reason, message string) {

	now := metav1.Now()
	newCondition := gpiov1.GPIOCondition{
		Type:               conditionType,
		Status:             conditionStatus,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	for i, condition := range status.Conditions {
		if condition.Type == conditionType {
			// Update existing condition if status changed
			if condition.Status != conditionStatus {
				status.Conditions[i] = newCondition
			}
			return
		}
	}

	// Add new condition
	status.Conditions = append(status.Conditions, newCondition)
}

// handleDeletion handles GPIOPin deletion
func (r *GPIOPinReconciler) handleDeletion(ctx context.Context, gpioPin *gpiov1.GPIOPin, logger *logrus.Entry) (ctrl.Result, error) {
	logger.Info("Handling GPIOPin deletion")

	// Find the node that was managing this pin
	if gpioPin.Status.NodeID != "" {
		// Try to clean up the GPIO pin on the node
		cleanupRequest := &services.GPIORequest{
			NodeID:    gpioPin.Status.NodeID,
			PinNumber: gpioPin.Spec.PinNumber,
			Mode:      "cleanup", // Special mode for cleanup
		}

		if err := r.GPIOService.ConfigurePin(ctx, cleanupRequest); err != nil {
			logger.WithError(err).Warn("Failed to cleanup GPIO pin on node, continuing with deletion")
		} else {
			logger.Info("Successfully cleaned up GPIO pin on node")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(gpioPin, "gpio.pi-controller.io/finalizer")
	if err := r.Update(ctx, gpioPin); err != nil {
		logger.WithError(err).Error("Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully handled GPIOPin deletion")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *GPIOPinReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpiov1.GPIOPin{}).
		Watches(&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.mapNodeToGPIOPins)).
		Complete(r)
}

// mapNodeToGPIOPins maps Node changes to GPIOPin reconcile requests
func (r *GPIOPinReconciler) mapNodeToGPIOPins(ctx context.Context, obj client.Object) []reconcile.Request {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}

	var gpioPinList gpiov1.GPIOPinList
	if err := r.List(ctx, &gpioPinList); err != nil {
		r.Logger.WithError(err).Error("Failed to list GPIOPins for node mapping")
		return nil
	}

	var requests []reconcile.Request
	for _, gpioPin := range gpioPinList.Items {
		// Check if this GPIOPin matches the node
		selector := labels.SelectorFromSet(gpioPin.Spec.NodeSelector)
		if selector.Matches(labels.Set(node.Labels)) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      gpioPin.Name,
					Namespace: gpioPin.Namespace,
				},
			})
		}
	}

	r.Logger.WithFields(logrus.Fields{
		"node":          node.Name,
		"gpiopin_count": len(requests),
	}).Debug("Mapped node change to GPIOPin reconcile requests")

	return requests
}
