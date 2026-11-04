// Command crm-commission-shadow is the OFFLINE commission SHADOW job
// (Architecture V1 §19; Slice 13). It is never run by the API server and
// never triggered by a GET, page render, session or report request. It
// records legacy commission evidence from ONE staged snapshot, observes
// policy versions, recalculates every historical entitlement from its
// event-time snapshot with the pure registry and records parity and
// reconciliation findings. It never calculates a new entitlement from the
// current hierarchy, never approves, posts, credits, settles or reverses
// anything, never dispatches a posting request, never writes another schema
// and never calls or writes the legacy system.
//
//	crm-commission-shadow run --snapshot-id ID --rounding LEGACY_PHP_ROUND_HALF_AWAY_2DP|NONE
//
// --rounding is mandatory: there is no default rounding (OPD-06). NONE
// records a run whose every calculation is BLOCKED. The legacy rounding is
// evidenced legacy behaviour used for PARITY only, not an approved target.
//
// CRM_COMMISSION_DATABASE_URL (a commission-owner login) is required.
// Evidence comes ONLY from a staged snapshot, read with
// CRM_STAGING_DATABASE_URL and resolved through the identity and billing
// published APIs (CRM_IDENTITY_DATABASE_URL, CRM_BILLING_DATABASE_URL). A
// missing source is recorded MISSING and the run PARTIAL — never SUCCESS,
// never zero. Identical inputs under the same version and rounding replay
// the existing run (nothing is written twice).
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
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	commissionshadow "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// DSN environment variables (one login per owner role).
const (
	EnvCommissionURL = "CRM_COMMISSION_DATABASE_URL"
	EnvStagingURL    = "CRM_STAGING_DATABASE_URL"
	EnvIdentityURL   = "CRM_IDENTITY_DATABASE_URL"
	EnvBillingURL    = "CRM_BILLING_DATABASE_URL"
)

const usage = "usage: crm-commission-shadow run --snapshot-id ID --rounding LEGACY_PHP_ROUND_HALF_AWAY_2DP|NONE\n(" + EnvCommissionURL + ", " +
	EnvStagingURL + ", " + EnvIdentityURL + " and " + EnvBillingURL + " required; there is no default rounding)"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type output struct {
	Command    string            `json:"command"`
	RecordKind string            `json:"record_kind"`
	RunID      string            `json:"run_id"`
	Replayed   bool              `json:"replayed"`
	Outcome    result.Status     `json:"outcome"`
	Reasons    []string          `json:"reasons"`
	Cutoff     string            `json:"source_cutoff"`
	Rounding   string            `json:"rounding"`
	Counts     map[string]int64  `json:"fact_counts"`
	Parity     api.ParitySummary `json:"parity"`
	InputSHA   string            `json:"input_sha256"`
	FactsSHA   string            `json:"facts_sha256"`
	Notice     string            `json:"notice"`
}

func parseArgs(args []string) (snapshotID string, rounding domain.Rounding, ok bool) {
	if len(args) != 5 || args[0] != "run" {
		return "", "", false
	}
	var haveRounding bool
	for i := 1; i+1 < len(args); i += 2 {
		switch args[i] {
		case "--snapshot-id":
			snapshotID = args[i+1]
		case "--rounding":
			switch args[i+1] {
			case string(domain.RoundingLegacyHalfAwayTwoDP):
				rounding, haveRounding = domain.RoundingLegacyHalfAwayTwoDP, true
			case "NONE":
				rounding, haveRounding = domain.RoundingNone, true
			default:
				return "", "", false
			}
		default:
			return "", "", false
		}
	}
	return snapshotID, rounding, snapshotID != "" && haveRounding
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	snapshotID, rounding, ok := parseArgs(args)
	if !ok {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	for _, e := range []string{EnvCommissionURL, EnvStagingURL, EnvIdentityURL, EnvBillingURL} {
		if getenv(e) == "" {
			fmt.Fprintln(stderr, "crm-commission-shadow:", e, "is not set (no default target exists)")
			return 2
		}
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
	for m, env := range map[owner.Module]string{commission.Module: EnvCommissionURL, integration.Module: EnvStagingURL, identity.Module: EnvIdentityURL,
		billing.Module: EnvBillingURL} {
		dsn := getenv(env)
		if prev, dup := seen[dsn]; dup {
			fmt.Fprintf(stderr, "crm-commission-shadow: %s must not reuse %s\n", env, prev)
			return 2
		}
		seen[dsn] = env
		dsns[m] = dsn
	}
	runners := map[owner.Module]*db.Runner{}
	for m, dsn := range dsns {
		cfg := db.DefaultConfig(dsn)
		cfg.ApplicationName = "crm-commission-shadow"
		cfg.MaxConns = 2
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "crm-commission-shadow: %s database: %v\n", m, err)
			return 1
		}
		defer pool.Close()
		r, err := pool.Runner(m)
		if err != nil {
			fmt.Fprintln(stderr, "crm-commission-shadow:", err)
			return 1
		}
		runners[m] = r
	}
	src, err := staged(ctx, runners[integration.Module], identityquery.New(runners[identity.Module]), billingquery.New(runners[billing.Module]), snapshotID)
	if err != nil {
		fmt.Fprintln(stderr, "crm-commission-shadow:", err)
		return 1
	}
	cmd, err := commissionshadow.New(runners[commission.Module], clock.System{})
	if err != nil {
		fmt.Fprintln(stderr, "crm-commission-shadow:", err)
		return 1
	}
	rep, err := cmd.Run(ctx, src, rounding)
	if err != nil {
		fmt.Fprintln(stderr, "crm-commission-shadow:", err)
		return 1
	}
	out := output{Command: "run", RecordKind: "SHADOW", RunID: rep.RunID, Replayed: rep.Replayed, Outcome: rep.Outcome, Reasons: []string{},
		Cutoff: rep.Cutoff.UTC().Format("2006-01-02T15:04:05Z"), Rounding: rep.Rounding, Counts: rep.Counts, Parity: rep.Parity,
		InputSHA: fmt.Sprintf("%x", rep.InputSHA), FactsSHA: fmt.Sprintf("%x", rep.FactsSHA), Notice: api.Notice}
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

// staged reads one staged snapshot (READ ONLY) and resolves its user and
// invoice keys through the identity and billing published APIs.
func staged(ctx context.Context, stagedRunner *db.Runner, ident *identityquery.Service, bq *billingquery.Service, snapshotID string) (shadow.StagedCommission, error) {
	b, err := shadow.ReadBatch(ctx, stagedRunner, spec.Default(), snapshotID)
	if err != nil {
		return shadow.StagedCommission{}, err
	}
	users, err := ident.ResolveUsers(ctx, shadow.SourceSystem, shadow.CommissionUserKeys(b))
	if err != nil {
		return shadow.StagedCommission{}, err
	}
	src, err := bq.ResolveInvoices(ctx, shadow.SourceSystem, shadow.SpecInvoice, shadow.CommissionInvoiceKeys(b))
	if err != nil {
		return shadow.StagedCommission{}, err
	}
	inv := map[string]string{}
	for k, v := range src {
		inv[k] = v.InvoiceID
	}
	return shadow.StagedCommission{Batch: b, Users: users, Invoices: inv}, nil
}
