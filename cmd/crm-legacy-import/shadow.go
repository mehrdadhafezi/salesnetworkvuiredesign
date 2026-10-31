package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casesimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/imports"
	casesquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	custodyimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/imports"
	custodyquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customersimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/imports"
	customersquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/imports"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationimports "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/imports"
	organizationquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// shadowResult is the JSON output of the Slice 4 import commands.
type shadowResult struct {
	Command    string             `json:"command"`
	SnapshotID string             `json:"snapshot_id"`
	BatchID    string             `json:"batch_id,omitempty"`
	Outcome    result.Status      `json:"outcome"`
	Reasons    []reasoncodes.Code `json:"reasons"`
	Replay     bool               `json:"replay"`
	RunID      string             `json:"run_id,omitempty"`
	Mapping    map[string]int64   `json:"mapping"`
	Counts     map[string]int64   `json:"counts"`
	Findings   map[string]int64   `json:"findings,omitempty"`
	Issues     any                `json:"issues"`
	Problem    string             `json:"problem,omitempty"`
}

// worst returns the less successful of two outcomes (never hides a failure).
func worst(a, b result.Status) result.Status {
	rank := map[result.Status]int{result.Success: 0, result.Partial: 1, result.Unknown: 2, result.Conflict: 3, result.Blocked: 4, result.Failed: 5}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

func nonNil(m map[string]int64) map[string]int64 {
	if m == nil {
		return map[string]int64{}
	}
	return m
}

func openRunner(ctx context.Context, dsn string, m owner.Module) (*db.Runner, func(), error) {
	cfg := db.DefaultConfig(dsn)
	cfg.ApplicationName = "crm-legacy-import"
	cfg.MaxConns = 2
	pool, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	r, err := pool.Runner(m)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	return r, pool.Close, nil
}

// shadowDSNs returns the DSNs a Slice 4 command needs, refusing any reuse.
func shadowDSNs(cmd string, getenv func(string) string) (map[owner.Module]string, error) {
	need := map[owner.Module]string{integration.Module: EnvStagingURL}
	switch cmd {
	case "import-customers", "reconcile-customers":
		// Slice 5 touches staging (read) and customers only.
		need[customers.Module] = EnvCustomersURL
	case "import-sales":
		// Slice 6: staging (read), identity (read: user references),
		// cases then custody — each in its own module transaction.
		need[identity.Module] = EnvIdentityURL
		need[cases.Module] = EnvCasesURL
		need[custody.Module] = EnvCustodyURL
	case "reconcile-sales":
		need[cases.Module] = EnvCasesURL
		need[custody.Module] = EnvCustodyURL
	case "import-identity":
		need[identity.Module] = EnvIdentityURL
	default:
		need[identity.Module] = EnvIdentityURL
		need[organization.Module] = EnvOrganizationURL
	}
	out := map[owner.Module]string{}
	seen := map[string]string{}
	for _, other := range []string{db.EnvDatabaseURL, migrations.EnvMigrateURL} {
		if v := getenv(other); v != "" {
			seen[v] = other
		}
	}
	for m, env := range need {
		v := getenv(env)
		if v == "" {
			return nil, fmt.Errorf("%s is not set (no default target exists)", env)
		}
		if prev, dup := seen[v]; dup {
			return nil, fmt.Errorf("%s must not reuse %s", env, prev)
		}
		seen[v] = env
		out[m] = v
	}
	return out, nil
}

func runShadow(ctx context.Context, cmd, snapshotID string, reg *spec.Registry, getenv func(string) string, stdout, stderr io.Writer) int {
	dsns, err := shadowDSNs(cmd, getenv)
	if err != nil {
		fmt.Fprintln(stderr, "crm-legacy-import:", err)
		return 2
	}
	runners := map[owner.Module]*db.Runner{}
	for m, dsn := range dsns {
		r, closeFn, err := openRunner(ctx, dsn, m)
		if err != nil {
			fmt.Fprintf(stderr, "crm-legacy-import: %s database: %v\n", m, err)
			return 1
		}
		defer closeFn()
		runners[m] = r
	}
	out := shadowResult{Command: cmd, SnapshotID: snapshotID, Reasons: []reasoncodes.Code{}, Mapping: map[string]int64{}, Counts: map[string]int64{}, Issues: []any{}}
	b, err := shadow.ReadBatch(ctx, runners[integration.Module], reg, snapshotID)
	switch {
	case errors.Is(err, shadow.ErrBatchNotFound):
		out.Outcome, out.Reasons, out.Problem = result.Failed, []reasoncodes.Code{reasoncodes.SourceUnavailable}, "no staged batch for snapshot_id "+snapshotID
		return emit(stdout, stderr, out, false)
	case errors.Is(err, shadow.ErrBatchNotLoaded):
		out.Outcome, out.Reasons, out.Problem = result.Blocked, []reasoncodes.Code{reasoncodes.CoverageIncomplete}, err.Error()
		return emit(stdout, stderr, out, false)
	case err != nil:
		fmt.Fprintln(stderr, "crm-legacy-import:", err)
		return 1
	}
	out.BatchID = b.BatchID
	clk := clock.System{}
	switch cmd {
	case "import-customers":
		// Staging was read in its own read-only transaction above; the
		// customers command commits in one customers-only transaction.
		in, mapping := shadow.CustomersCommand(b)
		rep, err := customersimports.New(runners[customers.Module], clk).ImportCustomers(ctx, in)
		out.Outcome, out.Replay, out.RunID, out.Mapping, out.Counts = rep.Outcome, rep.Replay, rep.RunID, mapping, nonNil(rep.Counts)
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			emit(stdout, stderr, out, false)
			return 1
		}
		if out.Outcome != result.Success && len(out.Reasons) == 0 {
			out.Reasons = []reasoncodes.Code{reasoncodes.CoverageIncomplete}
		}
		return emit(stdout, stderr, out, out.Outcome == result.Success)
	case "import-sales":
		users, err := identityquery.New(runners[identity.Module]).ResolveUsers(ctx, shadow.SourceSystem, shadow.UserKeys(b))
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			return 1
		}
		casesQ := casesquery.New(runners[cases.Module])
		// 1. cases (cases transaction only).
		cin, cmap := shadow.CasesCommand(b, users)
		crep, err := casesimports.New(runners[cases.Module], clk).ImportCases(ctx, cin)
		if err != nil {
			out.Outcome, out.Counts = crep.Outcome, nonNil(crep.Counts)
			fmt.Fprintln(stderr, "crm-legacy-import: cases:", err)
			emit(stdout, stderr, out, false)
			return 1
		}
		// 2. custody resolves Cases through the published cases API and
		//    writes its own schema (custody transaction only).
		sin, smap := shadow.CustodyCommand(b, users)
		srep, err := custodyimports.New(runners[custody.Module], casesQ, clk).ImportCustody(ctx, sin)
		out.Mapping = map[string]int64{}
		for k, v := range cmap {
			out.Mapping["cases."+k] = v
		}
		for k, v := range smap {
			out.Mapping["custody."+k] = v
		}
		out.Counts = map[string]int64{}
		for k, v := range crep.Counts {
			out.Counts["cases."+k] = v
		}
		for k, v := range srep.Counts {
			out.Counts["custody."+k] = v
		}
		out.Replay, out.RunID = crep.Replay && srep.Replay, crep.RunID
		out.Outcome = worst(crep.Outcome, srep.Outcome)
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import: custody:", err)
			emit(stdout, stderr, out, false)
			return 1
		}
		if out.Outcome != result.Success && len(out.Reasons) == 0 {
			out.Reasons = []reasoncodes.Code{reasoncodes.CoverageIncomplete}
		}
		return emit(stdout, stderr, out, out.Outcome == result.Success)
	case "reconcile-sales":
		rep, err := shadow.ReconcileSales(ctx, b, casesquery.New(runners[cases.Module]), custodyquery.New(runners[custody.Module]))
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			return 1
		}
		return emit(stdout, stderr, rep, rep.Outcome == result.Success)
	case "reconcile-customers":
		rep, err := shadow.ReconcileCustomers(ctx, b, customersquery.New(runners[customers.Module]))
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			return 1
		}
		// Non-zero exit on any invariant violation (phone-only RESOLVED or
		// merge) and on any non-SUCCESS outcome.
		return emit(stdout, stderr, rep, rep.Outcome == result.Success)
	}
	identQuery := identityquery.New(runners[identity.Module])
	switch cmd {
	case "import-identity":
		in, mapping := shadow.IdentityCommand(b)
		rep, err := identityimports.New(runners[identity.Module], clk).ImportIdentity(ctx, in)
		out.Outcome, out.Replay, out.RunID, out.Mapping, out.Counts, out.Issues = rep.Outcome, rep.Replay, rep.RunID, mapping, rep.Counts, rep.Issues
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			emit(stdout, stderr, out, false)
			return 1
		}
	case "import-organization":
		in, mapping := shadow.OrganizationCommand(b)
		rep, err := organizationimports.New(runners[organization.Module], identQuery, clk).ImportOrganization(ctx, in)
		out.Outcome, out.Replay, out.RunID, out.Mapping, out.Counts, out.Findings, out.Issues = rep.Outcome, rep.Replay, rep.RunID, mapping, rep.Counts, rep.Findings, rep.Issues
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			emit(stdout, stderr, out, false)
			return 1
		}
	case "reconcile-identity-org":
		rep, err := shadow.Reconcile(ctx, b, identQuery, organizationquery.New(runners[organization.Module]))
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			return 1
		}
		return emit(stdout, stderr, rep, rep.Outcome == result.Success)
	}
	if out.Outcome != result.Success && len(out.Reasons) == 0 {
		out.Reasons = []reasoncodes.Code{reasoncodes.CoverageIncomplete}
	}
	return emit(stdout, stderr, out, out.Outcome == result.Success)
}
