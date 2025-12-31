package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/targc/kontrol/pkg/k8s"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

type Watcher struct {
	DB            *gorm.DB
	ClusterID     string
	DynamicClient dynamic.Interface
}

func NewWatcher(db *gorm.DB, clusterID, kubeconfig string) (*Watcher, error) {
	config, err := k8s.BuildConfig(kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)

	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Watcher{
		DB:            db,
		ClusterID:     clusterID,
		DynamicClient: dynamicClient,
	}, nil
}

func (w *Watcher) Start(ctx context.Context) {
	log.Println("[Watcher] Starting watches for cluster:", w.ClusterID)

	var wg sync.WaitGroup

	for _, gvr := range k8s.SupportedGVRs {
		wg.Add(1)

		go func(gvr schema.GroupVersionResource) {
			defer wg.Done()
			w.watchGVR(ctx, gvr)
		}(gvr)
	}

	wg.Wait()
	log.Println("[Watcher] All watches stopped")
}

func (w *Watcher) watchGVR(ctx context.Context, gvr schema.GroupVersionResource) {
	log.Printf("[Watcher] Starting watch for %s", gvr.Resource)

	for {
		watcher, err := w.DynamicClient.Resource(gvr).Watch(ctx, metav1.ListOptions{})

		if err != nil {
			if ctx.Err() != nil {
				return
			}

			log.Printf("[Watcher] Failed to watch %s: %v, retrying...", gvr.Resource, err)
			time.Sleep(5 * time.Second)
			continue
		}

		for event := range watcher.ResultChan() {
			w.handleEvent(ctx, event)
		}

		if ctx.Err() != nil {
			return
		}

		log.Printf("[Watcher] Watch %s disconnected, reconnecting...", gvr.Resource)
	}
}

func (w *Watcher) handleEvent(ctx context.Context, event watch.Event) {
	obj, ok := event.Object.(*unstructured.Unstructured)

	if !ok {
		return
	}

	annotations := obj.GetAnnotations()
	resourceIDStr := annotations["kontrol/resource-id"]

	if resourceIDStr == "" {
		return
	}

	resourceID, err := uuid.Parse(resourceIDStr)

	if err != nil {
		return
	}

	switch event.Type {
	case watch.Added, watch.Modified:
		w.upsertCurrentState(ctx, resourceID, obj)
	case watch.Deleted:
		w.deleteCurrentState(ctx, resourceID)
	}
}

func (w *Watcher) upsertCurrentState(ctx context.Context, resourceID uuid.UUID, obj *unstructured.Unstructured) {
	annotations := obj.GetAnnotations()
	kontrolGeneration := annotations["kontrol/generation"]
	kontrolRevision := annotations["kontrol/revision"]
	k8sResourceVersion := obj.GetResourceVersion()

	if kontrolGeneration == "" {
		log.Printf("[Watcher] Missing kontrol/generation annotation on resource %s", resourceID)
		return
	}

	generation, _ := strconv.Atoi(kontrolGeneration)
	revision, _ := strconv.Atoi(kontrolRevision)

	tx := w.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var currentState models.ResourceCurrentState

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("resource_id = ?", resourceID).
		First(&currentState).
		Error

	if err == gorm.ErrRecordNotFound {
		currentState = models.ResourceCurrentState{
			ID:         uuid.Must(uuid.NewV7()),
			ResourceID: resourceID,
		}

		err = tx.Create(&currentState).Error
	}

	if err != nil {
		log.Printf("[Watcher] Failed to get/create current_state for resource %s: %v", resourceID, err)
		return
	}

	if currentState.K8sResourceVersion == k8sResourceVersion {
		err = tx.Commit().Error

		if err != nil {
			log.Printf("[Watcher] Failed to commit transaction for resource %s: %v", resourceID, err)
		}

		return
	}

	specBytes, err := json.Marshal(obj.Object["spec"])

	if err != nil {
		log.Printf("[Watcher] Failed to marshal spec for resource %s: %v", resourceID, err)
		return
	}

	err = tx.
		Model(&currentState).
		Updates(map[string]interface{}{
			"spec":                 specBytes,
			"generation":           generation,
			"revision":             revision,
			"k8s_resource_version": k8sResourceVersion,
		}).
		Error

	if err != nil {
		log.Printf("[Watcher] Failed to update current_state for resource %s: %v", resourceID, err)
		return
	}

	err = tx.Commit().Error

	if err != nil {
		log.Printf("[Watcher] Failed to commit transaction for resource %s: %v", resourceID, err)
		return
	}

	log.Printf("[Watcher] Updated current_state for resource %s (gen=%d, rev=%d)", resourceID, generation, revision)
}

func (w *Watcher) deleteCurrentState(ctx context.Context, resourceID uuid.UUID) {
	log.Printf("[Watcher] Resource %s deleted from K8s, removing current_state", resourceID)

	err := w.DB.
		WithContext(ctx).
		Unscoped().
		Where("resource_id = ?", resourceID).
		Delete(&models.ResourceCurrentState{}).
		Error

	if err != nil {
		log.Printf("[Watcher] Failed to delete current_state for resource %s: %v", resourceID, err)
	}
}
