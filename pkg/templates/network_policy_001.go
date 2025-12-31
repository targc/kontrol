package templates

import (
	"encoding/json"
	"fmt"
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

func (t *NetworkPolicy001) Decompile(spec json.RawMessage) error {
	var data map[string]interface{}

	if err := json.Unmarshal(spec, &data); err != nil {
		return fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	specMap, ok := data["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("spec field not found or invalid")
	}

	podSelector, ok := specMap["podSelector"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("podSelector field not found or invalid")
	}

	matchLabels, ok := podSelector["matchLabels"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("matchLabels field not found or invalid")
	}

	t.PodLabels = toStringMap(matchLabels)

	ingress, ok := specMap["ingress"].([]interface{})
	if ok && len(ingress) > 0 {
		firstRule, ok := ingress[0].(map[string]interface{})
		if !ok {
			return nil
		}

		from, ok := firstRule["from"].([]interface{})
		if !ok {
			return nil
		}

		t.AllowFrom = make([]map[string]string, 0, len(from))

		for _, f := range from {
			fMap, ok := f.(map[string]interface{})
			if !ok {
				continue
			}

			ps, ok := fMap["podSelector"].(map[string]interface{})
			if !ok {
				continue
			}

			ml, ok := ps["matchLabels"].(map[string]interface{})
			if !ok {
				continue
			}

			t.AllowFrom = append(t.AllowFrom, toStringMap(ml))
		}
	}

	return nil
}

func toStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string, len(m))

	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}

	return result
}
