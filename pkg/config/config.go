package config

import (
	"context"
	"log"

	"github.com/sethvargo/go-envconfig"
)

// APIConfig is used by cmd/api
type APIConfig struct {
	DBURL       string `env:"KONTROL_DB_URL,required"`
	ServerPort  string `env:"KONTROL_SERVER_PORT,default=8080"`
	AutoMigrate bool   `env:"KONTROL_AUTO_MIGRATE,default=false"`
}

// WorkerConfig is used by cmd/worker
type WorkerConfig struct {
	APIURL        string `env:"KONTROL_API_URL,required"`
	APIKey        string `env:"KONTROL_API_KEY,required"`
	ClusterID     string `env:"KONTROL_CLUSTER_ID,required"`
	Kubeconfig    string `env:"KONTROL_KUBECONFIG"`
	SupportedGVRs string `env:"KONTROL_SUPPORTED_GVRS"` // comma-separated list: deployment,pod,service
}

func LoadAPIConfig(ctx context.Context) *APIConfig {
	var cfg APIConfig

	err := envconfig.Process(ctx, &cfg)

	if err != nil {
		log.Fatalf("failed to load api config: %v", err)
	}

	return &cfg
}

func LoadWorkerConfig(ctx context.Context) *WorkerConfig {
	var cfg WorkerConfig

	err := envconfig.Process(ctx, &cfg)

	if err != nil {
		log.Fatalf("failed to load worker config: %v", err)
	}

	return &cfg
}
