package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-rest-homework/internal/config"
	"go-rest-homework/internal/consul"
	"go-rest-homework/internal/httpapi"
	"go-rest-homework/internal/kafka"
	"go-rest-homework/internal/metrics"
	"go-rest-homework/internal/security"
	"go-rest-homework/internal/service"
	"go-rest-homework/internal/storage"
	"go-rest-homework/internal/vault"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	userStore, err := storage.Open(ctx, cfg.DBPath)
	if err != nil {
		logger.Error("storage_open_failed", "error", err)
		os.Exit(1)
	}
	defer userStore.Close()

	vaultClient := vault.New(cfg, logger)
	signer := security.NewTokenSigner(vaultClient)
	publisher := kafka.NewPublisher(cfg, logger)
	defer publisher.Close()

	authService := service.NewAuthService(userStore, publisher, signer, logger)
	registry := metrics.NewRegistry()
	handler := httpapi.NewHandler(authService, registry, logger)
	consulClient := consul.New(cfg, logger)
	consulClient.Register(ctx)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	logger.Info("server stopping")
	consulClient.Deregister(shutdownCtx)
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
