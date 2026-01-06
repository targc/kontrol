package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/targc/kontrol/pkg/apiclient"
	"github.com/targc/kontrol/pkg/global_syncer"
	"github.com/targc/kontrol/pkg/reconciler"
	"github.com/targc/kontrol/pkg/watcher"
)

type Worker struct {
	Client       *apiclient.Client
	ClusterID    string
	Kubeconfig   string
	watcher      *watcher.Watcher
	reconciler   *reconciler.Reconciler
	globalSyncer *global_syncer.GlobalSyncer
	cancel       context.CancelFunc
}

func NewWorker(ctx context.Context, client *apiclient.Client, clusterID, kubeconfig string) (*Worker, error) {
	// Register cluster with API
	err := client.RegisterCluster(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to register cluster: %w", err)
	}

	log.Printf("[Worker] Registered cluster: %s", clusterID)

	w, err := watcher.NewWatcher(client, clusterID, kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	r, err := reconciler.NewReconciler(client, clusterID, kubeconfig)

	if err != nil {
		return nil, fmt.Errorf("failed to create reconciler: %w", err)
	}

	gs := global_syncer.NewGlobalSyncer(client, clusterID)

	return &Worker{
		Client:       client,
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
