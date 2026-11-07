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
// Slice 7 billing + payments read surface (optional, all or none):
// CRM_BILLING_READ_DATABASE_URL and CRM_PAYMENTS_READ_DATABASE_URL
// (module-dedicated logins) plus CRM_FEATUREFLAGS_DATABASE_URL; behind the
// flag billing_payments.read_shadow (default off).
//
// Slice 8 MIS / reporting read surface (optional): CRM_REPORTING_READ_DATABASE_URL
// plus CRM_FEATUREFLAGS_DATABASE_URL, behind reporting.read_shadow (default
// off). CRM_AUDIT_DATABASE_URL enables the audited Invoice Review query;
// without it the Invoice Review fails closed. GET never rebuilds projections.
//
// Slice 9 finance SHADOW ledger read surface (optional): CRM_FINANCE_READ_DATABASE_URL
// plus CRM_FEATUREFLAGS_DATABASE_URL, behind finance.shadow_ledger_read
// (default off). GET only; the shadow ledger is rebuilt offline by
// crm-finance-shadow and is never a financial system of record.
//
// Slice 10 Biawin / fulfillment READ shadow surface (optional):
// CRM_BIAWIN_READ_DATABASE_URL and CRM_FULFILLMENT_READ_DATABASE_URL plus
// CRM_FEATUREFLAGS_DATABASE_URL, behind biawin_fulfillment.read_shadow
// (default off). GET only; the import is the offline crm-biawin-shadow job
// and no provider is ever called.
//
// Slice 13 commission controlled SHADOW read surface (optional):
// CRM_COMMISSION_READ_DATABASE_URL plus CRM_FEATUREFLAGS_DATABASE_URL, behind
// commission.read_shadow (default off). GET only; the shadow run is the
// offline crm-commission-shadow job; nothing is approved, posted or credited.
//
// R1 production authentication (optional, OFF by default):
// CRM_IDENTITY_AUTH_DATABASE_URL plus CRM_FEATUREFLAGS_DATABASE_URL, behind
// identity.session_auth. Server-side sessions in the __Host-crm_session
// cookie; CRM_AUTH_ALLOWED_ORIGINS (https) is required in production. A
// session proves identity only: authority is the user's ACTIVE role grants
// (none exist). No enrolment, reset, recovery, MFA, OTP or customer login.
// Never combined with CRM_DEV_AUTH.
//
// Slice 14 controlled fulfillment / Biawin operations (optional, OFF by
// default): CRM_OPS_{FULFILLMENT,BIAWIN,CATALOG,INTEGRATION}_DATABASE_URL plus
// CRM_FEATUREFLAGS_DATABASE_URL, behind fulfillment.work_read,
// fulfillment.work_write and integration.provider_reconcile. No provider is
// called, no production activation (D-35), no wallet charge (D-38).
//
// Slice 11 controlled non-financial writes (optional, each OFF by default):
// CRM_CASES_WRITE_DATABASE_URL + CRM_CUSTODY_HOLDER_DATABASE_URL (contact
// outcome), CRM_ORGANIZATION_WRITE_DATABASE_URL (HR change requests) and
// CRM_WRITEBACK_READ_DATABASE_URL (write-back diagnostics), with the flag
// store. The API commits only the owning module's aggregate, history and
// outbox; legacy write-back is integration's, never the API's.
//
// It never runs migrations and never imports legacy data; its only writes
// are the Slice 11 and Slice 14 commands above, each in its owning module's schema.
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
	// R1 first: it sets the production subject resolver every surface uses.
	auth, closeAuth, err := openAuth(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeAuth()
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
	bill, closeBilling, err := openBilling(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeBilling()
	rep, closeReporting, err := openReporting(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeReporting()
	fin, closeFinance, err := openFinance(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeFinance()
	bw, closeBiawin, err := openBiawin(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeBiawin()
	com, closeCommission, err := openCommission(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeCommission()
	ops, closeOperations, err := openOperations(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeOperations()
	writes, closeWrites, err := openWrites(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closeWrites()
	pay, closePayments, err := openPayments(ctx, getenv, cfg.DevAuth)
	if err != nil {
		return err
	}
	defer closePayments()
	srv, err := apiserver.NewServerWithSurfaces(cfg, clock.System{}, logger, apiserver.Surfaces{Read: read, Customers: cust, Sales: sales, Billing: bill,
		Reporting: rep, Finance: fin, Biawin: bw, Commission: com, Operations: ops,
		CaseWrite: writes.CaseWrite, HRRequest: writes.HRRequest, Writeback: writes.Writeback, Payments: pay, Auth: auth})
	if err != nil {
		return err
	}
	logger.Info("crm-api listening", "addr", cfg.Addr, "environment", string(cfg.Environment), "dev_auth", cfg.DevAuth, "session_auth", auth != nil)
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
