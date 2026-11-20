package governance

// RM-E04I — R5-SEC-01 option A shadow implementation readiness (documentation-level guards only).
//
// These tests guard safety properties, not today's repository snapshot. Every repository-state check is conditional
// on the governing decision ("while R5-SEC-01 is not APPROVED_VERIFIED …"), so a legitimate future approval and its
// authorised PRs are not blocked here. They check that:
//   1. the project owner's preference for option A stays a non-authoritative record nobody consumes;
//   2. no person-registry schema, runtime person resolver or identity activation exists before R5-SEC-01 is approved;
//   3. Group B / S4 readiness and the runtime seller bundle stay bound to verified person independence, and no
//      ACTIVE grant path exists without D-42;
//   4. the D-42 unlock gap stays documented while it is open;
//   5. the proposed resolver contract (§5) and its integration with the unchanged RM-E04G evaluator (§7) fail
//      closed — as an executable specification over synthetic data, never wired to a gate;
//   6. the dependency graph is well formed, faithful to the tracker and records every documented cycle UNRESOLVED;
//   7. the documents stay DRAFT, NO-GO, review PENDING, with no claimed sign-off.

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
)

const (
	rmE04IContract   = "docs/security/rm-e04i-option-a-implementation-contract.md"
	rmE04IHandoff    = "docs/implementation/rm-e04i-review-handoff.md"
	rmE04IPreference = "docs/governance/proposals/rm-e04i-option-a-preference.NON-AUTHORITATIVE.json"
	rmE04IGraph      = "docs/governance/proposals/rm-e04i-dependency-graph.json"
	rmE04IPrefStem   = "rm-e04i-option-a-preference"
	rmE04ITestFile   = "docs/governance/rm_e04i_test.go"
)

var rmE04IDocs = []string{rmE04IContract, rmE04IHandoff}

func rmE04IR5Approved(t *testing.T) bool {
	t.Helper()
	return trackerByID(t)[rmE04HID].Status == "APPROVED_VERIFIED"
}

// ---------------- 1. preference record

type rmE04IPreferenceDoc struct {
	RecordType                  string            `json:"record_type"`
	Purpose                     string            `json:"purpose"`
	Baseline                    string            `json:"baseline"`
	DecisionID                  string            `json:"decision_id"`
	PreferredOption             string            `json:"preferred_option"`
	PreferenceSource            string            `json:"preference_source"`
	DecisionStatus              string            `json:"decision_status"`
	AuthorityStatus             string            `json:"authority_status"`
	ImplementationAuthorization string            `json:"implementation_authorization"`
	ProductionDeployment        string            `json:"production_deployment"`
	Layers                      map[string]string `json:"layers"`
	OriginalOptionsPreserved    []string          `json:"original_options_preserved"`
	Signatures                  []any             `json:"signatures"`
	T5Verification              any               `json:"t5_verification"`
	ConsumableBy                []string          `json:"consumable_by"`
	NotEvidenceOf               []string          `json:"not_evidence_of"`
	Contract                    string            `json:"contract"`
}

func rmE04ILoadPreference(t *testing.T) rmE04IPreferenceDoc {
	t.Helper()
	var p rmE04IPreferenceDoc
	dec := json.NewDecoder(strings.NewReader(read(t, rmE04IPreference)))
	dec.DisallowUnknownFields() // no approver, signature or verification field can be smuggled in
	if err := dec.Decode(&p); err != nil {
		t.Fatal(err)
	}
	return p
}

// An authoritative-looking token in a status field of a preference record.
var rmE04IAuthoritative = regexp.MustCompile(`\b(APPROVED|APPROVED_VERIFIED|VERIFIED|ACTIVE|GRANTED|AUTHORIZED|AUTHORISED|SELECTED|SIGNED)\b`)

func rmE04IPreferenceProblems(p rmE04IPreferenceDoc, original []string) []string {
	var out []string
	want := map[string]string{
		"record_type": "PROJECT_PREFERENCE_NON_AUTHORITATIVE", "decision_id": rmE04HID, "preferred_option": "A",
		"decision_status": "NOT_APPROVED", "authority_status": "UNVERIFIED", "implementation_authorization": "NOT_GRANTED",
		"production_deployment": "NO-GO",
	}
	got := map[string]string{
		"record_type": p.RecordType, "decision_id": p.DecisionID, "preferred_option": p.PreferredOption,
		"decision_status": p.DecisionStatus, "authority_status": p.AuthorityStatus,
		"implementation_authorization": p.ImplementationAuthorization, "production_deployment": p.ProductionDeployment,
	}
	for k, v := range want {
		if got[k] != v {
			out = append(out, "preference "+k+" = "+got[k]+", want "+v)
		}
	}
	for k, v := range got {
		if k != "record_type" && k != "decision_id" && rmE04IAuthoritative.MatchString(v) {
			out = append(out, "preference "+k+" carries an authoritative value: "+v)
		}
	}
	for _, k := range []string{"official_governance_decision", "human_approval", "independent_t5_verification"} {
		if !strings.HasPrefix(p.Layers[k], "NONE") {
			out = append(out, "layer "+k+" must stay NONE in a preference record: "+p.Layers[k])
		}
	}
	if p.Layers["implementation_authorization"] != "NOT_GRANTED" {
		out = append(out, "layer implementation_authorization is not NOT_GRANTED")
	}
	if !strings.HasPrefix(p.Layers["project_preference"], "A ") || !strings.Contains(p.Layers["engineering_recommendation"], "conditional") {
		out = append(out, "project preference and engineering recommendation are not kept as separate, non-binding layers")
	}
	if len(p.Signatures) != 0 || p.T5Verification != nil || len(p.ConsumableBy) != 0 {
		out = append(out, "preference record carries a signature, a T5 verification or a consumer")
	}
	if strings.Join(p.OriginalOptionsPreserved, "\n") != strings.Join(original, "\n") {
		out = append(out, "original options A–D are not preserved verbatim")
	}
	if !strings.Contains(p.Purpose, "DRAFT — NOT APPROVED") || !strings.Contains(p.Purpose, "NOT implementation authorization") {
		out = append(out, "purpose does not state DRAFT — NOT APPROVED and non-authorization")
	}
	return out
}

func TestRME04IPreferenceIsNotADecision(t *testing.T) {
	p := rmE04ILoadPreference(t)
	for _, x := range rmE04IPreferenceProblems(p, rmE04HOriginalOptions(t)) {
		t.Error(x)
	}
	if strings.Contains(rmE04IPreference, "/approvals/") {
		t.Error("the preference record must never live in docs/governance/approvals")
	}
	// Negative fixtures: the rule itself must catch an authoritative preference.
	for name, mutate := range map[string]func(*rmE04IPreferenceDoc){
		"decision approved":   func(d *rmE04IPreferenceDoc) { d.DecisionStatus = "APPROVED_VERIFIED" },
		"authority verified":  func(d *rmE04IPreferenceDoc) { d.AuthorityStatus = "VERIFIED" },
		"implementation go":   func(d *rmE04IPreferenceDoc) { d.ImplementationAuthorization = "GRANTED" },
		"signature added":     func(d *rmE04IPreferenceDoc) { d.Signatures = []any{"someone"} },
		"t5 added":            func(d *rmE04IPreferenceDoc) { d.T5Verification = map[string]string{"result": "VERIFIED"} },
		"consumer added":      func(d *rmE04IPreferenceDoc) { d.ConsumableBy = []string{"group-b-gate"} },
		"option dropped":      func(d *rmE04IPreferenceDoc) { d.OriginalOptionsPreserved = d.OriginalOptionsPreserved[:3] },
		"official decision A": func(d *rmE04IPreferenceDoc) { d.Layers = map[string]string{"official_governance_decision": "A"} },
		"production go":       func(d *rmE04IPreferenceDoc) { d.ProductionDeployment = "AUTHORIZED" },
		"preference selected": func(d *rmE04IPreferenceDoc) { d.PreferredOption = "SELECTED" },
	} {
		m := rmE04ILoadPreference(t)
		m.Layers = map[string]string{}
		for k, v := range p.Layers {
			m.Layers[k] = v
		}
		mutate(&m)
		if len(rmE04IPreferenceProblems(m, rmE04HOriginalOptions(t))) == 0 {
			t.Errorf("negative fixture %q passed the preference rule", name)
		}
	}
}

func TestRME04IPreferenceIsNotConsumed(t *testing.T) {
	scanned := map[string]bool{".go": true, ".json": true, ".sql": true, ".yaml": true, ".yml": true, ".ts": true,
		".tsx": true, ".js": true, ".sh": true, ".toml": true}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == ".git" || n == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !scanned[filepath.Ext(p)] || rel == rmE04IPreference || rel == rmE04ITestFile {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), rmE04IPrefStem) {
			t.Errorf("%s references the non-authoritative preference record; no gate, evaluator, migration, provisioning or test may consume it", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// A selected option in the tracker needs an approval record; a preference is never a substitute.
	for _, r := range loadTracker(t).Records {
		if strings.TrimSpace(r.Outcome["selected_option"]) != "" && r.ApprovalRecord == "" {
			t.Errorf("%s: selected_option %q without an approval record", r.ID, r.Outcome["selected_option"])
		}
	}
}

// ---------------- 2. no identity activation before an approved R5-SEC-01

var rmE04IRegistryTable = regexp.MustCompile(`\b(verified_person|person_identity_evidence|identity_verification|person_account_link|verified_person_account_link|person_link_review|identity_link_revocation|identity_conflict_finding)\b`)
var rmE04IRuntimeResolver = regexp.MustCompile(`(?i)\b(func\s+(\([^)]*\)\s*)?resolve_?person|resolve_?person\s*\(|type\s+verifiedpersonaccountlink)\b`)

func rmE04IAllMigrations(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	dirs, _ := filepath.Glob(filepath.Join(root, "internal", "*", "migrations", "sql"))
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".sql") {
				rel, _ := filepath.Rel(root, filepath.Join(dir, e.Name()))
				out[filepath.ToSlash(rel)] = read(t, filepath.ToSlash(rel))
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no migrations found; the scan is broken")
	}
	return out
}

func rmE04IRuntimeGoSources(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, top := range []string{"internal", "cmd", "api"} {
		_ = filepath.WalkDir(filepath.Join(root, top), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			b, err := os.ReadFile(p)
			if err == nil {
				rel, _ := filepath.Rel(root, p)
				out[filepath.ToSlash(rel)] = string(b)
			}
			return nil
		})
	}
	if len(out) == 0 {
		t.Fatal("no runtime Go sources found; the scan is broken")
	}
	return out
}

func rmE04IActivationProblems(r5Approved bool, migrations, goSrc map[string]string) []string {
	if r5Approved {
		return nil // PR-B / PR-C may then proceed under their own reviews; nothing here freezes them
	}
	var p []string
	for name, sql := range migrations {
		if m := rmE04IRegistryTable.FindString(sqlCode(sql)); m != "" {
			p = append(p, name+" defines person-registry object "+m+" while R5-SEC-01 is not APPROVED_VERIFIED")
		}
	}
	for name, src := range goSrc {
		if m := rmE04IRuntimeResolver.FindString(goCode(src)); m != "" {
			p = append(p, name+" introduces a runtime person resolver ("+m+") while R5-SEC-01 is not APPROVED_VERIFIED")
		}
	}
	return p
}

func TestRME04INoRuntimeResolverOrIdentityActivation(t *testing.T) {
	approved := rmE04IR5Approved(t)
	for _, x := range rmE04IActivationProblems(approved, rmE04IAllMigrations(t), rmE04IRuntimeGoSources(t)) {
		t.Error(x)
	}
	if approved {
		return
	}
	for _, in := range []string{"acct-1", "مهرداد حافظی", "+989120000000", "someone@example.com"} {
		if r := rmE04GRepositoryResolver(in); r.State != idUnknown || r.PersonID != "" {
			t.Errorf("repository resolver resolved %q while R5-SEC-01 is not approved", in)
		}
	}
	reg, _ := rmE04FLoadRegister(t)
	for _, n := range reg.Nominees {
		if n.PersonRef != "" {
			t.Errorf("nominee %s carries a person_ref while R5-SEC-01 is not approved", n.NomineeKey)
		}
	}
	// Negative fixtures for the rule itself.
	if len(rmE04IActivationProblems(false, map[string]string{"x.sql": "CREATE TABLE organization.verified_person_account_link (link_id uuid);"}, nil)) == 0 {
		t.Error("a link-table migration was not detected")
	}
	if len(rmE04IActivationProblems(false, nil, map[string]string{"x.go": "package q\nfunc ResolvePerson(a string) {}"})) == 0 {
		t.Error("a runtime resolver was not detected")
	}
	if len(rmE04IActivationProblems(true, map[string]string{"x.sql": "CREATE TABLE organization.verified_person (id uuid);"}, map[string]string{"x.go": "package q\nfunc ResolvePerson(a string) {}"})) != 0 {
		t.Error("the rule must not freeze authorised PR-B / PR-C work once R5-SEC-01 is APPROVED_VERIFIED")
	}
	if len(rmE04IActivationProblems(false, map[string]string{"x.sql": "-- verified_person comes later"}, map[string]string{"x.go": "// resolvePerson is planned"})) != 0 {
		t.Error("comments must not count as activation")
	}
}

// ---------------- 3. gates and grants stay bound to person independence

func TestRME04IGatesStayClosedWithoutPersonIndependence(t *testing.T) {
	g := loadRMEGate(t)
	recs, titles := trackerByID(t), r5Titles(t)
	ready, _ := rmE04GGroupBReadiness(g, recs, titles, repoLoader, true, rmE04GClaimsFromRecord, rmE04GRepositoryResolver)
	if ready && !rmE04IR5Approved(t) {
		t.Error("Group B is person-ready while R5-SEC-01 is not APPROVED_VERIFIED (F1 interim guard bypassed)")
	}
	if listed := s4Listed(g); !ready && len(listed) > 0 {
		t.Errorf("S4 %v is listed in the Group B gate while Group B lacks verified person independence", listed)
	}
	seller, ok := identityapi.LookupRole(identityapi.RoleSeller)
	if !ok {
		t.Fatal("seller role missing from the vocabulary")
	}
	for _, x := range rmE04CRuntimeProblems(ready, g, identityapi.KnownCapability, seller.Capabilities, read(t, "api/openapi/openapi.yaml")) {
		t.Error("runtime exceeds person-gated Group B readiness: " + x)
	}
	for _, x := range grantConstraintProblems(identityMigrations(t), recs["D-42"]) {
		t.Error(x)
	}
}

func TestRME04ID42GapIsDocumentedWhileOpen(t *testing.T) {
	src := goCode(read(t, "docs/governance/rm_e04d_test.go"))
	for _, fn := range []string{"grantConstraintProblems", "identityMigrations", "TestRME04DNoActiveGrantPathWithoutD42", "TestRME04DGrantConstraintRules"} {
		if !regexp.MustCompile(`\bfunc ` + fn + `\(`).MatchString(src) {
			t.Errorf("%s no longer exists; contract §8.1 / §8.3 cite it and must be updated", fn)
		}
	}
	def := identityMigrations(t)[grantDefiningMigration]
	if !shadowOnlyCheck.MatchString(def) {
		t.Error("role_grant_shadow_only is gone; contract §8.1 describes it as the DB guard")
	}
	if !strings.Contains(read(t, "internal/identity/store/store.go"), "status = 'ACTIVE'") {
		t.Error("runtime grant reader changed; contract §8.1 must be re-traced")
	}
	// Is the gap still open? A bare D-42 status with R5-SEC-01 undecided, and a migration dropping the guard.
	fx := map[string]string{grantDefiningMigration: def, "99999_unlock.sql": "ALTER TABLE identity.role_grant DROP CONSTRAINT role_grant_shadow_only;"}
	open := len(grantConstraintProblems(fx, trackerRecord{ID: "D-42", Status: "APPROVED_VERIFIED"})) == 0
	doc := read(t, rmE04IContract)
	if open {
		for _, s := range []string{"`grantConstraintProblems(migrations, d42)`", "**any** migration may touch `role_grant`", "H-1", "H-8", "ND-1", "ND-11"} {
			if !strings.Contains(doc, s) {
				t.Errorf("the D-42 unlock still accepts a bare status, but contract §8 no longer states %q", s)
			}
		}
	} else {
		t.Log("D-42 unlock hardened; contract §8.1 should be updated to record the closure")
	}
}

// ---------------- 4. executable resolver specification (synthetic, test-only; never wired to a gate)

const (
	rmE04IVerified = "VERIFIED"
	rmE04IUnknown  = "UNKNOWN"
	rmE04IConflict = "CONFLICT"
	rmE04IBlocked  = "BLOCKED"
)

var rmE04IPurposes = map[string]bool{"R6_APPROVAL_SIGNATURE": true, "R6_RECORDING": true, "T5_VERIFICATION": true,
	"DELEGATION_PRINCIPAL": true, "AUTHORITY_ASSIGNMENT": true, "PERSON_SOD_CHECK": true}

// Time is an integer instant; 0 in To / RevokedAt / EvidenceTo means "open".
type rmE04ILink struct {
	ID, Account, Person string
	State               string // PROPOSED, UNDER_REVIEW, EFFECTIVE
	From, To, Recorded  int
	RevokedAt           int
	EvidenceTo          int
	Purposes            []string
}

type rmE04IRegistry struct {
	Authorised  bool // R5-SEC-01 APPROVED_VERIFIED with option A
	NonPersonal map[string]bool
	SharedUser  map[string]bool
	OpenFinding map[string]bool // account or person
	MergedInto  map[string]string
	Split       map[string]bool
	Links       []rmE04ILink
}

type rmE04IResult struct {
	State, PersonID, Reason string
	EvidenceRefs            []string
}

// rmE04ISpecResolve implements contract §5.3 steps 1–10 over synthetic data. It is the reference behaviour for a
// future PR-C implementation and is not a runtime resolver.
func rmE04ISpecResolve(reg rmE04IRegistry, account string, at int, purpose string) rmE04IResult {
	switch {
	case !reg.Authorised:
		return rmE04IResult{State: rmE04IBlocked, Reason: "RESOLVER_NOT_AUTHORISED"}
	case !rmE04IPurposes[purpose]:
		return rmE04IResult{State: rmE04IBlocked, Reason: "PURPOSE_NOT_PERMITTED"}
	case strings.TrimSpace(account) == "":
		return rmE04IResult{State: rmE04IUnknown, Reason: "NO_ACCOUNT"}
	case reg.NonPersonal[account]:
		return rmE04IResult{State: rmE04IBlocked, Reason: "NON_PERSONAL_ACCOUNT"}
	case reg.SharedUser[account]:
		return rmE04IResult{State: rmE04IConflict, Reason: "SHARED_ACCOUNT"}
	case reg.OpenFinding[account]:
		return rmE04IResult{State: rmE04IConflict, Reason: "OPEN_FINDING"}
	}
	var hit []rmE04ILink
	pending := false
	for _, l := range reg.Links {
		if l.Account != account {
			continue
		}
		if l.State == "PROPOSED" || l.State == "UNDER_REVIEW" {
			pending = true
			continue
		}
		if l.State == "EFFECTIVE" && l.From <= at && (l.To == 0 || at < l.To) && l.Recorded < at {
			hit = append(hit, l)
		}
	}
	switch {
	case len(hit) > 1:
		return rmE04IResult{State: rmE04IConflict, Reason: "OVERLAPPING_LINKS"}
	case len(hit) == 0 && pending:
		return rmE04IResult{State: rmE04IUnknown, Reason: "LINK_PENDING"}
	case len(hit) == 0:
		return rmE04IResult{State: rmE04IUnknown, Reason: "NO_VERIFIED_LINK"}
	}
	l := hit[0]
	switch {
	case l.RevokedAt != 0 && l.RevokedAt <= at:
		return rmE04IResult{State: rmE04IUnknown, Reason: "LINK_REVOKED"}
	case l.EvidenceTo != 0 && l.EvidenceTo <= at:
		return rmE04IResult{State: rmE04IUnknown, Reason: "EVIDENCE_EXPIRED"}
	case !slicesContains(l.Purposes, purpose):
		return rmE04IResult{State: rmE04IBlocked, Reason: "PURPOSE_NOT_ON_LINK"}
	case reg.OpenFinding[l.Person]:
		return rmE04IResult{State: rmE04IConflict, Reason: "OPEN_FINDING"}
	case reg.Split[l.Person]:
		return rmE04IResult{State: rmE04IConflict, Reason: "SPLIT_SOURCE"}
	}
	pid, reason := l.Person, ""
	if s, ok := reg.MergedInto[pid]; ok {
		pid, reason = s, "MERGED_FLAGGED"
	}
	return rmE04IResult{State: rmE04IVerified, PersonID: pid, Reason: reason, EvidenceRefs: []string{l.ID}}
}

var rmE04IAllPurposes = []string{"R6_APPROVAL_SIGNATURE", "R6_RECORDING", "T5_VERIFICATION", "DELEGATION_PRINCIPAL", "AUTHORITY_ASSIGNMENT", "PERSON_SOD_CHECK"}

func rmE04IEff(id, acct, person string, from int) rmE04ILink {
	return rmE04ILink{ID: id, Account: acct, Person: person, State: "EFFECTIVE", From: from, Recorded: from - 1, Purposes: rmE04IAllPurposes}
}

func rmE04IFixtureRegistry() rmE04IRegistry {
	reassignedOld := rmE04IEff("l-re1", "acct-re", "p-old", 1)
	reassignedOld.To, reassignedOld.RevokedAt = 10, 10
	revoked := rmE04IEff("l-rev", "acct-rev", "p-rev", 1)
	revoked.RevokedAt = 5
	expired := rmE04IEff("l-exp", "acct-exp", "p-exp", 1)
	expired.EvidenceTo = 5
	narrow := rmE04IEff("l-nar", "acct-narrow", "p-nar", 1)
	narrow.Purposes = []string{"PERSON_SOD_CHECK"}
	late := rmE04IEff("l-late", "acct-late", "p-late", 1)
	late.Recorded = 50 // back-dated: recorded long after it claims to start
	return rmE04IRegistry{
		Authorised:  true,
		NonPersonal: map[string]bool{"acct-bot": true},
		SharedUser:  map[string]bool{"acct-shared": true},
		OpenFinding: map[string]bool{"acct-finding": true, "p-disputed": true},
		MergedInto:  map[string]string{"p-dup": "p-survivor"},
		Split:       map[string]bool{"p-split": true},
		Links: []rmE04ILink{
			rmE04IEff("l-a1", "acct-a1", "p-a", 1), rmE04IEff("l-a2", "acct-a2", "p-a", 1),
			rmE04IEff("l-b", "acct-b", "p-b", 1), rmE04IEff("l-c", "acct-c", "p-c", 1),
			rmE04IEff("l-r", "acct-r", "p-r", 1), rmE04IEff("l-r2", "acct-r2", "p-r", 1), rmE04IEff("l-v", "acct-v", "p-v", 1),
			rmE04IEff("l-h1", "acct-h1", "p-h1", 1), rmE04IEff("l-h2", "acct-h2", "p-h2", 1),
			rmE04IEff("l-o1", "acct-overlap", "p-o1", 1), rmE04IEff("l-o2", "acct-overlap", "p-o2", 3),
			{ID: "l-pend", Account: "acct-pending", Person: "p-pend", State: "UNDER_REVIEW", From: 1, Purposes: rmE04IAllPurposes},
			reassignedOld, rmE04IEff("l-re2", "acct-re", "p-new", 10), revoked, expired, narrow, late,
			rmE04IEff("l-dup", "acct-dup", "p-dup", 1), rmE04IEff("l-split", "acct-split", "p-split", 1),
			rmE04IEff("l-disp", "acct-disputed", "p-disputed", 1),
			rmE04IEff("l-later", "acct-later", "p-later", 20),
		},
	}
}

func TestRME04IResolverSpec(t *testing.T) {
	reg := rmE04IFixtureRegistry()
	const sig = "R6_APPROVAL_SIGNATURE"
	unauth := reg
	unauth.Authorised = false
	cases := []struct {
		name    string
		reg     rmE04IRegistry
		account string
		at      int
		purpose string
		state   string
		reason  string
		person  string
	}{
		{"resolver not authorised (R5-SEC-01 not approved)", unauth, "acct-a1", 10, sig, rmE04IBlocked, "RESOLVER_NOT_AUTHORISED", ""},
		{"unknown purpose", reg, "acct-a1", 10, "MARKETING", rmE04IBlocked, "PURPOSE_NOT_PERMITTED", ""},
		{"empty account", reg, "", 10, sig, rmE04IUnknown, "NO_ACCOUNT", ""},
		{"display name is not an account", reg, "مهرداد حافظی", 10, sig, rmE04IUnknown, "NO_VERIFIED_LINK", ""},
		{"phone is not an account", reg, "+989120000000", 10, sig, rmE04IUnknown, "NO_VERIFIED_LINK", ""},
		{"e-mail is not an account", reg, "someone@example.com", 10, sig, rmE04IUnknown, "NO_VERIFIED_LINK", ""},
		{"non-personal account", reg, "acct-bot", 10, sig, rmE04IBlocked, "NON_PERSONAL_ACCOUNT", ""},
		{"shared user", reg, "acct-shared", 10, sig, rmE04IConflict, "SHARED_ACCOUNT", ""},
		{"open finding on account", reg, "acct-finding", 10, sig, rmE04IConflict, "OPEN_FINDING", ""},
		{"open finding on person", reg, "acct-disputed", 10, sig, rmE04IConflict, "OPEN_FINDING", ""},
		{"no link", reg, "acct-unlinked", 10, sig, rmE04IUnknown, "NO_VERIFIED_LINK", ""},
		{"pending link", reg, "acct-pending", 10, sig, rmE04IUnknown, "LINK_PENDING", ""},
		{"overlapping links", reg, "acct-overlap", 10, sig, rmE04IConflict, "OVERLAPPING_LINKS", ""},
		{"revoked before act", reg, "acct-rev", 10, sig, rmE04IUnknown, "LINK_REVOKED", ""},
		{"act before revocation stays verified", reg, "acct-rev", 3, sig, rmE04IVerified, "", "p-rev"},
		{"expired evidence", reg, "acct-exp", 10, sig, rmE04IUnknown, "EVIDENCE_EXPIRED", ""},
		{"purpose not on link", reg, "acct-narrow", 10, sig, rmE04IBlocked, "PURPOSE_NOT_ON_LINK", ""},
		{"act before link", reg, "acct-later", 10, sig, rmE04IUnknown, "NO_VERIFIED_LINK", ""},
		{"back-dated link cannot validate earlier act", reg, "acct-late", 10, sig, rmE04IUnknown, "NO_VERIFIED_LINK", ""},
		{"reassigned account, before", reg, "acct-re", 5, sig, rmE04IVerified, "", "p-old"},
		{"reassigned account, after", reg, "acct-re", 15, sig, rmE04IVerified, "", "p-new"},
		{"merged person resolves to survivor and is flagged", reg, "acct-dup", 10, sig, rmE04IVerified, "MERGED_FLAGGED", "p-survivor"},
		{"split person", reg, "acct-split", 10, sig, rmE04IConflict, "SPLIT_SOURCE", ""},
		{"two accounts one person (1)", reg, "acct-a1", 10, sig, rmE04IVerified, "", "p-a"},
		{"two accounts one person (2)", reg, "acct-a2", 10, sig, rmE04IVerified, "", "p-a"},
	}
	for _, c := range cases {
		r := rmE04ISpecResolve(c.reg, c.account, c.at, c.purpose)
		if r.State != c.state || r.Reason != c.reason || r.PersonID != c.person {
			t.Errorf("%s: got %s/%s/%q, want %s/%s/%q", c.name, r.State, r.Reason, r.PersonID, c.state, c.reason, c.person)
		}
		if r.State == rmE04IVerified && (len(r.EvidenceRefs) == 0 || r.PersonID == c.account) {
			t.Errorf("%s: VERIFIED without evidence reference or with the account id as person", c.name)
		}
		if r.State != rmE04IVerified && r.PersonID != "" {
			t.Errorf("%s: non-VERIFIED outcome leaks a person id", c.name)
		}
	}
}

// rmE04IAdapter feeds the spec resolver into the unchanged RM-E04G evaluator (contract §7). Every non-VERIFIED
// state reaches rmE04GResolve as-is; that function already maps BLOCKED and unknown states to UNKNOWN.
func rmE04IAdapter(reg rmE04IRegistry, at int, purpose string) rmE04GResolver {
	return func(acct string) rmE04GResolution {
		r := rmE04ISpecResolve(reg, acct, at, purpose)
		return rmE04GResolution{PersonID: r.PersonID, State: r.State}
	}
}

func TestRME04IOptionAFeedsExistingEvaluator(t *testing.T) {
	reg := rmE04IFixtureRegistry()
	unauth := reg
	unauth.Authorised = false
	roles := []string{"Product owner", "Sales owner", "Security owner"}
	floor := [][3]string{{"Product owner", "acct-a1", ""}, {"Sales owner", "acct-b", ""}, {"Security owner", "acct-c", ""}}
	with := func(i int, acct, delegator string) [][3]string {
		out := append([][3]string(nil), floor...)
		out[i] = [3]string{out[i][0], acct, delegator}
		return out
	}
	noAccounts := []rmE04GClaim{{Slot: "approver", Role: "Product owner", DisplayName: "x"}, {Slot: "approver", Role: "Sales owner", DisplayName: "y"},
		{Slot: "approver", Role: "Security owner", DisplayName: "z"}, {Slot: "recorder", Role: "Recorder", DisplayName: "x"}, {Slot: "verifier", Role: "T5", DisplayName: "w"}}
	cases := []struct {
		name   string
		reg    rmE04IRegistry
		claims []rmE04GClaim
		want   string
	}{
		{"distinct verified persons (all share one display name)", reg, claims(floor, "acct-r", "acct-v"), indepVerified},
		{"one person counted once: two roles via two accounts", reg, claims(with(1, "acct-a2", ""), "acct-r", "acct-v"), indepNotIndepend},
		{"homonyms independently verified are not rejected", reg, claims([][3]string{{"Product owner", "acct-h1", ""}, {"Sales owner", "acct-h2", ""}, {"Security owner", "acct-c", ""}}, "acct-r", "acct-v"), indepVerified},
		{"recorder cannot self-verify through another account", reg, claims(floor, "acct-r", "acct-r2"), indepNotIndepend},
		{"T5 is an approver", reg, claims(floor, "acct-r", "acct-b"), indepNotIndepend},
		{"delegation to oneself via second account", reg, claims(with(0, "acct-a2", "acct-a1"), "acct-r", "acct-v"), indepNotIndepend},
		{"unlinked participant", reg, claims(with(2, "acct-unlinked", ""), "acct-r", "acct-v"), indepUnknown},
		{"revoked participant", reg, claims(with(2, "acct-rev", ""), "acct-r", "acct-v"), indepUnknown},
		{"shared account participant", reg, claims(with(2, "acct-shared", ""), "acct-r", "acct-v"), indepConflict},
		{"non-personal T5 (automation)", reg, claims(floor, "acct-r", "acct-bot"), indepUnknown},
		{"resolver not authorised", unauth, claims(floor, "acct-r", "acct-v"), indepUnknown},
		{"historical R6 v1 claims carry no account refs", reg, noAccounts, indepUnknown},
	}
	for _, c := range cases {
		got, p := rmE04GIndependence(c.claims, roles, rmE04IAdapter(c.reg, 10, "R6_APPROVAL_SIGNATURE"))
		if got != c.want {
			t.Errorf("%s: %s, want %s (%v)", c.name, got, c.want, p)
		}
	}
}

// ---------------- 5. dependency graph

type rmE04IGraphDoc struct {
	RecordType    string   `json:"record_type"`
	Purpose       string   `json:"purpose"`
	Authoritative bool     `json:"authoritative"`
	Baseline      string   `json:"baseline"`
	EdgeTypes     []string `json:"edge_types"`
	EdgeSources   []string `json:"edge_sources"`
	Nodes         []struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		TrackerID string `json:"tracker_id"`
		Alias     string `json:"alias"`
		State     string `json:"state"`
	} `json:"nodes"`
	Edges []struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Type      string `json:"type"`
		Source    string `json:"source"`
		Condition string `json:"condition"`
		Ref       string `json:"ref"`
	} `json:"edges"`
	UnresolvedConflicts []struct {
		ID     string   `json:"id"`
		Cycle  []string `json:"cycle"`
		Status string   `json:"status"`
		Owner  string   `json:"owner"`
		Ref    string   `json:"ref"`
	} `json:"unresolved_conflicts"`
	RecommendedOrdering struct {
		Status  string   `json:"status"`
		Order   []string `json:"order"`
		Reading string   `json:"reading"`
	} `json:"recommended_ordering_non_authoritative"`
}

func TestRME04IDependencyGraphRecordsCycle(t *testing.T) {
	var g rmE04IGraphDoc
	dec := json.NewDecoder(strings.NewReader(read(t, rmE04IGraph)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&g); err != nil {
		t.Fatal(err)
	}
	if g.Authoritative || !strings.Contains(g.Purpose, "DRAFT — NOT APPROVED") || !strings.HasPrefix(g.RecommendedOrdering.Status, "RECOMMENDATION ONLY") {
		t.Error("dependency graph must be non-authoritative and its ordering a recommendation only")
	}
	nodes, byTracker := map[string]bool{}, map[string]string{}
	for _, n := range g.Nodes {
		if nodes[n.ID] {
			t.Errorf("duplicate node %s", n.ID)
		}
		nodes[n.ID] = true
		if n.TrackerID != "" {
			byTracker[n.TrackerID] = n.ID
		}
	}
	for _, want := range []string{"R5-SEC-01", "D-34", "D-42", "ADR #18", "F-PR1", "F-PR2", "RM-E04G", "RM-E04F", "Group B", "S4", "Group C", "Production"} {
		if !nodes[want] {
			t.Errorf("graph lacks required node %s", want)
		}
	}
	types, sources := map[string]bool{}, map[string]bool{}
	for _, x := range g.EdgeTypes {
		types[x] = true
	}
	for _, x := range g.EdgeSources {
		sources[x] = true
	}
	adj := map[string][]string{}
	has := map[[2]string]bool{}
	for _, e := range g.Edges {
		if !nodes[e.From] || !nodes[e.To] || !types[e.Type] || !sources[e.Source] || e.Ref == "" {
			t.Errorf("malformed edge %+v", e)
		}
		if e.Type == "CONDITIONAL" && e.Condition == "" {
			t.Errorf("conditional edge %s→%s states no condition", e.From, e.To)
		}
		if e.Source == "DOCUMENTED" && e.Type != "CONDITIONAL" {
			adj[e.From] = append(adj[e.From], e.To)
			has[[2]string{e.From, e.To}] = true
		}
	}
	// Fidelity: every tracker prerequisite of a graph decision node appears as a documented edge.
	var tr struct {
		Records []struct {
			ID            string `json:"id"`
			Prerequisites string `json:"prerequisites"`
		} `json:"records"`
	}
	if err := json.Unmarshal([]byte(read(t, "docs/governance/r6-decision-tracker.json")), &tr); err != nil {
		t.Fatal(err)
	}
	for _, r := range tr.Records {
		node, ok := byTracker[r.ID]
		if !ok || strings.TrimSpace(r.Prerequisites) == "" {
			continue
		}
		for _, pre := range strings.Split(r.Prerequisites, ",") {
			pre = strings.TrimSpace(pre)
			if pre == "" || pre == "—" {
				continue
			}
			if !has[[2]string{pre, node}] {
				t.Errorf("tracker prerequisite %s → %s missing from the graph as a DOCUMENTED edge", pre, node)
			}
		}
	}
	// Every node on a documented cycle must be listed in an UNRESOLVED conflict, and every listed cycle must exist.
	reach := func(from, to string) bool {
		seen, stack := map[string]bool{}, []string{from}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, m := range adj[n] {
				if m == to {
					return true
				}
				if !seen[m] {
					seen[m] = true
					stack = append(stack, m)
				}
			}
		}
		return false
	}
	listed := map[string]bool{}
	for _, c := range g.UnresolvedConflicts {
		if c.Status != "UNRESOLVED" {
			t.Errorf("conflict %s is %s; this package may not resolve it", c.ID, c.Status)
		}
		for i, n := range c.Cycle {
			listed[n] = true
			if next := c.Cycle[(i+1)%len(c.Cycle)]; !has[[2]string{n, next}] {
				t.Errorf("conflict %s lists %s→%s, which is not a documented edge", c.ID, n, next)
			}
		}
	}
	var onCycle []string
	for n := range nodes {
		if reach(n, n) {
			onCycle = append(onCycle, n)
			if !listed[n] {
				t.Errorf("%s is on a documented dependency cycle not recorded as UNRESOLVED", n)
			}
		}
	}
	sort.Strings(onCycle)
	if len(onCycle) == 0 {
		t.Log("no documented cycle remains; contract §9 and the graph should record the resolution")
	}
}

// ---------------- 6. authority floor and documents

func TestRME04ISignerFloorPreserved(t *testing.T) {
	want := []string{"HR owner", "Legal owner", "Product owner", "Security owner"}
	got := append([]string(nil), m2Approvers[rmE04HID]...)
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("R5-SEC-01 signer floor changed: %v; Security, HR, Legal and Product must each be independently represented", got)
	}
	reg, _ := rmE04FLoadRegister(t)
	// A provisional NOMINATED CEO nominee (RM-E04Q) is not an assignment; any other status needs evidence.
	if a := reg.AppointingAuthority; a.HolderStatus != "UNASSIGNED" && a.HolderStatus != "NOMINATED" && strings.TrimSpace(a.EvidenceReference) == "" {
		t.Error("CEO appointing authority marked assigned without documentary evidence")
	}
	for _, n := range reg.Nominations {
		if (n.State != "NOMINATED" && n.State != "UNASSIGNED") && (n.AppointmentEvidence == "" || n.AuthorityEvidence == "" || n.VerifiedBy == "") {
			t.Errorf("%s is %s without appointment evidence, authority evidence and an independent verifier", n.AuthorityKey, n.State)
		}
	}
	doc, h := read(t, rmE04IContract), read(t, rmE04IHandoff)
	for _, s := range []string{"Security, HR, Legal **and** Product", "T5 must be independent of every signer and the recorder",
		"CEO appointment authority is a design input that needs documentary proof", "no candidate name substitutes for a signed appointment"} {
		if !strings.Contains(doc, s) {
			t.Errorf("contract §11 no longer states %q", s)
		}
	}
	if !strings.Contains(h, "minimum five distinct humans") || !strings.Contains(h, "no candidate name substitutes for a signed appointment") {
		t.Error("handoff does not state the five-human floor and the appointment rule")
	}
}

var rmE04IFinalStatus = []string{
	"R5-SEC-01   = PREFERRED_A / NOT_APPROVED",
	"F1          = INTERIM_GUARD_ONLY",
	"Group B     = BLOCKED",
	"S4          = BLOCKED",
	"Real grants = BLOCKED",
	"Production  = NO-GO",
}

func TestRME04IDocumentsMarkedNotApproved(t *testing.T) {
	notClaim := regexp.MustCompile(`(?i)pending|not (obtained|recorded|claimed)|no .*(sign-off|review)|required|requested`)
	promoted := regexp.MustCompile(`R5-SEC-01\s*=\s*(APPROVED|SELECTED|PREFERRED_A\s*/\s*APPROVED)`)
	for _, f := range rmE04IDocs {
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
			t.Errorf("%s: a choice or checklist item is ticked", f)
		}
		if promoted.MatchString(d) {
			t.Errorf("%s promotes the preference to a decision", f)
		}
		for i, l := range strings.Split(d, "\n") {
			if claimedSignOff.MatchString(l) && !notClaim.MatchString(l) {
				t.Errorf("%s:%d claims a sign-off: %s", f, i+1, strings.TrimSpace(l))
			}
		}
	}
	doc := read(t, rmE04IContract)
	for _, s := range rmE04IFinalStatus {
		if !strings.Contains(doc, s) {
			t.Errorf("contract final status lacks %q", s)
		}
	}
	h := read(t, rmE04IHandoff)
	if !strings.Contains(h, "| Status | **REVIEW PENDING.** |") {
		t.Error("handoff must state REVIEW PENDING until the reviewers record a review themselves")
	}
	rows := 0
	for _, l := range strings.Split(h, "\n") {
		if strings.HasPrefix(l, "| I-RV-") {
			rows++
			if !strings.HasSuffix(strings.TrimSpace(l), "| PENDING |") {
				t.Errorf("review item not PENDING: %s", l)
			}
		}
	}
	if rows < 12 {
		t.Errorf("handoff lists %d review items; expected the full packet", rows)
	}
	for _, r := range []string{"Codex", "Security", "HR", "Legal", "Product"} {
		if !strings.Contains(h, r+" owner") && !strings.Contains(h, r+" (") {
			t.Errorf("handoff does not request a %s review", r)
		}
	}
}
