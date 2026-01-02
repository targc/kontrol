package manager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GlobalResourceManager provides programmatic CRUD operations for global resources
type GlobalResourceManager struct {
	DB *gorm.DB
}

// NewGlobalResourceManager creates a new GlobalResourceManager
func NewGlobalResourceManager(db *gorm.DB) *GlobalResourceManager {
	return &GlobalResourceManager{DB: db}
}

// Create creates a new global resource
func (m *GlobalResourceManager) Create(ctx context.Context, req CreateGlobalResourceRequest) (*GlobalResourceWithSyncStatus, error) {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	globalResource := models.GlobalResource{
		ID:          uuid.Must(uuid.NewV7()),
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Generation:  1,
		Revision:    1,
	}

	err := tx.
		Create(&globalResource).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to create global resource: %w", err)
	}

	err = tx.Commit().Error

	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(ctx, globalResource.ID)
}

// Get retrieves a global resource by ID with its sync status
func (m *GlobalResourceManager) Get(ctx context.Context, id uuid.UUID) (*GlobalResourceWithSyncStatus, error) {
	var globalResource models.GlobalResource

	err := m.DB.
		WithContext(ctx).
		First(&globalResource, id).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("global resource not found")
		}
		return nil, fmt.Errorf("failed to get global resource: %w", err)
	}

	return m.buildGlobalResourceWithSyncStatus(ctx, &globalResource)
}

// GetByKindAndName retrieves a global resource by namespace, kind, and name
func (m *GlobalResourceManager) GetByKindAndName(ctx context.Context, namespace, kind, name string) (*GlobalResourceWithSyncStatus, error) {
	var gr models.GlobalResource

	err := m.DB.
		WithContext(ctx).
		Where("namespace = ? AND kind = ? AND name = ?", namespace, kind, name).
		First(&gr).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to get global resource: %w", err)
	}

	return m.buildGlobalResourceWithSyncStatus(ctx, &gr)
}

// Upsert creates or updates a global resource atomically using INSERT ON CONFLICT
func (m *GlobalResourceManager) Upsert(ctx context.Context, req CreateGlobalResourceRequest) (*GlobalResourceWithSyncStatus, error) {
	globalResource := models.GlobalResource{
		ID:          uuid.Must(uuid.NewV7()),
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
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "namespace"},
				{Name: "kind"},
				{Name: "name"},
			},
			Where: clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "deleted_at IS NULL"},
				},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"api_version":  req.APIVersion,
				"desired_spec": req.DesiredSpec,
				"revision":     gorm.Expr("k_global_resources.revision + 1"),
			}),
		}).
		Create(&globalResource).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to upsert global resource: %w", err)
	}

	return m.GetByKindAndName(ctx, req.Namespace, req.Kind, req.Name)
}

// List retrieves all global resources with their sync status
func (m *GlobalResourceManager) List(ctx context.Context) ([]*GlobalResourceWithSyncStatus, error) {
	var globalResources []models.GlobalResource

	err := m.DB.
		WithContext(ctx).
		Find(&globalResources).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to list global resources: %w", err)
	}

	result := make([]*GlobalResourceWithSyncStatus, len(globalResources))
	for i, gr := range globalResources {
		status, err := m.buildGlobalResourceWithSyncStatus(ctx, &gr)

		if err != nil {
			return nil, err
		}

		result[i] = status
	}

	return result, nil
}

// Update updates a global resource's desired spec (generation auto-increments via DB trigger)
func (m *GlobalResourceManager) Update(ctx context.Context, id uuid.UUID, desiredSpec json.RawMessage, revision *int) (*GlobalResourceWithSyncStatus, error) {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var globalResource models.GlobalResource

	err := tx.
		First(&globalResource, id).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("global resource not found")
		}
		return nil, fmt.Errorf("failed to get global resource: %w", err)
	}

	updates := map[string]interface{}{
		"desired_spec": desiredSpec,
	}

	if revision != nil {
		updates["revision"] = *revision
	} else {
		updates["revision"] = globalResource.Revision + 1
	}

	err = tx.
		Model(&globalResource).
		Updates(updates).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to update global resource: %w", err)
	}

	err = tx.Commit().Error

	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(ctx, id)
}

// Delete soft-deletes a global resource (generation auto-increments via DB trigger)
func (m *GlobalResourceManager) Delete(ctx context.Context, id uuid.UUID) error {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var globalResource models.GlobalResource

	err := tx.
		First(&globalResource, id).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("global resource not found")
		}
		return fmt.Errorf("failed to get global resource: %w", err)
	}

	err = tx.
		Delete(&globalResource).
		Error

	if err != nil {
		return fmt.Errorf("failed to delete global resource: %w", err)
	}

	err = tx.Commit().Error

	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// buildGlobalResourceWithSyncStatus builds a GlobalResourceWithSyncStatus from a GlobalResource
func (m *GlobalResourceManager) buildGlobalResourceWithSyncStatus(ctx context.Context, gr *models.GlobalResource) (*GlobalResourceWithSyncStatus, error) {
	var totalClusters int64

	err := m.DB.
		WithContext(ctx).
		Model(&models.Cluster{}).
		Count(&totalClusters).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to count clusters: %w", err)
	}

	var syncedStates []models.GlobalResourceSyncedState

	m.DB.
		WithContext(ctx).
		Where("global_resource_id = ?", gr.ID).
		Find(&syncedStates)

	clusterStatuses := make([]ClusterSyncStatus, len(syncedStates))
	syncedCount := 0

	for i, state := range syncedStates {
		isSynced := state.SyncedGeneration == gr.Generation

		if isSynced {
			syncedCount++
		}

		clusterStatuses[i] = ClusterSyncStatus{
			ClusterID:        state.ClusterID,
			SyncedGeneration: state.SyncedGeneration,
			IsSynced:         isSynced,
		}
	}

	return &GlobalResourceWithSyncStatus{
		GlobalResource:  *gr,
		TotalClusters:   int(totalClusters),
		SyncedClusters:  syncedCount,
		ClusterStatuses: clusterStatuses,
	}, nil
}

// CreateFromTemplate creates a global resource from a template
func (m *GlobalResourceManager) CreateFromTemplate(ctx context.Context, tmpl Template) (*GlobalResourceWithSyncStatus, error) {
	kind, apiVersion, namespace, name, spec, err := tmpl.Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build template %s: %w", tmpl.TemplateName(), err)
	}

	return m.Create(ctx, CreateGlobalResourceRequest{
		Namespace:   namespace,
		Kind:        kind,
		Name:        name,
		APIVersion:  apiVersion,
		DesiredSpec: spec,
	})
}

// UpdateFromTemplate updates a global resource from a template
func (m *GlobalResourceManager) UpdateFromTemplate(ctx context.Context, id uuid.UUID, tmpl Template) (*GlobalResourceWithSyncStatus, error) {
	_, _, _, _, spec, err := tmpl.Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build template %s: %w", tmpl.TemplateName(), err)
	}

	return m.Update(ctx, id, spec, nil)
}

// UpsertFromTemplate creates or updates a global resource from a template
func (m *GlobalResourceManager) UpsertFromTemplate(ctx context.Context, tmpl Template) (*GlobalResourceWithSyncStatus, error) {
	kind, apiVersion, namespace, name, spec, err := tmpl.Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build template %s: %w", tmpl.TemplateName(), err)
	}

	return m.Upsert(ctx, CreateGlobalResourceRequest{
		Namespace:   namespace,
		Kind:        kind,
		Name:        name,
		APIVersion:  apiVersion,
		DesiredSpec: spec,
	})
}

// DecompileToTemplate decompiles a global resource's spec into a template
func (m *GlobalResourceManager) DecompileToTemplate(ctx context.Context, id uuid.UUID, tmpl Template) error {
	gr, err := m.Get(ctx, id)

	if err != nil {
		return err
	}

	return tmpl.Decompile(gr.GlobalResource.DesiredSpec)
}
