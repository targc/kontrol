package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/targc/kontrol/pkg/config"
	"github.com/targc/kontrol/pkg/database"
	"github.com/targc/kontrol/pkg/reconciler"
	"github.com/targc/kontrol/pkg/watcher"
)

func main() {
	log.Println("Starting Kontrol Worker...")

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := watcher.NewWatcher(db, cfg.ClusterID, cfg.Kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	r, err := reconciler.NewReconciler(db, cfg.ClusterID, cfg.Kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create reconciler: %v", err)
	}

	log.Printf("Worker starting for cluster: %s", cfg.ClusterID)

	go w.Start(ctx)
	go r.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down worker...")
	cancel()
}
