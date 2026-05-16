package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	httphandlers "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	httprouter "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/router"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/release"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/scanner"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

func Run() error {
	cfg := config.Load()

	if err := RunMigrations(cfg.DatabaseURL); err != nil {
		return err
	}

	db, err := ConnectDB(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("failed to close db", "error", err)
		}
	}()

	slog.Info("database connected successfully")

	urls := urlbuilder.New(cfg.BaseURL)

	ghClient := github.NewClient(cfg.GithubToken)
	smtpMailer := mailer.NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass)

	subRepo := repository.NewSubscriptionRepository(db)
	repoRepo := repository.NewGitHubRepository(db)

	subService := subscription.NewSubscriptionService(subRepo, repoRepo, ghClient, smtpMailer, urls)
	poller := release.NewPoller(subRepo, repoRepo, ghClient, smtpMailer, urls)

	sc := scanner.New(poller, cfg.ScanInterval)
	sc.Start(context.Background())

	subHandler := httphandlers.NewSubscriptionHandler(subService)
	r := httprouter.New(subHandler)

	slog.Info("server started", "port", cfg.Port)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
