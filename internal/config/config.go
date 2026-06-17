package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	GithubToken   string
	NotifierAddr  string
	BaseURL       string
	ScanInterval  time.Duration
	LogLevel      string
	LogSampleRate uint64
}

func Load() *Config {
	scanInterval, err := time.ParseDuration(getEnv("SCAN_INTERVAL", "5m"))
	if err != nil {
		log.Fatal("invalid SCAN_INTERVAL")
	}

	logSampleRate, err := strconv.ParseUint(getEnv("LOG_SAMPLE_RATE", "10"), 10, 64)
	if err != nil || logSampleRate < 1 {
		log.Fatal("invalid LOG_SAMPLE_RATE")
	}

	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		GithubToken:   os.Getenv("GITHUB_TOKEN"),
		NotifierAddr:  getEnv("NOTIFIER_ADDR", "localhost:9091"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:8080"),
		ScanInterval:  scanInterval,
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LogSampleRate: logSampleRate,
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
