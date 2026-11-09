package governance

// R6 management decision workflow checks (offline). They keep the R6 tracker consistent with the R5
// records and make it impossible to record an approval in the repository without an approval record
// that passes the structural check and, for a verified state, a human verification entry.
//
// Limits: these checks cannot authenticate a person, a signature or an authority document. That is a
// human verification step (r6-approval-evidence-standard.md).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
//
// Three bounded record shapes, selected by record_type:
//   - REVIEW_NOTICE (T3): opens a review. Initiator with authority evidence, notice date, notice reference.
//     Carries no decision of any kind and can only justify UNDER_HUMAN_REVIEW.
//   - DEFERRAL (T6): formal postponement. Initiator with authority evidence, deferral date, reason and an
//     explicit revisit condition (a revisit date only if the condition itself provides one). Carries no
//     option or decision and can only justify DEFERRED, which is never a closed policy.
//   - DECISION (T4/T5): a decided outcome with approvers and their authority evidence; with a consistent
//     human verification entry it can justify exactly the *_VERIFIED state equal to its outcome.
// None of these checks authenticates a person, a signature or an authority document.

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

type authorizedPerson struct {
	Name              string `json:"name"`
	Role              string `json:"role"`
	AuthorityEvidence struct {
		Type      string `json:"type"`
		Reference string `json:"reference"`
	} `json:"authority_evidence"`
}

func (a authorizedPerson) complete() bool {
	return a.Name != "" && a.Role != "" && a.AuthorityEvidence.Type != "" && a.AuthorityEvidence.Reference != ""
}

type approvalRecord struct {
	RecordType           string             `json:"record_type"`
	DecisionID           string             `json:"decision_id"`
	DecisionVersion      string             `json:"decision_version"`
	Outcome              string             `json:"outcome"`
	SelectedOption       string             `json:"selected_option"`
	DecisionText         string             `json:"decision_text"`
	Initiator            *authorizedPerson  `json:"initiator"`
	Approvers            []authorizedPerson `json:"approvers"`
	IndependentApprovals []string           `json:"independent_approvals"`
	NoticeDate           string             `json:"notice_date"`
	DeferralDate         string             `json:"deferral_date"`
	DecisionDate         string             `json:"decision_date"`
	EffectiveDate        string             `json:"effective_date"`
	EvidenceSource       struct {
		Type      string `json:"type"`
		Reference string `json:"reference"`
	} `json:"evidence_source"`
	ImmutableRef      string   `json:"immutable_ref"`
	Supersedes        []string `json:"supersedes"`
	Conditions        string   `json:"conditions"`
	DeferralReason    string   `json:"deferral_reason"`
	RevisitCondition  string   `json:"revisit_condition"`
	RevisitDate       string   `json:"revisit_date"`
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

func forbidFields(kind string, m map[string]bool) []string {
	var p []string
	for k, present := range m {
		if present {
			p = append(p, kind+" must not carry "+k)
		}
	}
	return p
}

func isoDate(name, v string) []string {
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return []string{name + " must be an ISO date YYYY-MM-DD"}
	}
	return nil
}

// recordFileName is the only accepted file name under docs/governance/approvals/. Distinct record types and
// versions get distinct files, so a new record never overwrites an earlier one.
func recordFileName(r approvalRecord) string {
	id := regexp.MustCompile(`[^A-Za-z0-9-]+`).ReplaceAllString(r.DecisionID, "_")
	return fmt.Sprintf("%s-v%s-%s.json", strings.Trim(id, "_"), r.DecisionVersion, r.RecordType)
}

func validateApprovalRecord(r approvalRecord, inRepository bool) (string, []string) {
	p := requireFields(map[string]string{"decision_id": r.DecisionID, "decision_version": r.DecisionVersion,
		"evidence_source.type": r.EvidenceSource.Type, "evidence_source.reference": r.EvidenceSource.Reference,
		"immutable_ref": r.ImmutableRef, "recorded_by.name": r.RecordedBy.Name, "recorded_by.role": r.RecordedBy.Role})
	if inRepository {
		b, _ := json.Marshal(r)
		for _, marker := range []string{"SYNTHETIC", "PLACEHOLDER", "TODO", "EXAMPLE", "FIXTURE"} {
			if strings.Contains(strings.ToUpper(string(b)), marker) {
				p = append(p, "contains a synthetic / placeholder marker: "+marker)
			}
		}
	}
	noDecision := map[string]bool{"outcome": r.Outcome != "", "selected_option": r.SelectedOption != "",
		"decision_text": r.DecisionText != "", "decision_date": r.DecisionDate != "", "effective_date": r.EffectiveDate != "",
		"approvers": len(r.Approvers) > 0, "independent_approvals": len(r.IndependentApprovals) > 0,
		"supersedes": len(r.Supersedes) > 0, "conditions": r.Conditions != "", "human_verification": r.HumanVerification != nil}
	switch r.RecordType {
	case "REVIEW_NOTICE":
		if r.Initiator == nil || !r.Initiator.complete() {
			p = append(p, "initiator needs name, role and authority evidence")
		}
		p = append(p, isoDate("notice_date", r.NoticeDate)...)
		p = append(p, forbidFields("a review notice", noDecision)...)
		p = append(p, forbidFields("a review notice", map[string]bool{"deferral_date": r.DeferralDate != "",
			"deferral_reason": r.DeferralReason != "", "revisit_condition": r.RevisitCondition != "", "revisit_date": r.RevisitDate != ""})...)
		if len(p) > 0 {
			return levelIncomplete, p
		}
		return levelNotice, nil
	case "DEFERRAL":
		if r.Initiator == nil || !r.Initiator.complete() {
			p = append(p, "initiator needs name, role and authority evidence")
		}
		p = append(p, isoDate("deferral_date", r.DeferralDate)...)
		p = append(p, requireFields(map[string]string{"deferral_reason": r.DeferralReason, "revisit_condition": r.RevisitCondition})...)
		if r.RevisitDate != "" {
			p = append(p, isoDate("revisit_date", r.RevisitDate)...)
		}
		delete(noDecision, "outcome")
		if r.Outcome != "DEFERRED" {
			p = append(p, "a deferral has outcome DEFERRED")
		}
		p = append(p, forbidFields("a deferral", noDecision)...)
		p = append(p, forbidFields("a deferral", map[string]bool{"notice_date": r.NoticeDate != ""})...)
		if len(p) > 0 {
			return levelIncomplete, p
		}
		return levelDeferred, nil
	case "DECISION":
	default:
		return levelIncomplete, append(p, "unknown record_type "+r.RecordType)
	}
	p = append(p, requireFields(map[string]string{"outcome": r.Outcome, "selected_option": r.SelectedOption,
		"decision_text": r.DecisionText, "conditions": r.Conditions})...)
	p = append(p, forbidFields("a decision", map[string]bool{"initiator": r.Initiator != nil, "notice_date": r.NoticeDate != "",
		"deferral_date": r.DeferralDate != "", "deferral_reason": r.DeferralReason != "",
		"revisit_condition": r.RevisitCondition != "", "revisit_date": r.RevisitDate != ""})...)
	switch r.Outcome {
	case "APPROVED", "REJECTED", "VOID":
	case "SUPERSEDED":
		if len(r.Supersedes) == 0 {
			p = append(p, "SUPERSEDED without supersedes")
		}
	default:
		p = append(p, "unknown decision outcome "+r.Outcome)
	}
	dd, err1 := time.Parse("2006-01-02", r.DecisionDate)
	ed, err2 := time.Parse("2006-01-02", r.EffectiveDate)
	if err1 != nil || err2 != nil {
		p = append(p, "decision_date and effective_date must be ISO dates")
	} else if ed.Before(dd) {
		p = append(p, "effective date before decision date")
	}
	if len(r.Approvers) == 0 {
		p = append(p, "no approver")
	}
	names, roles := map[string]bool{}, map[string]bool{}
	for i, a := range r.Approvers {
		if !a.complete() {
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
	p = append(p, isoDate("human_verification.date", h.Date)...)
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

// closedPolicy: the only states that end a decision. DEFERRED, UNDER_HUMAN_REVIEW and
// DECISION_RECORDED_PENDING_VERIFICATION are never closed, and only APPROVED_VERIFIED (or a verified
// supersession / voiding that needs replacement work) can start engineering follow-up.
var closedPolicy = map[string]bool{"APPROVED_VERIFIED": true, "REJECTED_VERIFIED": true, "SUPERSEDED_VERIFIED": true, "VOID_VERIFIED": true}
var followUpEligible = map[string]bool{"APPROVED_VERIFIED": true, "SUPERSEDED_VERIFIED": true, "VOID_VERIFIED": true}

// appendOnlyChange: a committed record may later gain a human verification entry; nothing else may change.
func appendOnlyChange(old, new approvalRecord) error {
	o, n := old, new
	if o.HumanVerification != nil && (n.HumanVerification == nil || *n.HumanVerification != *o.HumanVerification) {
		return errors.New("an existing human verification entry was changed or removed")
	}
	o.HumanVerification, n.HumanVerification = nil, nil
	ob, _ := json.Marshal(o)
	nb, _ := json.Marshal(n)
	if string(ob) != string(nb) {
		return errors.New("a committed record was rewritten; corrections need a new version file")
	}
	return nil
}

// ---------------- fixtures (synthetic, built in test code, never written to approvals/)

func fixture(t *testing.T, raw string) approvalRecord {
	t.Helper()
	var r approvalRecord
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

const fxCommon = `"decision_id":"OPD-04","decision_version":"1.0","evidence_source":{"type":"signed minutes","reference":"fixture-ref"},
	"immutable_ref":"0000000","recorded_by":{"name":"Fixture Recorder","role":"Secretary"}`
const fxAuth = `{"name":"Fixture Approver","role":"Finance owner","authority_evidence":{"type":"delegation","reference":"fixture-auth"}}`

func fxDecision(t *testing.T, outcome, verification string) approvalRecord {
	hv := "null"
	if verification != "" {
		hv = fmt.Sprintf(`{"verifier_name":"Fixture Verifier","verifier_role":"Internal audit","method":"independent channel","date":"2030-02-02","result":%q}`, verification)
	}
	sup := "[]"
	if outcome == "SUPERSEDED" {
		sup = `["OPD-04 v0.1"]`
	}
	return fixture(t, fmt.Sprintf(`{"record_type":"DECISION",%s,"outcome":%q,"selected_option":"1","decision_text":"fixture",
	"approvers":[%s],"independent_approvals":["Finance owner"],"decision_date":"2030-02-01","effective_date":"2030-02-01",
	"supersedes":%s,"conditions":"none","human_verification":%s}`, fxCommon, outcome, fxAuth, sup, hv))
}

func fxNotice(t *testing.T) approvalRecord {
	return fixture(t, fmt.Sprintf(`{"record_type":"REVIEW_NOTICE",%s,"initiator":%s,"notice_date":"2030-01-15"}`, fxCommon, fxAuth))
}

func fxDeferral(t *testing.T) approvalRecord {
	return fixture(t, fmt.Sprintf(`{"record_type":"DEFERRAL",%s,"outcome":"DEFERRED","initiator":%s,"deferral_date":"2030-01-20",
	"deferral_reason":"fixture reason","revisit_condition":"when the fixture evidence report exists"}`, fxCommon, fxAuth))
}

// ---------------- regression tests

// P2-01: every outcome × verified-state pair; only the matching pair with a VERIFIED human entry passes.
func TestVerifiedStatusOutcomeMatrix(t *testing.T) {
	for _, outcome := range []string{"APPROVED", "REJECTED", "SUPERSEDED", "VOID"} {
		for _, result := range []string{"VERIFIED", "REJECTED", "INCONCLUSIVE", ""} {
			rec := fxDecision(t, outcome, result)
			lvl, p := validateApprovalRecord(rec, false)
			if len(p) > 0 {
				t.Fatalf("%s/%s fixture invalid: %v", outcome, result, p)
			}
			// Every one of the ten states: a decision justifies only "pending" and, with a VERIFIED
			// human entry, the single *_VERIFIED state equal to its outcome.
			for _, st := range lifecycle {
				ok := statusMatchesRecord(st, rec, lvl) == nil
				want := st == "DECISION_RECORDED_PENDING_VERIFICATION" || (result == "VERIFIED" && st == outcome+"_VERIFIED")
				if ok != want {
					t.Errorf("outcome %s, verification %q, status %s: accepted=%v, want %v", outcome, result, st, ok, want)
				}
			}
		}
	}
}

// P2-02: a review notice is bounded, needs initiator authority, and can never justify a decided state.
func TestReviewNoticeRecord(t *testing.T) {
	n := fxNotice(t)
	lvl, p := validateApprovalRecord(n, false)
	if lvl != levelNotice || len(p) > 0 {
		t.Fatalf("valid notice: %s %v", lvl, p)
	}
	for _, st := range lifecycle {
		if ok := statusMatchesRecord(st, n, lvl) == nil; ok != (st == "UNDER_HUMAN_REVIEW") {
			t.Errorf("notice justifies %s: %v", st, ok)
		}
	}
	bad := map[string]func(*approvalRecord){
		"no initiator":             func(r *approvalRecord) { r.Initiator = nil },
		"no initiator authority":   func(r *approvalRecord) { r.Initiator.AuthorityEvidence.Reference = "" },
		"no notice date":           func(r *approvalRecord) { r.NoticeDate = "" },
		"no notice reference":      func(r *approvalRecord) { r.EvidenceSource.Reference = "" },
		"no version":               func(r *approvalRecord) { r.DecisionVersion = "" },
		"no audit reference":       func(r *approvalRecord) { r.ImmutableRef = "" },
		"carries an outcome":       func(r *approvalRecord) { r.Outcome = "APPROVED" },
		"carries an option":        func(r *approvalRecord) { r.SelectedOption = "1" },
		"carries decision text":    func(r *approvalRecord) { r.DecisionText = "approved" },
		"carries approvers":        func(r *approvalRecord) { r.Approvers = []authorizedPerson{*r.Initiator} },
		"carries a decision date":  func(r *approvalRecord) { r.DecisionDate = "2030-01-15" },
		"carries a verification":   func(r *approvalRecord) { *r = fxNoticeWithVerification(t) },
		"carries deferral content": func(r *approvalRecord) { r.RevisitCondition = "later" },
	}
	for name, mutate := range bad {
		r := fxNotice(t)
		mutate(&r)
		if lvl, _ := validateApprovalRecord(r, false); lvl != levelIncomplete {
			t.Errorf("notice %s: accepted as %s", name, lvl)
		}
	}
}

func fxNoticeWithVerification(t *testing.T) approvalRecord {
	return fixture(t, fmt.Sprintf(`{"record_type":"REVIEW_NOTICE",%s,"initiator":%s,"notice_date":"2030-01-15",
	"human_verification":{"verifier_name":"Fixture Verifier","verifier_role":"Audit","method":"m","date":"2030-01-16","result":"VERIFIED"}}`, fxCommon, fxAuth))
}

// P2-03: a formal deferral is distinct from "no decision yet" and from a final decision, and is never closed.
func TestDeferralRecord(t *testing.T) {
	d := fxDeferral(t)
	lvl, p := validateApprovalRecord(d, false)
	if lvl != levelDeferred || len(p) > 0 {
		t.Fatalf("valid deferral: %s %v", lvl, p)
	}
	for _, st := range lifecycle {
		if ok := statusMatchesRecord(st, d, lvl) == nil; ok != (st == "DEFERRED") {
			t.Errorf("deferral justifies %s: %v", st, ok)
		}
	}
	withDate := fxDeferral(t)
	withDate.RevisitDate = "2030-06-01"
	if lvl, p := validateApprovalRecord(withDate, false); lvl != levelDeferred {
		t.Errorf("deferral with a revisit date from its condition rejected: %v", p)
	}
	bad := map[string]func(*approvalRecord){
		"no initiator":           func(r *approvalRecord) { r.Initiator = nil },
		"no initiator authority": func(r *approvalRecord) { r.Initiator.AuthorityEvidence.Type = "" },
		"no deferral date":       func(r *approvalRecord) { r.DeferralDate = "" },
		"no reason":              func(r *approvalRecord) { r.DeferralReason = "" },
		"no revisit condition":   func(r *approvalRecord) { r.RevisitCondition = "" },
		"bad revisit date":       func(r *approvalRecord) { r.RevisitDate = "soon" },
		"no evidence reference":  func(r *approvalRecord) { r.EvidenceSource.Reference = "" },
		"no audit reference":     func(r *approvalRecord) { r.ImmutableRef = "" },
		"wrong outcome":          func(r *approvalRecord) { r.Outcome = "APPROVED" },
		"selects an option":      func(r *approvalRecord) { r.SelectedOption = "1" },
		"carries decision text":  func(r *approvalRecord) { r.DecisionText = "approved" },
		"carries approvers":      func(r *approvalRecord) { r.Approvers = []authorizedPerson{*r.Initiator} },
	}
	for name, mutate := range bad {
		r := fxDeferral(t)
		mutate(&r)
		if lvl, _ := validateApprovalRecord(r, false); lvl != levelIncomplete {
			t.Errorf("deferral %s: accepted as %s", name, lvl)
		}
	}
	// "No decision yet" has no record at all; DEFERRED without a record is rejected by the tracker test.
	if closedPolicy["DEFERRED"] || followUpEligible["DEFERRED"] || closedPolicy["UNDER_HUMAN_REVIEW"] ||
		closedPolicy["DECISION_RECORDED_PENDING_VERIFICATION"] || followUpEligible["REJECTED_VERIFIED"] {
		t.Error("a deferral, an open review or an unverified decision must never count as a closed or actionable policy")
	}
	// A deferral can never be read as a decision either.
	asDecision := fxDeferral(t)
	asDecision.RecordType = "DECISION"
	if lvl, _ := validateApprovalRecord(asDecision, false); lvl != levelIncomplete {
		t.Errorf("deferral relabelled as DECISION accepted as %s", lvl)
	}
}

// Decision record structure (T4/T5) and human-verification guards.
func TestApprovalValidatorRules(t *testing.T) {
	if lvl, p := validateApprovalRecord(fxDecision(t, "APPROVED", ""), false); lvl != levelRecorded || len(p) > 0 {
		t.Fatalf("complete decision without verification must be %s, got %s %v", levelRecorded, lvl, p)
	}
	verified := func(name string) approvalRecord {
		r := fxDecision(t, "APPROVED", "VERIFIED")
		r.HumanVerification.VerifierName = name
		return r
	}
	bad := map[string]func(*approvalRecord){
		"no decision text":          func(r *approvalRecord) { r.DecisionText = "" },
		"no approver":               func(r *approvalRecord) { r.Approvers = nil },
		"no authority evidence":     func(r *approvalRecord) { r.Approvers[0].AuthorityEvidence.Reference = "" },
		"missing independent role":  func(r *approvalRecord) { r.IndependentApprovals = append(r.IndependentApprovals, "Legal owner") },
		"bad date":                  func(r *approvalRecord) { r.DecisionDate = "next week" },
		"effective before decision": func(r *approvalRecord) { r.EffectiveDate = "2029-12-31" },
		"unknown outcome":           func(r *approvalRecord) { r.Outcome = "APPROVED_VERIFIED" },
		"deferral outcome":          func(r *approvalRecord) { r.Outcome = "DEFERRED" },
		"superseded without target": func(r *approvalRecord) { r.Outcome, r.Supersedes = "SUPERSEDED", nil },
		"no immutable reference":    func(r *approvalRecord) { r.ImmutableRef = "" },
		"no conditions statement":   func(r *approvalRecord) { r.Conditions = "" },
		"no recorder":               func(r *approvalRecord) { r.RecordedBy = person{} },
		"unknown record type":       func(r *approvalRecord) { r.RecordType = "" },
		"carries notice fields":     func(r *approvalRecord) { r.NoticeDate = "2030-01-01" },
		"verifier is an approver":   func(r *approvalRecord) { *r = verified("Fixture Approver") },
		"verifier is the recorder":  func(r *approvalRecord) { *r = verified("Fixture Recorder") },
		"verifier is automation":    func(r *approvalRecord) { *r = verified("github-actions[bot]") },
		"unknown verification result": func(r *approvalRecord) {
			*r = verified("Fixture Verifier")
			r.HumanVerification.Result = "AUTO_APPROVED"
		},
	}
	for name, mutate := range bad {
		r := fxDecision(t, "APPROVED", "")
		mutate(&r)
		if lvl, p := validateApprovalRecord(r, false); lvl != levelIncomplete || len(p) == 0 {
			t.Errorf("%s: accepted (%s)", name, lvl)
		}
	}
	// Any fixture / placeholder content committed under approvals/ is rejected even when otherwise complete.
	for _, r := range []approvalRecord{fxDecision(t, "APPROVED", "VERIFIED"), fxNotice(t), fxDeferral(t)} {
		if lvl, _ := validateApprovalRecord(r, true); lvl != levelIncomplete {
			t.Errorf("%s fixture accepted as a repository record (%s)", r.RecordType, lvl)
		}
	}
}

// Every documented transition (T1–T6, T9) is representable, and each target state is validated with the
// evidence appropriate to it.
func TestLifecycleTransitionsRepresentable(t *testing.T) {
	type tr struct{ from, to string }
	impl := map[tr]bool{
		{"EVIDENCE_MISSING", "READY_FOR_REVIEW"}: true, {"DRAFT", "READY_FOR_REVIEW"}: true,
		{"READY_FOR_REVIEW", "UNDER_HUMAN_REVIEW"}: true, {"UNDER_HUMAN_REVIEW", "DECISION_RECORDED_PENDING_VERIFICATION"}: true,
		{"DECISION_RECORDED_PENDING_VERIFICATION", "APPROVED_VERIFIED"}: true, {"DECISION_RECORDED_PENDING_VERIFICATION", "REJECTED_VERIFIED"}: true,
		{"DECISION_RECORDED_PENDING_VERIFICATION", "SUPERSEDED_VERIFIED"}: true, {"DECISION_RECORDED_PENDING_VERIFICATION", "VOID_VERIFIED"}: true,
		{"READY_FOR_REVIEW", "DEFERRED"}: true, {"UNDER_HUMAN_REVIEW", "DEFERRED"}: true, {"DEFERRED", "READY_FOR_REVIEW"}: true,
	}
	doc := read(t, "docs/remediation/r6-human-approval-workflow.md")
	row := regexp.MustCompile("(?m)^\\| T[0-9]+ \\| (.*?) → (.*?) \\|")
	state := regexp.MustCompile("`([A-Z_]+)`")
	seen := map[tr]bool{}
	for _, m := range row.FindAllStringSubmatch(doc, -1) {
		for _, f := range state.FindAllStringSubmatch(m[1], -1) {
			for _, to := range state.FindAllStringSubmatch(m[2], -1) {
				k := tr{f[1], to[1]}
				seen[k] = true
				if !impl[k] {
					t.Errorf("documented transition %s → %s is not implemented", k.from, k.to)
				}
			}
		}
	}
	for k := range impl {
		if !seen[k] {
			t.Errorf("implemented transition %s → %s is not documented", k.from, k.to)
		}
	}
	// Target evidence: record-backed states accept exactly their record type; the others need no record.
	evidence := map[string]approvalRecord{"UNDER_HUMAN_REVIEW": fxNotice(t), "DEFERRED": fxDeferral(t),
		"DECISION_RECORDED_PENDING_VERIFICATION": fxDecision(t, "APPROVED", ""),
		"APPROVED_VERIFIED":                      fxDecision(t, "APPROVED", "VERIFIED"), "REJECTED_VERIFIED": fxDecision(t, "REJECTED", "VERIFIED"),
		"SUPERSEDED_VERIFIED": fxDecision(t, "SUPERSEDED", "VERIFIED"), "VOID_VERIFIED": fxDecision(t, "VOID", "VERIFIED")}
	for _, st := range lifecycle {
		rec, ok := evidence[st]
		if needsRecord[st] != ok {
			t.Errorf("%s: record requirement mismatch", st)
			continue
		}
		if ok {
			lvl, _ := validateApprovalRecord(rec, false)
			if err := statusMatchesRecord(st, rec, lvl); err != nil {
				t.Errorf("%s not representable: %v", st, err)
			}
		}
	}
}

// Records are append-only: distinct files per (decision, version, type); a committed file may only gain a
// human verification entry; no record is ever deleted or renamed.
func TestApprovalRecordsAppendOnly(t *testing.T) {
	d := fxDecision(t, "APPROVED", "")
	if recordFileName(d) != "OPD-04-v1.0-DECISION.json" || recordFileName(fxNotice(t)) == recordFileName(d) {
		t.Errorf("file names must separate record types: %s / %s", recordFileName(d), recordFileName(fxNotice(t)))
	}
	if got := recordFileName(approvalRecord{DecisionID: "D-29 / D-31", DecisionVersion: "2.0", RecordType: "DECISION"}); got != "D-29_D-31-v2.0-DECISION.json" {
		t.Errorf("sanitised file name %s", got)
	}
	if err := appendOnlyChange(d, fxDecision(t, "APPROVED", "VERIFIED")); err != nil {
		t.Errorf("adding a verification entry must be allowed: %v", err)
	}
	rewritten := fxDecision(t, "APPROVED", "")
	rewritten.DecisionText = "changed"
	if appendOnlyChange(d, rewritten) == nil {
		t.Error("rewriting a committed decision must be rejected")
	}
	changedVerification := fxDecision(t, "APPROVED", "VERIFIED")
	changedVerification.HumanVerification.Result = "REJECTED"
	if appendOnlyChange(fxDecision(t, "APPROVED", "VERIFIED"), changedVerification) == nil {
		t.Error("changing an existing verification entry must be rejected")
	}
	if appendOnlyChange(fxDecision(t, "APPROVED", "VERIFIED"), d) == nil {
		t.Error("removing a verification entry must be rejected")
	}

	// Repository history (needs full history; NOT VERIFIED and skipped otherwise unless required).
	requireHistory(t)
	const dir = "docs/governance/approvals"
	if out, _ := git("log", "--diff-filter=DR", "--name-only", "--format=", "--", dir); strings.TrimSpace(out) != "" {
		t.Errorf("approval records were deleted or renamed:\n%s", out)
	}
	files, _ := git("log", "--diff-filter=AM", "--name-only", "--format=", "--", dir)
	for _, f := range strings.Fields(files) {
		revs, _ := git("log", "--format=%H", "--reverse", "--", f)
		var prev *approvalRecord
		for _, rev := range strings.Fields(revs) {
			body, err := git("show", rev+":"+f)
			if err != nil {
				continue
			}
			var r approvalRecord
			if err := json.Unmarshal([]byte(body), &r); err != nil {
				t.Errorf("%s at %s: %v", f, rev, err)
				continue
			}
			if prev != nil {
				if err := appendOnlyChange(*prev, r); err != nil {
					t.Errorf("%s at %s: %v", f, rev, err)
				}
			}
			prev = &r
		}
	}
}

// Committed records live in correctly named files, and the R6 baseline holds zero real records.
func TestR6ZeroRecordedApprovals(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(root, "docs/governance/approvals"))
	if err == nil {
		for _, e := range entries {
			rec, err := loadApproval(filepath.Join(root, "docs/governance/approvals", e.Name()))
			if err == nil && recordFileName(rec) != e.Name() {
				t.Errorf("%s must be named %s", e.Name(), recordFileName(rec))
			}
		}
	}
	// R6 baseline: no real approval exists. A future PR that commits a verified record updates this check
	// together with the record and its human verification.
	if err == nil && len(entries) > 0 {
		t.Errorf("R6 baseline expects zero approval records, found %d", len(entries))
	}
	for _, r := range loadTracker(t).Records {
		if needsRecord[r.Status] {
			t.Errorf("%s: R6 baseline has no recorded decision, but status is %s", r.ID, r.Status)
		}
	}
	// Templates are blank and therefore never valid records.
	for _, tpl := range []string{"APPROVAL-RECORD-TEMPLATE.json", "REVIEW-NOTICE-TEMPLATE.json", "DEFERRAL-RECORD-TEMPLATE.json"} {
		var r approvalRecord
		if err := json.Unmarshal([]byte(read(t, "docs/governance/templates/"+tpl)), &r); err != nil {
			t.Fatalf("%s: %v", tpl, err)
		}
		if r.RecordType == "" {
			t.Errorf("%s has no record_type", tpl)
		}
		if lvl, _ := validateApprovalRecord(r, true); lvl != levelIncomplete {
			t.Errorf("blank template %s validates as %s", tpl, lvl)
		}
	}
}
