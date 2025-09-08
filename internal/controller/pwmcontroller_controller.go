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
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/dsyorkd/pi-controller/internal/services"
	gpiov1 "github.com/dsyorkd/pi-controller/pkg/apis/gpio/v1"
)

// PWMControllerReconciler reconciles a PWMController object
type PWMControllerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Logger      *logrus.Entry
	PWMService  services.PWMControllerService
	NodeService services.NodeControllerService
}

// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=pwmcontrollers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=pwmcontrollers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=pwmcontrollers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *PWMControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithFields(logrus.Fields{
		"pwmcontroller": req.NamespacedName,
		"namespace":     req.Namespace,
	})

	logger.Info("Starting PWMController reconciliation")

	// Fetch the PWMController instance
	var pwmController gpiov1.PWMController
	if err := r.Get(ctx, req.NamespacedName, &pwmController); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PWMController resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		logger.WithError(err).Error("Failed to get PWMController resource")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !pwmController.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &pwmController, logger)
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(&pwmController, "gpio.pi-controller.io/finalizer") {
		controllerutil.AddFinalizer(&pwmController, "gpio.pi-controller.io/finalizer")
		if err := r.Update(ctx, &pwmController); err != nil {
			logger.WithError(err).Error("Failed to add finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("Added finalizer to PWMController")
		return ctrl.Result{Requeue: true}, nil
	}

	// Find the target node
	targetNode, err := r.findTargetNode(ctx, &pwmController, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to find target node")
		return r.updateStatus(ctx, &pwmController, gpiov1.PWMPhaseFailed,
			"Failed to find target node", err.Error(), logger)
	}

	if targetNode == nil {
		logger.Warn("No suitable node found for PWMController")
		return r.updateStatus(ctx, &pwmController, gpiov1.PWMPhasePending,
			"No suitable node found", "Waiting for matching node", logger)
	}

	// Check if node is reachable
	nodeReachable, err := r.checkNodeReachability(ctx, targetNode, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to check node reachability")
		return r.updateStatus(ctx, &pwmController, gpiov1.PWMPhaseFailed,
			"Failed to check node reachability", err.Error(), logger)
	}

	if !nodeReachable {
		logger.Warn("Target node is not reachable")
		return r.updateStatus(ctx, &pwmController, gpiov1.PWMPhaseFailed,
			"Target node unreachable", "Node is not responding", logger)
	}

	// Update phase to configuring
	if pwmController.Status.Phase != gpiov1.PWMPhaseConfiguring {
		result, err := r.updateStatus(ctx, &pwmController, gpiov1.PWMPhaseConfiguring,
			"Configuring PWM controller", "Setting up hardware configuration", logger)
		if err != nil {
			return result, err
		}
	}

	// Configure the PWM controller on the target node
	if err := r.configurePWMController(ctx, &pwmController, targetNode, logger); err != nil {
		logger.WithError(err).Error("Failed to configure PWM controller")
		return r.updateStatus(ctx, &pwmController, gpiov1.PWMPhaseFailed,
			"Configuration failed", err.Error(), logger)
	}

	// Update phase to ready
	return r.updateStatus(ctx, &pwmController, gpiov1.PWMPhaseReady,
		"PWM controller ready", "Hardware configured successfully", logger)
}

// findTargetNode finds the node that matches the nodeSelector
func (r *PWMControllerReconciler) findTargetNode(ctx context.Context, pwmController *gpiov1.PWMController, logger *logrus.Entry) (*corev1.Node, error) {
	// List all nodes
	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Convert nodeSelector to label selector
	selector := labels.SelectorFromSet(pwmController.Spec.NodeSelector)

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

	logger.WithFields(logrus.Fields{
		"selected_node":  matchingNodes[0].Name,
		"matching_count": len(matchingNodes),
	}).Info("Found matching node for PWMController")

	return &matchingNodes[0], nil
}

// checkNodeReachability checks if the target node is reachable
func (r *PWMControllerReconciler) checkNodeReachability(ctx context.Context, node *corev1.Node, logger *logrus.Entry) (bool, error) {
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

// configurePWMController configures the PWM controller on the target node via gRPC
func (r *PWMControllerReconciler) configurePWMController(ctx context.Context, pwmController *gpiov1.PWMController, node *corev1.Node, logger *logrus.Entry) error {
	// Convert PWMController spec to PWM service request
	request := &services.PWMRequest{
		NodeID:        node.Name,
		Address:       pwmController.Spec.Address,
		BaseFrequency: pwmController.Spec.BaseFrequency,
		ChannelCount:  pwmController.Spec.ChannelCount,
		OutputEnable:  pwmController.Spec.OutputEnable,
		InvertOutput:  pwmController.Spec.InvertOutput,
		ExternalClock: pwmController.Spec.ExternalClock,
	}

	// Convert channel configurations
	for _, channel := range pwmController.Spec.Channels {
		request.Channels = append(request.Channels, services.PWMChannelRequest{
			Channel:     channel.Channel,
			DutyCycle:   channel.DutyCycle,
			PhaseOffset: channel.PhaseOffset,
			Enabled:     channel.Enabled,
		})
	}

	logger.WithFields(logrus.Fields{
		"node_id":        request.NodeID,
		"address":        request.Address,
		"base_frequency": request.BaseFrequency,
		"channel_count":  request.ChannelCount,
		"channels":       len(request.Channels),
	}).Info("Configuring PWM controller via service")

	// Call the PWM service to configure the controller
	if err := r.PWMService.ConfigureController(ctx, request); err != nil {
		return fmt.Errorf("failed to configure PWM controller via service: %w", err)
	}

	logger.Info("Successfully configured PWM controller")
	return nil
}

// updateStatus updates the PWMController status
func (r *PWMControllerReconciler) updateStatus(ctx context.Context, pwmController *gpiov1.PWMController,
	phase gpiov1.PWMPhase, message, reason string, logger *logrus.Entry) (ctrl.Result, error) {

	// Update basic status fields
	now := metav1.Now()
	pwmController.Status.Phase = phase
	pwmController.Status.Message = message
	pwmController.Status.LastUpdated = &now

	// Update conditions based on phase
	var conditionType gpiov1.PWMConditionType
	var conditionStatus metav1.ConditionStatus

	switch phase {
	case gpiov1.PWMPhaseReady:
		conditionType = gpiov1.PWMConditionReady
		conditionStatus = metav1.ConditionTrue
		// Also set other conditions
		r.setPWMCondition(&pwmController.Status, gpiov1.PWMConditionConfigured, metav1.ConditionTrue, "ConfiguredSuccessfully", "PWM controller configured successfully")
		r.setPWMCondition(&pwmController.Status, gpiov1.PWMConditionNodeReachable, metav1.ConditionTrue, "NodeReachable", "Target node is reachable")
		r.setPWMCondition(&pwmController.Status, gpiov1.PWMConditionI2CConnected, metav1.ConditionTrue, "I2CConnected", "I2C device is accessible")
	case gpiov1.PWMPhaseFailed:
		conditionType = gpiov1.PWMConditionReady
		conditionStatus = metav1.ConditionFalse
	case gpiov1.PWMPhaseConfiguring:
		conditionType = gpiov1.PWMConditionConfigured
		conditionStatus = metav1.ConditionUnknown
	default:
		conditionType = gpiov1.PWMConditionReady
		conditionStatus = metav1.ConditionUnknown
	}

	r.setPWMCondition(&pwmController.Status, conditionType, conditionStatus, reason, message)

	// Update the status
	if err := r.Status().Update(ctx, pwmController); err != nil {
		logger.WithError(err).Error("Failed to update PWMController status")
		return ctrl.Result{}, err
	}

	logger.WithFields(logrus.Fields{
		"phase":   phase,
		"message": message,
	}).Info("Updated PWMController status")

	// Determine requeue behavior
	switch phase {
	case gpiov1.PWMPhasePending:
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	case gpiov1.PWMPhaseConfiguring:
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	case gpiov1.PWMPhaseFailed:
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	default:
		return ctrl.Result{}, nil
	}
}

// setPWMCondition sets a condition in the PWMController status
func (r *PWMControllerReconciler) setPWMCondition(status *gpiov1.PWMControllerStatus,
	conditionType gpiov1.PWMConditionType, conditionStatus metav1.ConditionStatus,
	reason, message string) {

	now := metav1.Now()
	newCondition := gpiov1.PWMCondition{
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

// handleDeletion handles PWMController deletion
func (r *PWMControllerReconciler) handleDeletion(ctx context.Context, pwmController *gpiov1.PWMController, logger *logrus.Entry) (ctrl.Result, error) {
	logger.Info("Handling PWMController deletion")

	// Find the node that was managing this controller
	if pwmController.Status.NodeID != "" {
		// Try to clean up the PWM controller on the node
		cleanupRequest := &services.PWMRequest{
			NodeID:  pwmController.Status.NodeID,
			Address: pwmController.Spec.Address,
			Cleanup: true, // Special flag for cleanup
		}

		if err := r.PWMService.ConfigureController(ctx, cleanupRequest); err != nil {
			logger.WithError(err).Warn("Failed to cleanup PWM controller on node, continuing with deletion")
		} else {
			logger.Info("Successfully cleaned up PWM controller on node")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(pwmController, "gpio.pi-controller.io/finalizer")
	if err := r.Update(ctx, pwmController); err != nil {
		logger.WithError(err).Error("Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully handled PWMController deletion")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *PWMControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpiov1.PWMController{}).
		Watches(&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.mapNodeToPWMControllers)).
		Complete(r)
}

// mapNodeToPWMControllers maps Node changes to PWMController reconcile requests
func (r *PWMControllerReconciler) mapNodeToPWMControllers(ctx context.Context, obj client.Object) []reconcile.Request {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}

	var pwmControllerList gpiov1.PWMControllerList
	if err := r.List(ctx, &pwmControllerList); err != nil {
		r.Logger.WithError(err).Error("Failed to list PWMControllers for node mapping")
		return nil
	}

	var requests []reconcile.Request
	for _, pwmController := range pwmControllerList.Items {
		// Check if this PWMController matches the node
		selector := labels.SelectorFromSet(pwmController.Spec.NodeSelector)
		if selector.Matches(labels.Set(node.Labels)) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pwmController.Name,
					Namespace: pwmController.Namespace,
				},
			})
		}
	}

	r.Logger.WithFields(logrus.Fields{
		"node":                node.Name,
		"pwmcontroller_count": len(requests),
	}).Debug("Mapped node change to PWMController reconcile requests")

	return requests
}
