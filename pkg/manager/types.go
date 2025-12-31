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
	Resource     models.Resource              `json:"resource"`
	AppliedState *models.ResourceAppliedState `json:"applied_state,omitempty"`
	CurrentState *models.ResourceCurrentState `json:"current_state,omitempty"`
}

// CreateGlobalResourceRequest represents a request to create a new global resource
type CreateGlobalResourceRequest struct {
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
}

// ClusterSyncStatus represents sync status for a single cluster
type ClusterSyncStatus struct {
	ClusterID        string `json:"cluster_id"`
	SyncedGeneration int    `json:"synced_generation"`
	IsSynced         bool   `json:"is_synced"`
}

// GlobalResourceWithSyncStatus represents a global resource with its sync status across clusters
type GlobalResourceWithSyncStatus struct {
	GlobalResource  models.GlobalResource `json:"global_resource"`
	TotalClusters   int                   `json:"total_clusters"`
	SyncedClusters  int                   `json:"synced_clusters"`
	ClusterStatuses []ClusterSyncStatus   `json:"cluster_statuses,omitempty"`
}
