package config

import (
	"context"
	"log"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	DBURL         string `env:"KONTROL_DB_URL,required"`
	TablePrefix   string `env:"KONTROL_TABLE_PREFIX"`
	ClusterID     string `env:"KONTROL_CLUSTER_ID,required"`
	ServerPort    string `env:"KONTROL_SERVER_PORT,default=8080"`
	Kubeconfig    string `env:"KONTROL_KUBECONFIG"`
	AutoMigrate   bool   `env:"KONTROL_AUTO_MIGRATE,default=false"`
}

func Load(ctx context.Context) *Config {
	var cfg Config

	err := envconfig.Process(ctx, &cfg)

	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	return &cfg
}
