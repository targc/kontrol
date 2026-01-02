package config

import (
	"context"
	"log"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	DBURL         string `env:"KONTROL_DB_URL,required"`
	ClusterID     string `env:"KONTROL_CLUSTER_ID,required"`
	ServerPort    string `env:"KONTROL_SERVER_PORT,default=8080"`
	Kubeconfig    string `env:"KONTROL_KUBECONFIG"`
	AutoMigrate   bool   `env:"KONTROL_AUTO_MIGRATE,default=false"`
	SupportedGVRs string `env:"KONTROL_SUPPORTED_GVRS"` // comma-separated list: deployment,pod,service
}

func Load(ctx context.Context) *Config {
	var cfg Config

	err := envconfig.Process(ctx, &cfg)

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	return &cfg
}
