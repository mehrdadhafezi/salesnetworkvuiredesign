// Command crm-finance-shadow is the OFFLINE finance SHADOW ledger job
// (Architecture V1 §16; Slice 9). It is never run by the API server and never
// triggered by a GET, page render, session or report request. It derives a
// SHADOW journal for architecture validation and reconciliation only: it
// never posts, approves, settles, refunds, credits or debits anything, never
// writes another schema and never calls or writes the legacy system.
//
//	crm-finance-shadow rebuild [--snapshot-id ID]
//
// CRM_FINANCE_DATABASE_URL (a finance-owner login) is required. Billing and
// payments are read through their published APIs with their own owner
// logins: CRM_BILLING_DATABASE_URL, CRM_PAYMENTS_DATABASE_URL. Legacy wallet
// evidence comes ONLY from a staged snapshot (--snapshot-id), read with
// CRM_STAGING_DATABASE_URL and resolved through the identity and billing
// published APIs (CRM_IDENTITY_DATABASE_URL, CRM_BILLING_DATABASE_URL). A
// missing source is recorded MISSING and the run PARTIAL — never SUCCESS,
// never zero. Identical inputs under the same derivation version are a
// replay of the existing run (nothing is written twice).
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
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	financeshadow "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentsquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// DSN environment variables (one login per owner role).
const (
	EnvFinanceURL  = "CRM_FINANCE_DATABASE_URL"
	EnvBillingURL  = "CRM_BILLING_DATABASE_URL"
	EnvPaymentsURL = "CRM_PAYMENTS_DATABASE_URL"
	EnvStagingURL  = "CRM_STAGING_DATABASE_URL"
	EnvIdentityURL = "CRM_IDENTITY_DATABASE_URL"
)

const usage = "usage: crm-finance-shadow rebuild [--snapshot-id ID]\n(" + EnvFinanceURL + " required; " + EnvBillingURL + ", " + EnvPaymentsURL +
	" optional; --snapshot-id needs " + EnvStagingURL + ", " + EnvIdentityURL + " and " + EnvBillingURL + "; a missing source makes the run PARTIAL)"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type output struct {
	Command    string                               `json:"command"`
	RecordKind string                               `json:"record_kind"`
	RunID      string                               `json:"run_id"`
	Replayed   bool                                 `json:"replayed"`
	Outcome    result.Status                        `json:"outcome"`
	Reasons    []string                             `json:"reasons"`
	Cutoff     string                               `json:"source_cutoff"`
	Sources    map[string]financeshadow.SourceState `json:"sources"`
	Counts     map[string]int64                     `json:"fact_counts"`
	InputSHA   string                               `json:"input_sha256"`
	FactsSHA   string                               `json:"facts_sha256"`
	Notice     string                               `json:"notice"`
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	snapshotID := ""
	switch {
	case len(args) == 1 && args[0] == "rebuild":
	case len(args) == 3 && args[0] == "rebuild" && args[1] == "--snapshot-id" && args[2] != "":
		snapshotID = args[2]
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if getenv(EnvFinanceURL) == "" {
		fmt.Fprintln(stderr, "crm-finance-shadow:", EnvFinanceURL, "is not set (no default target exists)")
		return 2
	}
	if snapshotID != "" && (getenv(EnvStagingURL) == "" || getenv(EnvIdentityURL) == "" || getenv(EnvBillingURL) == "") {
		fmt.Fprintln(stderr, "crm-finance-shadow: --snapshot-id needs", EnvStagingURL, EnvIdentityURL, "and", EnvBillingURL)
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
	for m, env := range map[owner.Module]string{finance.Module: EnvFinanceURL, billing.Module: EnvBillingURL, payments.Module: EnvPaymentsURL,
		integration.Module: EnvStagingURL, identity.Module: EnvIdentityURL} {
		dsn := getenv(env)
		if dsn == "" || (snapshotID == "" && (m == integration.Module || m == identity.Module)) {
			continue
		}
		if prev, dup := seen[dsn]; dup {
			fmt.Fprintf(stderr, "crm-finance-shadow: %s must not reuse %s\n", env, prev)
			return 2
		}
		seen[dsn] = env
		dsns[m] = dsn
	}
	runners := map[owner.Module]*db.Runner{}
	for m, dsn := range dsns {
		cfg := db.DefaultConfig(dsn)
		cfg.ApplicationName = "crm-finance-shadow"
		cfg.MaxConns = 2
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "crm-finance-shadow: %s database: %v\n", m, err)
			return 1
		}
		defer pool.Close()
		r, err := pool.Runner(m)
		if err != nil {
			fmt.Fprintln(stderr, "crm-finance-shadow:", err)
			return 1
		}
		runners[m] = r
	}
	var src financeshadow.Sources
	var bq *billingquery.Service
	if r := runners[billing.Module]; r != nil {
		bq = billingquery.New(r)
		src.Billing, src.Amounts = bq, bq
	}
	if r := runners[payments.Module]; r != nil {
		src.Payments = paymentsquery.New(r)
	}
	if snapshotID != "" {
		w, err := stagedWallets(ctx, runners[integration.Module], identityquery.New(runners[identity.Module]), bq, snapshotID)
		if err != nil {
			fmt.Fprintln(stderr, "crm-finance-shadow:", err)
			return 1
		}
		src.Wallets = w
	}
	cmd, err := financeshadow.New(runners[finance.Module], clock.System{})
	if err != nil {
		fmt.Fprintln(stderr, "crm-finance-shadow:", err)
		return 1
	}
	rep, err := cmd.Rebuild(ctx, src)
	if err != nil {
		fmt.Fprintln(stderr, "crm-finance-shadow:", err)
		return 1
	}
	out := output{Command: "rebuild", RecordKind: "SHADOW", RunID: rep.RunID, Replayed: rep.Replayed, Outcome: rep.Outcome, Reasons: []string{},
		Cutoff: rep.Cutoff.UTC().Format("2006-01-02T15:04:05Z"), Sources: rep.Sources, Counts: rep.Counts,
		InputSHA: fmt.Sprintf("%x", rep.InputSHA), FactsSHA: fmt.Sprintf("%x", rep.FactsSHA),
		Notice: "SHADOW only: not the financial system of record; nothing was posted, approved, settled, refunded, credited or debited"}
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

// stagedWallets reads one staged snapshot (READ ONLY) and resolves its user
// and invoice keys through the identity and billing published APIs.
func stagedWallets(ctx context.Context, staged *db.Runner, ident *identityquery.Service, bq *billingquery.Service, snapshotID string) (shadow.StagedWallets, error) {
	b, err := shadow.ReadBatch(ctx, staged, spec.Default(), snapshotID)
	if err != nil {
		return shadow.StagedWallets{}, err
	}
	users, err := ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.FinanceUserKeys(b))
	if err != nil {
		return shadow.StagedWallets{}, err
	}
	src, err := bq.ResolveInvoices(ctx, shadow.SourceSystem, shadow.SpecInvoice, shadow.FinanceInvoiceKeys(b))
	if err != nil {
		return shadow.StagedWallets{}, err
	}
	inv := map[string]string{}
	for k, v := range src {
		inv[k] = v.InvoiceID
	}
	return shadow.StagedWallets{Batch: b, Users: users, Invoices: inv}, nil
}
