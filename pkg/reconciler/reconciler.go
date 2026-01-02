package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
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
			time.Sleep(10 * time.Second)
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) {
	var activeResources []models.Resource

	err := r.DB.
		WithContext(ctx).
		Where("cluster_id = ? AND deleted_at IS NULL", r.ClusterID).
		Find(&activeResources).
		Error

	if err != nil {
		log.Printf("[Reconciler] Failed to fetch active resources: %v", err)
		return
	}

	for _, resource := range activeResources {
		go r.reconcileResource(ctx, &resource)
	}

	var deletedResources []models.Resource

	err = r.DB.
		WithContext(ctx).
		Unscoped().
		Where("cluster_id = ? AND deleted_at IS NOT NULL", r.ClusterID).
		Find(&deletedResources).
		Error

	if err != nil {
		log.Printf("[Reconciler] Failed to fetch deleted resources: %v", err)
		return
	}

	for _, resource := range deletedResources {
		go r.deleteResource(ctx, &resource)
	}
}

func (r *Reconciler) reconcileResource(ctx context.Context, resource *models.Resource) {
	if resource.ClusterID != r.ClusterID {
		log.Printf("[Reconciler] SECURITY: resource %s belongs to cluster %s, not %s - skipping",
			resource.ID, resource.ClusterID, r.ClusterID)
		return
	}

	tx := r.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	var appliedState models.ResourceAppliedState

	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("resource_id = ?", resource.ID).
		First(&appliedState).
		Error

	if err == gorm.ErrRecordNotFound {
		appliedState = models.ResourceAppliedState{
			ID:         uuid.Must(uuid.NewV7()),
			ResourceID: resource.ID,
		}

		err = tx.Create(&appliedState).Error
	}

	if err != nil {
		log.Printf("[Reconciler] Failed to get/create applied_state for resource %s: %v", resource.ID, err)
		return
	}

	if appliedState.Generation == resource.Generation {
		err = tx.Commit().Error

		if err != nil {
			log.Printf("[Reconciler] Failed to commit transaction for resource %s: %v", resource.ID, err)
		}

		return
	}

	log.Printf("[Reconciler] Reconciling resource %s (gen=%d, rev=%d)", resource.ID, resource.Generation, resource.Revision)

	var spec map[string]interface{}
	json.Unmarshal(resource.DesiredSpec, &spec)

	obj := &unstructured.Unstructured{Object: spec}
	obj.SetAPIVersion(resource.APIVersion)
	obj.SetKind(resource.Kind)
	obj.SetName(resource.Name)
	obj.SetNamespace(resource.Namespace)

	annotations := obj.GetAnnotations()

	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations["kontrol/resource-id"] = resource.ID.String()
	annotations["kontrol/generation"] = fmt.Sprintf("%d", resource.Generation)
	annotations["kontrol/revision"] = fmt.Sprintf("%d", resource.Revision)
	obj.SetAnnotations(annotations)

	gvr := k8s.GetGVR(resource.Kind, resource.APIVersion)

	patchData, err := json.Marshal(obj)

	if err != nil {
		log.Printf("[Reconciler] Failed to marshal resource %s: %v", resource.ID, err)

		updateErr := tx.
			Model(&appliedState).
			Updates(map[string]interface{}{
				"status":        "error",
				"error_message": err.Error(),
			}).
			Error

		if updateErr != nil {
			log.Printf("[Reconciler] Failed to update applied_state for resource %s: %v", resource.ID, updateErr)
			return
		}

		commitErr := tx.Commit().Error

		if commitErr != nil {
			log.Printf("[Reconciler] Failed to commit transaction for resource %s: %v", resource.ID, commitErr)
		}

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
		log.Printf("[Reconciler] Failed to apply resource %s: %v", resource.ID, err)
		errMsg := err.Error()

		updateErr := tx.
			Model(&appliedState).
			Updates(map[string]interface{}{
				"status":        "error",
				"error_message": &errMsg,
			}).
			Error

		if updateErr != nil {
			log.Printf("[Reconciler] Failed to update applied_state for resource %s: %v", resource.ID, updateErr)
			return
		}

		commitErr := tx.Commit().Error

		if commitErr != nil {
			log.Printf("[Reconciler] Failed to commit transaction for resource %s: %v", resource.ID, commitErr)
		}

		return
	}

	resultBytes, _ := json.Marshal(obj)

	err = tx.
		Model(&appliedState).
		Updates(map[string]interface{}{
			"spec":          resultBytes,
			"generation":    resource.Generation,
			"revision":      resource.Revision,
			"status":        "success",
			"error_message": nil,
		}).
		Error

	if err != nil {
		log.Printf("[Reconciler] Failed to update applied_state for resource %s: %v", resource.ID, err)
		return
	}

	err = tx.Commit().Error

	if err != nil {
		log.Printf("[Reconciler] Failed to commit transaction for resource %s: %v", resource.ID, err)
		return
	}

	log.Printf("[Reconciler] Successfully applied resource %s (gen=%d, rev=%d)", resource.ID, resource.Generation, resource.Revision)
}

func (r *Reconciler) deleteResource(ctx context.Context, resource *models.Resource) {
	if resource.ClusterID != r.ClusterID {
		log.Printf("[Reconciler] SECURITY: resource %s belongs to cluster %s, not %s - skipping delete",
			resource.ID, resource.ClusterID, r.ClusterID)
		return
	}

	log.Printf("[Reconciler] Deleting resource %s from K8s", resource.ID)

	gvr := k8s.GetGVR(resource.Kind, resource.APIVersion)

	err := r.DynamicClient.Resource(gvr).Namespace(resource.Namespace).
		Delete(ctx, resource.Name, metav1.DeleteOptions{})

	if err != nil && !errors.IsNotFound(err) {
		log.Printf("[Reconciler] Failed to delete resource %s from K8s: %v", resource.ID, err)
		return
	}

	err = r.DB.
		WithContext(ctx).
		Unscoped().
		Delete(resource).
		Error

	if err != nil {
		log.Printf("[Reconciler] Failed to delete resource %s from DB: %v", resource.ID, err)
		return
	}

	log.Printf("[Reconciler] Successfully deleted resource %s", resource.ID)
}
