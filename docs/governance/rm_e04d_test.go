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
	"encoding/json"
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

// ---------------- S4 readiness: fail closed until contract, schema and enforcement are all proven (RM-E04D)
//
// A token in a migration (e.g. a column named conversion_payment_note) proves nothing. S4 may enter the Group B
// gate only when the S4 readiness manifest (rm-e04d-s4-readiness.json) carries three independent proofs:
//   1. contract   — an approved, verified conversion-payment data-contract decision (R6 rules, Finance + Security);
//   2. schema     — the declared billing column exists NOT NULL with a CHECK listing an explicit unknown state;
//   3. enforcement — named tests exist for every I-L case (seller/issuer mismatch, unknown conversion state, …)
//                    and exercise the declared column.
// Each proof is checked by its own function; the gate rule requires all of them.

var s4Operations = map[string]bool{"listMyInvoices": true, "getMyInvoice": true, "listMyInvoiceStages": true, "getMyInvoiceTotals": true, "listMyInvoicePayments": true}
var s4Capabilities = map[string]bool{"billing.own.read": true, "payments.own.read": true}

// Approver floor of the conversion-payment contract: roadmap B5 "Security + Finance review".
var s4ContractFloorRoles = []string{"Finance owner", "Security owner"}
var s4ContractTitle = regexp.MustCompile(`(?i)conversion[- ]payment`)

// The I-L cases every future S4 implementation must prove with synthetic fixtures.
var s4EnforcementCategories = []string{
	"foreign_conversion_hidden",            // seller = me, conversion payment, issuer ≠ me → invisible, not counted
	"own_conversion_visible",               // seller = me, conversion payment, issuer = me → visible
	"non_conversion_mismatch_visible",      // seller = me, NOT conversion, issuer ≠ me → visible (legacy 2.0.160)
	"unknown_conversion_state_fail_closed", // seller = me, conversion state UNKNOWN, issuer ≠ me → invisible
	"null_owner_hidden",                    // seller or conversion fact NULL → invisible for everyone
	"out_of_scope_not_found_parity",        // hidden invoice ≡ missing invoice (same 404 body)
}

type s4Manifest struct {
	Purpose            string `json:"purpose"`
	ContractDecisionID string `json:"contract_decision_id"`
	Schema             struct {
		Migration    string `json:"migration"`
		Table        string `json:"table"`
		Column       string `json:"column"`
		UnknownState string `json:"unknown_state"`
	} `json:"schema"`
	EnforcementTests map[string]string `json:"enforcement_tests"`
}

func s4Listed(g rmE04CGate) []string {
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
	return listed
}

var sqlIdent = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// s4ContractApprovalProblems — proof 1: the contract decision is registered, is the conversion-payment contract,
// and is APPROVED_VERIFIED through a valid repository-mode R6 record with the Finance and Security roles.
func s4ContractApprovalProblems(m s4Manifest, recs map[string]trackerRecord, titles map[string]string, load approvalLoader, inRepo bool) []string {
	id := m.ContractDecisionID
	if id == "" {
		return []string{"no conversion-payment contract decision named"}
	}
	var p []string
	if title, ok := titles[id]; !ok {
		p = append(p, id+" has no R5 inventory row")
	} else if !s4ContractTitle.MatchString(title) {
		p = append(p, fmt.Sprintf("%s (%q) is not the conversion-payment data contract", id, title))
	}
	r, ok := recs[id]
	if !ok {
		return append(p, id+" is not in the R6 tracker")
	}
	if r.Status != "APPROVED_VERIFIED" {
		return append(p, fmt.Sprintf("%s is %s, not APPROVED_VERIFIED", id, r.Status))
	}
	return append(p, rmE04CDecisionProblems(r, "", s4ContractFloorRoles, load, inRepo)...)
}

// s4SchemaProblems — proof 2: the declared column exists in the declared billing migration, NOT NULL, with a CHECK
// that lists the explicit unknown state. A column that merely contains a token does not qualify.
func s4SchemaProblems(m s4Manifest, billing map[string]string) []string {
	sc := m.Schema
	col, tbl, unk := strings.ToLower(sc.Column), strings.ToLower(sc.Table), sc.UnknownState
	if sc.Migration == "" || col == "" || tbl == "" || unk == "" {
		return []string{"schema proof incomplete (migration, table, column and unknown_state are all required)"}
	}
	if !sqlIdent.MatchString(col) || !regexp.MustCompile(`^[a-z_][a-z0-9_]*\.[a-z_][a-z0-9_]*$`).MatchString(tbl) {
		return []string{"schema proof: table must be schema.table and column a plain identifier"}
	}
	sql, ok := billing[sc.Migration]
	if !ok {
		return []string{sc.Migration + " is not a billing migration"}
	}
	code := sqlCode(sql)
	var p []string
	if !strings.Contains(code, tbl) {
		p = append(p, fmt.Sprintf("%s does not touch %s", sc.Migration, tbl))
	}
	if !regexp.MustCompile(`\b` + col + `\b[^,;]*\bnot null\b`).MatchString(code) {
		p = append(p, fmt.Sprintf("%s does not define %s NOT NULL (unknown must be explicit, never NULL)", sc.Migration, col))
	}
	if !regexp.MustCompile(`check\s*\([^;]*\b` + col + `\b[^;]*'` + regexp.QuoteMeta(strings.ToLower(unk)) + `'`).MatchString(code) {
		p = append(p, fmt.Sprintf("%s has no CHECK on %s listing the unknown state %q", sc.Migration, col, unk))
	}
	return p
}

// s4EnforcementProblems — proof 3: every I-L case names an existing Go test that references the declared column.
func s4EnforcementProblems(m s4Manifest, readFile func(path string) (string, bool)) []string {
	var p []string
	seen := map[string]string{}
	for _, cat := range s4EnforcementCategories {
		ref := m.EnforcementTests[cat]
		path, name, ok := strings.Cut(ref, ":")
		if ref == "" || !ok || name == "" {
			p = append(p, cat+": no enforcement test named (want path_test.go:TestName)")
			continue
		}
		if prev, dup := seen[ref]; dup {
			p = append(p, fmt.Sprintf("%s: same test as %s; each case needs its own test", cat, prev))
		}
		seen[ref] = cat
		if !strings.HasPrefix(path, "internal/") || !strings.HasSuffix(path, "_test.go") {
			p = append(p, cat+": test must be a _test.go file under internal/")
			continue
		}
		src, ok := readFile(path)
		if !ok {
			p = append(p, cat+": "+path+" does not exist")
			continue
		}
		if !regexp.MustCompile(`func ` + regexp.QuoteMeta(name) + `\(t \*testing\.T\)`).MatchString(src) {
			p = append(p, fmt.Sprintf("%s: %s does not define %s", cat, path, name))
		}
		if m.Schema.Column == "" || !strings.Contains(src, m.Schema.Column) {
			p = append(p, fmt.Sprintf("%s: %s does not exercise the declared column %q", cat, path, m.Schema.Column))
		}
	}
	for cat := range m.EnforcementTests {
		if !slicesContains(s4EnforcementCategories, cat) {
			p = append(p, "unknown enforcement category "+cat)
		}
	}
	return p
}

func slicesContains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// s4ReadinessProblems — the gate rule: S4 in the gate requires all three proofs; otherwise nothing is required.
func s4ReadinessProblems(g rmE04CGate, m s4Manifest, recs map[string]trackerRecord, titles map[string]string, load approvalLoader, inRepo bool,
	billing map[string]string, readFile func(string) (string, bool)) []string {
	listed := s4Listed(g)
	if len(listed) == 0 {
		return nil
	}
	var p []string
	for _, x := range s4ContractApprovalProblems(m, recs, titles, load, inRepo) {
		p = append(p, "contract: "+x)
	}
	for _, x := range s4SchemaProblems(m, billing) {
		p = append(p, "schema: "+x)
	}
	for _, x := range s4EnforcementProblems(m, readFile) {
		p = append(p, "enforcement: "+x)
	}
	if len(p) > 0 {
		p = append([]string{fmt.Sprintf("S4 %v listed in the Group B gate without the three S4 readiness proofs", listed)}, p...)
	}
	return p
}

func loadS4Manifest(t *testing.T) s4Manifest {
	t.Helper()
	var m s4Manifest
	dec := json.NewDecoder(strings.NewReader(read(t, "docs/governance/rm-e04d-s4-readiness.json")))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
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

func repoFile(path string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(root, filepath.Clean(path)))
	return string(b), err == nil
}

func TestRME04DS4NeedsConversionPaymentContract(t *testing.T) {
	m := loadS4Manifest(t)
	// The manifest's categories are exactly the required I-L cases (no silent removal of a case).
	if len(m.EnforcementTests) != len(s4EnforcementCategories) {
		t.Errorf("manifest has %d enforcement categories, want %d", len(m.EnforcementTests), len(s4EnforcementCategories))
	}
	for _, c := range s4EnforcementCategories {
		if _, ok := m.EnforcementTests[c]; !ok {
			t.Errorf("manifest lacks enforcement category %s", c)
		}
	}
	for _, x := range s4ReadinessProblems(loadRMEGate(t), m, trackerByID(t), r5Titles(t), repoLoader, true, billingMigrations(t), repoFile) {
		t.Error(x)
	}
}

// ---------------- S4 fixtures (synthetic, in code)

func s4FxManifest(contractID string) s4Manifest {
	var m s4Manifest
	m.ContractDecisionID = contractID
	m.Schema.Migration, m.Schema.Table, m.Schema.Column, m.Schema.UnknownState = "00020_conversion.sql", "billing.invoice", "conversion_payment_state", "UNKNOWN"
	m.EnforcementTests = map[string]string{}
	for i, c := range s4EnforcementCategories {
		m.EnforcementTests[c] = fmt.Sprintf("internal/billing/http/me_invoices_test.go:TestMeInvoices%d", i)
	}
	return m
}

const s4FxSchema = "ALTER TABLE billing.invoice ADD COLUMN conversion_payment_state text NOT NULL DEFAULT 'UNKNOWN',\n" +
	"  ADD CONSTRAINT invoice_conversion_payment_state CHECK (conversion_payment_state IN ('CONVERSION_PAYMENT', 'NOT_CONVERSION_PAYMENT', 'UNKNOWN'));"

func s4FxTests() func(string) (string, bool) {
	var b strings.Builder
	b.WriteString("package http_test\n// synthetic seller ≠ issuer and conversion_payment_state UNKNOWN fixtures\n")
	for i := range s4EnforcementCategories {
		fmt.Fprintf(&b, "func TestMeInvoices%d(t *testing.T) {}\n", i)
	}
	src := b.String()
	return func(p string) (string, bool) {
		if p == "internal/billing/http/me_invoices_test.go" {
			return src, true
		}
		return "", false
	}
}

func TestRME04DS4ContractRules(t *testing.T) {
	base := map[string]string{"00001_billing.sql": "CREATE TABLE billing.invoice (seller_user_id uuid); -- flow_kind / conversion_payment not imported"}
	withSchema := map[string]string{"00001_billing.sql": base["00001_billing.sql"], "00020_conversion.sql": s4FxSchema}
	const cid = "S4-CONTRACT-FIXTURE"
	titles := map[string]string{cid: "Conversion-payment data contract (sn_dot_invoice_links flow_kind)", "D-42": "Can HR edit sales capabilities?"}
	approved := func(id string, roles []string, verification string) (map[string]trackerRecord, approvalLoader) {
		return map[string]trackerRecord{id: rmFxTracker(id, "APPROVED_VERIFIED", rmFxRecordPath, "VERIFIED")}, loaderOf(rmFxDecision(t, id, "APPROVED", verification, roles))
	}
	goodRecs, goodLoad := approved(cid, s4ContractFloorRoles, "VERIFIED")
	s4Gate := rmE04CGate{ApprovedSellerCapabilities: []string{"billing.own.read"}, ApprovedMeOperations: []string{"listMyInvoices"}}
	full := s4FxManifest(cid)

	type tc struct {
		name    string
		g       rmE04CGate
		m       s4Manifest
		recs    map[string]trackerRecord
		load    approvalLoader
		billing map[string]string
		files   func(string) (string, bool)
		ok      bool
	}
	none := func(string) (string, bool) { return "", false }
	cases := []tc{
		{"closed gate needs nothing", rmE04CGate{}, s4Manifest{}, nil, goodLoad, base, none, true},
		{"S1 only needs nothing", rmE04CGate{ApprovedSellerCapabilities: []string{"cases.own.read"}, ApprovedMeOperations: []string{"listMyCases"}}, s4Manifest{}, nil, goodLoad, base, none, true},
		{"S4 with an empty manifest (today)", s4Gate, s4Manifest{}, nil, goodLoad, base, none, false},
		{"S4 with an unrelated column that merely contains the token", s4Gate, s4Manifest{}, nil, goodLoad,
			map[string]string{"00001_billing.sql": base["00001_billing.sql"], "00021_note.sql": "ALTER TABLE billing.invoice ADD COLUMN conversion_payment_note text;"}, none, false},
		{"S4 with a token-only manifest pointing at the unrelated column", s4Gate, func() s4Manifest {
			m := full
			m.Schema.Migration, m.Schema.Column = "00021_note.sql", "conversion_payment_note"
			return m
		}(), goodRecs, goodLoad, map[string]string{"00021_note.sql": "ALTER TABLE billing.invoice ADD COLUMN conversion_payment_note text;"}, s4FxTests(), false},
		{"S4 with all three proofs", s4Gate, full, goodRecs, goodLoad, withSchema, s4FxTests(), true},
		{"contract not approved (READY_FOR_REVIEW)", s4Gate, full, map[string]trackerRecord{cid: rmFxTracker(cid, "READY_FOR_REVIEW", "", "NOT_STARTED")}, goodLoad, withSchema, s4FxTests(), false},
		{"contract approval unverified", s4Gate, full, goodRecs, loaderOf(rmFxDecision(t, cid, "APPROVED", "", s4ContractFloorRoles)), withSchema, s4FxTests(), false},
		{"contract approved without Security", s4Gate, full, goodRecs, loaderOf(rmFxDecision(t, cid, "APPROVED", "VERIFIED", []string{"Finance owner"})), withSchema, s4FxTests(), false},
		{"an unrelated approved decision named as the contract", s4Gate, s4FxManifest("D-42"), func() map[string]trackerRecord { r, _ := approved("D-42", s4ContractFloorRoles, "VERIFIED"); return r }(),
			loaderOf(rmFxDecision(t, "D-42", "APPROVED", "VERIFIED", s4ContractFloorRoles)), withSchema, s4FxTests(), false},
		{"schema missing", s4Gate, full, goodRecs, goodLoad, base, s4FxTests(), false},
		{"schema column nullable", s4Gate, full, goodRecs, goodLoad, map[string]string{"00020_conversion.sql": strings.Replace(s4FxSchema, "NOT NULL ", "", 1)}, s4FxTests(), false},
		{"schema CHECK without the unknown state", s4Gate, full, goodRecs, goodLoad, map[string]string{"00020_conversion.sql": strings.Replace(s4FxSchema, ", 'UNKNOWN'))", "))", 1)}, s4FxTests(), false},
		{"schema only in a comment", s4Gate, full, goodRecs, goodLoad, map[string]string{"00020_conversion.sql": "-- " + strings.ReplaceAll(s4FxSchema, "\n", "\n-- ")}, s4FxTests(), false},
		{"enforcement tests missing", s4Gate, full, goodRecs, goodLoad, withSchema, none, false},
		{"one enforcement category missing", s4Gate, func() s4Manifest {
			m := s4FxManifest(cid)
			delete(m.EnforcementTests, "unknown_conversion_state_fail_closed")
			return m
		}(), goodRecs, goodLoad, withSchema, s4FxTests(), false},
		{"two categories share one test", s4Gate, func() s4Manifest {
			m := s4FxManifest(cid)
			m.EnforcementTests["own_conversion_visible"] = m.EnforcementTests["foreign_conversion_hidden"]
			return m
		}(), goodRecs, goodLoad, withSchema, s4FxTests(), false},
		{"enforcement tests that never touch the column", s4Gate, full, goodRecs, goodLoad, withSchema, func(p string) (string, bool) {
			src, ok := s4FxTests()(p)
			return strings.ReplaceAll(src, "conversion_payment_state", "something_else"), ok
		}, false},
	}
	for _, c := range cases {
		if p := s4ReadinessProblems(c.g, c.m, c.recs, titles, c.load, false, c.billing, c.files); (len(p) == 0) != c.ok {
			t.Errorf("%s: accepted=%v, want %v (%v)", c.name, len(p) == 0, c.ok, p)
		}
	}
	// In repository mode the synthetic contract approval can never satisfy proof 1.
	if p := s4ReadinessProblems(s4Gate, full, goodRecs, titles, goodLoad, true, withSchema, s4FxTests()); len(p) == 0 {
		t.Error("a synthetic contract approval satisfied the S4 rule in repository mode")
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
