package shadow_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	writebackhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	orghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

type actorKey struct{}

// testSubjects is a TEST-ONLY resolver: the synthetic actor header names a
// legacy user key; the subject carries its target user id and a synthetic
// direct capability grant chosen per test. It is never composed by crm-api.
type testSubjects struct {
	users map[string]string
	caps  map[string][]identityapi.Capability
}

func (s testSubjects) Subject(ctx context.Context) authorization.Subject {
	key, _ := ctx.Value(actorKey{}).(string)
	if key == "" {
		return authorization.Subject{}
	}
	sub := authorization.Subject{PrincipalID: "synthetic:" + key, UserID: s.users[key]}
	if caps := s.caps[key]; len(caps) > 0 {
		sub.Grants = []identityapi.Grant{{UserID: s.users[key], LegacyValue: "synthetic", Status: identityapi.GrantActive,
			Basis: authorization.DevGrantBasis, Capabilities: caps}}
	}
	return sub
}

const actorHeader = "X-Synthetic-Test-Actor"

func (e *writesEnv) setFlags(t *testing.T, kv map[string]bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-02T08:30:00Z")
	if err := e.Runner(t, owner.FeatureFlags).InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		for k, v := range kv {
			if err := featureflags.Set(ctx, tx, featureflags.Flag{Key: k, Enabled: v, UpdatedAt: at}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func (e *writesEnv) writesServer(t *testing.T, subs testSubjects) http.Handler {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	flagRead := func(ctx context.Context, key string) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, key)
			return err
		})
		return on, err
	}
	guard := func(key string) *authorization.Guard {
		g, err := authorization.NewGuard(func(ctx context.Context) (bool, error) { return flagRead(ctx, key) }, subs)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	coMode, _ := rollout.FromFlags("cases.nonfinancial", flagRead)
	hrMode, _ := rollout.FromFlags("organization.hr_request", flagRead)
	q, err := writeback.NewQuery(e.Runner(t, integration.Module))
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment = authstub.Development
	h, err := apiserver.NewHandlerWithSurfaces(cfg, e.clock, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{
		CaseWrite: caseshttp.NewWrite(guard("sales.read_shadow"), guard("cases.nonfinancial_write"), coMode, e.cases),
		HRRequest: orghttp.NewHRRequests(guard("organization.hr_request_write"), hrMode, e.hr),
		Writeback: writebackhttp.New(guard("integration.writeback_read"), q),
	})
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, r.Header.Get(actorHeader))))
	})
}

type wresp struct {
	code int
	body string
	json map[string]any
}

func call(t *testing.T, h http.Handler, method, target, actor, body string, hdr map[string]string) wresp {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rd)
	if actor != "" {
		req.Header.Set(actorHeader, actor)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := wresp{code: rec.Code, body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	return r
}

func cmdOf(r wresp) map[string]any {
	c, _ := r.json["command"].(map[string]any)
	return c
}

// HTTP proof of the Slice 11 command surface: flags, capability, row scope,
// field scope, If-Match / Idempotency-Key, read purity, blocked routes,
// diagnostics and rollback.
func TestWritesHTTP(t *testing.T) {
	e := setupWrites(t)
	all := []identityapi.Capability{identityapi.CapContactOutcomeRecord, identityapi.CapHRRequestCreate, identityapi.CapHRRequestReview,
		identityapi.CapHRRequestRead, identityapi.CapWritebackRead, identityapi.CapCaseRead}
	subs := testSubjects{users: e.users, caps: map[string][]identityapi.Capability{
		"1": all, "2": all, "3": all, "4": all, "6": all, "7": all,
		"9": {identityapi.CapCaseRead}, // no command capability
	}}
	h := e.writesServer(t, subs)
	c101 := e.caseIDs["101"]
	path := "/api/v1/cases/" + c101 + "/contact-outcomes"
	hdr := func(key, ver string) map[string]string {
		return map[string]string{"Idempotency-Key": key, "If-Match": ver}
	}

	// Everything OFF: reads BLOCKED, commands BLOCKED (legacy only).
	if r := call(t, h, "POST", path, "6", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000001", "1")); r.code != 200 ||
		r.json["outcome"] != "BLOCKED" || cmdOf(r)["rollout_mode"] != "TARGET_WRITE_DISABLED" {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.setFlags(t, map[string]bool{"sales.read_shadow": true, "cases.nonfinancial_write": true, "cases.nonfinancial_writeback": true,
		"organization.hr_request_write": true, "organization.hr_request_writeback": true, "integration.writeback_read": true})

	purity := []string{"cases.work_case", "cases.contact_outcome_event", "cases.command_record", "cases.outbox_event", "custody.assignment",
		"organization.hr_request", "organization.hr_request_event", "organization.outbox_event", "integration.writeback_command",
		"integration.writeback_attempt", "cases.idempotency_record", "organization.idempotency_record"}
	before := e.md5(t, purity...)
	get := call(t, h, "GET", path, "6", "", nil)
	data, _ := get.json["data"].(map[string]any)
	actions, _ := get.json["allowed_actions"].(map[string]any)
	rec, _ := actions["cases.contact_outcome.record"].(map[string]any)
	if get.code != 200 || data == nil || rec["allowed"] != true {
		t.Fatalf("holder read: %d %s", get.code, get.body)
	}
	if !strings.Contains(get.body, `"action":"return_case"`) || !strings.Contains(get.body, `"ADR-01"`) {
		t.Fatalf("blocked actions are listed with their decisions: %s", get.body)
	}
	if r := call(t, h, "GET", path, "4", "", nil); !strings.Contains(r.body, `"allowed":false,"reason_code":"ROW_SCOPE_DENIED"`) {
		t.Fatalf("supervisor (higher rank) gets no record action: %s", r.body)
	}
	version := data["case_version"].(float64)
	ver := func() string {
		return strings.TrimSuffix(strings.TrimSuffix(strings.Split(strings.Split(call(t, h, "GET", path, "6", "", nil).body, `"case_version":`)[1], ",")[0], "}"), " ")
	}
	if e.md5(t, purity...) != before {
		t.Fatal("a GET wrote something")
	}

	// Contract checks.
	for _, c := range []struct {
		body string
		h    map[string]string
		code int
	}{
		{`{"outcome":"NO_ANSWER"}`, map[string]string{"If-Match": "1"}, 400},                  // no Idempotency-Key
		{`{"outcome":"NO_ANSWER"}`, hdr("http-key-000000002", "W/\"1\""), 400},                // not a version
		{`{"outcome":"NO_ANSWER","holder_user_id":"x"}`, hdr("http-key-000000003", "1"), 400}, // field outside scope
		{`{"outcome":"SOLD"}`, hdr("http-key-000000004", "1"), 400},                           // unknown value
		{`{"outcome":"NOT_PURCHASED"}`, hdr("http-key-000000005", ver()), 400},                // reason required
		{`[{"outcome":"NO_ANSWER"}]`, hdr("http-key-000000006", "1"), 400},                    // not an object (no bulk)
	} {
		if r := call(t, h, "POST", path, "6", c.body, c.h); r.code != c.code {
			t.Fatalf("%s %v: %d %s", c.body, c.h, r.code, r.body)
		}
	}
	// Value-level blocks name their decisions.
	if r := call(t, h, "POST", path, "6", `{"outcome":"DUPLICATE"}`, hdr("http-key-000000007", ver())); r.json["outcome"] != "BLOCKED" ||
		!strings.Contains(r.body, `"D-29"`) {
		t.Fatalf("duplicate: %s", r.body)
	}
	if r := call(t, h, "POST", path, "6", `{"outcome":"PRE_INVOICE"}`, hdr("http-key-000000008", ver())); !strings.Contains(r.body, `"ADR-01"`) {
		t.Fatalf("pre-invoice: %s", r.body)
	}
	// Authority: no capability → 403; higher rank with the capability but no row scope → 403.
	if r := call(t, h, "POST", path, "9", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000009", ver())); r.code != 403 {
		t.Fatalf("no capability: %d", r.code)
	}
	if r := call(t, h, "POST", path, "4", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000010", ver())); r.code != 403 || !strings.Contains(r.body, "ROW_SCOPE_DENIED") {
		t.Fatalf("supervisor: %d %s", r.code, r.body)
	}
	if r := call(t, h, "POST", path, "", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000011", ver())); r.code != 403 {
		t.Fatalf("anonymous: %d", r.code)
	}
	// Stale version → 409.
	if r := call(t, h, "POST", path, "6", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000012", "999")); r.code != 409 || !strings.Contains(r.body, "STALE_STATE") {
		t.Fatalf("stale: %d %s", r.code, r.body)
	}
	// Apply, replay, conflicting reuse.
	ok := call(t, h, "POST", path, "6", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000013", ver()))
	if ok.code != 200 || ok.json["outcome"] != "SUCCESS" || cmdOf(ok)["applied"] != true || cmdOf(ok)["writeback_state"] != "LEGACY_WRITEBACK_PENDING" {
		t.Fatalf("apply: %d %s", ok.code, ok.body)
	}
	if r := call(t, h, "POST", path, "6", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000013", "1")); r.code != 200 || cmdOf(r)["replayed"] != true {
		t.Fatalf("same key, same request: replay: %d %s", r.code, r.body)
	}
	if r := call(t, h, "POST", path, "6", `{"outcome":"NO_ANSWER"}`, hdr("http-key-000000013", "7")); r.code != 409 {
		t.Fatalf("same key, different If-Match (another request): %d %s", r.code, r.body)
	}
	if v := ver(); v == "" || v == "0" || strings.Contains(v, "\"") || float64(len(v)) < 1 || version < 1 {
		t.Fatal(v)
	}

	// Blocked / guessed routes never answer 2xx: no invoice issue, return,
	// release, transfer, allocation, ready, HR apply or generic transition.
	for _, p := range []string{"/api/v1/cases/" + c101 + "/return", "/api/v1/cases/" + c101 + "/ready", "/api/v1/cases/" + c101 + "/claims",
		"/api/v1/custody/transfers", "/api/v1/custody/allocations", "/api/v1/custody/returns", "/api/v1/invoices", "/api/v1/invoices/" + c101 + "/issue",
		"/api/v1/cases/" + c101 + "/transitions", "/api/v1/hr/requests/" + c101 + "/apply", "/api/v1/cases/bulk", "/api/v1/writeback/commands/" + c101 + "/retry"} {
		for _, m := range []string{"POST", "PUT", "PATCH"} {
			if r := call(t, h, m, p, "6", `{}`, hdr("http-key-guess-0001", "1")); r.code >= 200 && r.code < 300 {
				t.Fatalf("%s %s answered %d", m, p, r.code)
			}
		}
	}

	// HR over HTTP: create (senior), list, detail, decisions, final apply blocked.
	cr := call(t, h, "POST", "/api/v1/hr/requests", "4", `{"request_type":"terminate_seller","subject_user_id":"`+e.users["7"]+`","reason":"synthetic"}`,
		map[string]string{"Idempotency-Key": "http-key-hr-000001"})
	reqID, _ := cmdOf(cr)["request_id"].(string)
	if cr.code != 200 || reqID == "" || cmdOf(cr)["request_status"] != "pending_review" {
		t.Fatalf("hr create: %d %s", cr.code, cr.body)
	}
	if r := call(t, h, "GET", "/api/v1/hr/requests", "3", "", nil); !strings.Contains(r.body, reqID) || !strings.Contains(r.body, `"approve_step":{"allowed":true}`) {
		t.Fatalf("reviewer sees an actionable request: %s", r.body)
	}
	if r := call(t, h, "GET", "/api/v1/hr/requests/"+reqID, "9", "", nil); r.code != 403 {
		t.Fatalf("no read capability: %d", r.code)
	}
	if r := call(t, h, "GET", "/api/v1/hr/requests/"+reqID, "6", "", nil); r.code != 404 {
		t.Fatalf("not a party to the request: %d", r.code)
	}
	dec := call(t, h, "POST", "/api/v1/hr/requests/"+reqID+"/decisions", "3", `{"decision":"approve"}`,
		map[string]string{"Idempotency-Key": "http-key-hr-000002", "If-Match": "1"})
	if dec.code != 200 || cmdOf(dec)["request_status"] != "pending_review" {
		t.Fatalf("step: %d %s", dec.code, dec.body)
	}
	fin := call(t, h, "POST", "/api/v1/hr/requests/"+reqID+"/decisions", "2", `{"decision":"approve"}`,
		map[string]string{"Idempotency-Key": "http-key-hr-000003", "If-Match": "2"})
	if cmdOf(fin)["request_status"] != "pending_hr" {
		t.Fatalf("to HR: %s", fin.body)
	}
	if r := call(t, h, "POST", "/api/v1/hr/requests/"+reqID+"/decisions", "1", `{"decision":"approve"}`,
		map[string]string{"Idempotency-Key": "http-key-hr-000004", "If-Match": "3"}); r.json["outcome"] != "BLOCKED" || !strings.Contains(r.body, "OPD-05") {
		t.Fatalf("final apply: %s", r.body)
	}
	if r := call(t, h, "GET", "/api/v1/hr/requests/"+reqID, "2", "", nil); !strings.Contains(r.body, `"apply_final"`) ||
		!strings.Contains(r.body, `"status":"pending_hr"`) {
		t.Fatalf("detail: %s", r.body)
	}

	// Write-back diagnostics: deliver, then read the states (no payloads).
	e.drain(t)
	e.step(t)
	e.step(t)
	sum := call(t, h, "GET", "/api/v1/writeback/summary", "1", "", nil)
	if sum.code != 200 || !strings.Contains(sum.body, `"key":"APPLIED"`) {
		t.Fatalf("summary: %s", sum.body)
	}
	if strings.Contains(sum.body, "synthetic") || strings.Contains(call(t, h, "GET", "/api/v1/writeback/commands", "1", "", nil).body, `"reason"`) {
		t.Fatal("diagnostics never carry payloads")
	}
	cid := cmdOf(ok)["command_id"].(string)
	detail := call(t, h, "GET", "/api/v1/writeback/commands/"+cid, "1", "", nil)
	if !strings.Contains(detail.body, `"service_actor":"`+writeback.ServiceActor+`"`) || !strings.Contains(detail.body, `"parity":"MATCH"`) {
		t.Fatalf("command detail: %s", detail.body)
	}
	if r := call(t, h, "GET", "/api/v1/writeback/commands?state=FIXED", "1", "", nil); r.code != 400 {
		t.Fatalf("unknown state filter: %d", r.code)
	}

	// Rollback: write flag OFF → no new target command; history stays readable.
	e.setFlags(t, map[string]bool{"cases.nonfinancial_write": false})
	if r := call(t, h, "POST", path, "6", `{"outcome":"CALLBACK","callback_at":"2026-11-05T10:00:00Z"}`, hdr("http-key-000000020", ver())); r.json["outcome"] != "BLOCKED" {
		t.Fatalf("rollback: %s", r.body)
	}
	if r := call(t, h, "GET", path, "6", "", nil); !strings.Contains(r.body, `"history":[{`) || !strings.Contains(r.body, `"rollout_mode":"TARGET_WRITE_DISABLED"`) {
		t.Fatalf("history after rollback: %s", r.body)
	}
	if n := e.count(t, `SELECT count(*) FROM cases.contact_outcome_event`); n != 1 {
		t.Fatalf("history intact, nothing new: %d", n)
	}
}
