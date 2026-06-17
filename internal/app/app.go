package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	notificationclient "github.com/Dashulya-coder/CaseTaskNotifier/internal/client/notification"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	httphandlers "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	httprouter "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/router"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/release"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/scanner"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/token"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

var _ subscription.TokenGenerator = (*token.Generator)(nil)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
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

	notifier, err := notificationclient.New(cfg.NotifierAddr)
	if err != nil {
		return err
	}
	defer func() {
		if err := notifier.Close(); err != nil {
			slog.Error("failed to close notifier client", "error", err)
		}
	}()

	subRepo := repository.NewSubscriptionRepository(db)
	repoRepo := repository.NewGitHubRepository(db)

	subService := subscription.NewSubscriptionService(
		subRepo, repoRepo, ghClient, notifier, urls, token.New(),
	)
	poller := release.NewPoller(subRepo, repoRepo, ghClient, notifier, urls)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := scanner.New(poller, cfg.ScanInterval)
	sc.Start(ctx)

	subHandler := httphandlers.NewSubscriptionHandler(subService)
	r := httprouter.New(subHandler)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	go func() {
		<-ctx.Done()
		slog.Info("server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("server started", "port", cfg.Port)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
