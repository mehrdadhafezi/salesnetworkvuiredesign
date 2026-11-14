package shadow_test

// R7-5 — deterministic integrated SYNTHETIC dataset (one coherent synthetic
// world). The source is legacytest.IntegratedScenario (existing read-world
// fixtures, composed); it is seeded into a DISPOSABLE pgtest database through
// the existing supported import pathway only:
//
//	stage (synthetic loader) → identity + organization → sales (cases,
//	custody) → customers → billing → payments → biawin + fulfillment shadow
//	→ finance shadow ledger → commission shadow → reporting projections.
//
// Nothing is written by a native command: no HR request, payment
// submission, work case, provider operation, credential or session is
// created, no feature flag is enabled and no grant is activated. See
// docs/remediation/r7-5-synthetic-dataset.md for the relationship map and
// the scenarios that cannot be seeded safely.

import (
	"context"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/domain"
	commissionmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/migrations"
	commissionshadow "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	financeshadow "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting"
	reportingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/projections"
	reportingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/query"
)

// integratedEnv reuses the per-module environments of the existing suites
// on ONE disposable database (each keeps its own owner runner and clock).
type integratedEnv struct {
	*billingEnv
	bw  *biawinEnv
	fin *financeEnv
	com *commissionEnv
	rep *reportingEnv
}

func setupIntegrated(t *testing.T) *integratedEnv {
	t.Helper()
	bw := setupBiawin(t, financemigrations.Source(), commissionmigrations.Source(), reportingmigrations.Source()) // billing + biawin + fulfillment + these
	e := bw.billingEnv
	for _, s := range []migrations.Source{financemigrations.Source(), commissionmigrations.Source(), reportingmigrations.Source()} {
		if _, err := migrations.Up(context.Background(), e.DSN, s); err != nil {
			t.Fatalf("migrations: %v", err)
		}
	}
	finAt, _ := timeutil.ParseRFC3339("2026-11-01T09:00:00Z")
	fr := e.Runner(t, finance.Module)
	fcmd, err := financeshadow.New(fr, timeutil.FixedClock{At: finAt})
	if err != nil {
		t.Fatal(err)
	}
	comAt, _ := timeutil.ParseRFC3339("2026-11-03T09:00:00Z")
	cr := e.Runner(t, commission.Module)
	ccmd, err := commissionshadow.New(cr, timeutil.FixedClock{At: comAt})
	if err != nil {
		t.Fatal(err)
	}
	repAt, _ := timeutil.ParseRFC3339("2026-11-01T09:00:00Z")
	rr := e.Runner(t, reporting.Module)
	pcmd, err := projections.New(rr, timeutil.FixedClock{At: repAt})
	if err != nil {
		t.Fatal(err)
	}
	return &integratedEnv{billingEnv: e, bw: bw,
		fin: &financeEnv{billingEnv: e, rebuild: fcmd, fin: fr},
		com: &commissionEnv{billingEnv: e, cmd: ccmd, cr: cr},
		rep: &reportingEnv{billingEnv: e, rebuild: pcmd, q: reportingquery.New(rr, timeutil.FixedClock{At: repAt})}}
}

// seedStep is one import step and its outcome.
type seedStep struct {
	Step     string
	Outcome  result.Status
	Replayed bool
}

// seedIntegrated stages the integrated package (the batch is staged once per
// database) and runs every import step in dependency order.
func (e *integratedEnv) seedIntegrated(t *testing.T) (*shadow.Batch, []seedStep) {
	t.Helper()
	b := e.prepare(t, legacytest.IntegratedSnapshotID, legacytest.IntegratedScenario()) // stage, identity, organization, sales, customers
	return b, e.importModules(t, b)
}

// importModules runs the module imports and projections on a staged batch.
func (e *integratedEnv) importModules(t *testing.T, b *shadow.Batch) []seedStep {
	t.Helper()
	ctx := context.Background()
	var steps []seedStep
	steps = append(steps, seedStep{Step: "billing", Outcome: e.importBillingOnly(t, b)})
	pay, replay := e.importPaymentsOnly(t, b)
	steps = append(steps, seedStep{Step: "payments", Outcome: pay, Replayed: replay})
	br, fr := e.bw.importBoth(t, b)
	steps = append(steps, seedStep{"biawin", br.Outcome, br.Replayed}, seedStep{"fulfillment", fr.Outcome, fr.Replayed})
	fin, err := e.fin.rebuild.Rebuild(ctx, e.fin.sources(e.fin.wallets(t, b)))
	if err != nil {
		t.Fatalf("finance shadow: %v", err)
	}
	steps = append(steps, seedStep{"finance", fin.Outcome, fin.Replayed})
	com, err := e.com.cmd.Run(ctx, e.com.staged(t, b), domain.RoundingLegacyHalfAwayTwoDP)
	if err != nil {
		t.Fatalf("commission shadow: %v", err)
	}
	steps = append(steps, seedStep{"commission", com.Outcome, com.Replayed})
	rep, err := e.rep.rebuild.Rebuild(ctx, e.rep.sources())
	if err != nil {
		t.Fatalf("reporting projections: %v", err)
	}
	steps = append(steps, seedStep{Step: "reporting", Outcome: rep.Outcome})
	return steps
}

// r75Schemas are every schema the dataset may populate.
var r75Schemas = []string{"integration", "identity", "organization", "cases", "custody", "customers", "billing", "payments", "biawin",
	"fulfillment", "finance", "commission", "reporting", "catalog", "platform_featureflags", "audit"}
