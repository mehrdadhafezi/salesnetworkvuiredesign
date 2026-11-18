package http_test

// RM-E04B / A4 — NT-22: API operation scope registry guard.
//
// Every operation of api/openapi/openapi.bundled.yaml must have exactly one
// entry in testdata/operation_scope_registry.yaml, and every entry must
// match an operation of the contract (both directions, by operationId,
// method and path). Each entry declares a category, a scope class, the gate
// capability, the feature flag, the subject requirement, the enforcement
// location, NT-07 applicability and test references.
//
// Labels alone are not trusted. The guard compares them with evidence that
// is observed from the composed handlers (the same NIL-dependency harness
// as NT-07, see scope_nt07_test.go):
//
//   - the capability each handler actually asks the guard for (a recording
//     deny-all decider); a label cannot rename or drop it;
//   - whether the operation is served by a business handler at all (a
//     capability-free label on a guarded handler fails);
//   - how each guarded GET answers an allowed ORGANIZATION decision for a
//     subject without a proven user id (own reads must match their
//     declared unproven_user behaviour; organization reads must not filter
//     by the caller);
//   - that referenced tests and enforcement locations exist in the tree.
//
// Capability-free entries (health, session) and NT-07 exemptions must also
// appear, with the same reason, in approvedCapabilityFree below; adding to
// that list is a security decision and needs review.
//
// TestNT22NegativeControls proves each rule fails on drift, using mutated
// in-memory copies of the contract inventory and of the registry. The real
// contract and registry are never edited by the tests.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

const (
	registryFile = "testdata/operation_scope_registry.yaml"
	repoRoot     = "../../.."
)

// ---- registry format -------------------------------------------------------

type regEntry struct {
	OperationID        string   `yaml:"operation_id"`
	Method             string   `yaml:"method"`
	Path               string   `yaml:"path"`
	Category           string   `yaml:"category"`
	Capability         string   `yaml:"capability"`
	FeatureFlag        string   `yaml:"feature_flag"`
	Scope              string   `yaml:"scope"`
	Subject            string   `yaml:"subject"`
	Enforcement        string   `yaml:"enforcement"`
	AlsoRequires       []string `yaml:"also_requires"`
	FieldCapabilities  []string `yaml:"field_capabilities"`
	NT07               string   `yaml:"nt07"`
	ExemptionReason    string   `yaml:"exemption_reason"`
	UnprovenUser       string   `yaml:"unproven_user"`
	RowAuthorization   string   `yaml:"row_authorization"`
	SeparationOfDuties string   `yaml:"separation_of_duties"`
	Tests              []string `yaml:"tests"`
	Governance         []string `yaml:"governance"`
}

type opRegistry struct {
	SchemaVersion int        `yaml:"schema_version"`
	Baseline      string     `yaml:"baseline"`
	Operations    []regEntry `yaml:"operations"`
}

// parseRegistry decodes strictly: an unknown or misspelt key is an error,
// never silently ignored.
func parseRegistry(b []byte) (opRegistry, error) {
	var r opRegistry
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return opRegistry{}, err
	}
	return r, nil
}

func readRegistryBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(registryFile)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func loadRegistry(t *testing.T) opRegistry {
	t.Helper()
	r, err := parseRegistry(readRegistryBytes(t))
	if err != nil {
		t.Fatalf("%s: %v", registryFile, err)
	}
	return r
}

// ---- closed vocabularies ---------------------------------------------------

// categoryRule is what a category implies. A category is a presentation of
// existing behaviour; it grants nothing.
type categoryRule struct {
	method     string
	scopeKind  string // ORGANIZATION, OWN, COMMAND, SESSION_SELF, NOT_APPLICABLE
	capability bool   // a gate capability is required (true) or forbidden (false)
	nt07       string
	subjects   []string
	needsTests bool // NT-07 does not cover it, so explicit test references are required
}

var categories = map[string]categoryRule{
	// Unauthenticated platform probes; no business data.
	"PUBLIC_HEALTH": {method: http.MethodGet, scopeKind: "NOT_APPLICABLE", nt07: "EXEMPT",
		subjects: []string{"NONE"}, needsTests: true},
	// The caller's own session principal / decisions; no module rows.
	"SESSION_READ": {method: http.MethodGet, scopeKind: "SESSION_SELF", nt07: "EXEMPT",
		subjects: []string{"SESSION_PRINCIPAL"}, needsTests: true},
	// Login / logout: changes only the caller's own session.
	"SESSION_COMMAND": {method: http.MethodPost, scopeKind: "SESSION_SELF", nt07: "NOT_A_READ",
		subjects: []string{"SESSION_PRINCIPAL", "ANONYMOUS_CREDENTIAL_EXCHANGE"}, needsTests: true},
	// Capability-guarded organization-wide read (ORGANIZATION decision only).
	"ORG_READ": {method: http.MethodGet, scopeKind: "ORGANIZATION", capability: true, nt07: "REQUIRED",
		subjects: []string{"AUTHENTICATED_PRINCIPAL"}},
	// Capability-guarded reviewer read (Finance review queue): organization
	// rows for a separate reviewer capability, never a submitter's own view.
	"REVIEW_READ": {method: http.MethodGet, scopeKind: "ORGANIZATION", capability: true, nt07: "REQUIRED",
		subjects: []string{"AUTHENTICATED_PRINCIPAL"}},
	// Capability-guarded read whose rows are filtered to the caller by a
	// server-side predicate (the decision is still ORGANIZATION; NT-07 applies).
	"OWN_READ": {method: http.MethodGet, scopeKind: "OWN", capability: true, nt07: "REQUIRED",
		subjects: []string{"AUTHENTICATED_USER_ID"}},
	// Business command: capability, feature flag and its own row / actor
	// predicate in the command service. GET row-scope (NT-07) does not apply.
	"BUSINESS_COMMAND": {method: http.MethodPost, scopeKind: "COMMAND", capability: true, nt07: "NOT_A_READ",
		subjects: []string{"AUTHENTICATED_USER_ID"}, needsTests: true},
}

// ownBases are the verified ownership bases of OWN:<basis> (code evidence in
// docs/remediation/rm-e04b-operation-scope-inventory.md). No Seller policy is
// introduced here: these describe existing predicates only.
var ownBases = map[string]string{
	"commission_holder_user_id":         "commission/query: holder_user_id = caller",
	"fulfillment_service_scope":         "fulfillment/http work: catalog service scope of the caller (not Seller)",
	"hr_request_participant":            "organization/http hrrequest: requester / reviewer / actor = caller",
	"payment_batch_actor_user_id":       "payments/command read: actor_user_id = caller",
	"invoice_seller_and_issuer_user_id": "payments/command ownsTarget: seller_user_id = issued_by_user_id = caller",
}

// commandBases are the row / actor authorization bases of COMMAND:<basis>.
// no_row_predicate records honestly that only the capability and a proven
// user id are checked (no ownership basis exists).
var commandBases = map[string]string{
	"case_current_holder":                  "cases/command: current custody holder only",
	"hr_subject_descendant_of_actor":       "organization/hrrequest: subject is a REPORTS_TO descendant of the actor",
	"hr_request_next_reviewer":             "organization/hrrequest: actor = next_reviewer_user_id",
	"upload_actor_user_id":                 "payments/command upload: bound to the actor",
	"invoice_seller_and_issuer_user_id":    "payments/command evaluateTarget: seller = issuer = actor",
	"fulfillment_supervisor_service_scope": "fulfillment/work: actor in supervisor scope, assignee in executor scope",
	"fulfillment_assignee_executor_scope":  "fulfillment/work: actor is the assignee and in executor scope",
	"no_row_predicate":                     "UNRESOLVED: capability + proven user id only",
}

var unprovenBehaviours = map[string]bool{
	"REFUSED_ROW_SCOPE_DENIED": true, // 403 ROW_SCOPE_DENIED before any query
	"NOT_FOUND_WITHOUT_QUERY":  true, // 404 before any query
	"EMPTY_WITHOUT_QUERY":      true, // 200 without rows, before any query
	"QUERY_BEFORE_IDENTITY":    true, // the data path runs before identity is proven (residual risk)
}

// approvedCapabilityFree is the reviewed list of operations that are NOT
// guarded by a business capability. Each must be registered with exactly
// this category and reason. Adding an entry is a security decision.
var approvedCapabilityFree = map[string]struct{ category, reason string }{
	"getHealthLive":      {"PUBLIC_HEALTH", "public liveness probe; no data"},
	"getHealthReady":     {"PUBLIC_HEALTH", "public readiness probe; no data"},
	"getSession":         {"SESSION_READ", "session-only: the caller's own principal and feature switches"},
	"getEffectiveAccess": {"SESSION_READ", "session-only: the caller's own decisions, from the same evaluator"},
	"loginWithPassword":  {"SESSION_COMMAND", "credential exchange; creates only the caller's own session"},
	"logout":             {"SESSION_COMMAND", "ends only the caller's own session"},
	"logoutAll":          {"SESSION_COMMAND", "ends only the caller's own sessions"},
}

// ---- derived NT-07 classifications (single source) -------------------------

// exemptionsOf returns the NT-07 exemptions (Go handler name → reason) of a
// registry. It refuses an exemption that is not on the approved list, so a
// registry label alone can never remove an operation from NT-07.
func exemptionsOf(r opRegistry) (map[string]string, error) {
	out := map[string]string{}
	for _, e := range r.Operations {
		if e.NT07 != "EXEMPT" {
			continue
		}
		a, ok := approvedCapabilityFree[e.OperationID]
		if !ok || a.category != e.Category || a.reason != e.ExemptionReason || e.Method != http.MethodGet {
			return nil, fmt.Errorf("NT-07 exemption of %s is not approved", e.OperationID)
		}
		out[goID(e.OperationID)] = e.ExemptionReason
	}
	return out, nil
}

// notScopeGuarded are the GET operations exempt from NT-07.
func notScopeGuarded(t *testing.T) map[string]string {
	t.Helper()
	m, err := exemptionsOf(loadRegistry(t))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// ownRowFiltered are the own reads that refuse an unproven user with
// ROW_SCOPE_DENIED before any query (checked by TestNT07MissingIdentity;
// the label itself is checked against observed behaviour by NT-22).
func ownRowFiltered(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, e := range loadRegistry(t).Operations {
		if e.Category == "OWN_READ" && e.UnprovenUser == "REFUSED_ROW_SCOPE_DENIED" {
			out[goID(e.OperationID)] = true
		}
	}
	return out
}

// ---- observed evidence -----------------------------------------------------

type observation struct {
	caps     []string // capabilities asked of the guard under a deny-all decision
	gates    []string // every capability whose denial alone refuses the request, in order
	platform bool     // served by the API server itself, not a business handler
}

type evidence struct {
	observed map[string]observation // operationId → observation
	unproven map[string]string      // operationId (guarded GET) → behaviour
	nt07     map[string]bool        // Go handler name → covered by NT-07 (guardedOps)
	fileHas  func(path string, re *regexp.Regexp) error
}

// serveOp drives one request of any method through h.
func serveOp(h http.Handler, op specOp) (o outcome) {
	defer func() {
		if r := recover(); r != nil {
			o.reached = r
		}
	}()
	var body *strings.Reader
	if op.method == http.MethodGet {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader("{}")
	}
	req := httptest.NewRequest(op.method, op.url, body)
	if op.method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("If-Match", `"1"`)
		req.Header.Set("Idempotency-Key", "synthetic-nt22-key")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	o.status = rec.Code
	var env struct {
		Outcome string   `json:"outcome"`
		Reasons []string `json:"reasons"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	_ = json.Unmarshal(rec.Body.Bytes(), &o.body)
	o.outcome, o.reasons = env.Outcome, env.Reasons
	if o.body == nil && strings.Contains(rec.Body.String(), errNotComposed.Error()) {
		o.body = map[string]any{"_platform": true}
	}
	return o
}

// recordingDeny denies every request and records the capability asked.
type recordingDeny struct {
	mu   sync.Mutex
	caps map[string]bool
}

func (r *recordingDeny) decide(m authorization.Mode, s authorization.Subject, req authorization.Request) authorization.Decision {
	r.mu.Lock()
	r.caps[string(req.Capability)] = true
	r.mu.Unlock()
	return denyAll(m, s, req)
}

func (r *recordingDeny) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.caps))
	for c := range r.caps {
		out = append(out, c)
	}
	sort.Strings(out)
	r.caps = map[string]bool{}
	return out
}

// gateChain discovers every gate capability of one guarded operation: it
// allows the gates found so far and denies everything else; while the
// answer is 403 ACTION_DENIED, the newly denied capability is the next
// gate. A capability whose denial only masks fields is never a gate.
func gateChain(t *testing.T, op specOp, first string) []string {
	gates := []string{first}
	for range 8 {
		allowed := map[string]bool{}
		for _, g := range gates {
			allowed[g] = true
		}
		var mu sync.Mutex
		var denied []string
		d := func(m authorization.Mode, s authorization.Subject, r authorization.Request) authorization.Decision {
			if allowed[string(r.Capability)] {
				return allowWith(authorization.ScopeOrganization)(m, s, r)
			}
			mu.Lock()
			denied = append(denied, string(r.Capability))
			mu.Unlock()
			return denyAll(m, s, r)
		}
		o := serveOp(compose(seamGuard(t, flagOn, withUser, d)), op)
		if o.reached != nil || o.status != http.StatusForbidden || strings.Join(o.reasons, ",") != string(reasoncodes.ActionDenied) {
			return gates
		}
		if len(denied) == 0 {
			t.Fatalf("%s: refused ACTION_DENIED with no denied capability", op.operationID)
		}
		gates = append(gates, denied[len(denied)-1])
	}
	t.Fatalf("%s: gate chain does not terminate: %v", op.operationID, gates)
	return nil
}

func classifyUnproven(o outcome) string {
	switch {
	case o.reached != nil || o.status >= 500:
		return "QUERY_BEFORE_IDENTITY"
	case o.status == http.StatusForbidden && strings.Join(o.reasons, ",") == string(reasoncodes.RowScopeDenied):
		return "REFUSED_ROW_SCOPE_DENIED"
	case o.status == http.StatusNotFound:
		return "NOT_FOUND_WITHOUT_QUERY"
	case o.status == http.StatusOK:
		for _, k := range dataKeys {
			if v, ok := o.body[k]; ok {
				if items, isList := v.([]any); !isList || len(items) > 0 {
					if k != "allowed_actions" && k != "page" {
						return fmt.Sprintf("OTHER(200 with %q)", k)
					}
				}
			}
		}
		return "EMPTY_WITHOUT_QUERY"
	}
	return fmt.Sprintf("OTHER(%d %v)", o.status, o.reasons)
}

func repoFileHas(path string, re *regexp.Regexp) error {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("%s is not a repository-relative path", path)
	}
	b, err := os.ReadFile(filepath.Join(repoRoot, clean))
	if err != nil {
		return err
	}
	if !re.Match(b) {
		return fmt.Errorf("%s has no match for %s", path, re)
	}
	return nil
}

var (
	evidenceOnce sync.Once
	evidenceVal  evidence
)

// observe collects the evidence once per test binary.
func observe(t *testing.T) evidence {
	t.Helper()
	evidenceOnce.Do(func() {
		ops := specOperations(t)
		ev := evidence{observed: map[string]observation{}, unproven: map[string]string{}, nt07: map[string]bool{}, fileHas: repoFileHas}

		rd := &recordingDeny{caps: map[string]bool{}}
		hDeny := compose(seamGuard(t, flagOn, withUser, rd.decide))
		hNoUser := compose(seamGuard(t, flagOn, withoutUser, allowWith(authorization.ScopeOrganization)))
		for _, op := range ops {
			o := serveOp(hDeny, op)
			caps := rd.take()
			_, plat := o.body["_platform"]
			obs := observation{caps: caps, platform: plat}
			if len(caps) == 1 {
				obs.gates = gateChain(t, op, caps[0])
			}
			ev.observed[op.operationID] = obs
			if op.method == http.MethodGet && len(caps) > 0 {
				ev.unproven[op.operationID] = classifyUnproven(serveOp(hNoUser, op))
			}
		}
		for _, op := range guardedOps(t) {
			ev.nt07[op.id] = true
		}
		evidenceVal = ev
	})
	if evidenceVal.observed == nil {
		t.Fatal("NT-22 evidence could not be collected")
	}
	return evidenceVal
}

// ---- the validator ---------------------------------------------------------

var testRef = regexp.MustCompile(`^([A-Za-z0-9_./-]+_test\.go):(Test[A-Za-z0-9_]+)$`)

// validate returns every registry problem, each prefixed with a stable code.
// It never edits its inputs.
func validate(spec []specOp, r opRegistry, ev evidence) []string {
	var p []string
	add := func(code, id, format string, a ...any) {
		p = append(p, fmt.Sprintf("%s %s: %s", code, id, fmt.Sprintf(format, a...)))
	}
	if r.SchemaVersion != 1 {
		add("SCHEMA", "-", "schema_version %d, want 1", r.SchemaVersion)
	}

	specByID := map[string]specOp{}
	for _, op := range spec {
		if _, dup := specByID[op.operationID]; dup {
			add("DUPLICATE", op.operationID, "operationId repeated in the contract")
		}
		specByID[op.operationID] = op
	}
	regByID := map[string]regEntry{}
	for _, e := range r.Operations {
		if _, dup := regByID[e.OperationID]; dup {
			add("DUPLICATE", e.OperationID, "operationId repeated in the registry")
		}
		regByID[e.OperationID] = e
	}
	// OpenAPI → registry.
	for _, op := range spec {
		if _, ok := regByID[op.operationID]; !ok {
			add("UNREGISTERED", op.operationID, "%s %s has no registry entry", op.method, op.path)
		}
	}

	// Registry → OpenAPI, and every entry on its own.
	for _, e := range r.Operations {
		id := e.OperationID
		op, inSpec := specByID[id]
		if !inSpec {
			add("ORPHAN", id, "registry entry has no operation in the contract")
		} else {
			if e.Method != op.method {
				add("METHOD_MISMATCH", id, "registry %q, contract %q", e.Method, op.method)
			}
			if e.Path != op.path {
				add("PATH_MISMATCH", id, "registry %q, contract %q", e.Path, op.path)
			}
		}

		rule, known := categories[e.Category]
		if !known {
			add("UNKNOWN_CATEGORY", id, "%q", e.Category)
			continue
		}
		if e.Method != rule.method {
			add("CATEGORY_METHOD", id, "category %s requires %s, entry is %s", e.Category, rule.method, e.Method)
		}

		// Scope class.
		kind, basis, hasBasis := strings.Cut(e.Scope, ":")
		switch {
		case e.Scope == "":
			add("INVALID_SCOPE", id, "scope is absent")
		case kind != rule.scopeKind:
			add("INVALID_SCOPE", id, "scope %q is not valid for category %s (want %s)", e.Scope, e.Category, rule.scopeKind)
		case kind == "OWN" && (!hasBasis || basis == ""):
			add("OWN_WITHOUT_BASIS", id, "scope %q names no ownership basis", e.Scope)
		case kind == "OWN" && ownBases[basis] == "":
			add("INVALID_SCOPE", id, "unknown ownership basis %q", basis)
		case kind == "COMMAND" && (!hasBasis || commandBases[basis] == ""):
			add("INVALID_SCOPE", id, "unknown command authorization basis %q", basis)
		case kind != "OWN" && kind != "COMMAND" && hasBasis:
			add("INVALID_SCOPE", id, "scope %q takes no basis", e.Scope)
		}

		// Capability: declared, known, and equal to what the handler asks.
		obs, observed := ev.observed[id]
		if rule.capability {
			switch {
			case e.Capability == "":
				add("MISSING_CAPABILITY", id, "category %s requires a capability", e.Category)
			case !identityapi.KnownCapability(identityapi.Capability(e.Capability)):
				add("UNKNOWN_CAPABILITY", id, "%q is not in the capability registry", e.Capability)
			}
			if e.FeatureFlag == "" {
				add("MISSING_FEATURE_FLAG", id, "capability-guarded operation has no feature flag")
			}
			if observed {
				switch {
				case obs.platform || len(obs.caps) == 0:
					add("UNGUARDED_HANDLER", id, "declared %s but the handler asks the guard for no capability", e.Category)
				case len(obs.caps) != 1 || obs.caps[0] != e.Capability:
					add("CAPABILITY_MISMATCH", id, "declared %q, handler asks %v", e.Capability, obs.caps)
				default:
					if declared := append([]string{e.Capability}, e.AlsoRequires...); strings.Join(declared, ",") != strings.Join(obs.gates, ",") {
						add("GATE_MISMATCH", id, "declared gates %v, observed %v", declared, obs.gates)
					}
				}
			}
		} else {
			if e.Capability != "" {
				add("UNEXPECTED_CAPABILITY", id, "category %s carries no capability", e.Category)
			}
			if observed && len(obs.caps) > 0 {
				add("GUARDED_AS_FREE", id, "classified %s but the handler is guarded by %v", e.Category, obs.caps)
			}
			if e.ExemptionReason == "" {
				add("EXEMPTION_WITHOUT_REASON", id, "capability-free entry has no documented reason")
			}
			if a, approved := approvedCapabilityFree[id]; !approved || a.category != e.Category || a.reason != e.ExemptionReason {
				add("UNAPPROVED_EXEMPTION", id, "capability-free classification %s is not on the approved list with this reason", e.Category)
			}
		}
		if !rule.capability && len(e.AlsoRequires) > 0 {
			add("UNEXPECTED_CAPABILITY", id, "category %s carries no gate capability", e.Category)
		}
		for _, c := range append(append([]string(nil), e.AlsoRequires...), e.FieldCapabilities...) {
			if !identityapi.KnownCapability(identityapi.Capability(c)) {
				add("UNKNOWN_CAPABILITY", id, "field capability %q is not in the capability registry", c)
			}
			if c == e.Capability {
				add("UNEXPECTED_CAPABILITY", id, "%q repeats the gate capability", c)
			}
		}
		if e.Category == "SESSION_COMMAND" && e.FeatureFlag == "" {
			add("MISSING_FEATURE_FLAG", id, "session command has no feature flag")
		}

		// Subject requirement.
		okSubject := false
		for _, s := range rule.subjects {
			okSubject = okSubject || s == e.Subject
		}
		if !okSubject {
			add("INVALID_SUBJECT", id, "subject %q is not valid for category %s", e.Subject, e.Category)
		}

		// NT-07 applicability.
		if e.NT07 != rule.nt07 {
			add("NT07_MISMATCH", id, "nt07 %q, category %s requires %q", e.NT07, e.Category, rule.nt07)
		}
		if e.NT07 == "EXEMPT" && e.ExemptionReason == "" {
			add("EXEMPTION_WITHOUT_REASON", id, "NT-07 exemption has no documented reason")
		}
		if e.NT07 != "EXEMPT" && rule.capability && e.ExemptionReason != "" {
			add("UNEXPECTED_FIELD", id, "exemption_reason on a non-exempt operation")
		}
		if e.Method == http.MethodGet && observed && len(obs.caps) > 0 {
			if e.NT07 != "REQUIRED" {
				add("NT07_MISMATCH", id, "capability-guarded GET must be covered by NT-07")
			}
			if ev.nt07 != nil && !ev.nt07[goID(id)] {
				add("NT07_UNCOVERED", id, "capability-guarded GET is not exercised by NT-07")
			}
		}

		// Caller-dependence: own reads must match their observed behaviour,
		// organization reads must not filter by the caller.
		beh, hasBeh := ev.unproven[id]
		if kind == "OWN" {
			if !unprovenBehaviours[e.UnprovenUser] {
				add("INVALID_UNPROVEN_USER", id, "unproven_user %q", e.UnprovenUser)
			} else if hasBeh && beh != e.UnprovenUser {
				add("UNPROVEN_USER_MISMATCH", id, "declared %s, observed %s", e.UnprovenUser, beh)
			}
		} else {
			if e.UnprovenUser != "" {
				add("UNEXPECTED_FIELD", id, "unproven_user applies to OWN_READ only")
			}
			if hasBeh && beh != "QUERY_BEFORE_IDENTITY" {
				add("CALLER_FILTERED_AS_ORG", id, "classified %s but an unproven user gets %s (the handler filters by caller: classify OWN)", e.Category, beh)
			}
		}

		// Commands carry their own authorization contract.
		if e.Category == "BUSINESS_COMMAND" {
			if strings.TrimSpace(e.RowAuthorization) == "" {
				add("MISSING_ROW_AUTHORIZATION", id, "command has no row / actor authorization statement")
			}
			if strings.TrimSpace(e.SeparationOfDuties) == "" {
				add("MISSING_SOD", id, "command has no separation-of-duties statement")
			}
		} else if e.RowAuthorization != "" || e.SeparationOfDuties != "" {
			add("UNEXPECTED_FIELD", id, "row_authorization / separation_of_duties apply to commands only")
		}

		// Enforcement location and test references must exist.
		if e.Enforcement == "" {
			add("MISSING_ENFORCEMENT", id, "no enforcement location")
		} else if ev.fileHas != nil {
			re := regexp.MustCompile(`(?m)^func \(\w+ \*?\w+\) ` + regexp.QuoteMeta(goID(id)) + `\(`)
			if err := ev.fileHas(e.Enforcement, re); err != nil {
				add("ENFORCEMENT_NOT_FOUND", id, "%v", err)
			}
		}
		if rule.needsTests && len(e.Tests) == 0 {
			add("MISSING_TESTS", id, "category %s is not covered by NT-07 and lists no tests", e.Category)
		}
		for _, ref := range e.Tests {
			m := testRef.FindStringSubmatch(ref)
			if m == nil {
				add("TEST_NOT_FOUND", id, "malformed test reference %q", ref)
				continue
			}
			if ev.fileHas != nil {
				if err := ev.fileHas(m[1], regexp.MustCompile(`(?m)^func `+m[2]+`\(t \*testing\.T\)`)); err != nil {
					add("TEST_NOT_FOUND", id, "%v", err)
				}
			}
		}
	}
	sort.Strings(p)
	return p
}

// ---- tests -----------------------------------------------------------------

// TestNT22RegistryMatchesContract is the CI gate: the registry and the
// contract agree in both directions and every entry is valid and consistent
// with the observed handler behaviour.
func TestNT22RegistryMatchesContract(t *testing.T) {
	for _, problem := range validate(specOperations(t), loadRegistry(t), observe(t)) {
		t.Error(problem)
	}
}

// TestNT22Inventory pins the reconciled counts and classification totals.
// A change here is a reviewed change to the inventory, never a side effect.
func TestNT22Inventory(t *testing.T) {
	spec := specOperations(t)
	methods := map[string]int{}
	for _, op := range spec {
		methods[op.method]++
	}
	if len(spec) != 105 || methods[http.MethodGet] != 92 || methods[http.MethodPost] != 13 || len(methods) != 2 {
		t.Errorf("contract inventory %d operations %v, want 105 (92 GET, 13 POST)", len(spec), methods)
	}
	got := map[string]int{}
	for _, e := range loadRegistry(t).Operations {
		got[e.Category]++
	}
	want := map[string]int{"ORG_READ": 79, "OWN_READ": 7, "REVIEW_READ": 2, "PUBLIC_HEALTH": 2, "SESSION_READ": 2,
		"SESSION_COMMAND": 3, "BUSINESS_COMMAND": 10}
	for c := range categories {
		if got[c] != want[c] {
			t.Errorf("category %s: %d entries, want %d", c, got[c], want[c])
		}
	}
	if len(approvedCapabilityFree) != 7 {
		t.Errorf("%d approved capability-free operations, want 7", len(approvedCapabilityFree))
	}
}

// TestNT22CommandsFailClosed: every business command is capability-guarded
// before any data dependency is touched: deny → 403 ACTION_DENIED, the real
// evaluator with the anonymous subject → 403, feature off → 200 without the
// data path. Row / actor predicates live in the command services and are
// covered by the tests each registry entry references.
func TestNT22CommandsFailClosed(t *testing.T) {
	reg := map[string]regEntry{}
	for _, e := range loadRegistry(t).Operations {
		reg[e.OperationID] = e
	}
	anon, err := authorization.NewGuard(flagOn, authorization.Anonymous{})
	if err != nil {
		t.Fatal(err)
	}
	hDeny := compose(seamGuard(t, flagOn, withUser, denyAll))
	hAnon := compose(anon)
	hOff := compose(seamGuard(t, flagOff, withUser, allowWith(authorization.ScopeOrganization)))
	n := 0
	for _, op := range specOperations(t) {
		if op.method == http.MethodGet {
			continue
		}
		e := reg[op.operationID]
		if e.Category != "BUSINESS_COMMAND" {
			if _, ok := approvedCapabilityFree[op.operationID]; !ok {
				t.Errorf("%s: %s command is neither a business command nor an approved session command", op.operationID, op.method)
			}
			continue
		}
		n++
		gop := getOp{id: goID(op.operationID), url: op.url}
		assertRefused(t, gop, serveOp(hDeny, op), reasoncodes.ActionDenied)
		assertRefused(t, gop, serveOp(hAnon, op), reasoncodes.ActionDenied)
		o := serveOp(hOff, op)
		if o.reached != nil || o.status != http.StatusOK {
			t.Errorf("%s: feature off answered %d (data path entered: %v), want 200 without the data path", op.operationID, o.status, o.reached != nil)
		}
		if o.outcome == result.Success.String() {
			t.Errorf("%s: feature off answered SUCCESS", op.operationID)
		}
	}
	if n != 10 {
		t.Errorf("checked %d business commands, want 10", n)
	}
}

// ---- NT-22 negative controls -----------------------------------------------

func cloneRegistry(r opRegistry) opRegistry {
	c := r
	c.Operations = make([]regEntry, len(r.Operations))
	for i, e := range r.Operations {
		e.Tests = append([]string(nil), e.Tests...)
		e.Governance = append([]string(nil), e.Governance...)
		e.FieldCapabilities = append([]string(nil), e.FieldCapabilities...)
		e.AlsoRequires = append([]string(nil), e.AlsoRequires...)
		c.Operations[i] = e
	}
	return c
}

func editEntry(r opRegistry, id string, f func(*regEntry)) opRegistry {
	c := cloneRegistry(r)
	for i := range c.Operations {
		if c.Operations[i].OperationID == id {
			f(&c.Operations[i])
			return c
		}
	}
	panic("nt22: no entry " + id)
}

func dropEntry(r opRegistry, id string) opRegistry {
	c := cloneRegistry(r)
	out := c.Operations[:0]
	for _, e := range c.Operations {
		if e.OperationID != id {
			out = append(out, e)
		}
	}
	c.Operations = out
	return c
}

func editSpec(spec []specOp, f func([]specOp) []specOp) []specOp {
	return f(append([]specOp(nil), spec...))
}

// TestNT22NegativeControls proves each rule fails on drift.
func TestNT22NegativeControls(t *testing.T) {
	spec := specOperations(t)
	reg := loadRegistry(t)
	ev := observe(t)
	if p := validate(spec, reg, ev); len(p) != 0 {
		t.Fatalf("baseline must be clean before the negative controls: %v", p)
	}
	renameSpec := func(from, to string) []specOp {
		return editSpec(spec, func(s []specOp) []specOp {
			for i := range s {
				if s[i].operationID == from {
					s[i].operationID = to
				}
			}
			return s
		})
	}

	cases := []struct {
		name string
		spec []specOp
		reg  opRegistry
		want []string
	}{
		{"synthetic operation added to the contract, not the registry",
			editSpec(spec, func(s []specOp) []specOp {
				return append(s, specOp{operationID: "listNt22Synthetic", method: http.MethodGet, path: "/api/v1/nt22/synthetic"})
			}), reg, []string{"UNREGISTERED listNt22Synthetic"}},
		{"synthetic command added to the contract, not the registry",
			editSpec(spec, func(s []specOp) []specOp {
				return append(s, specOp{operationID: "createNt22Synthetic", method: http.MethodPost, path: "/api/v1/nt22/synthetic"})
			}), reg, []string{"UNREGISTERED createNt22Synthetic"}},
		{"registry entry deleted", spec, dropEntry(reg, "listCases"), []string{"UNREGISTERED listCases"}},
		{"registry operationId changed", spec, editEntry(reg, "listCases", func(e *regEntry) { e.OperationID = "listCasez" }),
			[]string{"UNREGISTERED listCases", "ORPHAN listCasez"}},
		{"contract operationId changed", renameSpec("listCases", "listAllCases"), reg,
			[]string{"UNREGISTERED listAllCases", "ORPHAN listCases"}},
		{"registry path changed", spec, editEntry(reg, "getInvoice", func(e *regEntry) { e.Path = "/api/v1/invoices/{id}" }),
			[]string{"PATH_MISMATCH getInvoice"}},
		{"contract path changed", editSpec(spec, func(s []specOp) []specOp {
			for i := range s {
				if s[i].operationID == "getInvoice" {
					s[i].path = "/api/v2/invoices/{invoice_id}"
				}
			}
			return s
		}), reg, []string{"PATH_MISMATCH getInvoice"}},
		{"registry method changed", spec, editEntry(reg, "listInvoices", func(e *regEntry) { e.Method = http.MethodPost }),
			[]string{"METHOD_MISMATCH listInvoices", "CATEGORY_METHOD listInvoices"}},
		{"duplicate operationId", spec, func() opRegistry {
			c := cloneRegistry(reg)
			c.Operations = append(c.Operations, c.Operations[10])
			return c
		}(), []string{"DUPLICATE " + reg.Operations[10].OperationID}},
		{"bogus capability", spec, editEntry(reg, "listCustomers", func(e *regEntry) { e.Capability = "customers.customer.everything" }),
			[]string{"UNKNOWN_CAPABILITY listCustomers", "CAPABILITY_MISMATCH listCustomers"}},
		{"known but wrong capability (label cannot hide handler behaviour)", spec,
			editEntry(reg, "listCases", func(e *regEntry) { e.Capability = string(identityapi.CapInvoiceRead) }),
			[]string{"CAPABILITY_MISMATCH listCases"}},
		{"second gate dropped from the registry", spec, editEntry(reg, "getInvoiceTotals", func(e *regEntry) { e.AlsoRequires = nil }),
			[]string{"GATE_MISMATCH getInvoiceTotals"}},
		{"second gate invented in the registry", spec, editEntry(reg, "getInvoice", func(e *regEntry) { e.AlsoRequires = []string{string(identityapi.CapPaymentAttemptRead)} }),
			[]string{"GATE_MISMATCH getInvoice"}},
		{"capability missing", spec, editEntry(reg, "getCase", func(e *regEntry) { e.Capability = "" }),
			[]string{"MISSING_CAPABILITY getCase"}},
		{"bogus field capability", spec, editEntry(reg, "listOrganizationPersons", func(e *regEntry) { e.FieldCapabilities = []string{"org.everything.read"} }),
			[]string{"UNKNOWN_CAPABILITY listOrganizationPersons"}},
		{"invalid scope label", spec, editEntry(reg, "listInvoices", func(e *regEntry) { e.Scope = "SELLER_OWN" }),
			[]string{"INVALID_SCOPE listInvoices"}},
		{"scope absent", spec, editEntry(reg, "listInvoices", func(e *regEntry) { e.Scope = "" }),
			[]string{"INVALID_SCOPE listInvoices"}},
		{"OWN entry without ownership basis", spec, editEntry(reg, "listMyCommissionEarnings", func(e *regEntry) { e.Scope = "OWN:" }),
			[]string{"OWN_WITHOUT_BASIS listMyCommissionEarnings"}},
		{"OWN entry with bare OWN", spec, editEntry(reg, "getPaymentBatch", func(e *regEntry) { e.Scope = "OWN" }),
			[]string{"OWN_WITHOUT_BASIS getPaymentBatch"}},
		{"OWN entry with an invented basis", spec, editEntry(reg, "getPaymentBatch", func(e *regEntry) { e.Scope = "OWN:seller_hierarchy" }),
			[]string{"INVALID_SCOPE getPaymentBatch"}},
		{"unknown category", spec, editEntry(reg, "listInvoices", func(e *regEntry) { e.Category = "SELLER_READ" }),
			[]string{"UNKNOWN_CATEGORY listInvoices"}},
		{"protected operation classified NOT_APPLICABLE", spec, editEntry(reg, "listCases", func(e *regEntry) {
			e.Category, e.Scope, e.Capability, e.FeatureFlag, e.Subject = "PUBLIC_HEALTH", "NOT_APPLICABLE", "", "", "NONE"
			e.NT07, e.ExemptionReason, e.Tests = "EXEMPT", "no data", []string{"internal/platform/apiserver/server_test.go:TestHealth"}
		}), []string{"GUARDED_AS_FREE listCases", "UNAPPROVED_EXEMPTION listCases", "NT07_MISMATCH listCases"}},
		{"operation excluded from NT-07", spec, editEntry(reg, "listInvoices", func(e *regEntry) {
			e.NT07, e.ExemptionReason = "EXEMPT", "not needed"
		}), []string{"NT07_MISMATCH listInvoices"}},
		{"exemption without documented reason", spec, editEntry(reg, "getHealthLive", func(e *regEntry) { e.ExemptionReason = "" }),
			[]string{"EXEMPTION_WITHOUT_REASON getHealthLive"}},
		{"exemption reason changed without approval", spec, editEntry(reg, "getSession", func(e *regEntry) { e.ExemptionReason = "anything" }),
			[]string{"UNAPPROVED_EXEMPTION getSession"}},
		{"command treated as a scope-free session endpoint", spec, editEntry(reg, "recordContactOutcome", func(e *regEntry) {
			e.Category, e.Scope, e.Capability, e.Subject, e.ExemptionReason = "SESSION_COMMAND", "SESSION_SELF", "", "SESSION_PRINCIPAL", "harmless"
			e.RowAuthorization, e.SeparationOfDuties = "", ""
		}), []string{"GUARDED_AS_FREE recordContactOutcome", "UNAPPROVED_EXEMPTION recordContactOutcome"}},
		{"command treated as a public health endpoint", spec, editEntry(reg, "submitPaymentBatch", func(e *regEntry) {
			e.Category, e.Scope, e.Capability = "PUBLIC_HEALTH", "NOT_APPLICABLE", ""
		}), []string{"CATEGORY_METHOD submitPaymentBatch", "GUARDED_AS_FREE submitPaymentBatch", "UNAPPROVED_EXEMPTION submitPaymentBatch"}},
		{"command treated as an organization read", spec, editEntry(reg, "submitPaymentBatch", func(e *regEntry) {
			e.Category, e.Scope, e.NT07 = "ORG_READ", "ORGANIZATION", "REQUIRED"
		}), []string{"CATEGORY_METHOD submitPaymentBatch"}},
		{"command with an unknown authorization basis", spec, editEntry(reg, "decideHRRequest", func(e *regEntry) { e.Scope = "COMMAND:anyone" }),
			[]string{"INVALID_SCOPE decideHRRequest"}},
		{"command without row authorization", spec, editEntry(reg, "decideHRRequest", func(e *regEntry) { e.RowAuthorization = "" }),
			[]string{"MISSING_ROW_AUTHORIZATION decideHRRequest"}},
		{"command without separation-of-duties statement", spec, editEntry(reg, "decideHRRequest", func(e *regEntry) { e.SeparationOfDuties = "" }),
			[]string{"MISSING_SOD decideHRRequest"}},
		{"command without test references", spec, editEntry(reg, "stagePaymentUpload", func(e *regEntry) { e.Tests = nil }),
			[]string{"MISSING_TESTS stagePaymentUpload"}},
		{"command without feature flag", spec, editEntry(reg, "stagePaymentUpload", func(e *regEntry) { e.FeatureFlag = "" }),
			[]string{"MISSING_FEATURE_FLAG stagePaymentUpload"}},
		{"reference to a test that does not exist", spec, editEntry(reg, "createHRRequest", func(e *regEntry) {
			e.Tests = []string{"internal/integration/legacy/shadow/writes_e2e_test.go:TestNT22DoesNotExist"}
		}), []string{"TEST_NOT_FOUND createHRRequest"}},
		{"enforcement location that does not hold the handler", spec, editEntry(reg, "listCases", func(e *regEntry) {
			e.Enforcement = "internal/billing/http/http.go"
		}), []string{"ENFORCEMENT_NOT_FOUND listCases"}},
		{"own read relabelled as organization read", spec, editEntry(reg, "listMyCommissionEarnings", func(e *regEntry) {
			e.Category, e.Scope, e.Subject, e.UnprovenUser = "ORG_READ", "ORGANIZATION", "AUTHENTICATED_PRINCIPAL", ""
		}), []string{"CALLER_FILTERED_AS_ORG listMyCommissionEarnings"}},
		{"organization read relabelled as own read", spec, editEntry(reg, "listCases", func(e *regEntry) {
			e.Category, e.Scope, e.Subject, e.UnprovenUser = "OWN_READ", "OWN:commission_holder_user_id", "AUTHENTICATED_USER_ID", "REFUSED_ROW_SCOPE_DENIED"
		}), []string{"UNPROVEN_USER_MISMATCH listCases"}},
		{"residual risk R-2 hidden by a label", spec, editEntry(reg, "getFulfillmentWorkCase", func(e *regEntry) {
			e.UnprovenUser = "REFUSED_ROW_SCOPE_DENIED"
		}), []string{"UNPROVEN_USER_MISMATCH getFulfillmentWorkCase"}},
		{"invalid subject", spec, editEntry(reg, "listCases", func(e *regEntry) { e.Subject = "ANYONE" }),
			[]string{"INVALID_SUBJECT listCases"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := validate(c.spec, c.reg, ev)
			if len(p) == 0 {
				t.Fatal("drift was not detected")
			}
			joined := strings.Join(p, "\n")
			for _, w := range c.want {
				if !strings.Contains(joined, w+":") {
					t.Errorf("want %q, got:\n%s", w, joined)
				}
			}
		})
	}

	t.Run("protected operation removed from NT-07 by an exemption label", func(t *testing.T) {
		r := editEntry(reg, "listInvoices", func(e *regEntry) { e.NT07, e.ExemptionReason = "EXEMPT", "not needed" })
		if _, err := exemptionsOf(r); err == nil {
			t.Fatal("an unapproved NT-07 exemption was accepted; NT-07 would skip listInvoices")
		}
	})
	t.Run("NT-07 coverage gap", func(t *testing.T) {
		gap := ev
		gap.nt07 = map[string]bool{}
		for k, v := range ev.nt07 {
			gap.nt07[k] = v
		}
		delete(gap.nt07, "ListCases")
		if p := strings.Join(validate(spec, reg, gap), "\n"); !strings.Contains(p, "NT07_UNCOVERED listCases:") {
			t.Fatalf("an NT-07 coverage gap was not detected:\n%s", p)
		}
	})
	t.Run("entry deleted from the committed file bytes", func(t *testing.T) {
		b := readRegistryBytes(t)
		start := bytes.Index(b, []byte("  - operation_id: listInvoices\n"))
		if start < 0 {
			t.Fatal("fixture anchor missing")
		}
		end := bytes.Index(b[start+1:], []byte("\n  - operation_id: "))
		mutated := append(append([]byte(nil), b[:start]...), b[start+1+end+1:]...)
		r, err := parseRegistry(mutated)
		if err != nil {
			t.Fatal(err)
		}
		if p := strings.Join(validate(spec, r, ev), "\n"); !strings.Contains(p, "UNREGISTERED listInvoices:") {
			t.Fatalf("deleted entry not detected:\n%s", p)
		}
	})
	t.Run("unknown registry key", func(t *testing.T) {
		b := bytes.Replace(readRegistryBytes(t), []byte("    scope: ORGANIZATION\n"), []byte("    scopes: ORGANIZATION\n"), 1)
		if _, err := parseRegistry(b); err == nil {
			t.Fatal("a misspelt registry key was silently ignored")
		}
	})
	t.Run("schema version", func(t *testing.T) {
		r := cloneRegistry(reg)
		r.SchemaVersion = 2
		if p := strings.Join(validate(spec, r, ev), "\n"); !strings.Contains(p, "SCHEMA -:") {
			t.Fatalf("schema drift not detected:\n%s", p)
		}
	})
}
