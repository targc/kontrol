package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL       string
	TablePrefix string
	ClusterID   string
	ServerPort  string
	Kubeconfig  string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		DBURL:       getEnv("KONTROL_DB_URL", "postgres://postgres:postgres@localhost:5432/kontrol?sslmode=disable"),
		TablePrefix: getEnv("KONTROL_TABLE_PREFIX", ""),
		ClusterID:   getEnv("KONTROL_CLUSTER_ID", "default"),
		ServerPort:  getEnv("KONTROL_SERVER_PORT", "8080"),
		Kubeconfig:  getEnv("KONTROL_KUBECONFIG", os.Getenv("HOME")+"/.kube/config"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
