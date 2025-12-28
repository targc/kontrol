package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/targc/kontrol/pkg/k8s"
	"github.com/targc/kontrol/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type Reconciler struct {
	DB            *gorm.DB
	ClusterID     string
	DynamicClient dynamic.Interface
}

func NewReconciler(db *gorm.DB, clusterID, kubeconfig string) (*Reconciler, error) {
	config, err := k8s.BuildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Reconciler{
		DB:            db,
		ClusterID:     clusterID,
		DynamicClient: dynamicClient,
	}, nil
}

func (r *Reconciler) Start(ctx context.Context) {
	log.Println("[Reconciler] Starting reconciliation loop for cluster:", r.ClusterID)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Reconciler] Stopping reconciliation loop")
			return
		default:
			r.reconcile(ctx)
			time.Sleep(30 * time.Second)
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) {
	// Process active resources (deleted_at IS NULL)
	var activeResources []models.Resource
	r.DB.Where("cluster_id = ? AND deleted_at IS NULL", r.ClusterID).Find(&activeResources)

	for _, resource := range activeResources {
		go r.reconcileResource(ctx, &resource)
	}

	// Process soft-deleted resources (need cleanup)
	var deletedResources []models.Resource
	r.DB.Unscoped().Where("cluster_id = ? AND deleted_at IS NOT NULL", r.ClusterID).Find(&deletedResources)

	for _, resource := range deletedResources {
		go r.deleteResource(ctx, &resource)
	}
}

func (r *Reconciler) reconcileResource(ctx context.Context, resource *models.Resource) {
	tx := r.DB.Begin()
	defer tx.Rollback()

	var appliedState models.ResourceAppliedState
	tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		FirstOrCreate(&appliedState, models.ResourceAppliedState{ResourceID: resource.ID})

	if appliedState.Generation == resource.Generation {
		tx.Commit()
		return
	}

	log.Printf("[Reconciler] Reconciling resource %d (gen=%d, rev=%d)", resource.ID, resource.Generation, resource.Revision)

	var spec map[string]interface{}
	json.Unmarshal(resource.DesiredSpec, &spec)

	obj := &unstructured.Unstructured{Object: spec}
	obj.SetName(resource.Name)
	obj.SetNamespace(resource.Namespace)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["kontrol/resource-id"] = fmt.Sprintf("%d", resource.ID)
	annotations["kontrol/generation"] = fmt.Sprintf("%d", resource.Generation)
	annotations["kontrol/revision"] = fmt.Sprintf("%d", resource.Revision)
	obj.SetAnnotations(annotations)

	gvr := k8s.GetGVR(resource.Kind, resource.APIVersion)

	patchData, err := json.Marshal(obj)
	if err != nil {
		log.Printf("[Reconciler] Failed to marshal resource %d: %v", resource.ID, err)
		tx.Model(&appliedState).Updates(map[string]interface{}{
			"status":        "error",
			"error_message": err.Error(),
		})
		tx.Commit()
		return
	}

	_, err = r.DynamicClient.Resource(gvr).Namespace(resource.Namespace).Patch(
		ctx,
		resource.Name,
		types.ApplyPatchType,
		patchData,
		metav1.PatchOptions{
			FieldManager: "kontrol",
			Force:        func() *bool { b := true; return &b }(),
		},
	)

	if err != nil {
		log.Printf("[Reconciler] Failed to apply resource %d: %v", resource.ID, err)
		errMsg := err.Error()
		tx.Model(&appliedState).Updates(map[string]interface{}{
			"status":        "error",
			"error_message": &errMsg,
		})
		tx.Commit()
		return
	}

	resultBytes, _ := json.Marshal(obj)

	tx.Model(&appliedState).Updates(map[string]interface{}{
		"spec":          resultBytes,
		"generation":    resource.Generation,
		"revision":      resource.Revision,
		"status":        "success",
		"error_message": nil,
	})

	tx.Commit()

	log.Printf("[Reconciler] Successfully applied resource %d (gen=%d, rev=%d)", resource.ID, resource.Generation, resource.Revision)
}

func (r *Reconciler) deleteResource(ctx context.Context, resource *models.Resource) {
	log.Printf("[Reconciler] Deleting resource %d from K8s", resource.ID)

	gvr := k8s.GetGVR(resource.Kind, resource.APIVersion)

	// Delete from K8s
	err := r.DynamicClient.Resource(gvr).Namespace(resource.Namespace).
		Delete(ctx, resource.Name, metav1.DeleteOptions{})

	if err != nil && !errors.IsNotFound(err) {
		log.Printf("[Reconciler] Failed to delete resource %d from K8s: %v", resource.ID, err)
		return
	}

	// Hard delete from DB (CASCADE will delete applied_states and current_states)
	r.DB.Unscoped().Delete(resource)

	log.Printf("[Reconciler] Successfully deleted resource %d", resource.ID)
}
