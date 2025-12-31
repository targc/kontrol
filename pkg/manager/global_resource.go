package manager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
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
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Generation:  1,
		Revision:    1,
	}

	if err := tx.Create(&globalResource).Error; err != nil {
		return nil, fmt.Errorf("failed to create global resource: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(ctx, globalResource.ID)
}

// Get retrieves a global resource by ID with its sync status
func (m *GlobalResourceManager) Get(ctx context.Context, id uint) (*GlobalResourceWithSyncStatus, error) {
	var globalResource models.GlobalResource
	if err := m.DB.WithContext(ctx).First(&globalResource, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("global resource not found")
		}
		return nil, fmt.Errorf("failed to get global resource: %w", err)
	}

	return m.buildGlobalResourceWithSyncStatus(ctx, &globalResource)
}

// List retrieves all global resources with their sync status
func (m *GlobalResourceManager) List(ctx context.Context) ([]*GlobalResourceWithSyncStatus, error) {
	var globalResources []models.GlobalResource
	if err := m.DB.WithContext(ctx).Find(&globalResources).Error; err != nil {
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
func (m *GlobalResourceManager) Update(ctx context.Context, id uint, desiredSpec json.RawMessage, revision *int) (*GlobalResourceWithSyncStatus, error) {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var globalResource models.GlobalResource
	if err := tx.First(&globalResource, id).Error; err != nil {
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

	if err := tx.Model(&globalResource).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update global resource: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(ctx, id)
}

// Delete soft-deletes a global resource (generation auto-increments via DB trigger)
func (m *GlobalResourceManager) Delete(ctx context.Context, id uint) error {
	tx := m.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var globalResource models.GlobalResource
	if err := tx.First(&globalResource, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("global resource not found")
		}
		return fmt.Errorf("failed to get global resource: %w", err)
	}

	if err := tx.Delete(&globalResource).Error; err != nil {
		return fmt.Errorf("failed to delete global resource: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// buildGlobalResourceWithSyncStatus builds a GlobalResourceWithSyncStatus from a GlobalResource
func (m *GlobalResourceManager) buildGlobalResourceWithSyncStatus(ctx context.Context, gr *models.GlobalResource) (*GlobalResourceWithSyncStatus, error) {
	// Get total clusters
	var totalClusters int64
	if err := m.DB.WithContext(ctx).Model(&models.Cluster{}).Count(&totalClusters).Error; err != nil {
		return nil, fmt.Errorf("failed to count clusters: %w", err)
	}

	// Get synced states for this global resource
	var syncedStates []models.GlobalResourceSyncedState
	m.DB.WithContext(ctx).Where("global_resource_id = ?", gr.ID).Find(&syncedStates)

	// Build cluster statuses
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
