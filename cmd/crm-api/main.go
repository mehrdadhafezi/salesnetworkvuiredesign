// Command crm-api runs the CRM HTTP API (Slice 2: technical health and the
// session context only). Configuration comes from the environment:
//
//	CRM_API_ADDR   listen address (default 127.0.0.1:8080)
//	CRM_ENV        production (default) | development | test
//	CRM_DEV_AUTH   "1" enables the DEV/TEST auth stub; refused in production
//
// Slice 4 read surface (optional, all or none): CRM_IDENTITY_READ_DATABASE_URL,
// CRM_ORGANIZATION_READ_DATABASE_URL and CRM_FEATUREFLAGS_DATABASE_URL, each
// a login dedicated to that module's owner role. Every read runs in a READ
// ONLY transaction; the HR read surface is additionally behind the platform
// feature flag organization.hr_read_shadow (default off). Without them the
// surface answers BLOCKED / FEATURE_DISABLED.
//
// Slice 5 customer identity read surface (optional): CRM_CUSTOMERS_READ_DATABASE_URL
// (a customers-dedicated login) plus CRM_FEATUREFLAGS_DATABASE_URL; behind
// the flag customers.identity_shadow_read (default off).
//
// Slice 6 sales read surface (optional, all or none): CRM_CASES_READ_DATABASE_URL
// and CRM_CUSTODY_READ_DATABASE_URL (module-dedicated logins) plus
// CRM_FEATUREFLAGS_DATABASE_URL; behind the flag sales.read_shadow (default off).
//
// It never runs migrations, never imports legacy data and never writes.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
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
	read, closeRead, err := openRead(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeRead()
	cust, closeCustomers, err := openCustomers(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeCustomers()
	sales, closeSales, err := openSales(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeSales()
	srv, err := apiserver.NewServerWithSurfaces(cfg, clock.System{}, logger, apiserver.Surfaces{Read: read, Customers: cust, Sales: sales})
	if err != nil {
		return err
	}
	logger.Info("crm-api listening", "addr", cfg.Addr, "environment", string(cfg.Environment), "dev_auth", cfg.DevAuth)
	return apiserver.Serve(ctx, srv, cfg, logger)
}

// openRead opens the Slice 4 read pools when configured (all or none).
func openRead(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.ReadAPI, func(), error) {
	urls := map[owner.Module]string{identity.Module: getenv(EnvIdentityReadURL), organization.Module: getenv(EnvOrganizationReadURL),
		owner.FeatureFlags: getenv(EnvFeatureFlagsURL)}
	set := 0
	for _, u := range urls {
		if u != "" {
			set++
		}
	}
	switch set {
	case 0:
		return nil, func() {}, nil
	case len(urls):
	default:
		return nil, nil, ErrReadConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range urls {
		cfg := db.DefaultConfig(u)
		cfg.ApplicationName = "crm-api"
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("crm-api: %s read database: %w", m, err)
		}
		pools = append(pools, pool)
		if runners[m], err = pool.Runner(m); err != nil {
			closeAll()
			return nil, nil, err
		}
	}
	read, err := NewReadAPI(runners[identity.Module], runners[organization.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return read, closeAll, nil
}
