package governance

// RM-E04D — governance-test readiness for legitimate future approvals.
//
// Before RM-E04D three baseline tests pinned the tracker so that ANY transition failed, a verified one
// included: TestM2TrackerUnchanged (D-42, D-34, R5-SEC-01), TestR6BTrackerUnchanged (ADR #18, P-GOV and
// the global status counts, which also block registering a new record) and TestR6ZeroRecordedApprovals
// (any file under approvals/). They now apply the R6 rules instead of a freeze:
//
//   - a decision keeps its baseline status with nothing recorded, or holds a status justified by an approval
//     record that passes validateApprovalRecord in REPOSITORY mode (synthetic / placeholder markers rejected)
//     and statusMatchesRecord, with an independent approval for every R5 approver role established in the
//     repository for that decision (rmE04CDecisionProblems, shared with RM-E04C);
//   - every file under approvals/ is a valid repository-mode record, named by recordFileName, for a decision
//     that exists in the tracker (append-only history keeps superseded notices, so "referenced by the
//     current tracker entry" is not required);
//   - the identity schema keeps role_grant_shadow_only unless D-42 is APPROVED_VERIFIED (no ACTIVE grant
//     path can appear while grant authority is undecided).
//
// What still holds unchanged: TestR6NoStatusOrOutcomeWithoutApprovalRecord (no status without a valid
// record, R5-derived baseline otherwise), TestApprovalRecordsAppendOnly, the RM-E04C Group B gate, and the
// 41-record R5 tripwire in TestR6TrackerAccountsForEveryR5Record, which the inventory owner's registering
// PR updates deliberately together with the new inventory rows.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// rmE04DApproverRoles: the R5 approver roles already written as role strings in this repository.
func rmE04DApproverRoles(id string) []string { return rmE04CRequiredRoles(id) }

// baselineOrVerifiedProblems checks the named decisions against their baseline status.
func baselineOrVerifiedProblems(recs map[string]trackerRecord, baseline map[string]string, load approvalLoader, inRepo bool) []string {
	var p []string
	for id, base := range baseline {
		r, ok := recs[id]
		if !ok {
			p = append(p, id+" missing from the tracker")
			continue
		}
		p = append(p, rmE04CDecisionProblems(r, base, rmE04DApproverRoles(id), load, inRepo)...)
	}
	return p
}

// approvalFilesProblems checks every committed approval file.
func approvalFilesProblems(names []string, load func(name string) (approvalRecord, error), tracked map[string]bool) []string {
	var p []string
	for _, n := range names {
		rec, err := load(n)
		if err != nil {
			p = append(p, fmt.Sprintf("%s: %v", n, err))
			continue
		}
		if want := recordFileName(rec); want != n {
			p = append(p, fmt.Sprintf("%s must be named %s", n, want))
		}
		if _, vp := validateApprovalRecord(rec, true); len(vp) > 0 {
			p = append(p, fmt.Sprintf("%s: %v", n, vp))
		}
		if !tracked[rec.DecisionID] {
			p = append(p, fmt.Sprintf("%s: decision %q is not in the tracker", n, rec.DecisionID))
		}
	}
	return p
}

var shadowOnlyCheck = regexp.MustCompile(`CONSTRAINT role_grant_shadow_only CHECK \(status = 'SHADOW'\)`)

// grantDefiningMigration is the only migration allowed to define identity.role_grant (RM-E04D).
const grantDefiningMigration = "00001_identity.sql"

var sqlLineComment = regexp.MustCompile(`--[^\n]*`)
var sqlBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)

// sqlCode strips comments and identifier quotes so `"role_grant_shadow_only"` and role_grant_shadow_only match alike.
func sqlCode(sql string) string {
	return strings.ToLower(strings.ReplaceAll(sqlLineComment.ReplaceAllString(sqlBlockComment.ReplaceAllString(sql, " "), " "), `"`, ""))
}

// grantConstraintProblems: the DB keeps refusing ACTIVE grants unless D-42 is APPROVED_VERIFIED. The defining
// migration must still carry the SHADOW-only CHECK, and no other migration may touch identity.role_grant or its
// constraint in code (drop, rename, re-create, alter, a new permissive CHECK, …) while D-42 is not verified.
// Comments are ignored, so a note such as "role_grant_shadow_only stays" is fine.
func grantConstraintProblems(migrations map[string]string, d42 trackerRecord) []string {
	var p []string
	def, ok := migrations[grantDefiningMigration]
	if !ok || !shadowOnlyCheck.MatchString(def) {
		p = append(p, grantDefiningMigration+" no longer defines role_grant_shadow_only")
	}
	if d42.Status == "APPROVED_VERIFIED" {
		return p
	}
	for name, sql := range migrations {
		if name == grantDefiningMigration {
			continue
		}
		if strings.Contains(sqlCode(sql), "role_grant") {
			p = append(p, fmt.Sprintf("%s touches identity.role_grant while D-42 is %s", name, d42.Status))
		}
	}
	return p
}

// ---------------- repository checks (used by the rewritten baseline tests)

func identityMigrations(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(root, "internal/identity/migrations/sql")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			out[e.Name()] = read(t, "internal/identity/migrations/sql/"+e.Name())
		}
	}
	return out
}

func TestRME04DNoActiveGrantPathWithoutD42(t *testing.T) {
	for _, x := range grantConstraintProblems(identityMigrations(t), trackerByID(t)["D-42"]) {
		t.Error(x)
	}
}

// ---------------- synthetic fixtures (built in code; never written to approvals/)

func TestRME04DBaselineRules(t *testing.T) {
	d42Roles := rmE04DApproverRoles("D-42")
	adrRoles := rmE04DApproverRoles("ADR #18")
	if len(d42Roles) != 4 || len(adrRoles) != 3 {
		t.Fatalf("role lists changed: D-42 %v, ADR #18 %v", d42Roles, adrRoles)
	}
	baseline := map[string]string{"D-42": "READY_FOR_REVIEW", "ADR #18": "READY_FOR_REVIEW", "D-03": "EVIDENCE_MISSING"}
	unchanged := func() map[string]trackerRecord {
		return map[string]trackerRecord{
			"D-42":    rmFxTracker("D-42", "READY_FOR_REVIEW", "", "NOT_STARTED"),
			"ADR #18": rmFxTracker("ADR #18", "READY_FOR_REVIEW", "", "NOT_STARTED"),
			"D-03":    rmFxTracker("D-03", "EVIDENCE_MISSING", "", "NOT_STARTED"),
		}
	}
	approve := func(id string, roles []string, verification string) (map[string]trackerRecord, approvalLoader) {
		recs := unchanged()
		recs[id] = rmFxTracker(id, "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED")
		return recs, loaderOf(rmFxDecision(t, id, "APPROVED", verification, roles))
	}
	cases := []struct {
		name   string
		build  func() (map[string]trackerRecord, approvalLoader)
		inRepo bool
		ok     bool
	}{
		{"current unapproved state stays closed and valid", func() (map[string]trackerRecord, approvalLoader) { return unchanged(), loaderOf(approvalRecord{}) }, true, true},
		{"legitimate verified D-42 approval passes", func() (map[string]trackerRecord, approvalLoader) { return approve("D-42", d42Roles, "VERIFIED") }, false, true},
		{"legitimate verified ADR #18 approval passes", func() (map[string]trackerRecord, approvalLoader) { return approve("ADR #18", adrRoles, "VERIFIED") }, false, true},
		{"forged: APPROVED_VERIFIED with no record", func() (map[string]trackerRecord, approvalLoader) {
			r := unchanged()
			r["D-42"] = rmFxTracker("D-42", "APPROVED_VERIFIED", "", "NOT_STARTED")
			return r, loaderOf(approvalRecord{})
		}, false, false},
		{"forged: record without human verification", func() (map[string]trackerRecord, approvalLoader) { return approve("D-42", d42Roles, "") }, false, false},
		{"forged: verification INCONCLUSIVE", func() (map[string]trackerRecord, approvalLoader) { return approve("D-42", d42Roles, "INCONCLUSIVE") }, false, false},
		{"missing approver role (Sales owner)", func() (map[string]trackerRecord, approvalLoader) {
			return approve("D-42", []string{"Product owner", "HR owner", "Security owner"}, "VERIFIED")
		}, false, false},
		{"missing approver role (Legal owner for ADR #18)", func() (map[string]trackerRecord, approvalLoader) {
			return approve("ADR #18", []string{"Product owner", "Security owner"}, "VERIFIED")
		}, false, false},
		{"synthetic record cannot be committed (repository mode)", func() (map[string]trackerRecord, approvalLoader) { return approve("D-42", d42Roles, "VERIFIED") }, true, false},
		{"P-GOV status moved without a record", func() (map[string]trackerRecord, approvalLoader) {
			r := unchanged()
			r["D-03"] = rmFxTracker("D-03", "READY_FOR_REVIEW", "", "NOT_STARTED")
			return r, loaderOf(approvalRecord{})
		}, false, false},
	}
	for _, c := range cases {
		recs, load := c.build()
		p := baselineOrVerifiedProblems(recs, baseline, load, c.inRepo)
		if (len(p) == 0) != c.ok {
			t.Errorf("%s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
}

func TestRME04DApprovalFileRules(t *testing.T) {
	good := rmFxDecision(t, "D-42", "APPROVED", "VERIFIED", rmE04DApproverRoles("D-42"))
	// A record that would be valid in the repository: no synthetic markers anywhere.
	real := good
	real.Approvers = nil
	for i, role := range rmE04DApproverRoles("D-42") {
		var a authorizedPerson
		a.Name, a.Role = fmt.Sprintf("Approver %c", 'A'+i), role
		a.AuthorityEvidence.Type, a.AuthorityEvidence.Reference = "delegation", fmt.Sprintf("auth-ref-%d", i+1)
		real.Approvers = append(real.Approvers, a)
	}
	real.EvidenceSource.Reference, real.RecordedBy.Name, real.DecisionText = "minutes-ref-1", "Recorder Person", "text"
	hv := *good.HumanVerification
	hv.VerifierName = "Verifier Person"
	real.HumanVerification = &hv
	tracked := map[string]bool{"D-42": true}
	load := func(m map[string]approvalRecord) func(string) (approvalRecord, error) {
		return func(n string) (approvalRecord, error) {
			if r, ok := m[n]; ok {
				return r, nil
			}
			return approvalRecord{}, fmt.Errorf("unreadable")
		}
	}
	name := recordFileName(real)
	cases := []struct {
		name  string
		files map[string]approvalRecord
		ok    bool
	}{
		{"no approval files (today)", map[string]approvalRecord{}, true},
		{"one valid repository record for a tracked decision", map[string]approvalRecord{name: real}, true},
		{"fixture record (synthetic markers) is refused", map[string]approvalRecord{recordFileName(good): good}, false},
		{"wrong file name", map[string]approvalRecord{"D-42-approval.json": real}, false},
		{"record for an untracked decision", map[string]approvalRecord{recordFileName(withID(real, "D-99")): withID(real, "D-99")}, false},
	}
	for _, c := range cases {
		names := make([]string, 0, len(c.files))
		for n := range c.files {
			names = append(names, n)
		}
		p := approvalFilesProblems(names, load(c.files), tracked)
		if (len(p) == 0) != c.ok {
			t.Errorf("%s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
}

func withID(r approvalRecord, id string) approvalRecord { r.DecisionID = id; return r }

func TestRME04DGrantConstraintRules(t *testing.T) {
	def := "CREATE TABLE identity.role_grant (\n status text NOT NULL,\n CONSTRAINT role_grant_shadow_only CHECK (status = 'SHADOW'),\n);"
	with := func(extra string) map[string]string {
		return map[string]string{grantDefiningMigration: def, "00012_identity_authentication.sql": "-- no role grant becomes ACTIVE (role_grant_shadow_only stays)\n/* role_grant untouched */ SELECT 1;", "00099_x.sql": extra}
	}
	open := rmFxTracker("D-42", "READY_FOR_REVIEW", "", "NOT_STARTED")
	approved := rmFxTracker("D-42", "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED")
	cases := []struct {
		name string
		m    map[string]string
		d42  trackerRecord
		ok   bool
	}{
		{"baseline (comments mentioning the constraint are fine)", with("SELECT 1;"), open, true},
		{"drop constraint", with("ALTER TABLE identity.role_grant DROP CONSTRAINT role_grant_shadow_only;"), open, false},
		{"drop constraint, quoted identifiers", with(`ALTER TABLE "identity"."role_grant" DROP CONSTRAINT "role_grant_shadow_only";`), open, false},
		{"rename constraint", with("ALTER TABLE identity.role_grant RENAME CONSTRAINT role_grant_shadow_only TO x;"), open, false},
		{"drop and re-create the table", with("DROP TABLE identity.role_grant; CREATE TABLE identity.role_grant (status text);"), open, false},
		{"add a permissive CHECK", with("ALTER TABLE IDENTITY.ROLE_GRANT ADD CONSTRAINT g CHECK (status IN ('SHADOW','ACTIVE'));"), open, false},
		{"defining migration redefined", map[string]string{grantDefiningMigration: "CHECK (status IN ('SHADOW','ACTIVE'))"}, open, false},
		{"grant migration after D-42 is APPROVED_VERIFIED", with("ALTER TABLE identity.role_grant DROP CONSTRAINT role_grant_shadow_only;"), approved, true},
	}
	for _, c := range cases {
		if p := grantConstraintProblems(c.m, c.d42); (len(p) == 0) != c.ok {
			t.Errorf("%s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
}

// ---------------- S4 readiness: no invoice /me operation without the conversion-payment contract (RM-E04D)

var s4Operations = map[string]bool{"listMyInvoices": true, "getMyInvoice": true, "listMyInvoiceStages": true, "getMyInvoiceTotals": true, "listMyInvoicePayments": true}
var s4Capabilities = map[string]bool{"billing.own.read": true, "payments.own.read": true}

// conversionPaymentFact: the billing schema carries the legacy conversion-payment link (sn_dot_invoice_links
// flow_kind) as a fact. Comments do not count.
var conversionPaymentFact = regexp.MustCompile(`conversion_payment|flow_kind`)

// s4ContractProblems refuses an S4 invoice operation or capability in the Group B gate until the billing
// migrations define the conversion-payment fact the recommended S4 predicate (I-L) depends on. Without it the
// predicate collapses to seller AND issuer, which on the synthetic dataset admits no invoice.
func s4ContractProblems(g rmE04CGate, billing map[string]string) []string {
	var listed []string
	for _, o := range g.ApprovedMeOperations {
		if s4Operations[o] {
			listed = append(listed, o)
		}
	}
	for _, c := range g.ApprovedSellerCapabilities {
		if s4Capabilities[c] {
			listed = append(listed, c)
		}
	}
	if len(listed) == 0 {
		return nil
	}
	for _, sql := range billing {
		if conversionPaymentFact.MatchString(sqlCode(sql)) {
			return nil
		}
	}
	return []string{fmt.Sprintf("S4 %v listed in the Group B gate, but billing has no conversion-payment fact (B-PR4 data contract)", listed)}
}

func billingMigrations(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "internal/billing/migrations/sql"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			out[e.Name()] = read(t, "internal/billing/migrations/sql/"+e.Name())
		}
	}
	return out
}

func TestRME04DS4NeedsConversionPaymentContract(t *testing.T) {
	for _, x := range s4ContractProblems(loadRMEGate(t), billingMigrations(t)) {
		t.Error(x)
	}
}

func TestRME04DS4ContractRules(t *testing.T) {
	without := map[string]string{"00001_billing.sql": "CREATE TABLE billing.invoice (seller_user_id uuid); -- flow_kind is not imported"}
	with := map[string]string{"00001_billing.sql": without["00001_billing.sql"], "00020_x.sql": "ALTER TABLE billing.invoice ADD COLUMN conversion_payment_state text;"}
	cases := []struct {
		name string
		g    rmE04CGate
		m    map[string]string
		ok   bool
	}{
		{"closed gate", rmE04CGate{}, without, true},
		{"S1 only, no contract needed", rmE04CGate{ApprovedSellerCapabilities: []string{"cases.own.read"}, ApprovedMeOperations: []string{"listMyCases"}}, without, true},
		{"S4 operation without the contract (a comment does not count)", rmE04CGate{ApprovedMeOperations: []string{"listMyInvoices"}}, without, false},
		{"S4 capability without the contract", rmE04CGate{ApprovedSellerCapabilities: []string{"billing.own.read"}}, without, false},
		{"S4 with the contract", rmE04CGate{ApprovedSellerCapabilities: []string{"billing.own.read"}, ApprovedMeOperations: []string{"listMyInvoices"}}, with, true},
	}
	for _, c := range cases {
		if p := s4ContractProblems(c.g, c.m); (len(p) == 0) != c.ok {
			t.Errorf("%s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
}

// ---------------- no claimed sign-off (RM-E04D)

// claimedSignOff matches wording that would claim a Codex or Security review that was never recorded.
var claimedSignOff = regexp.MustCompile(`(?i)(codex|security)[^|\n]{0,40}(signed[ -]off|sign-off (obtained|received|recorded|done|granted)|approved (the|this))|(signed[ -]off|approved) by (codex|security)`)

func TestRME04DNoClaimedSignOff(t *testing.T) {
	docs := []string{
		"docs/implementation/rm-e04d-codex-security-review-package.md",
		"docs/governance/proposals/rm-e04d-seller-governance-readiness.fa.md",
		"docs/implementation/rm-e04c-seller-authorization-adr-PROPOSED.md",
		"docs/implementation/rm-e04c-seller-implementation-roadmap.md",
		"docs/remediation/rm-e04b-payment-submission-non-enumeration.md",
	}
	for _, f := range docs {
		for i, l := range strings.Split(read(t, f), "\n") {
			if claimedSignOff.MatchString(l) && !regexp.MustCompile(`(?i)pending|not (obtained|recorded|claimed)|no .*(sign-off|review)|required|owed|requested`).MatchString(l) {
				t.Errorf("%s:%d claims a sign-off: %s", f, i+1, strings.TrimSpace(l))
			}
		}
	}
	pkg := read(t, "docs/implementation/rm-e04d-codex-security-review-package.md")
	if !strings.Contains(pkg, "| Status | **REVIEW REQUESTED — PENDING.**") {
		t.Error("the review package must state PENDING until a review is actually recorded")
	}
	for _, s := range []string{"Codex sign-off: APPROVED", "Security sign-off: APPROVED"} {
		if strings.Contains(pkg, s) {
			t.Errorf("review package claims %q", s)
		}
	}
	if !claimedSignOff.MatchString("Codex signed off the contract") || claimedSignOff.MatchString("Codex review requested") {
		t.Error("sign-off detector is miscalibrated")
	}
}
