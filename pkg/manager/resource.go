package manager

import (
	"encoding/json"
	"fmt"

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
func (m *ResourceManager) Create(req CreateResourceRequest) (*ResourceWithState, error) {
	tx := m.DB.Begin()
	defer tx.Rollback()

	resource := models.Resource{
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		Kind:        req.Kind,
		Name:        req.Name,
		APIVersion:  req.APIVersion,
		DesiredSpec: req.DesiredSpec,
		Generation:  1,
		Revision:    1,
	}

	if err := tx.Create(&resource).Error; err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(resource.ID)
}

// Get retrieves a resource by ID with its applied and current states
func (m *ResourceManager) Get(id uint) (*ResourceWithState, error) {
	var resource models.Resource
	if err := m.DB.First(&resource, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	var appliedState models.ResourceAppliedState
	m.DB.Where("resource_id = ?", resource.ID).First(&appliedState)

	var currentState models.ResourceCurrentState
	m.DB.Where("resource_id = ?", resource.ID).First(&currentState)

	result := &ResourceWithState{
		Resource: resource,
	}

	if appliedState.ID != 0 {
		result.AppliedState = &appliedState
	}

	if currentState.ID != 0 {
		result.CurrentState = &currentState
	}

	return result, nil
}

// List retrieves all resources for a cluster with their states
func (m *ResourceManager) List(clusterID string) ([]*ResourceWithState, error) {
	var resources []models.Resource
	query := m.DB.Model(&models.Resource{})

	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if err := query.Find(&resources).Error; err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	result := make([]*ResourceWithState, len(resources))
	for i, r := range resources {
		var appliedState models.ResourceAppliedState
		m.DB.Where("resource_id = ?", r.ID).First(&appliedState)

		var currentState models.ResourceCurrentState
		m.DB.Where("resource_id = ?", r.ID).First(&currentState)

		result[i] = &ResourceWithState{
			Resource: r,
		}

		if appliedState.ID != 0 {
			result[i].AppliedState = &appliedState
		}

		if currentState.ID != 0 {
			result[i].CurrentState = &currentState
		}
	}

	return result, nil
}

// Update updates a resource's desired spec and increments generation atomically
func (m *ResourceManager) Update(id uint, desiredSpec json.RawMessage, revision *int) (*ResourceWithState, error) {
	tx := m.DB.Begin()
	defer tx.Rollback()

	var resource models.Resource
	if err := tx.First(&resource, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("resource not found")
		}
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	updates := map[string]interface{}{
		"desired_spec": desiredSpec,
		"generation":   resource.Generation + 1,
	}

	if revision != nil {
		updates["revision"] = *revision
	} else {
		updates["revision"] = resource.Revision + 1
	}

	if err := tx.Model(&resource).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return m.Get(id)
}

// Delete soft-deletes a resource atomically (increments generation then marks as deleted)
func (m *ResourceManager) Delete(id uint) error {
	tx := m.DB.Begin()
	defer tx.Rollback()

	var resource models.Resource
	if err := tx.First(&resource, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("resource not found")
		}
		return fmt.Errorf("failed to get resource: %w", err)
	}

	resource.Generation++

	if err := tx.Model(&resource).Update("generation", resource.Generation).Error; err != nil {
		return fmt.Errorf("failed to mark resource for deletion: %w", err)
	}

	if err := tx.Delete(&resource).Error; err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
