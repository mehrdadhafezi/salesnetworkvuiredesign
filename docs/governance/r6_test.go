package governance

// R6 management decision workflow checks (offline). They keep the R6 tracker consistent with the R5
// records and make it impossible to record an approval in the repository without an approval record
// that passes the structural check and, for a verified state, a human verification entry.
//
// Limits: these checks cannot authenticate a person, a signature or an authority document. That is a
// human verification step (r6-approval-evidence-standard.md).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var lifecycle = []string{"EVIDENCE_MISSING", "DRAFT", "READY_FOR_REVIEW", "UNDER_HUMAN_REVIEW",
	"DECISION_RECORDED_PENDING_VERIFICATION", "APPROVED_VERIFIED", "REJECTED_VERIFIED", "DEFERRED",
	"SUPERSEDED_VERIFIED", "VOID_VERIFIED"}

// States that exist only through an approval record (DEFERRED included: it is a decision of the approvers).
var needsRecord = map[string]bool{"UNDER_HUMAN_REVIEW": true, "DECISION_RECORDED_PENDING_VERIFICATION": true,
	"APPROVED_VERIFIED": true, "REJECTED_VERIFIED": true, "DEFERRED": true, "SUPERSEDED_VERIFIED": true, "VOID_VERIFIED": true}

var r5StatusOf = map[string]string{
	"READY FOR HUMAN REVIEW":                                                "READY_FOR_REVIEW",
	"NEEDS FURTHER EVIDENCE BEFORE A DECISION":                              "EVIDENCE_MISSING",
	"NEEDS PRODUCT + FINANCE MAPPING CONFIRMATION (C-03) BEFORE A DECISION": "DRAFT",
}

var meetingDoc = map[string]string{
	"M1": "r6-m1-governance-data-session-fa.md", "M2": "r6-m2-security-session-fa.md", "M3": "r6-m3-sales-session-fa.md",
	"M4-A": "r6-m4-finance-sessions-fa.md", "M4-B": "r6-m4-finance-sessions-fa.md", "M4-C": "r6-m4-finance-sessions-fa.md",
	"M4-D": "r6-m4-finance-sessions-fa.md", "M5": "r6-m5-m6-operations-biawin-reporting-fa.md",
	"M6": "r6-m5-m6-operations-biawin-reporting-fa.md",
}

type trackerRecord struct {
	ID             string            `json:"id"`
	R5Package      string            `json:"r5_package"`
	R5Readiness    string            `json:"r5_readiness"`
	Status         string            `json:"status"`
	BlockedBy      string            `json:"blocked_by"`
	Priority       string            `json:"priority"`
	Blocker        string            `json:"blocker"`
	Meeting        string            `json:"meeting"`
	Outcome        map[string]string `json:"outcome"`
	ApprovalRecord string            `json:"approval_record"`
	Verification   string            `json:"verification"`
}

type tracker struct {
	States    []string        `json:"lifecycle_states"`
	Records   []trackerRecord `json:"records"`
	Conflicts []struct {
		ID       string `json:"id"`
		R5Status string `json:"r5_status"`
	} `json:"conflicts"`
}

func loadTracker(t *testing.T) tracker {
	t.Helper()
	var tr tracker
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r6-decision-tracker.json")), &tr); err != nil {
		t.Fatal(err)
	}
	return tr
}

func pgovIDs() []string {
	var out []string
	for n := 3; n <= 22; n++ {
		out = append(out, fmt.Sprintf("D-%02d", n))
	}
	return out
}

// All 41 R5 records and the 20 P-GOV identifiers are tracked exactly once; nothing disappears.
func TestR6TrackerAccountsForEveryR5Record(t *testing.T) {
	tr := loadTracker(t)
	var idx struct {
		Records []struct{ ID, Package, Readiness string } `json:"decision_records"`
	}
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r5-evidence-index.json")), &idx); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	for _, r := range idx.Records {
		want[r.ID] = r.Readiness
	}
	if len(want) != 41 {
		t.Fatalf("R5 index has %d records, want 41", len(want))
	}
	for _, id := range pgovIDs() {
		want[id] = "EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED"
	}
	seen := map[string]bool{}
	for _, r := range tr.Records {
		if seen[r.ID] {
			t.Errorf("%s tracked twice", r.ID)
		}
		seen[r.ID] = true
		if rd, ok := want[r.ID]; !ok {
			t.Errorf("%s is not an R5 decision", r.ID)
		} else if r.R5Readiness != rd {
			t.Errorf("%s: R5 readiness %q differs from R5 (%q)", r.ID, r.R5Readiness, rd)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("%s from R5 is missing from the R6 tracker", id)
		}
	}
	if len(tr.States) != len(lifecycle) {
		t.Errorf("lifecycle states %v, want %v", tr.States, lifecycle)
	}
}

// Without an approval record a status is derived from R5 alone, and outcome fields stay empty.
func TestR6NoStatusOrOutcomeWithoutApprovalRecord(t *testing.T) {
	tr := loadTracker(t)
	allowed := map[string]bool{}
	for _, s := range lifecycle {
		allowed[s] = true
	}
	for _, r := range tr.Records {
		if !allowed[r.Status] {
			t.Errorf("%s: unknown status %q", r.ID, r.Status)
			continue
		}
		if needsRecord[r.Status] {
			checkRecordedDecision(t, r)
			continue
		}
		if r.ApprovalRecord != "" || r.Verification != "NOT_STARTED" {
			t.Errorf("%s: approval record / verification set but status %s", r.ID, r.Status)
		}
		for k, v := range r.Outcome {
			if strings.TrimSpace(v) != "" {
				t.Errorf("%s: outcome field %s is %q without an approval record", r.ID, k, v)
			}
		}
		exp := r5StatusOf[r.R5Readiness]
		if r.R5Package == "P-GOV" {
			exp = "EVIDENCE_MISSING"
		}
		if r.Status != exp {
			t.Errorf("%s: status %s does not follow R5 readiness (%s)", r.ID, r.Status, exp)
		}
		if strings.HasPrefix(r.ID, "D-35") && r.BlockedBy != "C-03" {
			t.Errorf("%s must stay blocked by C-03", r.ID)
		}
	}
	// No approval exists today; the directory must not appear without real records.
	if entries, err := os.ReadDir(filepath.Join(root, "docs/governance/approvals")); err == nil {
		for _, e := range entries {
			rec, err := loadApproval(filepath.Join(root, "docs/governance/approvals", e.Name()))
			if err != nil {
				t.Errorf("%s: %v", e.Name(), err)
				continue
			}
			if _, problems := validateApprovalRecord(rec, true); len(problems) > 0 {
				t.Errorf("%s: %v", e.Name(), problems)
			}
		}
	}
}

func checkRecordedDecision(t *testing.T, r trackerRecord) {
	t.Helper()
	if r.ApprovalRecord == "" {
		t.Errorf("%s: status %s requires an approval record", r.ID, r.Status)
		return
	}
	rec, err := loadApproval(filepath.Join(root, r.ApprovalRecord))
	if err != nil {
		t.Errorf("%s: %v", r.ID, err)
		return
	}
	level, problems := validateApprovalRecord(rec, true)
	if len(problems) > 0 {
		t.Errorf("%s: approval record invalid: %v", r.ID, problems)
	}
	if rec.DecisionID != r.ID {
		t.Errorf("%s: approval record is for %s", r.ID, rec.DecisionID)
	}
	if err := statusMatchesRecord(r.Status, rec, level); err != nil {
		t.Errorf("%s: %v", r.ID, err)
	}
}

// statusMatchesRecord ties each recorded state to the kind of record that can justify it:
// a review notice for UNDER_HUMAN_REVIEW, a deferral for DEFERRED, a decision for the rest, and for a
// *_VERIFIED state a consistent human verification of a decision whose outcome is that state.
func statusMatchesRecord(status string, rec approvalRecord, level string) error {
	switch status {
	case "UNDER_HUMAN_REVIEW":
		if level != levelNotice {
			return fmt.Errorf("%s needs a valid REVIEW_NOTICE record (level %s)", status, level)
		}
	case "DEFERRED":
		if level != levelDeferred {
			return fmt.Errorf("%s needs a valid DEFERRAL record (level %s)", status, level)
		}
	case "DECISION_RECORDED_PENDING_VERIFICATION":
		if level != levelRecorded && level != levelHumanVerified {
			return fmt.Errorf("%s needs a valid DECISION record (level %s)", status, level)
		}
	case "APPROVED_VERIFIED", "REJECTED_VERIFIED", "SUPERSEDED_VERIFIED", "VOID_VERIFIED":
		if level != levelHumanVerified {
			return fmt.Errorf("%s without a consistent human verification entry (level %s)", status, level)
		}
		if want := strings.TrimSuffix(status, "_VERIFIED"); rec.Outcome != want {
			return fmt.Errorf("%s does not match the recorded outcome %q", status, rec.Outcome)
		}
	default:
		return fmt.Errorf("status %s does not take an approval record", status)
	}
	return nil
}

// Conflict statuses are copied from R5 unchanged; C-03 stays unresolved.
func TestR6ConflictsPreserved(t *testing.T) {
	tr := loadTracker(t)
	reg := map[string]string{}
	for _, l := range strings.Split(read(t, "docs/remediation/r5-decision-conflict-register.md"), "\n") {
		if strings.HasPrefix(l, "| C-") {
			c := strings.Split(strings.Trim(strings.TrimSpace(l), "|"), " | ")
			reg[strings.TrimSpace(c[0])] = strings.TrimSpace(c[len(c)-1])
		}
	}
	if len(tr.Conflicts) != 12 || len(reg) != 12 {
		t.Fatalf("conflicts: tracker %d, register %d, want 12", len(tr.Conflicts), len(reg))
	}
	for _, c := range tr.Conflicts {
		if reg[c.ID] != c.R5Status {
			t.Errorf("%s: tracker status %q differs from the R5 register %q", c.ID, c.R5Status, reg[c.ID])
		}
	}
	if !strings.Contains(reg["C-03"], "UNRESOLVED") {
		t.Errorf("C-03 must remain unresolved, got %q", reg["C-03"])
	}
}

// P0 / P1 classification follows the R5 inventory blocker column.
func TestR6BlockersMatchInventory(t *testing.T) {
	inv := map[string]string{}
	for _, l := range strings.Split(read(t, "docs/remediation/r5-decision-inventory.md"), "\n") {
		cols := strings.Split(l, " | ")
		if strings.HasPrefix(l, "| ") && len(cols) >= 17 {
			inv[strings.TrimPrefix(cols[0], "| ")] = cols[12]
		}
	}
	for _, r := range loadTracker(t).Records {
		b, ok := inv[r.ID]
		if !ok {
			t.Errorf("%s has no inventory row", r.ID)
			continue
		}
		if b != r.Blocker {
			t.Errorf("%s: blocker %q differs from the R5 inventory %q", r.ID, r.Blocker, b)
		}
		want := "P1"
		if strings.HasPrefix(r.Blocker, "P0") {
			want = "P0"
		} else if strings.HasPrefix(r.Blocker, "P2") {
			want = "P2"
		}
		if r.Priority != want {
			t.Errorf("%s: priority %s, blocker %s", r.ID, r.Priority, r.Blocker)
		}
	}
}

var outcomeLabels = []string{"گزینهٔ انتخاب‌شده", "متن کامل تصمیم", "نام و نقش تأییدکنندگان", "مدرک اختیار تأییدکنندگان",
	"تاریخ تصمیم", "تاریخ اجرا", "شرایط", "مرجع مدرک تأیید", "وضعیت راستی‌آزمایی"}

// Persian worksheets: every decision has a DRAFT worksheet in its meeting document, no option is ticked
// and no outcome field is filled.
func TestR6PersianWorksheets(t *testing.T) {
	tr := loadTracker(t)
	docs := map[string]string{}
	for _, f := range []string{"r6-m1-governance-data-session-fa.md", "r6-m2-security-session-fa.md", "r6-m3-sales-session-fa.md",
		"r6-m4-finance-sessions-fa.md", "r6-m5-m6-operations-biawin-reporting-fa.md", "r6-management-decision-tracker.md",
		"r6-approval-form-fa.md", "r6-followup-action-register-fa.md", "r6-unresolved-evidence-conflicts-fa.md",
		"r6-management-executive-summary-fa.md"} {
		docs[f] = read(t, "docs/remediation/"+f)
	}
	for f, d := range docs {
		if strings.ContainsAny(d, "☑☒✓✔") || strings.Contains(d, "[x]") {
			t.Errorf("%s: a choice is pre-selected", f)
		}
		for _, l := range strings.Split(d, "\n") {
			for _, lab := range outcomeLabels {
				if strings.HasPrefix(l, "| "+lab+" |") && strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(l, "| "+lab+" |"), "|")) != "" {
					t.Errorf("%s: outcome field %q is filled: %q", f, lab, l)
				}
			}
		}
	}
	for _, r := range tr.Records {
		doc := docs[meetingDoc[r.Meeting]]
		if r.R5Package == "P-GOV" {
			if !strings.Contains(doc, "| "+r.ID+" | EVIDENCE NOT RECOVERED — HUMAN REVIEW REQUIRED | ☐ | ☐ جایگزین: ____ | ☐ |  |  |") {
				t.Errorf("%s: P-GOV row missing or pre-filled in M1", r.ID)
			}
			continue
		}
		blocks := strings.Split(doc, "\n### "+r.ID+" — ")
		if len(blocks) != 2 {
			t.Errorf("%s: expected exactly one worksheet in %s, found %d", r.ID, meetingDoc[r.Meeting], len(blocks)-1)
			continue
		}
		ws := strings.SplitN(blocks[1], "\n### ", 2)[0]
		if !strings.Contains(ws, "**DRAFT — NOT APPROVED / پیش‌نویس — تأییدنشده**") {
			t.Errorf("%s: worksheet lacks the DRAFT marker", r.ID)
		}
		if !strings.Contains(docs["r6-management-decision-tracker.md"], "| "+r.ID+" | ") {
			t.Errorf("%s: missing from the decision tracker table", r.ID)
		}
		if base := strings.SplitN(r.ID, " (", 2)[0]; r.Status == "READY_FOR_REVIEW" && !strings.Contains(docs["r6-management-executive-summary-fa.md"], base) {
			t.Errorf("%s: review-ready but absent from the executive summary", r.ID)
		}
	}
}

// ---------------- approval record validator (structural only)

const (
	levelIncomplete    = "STRUCTURALLY_INCOMPLETE"
	levelNotice        = "REVIEW_NOTICE_RECORDED"
	levelDeferred      = "DEFERRAL_RECORDED"
	levelRecorded      = "DECISION_RECORDED_PENDING_VERIFICATION"
	levelHumanVerified = "HUMAN_VERIFICATION_ENTRY_CONSISTENT"
)

type person struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type approvalRecord struct {
	RecordType      string `json:"record_type"`
	DecisionID      string `json:"decision_id"`
	DecisionVersion string `json:"decision_version"`
	Outcome         string `json:"outcome"`
	SelectedOption  string `json:"selected_option"`
	DecisionText    string `json:"decision_text"`
	Approvers       []struct {
		Name              string `json:"name"`
		Role              string `json:"role"`
		AuthorityEvidence struct {
			Type      string `json:"type"`
			Reference string `json:"reference"`
		} `json:"authority_evidence"`
	} `json:"approvers"`
	IndependentApprovals []string `json:"independent_approvals"`
	DecisionDate         string   `json:"decision_date"`
	EffectiveDate        string   `json:"effective_date"`
	EvidenceSource       struct {
		Type      string `json:"type"`
		Reference string `json:"reference"`
	} `json:"evidence_source"`
	ImmutableRef      string   `json:"immutable_ref"`
	Supersedes        []string `json:"supersedes"`
	Conditions        string   `json:"conditions"`
	DeferralReason    string   `json:"deferral_reason"`
	RevisitCondition  string   `json:"revisit_condition"`
	RecordedBy        person   `json:"recorded_by"`
	HumanVerification *struct {
		VerifierName string `json:"verifier_name"`
		VerifierRole string `json:"verifier_role"`
		Method       string `json:"method"`
		Date         string `json:"date"`
		Result       string `json:"result"`
	} `json:"human_verification"`
}

func loadApproval(path string) (approvalRecord, error) {
	var r approvalRecord
	b, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(b, &r)
}

func requireFields(m map[string]string) []string {
	var p []string
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			p = append(p, "missing "+k)
		}
	}
	return p
}

// commonChecks: identity of the record, evidence, recorder, approvers with authority evidence, and no
// placeholder content in a committed record.
func commonChecks(r approvalRecord, inRepository bool) ([]string, map[string]bool) {
	p := requireFields(map[string]string{"decision_id": r.DecisionID, "decision_version": r.DecisionVersion,
		"decision_date": r.DecisionDate, "evidence_source.type": r.EvidenceSource.Type,
		"evidence_source.reference": r.EvidenceSource.Reference, "immutable_ref": r.ImmutableRef,
		"recorded_by.name": r.RecordedBy.Name, "recorded_by.role": r.RecordedBy.Role})
	if _, err := time.Parse("2006-01-02", r.DecisionDate); err != nil {
		p = append(p, "decision_date must be ISO YYYY-MM-DD")
	}
	if len(r.Approvers) == 0 {
		p = append(p, "no approver")
	}
	names, roles := map[string]bool{}, map[string]bool{}
	for i, a := range r.Approvers {
		if a.Name == "" || a.Role == "" || a.AuthorityEvidence.Type == "" || a.AuthorityEvidence.Reference == "" {
			p = append(p, fmt.Sprintf("approver %d lacks name, role or authority evidence", i))
		}
		names[strings.ToLower(a.Name)] = true
		roles[a.Role] = true
	}
	for _, role := range r.IndependentApprovals {
		if !roles[role] {
			p = append(p, "independent approval missing for role "+role)
		}
	}
	if inRepository {
		b, _ := json.Marshal(r)
		for _, marker := range []string{"SYNTHETIC", "PLACEHOLDER", "TODO", "EXAMPLE"} {
			if strings.Contains(strings.ToUpper(string(b)), marker) {
				p = append(p, "contains a synthetic / placeholder marker: "+marker)
			}
		}
	}
	return p, names
}

// validateApprovalRecord checks structure only. It never authenticates a person, a signature or an
// authority document; the best it can return is HUMAN_VERIFICATION_ENTRY_CONSISTENT, meaning that a
// human verification entry exists and is consistent with the record.
//
// record_type selects the shape:
//   - REVIEW_NOTICE (T3): who opened the review, with authority evidence; carries no decision.
//   - DEFERRAL (T6): outcome DEFERRED with a reason and a revisit condition; carries no selected option.
//   - DECISION (T4/T5): a decided outcome, optionally with a human verification entry.
func validateApprovalRecord(r approvalRecord, inRepository bool) (string, []string) {
	p, names := commonChecks(r, inRepository)
	switch r.RecordType {
	case "REVIEW_NOTICE":
		if r.Outcome != "" || r.SelectedOption != "" || r.DecisionText != "" || r.HumanVerification != nil {
			p = append(p, "a review notice carries no outcome, option, decision text or verification")
		}
		if len(p) > 0 {
			return levelIncomplete, p
		}
		return levelNotice, nil
	case "DEFERRAL":
		p = append(p, requireFields(map[string]string{"deferral_reason": r.DeferralReason, "revisit_condition": r.RevisitCondition})...)
		if r.Outcome != "DEFERRED" {
			p = append(p, "a deferral has outcome DEFERRED")
		}
		if r.SelectedOption != "" {
			p = append(p, "a deferral selects no option")
		}
		if len(p) > 0 {
			return levelIncomplete, p
		}
		return levelDeferred, nil
	case "DECISION":
	default:
		return levelIncomplete, append(p, "unknown record_type "+r.RecordType)
	}
	p = append(p, requireFields(map[string]string{"outcome": r.Outcome, "selected_option": r.SelectedOption,
		"decision_text": r.DecisionText, "effective_date": r.EffectiveDate, "conditions": r.Conditions})...)
	switch r.Outcome {
	case "APPROVED", "REJECTED", "VOID":
	case "SUPERSEDED":
		if len(r.Supersedes) == 0 {
			p = append(p, "SUPERSEDED without supersedes")
		}
	default:
		p = append(p, "unknown outcome "+r.Outcome)
	}
	dd, err1 := time.Parse("2006-01-02", r.DecisionDate)
	ed, err2 := time.Parse("2006-01-02", r.EffectiveDate)
	if err1 != nil || err2 != nil {
		p = append(p, "dates must be ISO YYYY-MM-DD")
	} else if ed.Before(dd) {
		p = append(p, "effective date before decision date")
	}
	if len(p) > 0 {
		return levelIncomplete, p
	}
	h := r.HumanVerification
	if h == nil {
		return levelRecorded, nil
	}
	if h.VerifierName == "" || h.VerifierRole == "" || h.Method == "" {
		p = append(p, "human verification lacks verifier or method")
	}
	if _, err := time.Parse("2006-01-02", h.Date); err != nil {
		p = append(p, "human verification date invalid")
	}
	v := strings.ToLower(h.VerifierName)
	if names[v] || v == strings.ToLower(r.RecordedBy.Name) {
		p = append(p, "verifier must not be an approver or the recorder")
	}
	for _, bot := range []string{"bot", "automation", "ci", "claude", "codex", "github-actions"} {
		if v == bot || strings.Contains(v, "[bot]") {
			p = append(p, "verifier must be a human, not automation")
		}
	}
	if h.Result != "VERIFIED" && h.Result != "REJECTED" && h.Result != "INCONCLUSIVE" {
		p = append(p, "unknown verification result "+h.Result)
	}
	if len(p) > 0 {
		return levelIncomplete, p
	}
	if h.Result != "VERIFIED" {
		return levelRecorded, nil
	}
	return levelHumanVerified, nil
}

// Regression: synthetic fixtures (built here, never written to approvals/) show each structural defect
// is rejected and that nothing reaches a verified level without a human verification entry.
func TestApprovalValidatorRules(t *testing.T) {
	base := func() approvalRecord {
		var r approvalRecord
		raw := `{"record_type":"DECISION","decision_id":"D-45","decision_version":"1.0","outcome":"APPROVED","selected_option":"1",
		"decision_text":"fixture text","approvers":[{"name":"Fixture Approver One","role":"Security owner",
		"authority_evidence":{"type":"appointment letter","reference":"fixture-ref-1"}},{"name":"Fixture Approver Two",
		"role":"Product owner","authority_evidence":{"type":"delegation","reference":"fixture-ref-2"}}],
		"independent_approvals":["Security owner","Product owner"],"decision_date":"2030-01-02","effective_date":"2030-01-03",
		"evidence_source":{"type":"signed minutes","reference":"fixture-minutes"},"immutable_ref":"0000000",
		"supersedes":[],"conditions":"none","recorded_by":{"name":"Fixture Recorder","role":"Governance secretary"},
		"human_verification":null}`
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	if lvl, p := validateApprovalRecord(base(), false); lvl != levelRecorded || len(p) > 0 {
		t.Fatalf("complete record without verification must be %s, got %s %v", levelRecorded, lvl, p)
	}
	verified := func(name, result string) approvalRecord {
		r := base()
		raw := fmt.Sprintf(`{"verifier_name":%q,"verifier_role":"Internal audit","method":"independent channel","date":"2030-01-04","result":%q}`, name, result)
		if err := json.Unmarshal([]byte(`{"human_verification":`+raw+`}`), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	if lvl, _ := validateApprovalRecord(verified("Fixture Verifier", "VERIFIED"), false); lvl != levelHumanVerified {
		t.Fatalf("consistent human verification must reach %s, got %s", levelHumanVerified, lvl)
	}
	if lvl, _ := validateApprovalRecord(verified("Fixture Verifier", "INCONCLUSIVE"), false); lvl != levelRecorded {
		t.Errorf("INCONCLUSIVE verification must stay %s, got %s", levelRecorded, lvl)
	}
	bad := map[string]func(*approvalRecord){
		"no decision text":            func(r *approvalRecord) { r.DecisionText = "" },
		"no approver":                 func(r *approvalRecord) { r.Approvers = nil },
		"no authority evidence":       func(r *approvalRecord) { r.Approvers[0].AuthorityEvidence.Reference = "" },
		"missing independent role":    func(r *approvalRecord) { r.IndependentApprovals = append(r.IndependentApprovals, "Legal owner") },
		"bad date":                    func(r *approvalRecord) { r.DecisionDate = "next week" },
		"effective before decision":   func(r *approvalRecord) { r.EffectiveDate = "2029-12-31" },
		"unknown outcome":             func(r *approvalRecord) { r.Outcome = "APPROVED_VERIFIED" },
		"superseded without target":   func(r *approvalRecord) { r.Outcome = "SUPERSEDED" },
		"no immutable reference":      func(r *approvalRecord) { r.ImmutableRef = "" },
		"no conditions statement":     func(r *approvalRecord) { r.Conditions = "" },
		"no recorder":                 func(r *approvalRecord) { r.RecordedBy = person{} },
		"verifier is an approver":     func(r *approvalRecord) { *r = verified("Fixture Approver One", "VERIFIED") },
		"verifier is the recorder":    func(r *approvalRecord) { *r = verified("Fixture Recorder", "VERIFIED") },
		"verifier is automation":      func(r *approvalRecord) { *r = verified("github-actions[bot]", "VERIFIED") },
		"unknown verification result": func(r *approvalRecord) { *r = verified("Fixture Verifier", "AUTO_APPROVED") },
		"unknown record type":         func(r *approvalRecord) { r.RecordType = "" },
	}
	for name, mutate := range bad {
		r := base()
		mutate(&r)
		if lvl, p := validateApprovalRecord(r, false); lvl != levelIncomplete || len(p) == 0 {
			t.Errorf("%s: accepted (%s)", name, lvl)
		}
	}
	// A placeholder record committed to approvals/ is rejected even when otherwise complete.
	r := base()
	r.DecisionText = "PLACEHOLDER decision"
	if lvl, _ := validateApprovalRecord(r, true); lvl != levelIncomplete {
		t.Errorf("placeholder record in the repository accepted (%s)", lvl)
	}
}

// Regression (review findings on PR #46): a verified status must match the recorded outcome; a review
// notice and a deferral have their own record shapes and cannot stand in for a decision.
func TestRecordedStatusMatchesRecord(t *testing.T) {
	decision := func(outcome, result string) approvalRecord {
		var r approvalRecord
		raw := fmt.Sprintf(`{"record_type":"DECISION","decision_id":"OPD-04","decision_version":"1.0","outcome":%q,
		"selected_option":"1","decision_text":"fixture","approvers":[{"name":"Fixture Approver","role":"Finance owner",
		"authority_evidence":{"type":"delegation","reference":"fixture-ref"}}],"decision_date":"2030-02-01",
		"effective_date":"2030-02-01","evidence_source":{"type":"signed minutes","reference":"fixture-minutes"},
		"immutable_ref":"0000000","conditions":"none","recorded_by":{"name":"Fixture Recorder","role":"Secretary"},
		"human_verification":{"verifier_name":"Fixture Verifier","verifier_role":"Internal audit","method":"independent channel",
		"date":"2030-02-02","result":%q}}`, outcome, result)
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	check := func(name, status string, r approvalRecord, wantOK bool) {
		lvl, _ := validateApprovalRecord(r, false)
		err := statusMatchesRecord(status, r, lvl)
		if (err == nil) != wantOK {
			t.Errorf("%s: status %s, level %s, err %v, want ok=%v", name, status, lvl, err, wantOK)
		}
	}
	check("approved verified", "APPROVED_VERIFIED", decision("APPROVED", "VERIFIED"), true)
	check("rejected record shown as approved", "APPROVED_VERIFIED", decision("REJECTED", "VERIFIED"), false)
	check("rejected verified", "REJECTED_VERIFIED", decision("REJECTED", "VERIFIED"), true)
	check("void shown as superseded", "SUPERSEDED_VERIFIED", decision("VOID", "VERIFIED"), false)
	check("inconclusive verification", "APPROVED_VERIFIED", decision("APPROVED", "INCONCLUSIVE"), false)
	check("pending verification", "DECISION_RECORDED_PENDING_VERIFICATION", decision("APPROVED", "INCONCLUSIVE"), true)
	check("decision as review notice", "UNDER_HUMAN_REVIEW", decision("APPROVED", "VERIFIED"), false)

	var notice approvalRecord
	if err := json.Unmarshal([]byte(`{"record_type":"REVIEW_NOTICE","decision_id":"OPD-04","decision_version":"1.0",
		"approvers":[{"name":"Fixture Approver","role":"Finance owner","authority_evidence":{"type":"delegation","reference":"fixture-ref"}}],
		"decision_date":"2030-01-15","evidence_source":{"type":"dated message","reference":"fixture-notice"},"immutable_ref":"0000000",
		"recorded_by":{"name":"Fixture Recorder","role":"Secretary"}}`), &notice); err != nil {
		t.Fatal(err)
	}
	check("review notice", "UNDER_HUMAN_REVIEW", notice, true)
	check("review notice as decision", "DECISION_RECORDED_PENDING_VERIFICATION", notice, false)
	withOutcome := notice
	withOutcome.Outcome = "APPROVED"
	check("review notice carrying an outcome", "UNDER_HUMAN_REVIEW", withOutcome, false)

	deferral := notice
	deferral.RecordType, deferral.Outcome = "DEFERRAL", "DEFERRED"
	deferral.DeferralReason, deferral.RevisitCondition = "fixture reason", "after fixture evidence"
	check("deferral", "DEFERRED", deferral, true)
	check("deferral as approval", "APPROVED_VERIFIED", deferral, false)
	noRevisit := deferral
	noRevisit.RevisitCondition = ""
	check("deferral without revisit condition", "DEFERRED", noRevisit, false)
	withOption := deferral
	withOption.SelectedOption = "1"
	check("deferral selecting an option", "DEFERRED", withOption, false)
}
