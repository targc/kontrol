package k8s

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func BuildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)

		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}

		return config, nil
	}

	config, err := rest.InClusterConfig()

	if err != nil {
		return nil, fmt.Errorf("failed to build in-cluster config: %w", err)
	}

	return config, nil
}
