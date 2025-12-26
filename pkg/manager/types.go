package manager

import (
	"encoding/json"

	"github.com/targc/kontrol/pkg/models"
)

// CreateResourceRequest represents a request to create a new resource
type CreateResourceRequest struct {
	ClusterID   string          `json:"cluster_id"`
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
}

// UpdateResourceRequest represents a request to update a resource
type UpdateResourceRequest struct {
	DesiredSpec json.RawMessage `json:"desired_spec"`
}

// ResourceWithState represents a resource with its applied and current states
type ResourceWithState struct {
	Resource     models.Resource                   `json:"resource"`
	AppliedState *models.ResourceAppliedState      `json:"applied_state,omitempty"`
	CurrentState *models.ResourceCurrentState      `json:"current_state,omitempty"`
}
