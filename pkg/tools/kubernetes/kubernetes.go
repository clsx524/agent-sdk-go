package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
)

// Tool implements a Kubernetes provider tool
type Tool struct {
	clientset *kubernetes.Clientset
}

// Option represents an option for configuring the tool
type Option func(*Tool)

// WithClientset sets the Kubernetes clientset for the tool
func WithClientset(clientset *kubernetes.Clientset) Option {
	return func(t *Tool) {
		t.clientset = clientset
	}
}

// New creates a new Kubernetes provider tool
func New(options ...Option) (*Tool, error) {
	tool := &Tool{}

	for _, option := range options {
		option(tool)
	}

	// If no clientset is provided, try to create one
	if tool.clientset == nil {
		// Try in-cluster config first
		config, err := rest.InClusterConfig()
		if err != nil {
			// Fall back to kubeconfig
			kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
			}
		}

		// Create clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
		}

		tool.clientset = clientset
	}

	return tool, nil
}

// Name returns the name of the tool
func (t *Tool) Name() string {
	return "kubernetes_provider"
}

// Description returns a description of what the tool does
func (t *Tool) Description() string {
	return "Interact with Kubernetes resources like pods, services, etc."
}

// Parameters returns the parameters that the tool accepts
func (t *Tool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"resource": {
			Type:        "string",
			Description: "The Kubernetes resource type (pod, service, etc.)",
			Required:    true,
			Enum:        []interface{}{"pod", "service", "deployment", "namespace"},
		},
		"action": {
			Type:        "string",
			Description: "The action to perform (list, get, create, delete)",
			Required:    true,
			Enum:        []interface{}{"list", "get", "create", "delete"},
		},
		"namespace": {
			Type:        "string",
			Description: "The Kubernetes namespace",
			Required:    false,
			Default:     "default",
		},
		"params": {
			Type:        "object",
			Description: "Parameters for the action",
			Required:    false,
		},
	}
}

// Run executes the tool with the given input
func (t *Tool) Run(ctx context.Context, input string) (string, error) {
	// Parse input as JSON
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	// Get resource parameter
	resource, ok := params["resource"].(string)
	if !ok || resource == "" {
		return "", fmt.Errorf("resource parameter is required")
	}

	// Get action parameter
	action, ok := params["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("action parameter is required")
	}

	// Get namespace parameter
	namespace := "default"
	if ns, ok := params["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// Get organization ID for permission checking
	orgID, _ := multitenancy.GetOrgID(ctx)

	// Check permissions
	if err := t.checkPermissions(ctx, orgID, resource, action, namespace); err != nil {
		return "", err
	}

	// Execute action based on resource
	switch strings.ToLower(resource) {
	case "pod":
		return t.handlePod(ctx, action, namespace, params["params"])
	case "service":
		return t.handleService(ctx, action, namespace, params["params"])
	case "deployment":
		return t.handleDeployment(ctx, action, namespace, params["params"])
	case "namespace":
		return t.handleNamespace(ctx, action, params["params"])
	default:
		return "", fmt.Errorf("unsupported resource: %s", resource)
	}
}

// checkPermissions checks if the organization has permission to perform the action
func (t *Tool) checkPermissions(ctx context.Context, orgID, resource, action, namespace string) error {
	// In a real implementation, this would check against a permission system
	// For now, we'll just allow all actions
	return nil
}

// handlePod handles pod actions
func (t *Tool) handlePod(ctx context.Context, action, namespace string, params interface{}) (string, error) {
	switch strings.ToLower(action) {
	case "list":
		return t.listPods(ctx, namespace)
	case "get":
		name, ok := getStringParam(params, "name")
		if !ok {
			return "", fmt.Errorf("name parameter is required for get action")
		}
		return t.getPod(ctx, namespace, name)
	case "create":
		spec, ok := getMapParam(params, "spec")
		if !ok {
			return "", fmt.Errorf("spec parameter is required for create action")
		}
		return t.createPod(ctx, namespace, spec)
	case "delete":
		name, ok := getStringParam(params, "name")
		if !ok {
			return "", fmt.Errorf("name parameter is required for delete action")
		}
		return t.deletePod(ctx, namespace, name)
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

// listPods lists all pods in a namespace
func (t *Tool) listPods(ctx context.Context, namespace string) (string, error) {
	// List pods
	pods, err := t.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	// Format result
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pods in namespace '%s':\n", namespace))
	for _, pod := range pods.Items {
		sb.WriteString(fmt.Sprintf("- %s (status: %s)\n", pod.Name, pod.Status.Phase))
	}

	return sb.String(), nil
}

// getPod gets a pod by name
func (t *Tool) getPod(ctx context.Context, namespace, name string) (string, error) {
	// Get pod
	pod, err := t.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	// Format result
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pod '%s' in namespace '%s':\n", name, namespace))
	sb.WriteString(fmt.Sprintf("Status: %s\n", pod.Status.Phase))
	sb.WriteString("Containers:\n")
	for _, container := range pod.Spec.Containers {
		sb.WriteString(fmt.Sprintf("- %s (image: %s)\n", container.Name, container.Image))
	}

	return sb.String(), nil
}

// createPod creates a pod
func (t *Tool) createPod(ctx context.Context, namespace string, spec map[string]interface{}) (string, error) {
	// Convert spec to Pod
	var pod corev1.Pod
	specBytes, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pod spec: %w", err)
	}
	if err := json.Unmarshal(specBytes, &pod); err != nil {
		return "", fmt.Errorf("failed to unmarshal pod spec: %w", err)
	}

	// Create pod
	result, err := t.clientset.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create pod: %w", err)
	}

	return fmt.Sprintf("Pod '%s' created successfully in namespace '%s'", result.Name, namespace), nil
}

// deletePod deletes a pod
func (t *Tool) deletePod(ctx context.Context, namespace, name string) (string, error) {
	// Delete pod
	err := t.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to delete pod: %w", err)
	}

	return fmt.Sprintf("Pod '%s' deleted successfully from namespace '%s'", name, namespace), nil
}

// handleService handles service actions
func (t *Tool) handleService(ctx context.Context, action, namespace string, params interface{}) (string, error) {
	switch strings.ToLower(action) {
	case "list":
		return t.listServices(ctx, namespace)
	case "get":
		name, ok := getStringParam(params, "name")
		if !ok {
			return "", fmt.Errorf("name parameter is required for get action")
		}
		return t.getService(ctx, namespace, name)
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

// listServices lists all services in a namespace
func (t *Tool) listServices(ctx context.Context, namespace string) (string, error) {
	// List services
	services, err := t.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	// Format result
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Services in namespace '%s':\n", namespace))
	for _, svc := range services.Items {
		sb.WriteString(fmt.Sprintf("- %s (type: %s)\n", svc.Name, svc.Spec.Type))
	}

	return sb.String(), nil
}

// getService gets a service by name
func (t *Tool) getService(ctx context.Context, namespace, name string) (string, error) {
	// Get service
	svc, err := t.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service: %w", err)
	}

	// Format result
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Service '%s' in namespace '%s':\n", name, namespace))
	sb.WriteString(fmt.Sprintf("Type: %s\n", svc.Spec.Type))
	sb.WriteString("Ports:\n")
	for _, port := range svc.Spec.Ports {
		sb.WriteString(fmt.Sprintf("- %d:%d/%s\n", port.Port, port.TargetPort.IntVal, port.Protocol))
	}

	return sb.String(), nil
}

// handleDeployment handles deployment actions
func (t *Tool) handleDeployment(ctx context.Context, action, namespace string, params interface{}) (string, error) {
	// Implementation for deployment actions
	return "", fmt.Errorf("deployment actions not implemented yet")
}

// handleNamespace handles namespace actions
func (t *Tool) handleNamespace(ctx context.Context, action string, params interface{}) (string, error) {
	// Implementation for namespace actions
	return "", fmt.Errorf("namespace actions not implemented yet")
}

// Helper functions

// getStringParam gets a string parameter from a map
func getStringParam(params interface{}, key string) (string, bool) {
	if params == nil {
		return "", false
	}

	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "", false
	}

	value, ok := paramsMap[key].(string)
	return value, ok && value != ""
}

// getMapParam gets a map parameter from a map
func getMapParam(params interface{}, key string) (map[string]interface{}, bool) {
	if params == nil {
		return nil, false
	}

	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, false
	}

	value, ok := paramsMap[key].(map[string]interface{})
	return value, ok && len(value) > 0
}
