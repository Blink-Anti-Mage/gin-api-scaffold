package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/gin-api-scaffold/internal/app"
	"github.com/example/gin-api-scaffold/internal/config"
	"github.com/example/gin-api-scaffold/pkg/logger"
)

func main() {
	configPath := flag.String("c", "", "path to JSON config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load_config_failed", "error", err)
		os.Exit(1)
	}

	logger := logger.New(os.Stdout, cfg.Log.SlogLevel())

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("create_app_failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		logger.Error("app_stopped_with_error", "error", err)
		os.Exit(1)
	}
}
