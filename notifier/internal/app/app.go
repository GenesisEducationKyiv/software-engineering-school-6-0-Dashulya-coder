package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"buf.build/go/protovalidate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/notifier/gen/notification/v1"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/config"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/consumer"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/server"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/smtp"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/store"
)

const (
	readHeaderTimeout = 5 * time.Second
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

	slog.Info("notifier database connected successfully")

	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("create validator: %w", err)
	}

	ledger := store.NewLedger(db)
	sender := smtp.NewClient(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	svc := delivery.New(ledger, sender)
	srv := server.New(svc, validator)

	releaseConsumer, err := consumer.New(cfg.RabbitURL, svc)
	if err != nil {
		return err
	}
	defer func() {
		if err := releaseConsumer.Close(); err != nil {
			slog.Error("failed to close release consumer", "error", err)
		}
	}()

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(server.TraceInterceptor, server.RecoveryInterceptor),
	)
	notificationv1.RegisterNotificationServiceServer(grpcServer, srv)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:              ":" + cfg.MetricsPort,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		slog.Info("notifier metrics server started", "port", cfg.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server error", "error", err)
		}
	}()

	consumerErr := make(chan error, 1)
	go func() {
		if err := releaseConsumer.Run(ctx); err != nil {
			consumerErr <- err
			stop()
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("notifier shutting down")
		grpcServer.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("metrics server shutdown error", "error", err)
		}
	}()

	slog.Info("notifier grpc server started", "port", cfg.GRPCPort)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	select {
	case err := <-consumerErr:
		return fmt.Errorf("release consumer failed: %w", err)
	default:
		return nil
	}
}
