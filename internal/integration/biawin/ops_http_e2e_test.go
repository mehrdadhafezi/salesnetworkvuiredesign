package biawin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	biawinapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	biawinhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/http"
	fulfillmenthttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	provideropshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// opsSubjects maps the two synthetic dev principals to proven target user
// ids (the dev stub alone never proves a user id, so it always fails
// closed on row-scoped work).
type opsSubjects struct{ users map[string]string }

func (s opsSubjects) Subject(ctx context.Context) authorization.Subject {
	p, ok := authstub.FromContext(ctx)
	if !ok {
		return authorization.Subject{}
	}
	caps := []identityapi.Capability{identityapi.CapWorkRead, identityapi.CapWorkAssign, identityapi.CapWorkExecute, identityapi.CapActivationRead,
		identityapi.CapProviderOperationRead, identityapi.CapProviderOperationRecon}
	return authorization.Subject{PrincipalID: p.ID, UserID: s.users[p.ID], Grants: []identityapi.Grant{{UserID: p.ID, LegacyValue: "synthetic",
		Status: identityapi.GrantActive, Basis: authorization.DevGrantBasis, Capabilities: caps}}}
}

type opsFlags struct{ read, write, writeback, reconcile atomic.Bool }

func (e *env) opsServer(t *testing.T, f *opsFlags, subs authorization.SubjectResolver) http.Handler {
	t.Helper()
	guard := func(b *atomic.Bool) *authorization.Guard {
		g, err := authorization.NewGuard(func(context.Context) (bool, error) { return b.Load(), nil }, subs)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	mode, err := rollout.FromFlags("fulfillment.work", func(_ context.Context, key string) (bool, error) {
		switch key {
		case rollout.WriteFlag("fulfillment.work"):
			return f.write.Load(), nil
		case rollout.WritebackFlag("fulfillment.work"):
			return f.writeback.Load(), nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	read := guard(&f.read)
	surface := struct {
		*fulfillmenthttp.WorkHandlers
		*biawinhttp.ActivationHandlers
		*provideropshttp.Handlers
	}{fulfillmenthttp.NewWork(read, guard(&f.write), mode, e.work, e.cat), biawinhttp.NewActivation(read, e.act),
		provideropshttp.New(read, guard(&f.reconcile), e.ops)}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	h, err := apiserver.NewHandlerWithSurfaces(cfg, e.clock, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{Operations: surface})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

type opsResp struct {
	code int
	body string
	json map[string]any
}

func opsDo(t *testing.T, h http.Handler, method, target, user string, body any, headers map[string]string) opsResp {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, target, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set(authstub.Header, user)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := opsResp{code: rec.Code, body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	return r
}

func opsItems(r opsResp) []any { x, _ := r.json["items"].([]any); return x }

// 24 / 25 / 26 / 27 / 33: flags default OFF; capabilities alone never
// widen row scope; GETs never create or complete work; commands go
// through the same guarded domain; no activation or charge route exists.
func TestOperationsHTTP(t *testing.T) {
	e := setup(t)
	sup, ex, other := uid(t), uid(t), uid(t)
	keys := e.seed(t, "prod-http", 1, map[string]string{"prod-http.svc-01": sup}, map[string]string{"prod-http.svc-01": ex}, nil)
	e.seed(t, "prod-hidden", 1, map[string]string{"prod-hidden.svc-01": other}, nil, nil)
	customer := uid(t)
	if o := e.activate(t, paid(t, "prod-http", biawinapi.RouteAdditionalServices, "RESOLVED", &customer)); o.Decision != biawinapi.DecisionShadowActivated {
		t.Fatalf("activation: %+v", o)
	}
	hidden := uid(t)
	e.activate(t, paid(t, "prod-hidden", biawinapi.RouteAdditionalServices, "RESOLVED", &hidden))
	e.relay(t)
	var caseID string
	if err := e.admin(t).QueryRow(context.Background(), `SELECT case_id::text FROM fulfillment.work_case WHERE service_key = $1`, keys[0]).Scan(&caseID); err != nil {
		t.Fatal(err)
	}
	var hiddenCase string
	if err := e.admin(t).QueryRow(context.Background(), `SELECT case_id::text FROM fulfillment.work_case WHERE service_key = 'prod-hidden.svc-01'`).Scan(&hiddenCase); err != nil {
		t.Fatal(err)
	}
	f := &opsFlags{}
	h := e.opsServer(t, f, opsSubjects{users: map[string]string{"dev-user-001": sup, "dev-user-002": ex}})
	gets := []string{"/api/v1/fulfillment/work-cases", "/api/v1/fulfillment/work-cases/" + caseID, "/api/v1/biawin/activation-decisions",
		"/api/v1/integration/provider-operations"}
	fingerprint := func() int {
		return e.count(t, `SELECT (SELECT count(*) FROM fulfillment.work_case) + (SELECT count(*) FROM fulfillment.work_event) +
			(SELECT count(*) FROM fulfillment.work_command_record) + (SELECT count(*) FROM biawin.activation) + (SELECT count(*) FROM integration.provider_operation) +
			(SELECT coalesce(sum(version), 0) FROM fulfillment.work_case)`)
	}
	before := fingerprint()

	// Flags OFF (default): every read and command is BLOCKED, no data.
	for _, p := range gets {
		if r := opsDo(t, h, "GET", p, "dev-user-001", nil, nil); r.code != 200 || r.json["outcome"] != "BLOCKED" || r.json["items"] != nil || r.json["case"] != nil {
			t.Fatalf("%s with flags off: %d %s", p, r.code, r.body)
		}
	}
	assign := map[string]any{"assignee_user_id": ex}
	hdr := func(k string, v string) map[string]string {
		return map[string]string{"Idempotency-Key": k, "If-Match": v}
	}
	if r := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-001", assign, hdr("http-key-0001", `"1"`)); r.json["outcome"] != "BLOCKED" {
		t.Fatalf("assignment with flags off: %d %s", r.code, r.body)
	}

	// Read on: the supervisor sees only cases of services in scope.
	f.read.Store(true)
	r := opsDo(t, h, "GET", "/api/v1/fulfillment/work-cases", "dev-user-001", nil, nil)
	if r.code != 200 || r.json["outcome"] != "SUCCESS" || len(opsItems(r)) != 1 || !strings.Contains(r.body, caseID) || strings.Contains(r.body, hiddenCase) {
		t.Fatalf("scoped queue: %d %s", r.code, r.body)
	}
	if r := opsDo(t, h, "GET", "/api/v1/fulfillment/work-cases/"+hiddenCase, "dev-user-001", nil, nil); r.code != 404 {
		t.Fatalf("out-of-scope detail must be 404: %d %s", r.code, r.body)
	}
	// A capability without a proven user id reads nothing (fails closed).
	hNoUser := e.opsServer(t, f, opsSubjects{users: map[string]string{}})
	if r := opsDo(t, hNoUser, "GET", "/api/v1/fulfillment/work-cases", "dev-user-001", nil, nil); r.code != 403 {
		t.Fatalf("unproven user id must be denied: %d %s", r.code, r.body)
	}
	// Detail lists the decision-blocked commands, never as available.
	d := opsDo(t, h, "GET", "/api/v1/fulfillment/work-cases/"+caseID, "dev-user-001", nil, nil)
	if d.code != 200 || !strings.Contains(d.body, "D-38") || !strings.Contains(d.body, "D-35") || !strings.Contains(d.body, "OPD-04") {
		t.Fatalf("detail: %d %s", d.code, d.body)
	}
	// Provider operations: connector always NOT_CONFIGURED.
	if r := opsDo(t, h, "GET", "/api/v1/integration/provider-operations", "dev-user-001", nil, nil); r.json["connector"] != "NOT_CONFIGURED" {
		t.Fatalf("connector: %s", r.body)
	}
	if r := opsDo(t, h, "GET", "/api/v1/biawin/activation-decisions", "dev-user-001", nil, nil); len(opsItems(r)) != 2 || strings.Contains(r.body, "PRODUCTION") {
		t.Fatalf("activation decisions: %s", r.body)
	}
	if after := fingerprint(); after != before {
		t.Fatalf("GETs wrote: %d -> %d", before, after)
	}

	// Write on: commands run the same guarded domain.
	f.write.Store(true)
	if r := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-002", assign, hdr("http-key-0002", `"1"`)); r.code != 403 {
		t.Fatalf("executor cannot assign: %d %s", r.code, r.body)
	}
	if r := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-001", assign, nil); r.code != 400 {
		t.Fatalf("missing headers: %d %s", r.code, r.body)
	}
	// SHADOW_COMPARE (write flag only): evaluated and recorded, never applied.
	sh := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-001", assign, hdr("http-key-0011", `"1"`))
	if cmd, _ := sh.json["command"].(map[string]any); sh.code != 200 || cmd["rollout_mode"] != "SHADOW_COMPARE" || cmd["applied"] != false ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1::uuid`, caseID) != 0 {
		t.Fatalf("shadow assign: %d %s", sh.code, sh.body)
	}
	f.writeback.Store(true)
	a := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-001", assign, hdr("http-key-0003", `"1"`))
	if cmd, _ := a.json["command"].(map[string]any); a.code != 200 || a.json["outcome"] != "SUCCESS" || cmd["applied"] != true {
		t.Fatalf("assign: %d %s", a.code, a.body)
	}
	if again := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-001", assign, hdr("http-key-0003", `"1"`)); again.json["outcome"] != "SUCCESS" ||
		e.count(t, `SELECT count(*) FROM fulfillment.work_assignment WHERE case_id = $1::uuid`, caseID) != 1 {
		t.Fatalf("replay: %s", again.body)
	}
	if stale := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/assignment", "dev-user-001", assign, hdr("http-key-0004", `"1"`)); stale.code != 409 {
		t.Fatalf("stale version: %d %s", stale.code, stale.body)
	}
	// Completion without the required evidence stays BLOCKED.
	if c := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/outcome", "dev-user-002", map[string]any{"outcome": "EXECUTED"},
		hdr("http-key-0005", `"2"`)); c.json["outcome"] != "BLOCKED" || !strings.Contains(c.body, "REQUIRED_EVIDENCE_MISSING") {
		t.Fatalf("complete without evidence: %d %s", c.code, c.body)
	}
	if ev := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/evidence-records", "dev-user-002",
		map[string]any{"evidence_type": "service-form", "label": "form", "content_sha256": shaOf("form")}, hdr("http-key-0006", `"2"`)); ev.json["outcome"] != "SUCCESS" {
		t.Fatalf("evidence: %d %s", ev.code, ev.body)
	}
	if c := opsDo(t, h, "POST", "/api/v1/fulfillment/work-cases/"+caseID+"/outcome", "dev-user-002", map[string]any{"outcome": "EXECUTED"},
		hdr("http-key-0007", `"3"`)); c.json["outcome"] != "SUCCESS" {
		t.Fatalf("complete: %d %s", c.code, c.body)
	}
	// No activation, wallet, cancel or provider dispatch route exists.
	for _, p := range []string{"/api/v1/biawin/subscriptions/x/activate", "/api/v1/biawin/wallet/charge", "/api/v1/fulfillment/work-cases/" + caseID + "/cancel",
		"/api/v1/integration/provider-operations/x/dispatch"} {
		if r := opsDo(t, h, "POST", p, "dev-user-001", map[string]any{}, hdr("http-key-0009", `"1"`)); r.code != 404 && r.code != 405 {
			t.Fatalf("%s must not exist: %d", p, r.code)
		}
	}
	// Reconcile off: resolution BLOCKED.
	if r := opsDo(t, h, "POST", "/api/v1/integration/provider-operations/"+uid(t)+"/resolution", "dev-user-001",
		map[string]any{"present": true, "evidence_note": "provider console"}, hdr("http-key-0010", `"1"`)); r.json["outcome"] != "BLOCKED" {
		t.Fatalf("resolution with the flag off: %d %s", r.code, r.body)
	}
}
