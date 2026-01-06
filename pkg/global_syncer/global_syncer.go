package global_syncer

import (
	"context"
	"log"
	"time"

	"github.com/targc/kontrol/pkg/apiclient"
	"github.com/targc/kontrol/pkg/models"
)

type GlobalSyncer struct {
	Client    *apiclient.Client
	ClusterID string
}

func NewGlobalSyncer(client *apiclient.Client, clusterID string) *GlobalSyncer {
	return &GlobalSyncer{
		Client:    client,
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
	// Fetch out-of-sync global resources from API
	globalResources, err := g.Client.ListOutOfSyncGlobalResources(ctx, 100)

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to fetch out-of-sync global resources: %v", err)
		return
	}

	for _, gr := range globalResources {
		g.syncGlobalResource(ctx, &gr)
	}

	// Fetch deleted global resources from API
	deletedGlobalResources, err := g.Client.ListDeletedGlobalResources(ctx, 100)

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to fetch deleted global resources: %v", err)
		return
	}

	for _, gr := range deletedGlobalResources {
		g.cleanupDeletedGlobalResource(ctx, &gr)
	}
}

func (g *GlobalSyncer) syncGlobalResource(ctx context.Context, gr *apiclient.GlobalResourceForSync) {
	// Create resource for this cluster
	_, err := g.Client.CreateResource(ctx, &apiclient.CreateResourceRequest{
		Namespace:   gr.Namespace,
		Kind:        gr.Kind,
		Name:        gr.Name,
		APIVersion:  gr.APIVersion,
		DesiredSpec: gr.DesiredSpec,
		Revision:    gr.Revision,
	})

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to create resource for global resource %s: %v", gr.ID, err)
		return
	}

	// Update synced state
	err = g.Client.UpsertSyncedState(ctx, gr.ID, gr.Generation)

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to update synced state for global resource %s: %v", gr.ID, err)
		return
	}

	log.Printf("[GlobalSyncer] Synced global resource %s to cluster %s (gen=%d)", gr.ID, g.ClusterID, gr.Generation)
}

func (g *GlobalSyncer) cleanupDeletedGlobalResource(ctx context.Context, gr *models.GlobalResource) {
	// Soft-delete the resource for this cluster
	err := g.Client.SoftDeleteResourceByKey(ctx, gr.Namespace, gr.Kind, gr.Name)

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to delete resource for global resource %s: %v", gr.ID, err)
		return
	}

	// Delete synced state
	err = g.Client.DeleteSyncedState(ctx, gr.ID)

	if err != nil {
		log.Printf("[GlobalSyncer] Failed to delete synced state for global resource %s: %v", gr.ID, err)
		return
	}

	log.Printf("[GlobalSyncer] Cleaned up deleted global resource %s from cluster %s", gr.ID, g.ClusterID)
}
