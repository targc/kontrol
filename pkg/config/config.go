package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
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
		DBHost:      getEnv("KONTROL_DB_HOST", "localhost"),
		DBPort:      getEnv("KONTROL_DB_PORT", "5432"),
		DBUser:      getEnv("KONTROL_DB_USER", "postgres"),
		DBPassword:  getEnv("KONTROL_DB_PASSWORD", "postgres"),
		DBName:      getEnv("KONTROL_DB_NAME", "kontrol"),
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
