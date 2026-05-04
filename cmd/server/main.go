package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/app"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/router"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/scanner"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/service"
)

const readHeaderTimeout = 5 * time.Second

func main() {
	cfg := config.Load()

	if err := app.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	db, err := app.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()

	slog.Info("database connected successfully")

	ghClient := github.NewClient(cfg.GithubToken)
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)

	subRepo := repository.NewSubscriptionRepository(db)
	repoRepo := repository.NewGitHubRepository(db)

	subService := service.NewSubscriptionService(
		subRepo,
		repoRepo,
		ghClient,
		smtpMailer,
		cfg.BaseURL,
	)

	subHandler := handlers.NewSubscriptionHandler(subService)
	r := router.New(subHandler)

	sc := scanner.New(
		subRepo,
		repoRepo,
		ghClient,
		smtpMailer,
		cfg.ScanInterval,
		cfg.BaseURL,
	)
	sc.Start(context.Background())

	slog.Info("server started", "port", cfg.Port)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		return
	}
}
