package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
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

func TestBillingSurfaceIsAllOrNone(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	partial := map[string]string{"CRM_ENV": "test", EnvBillingReadURL: "postgres://x@127.0.0.1:1/x"}
	if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrBillingConfig) {
		t.Fatalf("partial billing config: %v", err)
	}
}

func TestReportingSurfaceNeedsFlagStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	partial := map[string]string{"CRM_ENV": "test", EnvReportingReadURL: "postgres://x@127.0.0.1:1/x"}
	if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrReportingConfig) {
		t.Fatalf("partial reporting config: %v", err)
	}
}

func TestFinanceSurfaceNeedsFlagStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	partial := map[string]string{"CRM_ENV": "test", EnvFinanceReadURL: "postgres://x@127.0.0.1:1/x"}
	if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrFinanceConfig) {
		t.Fatalf("partial finance config: %v", err)
	}
}

func TestBiawinSurfaceNeedsBothModulesAndFlagStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, partial := range []map[string]string{
		{"CRM_ENV": "test", EnvBiawinReadURL: "postgres://x@127.0.0.1:1/x"},
		{"CRM_ENV": "test", EnvFulfillmentReadURL: "postgres://x@127.0.0.1:1/x"},
		{"CRM_ENV": "test", EnvBiawinReadURL: "postgres://x@127.0.0.1:1/x", EnvFulfillmentReadURL: "postgres://x@127.0.0.1:1/x"},
	} {
		if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrBiawinConfig) {
			t.Fatalf("partial biawin config %v: %v", partial, err)
		}
	}
}

func TestCommissionSurfaceNeedsFlagStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	partial := map[string]string{"CRM_ENV": "test", EnvCommissionReadURL: "postgres://x@127.0.0.1:1/x"}
	if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrCommissionConfig) {
		t.Fatalf("partial commission config: %v", err)
	}
}

func TestOperationsSurfaceNeedsAllLoginsAndFlagStore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	u := "postgres://x@127.0.0.1:1/x"
	all := map[string]string{"CRM_ENV": "test", EnvOpsFulfillmentURL: u, EnvOpsBiawinURL: u, EnvOpsCatalogURL: u, EnvOpsIntegrationURL: u}
	for _, partial := range []map[string]string{
		{"CRM_ENV": "test", EnvOpsFulfillmentURL: u},
		{"CRM_ENV": "test", EnvOpsIntegrationURL: u},
		{"CRM_ENV": "test", EnvOpsFulfillmentURL: u, EnvOpsBiawinURL: u, EnvOpsCatalogURL: u},
		all, // all four logins but no flag store
	} {
		if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrOperationsConfig) {
			t.Fatalf("partial operations config %v: %v", partial, err)
		}
	}
}

func TestWritesSurfaceNeedsFlagStoreAndHolderRead(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, partial := range []map[string]string{
		{"CRM_ENV": "test", EnvCasesWriteURL: "postgres://x@127.0.0.1:1/x"},
		{"CRM_ENV": "test", EnvOrganizationWriteURL: "postgres://x@127.0.0.1:1/x"},
	} {
		if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrWritesConfig) {
			t.Fatalf("partial writes config %v: %v", partial, err)
		}
	}
}

func TestPaymentsSurfaceIsAllOrNone(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, partial := range []map[string]string{
		{"CRM_ENV": "test", EnvPaymentsWriteURL: "postgres://x@127.0.0.1:1/x"},
		{"CRM_ENV": "test", EnvPaymentsObjectRoot: "/tmp"},
		{"CRM_ENV": "test", EnvPaymentsWriteURL: "postgres://x@127.0.0.1:1/x", EnvPaymentsBillingURL: "postgres://x@127.0.0.1:1/x",
			EnvPaymentsObjectRoot: "/tmp"},
	} {
		if err := run(context.Background(), env(partial), logger); !errors.Is(err, ErrPaymentsConfig) {
			t.Fatalf("partial payments config %v: %v", partial, err)
		}
	}
}

// R1: session auth needs the flag store, can never coexist with the dev
// stub, and refuses out-of-bounds session parameters.
func TestAuthSurfaceConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	u := "postgres://x@127.0.0.1:1/x"
	if err := run(context.Background(), env(map[string]string{"CRM_ENV": "test", EnvIdentityAuthURL: u}), logger); !errors.Is(err, ErrAuthConfig) {
		t.Fatalf("auth without flag store: %v", err)
	}
	if err := run(context.Background(), env(map[string]string{"CRM_ENV": "development", "CRM_DEV_AUTH": "1", EnvIdentityAuthURL: u, EnvFeatureFlagsURL: u}), logger); !errors.Is(err, apiserver.ErrAuthConflict) {
		t.Fatalf("dev stub + sessions: %v", err)
	}
	for k, v := range map[string]string{EnvSessionIdle: "10s", EnvSessionAbsolute: "720h", EnvLoginMaxFailures: "0", EnvLoginFailureWindow: "nonsense"} {
		if _, err := SessionConfigFromEnv(env(map[string]string{k: v})); err == nil {
			t.Fatalf("%s=%s accepted", k, v)
		}
	}
	cfg, err := SessionConfigFromEnv(env(map[string]string{EnvSessionIdle: "15m", EnvLoginMaxFailures: "10"}))
	if err != nil || cfg.IdleTimeout.Minutes() != 15 || cfg.MaxFailures != 10 {
		t.Fatalf("overrides: %+v %v", cfg, err)
	}
	if err := run(context.Background(), env(map[string]string{"CRM_ENV": "production", apiserver.EnvAllowedOrigins: "http://crm.example.test"}), logger); err == nil {
		t.Fatal("plain-http allowed origin accepted in production")
	}
}
