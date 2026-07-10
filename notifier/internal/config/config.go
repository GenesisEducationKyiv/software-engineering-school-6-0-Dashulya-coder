package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	GRPCPort    string
	RESTPort    string
	MetricsPort string
	DatabaseURL string
	RabbitURL   string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
}

func Load() *Config {
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	if err != nil {
		log.Fatal("invalid SMTP_PORT")
	}

	cfg := &Config{
		GRPCPort:    getEnv("GRPC_PORT", "9091"),
		RESTPort:    getEnv("REST_PORT", "9094"),
		MetricsPort: getEnv("METRICS_PORT", "9092"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RabbitURL:   getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    smtpPort,
		SMTPUser:    os.Getenv("SMTP_USER"),
		SMTPPass:    os.Getenv("SMTP_PASS"),
		SMTPFrom:    getEnv("SMTP_FROM", "no-reply@example.com"),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if cfg.SMTPHost == "" {
		log.Fatal("SMTP_HOST is required")
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
