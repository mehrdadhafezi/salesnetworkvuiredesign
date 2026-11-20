package governance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// RM-E04F: dynamic governance authority and role-assignment architecture. Documentation only. These tests keep
// the provisional nominations provisional, keep the workflow free of shortcuts to ACTIVE, and pin the existing R6
// validator behaviour that the design relies on (one person = one approver; verifier independence; append-only
// history). They check documents, governance data and the offline validator. They do NOT enforce runtime
// behaviour: no governance runtime exists.

var rmE04FDocs = []string{
	"docs/governance/proposals/rm-e04f-authority-role-assignment-architecture.md",
	"docs/governance/proposals/rm-e04f-executive-summary.fa.md",
	"docs/implementation/rm-e04f-codex-security-review-handoff.md",
}

const rmE04FRegister = "docs/governance/rm-e04f-authority-nominations.json"

type rmE04FNomination struct {
	AuthorityKey        string `json:"authority_key"`
	R6Role              string `json:"r6_role"`
	PositionKey         string `json:"position_key"`
	NomineeKey          string `json:"nominee_key"`
	State               string `json:"state"`
	NominatedBy         string `json:"nominated_by"`
	AppointmentEvidence string `json:"appointment_evidence"`
	AuthorityEvidence   string `json:"authority_evidence"`
	VerifiedBy          string `json:"verified_by"`
	ValidFrom           string `json:"valid_from"`
	ValidTo             string `json:"valid_to"`
}

type rmE04FRegisterDoc struct {
	AllowedStates       []string `json:"allowed_states"`
	AppointingAuthority struct {
		PositionKey       string `json:"position_key"`
		PositionKind      string `json:"position_kind"`
		HolderNomineeKey  string `json:"holder_nominee_key"`
		HolderStatus      string `json:"holder_status"`
		EvidenceReference string `json:"evidence_reference"`
	} `json:"appointing_authority"`
	Nominees []struct {
		NomineeKey         string `json:"nominee_key"`
		DisplayName        string `json:"display_name"`
		PersonRef          string `json:"person_ref"`
		IdentityResolution string `json:"identity_resolution"`
	} `json:"nominees"`
	Nominations []rmE04FNomination `json:"nominations"`
}

func rmE04FLoadRegister(t *testing.T) (rmE04FRegisterDoc, map[string]any) {
	t.Helper()
	raw := read(t, rmE04FRegister)
	var r rmE04FRegisterDoc
	var generic map[string]any
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		t.Fatal(err)
	}
	return r, generic
}

func TestRME04FDocumentsMarkedNotApproved(t *testing.T) {
	for _, f := range append(rmE04FDocs, rmE04FRegister) {
		d := read(t, f)
		head := d
		if len(head) > 1500 {
			head = head[:1500]
		}
		if !strings.Contains(head, "DRAFT — NOT APPROVED") {
			t.Errorf("%s: no DRAFT — NOT APPROVED marker near the top", f)
		}
		if !strings.Contains(d, "NO-GO") {
			t.Errorf("%s: Production NO-GO not stated", f)
		}
		if strings.ContainsAny(d, "☑☒✔✓") || strings.Contains(d, "[x]") || strings.Contains(d, "[X]") {
			t.Errorf("%s: a choice or checklist item is pre-ticked", f)
		}
		for i, l := range strings.Split(d, "\n") {
			if claimedSignOff.MatchString(l) && !regexp.MustCompile(`(?i)pending|not (obtained|recorded|claimed)|no .*(sign-off|review)|required|owed|requested`).MatchString(l) {
				t.Errorf("%s:%d claims a sign-off: %s", f, i+1, strings.TrimSpace(l))
			}
		}
	}
	h := read(t, rmE04FDocs[2])
	if !strings.Contains(h, "| Status | **REVIEW PENDING.**") {
		t.Error("the handoff must state REVIEW PENDING until Codex / Security record a review themselves")
	}
	if regexp.MustCompile(`\|\s*(ANSWERED|ACCEPTED|APPROVED|PASSED)\s*\|`).MatchString(h) {
		t.Error("the handoff marks a review question as answered; only the reviewers may do that")
	}
}

// A candidate name never becomes a verified authority: every entry is NOMINATED or UNASSIGNED, nothing that
// would evidence or activate it is filled, the CEO position has at most a NOMINATED holder (never self-appointing),
// and T5 stays unassigned.
func TestRME04FNominationsStayProvisional(t *testing.T) {
	r, _ := rmE04FLoadRegister(t)
	allowed := map[string]bool{"NOMINATED": true, "UNASSIGNED": true}
	if len(r.AllowedStates) != 2 || !allowed[r.AllowedStates[0]] || !allowed[r.AllowedStates[1]] {
		t.Errorf("allowed_states widened: %v", r.AllowedStates)
	}
	a := r.AppointingAuthority
	if a.PositionKey != "CEO" || a.PositionKind != "ROOT" || !allowed[a.HolderStatus] || a.EvidenceReference != "" {
		t.Errorf("the CEO position must be a keyed ROOT position, at most NOMINATED, with no evidence: %+v", a)
	}
	// RM-E04Q: the CEO position may carry a provisional nominee (NOMINATED), never an appointed or verified holder.
	if (a.HolderStatus == "UNASSIGNED") != (a.HolderNomineeKey == "") {
		t.Errorf("CEO holder status %s inconsistent with nominee %q", a.HolderStatus, a.HolderNomineeKey)
	}
	nominees := map[string]bool{}
	for _, n := range r.Nominees {
		if nominees[n.NomineeKey] {
			t.Errorf("duplicate nominee key %s", n.NomineeKey)
		}
		nominees[n.NomineeKey] = true
		if n.PersonRef != "" || n.IdentityResolution != "NOT_STARTED" {
			t.Errorf("nominee %s: a person reference or identity resolution is claimed", n.NomineeKey)
		}
	}
	if a.HolderNomineeKey != "" && !nominees[a.HolderNomineeKey] {
		t.Errorf("CEO holder nominee %q is not in the nominee register", a.HolderNomineeKey)
	}
	// Rule SA (§5.4): the CEO nominee can never also be a nominee for an office the CEO appoints.
	for _, n := range r.Nominations {
		if a.HolderNomineeKey != "" && n.NomineeKey == a.HolderNomineeKey {
			t.Errorf("%s: the CEO nominee is also nominated for an office the CEO appoints (self-appointment)", n.AuthorityKey)
		}
	}
	if len(r.Nominations) == 0 {
		t.Fatal("no nominations")
	}
	t5 := 0
	for _, n := range r.Nominations {
		if !allowed[n.State] {
			t.Errorf("%s: state %q; only NOMINATED / UNASSIGNED are allowed here", n.AuthorityKey, n.State)
		}
		if n.AppointmentEvidence != "" || n.AuthorityEvidence != "" || n.VerifiedBy != "" || n.ValidFrom != "" || n.ValidTo != "" {
			t.Errorf("%s: evidence, verifier or validity filled for a nomination", n.AuthorityKey)
		}
		if (n.State == "UNASSIGNED") != (n.NomineeKey == "") {
			t.Errorf("%s: state %s inconsistent with nominee %q", n.AuthorityKey, n.State, n.NomineeKey)
		}
		if n.NomineeKey != "" && !nominees[n.NomineeKey] {
			t.Errorf("%s: unknown nominee %q", n.AuthorityKey, n.NomineeKey)
		}
		if !regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`).MatchString(n.PositionKey) ||
			!regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`).MatchString(n.AuthorityKey) {
			t.Errorf("%s / %s: positions and authorities must be stable keys, not names", n.PositionKey, n.AuthorityKey)
		}
		if n.AuthorityKey == "decision.verify_t5" {
			t5++
			if n.State != "UNASSIGNED" {
				t.Error("T5 verifier must stay UNASSIGNED until an independent person is identified")
			}
		}
	}
	if t5 != 1 {
		t.Errorf("want exactly one T5 verifier entry, found %d", t5)
	}
	// Stable identifiers: no nominee name appears in a key.
	for _, n := range r.Nominees {
		for _, nom := range r.Nominations {
			if strings.Contains(nom.PositionKey+nom.AuthorityKey, n.DisplayName) {
				t.Errorf("a position or authority key embeds a personal name")
			}
		}
	}
	// One person holding several nominations is one independent person: the nominations alone can never
	// satisfy the Group B floor roles.
	floorHolders := map[string]bool{}
	for _, n := range r.Nominations {
		for _, role := range rmE04CGateFloorRoles {
			if n.R6Role == role && n.NomineeKey != "" {
				floorHolders[n.NomineeKey] = true
			}
		}
	}
	if len(floorHolders) >= len(rmE04CGateFloorRoles) {
		t.Errorf("nominations name %d distinct people for %d floor roles; recheck independence before relying on it", len(floorHolders), len(rmE04CGateFloorRoles))
	}
}

// §5.2: ACTIVE is reachable only from PENDING_VERIFICATION, by a human verifier; no other shortcut exists.
func TestRME04FWorkflowHasNoShortcutToActive(t *testing.T) {
	d := read(t, rmE04FDocs[0])
	toActive := 0
	for _, l := range strings.Split(d, "\n") {
		if !strings.HasPrefix(l, "| T-A") {
			continue
		}
		cols := strings.Split(l, "|")
		if len(cols) < 5 {
			t.Fatalf("malformed transition row: %s", l)
		}
		parts := strings.Split(strings.ReplaceAll(cols[2], "`", ""), "→")
		target := strings.TrimSpace(parts[len(parts)-1])
		if target != "ACTIVE" {
			continue
		}
		toActive++
		src := strings.TrimSpace(parts[0])
		actor := strings.ToLower(cols[3])
		if src != "PENDING_VERIFICATION" {
			t.Errorf("ACTIVE reachable from %q", src)
		}
		if !strings.Contains(actor, "independent verifier") || !strings.Contains(actor, "human") || strings.Contains(actor, "system") {
			t.Errorf("activation actor must be an independent human verifier: %s", cols[3])
		}
	}
	if toActive != 1 {
		t.Errorf("want exactly one transition into ACTIVE, found %d", toActive)
	}
	if !strings.Contains(d, "PENDING_VERIFICATION ──▶ ACTIVE") || strings.Count(d, "──▶ ACTIVE") != 1 {
		t.Error("state diagram must show ACTIVE reached only from PENDING_VERIFICATION")
	}
}

func rmE04FSignedDecision(t *testing.T, signers map[string]string, recorder, verifier string) approvalRecord {
	t.Helper()
	r := fxDecision(t, "APPROVED", "")
	r.Approvers, r.IndependentApprovals = nil, nil
	roles := make([]string, 0, len(signers))
	for role := range signers {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		a := authorizedPerson{Name: signers[role], Role: role}
		a.AuthorityEvidence.Type, a.AuthorityEvidence.Reference = "appointment letter", "ref-"+role
		r.Approvers = append(r.Approvers, a)
		r.IndependentApprovals = append(r.IndependentApprovals, role)
	}
	r.RecordedBy = person{Name: recorder, Role: "Recorder"}
	if verifier != "" {
		v := fxDecision(t, "APPROVED", "VERIFIED").HumanVerification
		v.VerifierName = verifier
		r.HumanVerification = v
	}
	return r
}

// Scenario B: one person signing under Product, Sales and Security is one approver, whatever the spelling noise
// the current name key normalises (case, repeated / surrounding spaces).
func TestRME04FOnePersonIsOneApprover(t *testing.T) {
	r, _ := rmE04FLoadRegister(t)
	n1 := r.Nominees[0].DisplayName
	control := rmE04FSignedDecision(t, map[string]string{"Product owner": "Person A", "Sales owner": "Person B", "Security owner": "Person C"}, "Person R", "")
	if lvl, p := validateApprovalRecord(control, false); lvl != levelRecorded {
		t.Fatalf("control miscalibrated: three distinct signers should be structurally complete (%s %v)", lvl, p)
	}
	for _, variants := range [][3]string{{n1, n1, n1}, {n1, " " + n1, n1 + "  "}, {n1, strings.ReplaceAll(n1, " ", "  "), n1}} {
		rec := rmE04FSignedDecision(t, map[string]string{"Product owner": variants[0], "Sales owner": variants[1], "Security owner": variants[2]}, "Person R", "")
		if lvl, p := validateApprovalRecord(rec, false); lvl != levelIncomplete || !strings.Contains(strings.Join(p, ";"), "already signed under another role") {
			t.Errorf("one person under three roles accepted (%s %v)", lvl, p)
		}
	}
}

// Scenario C (name level): the recorder or an approver cannot be the T5 verifier.
func TestRME04FRecorderOrApproverCannotVerify(t *testing.T) {
	signers := map[string]string{"Product owner": "Person A", "Sales owner": "Person B", "Security owner": "Person C"}
	ok := rmE04FSignedDecision(t, signers, "Person R", "Person V")
	if lvl, p := validateApprovalRecord(ok, false); lvl != levelHumanVerified {
		t.Fatalf("control miscalibrated: independent verifier should verify (%s %v)", lvl, p)
	}
	for _, v := range []string{"Person R", "person  r", "Person B", "PERSON A"} {
		rec := rmE04FSignedDecision(t, signers, "Person R", v)
		if lvl, p := validateApprovalRecord(rec, false); lvl == levelHumanVerified || !strings.Contains(strings.Join(p, ";"), "verifier must not be an approver or the recorder") {
			t.Errorf("verifier %q accepted (%s %v)", v, lvl, p)
		}
	}
}

// Scenario A: replacing the Sales owner never rewrites a committed approval record.
func TestRME04FHistoricalApprovalNotRewritten(t *testing.T) {
	signers := map[string]string{"Product owner": "Person A", "Sales owner": "Original Sales Owner", "Security owner": "Person C"}
	old := rmE04FSignedDecision(t, signers, "Person R", "")
	rewritten := rmE04FSignedDecision(t, signers, "Person R", "")
	for i := range rewritten.Approvers {
		if rewritten.Approvers[i].Role == "Sales owner" {
			rewritten.Approvers[i].Name = "Successor Sales Owner"
			rewritten.Approvers[i].AuthorityEvidence.Reference = "successor-appointment"
		}
	}
	if err := appendOnlyChange(old, rewritten); err == nil {
		t.Error("replacing the historical signer in a committed record was accepted")
	}
	verified := rmE04FSignedDecision(t, signers, "Person R", "Person V")
	if err := appendOnlyChange(old, verified); err != nil {
		t.Errorf("adding a verification entry must stay allowed: %v", err)
	}
}

// A nominee whose authority is only nominated cannot appear as approver, recorder or verifier in any committed
// approval record or tracker outcome.
func TestRME04FNomineeIsNotAnApprover(t *testing.T) {
	r, _ := rmE04FLoadRegister(t)
	keys := map[string]bool{}
	for _, n := range r.Nominees {
		keys[personKey(n.DisplayName)] = true
	}
	dir := filepath.Join(root, "docs/governance/approvals")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		rec, err := loadApproval(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		names := []string{rec.RecordedBy.Name}
		for _, a := range rec.Approvers {
			names = append(names, a.Name)
		}
		if rec.Initiator != nil {
			names = append(names, rec.Initiator.Name)
		}
		if rec.HumanVerification != nil {
			names = append(names, rec.HumanVerification.VerifierName)
		}
		for _, n := range names {
			if keys[personKey(n)] {
				t.Errorf("%s names nominee %q, whose authority is only NOMINATED", e.Name(), n)
			}
		}
	}
	for _, rec := range loadTracker(t).Records {
		for k := range keys {
			if k != "" && strings.Contains(personKey(rec.Outcome["approvers"]), k) {
				t.Errorf("tracker %s names a nominee as approver", rec.ID)
			}
		}
	}
}

// Governance authority is never a CRM grant: the register has no role, capability or grant field; identity still
// keeps every role grant SHADOW; no migration gives the seller bundle a capability.
func TestRME04FAuthorityIsNotAGrant(t *testing.T) {
	_, generic := rmE04FLoadRegister(t)
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				for _, bad := range []string{"role_key", "capabilit", "grant", "bundle", "credential"} {
					if strings.Contains(strings.ToLower(k), bad) {
						t.Errorf("register field %q looks like a CRM grant", k)
					}
				}
				walk(val)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(generic)
	migs := identityMigrations(t)
	all := ""
	for _, body := range migs {
		all += body
	}
	if !strings.Contains(all, "CONSTRAINT role_grant_shadow_only CHECK (status = 'SHADOW')") {
		t.Error("identity.role_grant_shadow_only is gone")
	}
	if regexp.MustCompile(`\(\s*'seller'\s*,\s*'[a-z_.]+'\s*\)`).MatchString(all) {
		t.Error("a migration gives the seller bundle a capability")
	}
}

// Only tracker or R5 inventory ids may be cited.
func TestRME04FNoInventedDecisionID(t *testing.T) {
	known := ids(read(t, "docs/remediation/r5-decision-inventory.md"))
	for _, r := range loadTracker(t).Records {
		known[strings.SplitN(r.ID, " ", 2)[0]] = true
	}
	idPattern := regexp.MustCompile(`\b(?:D-[0-9]+[a-e]?|OPD-[0-9]+|R5-[A-Z]+-[0-9]+)\b`)
	for _, f := range rmE04FDocs {
		var unknown []string
		for _, id := range idPattern.FindAllString(read(t, f), -1) {
			if !known[id] && !known[strings.TrimRight(id, "abcde")] {
				unknown = append(unknown, id)
			}
		}
		if len(unknown) > 0 {
			t.Errorf("%s cites decision IDs not in the tracker: %v", f, unknown)
		}
	}
}
