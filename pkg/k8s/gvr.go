package k8s

import "k8s.io/apimachinery/pkg/runtime/schema"

// GetGVR returns the GroupVersionResource for a given kind
func GetGVR(kind, apiVersion string) schema.GroupVersionResource {
	mapping := map[string]schema.GroupVersionResource{
		"Deployment":     {Group: "apps", Version: "v1", Resource: "deployments"},
		"StatefulSet":    {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"DaemonSet":      {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"ReplicaSet":     {Group: "apps", Version: "v1", Resource: "replicasets"},
		"Service":        {Version: "v1", Resource: "services"},
		"ConfigMap":      {Version: "v1", Resource: "configmaps"},
		"Secret":         {Version: "v1", Resource: "secrets"},
		"Pod":            {Version: "v1", Resource: "pods"},
		"Namespace":      {Version: "v1", Resource: "namespaces"},
		"Ingress":        {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"NetworkPolicy":  {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
		"Job":            {Group: "batch", Version: "v1", Resource: "jobs"},
		"CronJob":        {Group: "batch", Version: "v1", Resource: "cronjobs"},
	}

	if gvr, ok := mapping[kind]; ok {
		return gvr
	}

	// Fallback: use kind as resource name (lowercase)
	return schema.GroupVersionResource{Resource: kind}
}
