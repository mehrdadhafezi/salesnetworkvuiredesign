// Command crm-reporting is the OFFLINE reporting projection job (Architecture
// V1 §20; Slice 8). It is never run by the API server and never triggered by
// a GET.
//
//	crm-reporting rebuild    one projection run from owner published read APIs
//
// CRM_REPORTING_DATABASE_URL (a reporting-owner login) is required. Each
// owner source is read through its published API with its own owner login:
// CRM_CASES_DATABASE_URL, CRM_CUSTODY_DATABASE_URL, CRM_BILLING_DATABASE_URL,
// CRM_PAYMENTS_DATABASE_URL, CRM_CUSTOMERS_DATABASE_URL. A missing source DSN
// records that source MISSING and the run PARTIAL — never SUCCESS, never
// zero. Owner reads are read-only; the only write is the reporting run.
//
// Exit status: 0 SUCCESS · 3 PARTIAL · 1 infrastructure error · 2 usage.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casesquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	custodyquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customersquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentsquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/projections"
)

// DSN environment variables (one login per owner role).
const (
	EnvReportingURL = "CRM_REPORTING_DATABASE_URL"
	EnvCasesURL     = "CRM_CASES_DATABASE_URL"
	EnvCustodyURL   = "CRM_CUSTODY_DATABASE_URL"
	EnvBillingURL   = "CRM_BILLING_DATABASE_URL"
	EnvPaymentsURL  = "CRM_PAYMENTS_DATABASE_URL"
	EnvCustomersURL = "CRM_CUSTOMERS_DATABASE_URL"
)

const usage = "usage: crm-reporting rebuild\n(" + EnvReportingURL + " required; " + EnvCasesURL + ", " + EnvCustodyURL + ", " + EnvBillingURL + ", " +
	EnvPaymentsURL + ", " + EnvCustomersURL + " optional: a missing source makes the run PARTIAL)"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type output struct {
	Command  string                             `json:"command"`
	RunID    string                             `json:"run_id"`
	Outcome  result.Status                      `json:"outcome"`
	Reasons  []string                           `json:"reasons"`
	Cutoff   string                             `json:"source_cutoff"`
	Sources  map[string]projections.SourceState `json:"sources"`
	Counts   map[string]int64                   `json:"fact_counts"`
	FactsSHA string                             `json:"facts_sha256"`
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "rebuild" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if getenv(EnvReportingURL) == "" {
		fmt.Fprintln(stderr, "crm-reporting:", EnvReportingURL, "is not set (no default target exists)")
		return 2
	}
	seen := map[string]string{}
	for _, other := range []string{db.EnvDatabaseURL, migrations.EnvMigrateURL} {
		if v := getenv(other); v != "" {
			seen[v] = other
		}
	}
	// Validate every DSN before opening any connection: no reuse of another
	// login (each owner role has its own), no default target.
	dsns := map[owner.Module]string{}
	for m, env := range map[owner.Module]string{reporting.Module: EnvReportingURL, cases.Module: EnvCasesURL, custody.Module: EnvCustodyURL,
		billing.Module: EnvBillingURL, payments.Module: EnvPaymentsURL, customers.Module: EnvCustomersURL} {
		dsn := getenv(env)
		if dsn == "" {
			continue
		}
		if prev, dup := seen[dsn]; dup {
			fmt.Fprintf(stderr, "crm-reporting: %s must not reuse %s\n", env, prev)
			return 2
		}
		seen[dsn] = env
		dsns[m] = dsn
	}
	runners := map[owner.Module]*db.Runner{}
	for m, dsn := range dsns {
		cfg := db.DefaultConfig(dsn)
		cfg.ApplicationName = "crm-reporting"
		cfg.MaxConns = 2
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "crm-reporting: %s database: %v\n", m, err)
			return 1
		}
		defer pool.Close()
		r, err := pool.Runner(m)
		if err != nil {
			fmt.Fprintln(stderr, "crm-reporting:", err)
			return 1
		}
		runners[m] = r
	}
	var src projections.Sources
	if r := runners[cases.Module]; r != nil {
		src.Cases = casesquery.New(r)
	}
	if r := runners[custody.Module]; r != nil {
		src.Custody = custodyquery.New(r)
	}
	if r := runners[billing.Module]; r != nil {
		src.Billing = billingquery.New(r)
	}
	if r := runners[payments.Module]; r != nil {
		q := paymentsquery.New(r)
		src.Payments, src.PayReader = q, q
	}
	if r := runners[customers.Module]; r != nil {
		src.Customers = customersquery.New(r)
	}
	cmd, err := projections.New(runners[reporting.Module], clock.System{})
	if err != nil {
		fmt.Fprintln(stderr, "crm-reporting:", err)
		return 1
	}
	rep, err := cmd.Rebuild(ctx, src)
	if err != nil {
		fmt.Fprintln(stderr, "crm-reporting:", err)
		return 1
	}
	out := output{Command: "rebuild", RunID: rep.RunID, Outcome: rep.Outcome, Reasons: []string{}, Cutoff: rep.Cutoff.UTC().Format("2006-01-02T15:04:05Z"),
		Sources: rep.Sources, Counts: rep.Counts, FactsSHA: fmt.Sprintf("%x", rep.FactsSHA)}
	for _, r := range rep.Reasons {
		out.Reasons = append(out.Reasons, string(r))
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return 1
	}
	if rep.Outcome != result.Success {
		return 3
	}
	return 0
}
