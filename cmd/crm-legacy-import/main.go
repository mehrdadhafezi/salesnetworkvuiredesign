// Command crm-legacy-import is the offline legacy snapshot tool (Architecture
// V1 §21, §24; Slice 3). It reads intentionally supplied snapshot packages
// from disk and stages them raw into the integration schema. It never
// connects to WordPress/MySQL, never calls a legacy endpoint and never writes
// to any legacy system; the target API server does not import it.
//
//	crm-legacy-import specs                                  published extract specs (JSON)
//	crm-legacy-import inspect   <snapshot-dir>               manifest summary (no table scan)
//	crm-legacy-import validate  <snapshot-dir>               full offline validation (JSON report)
//	crm-legacy-import load      <snapshot-dir>               stage a SYNTHETIC package (others are BLOCKED)
//	crm-legacy-import reconcile <snapshot-id>                staging vs manifest (read-only)
//	crm-legacy-import import-identity <snapshot-id>          Slice 4: staging → identity shadow model
//	crm-legacy-import import-organization <snapshot-id>      Slice 4: staging → organization shadow model
//	crm-legacy-import reconcile-identity-org <snapshot-id>   Slice 4: counts, hierarchy findings, capability diff (read-only)
//
// load and reconcile need CRM_STAGING_DATABASE_URL: a login dedicated to the
// integration owner role (the platform runner refuses any other). It must
// differ from the application and migrator DSNs. The Slice 4 commands also
// need CRM_IDENTITY_DATABASE_URL and (organization, reconcile)
// CRM_ORGANIZATION_DATABASE_URL, each a login dedicated to that module's
// owner role and distinct from every other DSN. Each module writes only in
// its own transaction: identity first, then organization (which resolves
// accounts through the published identity API). Nothing here runs on API
// startup, and no command reads WordPress, MySQL or snapshot files.
//
// Exit status: 0 SUCCESS · 3 any other outcome (PARTIAL, UNKNOWN, CONFLICT,
// BLOCKED, FAILED) · 1 infrastructure error · 2 usage. Reports never contain
// row values.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/staging"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
)

// EnvStagingURL is the staging (target) DSN. There is deliberately no legacy
// source DSN: Slice 3 imports snapshot packages only.
const EnvStagingURL = "CRM_STAGING_DATABASE_URL"

// Slice 4 module DSNs (one login per owner role).
const (
	EnvIdentityURL     = "CRM_IDENTITY_DATABASE_URL"
	EnvOrganizationURL = "CRM_ORGANIZATION_DATABASE_URL"
	// EnvCustomersURL is the Slice 5 customers DSN (customers-owner login).
	EnvCustomersURL = "CRM_CUSTOMERS_DATABASE_URL"
)

// ToolVersion is recorded on every batch.
const ToolVersion = "crm-legacy-import/1.0.0"

const usage = `usage:
  crm-legacy-import specs
  crm-legacy-import inspect   <snapshot-dir>
  crm-legacy-import validate  <snapshot-dir>
  crm-legacy-import load      <snapshot-dir>
  crm-legacy-import reconcile <snapshot-id>
  crm-legacy-import import-identity <snapshot-id>
  crm-legacy-import import-organization <snapshot-id>
  crm-legacy-import reconcile-identity-org <snapshot-id>
  crm-legacy-import import-customers <snapshot-id>
  crm-legacy-import reconcile-customers <snapshot-id>
(` + EnvStagingURL + `; Slice 4 also ` + EnvIdentityURL + ` and ` + EnvOrganizationURL + `;
 Slice 5 also ` + EnvCustomersURL + `)`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	reg := spec.Default()
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "specs":
		if len(rest) != 0 {
			break
		}
		return emit(stdout, stderr, reg.Specs(), true)
	case "inspect", "validate":
		if len(rest) != 1 {
			break
		}
		p, err := snapshot.Open(rest[0], reg)
		if err != nil {
			return emit(stdout, stderr, invalidPackage(cmd, err), false)
		}
		if cmd == "inspect" {
			return emit(stdout, stderr, inspect(p), true)
		}
		reps, err := p.Validate()
		if err != nil {
			fmt.Fprintln(stderr, "crm-legacy-import:", err)
			return 1
		}
		out := validation{Command: cmd, SnapshotID: p.Manifest.SnapshotID, ContentSHA256: p.Manifest.ContentIdentity(),
			ManifestSHA256: p.ManifestSHA256, Entities: reps}
		statuses := make([]result.Status, len(reps))
		for i, r := range reps {
			statuses[i] = r.Status
		}
		out.Outcome, _ = result.Aggregate(statuses)
		return emit(stdout, stderr, out, out.Outcome == result.Success)
	case "load":
		// No flags: non-synthetic packages are BLOCKED by the loader and
		// nothing on the command line can override that.
		if len(rest) != 1 {
			break
		}
		p, err := snapshot.Open(rest[0], reg)
		if err != nil {
			return emit(stdout, stderr, invalidPackage(cmd, err), false)
		}
		return withLoader(ctx, getenv, stderr, func(l *staging.Loader) int {
			rep, err := l.Load(ctx, p)
			code := emit(stdout, stderr, rep, rep.Outcome == result.Success)
			if err != nil {
				fmt.Fprintln(stderr, "crm-legacy-import:", err)
				return 1
			}
			return code
		})
	case "import-identity", "import-organization", "reconcile-identity-org", "import-customers", "reconcile-customers":
		if len(rest) != 1 {
			break
		}
		return runShadow(ctx, cmd, rest[0], reg, getenv, stdout, stderr)
	case "reconcile":
		if len(rest) != 1 {
			break
		}
		return withLoader(ctx, getenv, stderr, func(l *staging.Loader) int {
			rep, err := l.Reconcile(ctx, reg, rest[0])
			code := emit(stdout, stderr, rep, rep.Outcome == result.Success)
			if err != nil {
				fmt.Fprintln(stderr, "crm-legacy-import:", err)
				return 1
			}
			return code
		})
	}
	fmt.Fprintln(stderr, usage)
	return 2
}

func withLoader(ctx context.Context, getenv func(string) string, stderr io.Writer, fn func(*staging.Loader) int) int {
	dsn := getenv(EnvStagingURL)
	if dsn == "" {
		fmt.Fprintln(stderr, "crm-legacy-import: "+EnvStagingURL+" is not set (no default target exists)")
		return 2
	}
	for _, other := range []string{db.EnvDatabaseURL, migrations.EnvMigrateURL} {
		if v := getenv(other); v != "" && v == dsn {
			fmt.Fprintln(stderr, "crm-legacy-import: "+EnvStagingURL+" must not reuse "+other)
			return 2
		}
	}
	cfg := db.DefaultConfig(dsn)
	cfg.ApplicationName = "crm-legacy-import"
	cfg.MaxConns = 2
	pool, err := db.Open(ctx, cfg)
	if err != nil {
		fmt.Fprintln(stderr, "crm-legacy-import: staging database:", err)
		return 1
	}
	defer pool.Close()
	runner, err := pool.Runner(integration.Module)
	if err != nil {
		fmt.Fprintln(stderr, "crm-legacy-import:", err)
		return 1
	}
	l, err := staging.NewLoader(runner, clock.System{}, ToolVersion)
	if err != nil {
		fmt.Fprintln(stderr, "crm-legacy-import:", err)
		return 1
	}
	return fn(l)
}

type invalid struct {
	Command string        `json:"command"`
	Outcome result.Status `json:"outcome"`
	Reason  string        `json:"reason_code"`
	Problem string        `json:"problem"`
}

func invalidPackage(cmd string, err error) invalid {
	return invalid{Command: cmd, Outcome: result.Failed, Reason: "CONTRACT_INVALID", Problem: err.Error()}
}

type validation struct {
	Command        string                  `json:"command"`
	SnapshotID     string                  `json:"snapshot_id"`
	ContentSHA256  string                  `json:"content_sha256"`
	ManifestSHA256 string                  `json:"manifest_sha256"`
	Outcome        result.Status           `json:"outcome"`
	Entities       []snapshot.EntityReport `json:"entities"`
}

type inspection struct {
	Command        string            `json:"command"`
	SnapshotID     string            `json:"snapshot_id"`
	Legacy         snapshot.Legacy   `json:"legacy"`
	Source         snapshot.Source   `json:"source"`
	DataOrigin     string            `json:"data_origin"`
	ExtractedAt    string            `json:"extracted_at"`
	ContentSHA256  string            `json:"content_sha256"`
	ManifestSHA256 string            `json:"manifest_sha256"`
	Entities       []inspectedEntity `json:"entities"`
	Warnings       []string          `json:"warnings"`
}

type inspectedEntity struct {
	SpecID         string `json:"spec_id"`
	SourceTable    string `json:"source_table"`
	Classification string `json:"classification"`
	RowCount       int64  `json:"row_count"`
	ByteSize       int64  `json:"byte_size"`
	Completeness   string `json:"completeness"`
}

func inspect(p *snapshot.Package) inspection {
	m := p.Manifest
	out := inspection{Command: "inspect", SnapshotID: m.SnapshotID, Legacy: m.Legacy, Source: m.Source,
		DataOrigin: string(m.DataOrigin), ExtractedAt: m.ExtractedAt, ContentSHA256: m.ContentIdentity(),
		ManifestSHA256: p.ManifestSHA256, Warnings: m.Warnings}
	for _, e := range m.Entities {
		s, _ := p.Spec(e.SpecID)
		out.Entities = append(out.Entities, inspectedEntity{SpecID: e.SpecID, SourceTable: s.SourceTable,
			Classification: string(s.Classification()), RowCount: e.RowCount, ByteSize: e.ByteSize, Completeness: string(e.Completeness)})
	}
	return out
}

func emit(stdout, stderr io.Writer, v any, success bool) int {
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(stderr, "crm-legacy-import:", err)
		return 1
	}
	if success {
		return 0
	}
	return 3
}
