package legacytest

import (
	"fmt"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
)

// R7-5 integrated SYNTHETIC dataset ("one coherent synthetic world").
//
// IntegratedScenario composes the EXISTING read-world fixtures, the ones the
// module suites already layer on the HR organisation, into one snapshot
// package:
//
//	HRScenario            identity + organisation (users 1..13, hierarchy findings)
//	SalesScenario         leads, distribution items/logs → cases + custody
//	CustomerScenario      issued-to subjects 1001..1005 (phone edge cases)
//	  + issued-to 2001    the billing customer used by the module suites
//	BillingScenario       invoices 2001..2004, items, stages, payments
//	BiawinScenario        memberships/items on 2001..2003 (+ 2999 UNKNOWN)
//	WalletScenario        legacy wallets and wallet transactions
//	CommissionScenario    commission rules, dry runs, posting logs, tx
//
// Nothing is invented: every record comes from those fixtures unchanged;
// entities that appear in more than one fixture (wallet transactions) are
// merged and a duplicate key fails construction. References that the
// fixtures deliberately leave unresolved (for example invoice 2999 and the
// phone-derived account 71) stay unresolved and are documented in
// IntegratedKnownUnresolved.
//
// The controlled-write world (WritesScenario / PaymentsScenario: Slice 11 HR
// requests, Slice 12 payment submissions, R7-1..R7-3 browser suites) reuses
// legacy user ids 1..10 with other meanings and cannot be merged into this
// package without re-keying records, which would invent relationships. It
// stays a separate fixture (docs/remediation/r7-5-synthetic-dataset.md).
//
// Deterministic: the same IntegratedVersion always yields the same rows, the
// same snapshot id and the same extraction time, so the written package is
// byte-identical. Runtime metadata (import run ids, pgtest role names) is
// produced by the import pipeline, not by this source.

// Dataset identity. Bump IntegratedVersion with any change to the source.
const (
	IntegratedVersion     = "r7-5.v1"
	IntegratedSnapshotID  = "syn-r7-5-integrated-v1"
	IntegratedExtractedAt = "2026-10-10T04:30:00Z"
)

// IntegratedKnownUnresolved lists references the source fixtures leave
// unresolved on purpose (spec.column → value). Integrity tests treat every
// other dangling reference as a defect.
var IntegratedKnownUnresolved = map[string][]string{
	"invoice": {
		"2999", // BillingScenario payment 9 and BiawinScenario membership 504: invoice not staged → UNKNOWN
		"0",    // CommissionScenario dry-run item 2: legacy "no invoice" sentinel of a daily attendance bonus (NOT_COMPARABLE)
	},
}

// IntegratedScenario returns the composed source (all entities COMPLETE).
func IntegratedScenario() []snapshot.EntityRows {
	c := snapshot.Complete
	billingCustomer := rows("legacy.invoice_issued_to.v1", []Rec{
		IssuedTo("2001", "", "Synthetic Customer D", "09000000901", "", "2025-03-01 10:00:00")})
	billingCustomer.Completeness = c
	cust := append(CustomerScenario(c), billingCustomer)
	parts := [][]snapshot.EntityRows{
		HRScenario(), SalesScenario(c), cust, BillingScenario(c), BiawinScenario(c), WalletScenario(c), CommissionScenario(c),
	}
	var all []snapshot.EntityRows
	for _, p := range parts {
		all = append(all, p...)
	}
	return mergeEntities(all)
}

// mergeEntities combines entities that share a spec id. Rows keep their
// fixture order; a duplicate primary key or a completeness mismatch panics
// (a construction error, never data to import).
func mergeEntities(in []snapshot.EntityRows) []snapshot.EntityRows {
	reg := spec.Default()
	idx := map[string]int{}
	var out []snapshot.EntityRows
	keys := map[string]map[string]bool{}
	for _, e := range in {
		s, err := reg.Lookup(e.SpecID)
		if err != nil {
			panic(fmt.Sprintf("integrated: %v", err))
		}
		i, ok := idx[e.SpecID]
		if !ok {
			idx[e.SpecID] = len(out)
			out = append(out, snapshot.EntityRows{SpecID: e.SpecID, Completeness: e.Completeness})
			keys[e.SpecID] = map[string]bool{}
			i = idx[e.SpecID]
		} else if out[i].Completeness != e.Completeness {
			panic(fmt.Sprintf("integrated: %s completeness %v vs %v", e.SpecID, out[i].Completeness, e.Completeness))
		}
		for _, r := range e.Rows {
			var parts []string
			for _, k := range s.KeyIndexes() {
				parts = append(parts, r[k].Text)
			}
			k := strings.Join(parts, "|")
			if keys[e.SpecID][k] {
				panic(fmt.Sprintf("integrated: duplicate key %s in %s", k, e.SpecID))
			}
			keys[e.SpecID][k] = true
			out[i].Rows = append(out[i].Rows, r)
		}
	}
	return out
}
