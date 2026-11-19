package governance

// RM-E04H — R5-SEC-01 person-identity decision package (documentation-level guards only).
//
// These tests protect the decision package, not runtime identity: nothing in the repository resolves an account to a
// verified person until a reviewed F-PR2 exists. They check that:
//   1. the package quotes the original R5-SEC-01 options A–D verbatim and the dashboard pre-selects none;
//   2. the repository state the package describes still holds (R5-SEC-01 / D-34 / D-42 undecided, no approval
//      record, nominations provisional, no identity or organization migration added, resolver resolves nothing,
//      Group B gate and S4 manifest closed);
//   3. the approver slots the package relies on ("five distinct humans") match the intake blank and m2Approvers;
//   4. a future BLOCKED resolver state (proposed contract §5) fails closed in the existing RM-E04G evaluator;
//   5. the documents stay DRAFT, NO-GO, review PENDING, with a blank decision form and no claimed sign-off.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	rmE04HPackage   = "docs/security/rm-e04h-person-identity-decision-package.md"
	rmE04HDashboard = "docs/governance/proposals/rm-e04h-person-identity-decision-dashboard.fa.md"
	rmE04HHandoff   = "docs/implementation/rm-e04h-review-handoff.md"
	rmE04HR5Record  = "docs/remediation/r5-person-identity-sod-DRAFT.md"
	rmE04HIntake    = "docs/governance/intake/m2/R5-SEC-01_proposed_ADR-v1.0-DECISION.BLANK.json"
	rmE04HID        = "R5-SEC-01 (proposed ADR)"
	rmE04HBoxes     = "☐ الف  ☐ ب  ☐ ج  ☐ د"
)

var rmE04HDocs = []string{rmE04HPackage, rmE04HDashboard, rmE04HHandoff}

// Migration files present at the RM-E04H baseline (19ab171). This package adds none; an identity or organization
// migration (e.g. a person-link table) needs its own reviewed authorisation (roadmap F-PR5).
var rmE04HBaselineMigrations = map[string][]string{
	"internal/identity/migrations/sql": {
		"00001_identity.sql", "00002_identity_customer_capabilities.sql", "00003_identity_sales_capabilities.sql",
		"00004_identity_billing_capabilities.sql", "00005_identity_reporting_capabilities.sql",
		"00006_identity_finance_capabilities.sql", "00007_identity_biawin_capabilities.sql",
		"00008_identity_write_capabilities.sql", "00009_identity_payment_capabilities.sql",
		"00010_identity_commission_capabilities.sql", "00011_identity_operations_capabilities.sql",
		"00012_identity_authentication.sql",
	},
	"internal/organization/migrations/sql": {"00001_organization.sql", "00002_organization_hr_requests.sql"},
}

// rmE04HOriginalOptions extracts the "Alternatives" cell of the R5-SEC-01 record (the authoritative wording).
func rmE04HOriginalOptions(t *testing.T) []string {
	t.Helper()
	doc := read(t, rmE04HR5Record)
	i := strings.Index(doc, "### R5-SEC-01 (proposed ADR)")
	if i < 0 {
		t.Fatal("R5-SEC-01 record not found in the R5 package")
	}
	block := doc[i:]
	if j := strings.Index(block[4:], "\n### "); j >= 0 {
		block = block[:j+4]
	}
	m := regexp.MustCompile(`(?m)^\| Alternatives \| (.+) \|$`).FindStringSubmatch(block)
	if m == nil {
		t.Fatal("Alternatives row not found in the R5-SEC-01 record")
	}
	return strings.Split(m[1], "<br>")
}

func TestRME04HOriginalOptionsQuotedVerbatim(t *testing.T) {
	opts := rmE04HOriginalOptions(t)
	if len(opts) != 4 {
		t.Fatalf("R5-SEC-01 must keep exactly four original options, got %d", len(opts))
	}
	pkg := read(t, rmE04HPackage)
	for i, o := range opts {
		letter := string("ABCD"[i])
		if !strings.HasPrefix(o, letter+" — ") {
			t.Errorf("original option %d does not start with %q: %q", i+1, letter, o)
		}
		if !strings.Contains(pkg, "| "+letter+" | "+o+" |") {
			t.Errorf("package §2.2 does not quote option %s verbatim: %q", letter, o)
		}
	}
	dash := read(t, rmE04HDashboard)
	if strings.Count(dash, rmE04HBoxes) != 1 {
		t.Errorf("dashboard decision form must list the four options as unticked boxes exactly once (%q)", rmE04HBoxes)
	}
	for _, letter := range []string{"الف (A)", "ب (B)", "ج (C)", "د (D)"} {
		if !strings.Contains(dash, "| "+letter+" |") {
			t.Errorf("dashboard does not explain option %s", letter)
		}
	}
	for _, f := range []string{rmE04HPackage, rmE04HDashboard} {
		if !strings.Contains(read(t, f), "نه انتخاب") && !strings.Contains(read(t, f), "not a selection") {
			t.Errorf("%s: the recommendation is not marked as a non-selection", f)
		}
	}
}

func TestRME04HRepositoryStateMatchesPackage(t *testing.T) {
	recs := trackerByID(t)
	for _, id := range []string{rmE04HID, "D-34", "D-42"} {
		r, ok := recs[id]
		if !ok {
			t.Fatalf("%s missing from the tracker", id)
		}
		if r.Status != "READY_FOR_REVIEW" || r.ApprovalRecord != "" || r.Verification != "NOT_STARTED" {
			t.Errorf("%s changed (%s, %q, %s); the RM-E04H package states it is undecided", id, r.Status, r.ApprovalRecord, r.Verification)
		}
		for k, v := range r.Outcome {
			if strings.TrimSpace(v) != "" {
				t.Errorf("%s: outcome %s filled; the package states no option is selected", id, k)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "docs/governance/approvals")); !os.IsNotExist(err) {
		t.Error("docs/governance/approvals exists; the package states no approval record is registered")
	}
	reg, _ := rmE04FLoadRegister(t)
	for _, n := range reg.Nominees {
		if n.PersonRef != "" || n.IdentityResolution != "NOT_STARTED" {
			t.Errorf("nominee %s has an identity claim; no identity link may be created", n.NomineeKey)
		}
	}
	for _, n := range reg.Nominations {
		if n.State != "NOMINATED" && n.State != "UNASSIGNED" {
			t.Errorf("%s: %s; no authority may be verified without evidence", n.AuthorityKey, n.State)
		}
	}
	for dir, want := range rmE04HBaselineMigrations {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, e := range entries {
			got = append(got, e.Name())
		}
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s changed: %v; RM-E04H authorises no identity / organization migration", dir, got)
		}
	}
	for _, in := range []string{"acct-1", "مهرداد حافظی", "Mehrdad Hafezi", "+989120000000", "someone@example.com", "github:login"} {
		if r := rmE04GRepositoryResolver(in); r.State != idUnknown || r.PersonID != "" {
			t.Errorf("repository resolver resolved %q (%+v); no identity link exists", in, r)
		}
	}
	g := loadRMEGate(t)
	if g.M2Record1ID != "" || g.M2Record2ID != "" || len(g.ApprovedSellerCapabilities) > 0 || len(g.ApprovedMeOperations) > 0 {
		t.Error("Group B gate is not closed; the package states Group B is BLOCKED")
	}
	if m := loadS4Manifest(t); m.ContractDecisionID != "" {
		t.Error("S4 manifest is not empty; the package states S4 is BLOCKED")
	}
}

func TestRME04HSignerSlotsMatchSources(t *testing.T) {
	var blank struct {
		IndependentApprovals []string `json:"independent_approvals"`
	}
	if err := json.Unmarshal([]byte(read(t, rmE04HIntake)), &blank); err != nil {
		t.Fatal(err)
	}
	want := append([]string(nil), m2Approvers[rmE04HID]...)
	got := append([]string(nil), blank.IndependentApprovals...)
	sort.Strings(want)
	sort.Strings(got)
	if len(want) != 4 || strings.Join(want, ",") != strings.Join(got, ",") {
		t.Fatalf("R5-SEC-01 approver roles drifted: m2Approvers %v, intake %v; the package's signer analysis must be redone", want, got)
	}
	pkg, dash := read(t, rmE04HPackage), read(t, rmE04HDashboard)
	for _, role := range want {
		if !strings.Contains(pkg, role) {
			t.Errorf("package does not name required approver %s", role)
		}
	}
	if !strings.Contains(pkg, "Minimum distinct humans for a valid R5-SEC-01 decision: five") ||
		!strings.Contains(dash, "حداقل پنج انسان متفاوت") {
		t.Error("minimum of five distinct humans (four approvers + one T5 verifier) not stated")
	}
	reg, _ := rmE04FLoadRegister(t)
	for _, n := range reg.Nominations {
		if n.R6Role == "HR owner" && n.NomineeKey != "" {
			t.Error("an HR owner nominee now exists; the package and dashboard state HR owner is UNASSIGNED")
		}
		if n.R6Role == "T5 verifier" && (n.State != "UNASSIGNED" || n.NomineeKey != "") {
			t.Error("T5 verifier is no longer UNASSIGNED; the package must be updated by a reviewed change")
		}
	}
}

func TestRME04HResolverContractFailsClosed(t *testing.T) {
	for _, st := range []string{"BLOCKED", "PENDING", "", "verified"} {
		r := rmE04GResolve("acct-x", func(string) rmE04GResolution { return rmE04GResolution{PersonID: "p-x", State: st} })
		if r.State == idVerified {
			t.Errorf("resolver state %q resolved to VERIFIED", st)
		}
		res, _ := rmE04GIndependence(claims(floor3, "acct-r", "acct-v"), []string{"Product owner", "Sales owner", "Security owner"},
			func(string) rmE04GResolution { return rmE04GResolution{PersonID: "p-x", State: st} })
		if res == indepVerified {
			t.Errorf("independence VERIFIED with every participant in state %q", st)
		}
	}
	pkg := read(t, rmE04HPackage)
	for _, s := range []string{"VERIFIED(person_id, evidence_reference, validity)", "UNKNOWN(reason)", "CONFLICT(reason)", "BLOCKED(reason)"} {
		if !strings.Contains(pkg, s) {
			t.Errorf("contract outcome %s missing from the package", s)
		}
	}
}

func TestRME04HDocumentsMarkedNotApproved(t *testing.T) {
	known := ids(read(t, "docs/remediation/r5-decision-inventory.md"))
	for _, r := range loadTracker(t).Records {
		known[strings.SplitN(r.ID, " ", 2)[0]] = true
	}
	idPattern := regexp.MustCompile(`\b(?:D-[0-9]+[a-e]?|OPD-[0-9]+|R5-[A-Z]+-[0-9]+)\b`)
	notClaim := regexp.MustCompile(`(?i)pending|not (obtained|recorded|claimed)|no .*(sign-off|review)|required|requested`)
	for _, f := range rmE04HDocs {
		d := read(t, f)
		if head := d[:min(len(d), 600)]; !strings.Contains(head, "DRAFT — NOT APPROVED") {
			t.Errorf("%s: no DRAFT — NOT APPROVED marker near the top", f)
		}
		for _, s := range []string{"NO-GO", "Group B", "S4", "BLOCKED"} {
			if !strings.Contains(d, s) {
				t.Errorf("%s: %q not stated", f, s)
			}
		}
		if strings.ContainsAny(d, "☑☒✔✓■") || strings.Contains(d, "[x]") || strings.Contains(d, "[X]") {
			t.Errorf("%s: a choice or checklist item is pre-ticked", f)
		}
		for i, l := range strings.Split(d, "\n") {
			if claimedSignOff.MatchString(l) && !notClaim.MatchString(l) {
				t.Errorf("%s:%d claims a sign-off: %s", f, i+1, strings.TrimSpace(l))
			}
		}
		for _, id := range idPattern.FindAllString(d, -1) {
			if !known[id] && !known[strings.TrimRight(id, "abcde")] {
				t.Errorf("%s cites decision id %s not in the tracker / inventory", f, id)
			}
		}
	}
	h := read(t, rmE04HHandoff)
	if !strings.Contains(h, "| Status | **REVIEW PENDING.** |") {
		t.Error("handoff must state REVIEW PENDING until the reviewers record a review themselves")
	}
	rows := 0
	for _, l := range strings.Split(h, "\n") {
		if strings.HasPrefix(l, "| H-RV-") {
			rows++
			if !strings.HasSuffix(strings.TrimSpace(l), "| PENDING |") {
				t.Errorf("review item not PENDING: %s", l)
			}
		}
	}
	if rows == 0 {
		t.Error("handoff has no review items")
	}
	rmE04HBlankFormProblems(t, read(t, rmE04HDashboard))
}

// rmE04HBlankFormProblems: every value cell of the dashboard decision form (§۸) is empty or "—", except the fixed
// decision id and the unticked option boxes. No outcome, name, authority evidence, signature or verification may be
// pre-filled.
func rmE04HBlankFormProblems(t *testing.T, dash string) {
	t.Helper()
	i := strings.Index(dash, "## ۸.")
	if i < 0 {
		t.Fatal("dashboard decision form (§۸) not found")
	}
	lines := strings.Split(dash[i:], "\n")
	sep := regexp.MustCompile(`^\|(\s*-+\s*\|)+$`)
	fixed := map[string]string{"Decision ID": "R5-SEC-01 (proposed ADR) — نسخهٔ 1.0", "گزینهٔ انتخاب‌شده": rmE04HBoxes}
	checked := 0
	for k, l := range lines {
		if !strings.HasPrefix(l, "|") || sep.MatchString(l) || (k+1 < len(lines) && sep.MatchString(lines[k+1])) {
			continue
		}
		cells := strings.Split(strings.Trim(l, "|"), "|")
		key := strings.TrimSpace(cells[0])
		for _, c := range cells[1:] {
			v := strings.TrimSpace(c)
			if want, ok := fixed[key]; ok {
				if v != want {
					t.Errorf("decision form %q altered: %q", key, v)
				}
				continue
			}
			if v != "" && v != "—" {
				t.Errorf("decision form row %q is pre-filled: %q", key, v)
			}
		}
		checked++
	}
	if checked < 12 {
		t.Errorf("decision form has %d value rows; expected the full blank form", checked)
	}
}
