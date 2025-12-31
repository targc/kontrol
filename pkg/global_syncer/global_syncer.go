package global_syncer

import (
	"context"
	"log"
	"time"

	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GlobalSyncer struct {
	DB        *gorm.DB
	ClusterID string
}

func NewGlobalSyncer(db *gorm.DB, clusterID string) *GlobalSyncer {
	return &GlobalSyncer{
		DB:        db,
		ClusterID: clusterID,
	}
}

func (g *GlobalSyncer) Start(ctx context.Context) {
	log.Println("[GlobalSyncer] Starting global resource sync loop for cluster:", g.ClusterID)

	// Run immediately on start
	g.sync()

	for {
		select {
		case <-ctx.Done():
			log.Println("[GlobalSyncer] Stopping global resource sync loop")
			return
		case <-time.After(30 * time.Second):
			g.sync()
		}
	}
}

func (g *GlobalSyncer) sync() {
	// Sync active global resources
	var globalResources []models.GlobalResource
	g.DB.Where("deleted_at IS NULL").Find(&globalResources)

	for _, gr := range globalResources {
		g.syncGlobalResource(&gr)
	}

	// Handle deleted global resources
	var deletedGlobalResources []models.GlobalResource
	g.DB.Unscoped().Where("deleted_at IS NOT NULL").Find(&deletedGlobalResources)

	for _, gr := range deletedGlobalResources {
		g.cleanupDeletedGlobalResource(&gr)
	}
}

func (g *GlobalSyncer) syncGlobalResource(gr *models.GlobalResource) {
	tx := g.DB.Begin()
	defer tx.Rollback()

	// Check if synced state exists for this cluster
	var syncedState models.GlobalResourceSyncedState
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("global_resource_id = ? AND cluster_id = ?", gr.ID, g.ClusterID).
		First(&syncedState).Error

	if err == gorm.ErrRecordNotFound {
		// First time: create resource and synced state
		g.createResourceForCluster(tx, gr)
		tx.Create(&models.GlobalResourceSyncedState{
			GlobalResourceID: gr.ID,
			ClusterID:        g.ClusterID,
			SyncedGeneration: gr.Generation,
		})
		tx.Commit()
		log.Printf("[GlobalSyncer] Created resource for global resource %d in cluster %s", gr.ID, g.ClusterID)
		return
	}

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to query synced state for global resource %d: %v", gr.ID, err)
		return
	}

	// Check if out of sync
	if syncedState.SyncedGeneration < gr.Generation {
		g.updateResourceForCluster(tx, gr)
		tx.Model(&syncedState).Update("synced_generation", gr.Generation)
		tx.Commit()
		log.Printf("[GlobalSyncer] Updated resource for global resource %d in cluster %s (gen %d -> %d)",
			gr.ID, g.ClusterID, syncedState.SyncedGeneration, gr.Generation)
		return
	}

	tx.Commit()
}

func (g *GlobalSyncer) createResourceForCluster(tx *gorm.DB, gr *models.GlobalResource) {
	resource := &models.Resource{
		ClusterID:   g.ClusterID,
		Namespace:   gr.Namespace,
		Kind:        gr.Kind,
		Name:        gr.Name,
		APIVersion:  gr.APIVersion,
		DesiredSpec: gr.DesiredSpec,
		Revision:    gr.Revision,
	}
	tx.Create(resource)
}

func (g *GlobalSyncer) updateResourceForCluster(tx *gorm.DB, gr *models.GlobalResource) {
	// Find the resource by cluster_id, namespace, kind, name
	var resource models.Resource
	err := tx.Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?",
		g.ClusterID, gr.Namespace, gr.Kind, gr.Name).First(&resource).Error

	if err == gorm.ErrRecordNotFound {
		// Resource doesn't exist, create it
		g.createResourceForCluster(tx, gr)
		return
	}

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to find resource for global resource %d: %v", gr.ID, err)
		return
	}

	// Update the resource spec (this will trigger generation bump via DB trigger)
	tx.Model(&resource).Updates(map[string]interface{}{
		"desired_spec": gr.DesiredSpec,
		"revision":     gr.Revision,
	})
}

func (g *GlobalSyncer) cleanupDeletedGlobalResource(gr *models.GlobalResource) {
	// Find and soft-delete the corresponding resource
	var resource models.Resource
	err := g.DB.Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?",
		g.ClusterID, gr.Namespace, gr.Kind, gr.Name).First(&resource).Error

	if err == gorm.ErrRecordNotFound {
		// Resource already doesn't exist, just cleanup synced state
		g.DB.Unscoped().Where("global_resource_id = ? AND cluster_id = ?", gr.ID, g.ClusterID).
			Delete(&models.GlobalResourceSyncedState{})
		return
	}

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to find resource for deleted global resource %d: %v", gr.ID, err)
		return
	}

	// Soft-delete the resource (reconciler will delete from K8s)
	g.DB.Delete(&resource)

	// Cleanup synced state
	g.DB.Unscoped().Where("global_resource_id = ? AND cluster_id = ?", gr.ID, g.ClusterID).
		Delete(&models.GlobalResourceSyncedState{})

	log.Printf("[GlobalSyncer] Cleaned up resource for deleted global resource %d in cluster %s", gr.ID, g.ClusterID)
}
