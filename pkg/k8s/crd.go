package k8s

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// CRDClient provides functionality for working with Custom Resource Definitions
type CRDClient struct {
	client        *Client
	apiExtensions apiextensionsclientset.Interface
	dynamicClient dynamic.Interface
}

// NewCRDClient creates a new CRD client
func (c *Client) NewCRDClient() (*CRDClient, error) {
	// Create API extensions client for managing CRDs
	apiExtClient, err := apiextensionsclientset.NewForConfig(c.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API extensions client: %w", err)
	}

	// Create dynamic client for working with custom resources
	dynamicClient, err := dynamic.NewForConfig(c.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &CRDClient{
		client:        c,
		apiExtensions: apiExtClient,
		dynamicClient: dynamicClient,
	}, nil
}

// CRDInfo represents information about a Custom Resource Definition
type CRDInfo struct {
	Name              string                           `json:"name"`
	Group             string                           `json:"group"`
	Kind              string                           `json:"kind"`
	Version           string                           `json:"version"`
	Scope             string                           `json:"scope"`
	ShortNames        []string                         `json:"short_names"`
	Categories        []string                         `json:"categories"`
	StoredVersions    []string                         `json:"stored_versions"`
	ServedVersions    []string                         `json:"served_versions"`
	Conditions        []CRDCondition                   `json:"conditions"`
	AdditionalColumns []CRDAdditionalColumn            `json:"additional_columns"`
	Schema            *apiextensionsv1.JSONSchemaProps `json:"schema,omitempty"`
}

// CRDCondition represents a CRD condition
type CRDCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason"`
	Message            string `json:"message"`
	LastTransitionTime string `json:"last_transition_time"`
}

// CRDAdditionalColumn represents additional printer columns for a CRD
type CRDAdditionalColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	JSONPath string `json:"json_path"`
}

// ListCRDs returns information about all Custom Resource Definitions
func (crd *CRDClient) ListCRDs(ctx context.Context) ([]CRDInfo, error) {
	crdList, err := crd.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	var crds []CRDInfo
	for _, c := range crdList.Items {
		crds = append(crds, crd.convertCRDToInfo(c))
	}

	return crds, nil
}

// GetCRD returns information about a specific Custom Resource Definition
func (crd *CRDClient) GetCRD(ctx context.Context, name string) (*CRDInfo, error) {
	c, err := crd.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get CRD %s: %w", name, err)
	}

	info := crd.convertCRDToInfo(*c)
	return &info, nil
}

// CreateCRD creates a new Custom Resource Definition
func (crd *CRDClient) CreateCRD(ctx context.Context, crdDef *apiextensionsv1.CustomResourceDefinition) (*CRDInfo, error) {
	created, err := crd.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crdDef, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create CRD: %w", err)
	}

	info := crd.convertCRDToInfo(*created)
	return &info, nil
}

// UpdateCRD updates an existing Custom Resource Definition
func (crd *CRDClient) UpdateCRD(ctx context.Context, crdDef *apiextensionsv1.CustomResourceDefinition) (*CRDInfo, error) {
	updated, err := crd.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, crdDef, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update CRD: %w", err)
	}

	info := crd.convertCRDToInfo(*updated)
	return &info, nil
}

// DeleteCRD deletes a Custom Resource Definition
func (crd *CRDClient) DeleteCRD(ctx context.Context, name string) error {
	err := crd.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete CRD %s: %w", name, err)
	}

	crd.client.logger.WithField("crd_name", name).Info("CRD deleted successfully")
	return nil
}

// WaitForCRDEstablished waits for a CRD to become established
func (crd *CRDClient) WaitForCRDEstablished(ctx context.Context, name string) error {
	crd.client.logger.WithField("crd_name", name).Info("Waiting for CRD to be established")

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for CRD %s to be established", name)
		default:
			c, err := crd.apiExtensions.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get CRD %s: %w", name, err)
			}

			// Check if CRD is established
			for _, condition := range c.Status.Conditions {
				if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
					crd.client.logger.WithField("crd_name", name).Info("CRD established successfully")
					return nil
				}
			}

			// Wait a bit before checking again
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while waiting for CRD %s to be established", name)
			case <-ctx.Done():
				// Small delay to prevent tight polling
			}
		}
	}
}

// GetCustomResource retrieves a custom resource by GVR (Group Version Resource)
func (crd *CRDClient) GetCustomResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (map[string]interface{}, error) {
	var resource dynamic.ResourceInterface
	if namespace == "" {
		// Cluster-scoped resource
		resource = crd.dynamicClient.Resource(gvr)
	} else {
		// Namespaced resource
		resource = crd.dynamicClient.Resource(gvr).Namespace(namespace)
	}

	obj, err := resource.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get custom resource %s: %w", name, err)
	}

	return obj.Object, nil
}

// ListCustomResources lists all custom resources of a given type
func (crd *CRDClient) ListCustomResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]map[string]interface{}, error) {
	var resource dynamic.ResourceInterface
	if namespace == "" {
		// Cluster-scoped resources
		resource = crd.dynamicClient.Resource(gvr)
	} else {
		// Namespaced resources
		resource = crd.dynamicClient.Resource(gvr).Namespace(namespace)
	}

	list, err := resource.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list custom resources: %w", err)
	}

	var resources []map[string]interface{}
	for _, item := range list.Items {
		resources = append(resources, item.Object)
	}

	return resources, nil
}

// CheckPiControllerCRDs checks if Pi Controller CRDs are installed and ready
func (crd *CRDClient) CheckPiControllerCRDs(ctx context.Context) (bool, []string, error) {
	expectedCRDs := []string{
		"gpiopins.gpio.pi-controller.io",
		"pwmcontrollers.gpio.pi-controller.io",
		"i2cdevices.gpio.pi-controller.io",
	}

	var missingCRDs []string
	allPresent := true

	for _, crdName := range expectedCRDs {
		_, err := crd.GetCRD(ctx, crdName)
		if err != nil {
			missingCRDs = append(missingCRDs, crdName)
			allPresent = false
			crd.client.logger.WithField("crd_name", crdName).Warn("Pi Controller CRD not found")
		} else {
			crd.client.logger.WithField("crd_name", crdName).Info("Pi Controller CRD found and ready")
		}
	}

	return allPresent, missingCRDs, nil
}

// convertCRDToInfo converts a Kubernetes CRD object to our CRDInfo struct
func (crd *CRDClient) convertCRDToInfo(c apiextensionsv1.CustomResourceDefinition) CRDInfo {
	var conditions []CRDCondition
	for _, condition := range c.Status.Conditions {
		conditions = append(conditions, CRDCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime.Format("2006-01-02T15:04:05Z"),
		})
	}

	// Get the stored version (there should be exactly one)
	var storedVersion string
	var additionalColumns []CRDAdditionalColumn
	var servedVersions, storedVersions []string

	for _, version := range c.Spec.Versions {
		servedVersions = append(servedVersions, version.Name)
		if version.Storage {
			storedVersion = version.Name
			storedVersions = append(storedVersions, version.Name)

			// Get additional printer columns
			for _, col := range version.AdditionalPrinterColumns {
				additionalColumns = append(additionalColumns, CRDAdditionalColumn{
					Name:     col.Name,
					Type:     col.Type,
					JSONPath: col.JSONPath,
				})
			}
		}
	}

	var shortNames, categories []string
	if c.Spec.Names.ShortNames != nil {
		shortNames = c.Spec.Names.ShortNames
	}
	if c.Spec.Names.Categories != nil {
		categories = c.Spec.Names.Categories
	}

	return CRDInfo{
		Name:              c.Name,
		Group:             c.Spec.Group,
		Kind:              c.Spec.Names.Kind,
		Version:           storedVersion,
		Scope:             string(c.Spec.Scope),
		ShortNames:        shortNames,
		Categories:        categories,
		StoredVersions:    storedVersions,
		ServedVersions:    servedVersions,
		Conditions:        conditions,
		AdditionalColumns: additionalColumns,
	}
}
