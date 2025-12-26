package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type Watcher struct {
	DB            *gorm.DB
	ClusterID     string
	DynamicClient dynamic.Interface
}

func NewWatcher(db *gorm.DB, clusterID, kubeconfig string) (*Watcher, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
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
	log.Println("[Watcher] Starting watch for cluster:", w.ClusterID)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Watcher] Stopping watch")
			return
		case <-ticker.C:
			w.watchResources(ctx)
		}
	}
}

func (w *Watcher) watchResources(ctx context.Context) {
	var resources []models.Resource
	w.DB.Where("cluster_id = ?", w.ClusterID).Find(&resources)

	for _, resource := range resources {
		go w.watchResource(ctx, &resource)
	}
}

func (w *Watcher) watchResource(ctx context.Context, resource *models.Resource) {
	gvr := w.getGVR(resource.Kind, resource.APIVersion)

	watcher, err := w.DynamicClient.Resource(gvr).Namespace(resource.Namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", resource.Name),
		TimeoutSeconds: func() *int64 { t := int64(30); return &t }(),
	})
	if err != nil {
		log.Printf("[Watcher] Failed to watch resource %d: %v", resource.ID, err)
		return
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified, watch.Added:
			w.handleEvent(event, resource.ID)
		case watch.Deleted:
			w.handleDeleteEvent(resource.ID)
		}
	}
}

func (w *Watcher) handleEvent(event watch.Event, resourceID uint) {
	obj, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		return
	}

	annotations := obj.GetAnnotations()
	kontrolGeneration := annotations["kontrol/generation"]
	kontrolRevision := annotations["kontrol/revision"]
	k8sResourceVersion := obj.GetResourceVersion()

	if kontrolGeneration == "" {
		log.Printf("[Watcher] Missing kontrol annotations on resource %d", resourceID)
		return
	}

	generation, _ := strconv.Atoi(kontrolGeneration)
	revision, _ := strconv.Atoi(kontrolRevision)

	tx := w.DB.Begin()
	defer tx.Rollback()

	var currentState models.ResourceCurrentState
	tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		FirstOrCreate(&currentState, models.ResourceCurrentState{ResourceID: resourceID})

	if currentState.K8sResourceVersion == k8sResourceVersion {
		tx.Commit()
		return
	}

	specBytes, _ := json.Marshal(obj.Object["spec"])

	tx.Model(&currentState).Updates(map[string]interface{}{
		"spec":                 specBytes,
		"generation":           generation,
		"revision":             revision,
		"k8s_resource_version": k8sResourceVersion,
	})

	tx.Commit()

	log.Printf("[Watcher] Updated current_state for resource %d (gen=%d, rev=%d)", resourceID, generation, revision)
}

func (w *Watcher) handleDeleteEvent(resourceID uint) {
	log.Printf("[Watcher] Resource %d deleted from K8s, removing current_state", resourceID)

	w.DB.Unscoped().Where("resource_id = ?", resourceID).Delete(&models.ResourceCurrentState{})
}

func (w *Watcher) getGVR(kind, apiVersion string) schema.GroupVersionResource {
	mapping := map[string]schema.GroupVersionResource{
		"Deployment": {Group: "apps", Version: "v1", Resource: "deployments"},
		"Service":    {Version: "v1", Resource: "services"},
		"ConfigMap":  {Version: "v1", Resource: "configmaps"},
		"Pod":        {Version: "v1", Resource: "pods"},
	}

	if gvr, ok := mapping[kind]; ok {
		return gvr
	}

	return schema.GroupVersionResource{Resource: kind}
}
