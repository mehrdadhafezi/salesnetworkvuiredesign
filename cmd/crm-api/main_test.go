package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestRunRefusesDevAuthInProduction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run(context.Background(), env(map[string]string{"CRM_DEV_AUTH": "1"}), logger); err == nil {
		t.Fatal("crm-api must refuse to start with dev auth in production")
	}
	if err := run(context.Background(), env(map[string]string{"CRM_ENV": "staging"}), logger); err == nil {
		t.Fatal("unknown environment must be refused")
	}
}

func TestRunStartsAndShutsDown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := run(ctx, env(map[string]string{"CRM_API_ADDR": "127.0.0.1:0", "CRM_ENV": "test", "CRM_DEV_AUTH": "1"}), logger); err != nil {
		t.Fatalf("run: %v", err)
	}
}
