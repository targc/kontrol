package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/targc/kontrol/pkg/apiclient"
	"github.com/targc/kontrol/pkg/k8s"
	"github.com/targc/kontrol/pkg/models"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type Reconciler struct {
	Client        *apiclient.Client
	ClusterID     string
	DynamicClient dynamic.Interface
}

func NewReconciler(client *apiclient.Client, clusterID, kubeconfig string) (*Reconciler, error) {
	config, err := k8s.BuildConfig(kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)

	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Reconciler{
		Client:        client,
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
	// Fetch out-of-sync resources from API
	outOfSyncResources, err := r.Client.ListOutOfSyncResources(ctx, 100)

	if err != nil {
		log.Printf("[Reconciler] Failed to fetch out-of-sync resources: %v", err)
		return
	}

	for _, resource := range outOfSyncResources {
		r.reconcileResource(ctx, &resource)
	}

	// Fetch deleted resources from API
	deletedResources, err := r.Client.ListDeletedResources(ctx, 100)

	if err != nil {
		log.Printf("[Reconciler] Failed to fetch deleted resources: %v", err)
		return
	}

	for _, resource := range deletedResources {
		r.deleteResource(ctx, &resource)
	}
}

func (r *Reconciler) reconcileResource(ctx context.Context, resource *models.Resource) {
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
		errMsg := err.Error()

		updateErr := r.Client.UpsertAppliedState(ctx, resource.ID, &apiclient.UpsertAppliedStateRequest{
			Status:       "error",
			ErrorMessage: &errMsg,
		})

		if updateErr != nil {
			log.Printf("[Reconciler] Failed to update applied_state for resource %s: %v", resource.ID, updateErr)
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

		updateErr := r.Client.UpsertAppliedState(ctx, resource.ID, &apiclient.UpsertAppliedStateRequest{
			Status:       "error",
			ErrorMessage: &errMsg,
		})

		if updateErr != nil {
			log.Printf("[Reconciler] Failed to update applied_state for resource %s: %v", resource.ID, updateErr)
		}

		return
	}

	resultBytes, _ := json.Marshal(obj)

	err = r.Client.UpsertAppliedState(ctx, resource.ID, &apiclient.UpsertAppliedStateRequest{
		Spec:         resultBytes,
		Generation:   resource.Generation,
		Revision:     resource.Revision,
		Status:       "success",
		ErrorMessage: nil,
	})

	if err != nil {
		log.Printf("[Reconciler] Failed to update applied_state for resource %s: %v", resource.ID, err)
		return
	}

	log.Printf("[Reconciler] Successfully applied resource %s (gen=%d, rev=%d)", resource.ID, resource.Generation, resource.Revision)
}

func (r *Reconciler) deleteResource(ctx context.Context, resource *models.Resource) {
	log.Printf("[Reconciler] Deleting resource %s from K8s", resource.ID)

	gvr := k8s.GetGVR(resource.Kind, resource.APIVersion)

	err := r.DynamicClient.Resource(gvr).Namespace(resource.Namespace).
		Delete(ctx, resource.Name, metav1.DeleteOptions{})

	if err != nil && !errors.IsNotFound(err) {
		log.Printf("[Reconciler] Failed to delete resource %s from K8s: %v", resource.ID, err)
		return
	}

	err = r.Client.HardDeleteResource(ctx, resource.ID)

	if err != nil {
		log.Printf("[Reconciler] Failed to hard delete resource %s: %v", resource.ID, err)
		return
	}

	log.Printf("[Reconciler] Successfully deleted resource %s", resource.ID)
}
