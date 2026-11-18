package shadow_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
)

// rawResp keeps the full observable response: status, every header and the
// exact body bytes.
type rawResp struct {
	code int
	hdr  http.Header
	body string
	json map[string]any
}

func rawCall(t *testing.T, h http.Handler, path, actor, correlation string) rawResp {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if actor != "" {
		req.Header.Set(actorHeader, actor)
	}
	if correlation != "" {
		req.Header.Set("X-Correlation-ID", correlation)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := rawResp{code: rec.Code, hdr: rec.Header().Clone(), body: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &r.json)
	return r
}

func headerNames(h http.Header) []string {
	out := make([]string, 0, len(h))
	for k := range h {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// NT-03 for the payment submission context (RM-E04B / A3, F-09): an invoice
// the caller does not provably own (payment submission predicate: seller AND
// issuer = the proven user; NULL never owns) answers exactly like a missing
// invoice. The owner's response, including BLOCKED and UNKNOWN business
// outcomes, is unchanged. 403 (no capability) and BLOCKED / FEATURE_DISABLED
// (flag off) are unchanged. Synthetic data, disposable PostgreSQL, reads only.
func TestPaymentSubmissionNonEnumeration(t *testing.T) {
	e := setupPayments(t)
	ctx := context.Background()
	payCaps := []identityapi.Capability{identityapi.CapPaymentUploadStage, identityapi.CapPaymentBatchSubmit, identityapi.CapPaymentSubmissionRead,
		identityapi.CapPaymentEvidenceRead}
	subs := testSubjects{users: e.users, caps: map[string][]identityapi.Capability{
		"6": payCaps, "7": payCaps, "2": payCaps, "4": payCaps,
		"ghost": payCaps, // capability grant but no resolved user id (T9)
	}}
	h := e.server(t, subs)
	path := func(inv string) string { return "/api/v1/payments/invoices/" + inv + "/submission" }
	const random = "5b0c6f0e-9d1a-4c37-8a3e-2f6d7e8a9b10" // T3: no such invoice

	// T11 — flag off: BLOCKED / FEATURE_DISABLED (unchanged), owner or not.
	for _, who := range []string{"6", "7"} {
		r := rawCall(t, h, path(e.invoices["3001"]), who, "")
		if r.code != 200 || r.json["outcome"] != "BLOCKED" || !strings.Contains(r.body, "FEATURE_DISABLED") || r.json["data"] != nil {
			t.Fatalf("T11 flag off (%s): %d %s", who, r.code, r.body)
		}
	}
	e.setFlags(t, map[string]bool{"payments.manual_write": true, "payments.manual_writeback": true})

	// T10 — no capability: 403 UNAUTHORIZED (unchanged), before any lookup.
	for _, inv := range []string{e.invoices["3001"], random} {
		if r := rawCall(t, h, path(inv), "9", ""); r.code != 403 || r.json["outcome"] != "UNAUTHORIZED" {
			t.Fatalf("T10 no capability: %d %s", r.code, r.body)
		}
	}

	// T1 — the owner (seller 6 = issuer 6): full context, unchanged shape.
	own := rawCall(t, h, path(e.invoices["3001"]), "6", "")
	od := own.json["data"].(map[string]any)
	allowed := own.json["allowed_actions"].(map[string]any)["payments.batch.submit"].(map[string]any)
	if own.code != 200 || own.json["outcome"] != "SUCCESS" || od["invoice_id"] != e.invoices["3001"] || od["eligible"] != true ||
		od["evaluation_status"] != "SUCCESS" || od["stage_no"] != float64(1) || od["unit"] != "IRT" || od["due_amount"] != float64(1000) ||
		od["remaining_amount"] != float64(1000) || od["pending_or_accepted_amount"] != float64(0) || od["available_amount"] != float64(1000) ||
		od["collection_version"] != float64(0) || od["rollout_mode"] != "CONTROLLED_WRITEBACK" || allowed["allowed"] != true ||
		!strings.Contains(own.body, `"issue_next_payment_stage"`) || od["attempts"] == nil {
		t.Fatalf("T1 owner: %d %s", own.code, own.body)
	}
	// T12 — owner of a paid invoice: business BLOCKED / ACTION_DENIED, 200.
	paid := rawCall(t, h, path(e.invoices["3003"]), "6", "")
	if pd, _ := paid.json["data"].(map[string]any); paid.code != 200 || pd["evaluation_status"] != "BLOCKED" || pd["eligible"] != false ||
		!strings.Contains(paid.body, `"reasons":["ACTION_DENIED"]`) {
		t.Fatalf("T12 owner blocked: %d %s", paid.code, paid.body)
	}
	// T13 — owner, unmapped (UNKNOWN) stage state: BLOCKED / DEPENDENCY_UNKNOWN,
	// amounts ABSENT (never zero), not 404.
	unk := rawCall(t, h, path(e.invoices["3006"]), "6", "")
	ud, _ := unk.json["data"].(map[string]any)
	if unk.code != 200 || ud["evaluation_status"] != "BLOCKED" || !strings.Contains(unk.body, `"DEPENDENCY_UNKNOWN"`) {
		t.Fatalf("T13 owner unknown: %d %s", unk.code, unk.body)
	}
	for _, k := range []string{"due_amount", "remaining_amount", "pending_or_accepted_amount", "available_amount", "stage_no"} {
		if v, present := ud[k]; present && v != nil {
			t.Fatalf("T13 %s must stay absent for an UNKNOWN stage, got %v", k, v)
		}
	}

	// Synthetic ownership variants (disposable DB only): NULL seller, NULL
	// issuer, both NULL. 3004 is seller 6 / issuer 4 in the fixture.
	for inv, sql := range map[string]string{
		"3007": `UPDATE billing.invoice SET seller_user_id = NULL WHERE invoice_id = $1`,
		"3008": `UPDATE billing.invoice SET issued_by_user_id = NULL WHERE invoice_id = $1`,
		"3005": `UPDATE billing.invoice SET seller_user_id = NULL, issued_by_user_id = NULL WHERE invoice_id = $1`,
	} {
		if _, err := e.admin.Exec(ctx, sql, e.invoices[inv]); err != nil {
			t.Fatalf("fixture %s: %v", inv, err)
		}
	}
	before := e.financial(t)
	paymentsBefore := e.md5(t, "payments.command_record", "payments.collection_account", "payments.submitted_attempt_event", "payments.receipt_state_history")

	// T3 — the reference: a random UUID that names no invoice.
	const corr = "a3-nonenum-corr-0001"
	missing := rawCall(t, h, path(random), "6", corr)
	if missing.code != 404 {
		t.Fatalf("T3 missing: %d %s", missing.code, missing.body)
	}
	missingFresh := rawCall(t, h, path(random), "6", "")

	cases := []struct {
		name, inv, who string
	}{
		{"T2 another seller's invoice (7/7) for seller 6", "3002", "6"},
		{"T2 seller 7 on seller 6's invoice", "3001", "7"},
		{"T2 manager above the seller", "3001", "2"},
		{"T2 supervisor of the seller", "3001", "4"},
		{"T4 seller matches, issuer differs (6/4) for 6", "3004", "6"},
		{"T5 issuer matches, seller differs (6/4) for 4", "3004", "4"},
		{"T6 NULL seller (issuer 6) for 6", "3007", "6"},
		{"T7 NULL issuer (seller 6) for 6", "3008", "6"},
		{"T8 both NULL for 6", "3005", "6"},
		{"T9 capability but no user id, owner's invoice", "3001", "ghost"},
		{"T9 capability but no user id, missing invoice", "", "ghost"},
	}
	leaks := []string{"ROW_SCOPE_DENIED", "DEPENDENCY_UNKNOWN", "eligible", "evaluation_status", "attempts", "amount", "stage", "collection_version",
		"allowed_actions", "blocked_by", "OPD-10", "receipt", `"data"`}
	for _, c := range cases {
		inv := random
		if c.inv != "" {
			inv = e.invoices[c.inv]
		}
		r := rawCall(t, h, path(inv), c.who, corr)
		// Byte-identical to the missing invoice with the same correlation id:
		// status, every header (Content-Type, Cache-Control, Vary, security
		// headers, X-Correlation-ID) and the body.
		if r.code != 404 || r.body != missing.body || !reflect.DeepEqual(r.hdr, missing.hdr) {
			t.Fatalf("%s: not identical to missing\n got  %d %v %s\n want %d %v %s", c.name, r.code, r.hdr, r.body, missing.code, missing.hdr, missing.body)
		}
		// T14 — nothing invoice-specific: no id echo, no amounts, no
		// ownership or eligibility facts, no user ids.
		for _, s := range append(leaks, inv, e.users["6"], e.users["4"], e.users["7"], "1000") {
			if s != "" && strings.Contains(r.body, s) {
				t.Fatalf("%s: response leaks %q: %s", c.name, s, r.body)
			}
		}
		// Without an inbound correlation id: identical except the generated
		// per-request correlation id (body field and header).
		f := rawCall(t, h, path(inv), c.who, "")
		if f.code != 404 || !reflect.DeepEqual(headerNames(f.hdr), headerNames(missingFresh.hdr)) {
			t.Fatalf("%s fresh: %d %v", c.name, f.code, f.hdr)
		}
		got, want := map[string]any{}, map[string]any{}
		for k, v := range f.json {
			got[k] = v
		}
		for k, v := range missingFresh.json {
			want[k] = v
		}
		delete(got, "correlation_id")
		delete(want, "correlation_id")
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s fresh body: %v != %v", c.name, got, want)
		}
		for _, k := range headerNames(f.hdr) {
			if k != "X-Correlation-Id" && f.hdr.Get(k) != missingFresh.hdr.Get(k) {
				t.Fatalf("%s fresh header %s: %q != %q", c.name, k, f.hdr.Get(k), missingFresh.hdr.Get(k))
			}
		}
	}
	// The 404 is the generic problem: application/problem+json, no-store.
	if ct := missing.hdr.Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") ||
		!strings.Contains(missing.hdr.Get("Cache-Control"), "no-store") || missing.json["reason_code"] != "CONTRACT_INVALID" ||
		missing.json["correlation_id"] != corr {
		t.Fatalf("404 problem shape: %v %s", missing.hdr, missing.body)
	}

	// Reads only: no target financial row and no payments command row moved.
	if e.financial(t) != before || e.md5(t, "payments.command_record", "payments.collection_account", "payments.submitted_attempt_event",
		"payments.receipt_state_history") != paymentsBefore {
		t.Fatal("the submission context read mutated state")
	}
	// The owner still reads the same context after all the probes.
	if again := rawCall(t, h, path(e.invoices["3001"]), "6", ""); again.code != 200 || again.json["data"].(map[string]any)["available_amount"] != float64(1000) {
		t.Fatalf("owner after probes: %d %s", again.code, again.body)
	}
}
