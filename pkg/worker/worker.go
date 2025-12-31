package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/targc/kontrol/pkg/global_syncer"
	"github.com/targc/kontrol/pkg/models"
	"github.com/targc/kontrol/pkg/reconciler"
	"github.com/targc/kontrol/pkg/watcher"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Worker struct {
	DB           *gorm.DB
	ClusterID    string
	Kubeconfig   string
	watcher      *watcher.Watcher
	reconciler   *reconciler.Reconciler
	globalSyncer *global_syncer.GlobalSyncer
	cancel       context.CancelFunc
}

func NewWorker(ctx context.Context, db *gorm.DB, clusterID, kubeconfig string) (*Worker, error) {
	cluster := models.Cluster{ID: clusterID}

	err := db.
		WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&cluster).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to register cluster: %w", err)
	}

	log.Printf("[Worker] Registered cluster: %s", clusterID)

	w, err := watcher.NewWatcher(db, clusterID, kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	r, err := reconciler.NewReconciler(db, clusterID, kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create reconciler: %w", err)
	}

	gs := global_syncer.NewGlobalSyncer(db, clusterID)

	return &Worker{
		DB:           db,
		ClusterID:    clusterID,
		Kubeconfig:   kubeconfig,
		watcher:      w,
		reconciler:   r,
		globalSyncer: gs,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	log.Printf("[Worker] Starting for cluster: %s", w.ClusterID)

	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	go w.watcher.Start(ctx)
	go w.reconciler.Start(ctx)
	go w.globalSyncer.Start(ctx)

	<-ctx.Done()
	log.Println("[Worker] Stopped")

	return nil
}

func (w *Worker) Stop() {
	if w.cancel != nil {
		log.Println("[Worker] Shutting down...")
		w.cancel()
	}
}
