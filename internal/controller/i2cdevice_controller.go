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

	gpiov1 "github.com/dsyorkd/pi-controller/pkg/apis/gpio/v1"
	"github.com/dsyorkd/pi-controller/internal/services"
)

// I2CDeviceReconciler reconciles an I2CDevice object
type I2CDeviceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Logger      *logrus.Entry
	I2CService  services.I2CControllerService
	NodeService services.NodeControllerService
}

// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=i2cdevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=i2cdevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gpio.pi-controller.io,resources=i2cdevices/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *I2CDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithFields(logrus.Fields{
		"i2cdevice": req.NamespacedName,
		"namespace": req.Namespace,
	})

	logger.Info("Starting I2CDevice reconciliation")

	// Fetch the I2CDevice instance
	var i2cDevice gpiov1.I2CDevice
	if err := r.Get(ctx, req.NamespacedName, &i2cDevice); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("I2CDevice resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		logger.WithError(err).Error("Failed to get I2CDevice resource")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !i2cDevice.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &i2cDevice, logger)
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(&i2cDevice, "gpio.pi-controller.io/finalizer") {
		controllerutil.AddFinalizer(&i2cDevice, "gpio.pi-controller.io/finalizer")
		if err := r.Update(ctx, &i2cDevice); err != nil {
			logger.WithError(err).Error("Failed to add finalizer")
			return ctrl.Result{}, err
		}
		logger.Info("Added finalizer to I2CDevice")
		return ctrl.Result{Requeue: true}, nil
	}

	// Find the target node
	targetNode, err := r.findTargetNode(ctx, &i2cDevice, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to find target node")
		return r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhaseFailed, 
			"Failed to find target node", err.Error(), logger)
	}

	if targetNode == nil {
		logger.Warn("No suitable node found for I2CDevice")
		return r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhasePending,
			"No suitable node found", "Waiting for matching node", logger)
	}

	// Check if node is reachable
	nodeReachable, err := r.checkNodeReachability(ctx, targetNode, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to check node reachability")
		return r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhaseFailed,
			"Failed to check node reachability", err.Error(), logger)
	}

	if !nodeReachable {
		logger.Warn("Target node is not reachable")
		return r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhaseFailed,
			"Target node unreachable", "Node is not responding", logger)
	}

	// Update phase to configuring
	if i2cDevice.Status.Phase != gpiov1.I2CPhaseConfiguring {
		result, err := r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhaseConfiguring,
			"Configuring I2C device", "Setting up hardware configuration", logger)
		if err != nil {
			return result, err
		}
	}

	// Configure the I2C device on the target node
	if err := r.configureI2CDevice(ctx, &i2cDevice, targetNode, logger); err != nil {
		logger.WithError(err).Error("Failed to configure I2C device")
		return r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhaseFailed,
			"Configuration failed", err.Error(), logger)
	}

	// Update phase to ready and schedule periodic scans if configured
	result, err := r.updateStatus(ctx, &i2cDevice, gpiov1.I2CPhaseReady,
		"I2C device ready", "Hardware configured successfully", logger)
	
	// If scan interval is configured, schedule the next scan
	if i2cDevice.Spec.ScanInterval > 0 {
		result.RequeueAfter = time.Duration(i2cDevice.Spec.ScanInterval) * time.Second
	}
	
	return result, err
}

// findTargetNode finds the node that matches the nodeSelector
func (r *I2CDeviceReconciler) findTargetNode(ctx context.Context, i2cDevice *gpiov1.I2CDevice, logger *logrus.Entry) (*corev1.Node, error) {
	// List all nodes
	var nodeList corev1.NodeList
	if err := r.List(ctx, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Convert nodeSelector to label selector
	selector := labels.SelectorFromSet(i2cDevice.Spec.NodeSelector)

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
	}).Info("Found matching node for I2CDevice")

	return &matchingNodes[0], nil
}

// checkNodeReachability checks if the target node is reachable
func (r *I2CDeviceReconciler) checkNodeReachability(ctx context.Context, node *corev1.Node, logger *logrus.Entry) (bool, error) {
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

// configureI2CDevice configures the I2C device on the target node via gRPC
func (r *I2CDeviceReconciler) configureI2CDevice(ctx context.Context, i2cDevice *gpiov1.I2CDevice, node *corev1.Node, logger *logrus.Entry) error {
	// Convert I2CDevice spec to I2C service request
	request := &services.I2CRequest{
		NodeID:       node.Name,
		Address:      i2cDevice.Spec.Address,
		DeviceType:   string(i2cDevice.Spec.DeviceType),
		BusNumber:    i2cDevice.Spec.BusNumber,
		DataFormat:   string(i2cDevice.Spec.DataFormat),
		ScanInterval: i2cDevice.Spec.ScanInterval,
	}

	// Convert register configurations
	for _, register := range i2cDevice.Spec.Registers {
		request.Registers = append(request.Registers, services.I2CRegisterRequest{
			Address:      register.Address,
			Name:         register.Name,
			Mode:         string(register.Mode),
			Size:         register.Size,
			InitialValue: register.InitialValue,
		})
	}

	logger.WithFields(logrus.Fields{
		"node_id":       request.NodeID,
		"address":       request.Address,
		"device_type":   request.DeviceType,
		"bus_number":    request.BusNumber,
		"data_format":   request.DataFormat,
		"scan_interval": request.ScanInterval,
		"registers":     len(request.Registers),
	}).Info("Configuring I2C device via service")

	// Call the I2C service to configure the device
	if err := r.I2CService.ConfigureDevice(ctx, request); err != nil {
		return fmt.Errorf("failed to configure I2C device via service: %w", err)
	}

	logger.Info("Successfully configured I2C device")
	return nil
}

// updateStatus updates the I2CDevice status
func (r *I2CDeviceReconciler) updateStatus(ctx context.Context, i2cDevice *gpiov1.I2CDevice, 
	phase gpiov1.I2CPhase, message, reason string, logger *logrus.Entry) (ctrl.Result, error) {

	// Update basic status fields
	now := metav1.Now()
	i2cDevice.Status.Phase = phase
	i2cDevice.Status.Message = message
	i2cDevice.Status.LastUpdated = &now

	// If the device is ready and configured for scanning, read current data
	if phase == gpiov1.I2CPhaseReady && i2cDevice.Spec.ScanInterval > 0 {
		if err := r.readDeviceData(ctx, i2cDevice, logger); err != nil {
			logger.WithError(err).Warn("Failed to read device data during status update")
		}
	}

	// Update conditions based on phase
	var conditionType gpiov1.I2CConditionType
	var conditionStatus metav1.ConditionStatus

	switch phase {
	case gpiov1.I2CPhaseReady:
		conditionType = gpiov1.I2CConditionReady
		conditionStatus = metav1.ConditionTrue
		// Also set other conditions
		r.setI2CCondition(&i2cDevice.Status, gpiov1.I2CConditionConfigured, metav1.ConditionTrue, "ConfiguredSuccessfully", "I2C device configured successfully")
		r.setI2CCondition(&i2cDevice.Status, gpiov1.I2CConditionNodeReachable, metav1.ConditionTrue, "NodeReachable", "Target node is reachable")
		r.setI2CCondition(&i2cDevice.Status, gpiov1.I2CConditionDeviceConnected, metav1.ConditionTrue, "DeviceConnected", "I2C device is accessible")
	case gpiov1.I2CPhaseFailed:
		conditionType = gpiov1.I2CConditionReady
		conditionStatus = metav1.ConditionFalse
	case gpiov1.I2CPhaseConfiguring:
		conditionType = gpiov1.I2CConditionConfigured
		conditionStatus = metav1.ConditionUnknown
	default:
		conditionType = gpiov1.I2CConditionReady
		conditionStatus = metav1.ConditionUnknown
	}

	r.setI2CCondition(&i2cDevice.Status, conditionType, conditionStatus, reason, message)

	// Update the status
	if err := r.Status().Update(ctx, i2cDevice); err != nil {
		logger.WithError(err).Error("Failed to update I2CDevice status")
		return ctrl.Result{}, err
	}

	logger.WithFields(logrus.Fields{
		"phase":   phase,
		"message": message,
	}).Info("Updated I2CDevice status")

	// Determine requeue behavior
	switch phase {
	case gpiov1.I2CPhasePending:
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	case gpiov1.I2CPhaseConfiguring:
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	case gpiov1.I2CPhaseFailed:
		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	default:
		return ctrl.Result{}, nil
	}
}

// readDeviceData reads current data from the I2C device
func (r *I2CDeviceReconciler) readDeviceData(ctx context.Context, i2cDevice *gpiov1.I2CDevice, logger *logrus.Entry) error {
	// Create read request
	readRequest := &services.I2CReadRequest{
		NodeID:  i2cDevice.Status.NodeID,
		Address: i2cDevice.Spec.Address,
	}

	// Read data from the I2C service
	data, err := r.I2CService.ReadDevice(ctx, readRequest)
	if err != nil {
		return fmt.Errorf("failed to read I2C device data: %w", err)
	}

	// Update the register data in status
	now := metav1.Now()
	i2cDevice.Status.RegisterData = data
	i2cDevice.Status.LastScan = &now

	logger.WithField("data_keys", len(data)).Debug("Successfully read I2C device data")
	return nil
}

// setI2CCondition sets a condition in the I2CDevice status
func (r *I2CDeviceReconciler) setI2CCondition(status *gpiov1.I2CDeviceStatus, 
	conditionType gpiov1.I2CConditionType, conditionStatus metav1.ConditionStatus,
	reason, message string) {

	now := metav1.Now()
	newCondition := gpiov1.I2CCondition{
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

// handleDeletion handles I2CDevice deletion
func (r *I2CDeviceReconciler) handleDeletion(ctx context.Context, i2cDevice *gpiov1.I2CDevice, logger *logrus.Entry) (ctrl.Result, error) {
	logger.Info("Handling I2CDevice deletion")

	// Find the node that was managing this device
	if i2cDevice.Status.NodeID != "" {
		// Try to clean up the I2C device on the node
		cleanupRequest := &services.I2CRequest{
			NodeID:  i2cDevice.Status.NodeID,
			Address: i2cDevice.Spec.Address,
			Cleanup: true, // Special flag for cleanup
		}

		if err := r.I2CService.ConfigureDevice(ctx, cleanupRequest); err != nil {
			logger.WithError(err).Warn("Failed to cleanup I2C device on node, continuing with deletion")
		} else {
			logger.Info("Successfully cleaned up I2C device on node")
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(i2cDevice, "gpio.pi-controller.io/finalizer")
	if err := r.Update(ctx, i2cDevice); err != nil {
		logger.WithError(err).Error("Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully handled I2CDevice deletion")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *I2CDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpiov1.I2CDevice{}).
		Watches(&corev1.Node{}, 
			handler.EnqueueRequestsFromMapFunc(r.mapNodeToI2CDevices)).
		Complete(r)
}

// mapNodeToI2CDevices maps Node changes to I2CDevice reconcile requests
func (r *I2CDeviceReconciler) mapNodeToI2CDevices(ctx context.Context, obj client.Object) []reconcile.Request {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}

	var i2cDeviceList gpiov1.I2CDeviceList
	if err := r.List(ctx, &i2cDeviceList); err != nil {
		r.Logger.WithError(err).Error("Failed to list I2CDevices for node mapping")
		return nil
	}

	var requests []reconcile.Request
	for _, i2cDevice := range i2cDeviceList.Items {
		// Check if this I2CDevice matches the node
		selector := labels.SelectorFromSet(i2cDevice.Spec.NodeSelector)
		if selector.Matches(labels.Set(node.Labels)) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      i2cDevice.Name,
					Namespace: i2cDevice.Namespace,
				},
			})
		}
	}

	r.Logger.WithFields(logrus.Fields{
		"node":            node.Name,
		"i2cdevice_count": len(requests),
	}).Debug("Mapped node change to I2CDevice reconcile requests")

	return requests
}