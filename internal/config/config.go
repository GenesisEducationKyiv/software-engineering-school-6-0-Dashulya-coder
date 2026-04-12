package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         string
	DatabaseURL  string
	GithubToken  string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	BaseURL      string
	ScanInterval time.Duration
}

func Load() *Config {
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	if err != nil {
		log.Fatal("invalid SMTP_PORT")
	}

	scanInterval, err := time.ParseDuration(getEnv("SCAN_INTERVAL", "5m"))
	if err != nil {
		log.Fatal("invalid SCAN_INTERVAL")
	}

	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		GithubToken:  os.Getenv("GITHUB_TOKEN"),
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     smtpPort,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPass:     os.Getenv("SMTP_PASS"),
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
