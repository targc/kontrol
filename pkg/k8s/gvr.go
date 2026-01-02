package k8s

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// gvrMapping maps Kind names to their GVRs
var gvrMapping = map[string]schema.GroupVersionResource{
	"Deployment":    {Group: "apps", Version: "v1", Resource: "deployments"},
	"StatefulSet":   {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"DaemonSet":     {Group: "apps", Version: "v1", Resource: "daemonsets"},
	"ReplicaSet":    {Group: "apps", Version: "v1", Resource: "replicasets"},
	"Service":       {Version: "v1", Resource: "services"},
	"ConfigMap":     {Version: "v1", Resource: "configmaps"},
	"Secret":        {Version: "v1", Resource: "secrets"},
	"Pod":           {Version: "v1", Resource: "pods"},
	"Namespace":     {Version: "v1", Resource: "namespaces"},
	"Ingress":       {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	"NetworkPolicy": {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
	"Job":           {Group: "batch", Version: "v1", Resource: "jobs"},
	"CronJob":       {Group: "batch", Version: "v1", Resource: "cronjobs"},
}

// aliasToKind maps lowercase aliases to Kind names for env var config
var aliasToKind = map[string]string{
	"deployment":    "Deployment",
	"statefulset":   "StatefulSet",
	"daemonset":     "DaemonSet",
	"replicaset":    "ReplicaSet",
	"service":       "Service",
	"configmap":     "ConfigMap",
	"secret":        "Secret",
	"pod":           "Pod",
	"namespace":     "Namespace",
	"ingress":       "Ingress",
	"networkpolicy": "NetworkPolicy",
	"job":           "Job",
	"cronjob":       "CronJob",
}

// SupportedGVRs holds the filtered list of GVRs to watch
var SupportedGVRs []schema.GroupVersionResource

// InitSupportedGVRs initializes SupportedGVRs based on the filter string.
// If filter is empty, all GVRs are enabled.
// Filter format: comma-separated lowercase names (e.g., "deployment,pod,service")
func InitSupportedGVRs(filter string) {
	if filter == "" {
		// No filter - enable all
		SupportedGVRs = make([]schema.GroupVersionResource, 0, len(gvrMapping))
		for _, gvr := range gvrMapping {
			SupportedGVRs = append(SupportedGVRs, gvr)
		}
		return
	}

	parts := strings.Split(filter, ",")
	SupportedGVRs = make([]schema.GroupVersionResource, 0, len(parts))

	for _, part := range parts {
		alias := strings.TrimSpace(strings.ToLower(part))
		if kind, ok := aliasToKind[alias]; ok {
			if gvr, ok := gvrMapping[kind]; ok {
				SupportedGVRs = append(SupportedGVRs, gvr)
			}
		}
	}
}

func GetGVR(kind, apiVersion string) schema.GroupVersionResource {
	if gvr, ok := gvrMapping[kind]; ok {
		return gvr
	}

	return schema.GroupVersionResource{Resource: kind}
}
