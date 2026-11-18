package http_test

// RM-E04B / NT-07 (alias T7; F-02 regression).
//
// Every data-returning GET operation of the OpenAPI contract must refuse an
// allowed authorization decision whose scope the handler does not
// explicitly implement, with 403 UNAUTHORIZED / ROW_SCOPE_DENIED, BEFORE any
// module data query runs.
//
// How "no query executed" is proven: every handler is composed with NIL data
// services (query services, command services, catalog / payments readers,
// write-back query, reporting auditor). Any attempt to read data therefore
// dereferences nil and panics, or returns an error (5xx). The positive
// control (an allowed ORGANIZATION decision) must trip that sentinel on every
// guarded operation, which proves the sentinel sits on the data path; the
// negative cases must answer cleanly without tripping it.
//
// The operation list is read from api/openapi/openapi.bundled.yaml and
// served through the generated router, so a new GET operation is covered
// automatically and fails this test unless it either refuses an unsupported
// scope or is explicitly classified below as not capability-guarded.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	biawinhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/http"
	billinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/http"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	commissionhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/http"
	custodyhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/http"
	customershttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/http"
	financehttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/http"
	fulfillmenthttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	provideropshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops/http"
	writebackhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	organizationhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
	reportinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/http"
)

const bundledSpec = "../../../api/openapi/openapi.bundled.yaml"

// The GET operations exempt from NT-07 (not guarded by a business
// capability) and the own reads that refuse an unproven user before any
// query are no longer hand-maintained here: both are derived from the
// committed operation scope registry (testdata/operation_scope_registry.yaml,
// RM-E04B / A4, NT-22), whose exemptions must also match the approved list
// in operation_registry_test.go and whose unproven-user behaviour is
// observed, not declared. See notScopeGuarded and ownRowFiltered there.
//
// Recorded as RM-E04B remaining risk R-2 (existing behaviour, unchanged):
// GetFulfillmentWorkCase reads the case, then resolves the caller's service
// scope; an unproven user has an empty scope and gets 404, so nothing is
// returned, but the detail query runs before identity is proven (registry
// unproven_user: QUERY_BEFORE_IDENTITY).

// platformStub serves the operations that the API server composes itself.
// They are never reached by NT-07 (classified above).
type platformStub struct{}

var errNotComposed = errors.New("nt07: platform operation not composed in this test")

func (platformStub) GetHealthLive(context.Context, oapi.GetHealthLiveRequestObject) (oapi.GetHealthLiveResponseObject, error) {
	return nil, errNotComposed
}
func (platformStub) GetHealthReady(context.Context, oapi.GetHealthReadyRequestObject) (oapi.GetHealthReadyResponseObject, error) {
	return nil, errNotComposed
}
func (platformStub) GetSession(context.Context, oapi.GetSessionRequestObject) (oapi.GetSessionResponseObject, error) {
	return nil, errNotComposed
}
func (platformStub) LoginWithPassword(context.Context, oapi.LoginWithPasswordRequestObject) (oapi.LoginWithPasswordResponseObject, error) {
	return nil, errNotComposed
}
func (platformStub) Logout(context.Context, oapi.LogoutRequestObject) (oapi.LogoutResponseObject, error) {
	return nil, errNotComposed
}
func (platformStub) LogoutAll(context.Context, oapi.LogoutAllRequestObject) (oapi.LogoutAllResponseObject, error) {
	return nil, errNotComposed
}

// One wrapper per handler type: Go names an embedded field after its type,
// and several packages call theirs Handlers.
type (
	authzH    struct{ *authzhttp.Handlers }
	identityH struct{ *identityhttp.Handlers }
	orgH      struct{ *organizationhttp.Handlers }
	hrH       struct {
		*organizationhttp.HRRequestHandlers
	}
	customersH   struct{ *customershttp.Handlers }
	casesH       struct{ *caseshttp.Handlers }
	casesWriteH  struct{ *caseshttp.WriteHandlers }
	custodyH     struct{ *custodyhttp.Handlers }
	billingH     struct{ *billinghttp.Handlers }
	paymentsH    struct{ *paymentshttp.Handlers }
	paymentsWH   struct{ *paymentshttp.WriteHandlers }
	reportingH   struct{ *reportinghttp.Handlers }
	financeH     struct{ *financehttp.Handlers }
	reviewH      struct{ *financehttp.ReviewHandlers }
	biawinH      struct{ *biawinhttp.Handlers }
	activationH  struct{ *biawinhttp.ActivationHandlers }
	fulfillmentH struct{ *fulfillmenthttp.Handlers }
	workH        struct{ *fulfillmenthttp.WorkHandlers }
	commissionH  struct{ *commissionhttp.Handlers }
	provideropsH struct{ *provideropshttp.Handlers }
	writebackH   struct{ *writebackhttp.Handlers }
)

// composed serves the whole strict interface. Every data dependency is nil.
type composed struct {
	platformStub
	authzH
	identityH
	orgH
	hrH
	customersH
	casesH
	casesWriteH
	custodyH
	billingH
	paymentsH
	paymentsWH
	reportingH
	financeH
	reviewH
	biawinH
	activationH
	fulfillmentH
	workH
	commissionH
	provideropsH
	writebackH
}

var _ oapi.StrictServerInterface = composed{}

func shadowMode(context.Context) (rollout.Mode, error) { return rollout.Shadow, nil }

// compose wires every handler to guard g with NIL data services.
func compose(g *authorization.Guard) http.Handler {
	var mode rollout.Source = shadowMode
	c := composed{
		authzH:       authzH{authzhttp.New(g, nil)},
		identityH:    identityH{identityhttp.New(g, nil)},
		orgH:         orgH{organizationhttp.New(g, nil)},
		hrH:          hrH{organizationhttp.NewHRRequests(g, mode, nil)},
		customersH:   customersH{customershttp.New(g, nil)},
		casesH:       casesH{caseshttp.New(g, nil)},
		casesWriteH:  casesWriteH{caseshttp.NewWrite(g, g, mode, nil)},
		custodyH:     custodyH{custodyhttp.New(g, nil)},
		billingH:     billingH{billinghttp.New(g, nil, nil)},
		paymentsH:    paymentsH{paymentshttp.New(g, nil)},
		paymentsWH:   paymentsWH{paymentshttp.NewWrite(g, mode, nil)},
		reportingH:   reportingH{reportinghttp.New(g, nil, nil, timeutil.FixedClock{})},
		financeH:     financeH{financehttp.New(g, nil)},
		reviewH:      reviewH{financehttp.NewReview(g, nil)},
		biawinH:      biawinH{biawinhttp.New(g, nil)},
		activationH:  activationH{biawinhttp.NewActivation(g, nil)},
		fulfillmentH: fulfillmentH{fulfillmenthttp.New(g, nil)},
		workH:        workH{fulfillmenthttp.NewWork(g, g, mode, nil, nil)},
		commissionH:  commissionH{commissionhttp.New(g, nil)},
		provideropsH: provideropsH{provideropshttp.New(g, g, nil)},
		writebackH:   writebackH{writebackhttp.New(g, nil)},
	}
	return oapi.Handler(oapi.NewStrictHandler(c, nil))
}

// ---- synthetic subjects and decisions ------------------------------------

const synUserID = "00000000-0000-4000-8000-0000000000e4"

type subjectStub struct{ s authorization.Subject }

func (r subjectStub) Subject(context.Context) authorization.Subject { return r.s }

var (
	withUser    = authorization.Subject{PrincipalID: "syn-nt07", UserID: synUserID}
	withoutUser = authorization.Subject{PrincipalID: "syn-nt07"}
)

func flagOn(context.Context) (bool, error)  { return true, nil }
func flagOff(context.Context) (bool, error) { return false, nil }

var errFlagSource = errors.New("nt07: synthetic flag-source failure")

func flagErr(context.Context) (bool, error) { return false, errFlagSource }

// allowWith returns an allowed decision carrying scope s for every request.
func allowWith(s authorization.Scope) authorization.Decider {
	return func(_ authorization.Mode, _ authorization.Subject, r authorization.Request) authorization.Decision {
		mod, _, _ := strings.Cut(string(r.Capability), ".")
		return authorization.Decision{Allowed: true, Outcome: result.Success, Module: mod, Scope: s,
			Matched: []authorization.MatchedGrant{{Role: "synthetic_nt07", Basis: "SYNTHETIC_NT07", Status: identityapi.GrantActive}}}
	}
}

func denyAll(_ authorization.Mode, _ authorization.Subject, r authorization.Request) authorization.Decision {
	mod, _, _ := strings.Cut(string(r.Capability), ".")
	code := reasoncodes.ActionDenied
	return authorization.Decision{Outcome: result.Unauthorized, Reason: &code, Module: mod, Scope: authorization.ScopeNone}
}

func seamGuard(t *testing.T, flag authorization.FlagFunc, s authorization.Subject, d authorization.Decider) *authorization.Guard {
	t.Helper()
	g, err := authorization.NewGuardWithDecider(flag, subjectStub{s}, d)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// ---- the operation inventory ----------------------------------------------

type getOp struct {
	id, path, url string
}

var pathParam = regexp.MustCompile(`\{([a-z_]+)\}`)

// synthetic path / query values: well-formed so request binding passes and
// the handler itself decides.
const synID = "00000000-0000-4000-8000-0000000000a1"

// paramOverride gives a parameter a value that exists in a static
// registry, so the positive control reaches the data path (an unknown
// metric id answers 404 from the in-memory contract registry, no query).
var paramOverride = map[string]string{"metric_id": "cases.staged_total"}

func sample(p *openapi3.Parameter) string {
	if v, ok := paramOverride[p.Name]; ok {
		return v
	}
	if p.Schema == nil || p.Schema.Value == nil {
		return "x"
	}
	v := p.Schema.Value
	switch {
	case len(v.Enum) > 0:
		return fmt.Sprint(v.Enum[0])
	case v.Format == "uuid":
		return synID
	case v.Type != nil && (v.Type.Is("integer") || v.Type.Is("number")):
		return "1"
	case v.Format == "date":
		return "2026-01-01"
	case v.Format == "date-time":
		return "2026-01-01T00:00:00Z"
	}
	return synID
}

// specOp is one operation of the contract, with a request URL whose path
// and required query parameters carry well-formed synthetic values.
type specOp struct {
	operationID, method, path, url string
}

// goID is the Go method name of the strict handler: the operationId with
// its first letter upper-cased.
func goID(operationID string) string {
	return strings.ToUpper(operationID[:1]) + operationID[1:]
}

// specOperations enumerates every operation (paths x methods) of the
// bundled contract. NT-07 and the NT-22 registry guard share this one
// inventory, so the two cannot drift.
func specOperations(t *testing.T) []specOp {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromFile(bundledSpec)
	if err != nil {
		t.Fatal(err)
	}
	var out []specOp
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			params := append(openapi3.Parameters{}, item.Parameters...)
			params = append(params, op.Parameters...)
			url := path
			var q []string
			for _, pr := range params {
				p := pr.Value
				switch p.In {
				case "path":
					url = strings.ReplaceAll(url, "{"+p.Name+"}", sample(p))
				case "query":
					if p.Required {
						q = append(q, p.Name+"="+sample(p))
					}
				}
			}
			if pathParam.MatchString(url) {
				t.Fatalf("%s: unresolved path parameter in %s", op.OperationID, url)
			}
			if len(q) > 0 {
				url += "?" + strings.Join(q, "&")
			}
			out = append(out, specOp{operationID: op.OperationID, method: method, path: path, url: url})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].operationID < out[j].operationID })
	return out
}

func getOperations(t *testing.T) []getOp {
	t.Helper()
	all := specOperations(t)
	var out []getOp
	for _, op := range all {
		if op.method != http.MethodGet {
			continue
		}
		out = append(out, getOp{id: goID(op.operationID), path: op.path, url: op.url})
	}
	if len(all) != 105 {
		t.Errorf("contract has %d operations, the reconciled inventory has 105; re-run the RM-E04B reconciliation", len(all))
	}
	return out
}

// ---- driving one request ----------------------------------------------------

type outcome struct {
	status  int
	outcome string
	reasons []string
	body    map[string]any
	// reached is set when the handler touched a nil data dependency (the
	// sentinel): the data path was entered.
	reached any
}

func serve(h http.Handler, url string) (o outcome) {
	defer func() {
		if r := recover(); r != nil {
			o.reached = r
		}
	}()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	o.status = rec.Code
	var env struct {
		Outcome string   `json:"outcome"`
		Reasons []string `json:"reasons"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	_ = json.Unmarshal(rec.Body.Bytes(), &o.body)
	o.outcome, o.reasons = env.Outcome, env.Reasons
	return o
}

// dataKeys are envelope keys that would carry rows or aggregates.
var dataKeys = []string{"items", "data", "page", "totals", "allowed_actions", "summary", "counts"}

func assertRefused(t *testing.T, op getOp, o outcome, reason reasoncodes.Code) {
	t.Helper()
	if o.reached != nil {
		t.Errorf("%s: data path entered after refusal (sentinel tripped: %v)", op.id, o.reached)
		return
	}
	if o.status != http.StatusForbidden || o.outcome != result.Unauthorized.String() || strings.Join(o.reasons, ",") != string(reason) {
		t.Errorf("%s: got %d %s %v, want 403 UNAUTHORIZED [%s]", op.id, o.status, o.outcome, o.reasons, reason)
	}
	for _, k := range dataKeys {
		if _, ok := o.body[k]; ok {
			t.Errorf("%s: refusal body carries %q", op.id, k)
		}
	}
}

func guardedOps(t *testing.T) []getOp {
	var out []getOp
	skip := notScopeGuarded(t)
	for _, op := range getOperations(t) {
		if _, exempt := skip[op.id]; !exempt {
			out = append(out, op)
		}
	}
	return out
}

// ---- tests ------------------------------------------------------------------

// TestNT07Inventory reconciles the GET inventory with the RM-E04A audit and
// keeps the not-scope-guarded classification explicit.
func TestNT07Inventory(t *testing.T) {
	ops := getOperations(t)
	if len(ops) != 92 {
		t.Errorf("contract has %d GET operations, the RM-E04A inventory has 92", len(ops))
	}
	seen := map[string]bool{}
	for _, op := range ops {
		seen[op.id] = true
	}
	exempt := notScopeGuarded(t)
	for id := range exempt {
		if !seen[id] {
			t.Errorf("classified operation %s is not a GET operation of the contract", id)
		}
	}
	if g := len(ops) - len(exempt); g != 88 {
		t.Errorf("%d capability-guarded GET operations, want 88 (51 previously MISSING + 37 PRESENT)", g)
	}
}

// TestNT07PositiveControl: an allowed ORGANIZATION decision proceeds to the
// data path of every guarded GET operation (the existing supported read is
// not refused), which also proves the nil-dependency sentinel is on the
// data path of each operation.
func TestNT07PositiveControl(t *testing.T) {
	h := compose(seamGuard(t, flagOn, withUser, allowWith(authorization.ScopeOrganization)))
	for _, op := range guardedOps(t) {
		o := serve(h, op.url)
		if o.reached == nil && o.status < 500 {
			t.Errorf("%s: ORGANIZATION decision did not reach the data path (got %d %s %v)", op.id, o.status, o.outcome, o.reasons)
		}
		if o.status == http.StatusForbidden {
			t.Errorf("%s: supported ORGANIZATION read refused", op.id)
		}
	}
}

// TestNT07UnsupportedScopeRefused: an allowed decision with a scope the
// handler does not implement (unknown, NONE, empty) is refused with
// UNAUTHORIZED / ROW_SCOPE_DENIED before any data query, on every guarded GET
// operation, whether or not the caller's user id is proven.
func TestNT07UnsupportedScopeRefused(t *testing.T) {
	for _, sc := range []authorization.Scope{"OWN_TEST", "SELLER_OWN", "organization", authorization.ScopeNone, ""} {
		for _, subj := range []authorization.Subject{withUser, withoutUser} {
			h := compose(seamGuard(t, flagOn, subj, allowWith(sc)))
			for _, op := range guardedOps(t) {
				t.Run(fmt.Sprintf("%s/scope=%q/user=%t", op.id, sc, subj.UserID != ""), func(t *testing.T) {
					assertRefused(t, op, serve(h, op.url), reasoncodes.RowScopeDenied)
				})
			}
		}
	}
}

// TestNT07DeniedStaysDenied: a denied decision keeps its own reason.
func TestNT07DeniedStaysDenied(t *testing.T) {
	h := compose(seamGuard(t, flagOn, withUser, denyAll))
	for _, op := range guardedOps(t) {
		assertRefused(t, op, serve(h, op.url), reasoncodes.ActionDenied)
	}
}

// TestNT07Anonymous: the real evaluator with the anonymous subject denies
// every guarded GET operation before any query.
func TestNT07Anonymous(t *testing.T) {
	g, err := authorization.NewGuard(flagOn, authorization.Anonymous{})
	if err != nil {
		t.Fatal(err)
	}
	h := compose(g)
	for _, op := range guardedOps(t) {
		assertRefused(t, op, serve(h, op.url), reasoncodes.ActionDenied)
	}
}

// TestNT07FeatureDisabled: the flag is checked first and still answers
// 200 BLOCKED / FEATURE_DISABLED with no rows, even for an unsupported
// scope (flag behaviour unchanged).
func TestNT07FeatureDisabled(t *testing.T) {
	h := compose(seamGuard(t, flagOff, withUser, allowWith("OWN_TEST")))
	for _, op := range guardedOps(t) {
		o := serve(h, op.url)
		if o.reached != nil {
			t.Errorf("%s: data path entered with the feature disabled", op.id)
			continue
		}
		if o.status != http.StatusOK || o.outcome != result.Blocked.String() || strings.Join(o.reasons, ",") != string(reasoncodes.FeatureDisabled) {
			t.Errorf("%s: got %d %s %v, want 200 BLOCKED [FEATURE_DISABLED]", op.id, o.status, o.outcome, o.reasons)
		}
		for _, k := range []string{"items", "data", "totals"} {
			if _, ok := o.body[k]; ok {
				t.Errorf("%s: disabled body carries %q", op.id, k)
			}
		}
	}
}

// TestNT07GuardError: a guard that cannot evaluate (flag-source failure,
// the only error path of Guard.Check) fails closed with a server error and
// no data query.
func TestNT07GuardError(t *testing.T) {
	h := compose(seamGuard(t, flagErr, withUser, allowWith(authorization.ScopeOrganization)))
	for _, op := range guardedOps(t) {
		o := serve(h, op.url)
		if o.reached != nil {
			t.Errorf("%s: data path entered after a guard error", op.id)
			continue
		}
		if o.status < 500 {
			t.Errorf("%s: guard error answered %d, want a 5xx", op.id, o.status)
		}
	}
}

// TestNT07MissingIdentity: the own / caller-filtered reads refuse an
// allowed ORGANIZATION decision without a proven user id, before any query
// (never broadened to "all"); this is the existing behaviour.
func TestNT07MissingIdentity(t *testing.T) {
	h := compose(seamGuard(t, flagOn, withoutUser, allowWith(authorization.ScopeOrganization)))
	n := 0
	own := ownRowFiltered(t)
	if len(own) == 0 {
		t.Fatal("registry declares no own read that refuses an unproven user; NT-07 missing-identity coverage would be vacuous")
	}
	for _, op := range guardedOps(t) {
		if !own[op.id] {
			continue
		}
		n++
		assertRefused(t, op, serve(h, op.url), reasoncodes.RowScopeDenied)
	}
	if n != len(own) {
		t.Fatalf("checked %d own-filtered operations, want %d", n, len(own))
	}
}

// TestNT07OwnReadSemantics: the commission own read is unchanged for every
// decision the evaluator can produce: the real evaluator with the synthetic
// DEV grant of commission.own.read allows it (ORGANIZATION scope) and the
// handler proceeds to the caller-filtered query; without the capability it
// is denied ACTION_DENIED; without a user id it is ROW_SCOPE_DENIED.
func TestNT07OwnReadSemantics(t *testing.T) {
	const url = "/api/v1/commission/me/earnings"
	grant := identityapi.Grant{UserID: "syn-nt07", LegacyValue: "synthetic", Status: identityapi.GrantActive,
		Basis: authorization.DevGrantBasis, Capabilities: []identityapi.Capability{identityapi.CapCommissionOwnRead}}
	own := authorization.Subject{PrincipalID: "syn-nt07", UserID: synUserID, Grants: []identityapi.Grant{grant}}
	d := authorization.Evaluate(authorization.Runtime, own, authorization.Request{Capability: identityapi.CapCommissionOwnRead})
	if !d.Allowed || d.Scope != authorization.ScopeOrganization {
		t.Fatalf("evaluator decision for commission.own.read = %+v; RM-E04B assumes allowed ORGANIZATION", d)
	}
	mk := func(s authorization.Subject) http.Handler {
		g, err := authorization.NewGuard(flagOn, subjectStub{s})
		if err != nil {
			t.Fatal(err)
		}
		return compose(g)
	}
	if o := serve(mk(own), url); o.reached == nil && o.status < 500 {
		t.Errorf("own read with a proven user did not reach its caller-filtered query: %d %v", o.status, o.reasons)
	}
	op := getOp{id: "ListMyCommissionEarnings", url: url}
	noUser := own
	noUser.UserID = ""
	assertRefused(t, op, serve(mk(noUser), url), reasoncodes.RowScopeDenied)
	noCap := authorization.Subject{PrincipalID: "syn-nt07", UserID: synUserID}
	assertRefused(t, op, serve(mk(noCap), url), reasoncodes.ActionDenied)
}
