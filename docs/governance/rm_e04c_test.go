package governance

// RM-E04C (Seller authorization architecture, decision package, screen designs, API proposal, traceability,
// roadmap) consistency checks.
//
// The RM-E04C documents are proposals. The checks below keep them out of runtime and keep the decisions they quote
// honest, WITHOUT freezing the governance process:
//
//   - a quoted decision may keep its quoted status (then no record, no outcome, verification NOT_STARTED), or move
//     to another status only through an approval record that passes the R6 rules (validateApprovalRecord in
//     repository mode, which rejects synthetic markers, and statusMatchesRecord) and, for a DECISION record, carries
//     an independent approval for every R5 approver role established in this repository for that decision;
//   - Group B runtime artefacts (Seller capabilities, a populated seller bundle, /me operations) are refused while
//     the Group B gate (rm-e04c-group-b-gate.json) is closed; the gate opens only when both M2 record ids it names
//     are APPROVED_VERIFIED under the same rules, and then only the capabilities and operations it lists are allowed;
//   - every RM-E04C document carries a not-approved marker and Production NO-GO;
//   - every NT-xx referenced by the traceability matrix is defined, and every SR-01..SR-16 is covered.
//
// The transition and gate rules are pure functions, exercised with synthetic fixtures built in test code
// (never written to docs/governance/approvals/).

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
)

var rmE04CDocs = []string{
	"docs/implementation/rm-e04c-seller-authorization-adr-PROPOSED.md",
	"docs/implementation/rm-e04c-seller-api-contract-PROPOSAL.md",
	"docs/implementation/rm-e04c-seller-implementation-roadmap.md",
	"docs/security/rm-e04c-seller-screen-designs.md",
	"docs/security/rm-e04c-seller-traceability-matrix.md",
	"docs/governance/proposals/rm-e04c-seller-decision-package.fa.md",
}

const rmE04CADR = "docs/implementation/rm-e04c-seller-authorization-adr-PROPOSED.md"

// Tracker status quoted by the RM-E04C decision package at baseline c0d7f4b.
var rmE04CQuotedStatus = map[string]string{
	"D-42":                     "READY_FOR_REVIEW",
	"D-34":                     "READY_FOR_REVIEW",
	"R5-SEC-01 (proposed ADR)": "READY_FOR_REVIEW",
	"ADR #18":                  "READY_FOR_REVIEW",
	"D-41":                     "EVIDENCE_MISSING",
	"OPD-09":                   "EVIDENCE_MISSING",
	"D-36":                     "EVIDENCE_MISSING",
	"OPD-10":                   "READY_FOR_REVIEW",
	"D-30":                     "EVIDENCE_MISSING",
	"D-45":                     "READY_FOR_REVIEW",
}

// R5 approver roles already established as role strings in this repository (m2_test.go, r6b_test.go). For the
// other quoted decisions the R6 record rules apply unchanged; this file does not invent role strings for them.
func rmE04CRequiredRoles(id string) []string {
	if r, ok := m2Approvers[id]; ok {
		return r
	}
	if id == "ADR #18" {
		return []string{"Product owner", "Security owner", "Legal owner"}
	}
	return nil
}

// Both M2 records are decided with Security as an approver (M2 §2-1, §3); their full role lists are set when the
// governance owner registers them.
var rmE04CGateFloorRoles = []string{"Security owner"}

// IDs that can never stand in for M2 records 1 / 2: the decisions RM-E04C keeps separate from them, and the
// authoritative E0 approvals.
var rmE04CGateReserved = map[string]bool{"D-01": true, "D-02": true, "MONEY (E0)": true}

var rmE04CCandidateCaps = []identityapi.Capability{"cases.own.read", "billing.own.read", "payments.own.read"}

var rmE04CMeOperations = []string{"listMyCases", "getMyCase", "listMyCaseHistory", "getMyCaseContactOutcomes",
	"listMyCaseAttribution", "listMyInvoices", "getMyInvoice", "listMyInvoiceStages", "getMyInvoiceTotals", "listMyInvoicePayments"}

type approvalLoader func(rel string) (approvalRecord, error)

func repoLoader(rel string) (approvalRecord, error) { return loadApproval(filepath.Join(root, rel)) }

// rmE04CDecisionProblems: a decision either keeps the status quoted by the package (and then nothing is recorded),
// or holds a status justified by a valid approval record. inRepo selects the R6 repository mode (synthetic markers
// rejected); only the synthetic-fixture tests pass false.
func rmE04CDecisionProblems(r trackerRecord, quoted string, roles []string, load approvalLoader, inRepo bool) []string {
	var p []string
	if quoted != "" && r.Status == quoted {
		if r.ApprovalRecord != "" || r.Verification != "NOT_STARTED" {
			p = append(p, fmt.Sprintf("%s: record %q / verification %s without a status change", r.ID, r.ApprovalRecord, r.Verification))
		}
		for k, v := range r.Outcome {
			if strings.TrimSpace(v) != "" {
				p = append(p, fmt.Sprintf("%s: outcome %s filled without an approval record", r.ID, k))
			}
		}
		return p
	}
	if !needsRecord[r.Status] {
		return append(p, fmt.Sprintf("%s: status %s (quoted %s) is not a state an approval record can justify", r.ID, r.Status, quoted))
	}
	if r.ApprovalRecord == "" {
		return append(p, fmt.Sprintf("%s: status %s without an approval record", r.ID, r.Status))
	}
	rec, err := load(r.ApprovalRecord)
	if err != nil {
		return append(p, fmt.Sprintf("%s: %v", r.ID, err))
	}
	level, vp := validateApprovalRecord(rec, inRepo)
	for _, x := range vp {
		p = append(p, fmt.Sprintf("%s: approval record invalid: %s", r.ID, x))
	}
	if rec.DecisionID != r.ID {
		p = append(p, fmt.Sprintf("%s: approval record is for %s", r.ID, rec.DecisionID))
	}
	if err := statusMatchesRecord(r.Status, rec, level); err != nil {
		p = append(p, fmt.Sprintf("%s: %v", r.ID, err))
	}
	if rec.RecordType == "DECISION" {
		have := map[string]bool{}
		for _, x := range rec.IndependentApprovals {
			have[x] = true
		}
		for _, role := range roles {
			if !have[role] {
				p = append(p, fmt.Sprintf("%s: no independent approval by %s", r.ID, role))
			}
		}
	}
	return p
}

type rmE04CGate struct {
	Purpose                    string   `json:"purpose"`
	M2Record1ID                string   `json:"m2_record_1_id"`
	M2Record2ID                string   `json:"m2_record_2_id"`
	ApprovedSellerCapabilities []string `json:"approved_seller_capabilities"`
	ApprovedMeOperations       []string `json:"approved_me_operations"`
}

// rmE04CGateState: the gate is open only when both M2 ids are named, distinct, not reserved, present in the
// tracker and APPROVED_VERIFIED through valid records. A closed gate must not list anything.
func rmE04CGateState(g rmE04CGate, recs map[string]trackerRecord, load approvalLoader, inRepo bool) (bool, []string) {
	var p []string
	listed := len(g.ApprovedSellerCapabilities) > 0 || len(g.ApprovedMeOperations) > 0
	if g.M2Record1ID == "" || g.M2Record2ID == "" {
		if listed {
			p = append(p, "gate closed (an M2 record id is empty) but approved capabilities / operations are listed")
		}
		return false, p
	}
	if g.M2Record1ID == g.M2Record2ID {
		p = append(p, "M2 record 1 and record 2 must be distinct decisions")
	}
	for _, id := range []string{g.M2Record1ID, g.M2Record2ID} {
		if rmE04CGateReserved[id] {
			p = append(p, id+" is reserved and cannot stand for an M2 record")
		}
		if _, quoted := rmE04CQuotedStatus[id]; quoted {
			p = append(p, id+" is a separate decision (RM-E04A §1) and cannot stand for an M2 record")
		}
		r, ok := recs[id]
		if !ok {
			p = append(p, id+" is not in the R6 tracker")
			continue
		}
		if r.Status != "APPROVED_VERIFIED" {
			p = append(p, fmt.Sprintf("%s is %s, not APPROVED_VERIFIED", id, r.Status))
			continue
		}
		p = append(p, rmE04CDecisionProblems(r, "", rmE04CGateFloorRoles, load, inRepo)...)
	}
	if len(p) > 0 {
		return false, p
	}
	return true, nil
}

// rmE04CRuntimeProblems: with a closed gate nothing of Group B may exist at runtime; with an open gate only what
// the gate lists may exist.
func rmE04CRuntimeProblems(open bool, g rmE04CGate, known func(identityapi.Capability) bool, seller []identityapi.Capability, spec string) []string {
	var p []string
	caps, ops := map[string]bool{}, map[string]bool{}
	if open {
		for _, c := range g.ApprovedSellerCapabilities {
			caps[c] = true
		}
		for _, o := range g.ApprovedMeOperations {
			ops[o] = true
		}
	}
	for _, c := range rmE04CCandidateCaps {
		if known(c) && !caps[string(c)] {
			p = append(p, fmt.Sprintf("capability %s is registered without an open Group B gate listing it", c))
		}
	}
	for _, c := range seller {
		if !caps[string(c)] {
			p = append(p, fmt.Sprintf("seller bundle holds %s without an open Group B gate listing it", c))
		}
	}
	for _, o := range rmE04CMeOperations {
		if regexp.MustCompile(`\boperationId: ` + o + `\b`).MatchString(spec) && !ops[o] {
			p = append(p, fmt.Sprintf("operation %s is in the contract without an open Group B gate listing it", o))
		}
	}
	if strings.Contains(spec, "/api/v1/me/") && len(ops) == 0 {
		p = append(p, "a /api/v1/me/ path is in the contract without an open Group B gate")
	}
	return p
}

func loadRMEGate(t *testing.T) rmE04CGate {
	t.Helper()
	var g rmE04CGate
	dec := json.NewDecoder(strings.NewReader(read(t, "docs/governance/rm-e04c-group-b-gate.json")))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&g); err != nil {
		t.Fatal(err)
	}
	return g
}

func trackerByID(t *testing.T) map[string]trackerRecord {
	out := map[string]trackerRecord{}
	for _, r := range loadTracker(t).Records {
		out[r.ID] = r
	}
	return out
}

// ---------------- repository checks

func TestRME04CTrackerTransitionsAreVerified(t *testing.T) {
	recs := trackerByID(t)
	for id, quoted := range rmE04CQuotedStatus {
		r, ok := recs[id]
		if !ok {
			t.Errorf("%s missing from the tracker", id)
			continue
		}
		for _, x := range rmE04CDecisionProblems(r, quoted, rmE04CRequiredRoles(id), repoLoader, true) {
			t.Error(x)
		}
	}
}

func TestRME04CProposalsStayOutOfRuntime(t *testing.T) {
	g := loadRMEGate(t)
	open, p := rmE04CGateState(g, trackerByID(t), repoLoader, true)
	for _, x := range p {
		t.Error(x)
	}
	seller, ok := identityapi.LookupRole(identityapi.RoleSeller)
	if !ok {
		t.Fatal("seller role missing from the vocabulary")
	}
	for _, x := range rmE04CRuntimeProblems(open, g, identityapi.KnownCapability, seller.Capabilities, read(t, "api/openapi/openapi.yaml")) {
		t.Error(x)
	}
}

func TestRME04CDocumentsAreMarkedNotApproved(t *testing.T) {
	open, _ := rmE04CGateState(loadRMEGate(t), trackerByID(t), repoLoader, true)
	marker := regexp.MustCompile(`(DRAFT — NOT APPROVED|PROPOSED — NOT ACCEPTED|PROPOSAL — NOT IMPLEMENTED, NOT APPROVED|PLAN ONLY)`)
	accepted := regexp.MustCompile(`\| Status \| \*\*ACCEPTED\b`)
	for _, f := range rmE04CDocs {
		d := read(t, f)
		head := d
		if len(head) > 1500 {
			head = head[:1500]
		}
		ok := marker.MatchString(head)
		if f == rmE04CADR && accepted.MatchString(head) {
			// Acceptance (roadmap B1) is possible only once the Group B gate is open.
			ok = open
		}
		if !ok {
			t.Errorf("%s: no not-approved marker near the top (or ACCEPTED while the Group B gate is closed)", f)
		}
		if !strings.Contains(d, "NO-GO") {
			t.Errorf("%s: Production NO-GO not stated", f)
		}
	}
}

func TestRME04CTraceabilityIsClosed(t *testing.T) {
	matrix := read(t, "docs/security/rm-e04c-seller-traceability-matrix.md")
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^### (NT-\d{2}) `).FindAllStringSubmatch(read(t, "docs/security/rm-e04a-seller-negative-test-plan.md"), -1) {
		defined[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`(?m)^\| \*\*(NT-\d{2})\*\* \|`).FindAllStringSubmatch(matrix, -1) {
		if defined[m[1]] {
			t.Errorf("%s defined twice", m[1])
		}
		defined[m[1]] = true
	}
	for _, f := range rmE04CDocs {
		for _, id := range regexp.MustCompile(`NT-\d{2}`).FindAllString(read(t, f), -1) {
			if !defined[id] {
				t.Errorf("%s references undefined %s", f, id)
			}
		}
	}
	cov := matrix[strings.Index(matrix, "## 4. Requirement coverage"):]
	for n := 1; n <= 16; n++ {
		sr := fmt.Sprintf("SR-%02d", n)
		if !regexp.MustCompile(`(?m)^\| ` + sr + ` .*\| NT-\d{2}`).MatchString(cov) {
			t.Errorf("%s has no covering test in §4", sr)
		}
	}
}

// ---------------- synthetic fixtures (built in test code, never written to approvals/)

const rmFxRecordPath = "docs/governance/approvals/rm-e04c-fixture.json"

// rmFxDecision is a complete DECISION record for id, with one approver per role and the given verification result.
func rmFxDecision(t *testing.T, id, outcome, verification string, roles []string) approvalRecord {
	t.Helper()
	rec := fxDecision(t, outcome, verification)
	rec.DecisionID = id
	rec.Approvers, rec.IndependentApprovals = nil, nil
	for i, role := range roles {
		var a authorizedPerson
		a.Name, a.Role = fmt.Sprintf("Fixture Approver %d", i+1), role
		a.AuthorityEvidence.Type, a.AuthorityEvidence.Reference = "delegation", fmt.Sprintf("fixture-auth-%d", i+1)
		rec.Approvers = append(rec.Approvers, a)
		rec.IndependentApprovals = append(rec.IndependentApprovals, role)
	}
	return rec
}

func loaderOf(rec approvalRecord) approvalLoader {
	return func(rel string) (approvalRecord, error) {
		if rel != rmFxRecordPath {
			return approvalRecord{}, fmt.Errorf("no record at %s", rel)
		}
		return rec, nil
	}
}

func rmFxTracker(id, status, record, verification string) trackerRecord {
	return trackerRecord{ID: id, Status: status, ApprovalRecord: record, Verification: verification, Outcome: map[string]string{}}
}

func TestRME04CTransitionRules(t *testing.T) {
	roles := rmE04CRequiredRoles("D-42")
	valid := rmFxDecision(t, "D-42", "APPROVED", "VERIFIED", roles)
	cases := []struct {
		name   string
		r      trackerRecord
		rec    approvalRecord
		inRepo bool
		ok     bool
	}{
		{"unchanged quoted status, nothing recorded", rmFxTracker("D-42", "READY_FOR_REVIEW", "", "NOT_STARTED"), valid, false, true},
		{"legitimate verified approval advances", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"), valid, false, true},
		{"recorded decision pending verification", rmFxTracker("D-42", "DECISION_RECORDED_PENDING_VERIFICATION", rmFxRecordPath, "NOT_STARTED"),
			rmFxDecision(t, "D-42", "APPROVED", "", roles), false, true},
		{"approval without a record", rmFxTracker("D-42", "APPROVED_VERIFIED", "", "NOT_STARTED"), valid, false, false},
		{"record path does not exist", rmFxTracker("D-42", "APPROVED_VERIFIED", "docs/governance/approvals/missing.json", "VERIFIED"), valid, false, false},
		{"verification inconclusive", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"),
			rmFxDecision(t, "D-42", "APPROVED", "INCONCLUSIVE", roles), false, false},
		{"no human verification", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"),
			rmFxDecision(t, "D-42", "APPROVED", "", roles), false, false},
		{"recorded outcome differs", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"),
			rmFxDecision(t, "D-42", "REJECTED", "VERIFIED", roles), false, false},
		{"record for another decision", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"),
			rmFxDecision(t, "D-34", "APPROVED", "VERIFIED", roles), false, false},
		{"required R5 role missing", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"),
			rmFxDecision(t, "D-42", "APPROVED", "VERIFIED", roles[:3]), false, false},
		{"synthetic record cannot be committed", rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED"), valid, true, false},
		{"outcome filled without a status change", trackerRecord{ID: "D-42", Status: "READY_FOR_REVIEW", Verification: "NOT_STARTED",
			Outcome: map[string]string{"selected_option": "2"}}, valid, false, false},
		{"record attached without a status change", rmFxTracker("D-42", "READY_FOR_REVIEW", rmFxRecordPath, "VERIFIED"), valid, false, false},
		{"status moved to a state no record justifies", rmFxTracker("D-42", "DRAFT", "", "NOT_STARTED"), valid, false, false},
	}
	for _, c := range cases {
		p := rmE04CDecisionProblems(c.r, "READY_FOR_REVIEW", roles, loaderOf(c.rec), c.inRepo)
		if (len(p) == 0) != c.ok {
			t.Errorf("%s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
}

func TestRME04CGroupBGate(t *testing.T) {
	const r1, r2 = "M2-REC-1-FIXTURE", "M2-REC-2-FIXTURE"
	floor := rmE04CGateFloorRoles
	approved := func(id string) trackerRecord { return rmFxTracker(id, "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED") }
	recsOf := func(a, b trackerRecord) map[string]trackerRecord { return map[string]trackerRecord{a.ID: a, b.ID: b} }
	// One loader serving a record whose id is chosen per call.
	loadFor := func(rec map[string]approvalRecord) approvalLoader {
		return func(rel string) (approvalRecord, error) {
			id := strings.TrimSuffix(strings.TrimPrefix(rel, "approvals/"), ".json")
			if r, ok := rec[id]; ok {
				return r, nil
			}
			return approvalRecord{}, fmt.Errorf("no record at %s", rel)
		}
	}
	at := func(id string) trackerRecord {
		r := approved(id)
		r.ApprovalRecord = "approvals/" + id + ".json"
		return r
	}
	good := loadFor(map[string]approvalRecord{
		r1: rmFxDecision(t, r1, "APPROVED", "VERIFIED", floor),
		r2: rmFxDecision(t, r2, "APPROVED", "VERIFIED", floor),
	})
	open := rmE04CGate{M2Record1ID: r1, M2Record2ID: r2,
		ApprovedSellerCapabilities: []string{"cases.own.read", "commission.own.read"}, ApprovedMeOperations: []string{"listMyCases"}}
	spec := "  /api/v1/me/cases:\n    get:\n      operationId: listMyCases\n"
	knownOnly := func(set ...identityapi.Capability) func(identityapi.Capability) bool {
		return func(c identityapi.Capability) bool {
			for _, s := range set {
				if s == c {
					return true
				}
			}
			return false
		}
	}

	gateCases := []struct {
		name string
		g    rmE04CGate
		recs map[string]trackerRecord
		load approvalLoader
		open bool
		ok   bool
	}{
		{"closed gate today (no ids, nothing listed)", rmE04CGate{}, nil, good, false, true},
		{"closed gate listing capabilities", rmE04CGate{ApprovedSellerCapabilities: []string{"cases.own.read"}}, nil, good, false, false},
		{"one id only stays closed", rmE04CGate{M2Record1ID: r1}, recsOf(at(r1), at(r2)), good, false, true},
		{"both verified approvals open the gate", open, recsOf(at(r1), at(r2)), good, true, true},
		{"record 2 still under review", open, recsOf(at(r1), rmFxTracker(r2, "READY_FOR_REVIEW", "", "NOT_STARTED")), good, false, false},
		{"record 2 claims approval without a valid record", open, recsOf(at(r1), at(r2)),
			loadFor(map[string]approvalRecord{r1: rmFxDecision(t, r1, "APPROVED", "VERIFIED", floor), r2: rmFxDecision(t, r2, "APPROVED", "", floor)}), false, false},
		{"approval lacks Security", open, recsOf(at(r1), at(r2)),
			loadFor(map[string]approvalRecord{r1: rmFxDecision(t, r1, "APPROVED", "VERIFIED", []string{"Product owner"}), r2: rmFxDecision(t, r2, "APPROVED", "VERIFIED", floor)}), false, false},
		{"id not in tracker", open, map[string]trackerRecord{r1: at(r1)}, good, false, false},
		{"same id twice", rmE04CGate{M2Record1ID: r1, M2Record2ID: r1}, recsOf(at(r1), at(r2)), good, false, false},
		{"separate decision used as M2 record", rmE04CGate{M2Record1ID: "D-42", M2Record2ID: r2}, recsOf(at("D-42"), at(r2)),
			loadFor(map[string]approvalRecord{"D-42": rmFxDecision(t, "D-42", "APPROVED", "VERIFIED", floor), r2: rmFxDecision(t, r2, "APPROVED", "VERIFIED", floor)}), false, false},
		{"E0 approval used as M2 record", rmE04CGate{M2Record1ID: "D-01", M2Record2ID: r2}, recsOf(at("D-01"), at(r2)),
			loadFor(map[string]approvalRecord{"D-01": rmFxDecision(t, "D-01", "APPROVED", "VERIFIED", floor), r2: rmFxDecision(t, r2, "APPROVED", "VERIFIED", floor)}), false, false},
	}
	for _, c := range gateCases {
		gotOpen, p := rmE04CGateState(c.g, c.recs, c.load, false)
		if gotOpen != c.open || (len(p) == 0) != c.ok {
			t.Errorf("gate %s: open=%v ok=%v, want open=%v ok=%v (%v)", c.name, gotOpen, len(p) == 0, c.open, c.ok, p)
		}
	}
	// In repository mode the synthetic approvals can never open the gate.
	if o, _ := rmE04CGateState(open, recsOf(at(r1), at(r2)), good, true); o {
		t.Error("synthetic approval records opened the gate in repository mode")
	}

	runtimeCases := []struct {
		name   string
		open   bool
		known  func(identityapi.Capability) bool
		seller []identityapi.Capability
		spec   string
		ok     bool
	}{
		{"closed: nothing of Group B exists", false, knownOnly(), nil, "", true},
		{"closed: candidate capability registered", false, knownOnly("cases.own.read"), nil, "", false},
		{"closed: seller bundle populated", false, knownOnly(), []identityapi.Capability{"commission.own.read"}, "", false},
		{"closed: /me operation in contract", false, knownOnly(), nil, spec, false},
		{"open: approved implementation allowed", true, knownOnly("cases.own.read"), []identityapi.Capability{"cases.own.read", "commission.own.read"}, spec, true},
		{"open: unapproved capability registered", true, knownOnly("cases.own.read", "billing.own.read"), nil, spec, false},
		{"open: unapproved capability in bundle", true, knownOnly("cases.own.read"), []identityapi.Capability{"cases.own.read", "custody.custody.read"}, spec, false},
		{"open: unapproved operation in contract", true, knownOnly(), nil, spec + "      operationId: listMyInvoices\n", false},
	}
	for _, c := range runtimeCases {
		p := rmE04CRuntimeProblems(c.open, open, c.known, c.seller, c.spec)
		if (len(p) == 0) != c.ok {
			t.Errorf("runtime %s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
}
