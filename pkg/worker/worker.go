package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/targc/kontrol/pkg/reconciler"
	"github.com/targc/kontrol/pkg/watcher"
	"gorm.io/gorm"
)

// Worker encapsulates the watcher and reconciler components
type Worker struct {
	DB         *gorm.DB
	ClusterID  string
	Kubeconfig string
	watcher    *watcher.Watcher
	reconciler *reconciler.Reconciler
	cancel     context.CancelFunc
}

// NewWorker creates a new Worker instance
func NewWorker(db *gorm.DB, clusterID, kubeconfig string) (*Worker, error) {
	w, err := watcher.NewWatcher(db, clusterID, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	r, err := reconciler.NewReconciler(db, clusterID, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create reconciler: %w", err)
	}

	return &Worker{
		DB:         db,
		ClusterID:  clusterID,
		Kubeconfig: kubeconfig,
		watcher:    w,
		reconciler: r,
	}, nil
}

// Start begins the worker's watcher and reconciler loops
func (w *Worker) Start(ctx context.Context) error {
	log.Printf("[Worker] Starting for cluster: %s", w.ClusterID)

	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	go w.watcher.Start(ctx)
	go w.reconciler.Start(ctx)

	<-ctx.Done()
	log.Println("[Worker] Stopped")

	return nil
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	if w.cancel != nil {
		log.Println("[Worker] Shutting down...")
		w.cancel()
	}
}
