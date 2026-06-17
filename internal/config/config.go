package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	Port         string
	DatabaseURL  string
	GithubToken  string
	NotifierAddr string
	BaseURL      string
	ScanInterval time.Duration
}

func Load() *Config {
	scanInterval, err := time.ParseDuration(getEnv("SCAN_INTERVAL", "5m"))
	if err != nil {
		log.Fatal("invalid SCAN_INTERVAL")
	}

	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		GithubToken:  os.Getenv("GITHUB_TOKEN"),
		NotifierAddr: getEnv("NOTIFIER_ADDR", "localhost:9090"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
		ScanInterval: scanInterval,
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
