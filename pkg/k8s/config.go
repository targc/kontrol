package k8s

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// BuildConfig creates a Kubernetes client configuration.
// If kubeconfig is provided, it uses that file.
// If kubeconfig is empty, it attempts to use in-cluster configuration.
func BuildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		// Use provided kubeconfig file
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
		return config, nil
	}

	// Try in-cluster config (when running inside Kubernetes)
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build in-cluster config (not running in K8s or kubeconfig not provided): %w", err)
	}

	return config, nil
}
