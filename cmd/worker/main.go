package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/targc/kontrol/pkg/apiclient"
	"github.com/targc/kontrol/pkg/config"
	"github.com/targc/kontrol/pkg/k8s"
	"github.com/targc/kontrol/pkg/worker"
)

func main() {
	log.Println("Starting Kontrol Worker...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.LoadWorkerConfig(ctx)

	k8s.InitSupportedGVRs(cfg.SupportedGVRs)
	log.Printf("Watching %d GVRs", len(k8s.SupportedGVRs))

	client := apiclient.NewClient(cfg.APIURL, cfg.APIKey, cfg.ClusterID)

	w, err := worker.NewWorker(ctx, client, cfg.ClusterID, cfg.Kubeconfig)

	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	go w.Start(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.Stop()
}
