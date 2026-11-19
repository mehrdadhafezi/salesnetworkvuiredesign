package shadow_test

// RM-E04D — ownership-predicate evidence on the R7-5 SYNTHETIC integrated dataset (read only).
//
// Purpose: measure how each candidate Seller row-scope predicate (M2 proposed record 2) behaves on the
// synthetic legacy snapshot, so that the decision sheet can show the categories each predicate admits,
// excludes, or leaves UNKNOWN. It is NOT real-data evidence: rates on synthetic fixtures say nothing about
// production volumes (ADR #18 governs real data). It changes no row, grant, flag or status.
//
// Opt-in output: CRM_RME04D_EVIDENCE=1 prints the measured table (go test -v). Every value is PINNED to the
// number quoted in docs/implementation/rm-e04d-codex-security-review-package.md §2: the R7-5 dataset is
// deterministic, so an empty or changed seed fails here instead of passing vacuously on zero rows, and the
// documents cannot drift from the fixture without a reviewed change to both.

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
	// Exact population quoted in the review package §2 (RM-E04D). Non-zero categories are the ones the decision
	// sheet relies on; a missing seed makes every one of them fail.
	want := map[string]string{
		"case.total_assignments": "6", "case.present": "6",
		"case.P1_derived_holder_known": "3", "case.P2_legacy_holder_guarded_known": "4", "case.P2_legacy_holder_unguarded_known": "5",
		"case.P1_and_P2_agree": "2", "case.P1_P2_both_known_but_differ": "1", "case.legacy_holder_unresolved": "1",
		"case.historical_reassigned": "1", "case.events_unknown_kind": "1",
		"inv.total": "4", "inv.seller_null": "0", "inv.issuer_null": "0", "inv.seller_eq_issuer": "0", "inv.seller_ne_issuer": "4",
		"inv.case_link_resolved": "1", "inv.case_link_not_resolved": "3", "inv.via_case_holder_eq_seller": "1", "inv.via_case_holder_ne_seller": "0",
		"attr.facet_resolution":      "CREDIT_OWNER:UNKNOWN=4, ORIGINAL_OWNER:NOT_RECOVERABLE=2, ORIGINAL_OWNER:RESOLVED=4, ORIGINAL_SELLER:NOT_RECOVERABLE=2, ORIGINAL_SELLER:RESOLVED=3, ORIGINAL_SELLER:UNKNOWN=1",
		"attr.resolved_without_user": "0",
		"com.entitlements":           "19", "com.holder_null": "0", "com.holder_ne_invoice_seller": "4", "com.invoice_unlinked": "0",
	}
	if len(want) != len(rmE04DQueries) {
		t.Fatalf("%d queries but %d pinned values; pin every measure", len(rmE04DQueries), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %s, want %s (R7-5 population changed or missing; update the review package §2 with it)", k, got[k], v)
		}
	}
}
