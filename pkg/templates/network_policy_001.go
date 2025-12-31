package templates

import (
	"encoding/json"
)

type NetworkPolicy001 struct {
	Namespace string
	Name      string
	PodLabels map[string]string
	AllowFrom []map[string]string
}

func (t NetworkPolicy001) TemplateName() string {
	return "network-policy-001"
}

func (t NetworkPolicy001) Build() (kind, apiVersion, namespace, name string, spec json.RawMessage, err error) {
	kind = "NetworkPolicy"
	apiVersion = "networking.k8s.io/v1"
	namespace = t.Namespace
	name = t.Name

	ingressFrom := make([]map[string]interface{}, len(t.AllowFrom))

	for i, labels := range t.AllowFrom {
		ingressFrom[i] = map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": labels,
			},
		}
	}

	policy := map[string]interface{}{
		"spec": map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": t.PodLabels,
			},
			"policyTypes": []string{"Ingress"},
			"ingress": []map[string]interface{}{
				{
					"from": ingressFrom,
				},
			},
			"egress": []map[string]interface{}{
				{},
			},
		},
	}

	spec, err = json.Marshal(policy)

	if err != nil {
		return "", "", "", "", nil, err
	}

	return kind, apiVersion, namespace, name, spec, nil
}
