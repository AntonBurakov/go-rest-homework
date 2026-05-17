package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go-rest-homework/internal/config"
	"go-rest-homework/internal/kafka"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	consumer := kafka.NewConsumer(cfg, logger)
	defer consumer.Close()

	consumer.Run(ctx)
}
