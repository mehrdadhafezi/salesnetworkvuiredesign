// Command crm-biawin-shadow is the OFFLINE Biawin / fulfillment READ shadow
// job (Slice 10). It is never run by the API server and never triggered by a
// GET, page render, session or report request. It reads ONE staged legacy
// snapshot, resolves user and invoice keys through the identity and billing
// published APIs, and records run-pinned, append-only shadow facts plus the
// active subscription parity. It never activates, suspends or terminates a
// subscription, never calls Biawin (no order creation, no wallet charge, no
// network of any kind), never ships, assigns or completes anything, never
// writes another schema and never writes the legacy system.
//
//	crm-biawin-shadow import --snapshot-id ID
//	crm-biawin-shadow parity --snapshot-id ID [--run-id RUN]
//
// import records a biawin run (with its parity against the same snapshot)
// and a fulfillment run. parity compares the legacy evidence of a snapshot
// against an EXISTING biawin run (default: the latest) — e.g. a newer
// snapshot against an older run shows the target as stale.
//
// Required logins (one per owner role, never reused): CRM_BIAWIN_DATABASE_URL,
// CRM_FULFILLMENT_DATABASE_URL (import only), CRM_STAGING_DATABASE_URL,
// CRM_IDENTITY_DATABASE_URL, CRM_BILLING_DATABASE_URL. Identical inputs under
// the same normalization version replay the existing run (nothing is
// written twice).
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
	"regexp"
	"syscall"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	biawinimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/imports"
	biawinquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	billingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	fulfillmentimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/imports"
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
	EnvBiawinURL      = "CRM_BIAWIN_DATABASE_URL"
	EnvFulfillmentURL = "CRM_FULFILLMENT_DATABASE_URL"
	EnvStagingURL     = "CRM_STAGING_DATABASE_URL"
	EnvIdentityURL    = "CRM_IDENTITY_DATABASE_URL"
	EnvBillingURL     = "CRM_BILLING_DATABASE_URL"
)

const usage = "usage: crm-biawin-shadow import --snapshot-id ID | parity --snapshot-id ID [--run-id RUN]\n(" + EnvBiawinURL + ", " +
	EnvStagingURL + ", " + EnvIdentityURL + ", " + EnvBillingURL + " required; import also needs " + EnvFulfillmentURL + ")"

const notice = "SHADOW only: offline import of staged legacy evidence; nothing was activated, suspended, terminated, shipped, assigned or completed, and no provider was called"

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

type runOut struct {
	RunID    string           `json:"run_id"`
	Replayed bool             `json:"replayed"`
	Outcome  result.Status    `json:"outcome"`
	Reasons  []string         `json:"reasons"`
	Cutoff   string           `json:"source_cutoff"`
	Counts   map[string]int64 `json:"counts"`
	InputSHA string           `json:"input_sha256"`
	FactsSHA string           `json:"facts_sha256"`
}

type parityOut struct {
	CheckID        string   `json:"check_id"`
	RunID          string   `json:"run_id"`
	EvidenceStatus string   `json:"evidence_status"`
	Outcome        string   `json:"outcome"`
	Reasons        []string `json:"reasons"`
	LegacyCount    int64    `json:"legacy_count"`
	TargetCount    int64    `json:"target_count"`
	Matched        int64    `json:"matched"`
	MissingTarget  int64    `json:"missing_target"`
	ExtraTarget    int64    `json:"extra_target"`
	Unknown        int64    `json:"unknown"`
	Conflict       int64    `json:"conflict"`
	NotRecoverable int64    `json:"not_recoverable"`
}

type output struct {
	Command     string     `json:"command"`
	RecordKind  string     `json:"record_kind"`
	SnapshotID  string     `json:"snapshot_id"`
	Biawin      *runOut    `json:"biawin,omitempty"`
	Fulfillment *runOut    `json:"fulfillment,omitempty"`
	Parity      *parityOut `json:"parity,omitempty"`
	Notice      string     `json:"notice"`
}

type args struct {
	command, snapshotID, runID string
}

func parse(a []string) (args, bool) {
	var out args
	if len(a) < 3 || (a[0] != "import" && a[0] != "parity") {
		return out, false
	}
	out.command = a[0]
	for i := 1; i < len(a); i += 2 {
		if i+1 >= len(a) || a[i+1] == "" {
			return out, false
		}
		switch {
		case a[i] == "--snapshot-id" && out.snapshotID == "":
			out.snapshotID = a[i+1]
		case a[i] == "--run-id" && out.command == "parity" && out.runID == "" && uuidPattern.MatchString(a[i+1]):
			out.runID = a[i+1]
		default:
			return out, false
		}
	}
	return out, out.snapshotID != ""
}

func run(ctx context.Context, argv []string, getenv func(string) string, stdout, stderr io.Writer) int {
	a, ok := parse(argv)
	if !ok {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	need := map[owner.Module]string{biawin.Module: EnvBiawinURL, integration.Module: EnvStagingURL, identity.Module: EnvIdentityURL,
		billing.Module: EnvBillingURL}
	if a.command == "import" {
		need[fulfillment.Module] = EnvFulfillmentURL
	}
	seen := map[string]string{}
	for _, other := range []string{db.EnvDatabaseURL, migrations.EnvMigrateURL} {
		if v := getenv(other); v != "" {
			seen[v] = other
		}
	}
	// Validate every DSN before opening any connection: no default target,
	// no reuse of another login (each owner role has its own).
	dsns := map[owner.Module]string{}
	for _, m := range []owner.Module{biawin.Module, fulfillment.Module, integration.Module, identity.Module, billing.Module} {
		env, ok := need[m]
		if !ok {
			continue
		}
		dsn := getenv(env)
		if dsn == "" {
			fmt.Fprintln(stderr, "crm-biawin-shadow:", env, "is not set (no default target exists)")
			return 2
		}
		if prev, dup := seen[dsn]; dup {
			fmt.Fprintf(stderr, "crm-biawin-shadow: %s must not reuse %s\n", env, prev)
			return 2
		}
		seen[dsn] = env
		dsns[m] = dsn
	}
	runners := map[owner.Module]*db.Runner{}
	for m, dsn := range dsns {
		cfg := db.DefaultConfig(dsn)
		cfg.ApplicationName = "crm-biawin-shadow"
		cfg.MaxConns = 2
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			fmt.Fprintf(stderr, "crm-biawin-shadow: %s database: %v\n", m, err)
			return 1
		}
		defer pool.Close()
		r, err := pool.Runner(m)
		if err != nil {
			fmt.Fprintln(stderr, "crm-biawin-shadow:", err)
			return 1
		}
		runners[m] = r
	}
	code, err := execute(ctx, a, runners, stdout)
	if err != nil {
		fmt.Fprintln(stderr, "crm-biawin-shadow:", err)
		return 1
	}
	return code
}

// staged reads one staged snapshot (READ ONLY) and resolves its user and
// invoice keys through the identity and billing published APIs.
func staged(ctx context.Context, runners map[owner.Module]*db.Runner, snapshotID string) (*shadow.Batch, shadow.Users, shadow.Invoices, error) {
	b, err := shadow.ReadBatch(ctx, runners[integration.Module], spec.Default(), snapshotID)
	if err != nil {
		return nil, nil, nil, err
	}
	users, err := identityquery.New(runners[identity.Module]).ResolveUsers(ctx, shadow.SourceSystem, shadow.BiawinUserKeys(b))
	if err != nil {
		return nil, nil, nil, err
	}
	want := map[string]bool{}
	for _, k := range shadow.BiawinInvoiceKeys(b) {
		want[k] = true
	}
	inv := shadow.Invoices{}
	if err := billingquery.New(runners[billing.Module]).InvoiceFacts(ctx, func(f billingapi.InvoiceFact) error {
		if want[f.SourceKey] {
			inv[f.SourceKey] = f
		}
		return nil
	}); err != nil {
		return nil, nil, nil, err
	}
	return b, users, inv, nil
}

func runView(r biawinimports.Run) *runOut {
	out := &runOut{RunID: r.RunID, Replayed: r.Replayed, Outcome: r.Outcome, Reasons: []string{}, Cutoff: r.Cutoff.UTC().Format("2006-01-02T15:04:05Z"),
		Counts: r.Counts, InputSHA: fmt.Sprintf("%x", r.InputSHA), FactsSHA: fmt.Sprintf("%x", r.FactsSHA)}
	for _, c := range r.Reasons {
		out.Reasons = append(out.Reasons, string(c))
	}
	return out
}

func parityView(p *biawinimports.ParityResult, runID, evidenceStatus string) *parityOut {
	if p == nil {
		return nil
	}
	out := &parityOut{CheckID: p.CheckID, RunID: runID, EvidenceStatus: evidenceStatus, Outcome: p.Outcome, Reasons: []string{},
		LegacyCount: p.LegacyCount, TargetCount: p.TargetCount, Matched: p.Matched, MissingTarget: p.MissingTarget, ExtraTarget: p.ExtraTarget,
		Unknown: p.Unknown, Conflict: p.Conflict, NotRecoverable: p.NotRec}
	for _, c := range p.Reasons {
		out.Reasons = append(out.Reasons, string(c))
	}
	return out
}

func execute(ctx context.Context, a args, runners map[owner.Module]*db.Runner, stdout io.Writer) (int, error) {
	b, users, inv, err := staged(ctx, runners, a.snapshotID)
	if err != nil {
		return 0, err
	}
	in, ev := shadow.BiawinCommand(b, users, inv)
	bc, err := biawinimports.New(runners[biawin.Module], clock.System{})
	if err != nil {
		return 0, err
	}
	out := output{Command: a.command, RecordKind: "SHADOW", SnapshotID: a.snapshotID, Notice: notice}
	ok := true
	switch a.command {
	case "import":
		fc, err := fulfillmentimports.New(runners[fulfillment.Module], clock.System{})
		if err != nil {
			return 0, err
		}
		br, err := bc.Import(ctx, in, ev)
		if err != nil {
			return 0, err
		}
		fr, err := fc.Import(ctx, shadow.FulfillmentCommand(b, users, inv))
		if err != nil {
			return 0, err
		}
		out.Biawin = runView(br)
		f := runView(biawinimports.Run{RunID: fr.RunID, Replayed: fr.Replayed, Outcome: fr.Outcome, Reasons: fr.Reasons, Counts: fr.Counts,
			InputSHA: fr.InputSHA, FactsSHA: fr.FactsSHA, Cutoff: fr.Cutoff})
		out.Fulfillment = f
		out.Parity = parityView(br.Parity, br.RunID, ev.Status)
		ok = br.Outcome == result.Success && fr.Outcome == result.Success
	case "parity":
		q, err := biawinquery.New(runners[biawin.Module], clock.System{})
		if err != nil {
			return 0, err
		}
		var rid *string
		if a.runID != "" {
			rid = &a.runID
		}
		target, err := q.RunByID(ctx, rid)
		if err != nil {
			return 0, err
		}
		p, err := bc.Parity(ctx, target.RunID, ev)
		if err != nil {
			return 0, err
		}
		out.Parity = parityView(p, target.RunID, ev.Status)
		ok = p.Outcome == "MATCH"
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return 1, nil
	}
	if !ok {
		return 3, nil
	}
	return 0, nil
}
