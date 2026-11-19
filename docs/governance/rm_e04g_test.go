package governance

// RM-E04G — F1 person-identity independence hardening (SAFE_NOW scope only).
//
// F1 (RM-E04F §2.3): the R6 validator keys approver / recorder / T5-verifier independence by a normalised display
// name (personKey). A name is presentation data: it cannot prove that two entries are the same human, nor that they
// are different humans.
//
// This file:
//   1. reproduces F1 against the unchanged R6 validator (TestRME04GReproduceF1*), recording the exact current outcome
//      of each case so that a future validator change is a visible, reviewed change;
//   2. defines a pure person-keyed independence evaluator (rmE04GIndependence) that never reads display names and
//      fails closed on UNKNOWN / CONFLICT / SHARED_USER / incomplete resolution. It takes the account→person resolver
//      as a parameter: it does NOT choose the R5-SEC-01 source of truth;
//   3. binds the repository to rmE04GRepositoryResolver, which resolves nothing (R5-SEC-01 is not APPROVED_VERIFIED
//      and no reviewed resolver exists), and to rmE04GClaimsFromRecord, which carries no account or person evidence
//      (R6 v1 records have none). Together they make every M2 record UNKNOWN for person independence;
//   4. adds the interim Group B identity-independence readiness rule (rmE04GGroupBReadiness): Group B readiness needs
//      the existing RM-E04C gate AND person independence of both M2 records. Scope: the Group B gate only (S4 lives
//      inside it). It does not block recording, verifying or tracking any R6 decision;
//   5. refuses identity-shaped (or any other unknown) fields in committed approval records, because loadApproval
//      silently drops unknown JSON keys and appendOnlyChange therefore cannot see them (finding F1-I).
//
// Replacing rmE04GRepositoryResolver or rmE04GClaimsFromRecord is the only way to relax the interim rule; that is a
// separately reviewed change after R5-SEC-01 (roadmap F-PR1 / F-PR2). Nothing here creates an approval, a person id,
// an account link, a grant or a gate change. Fixtures are synthetic and built in test code only.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ---------------- resolution vocabulary (AGENTS.md outcome vocabulary; RM-E04F §6.1)

const (
	idVerified   = "VERIFIED"
	idUnknown    = "UNKNOWN"
	idConflict   = "CONFLICT"
	idSharedUser = "SHARED_USER" // organization.employment.user_link_state; always treated as CONFLICT

	indepVerified    = "PERSON_INDEPENDENCE_VERIFIED"
	indepUnknown     = "PERSON_INDEPENDENCE_UNKNOWN"
	indepConflict    = "PERSON_INDEPENDENCE_CONFLICT"
	indepNotIndepend = "PERSON_INDEPENDENCE_VIOLATED"
)

type rmE04GResolution struct {
	PersonID string
	State    string
}

// rmE04GResolver maps an account reference to a person. Only a reviewed R5-SEC-01 implementation may return
// VERIFIED in the repository.
type rmE04GResolver func(accountRef string) rmE04GResolution

// rmE04GClaim is one participant of a decision. DisplayName is carried for presentation only and is never read by
// the evaluator (TestRME04GDisplayNameIsNotEvidence).
type rmE04GClaim struct {
	Slot                string // "approver", "recorder", "verifier"
	Role                string
	DisplayName         string
	AccountRef          string
	DelegatorAccountRef string // set when an approver signs under a delegation
}

// rmE04GRepositoryResolver is the resolver the repository uses today. R5-SEC-01 (authoritative person identity) is
// not APPROVED_VERIFIED and no reviewed account→person source exists, so every account is UNKNOWN. In particular a
// display name, phone number, e-mail, GitHub login or PR reviewer is never resolved to a person.
func rmE04GRepositoryResolver(string) rmE04GResolution { return rmE04GResolution{State: idUnknown} }

// rmE04GClaimsFromRecord adapts an R6 v1 record. R6 v1 records carry names and roles only — no account or person
// reference — so every claim has an empty AccountRef and resolves UNKNOWN. Adding identity fields to R6 records
// requires a reviewed schema change (RQ-13; F-PR1), not this file.
func rmE04GClaimsFromRecord(rec approvalRecord) []rmE04GClaim {
	var c []rmE04GClaim
	for _, a := range rec.Approvers {
		c = append(c, rmE04GClaim{Slot: "approver", Role: a.Role, DisplayName: a.Name})
	}
	c = append(c, rmE04GClaim{Slot: "recorder", Role: rec.RecordedBy.Role, DisplayName: rec.RecordedBy.Name})
	if h := rec.HumanVerification; h != nil {
		c = append(c, rmE04GClaim{Slot: "verifier", Role: h.VerifierRole, DisplayName: h.VerifierName})
	}
	return c
}

func rmE04GResolve(account string, resolve rmE04GResolver) rmE04GResolution {
	if strings.TrimSpace(account) == "" {
		return rmE04GResolution{State: idUnknown}
	}
	r := resolve(account)
	switch r.State {
	case idVerified:
		if strings.TrimSpace(r.PersonID) == "" {
			return rmE04GResolution{State: idUnknown} // incomplete proof is not proof
		}
		return r
	case idConflict, idSharedUser:
		return rmE04GResolution{State: idConflict}
	default:
		return rmE04GResolution{State: idUnknown}
	}
}

// rmE04GIndependence evaluates RM-E04G safety rules 1–9 for one decision version. Only PERSON_INDEPENDENCE_VERIFIED
// may satisfy a gate. Precedence: CONFLICT > UNKNOWN > VIOLATED > VERIFIED (a conflict is never hidden by a
// violation found elsewhere).
func rmE04GIndependence(claims []rmE04GClaim, requiredRoles []string, resolve rmE04GResolver) (string, []string) {
	var p []string
	conflict, unknown, violated := false, false, false
	note := func(slot string, i int, what string, r rmE04GResolution) {
		switch r.State {
		case idConflict:
			conflict = true
			p = append(p, fmt.Sprintf("%s %d: %s identity CONFLICT", slot, i, what))
		case idUnknown:
			unknown = true
			p = append(p, fmt.Sprintf("%s %d: %s identity UNKNOWN", slot, i, what))
		}
	}
	approverPersons := map[string]string{} // person -> role that reserved it
	roleCoveredBy := map[string]string{}
	var recorder, verifier *rmE04GResolution
	recorders, verifiers := 0, 0
	for i, c := range claims {
		switch c.Slot {
		case "approver":
			signer := rmE04GResolve(c.AccountRef, resolve)
			note("approver", i, "signer", signer)
			principal := signer
			if c.DelegatorAccountRef != "" {
				principal = rmE04GResolve(c.DelegatorAccountRef, resolve)
				note("approver", i, "delegating principal", principal)
				if signer.State == idVerified && principal.State == idVerified && signer.PersonID == principal.PersonID {
					violated = true
					p = append(p, fmt.Sprintf("approver %d: delegation to oneself", i))
				}
			}
			// Rule 8 / I4: a delegated approval reserves both the delegate and the delegator.
			for _, r := range []rmE04GResolution{signer, principal} {
				if r.State != idVerified {
					continue
				}
				if prev, dup := approverPersons[r.PersonID]; dup && prev != c.Role+"#"+fmt.Sprint(i) {
					violated = true
					p = append(p, fmt.Sprintf("approver %d (%s): person already counted for %s", i, c.Role, strings.SplitN(prev, "#", 2)[0]))
				}
				approverPersons[r.PersonID] = c.Role + "#" + fmt.Sprint(i)
			}
			if signer.State == idVerified && principal.State == idVerified {
				if _, seen := roleCoveredBy[c.Role]; !seen {
					roleCoveredBy[c.Role] = principal.PersonID
				}
			}
		case "recorder":
			recorders++
			r := rmE04GResolve(c.AccountRef, resolve)
			note("recorder", i, "recorder", r)
			recorder = &r
		case "verifier":
			verifiers++
			r := rmE04GResolve(c.AccountRef, resolve)
			note("verifier", i, "T5 verifier", r)
			verifier = &r
		default:
			unknown = true
			p = append(p, fmt.Sprintf("claim %d: unknown slot %q", i, c.Slot))
		}
	}
	if recorders != 1 {
		unknown = true
		p = append(p, fmt.Sprintf("want exactly one recorder, found %d", recorders))
	}
	if verifiers != 1 {
		unknown = true
		p = append(p, fmt.Sprintf("want exactly one T5 verifier, found %d", verifiers))
	}
	// Rule 2: every required role satisfied by a distinct verified person.
	distinct := map[string]bool{}
	for _, role := range requiredRoles {
		pid, ok := roleCoveredBy[role]
		if !ok {
			unknown = true
			p = append(p, "no person-resolved approval for required role "+role)
			continue
		}
		if distinct[pid] {
			violated = true
			p = append(p, "required role "+role+" satisfied by a person already counted")
		}
		distinct[pid] = true
	}
	// Rule 5: verifier differs from every approver person (signer and principal) and from the recorder.
	if verifier != nil && verifier.State == idVerified {
		if _, clash := approverPersons[verifier.PersonID]; clash {
			violated = true
			p = append(p, "T5 verifier is the same person as an approver or delegating principal")
		}
		if recorder != nil && recorder.State == idVerified && recorder.PersonID == verifier.PersonID {
			violated = true
			p = append(p, "T5 verifier is the same person as the recorder")
		}
	}
	switch {
	case conflict:
		return indepConflict, p
	case unknown:
		return indepUnknown, p
	case violated:
		return indepNotIndepend, p
	}
	return indepVerified, nil
}

// rmE04GGroupBReadiness: the interim fail-closed Group B rule. Ready only when the existing RM-E04C gate is open AND
// both M2 records are person-independent under the given claims source and resolver. The repository passes
// rmE04GClaimsFromRecord and rmE04GRepositoryResolver.
func rmE04GGroupBReadiness(g rmE04CGate, recs map[string]trackerRecord, titles map[string]string, load approvalLoader, inRepo bool,
	claimsOf func(approvalRecord) []rmE04GClaim, resolve rmE04GResolver) (bool, []string) {
	open, p := rmE04CGateState(g, recs, titles, load, inRepo)
	if !open {
		return false, p
	}
	for _, id := range []string{g.M2Record1ID, g.M2Record2ID} {
		rec, err := load(recs[id].ApprovalRecord)
		if err != nil {
			p = append(p, fmt.Sprintf("%s: %v", id, err))
			continue
		}
		outcome, ip := rmE04GIndependence(claimsOf(rec), rmE04CGateFloorRoles, resolve)
		if outcome != indepVerified {
			p = append(p, fmt.Sprintf("%s: %s — Group B needs verified person independence, not names (F1)", id, outcome))
			for _, x := range ip {
				p = append(p, id+": "+x)
			}
		}
	}
	return len(p) == 0, p
}

// ---------------- F1-I: committed approval records may carry only R6 v1 fields

var rmE04GR6V1Fields = map[string]map[string]bool{
	"": {"record_type": true, "decision_id": true, "decision_version": true, "outcome": true, "selected_option": true,
		"decision_text": true, "initiator": true, "approvers": true, "independent_approvals": true, "notice_date": true,
		"deferral_date": true, "decision_date": true, "effective_date": true, "evidence_source": true, "immutable_ref": true,
		"supersedes": true, "conditions": true, "deferral_reason": true, "revisit_condition": true, "revisit_date": true,
		"recorded_by": true, "human_verification": true,
		"_comment": true}, // present in every template / intake blank; a human copying a template must not be refused
	"initiator":                      {"name": true, "role": true, "authority_evidence": true},
	"approvers[]":                    {"name": true, "role": true, "authority_evidence": true},
	"initiator.authority_evidence":   {"type": true, "reference": true},
	"approvers[].authority_evidence": {"type": true, "reference": true},
	"evidence_source":                {"type": true, "reference": true},
	"recorded_by":                    {"name": true, "role": true},
	"human_verification":             {"verifier_name": true, "verifier_role": true, "method": true, "date": true, "result": true},
}

// rmE04GUnknownFieldProblems lists every key that the R6 v1 schema does not define. loadApproval / appendOnlyChange
// drop such keys silently, so an identity claim (person_ref, verifier_account, …) could be added or rewritten in a
// committed record without any check noticing. Identity fields need a reviewed schema change first (RQ-13).
func rmE04GUnknownFieldProblems(raw []byte) []string {
	var v any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return []string{"not JSON: " + err.Error()}
	}
	var p []string
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			allowed, known := rmE04GR6V1Fields[path]
			if !known {
				p = append(p, fmt.Sprintf("%s: object not allowed by the R6 v1 schema", path))
				return
			}
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if !allowed[k] {
					p = append(p, fmt.Sprintf("field %q not in the R6 v1 schema (identity fields need a reviewed schema change)", strings.TrimPrefix(path+"."+k, ".")))
					continue
				}
				child := k
				if path != "" {
					child = path + "." + k
				}
				walk(child, x[k])
			}
		case []any:
			for _, e := range x {
				walk(path+"[]", e)
			}
		}
	}
	walk("", v)
	return p
}

// ---------------- synthetic fixtures (test code only; never written to approvals/)

// A synthetic Persian name and its Latin transliteration ("test sample"): one fictional person, two spellings.
const (
	fxFaName = "نمونه آزمایشی"
	fxLaName = "Nemooneh Azmayeshi"
)

func rmE04GRecord(t *testing.T, approvers [][2]string, recorder, verifier string) approvalRecord {
	t.Helper()
	signers := map[string]string{}
	for _, a := range approvers {
		signers[a[0]] = a[1]
	}
	return rmE04FSignedDecision(t, signers, recorder, verifier)
}

// fxDirectory is a synthetic account→person directory standing in for a future reviewed R5-SEC-01 resolver.
func fxDirectory(m map[string]rmE04GResolution) rmE04GResolver {
	return func(acct string) rmE04GResolution {
		if r, ok := m[acct]; ok {
			return r
		}
		return rmE04GResolution{State: idUnknown}
	}
}

func fxV(pid string) rmE04GResolution { return rmE04GResolution{PersonID: pid, State: idVerified} }

var fxDir = fxDirectory(map[string]rmE04GResolution{
	"acct-a1": fxV("person-A"), "acct-a2": fxV("person-A"), // two accounts, one human
	"acct-b": fxV("person-B"), "acct-c": fxV("person-C"), "acct-r": fxV("person-R"), "acct-r2": fxV("person-R"),
	"acct-v": fxV("person-V"), "acct-d": fxV("person-D"),
	"acct-h1": fxV("person-H1"), "acct-h2": fxV("person-H2"), // homonyms: same display name, different people
	"acct-shared":     {State: idSharedUser},
	"acct-conflict":   {State: idConflict},
	"acct-unlinked":   {State: idUnknown},
	"acct-incomplete": {State: idVerified}, // VERIFIED without a person id
})

func claims(approvers [][3]string, recorder, verifier string) []rmE04GClaim {
	var c []rmE04GClaim
	for _, a := range approvers { // role, account, delegator account
		c = append(c, rmE04GClaim{Slot: "approver", Role: a[0], DisplayName: "Same Display Name", AccountRef: a[1], DelegatorAccountRef: a[2]})
	}
	c = append(c, rmE04GClaim{Slot: "recorder", Role: "Recorder", DisplayName: "Same Display Name", AccountRef: recorder})
	if verifier != "" {
		c = append(c, rmE04GClaim{Slot: "verifier", Role: "Internal audit", DisplayName: "Same Display Name", AccountRef: verifier})
	}
	return c
}

var floor3 = [][3]string{{"Product owner", "acct-a1", ""}, {"Sales owner", "acct-b", ""}, {"Security owner", "acct-c", ""}}

func withApprover(i int, acct, delegator string) [][3]string {
	out := append([][3]string(nil), floor3...)
	out[i] = [3]string{out[i][0], acct, delegator}
	return out
}

// ---------------- Phase 2: F1 reproduction against the UNCHANGED R6 validator
//
// Each case pins today's outcome. "REPRODUCED" = the validator accepts what a person-keyed rule must reject (or
// rejects what it must accept). "NOT REPRESENTABLE" = R6 v1 records carry no account / delegation data, so the
// validator cannot even see the case; it needs a reviewed resolver (REQUIRES_R5_SEC_01).

func TestRME04GReproduceF1WithCurrentValidator(t *testing.T) {
	floor := [][2]string{{"Product owner", "Person A"}, {"Sales owner", "Person B"}, {"Security owner", "Person C"}}
	cases := []struct {
		id, name string
		rec      approvalRecord
		want     string // current validator level, pinned
	}{
		{"F1-A", "same person signs Product (Persian) and Sales (Latin)",
			rmE04GRecord(t, [][2]string{{"Product owner", fxFaName}, {"Sales owner", fxLaName}, {"Security owner", "Person C"}}, "Person R", ""),
			levelRecorded},
		{"F1-A/V", "same, after T5 verification",
			rmE04GRecord(t, [][2]string{{"Product owner", fxFaName}, {"Sales owner", fxLaName}, {"Security owner", "Person C"}}, "Person R", "Person V"),
			levelHumanVerified},
		{"F1-B", "same person, second account, signs under a different spelling (account not in the record)",
			rmE04GRecord(t, [][2]string{{"Product owner", "Person A"}, {"Sales owner", "P. A."}, {"Security owner", "Person C"}}, "Person R", "Person V"),
			levelHumanVerified},
		{"F1-C", "recorder verifies own decision under another spelling",
			rmE04GRecord(t, floor, fxFaName, fxLaName), levelHumanVerified},
		{"F1-D", "approver verifies the same decision through another account / spelling",
			rmE04GRecord(t, floor, "Person R", "person-a (second account)"), levelHumanVerified},
		{"F1-E", "two different people share one display name (false rejection)",
			rmE04GRecord(t, [][2]string{{"Product owner", "Same Name"}, {"Sales owner", "Same Name"}, {"Security owner", "Person C"}}, "Person R", "Person V"),
			levelIncomplete},
	}
	for _, c := range cases {
		lvl, p := validateApprovalRecord(c.rec, false)
		if lvl != c.want {
			t.Errorf("%s (%s): current validator now returns %s %v, pinned %s — a validator change must update this reproduction in a reviewed PR",
				c.id, c.name, lvl, p, c.want)
		}
		// None of these is evidence of person independence under the interim rule.
		if out, _ := rmE04GIndependence(rmE04GClaimsFromRecord(c.rec), rmE04CGateFloorRoles, rmE04GRepositoryResolver); out != indepUnknown {
			t.Errorf("%s: name-only record evaluated as %s, want %s", c.id, out, indepUnknown)
		}
	}
	// The R6 v1 adapter never fabricates an account reference (from a name or anything else).
	for _, c := range cases {
		for _, cl := range rmE04GClaimsFromRecord(c.rec) {
			if cl.AccountRef != "" || cl.DelegatorAccountRef != "" {
				t.Errorf("%s: adapter fabricated an account reference %+v", c.id, cl)
			}
		}
	}
	// F1-F / F1-G / F1-H have no representation in an R6 v1 record (no account, link state or delegation field):
	// the struct simply has nowhere to put them. Keep this explicit so a schema change cannot slip in unnoticed.
	b, _ := json.Marshal(approvalRecord{})
	for _, f := range []string{"person", "account", "user_id", "delegat", "link_state"} {
		if strings.Contains(strings.ToLower(string(b)), f) {
			t.Errorf("approvalRecord gained a %q field: R6 identity fields need RQ-13 review and an update of this reproduction", f)
		}
	}
}

// ---------------- Phase 4: person-keyed rule, exercised with a synthetic directory

func TestRME04GPersonIndependenceRules(t *testing.T) {
	cases := []struct {
		id, name string
		c        []rmE04GClaim
		want     string
	}{
		{"CTRL", "three distinct verified people, distinct recorder and verifier", claims(floor3, "acct-r", "acct-v"), indepVerified},
		{"CTRL-I7", "recorder may also be an approver (existing R6 rule I7)", claims(floor3, "acct-a1", "acct-v"), indepVerified},
		{"F1-A", "same person under Product and Sales (accounts a1, a1)", claims(withApprover(1, "acct-a1", ""), "acct-r", "acct-v"), indepNotIndepend},
		{"F1-B", "same person under Product and Sales via two accounts", claims(withApprover(1, "acct-a2", ""), "acct-r", "acct-v"), indepNotIndepend},
		{"F1-C", "recorder verifies through a second account", claims(floor3, "acct-r", "acct-r2"), indepNotIndepend},
		{"F1-D", "approver verifies through a second account", claims(floor3, "acct-r", "acct-a2"), indepNotIndepend},
		{"F1-E", "homonyms (same display name) are distinct people", claims([][3]string{{"Product owner", "acct-h1", ""}, {"Sales owner", "acct-h2", ""}, {"Security owner", "acct-c", ""}}, "acct-r", "acct-v"), indepVerified},
		{"F1-F", "approver account link UNKNOWN", claims(withApprover(2, "acct-unlinked", ""), "acct-r", "acct-v"), indepUnknown},
		{"F1-F/acct", "approver with no account reference", claims(withApprover(2, "", ""), "acct-r", "acct-v"), indepUnknown},
		{"F1-F/rec", "recorder UNKNOWN", claims(floor3, "acct-unlinked", "acct-v"), indepUnknown},
		{"F1-F/ver", "T5 verifier UNKNOWN", claims(floor3, "acct-r", "acct-unlinked"), indepUnknown},
		{"F1-F/inc", "VERIFIED without a person id (incomplete proof)", claims(withApprover(0, "acct-incomplete", ""), "acct-r", "acct-v"), indepUnknown},
		{"F1-G", "approver account CONFLICT", claims(withApprover(1, "acct-conflict", ""), "acct-r", "acct-v"), indepConflict},
		{"F1-G/shared", "approver on a SHARED_USER account", claims(withApprover(1, "acct-shared", ""), "acct-r", "acct-v"), indepConflict},
		{"F1-G/ver", "T5 verifier on a SHARED_USER account", claims(floor3, "acct-r", "acct-shared"), indepConflict},
		{"F1-G/prec", "CONFLICT is not hidden by a violation", claims(withApprover(1, "acct-a2", ""), "acct-r", "acct-shared"), indepConflict},
		{"F1-H", "delegate D signs Sales for A, who also signs Product", claims(withApprover(1, "acct-d", "acct-a1"), "acct-r", "acct-v"), indepNotIndepend},
		{"F1-H/2", "delegate is an approver of another role", claims(withApprover(1, "acct-c", "acct-d"), "acct-r", "acct-v"), indepNotIndepend},
		{"F1-H/3", "T5 verifier is the delegating principal", claims(withApprover(1, "acct-d", "acct-v"), "acct-r", "acct-v"), indepNotIndepend},
		{"F1-H/4", "delegation to oneself through a second account", claims(withApprover(0, "acct-a2", "acct-a1"), "acct-r", "acct-v"), indepNotIndepend},
		{"F1-H/5", "delegating principal UNKNOWN", claims(withApprover(1, "acct-d", "acct-unlinked"), "acct-r", "acct-v"), indepUnknown},
		{"F1-H/ok", "independent delegation (D for B; D and B both unused elsewhere)", claims(withApprover(1, "acct-d", "acct-b"), "acct-r", "acct-v"), indepVerified},
		{"MISS-V", "no T5 verifier", claims(floor3, "acct-r", ""), indepUnknown},
		{"MISS-R", "required Security role missing", claims(floor3[:2], "acct-r", "acct-v"), indepUnknown},
	}
	for _, c := range cases {
		got, p := rmE04GIndependence(c.c, rmE04CGateFloorRoles, fxDir)
		if got != c.want {
			t.Errorf("%s (%s): %s, want %s (%v)", c.id, c.name, got, c.want, p)
		}
		if (got == indepVerified) != (len(p) == 0) {
			t.Errorf("%s: outcome %s inconsistent with problems %v", c.id, got, p)
		}
	}
}

// Rule 9: display names are not evidence. Changing every name (to one value, to all-distinct values, to Persian /
// Latin aliases) never changes an outcome.
func TestRME04GDisplayNameIsNotEvidence(t *testing.T) {
	base := []([]rmE04GClaim){claims(floor3, "acct-r", "acct-v"), claims(withApprover(1, "acct-a2", ""), "acct-r", "acct-v"),
		claims(floor3, "acct-r", "acct-r2"), claims(withApprover(2, "acct-unlinked", ""), "acct-r", "acct-v")}
	renames := []func(i int) string{
		func(int) string { return "" },
		func(i int) string { return fmt.Sprintf("Distinct Person %d", i) },
		func(i int) string { return []string{fxFaName, fxLaName}[i%2] },
	}
	for bi, cs := range base {
		want, _ := rmE04GIndependence(cs, rmE04CGateFloorRoles, fxDir)
		for ri, rn := range renames {
			mod := append([]rmE04GClaim(nil), cs...)
			for i := range mod {
				mod[i].DisplayName = rn(i)
			}
			if got, _ := rmE04GIndependence(mod, rmE04CGateFloorRoles, fxDir); got != want {
				t.Errorf("base %d rename %d: outcome moved %s → %s", bi, ri, want, got)
			}
		}
	}
}

// The repository resolver resolves nothing: not a name, alias, phone, e-mail, GitHub login, bot or PR reviewer.
func TestRME04GRepositoryResolverIsFailClosed(t *testing.T) {
	r, _ := rmE04FLoadRegister(t)
	inputs := []string{"", "acct-a1", fxFaName, fxLaName, "+98 912 000 0000", "someone@example.invalid", "github:reviewer",
		"github-actions[bot]", "codex", "claude", "pr-reviewer", "00000000-0000-0000-0000-000000000000"}
	for _, n := range r.Nominees {
		inputs = append(inputs, n.DisplayName, n.NomineeKey)
	}
	for _, in := range inputs {
		if got := rmE04GResolve(in, rmE04GRepositoryResolver); got.State != idUnknown || got.PersonID != "" {
			t.Errorf("repository resolver resolved %q to %+v", in, got)
		}
	}
	// Even a perfectly independent claim set is UNKNOWN in the repository today.
	if got, _ := rmE04GIndependence(claims(floor3, "acct-r", "acct-v"), rmE04CGateFloorRoles, rmE04GRepositoryResolver); got != indepUnknown {
		t.Errorf("repository evaluates an account-referenced decision as %s", got)
	}
}

// ---------------- Phase 6: interim Group B identity-independence readiness rule

func TestRME04GGroupBReadinessRule(t *testing.T) {
	const r1, r2 = "M2-REC-1-FIXTURE", "M2-REC-2-FIXTURE"
	titles := map[string]string{r1: "Seller first-release capability eligibility", r2: "Seller row-scope (ownership predicate) per destination"}
	at := func(id string) trackerRecord {
		return rmFxTracker(id, "APPROVED_VERIFIED", "approvals/"+id+".json", "VERIFIED")
	}
	recs := map[string]trackerRecord{r1: at(r1), r2: at(r2)}
	gate := rmE04CGate{M2Record1ID: r1, M2Record2ID: r2, ApprovedSellerCapabilities: []string{"cases.own.read"}, ApprovedMeOperations: []string{"listMyCases"}}
	loadFor := func(m map[string]approvalRecord) approvalLoader {
		return func(rel string) (approvalRecord, error) {
			id := strings.TrimSuffix(strings.TrimPrefix(rel, "approvals/"), ".json")
			if r, ok := m[id]; ok {
				return r, nil
			}
			return approvalRecord{}, fmt.Errorf("no record at %s", rel)
		}
	}
	nameOnly := func(id string, signers [][2]string, recorder, verifier string) approvalRecord {
		r := rmE04GRecord(t, signers, recorder, verifier)
		r.DecisionID = id
		return r
	}
	floorNames := [][2]string{{"Product owner", "Person A"}, {"Sales owner", "Person B"}, {"Security owner", "Person C"}}
	aliasNames := [][2]string{{"Product owner", fxFaName}, {"Sales owner", fxLaName}, {"Security owner", "Person C"}}
	good := loadFor(map[string]approvalRecord{r1: nameOnly(r1, floorNames, "Person R", "Person V"), r2: nameOnly(r2, floorNames, "Person R", "Person V")})
	alias := loadFor(map[string]approvalRecord{r1: nameOnly(r1, aliasNames, "Person R", "Person V"), r2: nameOnly(r2, floorNames, fxFaName, fxLaName)})

	// Synthetic claims source: stands in for a future reviewed R6 identity extension.
	accountsOf := func(m map[string][]rmE04GClaim) func(approvalRecord) []rmE04GClaim {
		return func(rec approvalRecord) []rmE04GClaim { return m[rec.DecisionID] }
	}
	independent := accountsOf(map[string][]rmE04GClaim{r1: claims(floor3, "acct-r", "acct-v"), r2: claims(floor3, "acct-r", "acct-v")})
	aliased := accountsOf(map[string][]rmE04GClaim{r1: claims(withApprover(1, "acct-a2", ""), "acct-r", "acct-v"), r2: claims(floor3, "acct-r", "acct-v")})
	t5Self := accountsOf(map[string][]rmE04GClaim{r1: claims(floor3, "acct-r", "acct-v"), r2: claims(floor3, "acct-r", "acct-r2")})
	shared := accountsOf(map[string][]rmE04GClaim{r1: claims(floor3, "acct-r", "acct-v"), r2: claims(withApprover(2, "acct-shared", ""), "acct-r", "acct-v")})

	cases := []struct {
		name     string
		load     approvalLoader
		claimsOf func(approvalRecord) []rmE04GClaim
		resolve  rmE04GResolver
		gateOpen bool // existing RM-E04C gate result (pinned: the interim rule never loosens it)
		ready    bool
	}{
		{"name-only valid approvals, repository binding", good, rmE04GClaimsFromRecord, rmE04GRepositoryResolver, true, false},
		{"name-only valid approvals, even with a resolver", good, rmE04GClaimsFromRecord, fxDir, true, false},
		{"Persian/Latin alias approvals (F1-A, F1-C), repository binding", alias, rmE04GClaimsFromRecord, rmE04GRepositoryResolver, true, false},
		{"account-referenced claims, repository resolver (R5-SEC-01 open)", good, independent, rmE04GRepositoryResolver, true, false},
		{"person-independent claims with a (synthetic) reviewed resolver", good, independent, fxDir, true, true},
		{"two accounts one human under the resolver", good, aliased, fxDir, true, false},
		{"recorder verifies via second account under the resolver", good, t5Self, fxDir, true, false},
		{"SHARED_USER approver under the resolver", good, shared, fxDir, true, false},
	}
	for _, c := range cases {
		if o, _ := rmE04CGateState(gate, recs, titles, c.load, false); o != c.gateOpen {
			t.Fatalf("%s: RM-E04C gate %v, pinned %v", c.name, o, c.gateOpen)
		}
		ready, p := rmE04GGroupBReadiness(gate, recs, titles, c.load, false, c.claimsOf, c.resolve)
		if ready != c.ready {
			t.Errorf("%s: ready=%v, want %v (%v)", c.name, ready, c.ready, p)
		}
	}
	// A forged APPROVED_VERIFIED (no valid record) is refused before identity is even considered.
	forged := map[string]trackerRecord{r1: at(r1), r2: rmFxTracker(r2, "APPROVED_VERIFIED", "", "NOT_STARTED")}
	if ready, _ := rmE04GGroupBReadiness(gate, forged, titles, good, false, independent, fxDir); ready {
		t.Error("forged APPROVED_VERIFIED passed Group B readiness")
	}
	// A closed gate stays closed whatever the identity evidence.
	if ready, _ := rmE04GGroupBReadiness(rmE04CGate{}, recs, titles, good, false, independent, fxDir); ready {
		t.Error("identity evidence opened a closed gate")
	}
	// Repository mode: synthetic records are refused regardless of identity.
	if ready, _ := rmE04GGroupBReadiness(gate, recs, titles, good, true, independent, fxDir); ready {
		t.Error("synthetic records passed Group B readiness in repository mode")
	}
}

// The repository's Group B state under the interim rule: closed, and if the RM-E04C gate were ever opened by
// name-only records the runtime check is evaluated as if it were closed, so any Group B artefact fails CI.
func TestRME04GRepositoryGroupBNeedsPersonIndependence(t *testing.T) {
	g := loadRMEGate(t)
	recs, titles := trackerByID(t), r5Titles(t)
	gateOpen, _ := rmE04CGateState(g, recs, titles, repoLoader, true)
	ready, p := rmE04GGroupBReadiness(g, recs, titles, repoLoader, true, rmE04GClaimsFromRecord, rmE04GRepositoryResolver)
	if gateOpen && !ready {
		t.Errorf("Group B gate is open without verified person independence (F1 interim rule): %v", p)
	}
	if ready {
		t.Error("Group B reports ready while R5-SEC-01 has no reviewed resolver; the interim rule must be relaxed only by a reviewed change")
	}
	if g.M2Record1ID != "" || g.M2Record2ID != "" || len(g.ApprovedSellerCapabilities) > 0 || len(g.ApprovedMeOperations) > 0 {
		t.Error("RM-E04G must not change the Group B gate file")
	}
	if m := loadS4Manifest(t); m.ContractDecisionID != "" {
		t.Error("RM-E04G must not change the S4 manifest")
	}
	if r := recs["R5-SEC-01 (proposed ADR)"]; r.Status != "READY_FOR_REVIEW" || r.ApprovalRecord != "" {
		t.Errorf("R5-SEC-01 tracker entry changed (%s, %q); RM-E04G records no decision", r.Status, r.ApprovalRecord)
	}
}

// ---------------- F1-I: identity fields cannot be smuggled into committed records

func TestRME04GApprovalRecordsCarryOnlyR6V1Fields(t *testing.T) {
	// Calibration: every template and intake blank is accepted (a human copying one is not refused).
	for _, glob := range []string{"docs/governance/templates/*.json", "docs/governance/intake/*/*.json"} {
		files, _ := filepath.Glob(filepath.Join(root, glob))
		if len(files) == 0 {
			t.Fatalf("no files for %s", glob)
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if p := rmE04GUnknownFieldProblems(b); len(p) > 0 {
				t.Errorf("%s: calibration failed: %v", filepath.Base(f), p)
			}
		}
	}
	base, _ := json.Marshal(fxDecision(t, "APPROVED", "VERIFIED"))
	smuggle := map[string]string{
		"approver person_ref":       `"approvers":[{"name":"x","role":"y","person_ref":"p-1","authority_evidence":{"type":"a","reference":"b"}}]`,
		"verifier account":          `"human_verification":{"verifier_name":"v","verifier_role":"r","method":"m","date":"2030-02-02","result":"VERIFIED","verifier_account":"u-2"}`,
		"top-level identity claim":  `"identity_independence":"VERIFIED"`,
		"recorder person id":        `"recorded_by":{"name":"n","role":"r","person_id":"p-9"}`,
		"authority evidence sha256": `"approvers":[{"name":"x","role":"y","authority_evidence":{"type":"a","reference":"b","sha256":"00"}}]`,
	}
	if p := rmE04GUnknownFieldProblems(base); len(p) > 0 {
		t.Fatalf("calibration: a complete R6 v1 record is refused: %v", p)
	}
	for name, frag := range smuggle {
		// Go's decoder keeps the LAST duplicate key, so the smuggled value is the one that must be caught.
		raw := []byte(strings.TrimSuffix(string(base), "}") + `,` + frag + `}`)
		if p := rmE04GUnknownFieldProblems(raw); len(p) == 0 {
			t.Errorf("%s: unknown identity field accepted", name)
		}
		// Demonstrates why the guard is needed: the existing append-only check cannot see the change.
		var rec approvalRecord
		if err := json.Unmarshal(raw, &rec); err == nil && name == "top-level identity claim" {
			if appendOnlyChange(fxDecision(t, "APPROVED", "VERIFIED"), rec) != nil {
				t.Error("calibration: appendOnlyChange unexpectedly saw an unknown field")
			}
		}
	}
	// Repository: every committed approval file, in every committed revision, carries only R6 v1 fields.
	dir := filepath.Join(root, "docs/governance/approvals")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range rmE04GUnknownFieldProblems(b) {
			t.Errorf("%s: %s", e.Name(), x)
		}
	}
}

// F1-I history: no committed revision of any approval file ever carried a field outside R6 v1.
func TestRME04GApprovalHistoryCarriesOnlyR6V1Fields(t *testing.T) {
	requireHistory(t)
	files, _ := git("log", "--diff-filter=AM", "--name-only", "--format=", "--", "docs/governance/approvals")
	for _, f := range strings.Fields(files) {
		revs, _ := git("log", "--format=%H", "--", f)
		for _, rev := range strings.Fields(revs) {
			if body, err := git("show", rev+":"+f); err == nil {
				for _, x := range rmE04GUnknownFieldProblems([]byte(body)) {
					t.Errorf("%s at %s: %s", f, rev[:7], x)
				}
			}
		}
	}
}

// ---------------- documents

var rmE04GDocs = []string{
	"docs/security/rm-e04g-f1-person-identity-independence.md",
	"docs/implementation/rm-e04g-codex-security-review-handoff.md",
}

func TestRME04GDocumentsMarkedNotApproved(t *testing.T) {
	known := ids(read(t, "docs/remediation/r5-decision-inventory.md"))
	for _, r := range loadTracker(t).Records {
		known[strings.SplitN(r.ID, " ", 2)[0]] = true
	}
	idPattern := regexp.MustCompile(`\b(?:D-[0-9]+[a-e]?|OPD-[0-9]+|R5-[A-Z]+-[0-9]+)\b`)
	for _, f := range rmE04GDocs {
		d := read(t, f)
		if head := d[:min(len(d), 600)]; !strings.Contains(head, "DRAFT — NOT APPROVED") {
			t.Errorf("%s: no DRAFT — NOT APPROVED marker near the top", f)
		}
		if !strings.Contains(d, "NO-GO") {
			t.Errorf("%s: Production NO-GO not stated", f)
		}
		if strings.ContainsAny(d, "☑☒✔✓") || strings.Contains(d, "[x]") || strings.Contains(d, "[X]") {
			t.Errorf("%s: a choice is pre-ticked", f)
		}
		for i, l := range strings.Split(d, "\n") {
			if claimedSignOff.MatchString(l) && !regexp.MustCompile(`(?i)pending|not (obtained|recorded|claimed)|no .*(sign-off|review)|required|requested`).MatchString(l) {
				t.Errorf("%s:%d claims a sign-off: %s", f, i+1, strings.TrimSpace(l))
			}
		}
		for _, id := range idPattern.FindAllString(d, -1) {
			if !known[id] && !known[strings.TrimRight(id, "abcde")] {
				t.Errorf("%s cites decision id %s not in the tracker / inventory", f, id)
			}
		}
	}
	h := read(t, rmE04GDocs[1])
	if !strings.Contains(h, "| Status | **REVIEW PENDING.** |") {
		t.Error("the handoff must state REVIEW PENDING until the reviewers record a review themselves")
	}
	for _, l := range strings.Split(h, "\n") {
		if strings.HasPrefix(l, "| RV-") && !strings.HasSuffix(strings.TrimSpace(l), "| PENDING |") {
			t.Errorf("review item not PENDING: %s", l)
		}
	}
}
