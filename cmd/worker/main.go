package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/targc/kontrol/pkg/config"
	"github.com/targc/kontrol/pkg/database"
	"github.com/targc/kontrol/pkg/worker"
)

func main() {
	log.Println("Starting Kontrol Worker...")

	cfg := config.Load()

	db, err := database.Connect(cfg)

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = database.AutoMigrate(db)

	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := worker.NewWorker(ctx, db, cfg.ClusterID, cfg.Kubeconfig)

	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	go w.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.Stop()
}
