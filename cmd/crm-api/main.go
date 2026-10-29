// Command crm-api runs the CRM HTTP API (Slice 2: technical health and the
// session context only). Configuration comes from the environment:
//
//	CRM_API_ADDR   listen address (default 127.0.0.1:8080)
//	CRM_ENV        production (default) | development | test
//	CRM_DEV_AUTH   "1" enables the DEV/TEST auth stub; refused in production
//
// It never runs migrations and connects to no database.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Getenv, logger); err != nil {
		logger.Error("crm-api stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string, logger *slog.Logger) error {
	cfg, err := apiserver.ConfigFromEnv(getenv)
	if err != nil {
		return err
	}
	srv, err := apiserver.NewServer(cfg, clock.System{}, logger)
	if err != nil {
		return err
	}
	logger.Info("crm-api listening", "addr", cfg.Addr, "environment", string(cfg.Environment), "dev_auth", cfg.DevAuth)
	return apiserver.Serve(ctx, srv, cfg, logger)
}
