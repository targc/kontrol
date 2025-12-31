package global_syncer

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
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

	g.sync(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[GlobalSyncer] Stopping global resource sync loop")
			return
		case <-time.After(10 * time.Second):
			g.sync(ctx)
		}
	}
}

func (g *GlobalSyncer) sync(ctx context.Context) {
	var globalResources []models.GlobalResource

	err := g.DB.
		WithContext(ctx).
		Where("deleted_at IS NULL").
		Find(&globalResources).
		Error

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to fetch global resources: %v", err)
		return
	}

	for _, gr := range globalResources {
		g.syncGlobalResource(ctx, &gr)
	}

	var deletedGlobalResources []models.GlobalResource

	err = g.DB.
		WithContext(ctx).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Find(&deletedGlobalResources).
		Error

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to fetch deleted global resources: %v", err)
		return
	}

	for _, gr := range deletedGlobalResources {
		g.cleanupDeletedGlobalResource(ctx, &gr)
	}
}

func (g *GlobalSyncer) syncGlobalResource(ctx context.Context, gr *models.GlobalResource) {
	tx := g.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var syncedState models.GlobalResourceSyncedState

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("global_resource_id = ? AND cluster_id = ?", gr.ID, g.ClusterID).
		First(&syncedState).
		Error

	if err == gorm.ErrRecordNotFound {
		err = g.createResourceForCluster(tx, gr)

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to create resource for global resource %d: %v", gr.ID, err)
			return
		}

		err = tx.
			Create(&models.GlobalResourceSyncedState{
				ID:               uuid.Must(uuid.NewV7()),
				GlobalResourceID: gr.ID,
				ClusterID:        g.ClusterID,
				SyncedGeneration: gr.Generation,
			}).
			Error

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to create synced state for global resource %d: %v", gr.ID, err)
			return
		}

		err = tx.Commit().Error

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to commit transaction for global resource %d: %v", gr.ID, err)
			return
		}

		log.Printf("[GlobalSyncer] Created resource for global resource %d in cluster %s", gr.ID, g.ClusterID)
		return
	}

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to query synced state for global resource %d: %v", gr.ID, err)
		return
	}

	if syncedState.SyncedGeneration < gr.Generation {
		err = g.updateResourceForCluster(tx, gr)

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to update resource for global resource %d: %v", gr.ID, err)
			return
		}

		err = tx.
			Model(&syncedState).
			Update("synced_generation", gr.Generation).
			Error

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to update synced state for global resource %d: %v", gr.ID, err)
			return
		}

		err = tx.Commit().Error

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to commit transaction for global resource %d: %v", gr.ID, err)
			return
		}

		log.Printf("[GlobalSyncer] Updated resource for global resource %d in cluster %s (gen %d -> %d)",
			gr.ID, g.ClusterID, syncedState.SyncedGeneration, gr.Generation)
		return
	}

	err = tx.Commit().Error

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to commit transaction for global resource %d: %v", gr.ID, err)
	}
}

func (g *GlobalSyncer) createResourceForCluster(tx *gorm.DB, gr *models.GlobalResource) error {
	resource := &models.Resource{
		ID:          uuid.Must(uuid.NewV7()),
		ClusterID:   g.ClusterID,
		Namespace:   gr.Namespace,
		Kind:        gr.Kind,
		Name:        gr.Name,
		APIVersion:  gr.APIVersion,
		DesiredSpec: gr.DesiredSpec,
		Revision:    gr.Revision,
	}

	err := tx.
		Create(resource).
		Error

	if err != nil {
		return err
	}

	return nil
}

func (g *GlobalSyncer) updateResourceForCluster(tx *gorm.DB, gr *models.GlobalResource) error {
	var resource models.Resource

	err := tx.
		Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?",
			g.ClusterID, gr.Namespace, gr.Kind, gr.Name).
		First(&resource).
		Error

	if err == gorm.ErrRecordNotFound {
		return g.createResourceForCluster(tx, gr)
	}

	if err != nil {
		return err
	}

	err = tx.
		Model(&resource).
		Updates(map[string]interface{}{
			"desired_spec": gr.DesiredSpec,
			"revision":     gr.Revision,
		}).
		Error

	if err != nil {
		return err
	}

	return nil
}

func (g *GlobalSyncer) cleanupDeletedGlobalResource(ctx context.Context, gr *models.GlobalResource) {
	var resource models.Resource

	err := g.DB.
		WithContext(ctx).
		Where("cluster_id = ? AND namespace = ? AND kind = ? AND name = ?",
			g.ClusterID, gr.Namespace, gr.Kind, gr.Name).
		First(&resource).
		Error

	if err == gorm.ErrRecordNotFound {
		err = g.DB.
			WithContext(ctx).
			Unscoped().
			Where("global_resource_id = ? AND cluster_id = ?", gr.ID, g.ClusterID).
			Delete(&models.GlobalResourceSyncedState{}).
			Error

		if err != nil {
			log.Printf("[GlobalSyncer] Failed to delete synced state for global resource %d: %v", gr.ID, err)
		}

		return
	}

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to find resource for deleted global resource %d: %v", gr.ID, err)
		return
	}

	err = g.DB.
		WithContext(ctx).
		Delete(&resource).
		Error

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to delete resource for global resource %d: %v", gr.ID, err)
		return
	}

	err = g.DB.
		WithContext(ctx).
		Unscoped().
		Where("global_resource_id = ? AND cluster_id = ?", gr.ID, g.ClusterID).
		Delete(&models.GlobalResourceSyncedState{}).
		Error

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to delete synced state for global resource %d: %v", gr.ID, err)
		return
	}

	log.Printf("[GlobalSyncer] Cleaned up resource for deleted global resource %d in cluster %s", gr.ID, g.ClusterID)
}
