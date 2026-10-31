package main

import (
	"context"
	"errors"
	"fmt"
	"io"

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
	need := map[owner.Module]string{integration.Module: EnvStagingURL, identity.Module: EnvIdentityURL}
	if cmd != "import-identity" {
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
