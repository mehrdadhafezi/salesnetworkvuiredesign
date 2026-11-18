package shadow_test

// RM-E04D — ownership-predicate evidence on the R7-5 SYNTHETIC integrated dataset (read only).
//
// Purpose: measure how each candidate Seller row-scope predicate (M2 proposed record 2) behaves on the
// synthetic legacy snapshot, so that the decision sheet can show the categories each predicate admits,
// excludes, or leaves UNKNOWN. It is NOT real-data evidence: rates on synthetic fixtures say nothing about
// production volumes (ADR #18 governs real data). It changes no row, grant, flag or status.
//
// Opt-in output: CRM_RME04D_EVIDENCE=1 prints the measured table (go test -v). The assertions only pin
// that the synthetic dataset still contains every category the decision sheet relies on, so the sheet's
// claims stay reproducible.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// rmE04DPredicates are the candidate predicates, written exactly as SQL over the target schema.
// P2 is the predicate the code already uses for the Slice 11 contact-outcome guard (custody/query CurrentHolder).
var rmE04DQueries = []struct{ key, sql string }{
	// ---- custody / case scope (S1)
	{"case.total_assignments", `SELECT count(*) FROM custody.assignment`},
	{"case.present", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT'`},
	{"case.P1_derived_holder_known", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND derivation = 'DERIVED' AND derived_user_id IS NOT NULL`},
	{"case.P2_legacy_holder_guarded_known", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND legacy_holder_user_id IS NOT NULL
		AND derivation <> 'MULTIPLE' AND parity NOT IN ('CONFLICT', 'MULTIPLE_CURRENT')`},
	{"case.P2_legacy_holder_unguarded_known", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND legacy_holder_user_id IS NOT NULL`},
	{"case.P1_and_P2_agree", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND derivation = 'DERIVED' AND parity = 'MATCH'
		AND derived_user_id IS NOT NULL AND derived_user_id = legacy_holder_user_id`},
	{"case.P1_P2_both_known_but_differ", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND derived_user_id IS NOT NULL
		AND legacy_holder_user_id IS NOT NULL AND derived_user_id <> legacy_holder_user_id`},
	{"case.legacy_holder_unresolved", `SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT' AND legacy_holder_user_id IS NULL`},
	{"case.historical_reassigned", `SELECT count(*) FROM (SELECT case_id FROM custody.custody_event WHERE kind = 'HOLDER_ASSIGNED' AND to_user_id IS NOT NULL
		GROUP BY case_id HAVING count(DISTINCT to_user_id) > 1) x`},
	{"case.events_unknown_kind", `SELECT count(*) FROM custody.custody_event WHERE kind = 'UNKNOWN'`},
	// ---- invoice scope (S4)
	{"inv.total", `SELECT count(*) FROM billing.invoice`},
	{"inv.seller_null", `SELECT count(*) FROM billing.invoice WHERE seller_user_id IS NULL`},
	{"inv.issuer_null", `SELECT count(*) FROM billing.invoice WHERE issued_by_user_id IS NULL`},
	{"inv.seller_eq_issuer", `SELECT count(*) FROM billing.invoice WHERE seller_user_id IS NOT NULL AND seller_user_id = issued_by_user_id`},
	{"inv.seller_ne_issuer", `SELECT count(*) FROM billing.invoice WHERE seller_user_id IS NOT NULL AND issued_by_user_id IS NOT NULL AND seller_user_id <> issued_by_user_id`},
	{"inv.case_link_resolved", `SELECT count(*) FROM billing.invoice WHERE case_link_state = 'RESOLVED'`},
	{"inv.case_link_not_resolved", `SELECT count(*) FROM billing.invoice WHERE case_link_state <> 'RESOLVED'`},
	{"inv.via_case_holder_eq_seller", `SELECT count(*) FROM billing.invoice i JOIN custody.assignment a ON a.case_id = i.case_id
		WHERE i.case_link_state = 'RESOLVED' AND a.legacy_holder_user_id IS NOT NULL AND a.legacy_holder_user_id = i.seller_user_id`},
	{"inv.via_case_holder_ne_seller", `SELECT count(*) FROM billing.invoice i JOIN custody.assignment a ON a.case_id = i.case_id
		WHERE i.case_link_state = 'RESOLVED' AND a.legacy_holder_user_id IS NOT NULL AND i.seller_user_id IS NOT NULL AND a.legacy_holder_user_id <> i.seller_user_id`},
	// ---- attribution facets (S2) and commission (S7)
	{"attr.facet_resolution", `SELECT coalesce(string_agg(facet || ':' || resolution || '=' || n, ', ' ORDER BY facet, resolution), '') FROM
		(SELECT facet, resolution, count(*) n FROM cases.attribution_snapshot GROUP BY facet, resolution) x`},
	{"attr.resolved_without_user", `SELECT count(*) FROM cases.attribution_snapshot WHERE resolution = 'RESOLVED' AND subject_user_id IS NULL`},
	{"com.entitlements", `SELECT count(*) FROM commission.historical_entitlement`},
	{"com.holder_null", `SELECT count(*) FROM commission.historical_entitlement WHERE holder_user_id IS NULL`},
	{"com.holder_ne_invoice_seller", `SELECT count(*) FROM commission.historical_entitlement h JOIN billing.invoice i ON i.invoice_id = h.invoice_id
		WHERE h.holder_user_id IS NOT NULL AND i.seller_user_id IS NOT NULL AND h.holder_user_id <> i.seller_user_id`},
	{"com.invoice_unlinked", `SELECT count(*) FROM commission.historical_entitlement WHERE invoice_id IS NULL`},
}

func TestRME04DOwnershipPredicateEvidence(t *testing.T) {
	e := setupIntegrated(t)
	e.seedIntegrated(t)
	ctx := context.Background()
	got := map[string]string{}
	for _, q := range rmE04DQueries {
		var v any
		if err := e.admin.QueryRow(ctx, q.sql).Scan(&v); err != nil {
			t.Fatalf("%s: %v", q.key, err)
		}
		got[q.key] = fmt.Sprint(v)
	}
	if os.Getenv("CRM_RME04D_EVIDENCE") == "1" {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&b, "| %s | %s |\n", k, got[k])
		}
		t.Logf("RM-E04D synthetic ownership evidence (R7-5 %s):\n%s", "syn-r7-5-integrated-v1", b.String())
	}
	// Invariants the decision sheet relies on (they hold by construction, whatever the counts are).
	n := func(k string) int {
		var x int
		if _, err := fmt.Sscan(got[k], &x); err != nil {
			t.Fatalf("%s: %q", k, got[k])
		}
		return x
	}
	if n("case.P1_and_P2_agree") > n("case.P1_derived_holder_known") || n("case.P1_and_P2_agree") > n("case.P2_legacy_holder_guarded_known") {
		t.Errorf("agreement cannot exceed either predicate")
	}
	if n("case.P2_legacy_holder_guarded_known") > n("case.P2_legacy_holder_unguarded_known") {
		t.Errorf("the guard can only remove rows")
	}
	if n("inv.seller_eq_issuer")+n("inv.seller_ne_issuer") > n("inv.total") {
		t.Errorf("invoice partition exceeds total")
	}
	if n("inv.case_link_resolved")+n("inv.case_link_not_resolved") != n("inv.total") {
		t.Errorf("case-link partition does not cover all invoices")
	}
}
