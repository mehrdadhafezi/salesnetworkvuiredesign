package shadow_test

import (
	"context"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Reconciliation reports SUCCESS only with full coverage; a missing or
// incomplete organization entity is never presented as reconciled.
func TestReconcileRequiresEntityCoverage(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	b := e.stage(t, "synthetic-min-cover-01", legacytest.MinimalHRScenario())
	e.importAll(t, b)
	rep, err := reconcileReport(ctx, e, b)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Organization.Outcome != result.Success || rep.Identity.Outcome != result.Success {
		t.Fatalf("clean scenario: identity=%s organization=%s notes=%v", rep.Identity.Outcome, rep.Organization.Outcome, rep.Organization.Notes)
	}

	// Assignment entity not staged at all: UNKNOWN, not SUCCESS.
	e2 := setup(t)
	b2 := e2.stage(t, "synthetic-min-cover-02", legacytest.Without(legacytest.MinimalHRScenario(), "legacy.hr_assignment.v1"))
	e2.importAll(t, b2)
	rep, err = reconcileReport(ctx, e2, b2)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Organization.Outcome != result.Unknown {
		t.Fatalf("missing assignments: %s %v", rep.Organization.Outcome, rep.Organization.Notes)
	}

	// History staged but INCOMPLETE: PARTIAL.
	e3 := setup(t)
	min := legacytest.MinimalHRScenario()
	var hist []legacytest.Rec
	hist = append(hist, legacytest.Rec{"id": "301", "assignment_id": "201", "child_profile_id": "102", "new_parent_profile_id": "101",
		"actor_user_id": "1", "action": "synthetic_assign", "created_at": "2026-01-01 00:00:00"})
	b3 := e3.stage(t, "synthetic-min-cover-03", legacytest.With(min, "legacy.hr_assignment_history.v1", hist, snapshot.Incomplete))
	e3.importAll(t, b3)
	rep, err = reconcileReport(ctx, e3, b3)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Organization.Outcome != result.Partial {
		t.Fatalf("incomplete history: %s %v", rep.Organization.Outcome, rep.Organization.Notes)
	}
}

// A profile whose account no longer resolves withdraws its earlier grant;
// COMPLETE but empty entities mark every prior row absent.
func TestStaleGrantsAndEmptyCompleteEntities(t *testing.T) {
	e := setup(t)
	b := e.stage(t, "synthetic-min-stale-01", legacytest.MinimalHRScenario())
	e.importAll(t, b)
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant WHERE evidence_key = '101' AND presence = 'PRESENT'`); n != 1 {
		t.Fatalf("grant not imported: %d", n)
	}
	min := legacytest.MinimalHRScenario()
	moved := legacytest.Profile("101", "99", "Synthetic Minimal Lead", "1", "1", "active", "sn_hr", "1", "1", "1")
	seller := legacytest.Profile("102", "2", "Synthetic Minimal Seller", "2", "1", "active", "sn_seller", "1", "1", "1")
	ents := legacytest.With(min, "legacy.hr_profile.v1", []legacytest.Rec{moved, seller}, snapshot.Incomplete)
	ents = legacytest.With(ents, "legacy.hr_team.v1", nil, snapshot.Complete)
	b2 := e.stage(t, "synthetic-min-stale-02", ents)
	e.importAll(t, b2)
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant WHERE evidence_key = '101' AND presence = 'PRESENT'`); n != 0 {
		t.Fatal("stale grant still evaluates for the previous account")
	}
	grants, err := e.ident.Grants(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range grants {
		if g.RoleKey != nil && *g.RoleKey == "hr" {
			t.Fatalf("stale hr grant returned: %+v", g)
		}
	}
	if n := e.count(t, `SELECT count(*) FROM organization.org_unit WHERE kind = 'TEAM' AND presence = 'PRESENT'`); n != 0 {
		t.Fatal("empty COMPLETE team entity left teams PRESENT")
	}

	// An empty COMPLETE account entity marks every account absent (never deleted).
	b3 := e.stage(t, "synthetic-min-stale-03", legacytest.With(min, "legacy.wp_user.v1", nil, snapshot.Complete))
	e.importAll(t, b3)
	if n := e.count(t, `SELECT count(*) FROM identity.user_account WHERE presence = 'PRESENT'`); n != 0 {
		t.Fatal("empty COMPLETE user entity left accounts PRESENT")
	}
	if n := e.count(t, `SELECT count(*) FROM identity.user_account`); n != 2 {
		t.Fatalf("accounts deleted: %d", n)
	}
}

func reconcileReport(ctx context.Context, e *env, b *shadow.Batch) (shadow.Report, error) {
	return shadow.Reconcile(ctx, b, e.ident, e.org)
}
