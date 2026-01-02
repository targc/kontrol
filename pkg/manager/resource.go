package manager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
)

// ResourceManager provides programmatic CRUD operations for resources
type ResourceManager struct {
	DB *gorm.DB
}

// NewResourceManager creates a new ResourceManager
func NewResourceManager(db *gorm.DB) *ResourceManager {
	return &ResourceManager{DB: db}
}

// Create creates a new resource atomically
func (m *ResourceManager) Create(ctx context.Context, req CreateResourceRequest) (*ResourceWithState, error) {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	resource := models.Resource{
		ID:          uuid.Must(uuid.NewV7()),
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Generation:  1,
		Revision:    1,
	}

	err := tx.
		Create(&resource).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	err = tx.Commit().Error

	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(ctx, resource.ID)
}

// Get retrieves a resource by ID with its applied and current states
func (m *ResourceManager) Get(ctx context.Context, id uuid.UUID) (*ResourceWithState, error) {
	var resource models.Resource

	err := m.DB.
		WithContext(ctx).
		First(&resource, id).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	var appliedState models.ResourceAppliedState

	m.DB.
		WithContext(ctx).
		Where("resource_id = ?", resource.ID).
		First(&appliedState)

	var currentState models.ResourceCurrentState

	m.DB.
		WithContext(ctx).
		Where("resource_id = ?", resource.ID).
		First(&currentState)

	result := &ResourceWithState{
		Resource: resource,
	}

	if appliedState.ID != uuid.Nil {
		result.AppliedState = &appliedState
	}

	if currentState.ID != uuid.Nil {
		result.CurrentState = &currentState
	}

	return result, nil
}

// List retrieves all resources for a cluster with their states
func (m *ResourceManager) List(ctx context.Context, clusterID string) ([]*ResourceWithState, error) {
	var resources []models.Resource

	query := m.DB.
		WithContext(ctx).
		Model(&models.Resource{})

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	err := query.
		Find(&resources).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	result := make([]*ResourceWithState, len(resources))
	for i, r := range resources {
		var appliedState models.ResourceAppliedState

		m.DB.
			WithContext(ctx).
			Where("resource_id = ?", r.ID).
			First(&appliedState)

		var currentState models.ResourceCurrentState

		m.DB.
			WithContext(ctx).
			Where("resource_id = ?", r.ID).
			First(&currentState)

		result[i] = &ResourceWithState{
			Resource: r,
		}

		if appliedState.ID != uuid.Nil {
			result[i].AppliedState = &appliedState
		}

		if currentState.ID != uuid.Nil {
			result[i].CurrentState = &currentState
		}
	}

	return result, nil
}

// Update updates a resource's desired spec (generation auto-increments via DB trigger)
func (m *ResourceManager) Update(ctx context.Context, id uuid.UUID, desiredSpec json.RawMessage, revision *int) (*ResourceWithState, error) {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var resource models.Resource

	err := tx.
		First(&resource, id).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	updates := map[string]interface{}{
		"desired_spec": desiredSpec,
	}

	if revision != nil {
		updates["revision"] = *revision
	} else {
		updates["revision"] = resource.Revision + 1
	}

	err = tx.
		Model(&resource).
		Updates(updates).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	err = tx.Commit().Error

	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(ctx, id)
}

// Delete soft-deletes a resource atomically (generation auto-increments via DB trigger)
func (m *ResourceManager) Delete(ctx context.Context, id uuid.UUID) error {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var resource models.Resource

	err := tx.
		First(&resource, id).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("resource not found")
		}
		return fmt.Errorf("failed to get resource: %w", err)
	}

	err = tx.
		Delete(&resource).
		Error

	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	err = tx.Commit().Error

	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateFromTemplate creates a resource from a template
func (m *ResourceManager) CreateFromTemplate(ctx context.Context, clusterID string, tmpl Template) (*ResourceWithState, error) {
	kind, apiVersion, namespace, name, spec, err := tmpl.Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build template %s: %w", tmpl.TemplateName(), err)
	}

	return m.Create(ctx, CreateResourceRequest{
		ClusterID:   clusterID,
		Namespace:   namespace,
		Kind:        kind,
		Name:        name,
		APIVersion:  apiVersion,
		DesiredSpec: spec,
	})
}

// UpdateFromTemplate updates a resource from a template
func (m *ResourceManager) UpdateFromTemplate(ctx context.Context, id uuid.UUID, tmpl Template) (*ResourceWithState, error) {
	_, _, _, _, spec, err := tmpl.Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build template %s: %w", tmpl.TemplateName(), err)
	}

	return m.Update(ctx, id, spec, nil)
}

// GetByKey retrieves a resource by its unique key (cluster_id, namespace, kind, name)
func (m *ResourceManager) GetByKey(ctx context.Context, clusterID, namespace, kind, name string) (*ResourceWithState, error) {
	var resource models.Resource

	err := m.DB.
		WithContext(ctx).
		Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?",
			clusterID, namespace, kind, name).
		First(&resource).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return m.Get(ctx, resource.ID)
}

// Upsert creates or updates a resource atomically using INSERT ON CONFLICT
func (m *ResourceManager) Upsert(ctx context.Context, req CreateResourceRequest) (*ResourceWithState, error) {
	resource := models.Resource{
		ID:          uuid.Must(uuid.NewV7()),
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Generation:  1,
		Revision:    1,
	}

	err := m.DB.
		WithContext(ctx).
		Exec(`
			INSERT INTO k_resources (id, cluster_id, namespace, kind, name, api_version, desired_spec, generation, revision, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (cluster_id, namespace, kind, name) WHERE deleted_at IS NULL
			DO UPDATE SET
				api_version = EXCLUDED.api_version,
				desired_spec = EXCLUDED.desired_spec,
				revision = k_resources.revision + 1,
				updated_at = NOW()
		`, resource.ID, resource.ClusterID, resource.Namespace, resource.Kind, resource.Name,
			resource.APIVersion, resource.DesiredSpec, resource.Generation, resource.Revision).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to upsert resource: %w", err)
	}

	return m.GetByKey(ctx, req.ClusterID, req.Namespace, req.Kind, req.Name)
}

// UpsertFromTemplate creates or updates a resource from a template
func (m *ResourceManager) UpsertFromTemplate(ctx context.Context, clusterID string, tmpl Template) (*ResourceWithState, error) {
	kind, apiVersion, namespace, name, spec, err := tmpl.Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build template %s: %w", tmpl.TemplateName(), err)
	}

	return m.Upsert(ctx, CreateResourceRequest{
		ClusterID:   clusterID,
		Namespace:   namespace,
		Kind:        kind,
		Name:        name,
		APIVersion:  apiVersion,
		DesiredSpec: spec,
	})
}
