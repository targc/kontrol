package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
)

type Client struct {
	baseURL    string
	apiKey     string
	clusterID  string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, clusterID string) *Client {
	return &Client{
		baseURL:   baseURL,
		apiKey:    apiKey,
		clusterID: clusterID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader

	if body != nil {
		data, err := json.Marshal(body)

		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Cluster-ID", c.clusterID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}

		json.NewDecoder(resp.Body).Decode(&errResp)

		return fmt.Errorf("api error (%d): %s", resp.StatusCode, errResp.Error)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// RegisterCluster registers or updates the cluster
func (c *Client) RegisterCluster(ctx context.Context) error {
	return c.doRequest(ctx, "POST", "/int/api/v1/cluster/register", nil, nil)
}

// ListOutOfSyncResources fetches resources that need reconciliation
func (c *Client) ListOutOfSyncResources(ctx context.Context, limit int) ([]models.Resource, error) {
	var resp struct {
		Data []models.Resource `json:"data"`
	}

	path := fmt.Sprintf("/int/api/v1/resources/out-of-sync?limit=%d", limit)
	err := c.doRequest(ctx, "GET", path, nil, &resp)

	return resp.Data, err
}

// ListDeletedResources fetches soft-deleted resources for cleanup
func (c *Client) ListDeletedResources(ctx context.Context, limit int) ([]models.Resource, error) {
	var resp struct {
		Data []models.Resource `json:"data"`
	}

	path := fmt.Sprintf("/int/api/v1/resources/deleted?limit=%d", limit)
	err := c.doRequest(ctx, "GET", path, nil, &resp)

	return resp.Data, err
}

// UpsertAppliedStateRequest is the request body for UpsertAppliedState
type UpsertAppliedStateRequest struct {
	Spec         json.RawMessage `json:"spec"`
	Generation   int             `json:"generation"`
	Revision     int             `json:"revision"`
	Status       string          `json:"status"`
	ErrorMessage *string         `json:"error_message"`
}

// UpsertAppliedState updates the applied state for a resource
func (c *Client) UpsertAppliedState(ctx context.Context, resourceID uuid.UUID, req *UpsertAppliedStateRequest) error {
	path := fmt.Sprintf("/int/api/v1/resources/%s/applied-state", resourceID)
	return c.doRequest(ctx, "POST", path, req, nil)
}

// HardDeleteResource permanently removes a resource
func (c *Client) HardDeleteResource(ctx context.Context, resourceID uuid.UUID) error {
	path := fmt.Sprintf("/int/api/v1/resources/%s", resourceID)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// UpsertCurrentStateRequest is the request body for UpsertCurrentState
type UpsertCurrentStateRequest struct {
	Spec               json.RawMessage `json:"spec"`
	Generation         int             `json:"generation"`
	Revision           int             `json:"revision"`
	K8sResourceVersion string          `json:"k8s_resource_version"`
}

// UpsertCurrentState updates the current state for a resource
func (c *Client) UpsertCurrentState(ctx context.Context, resourceID uuid.UUID, req *UpsertCurrentStateRequest) error {
	path := fmt.Sprintf("/int/api/v1/resources/%s/current-state", resourceID)
	return c.doRequest(ctx, "POST", path, req, nil)
}

// DeleteCurrentState removes the current state for a resource
func (c *Client) DeleteCurrentState(ctx context.Context, resourceID uuid.UUID) error {
	path := fmt.Sprintf("/int/api/v1/resources/%s/current-state", resourceID)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// GlobalResourceForSync represents a global resource that needs syncing
type GlobalResourceForSync struct {
	ID          uuid.UUID       `json:"id"`
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
	Generation  int             `json:"generation"`
	Revision    int             `json:"revision"`
}

// ListOutOfSyncGlobalResources fetches global resources that need syncing
func (c *Client) ListOutOfSyncGlobalResources(ctx context.Context, limit int) ([]GlobalResourceForSync, error) {
	var resp struct {
		Data []GlobalResourceForSync `json:"data"`
	}

	path := fmt.Sprintf("/int/api/v1/global-resources/out-of-sync?limit=%d", limit)
	err := c.doRequest(ctx, "GET", path, nil, &resp)

	return resp.Data, err
}

// ListDeletedGlobalResources fetches deleted global resources for cleanup
func (c *Client) ListDeletedGlobalResources(ctx context.Context, limit int) ([]models.GlobalResource, error) {
	var resp struct {
		Data []models.GlobalResource `json:"data"`
	}

	path := fmt.Sprintf("/int/api/v1/global-resources/deleted?limit=%d", limit)
	err := c.doRequest(ctx, "GET", path, nil, &resp)

	return resp.Data, err
}

// UpsertSyncedState updates the synced state for a global resource
func (c *Client) UpsertSyncedState(ctx context.Context, globalResourceID uuid.UUID, syncedGeneration int) error {
	path := fmt.Sprintf("/int/api/v1/global-resources/%s/synced-state", globalResourceID)
	req := map[string]int{"synced_generation": syncedGeneration}

	return c.doRequest(ctx, "POST", path, req, nil)
}

// DeleteSyncedState removes the synced state for a global resource
func (c *Client) DeleteSyncedState(ctx context.Context, globalResourceID uuid.UUID) error {
	path := fmt.Sprintf("/int/api/v1/global-resources/%s/synced-state", globalResourceID)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// CreateResourceRequest is the request body for CreateResource
type CreateResourceRequest struct {
	Namespace   string          `json:"namespace"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	APIVersion  string          `json:"api_version"`
	DesiredSpec json.RawMessage `json:"desired_spec"`
	Revision    int             `json:"revision"`
}

// CreateResource creates a new resource for the cluster
func (c *Client) CreateResource(ctx context.Context, req *CreateResourceRequest) (*models.Resource, error) {
	var resp struct {
		Data *models.Resource `json:"data"`
	}

	err := c.doRequest(ctx, "POST", "/int/api/v1/resources", req, &resp)

	return resp.Data, err
}

// SoftDeleteResourceByKey soft-deletes a resource by its key (namespace, kind, name)
func (c *Client) SoftDeleteResourceByKey(ctx context.Context, namespace, kind, name string) error {
	req := map[string]string{
		"namespace": namespace,
		"kind":      kind,
		"name":      name,
	}

	return c.doRequest(ctx, "DELETE", "/int/api/v1/resources/by-key", req, nil)
}
