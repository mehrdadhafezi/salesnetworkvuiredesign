package shadow_test

// RM-P07 Finance operational workspace: the narrow read additions and the
// authorization boundaries the workspace composes.
//
//   - GET /api/v1/finance/review/attempts?evidence_state= keeps only target
//     attempts whose stored evidence_state equals it (conjunctive with
//     invoice_id, keyset paged, finance.review.read only).
//   - GET /api/v1/payments/attempts?has_state_regression= keeps legacy
//     attempts with (true) or without (false) an append-only state-history
//     row flagged regression (payments.attempt.read only).
//
// Disposable PostgreSQL, synthetic fixtures. Target attempts are created
// through the real Slice 12 submission command; one regression history row
// is a synthetic fixture inserted before the read-only fingerprint. Every
// expectation is the stored rows; every read is proven to write nothing.
// Review, invoice and payment-attempt reads are authorized independently:
// none implies another. No decision, posting, refund or retry route exists.

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	billinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/http"
	financehttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// p07Server composes the billing / payments read surface and the Finance
// review surface, each behind its own existing feature flag, with a
// TEST-ONLY synthetic subject resolver (actor header → direct capability
// grant). No grant, capability, role or flag default is added.
func (e *paymentsEnv) p07Server(t *testing.T, subs testSubjects) http.Handler {
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
	mode, _ := rollout.FromFlags("payments.manual", flagRead)
	cfg := apiserver.DefaultConfig()
	cfg.Environment = authstub.Development
	bg := guard(billingFlag)
	h, err := apiserver.NewHandlerWithSurfaces(cfg, e.clock, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{
		Billing:  billingSurface{billinghttp.New(bg, e.billingQ, e.paymentsQ), paymentshttp.New(bg, e.paymentsQ)},
		Payments: paymentsSurface{paymentshttp.NewWrite(guard("payments.manual_write"), mode, e.svc), financehttp.NewReview(guard("finance.review_read"), e.svc)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, r.Header.Get(actorHeader))))
	})
}

// p07Walk follows next_cursor for actor and returns every id of field in
// order, failing on a non-SUCCESS page or a duplicate across pages.
func p07Walk(t *testing.T, h http.Handler, actor, base, list, field string) ([]string, int) {
	t.Helper()
	var ids []string
	next, pages := base, 0
	for next != "" {
		if pages++; pages > 30 {
			t.Fatalf("%s: runaway pagination", base)
		}
		r := call(t, h, http.MethodGet, next, actor, "", nil)
		if r.code != 200 || r.json["outcome"] != "SUCCESS" {
			t.Fatalf("GET %s = %d %s", next, r.code, r.body)
		}
		rows, _ := r.json[list].([]any)
		for _, x := range rows {
			id := x.(map[string]any)[field].(string)
			if slices.Contains(ids, id) {
				t.Fatalf("%s: duplicate %s across pages", base, id)
			}
			ids = append(ids, id)
		}
		next = ""
		if pg, _ := r.json["page"].(map[string]any); pg != nil && pg["has_more"] == true {
			sep := "&"
			if !strings.Contains(base, "?") {
				sep = "?"
			}
			next = base + sep + "cursor=" + url.QueryEscape(pg["next_cursor"].(string))
		}
	}
	return ids, pages
}

func (e *paymentsEnv) p07IDs(t *testing.T, sql string, args ...any) []string {
	t.Helper()
	rows, err := e.admin.Query(context.Background(), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func p07Sorted(xs []string) []string {
	out := slices.Clone(xs)
	sort.Strings(out)
	return out
}

var p07Model = []string{"billing.invoice", "billing.invoice_state_evidence", "billing.amount_evidence", "billing.invoice_item", "billing.payment_stage",
	"payments.payment_attempt", "payments.gateway_attempt", "payments.attempt_state_history", "payments.receipt",
	"payments.submitted_attempt", "payments.payment_batch", "payments.receipt_evidence", "payments.receipt_state_history", "payments.upload_object",
	"payments.evidence_reconciliation", "finance.ledger_entry", "finance.ledger_line", "finance.shadow_run"}

func TestRMP07FinanceWorkspaceReads(t *testing.T) {
	ctx := context.Background()
	e := setupPayments(t)
	seller := e.actor("6")
	inv1, inv5 := e.invoices["3001"], e.invoices["3005"]
	if r := e.submit(t, batch("p07-ready-0001", inv1, 0, seller, item("a", 300, e.upload(t, "6", 71)), item("b", 200, e.upload(t, "6", 72)))); r.Status != result.Success {
		t.Fatalf("submit 3001: %+v", r)
	}
	if rep, err := e.svc.Finalise(ctx, 10); err != nil || rep.AttemptsReady < 1 {
		t.Fatalf("finalise: %+v %v", rep, err)
	}
	if r := e.submit(t, batch("p07-pending-0002", inv5, 0, seller, item("a", 100, e.upload(t, "6", 73)))); r.Status != result.Success {
		t.Fatalf("submit 3005: %+v", r)
	}
	// Synthetic fixture: one observed regression of the legacy attempt.
	legacyAttempt := e.q(t, `SELECT min(attempt_id::text) FROM payments.payment_attempt`)
	if _, err := e.admin.Exec(ctx, `INSERT INTO payments.attempt_state_history (history_id, attempt_id, old_state_raw, new_state_raw, regression, batch_id, recorded_at)
		VALUES ('70000000-0000-4000-8000-000000000001', $1, 'approved', 'pending', true, '70000000-0000-4000-8000-0000000000b1', '2026-11-03T08:00:00Z')`, legacyAttempt); err != nil {
		t.Fatal(err)
	}

	subs := testSubjects{users: e.users, caps: map[string][]identityapi.Capability{
		"1": {identityapi.CapFinanceReviewRead},                              // review only
		"2": {identityapi.CapInvoiceRead, identityapi.CapPaymentAttemptRead}, // billing / payments only
		"3": {identityapi.CapFinanceReviewRead, identityapi.CapInvoiceRead, identityapi.CapPaymentAttemptRead},
		"4": {identityapi.CapInvoiceRead},       // invoices without attempts
		"6": {identityapi.CapFinanceReviewRead}, // the submitter (maker)
	}}
	h := e.p07Server(t, subs)
	e.setFlags(t, map[string]bool{billingFlag: true, "finance.review_read": true})
	fp := e.md5(t, p07Model...)

	const queue = "/api/v1/finance/review/attempts"
	t.Run("evidence_state filter partitions the queue and pages", func(t *testing.T) {
		all, _ := p07Walk(t, h, "3", queue, "data", "attempt_id")
		if len(all) != 3 {
			t.Fatalf("queue = %d, want 3", len(all))
		}
		var union []string
		for _, st := range []string{"PENDING_FINALISATION", "READY", "NOT_RECOVERABLE"} {
			got, _ := p07Walk(t, h, "3", queue+"?evidence_state="+st, "data", "attempt_id")
			want := e.p07IDs(t, `SELECT attempt_id::text FROM payments.submitted_attempt WHERE state = 'PENDING_REVIEW' AND evidence_state = $1 ORDER BY attempt_id`, st)
			if !slices.Equal(got, want) {
				t.Fatalf("%s: got %v want %v", st, got, want)
			}
			union = append(union, got...)
		}
		if !slices.Equal(p07Sorted(union), p07Sorted(all)) {
			t.Fatalf("evidence states must partition the queue: %v vs %v", union, all)
		}
		ready := e.p07IDs(t, `SELECT attempt_id::text FROM payments.submitted_attempt WHERE evidence_state = 'READY' ORDER BY attempt_id`)
		pending := e.p07IDs(t, `SELECT attempt_id::text FROM payments.submitted_attempt WHERE evidence_state = 'PENDING_FINALISATION' ORDER BY attempt_id`)
		if len(ready) == 0 || len(pending) == 0 {
			t.Fatalf("fixture must hold both states: ready=%v pending=%v", ready, pending)
		}
		// Conjunctive with invoice_id; keyset pagination with limit=1.
		both, pages := p07Walk(t, h, "3", queue+"?limit=1&evidence_state=READY&invoice_id="+inv1, "data", "attempt_id")
		want := e.p07IDs(t, `SELECT attempt_id::text FROM payments.submitted_attempt WHERE evidence_state = 'READY' AND invoice_id = $1 ORDER BY attempt_id`, inv1)
		if !slices.Equal(both, want) || pages != max(1, len(want)) {
			t.Fatalf("READY ∧ 3001: %v in %d pages, want %v", both, pages, want)
		}
		if got, _ := p07Walk(t, h, "3", queue+"?evidence_state=READY&invoice_id="+inv5, "data", "attempt_id"); int64(len(got)) != e.count(t,
			`SELECT count(*) FROM payments.submitted_attempt WHERE evidence_state = 'READY' AND invoice_id = $1`, inv5) {
			t.Fatalf("READY ∧ 3005: %v", got)
		}
		// Malformed or non-contract values: 400, never data.
		for _, p := range []string{"evidence_state=ready", "evidence_state=RECEIVED", "evidence_state=PAID", "evidence_state=", "receipt_state=READY"} {
			if r := call(t, h, http.MethodGet, queue+"?"+p, "3", "", nil); r.code != 400 || r.json["data"] != nil {
				t.Fatalf("%s must be 400: %d %s", p, r.code, r.body)
			}
		}
	})

	t.Run("review, invoice and attempt reads are authorized independently", func(t *testing.T) {
		// Review only: queue yes; the invoice and the legacy attempts are denied.
		if ids, _ := p07Walk(t, h, "1", queue, "data", "attempt_id"); len(ids) != 3 {
			t.Fatalf("review-only queue: %v", ids)
		}
		for _, p := range []string{"/api/v1/invoices/" + inv1, "/api/v1/invoices", "/api/v1/payments/attempts", "/api/v1/invoices/" + inv1 + "/payments",
			"/api/v1/invoices/" + inv1 + "/receipts"} {
			if r := call(t, h, http.MethodGet, p, "1", "", nil); r.code != 403 || r.json["data"] != nil || r.json["items"] != nil {
				t.Fatalf("review-only %s must be 403: %d %s", p, r.code, r.body)
			}
		}
		// Billing / payments only: invoice and attempts yes; the queue and one review item are denied.
		if r := call(t, h, http.MethodGet, "/api/v1/invoices/"+inv1, "2", "", nil); r.code != 200 || r.json["outcome"] != "SUCCESS" {
			t.Fatalf("invoice read: %d %s", r.code, r.body)
		}
		if r := call(t, h, http.MethodGet, "/api/v1/payments/attempts", "2", "", nil); r.code != 200 || r.json["outcome"] != "SUCCESS" {
			t.Fatalf("attempts read: %d %s", r.code, r.body)
		}
		attempt := e.q(t, `SELECT min(attempt_id::text) FROM payments.submitted_attempt`)
		for _, p := range []string{queue, queue + "?invoice_id=" + inv1, queue + "?evidence_state=READY", queue + "/" + attempt} {
			if r := call(t, h, http.MethodGet, p, "2", "", nil); r.code != 403 || r.json["data"] != nil {
				t.Fatalf("invoice reader %s must be 403: %d %s", p, r.code, r.body)
			}
		}
		// Invoice read does not imply payment-attempt read.
		if r := call(t, h, http.MethodGet, "/api/v1/payments/attempts?has_state_regression=true", "4", "", nil); r.code != 403 || r.json["items"] != nil {
			t.Fatalf("invoice-only attempts must be 403: %d %s", r.code, r.body)
		}
		// No capability at all.
		for _, p := range []string{queue, "/api/v1/invoices", "/api/v1/payments/attempts"} {
			if r := call(t, h, http.MethodGet, p, "9", "", nil); r.code != 403 {
				t.Fatalf("no-capability %s: %d", p, r.code)
			}
		}
	})

	t.Run("has_state_regression filter matches the append-only history", func(t *testing.T) {
		const list = "/api/v1/payments/attempts"
		with, _ := p07Walk(t, h, "2", list+"?has_state_regression=true", "items", "attempt_id")
		want := e.p07IDs(t, `SELECT a.attempt_id::text FROM payments.payment_attempt a WHERE EXISTS (SELECT 1 FROM payments.attempt_state_history h
			WHERE h.attempt_id = a.attempt_id AND h.regression) ORDER BY a.attempt_id`)
		if len(want) == 0 || !slices.Equal(with, want) {
			t.Fatalf("regression: got %v want %v", with, want)
		}
		without, _ := p07Walk(t, h, "2", list+"?has_state_regression=false", "items", "attempt_id")
		all, _ := p07Walk(t, h, "2", list, "items", "attempt_id")
		if !slices.Equal(p07Sorted(append(slices.Clone(with), without...)), p07Sorted(all)) || len(slices.DeleteFunc(slices.Clone(without), func(id string) bool { return !slices.Contains(with, id) })) != 0 {
			t.Fatalf("true/false must partition the attempts: %v + %v vs %v", with, without, all)
		}
		// Conjunctive with an existing filter.
		if got, _ := p07Walk(t, h, "2", list+"?has_state_regression=true&invoice_link_state=UNKNOWN", "items", "attempt_id"); int64(len(got)) != e.count(t,
			`SELECT count(*) FROM payments.payment_attempt a WHERE a.invoice_link_state = 'UNKNOWN' AND EXISTS (SELECT 1 FROM payments.attempt_state_history h
			WHERE h.attempt_id = a.attempt_id AND h.regression)`) {
			t.Fatalf("regression ∧ UNKNOWN link: %v", got)
		}
		for _, p := range []string{"has_state_regression=yes", "has_state_regression=", "regression=true"} {
			if r := call(t, h, http.MethodGet, list+"?"+p, "2", "", nil); r.code != 400 || r.json["items"] != nil {
				t.Fatalf("%s must be 400: %d %s", p, r.code, r.body)
			}
		}
		// The regression row is evidence only: the attempt's state is unchanged.
		if e.q(t, `SELECT state_raw FROM payments.payment_attempt WHERE attempt_id = $1`, legacyAttempt) == "" {
			t.Fatal("legacy attempt lost its raw state")
		}
	})

	t.Run("maker / checker: a submitter is never an independent reviewer and no decision is allowed", func(t *testing.T) {
		for _, who := range []string{"6", "3"} {
			r := call(t, h, http.MethodGet, queue, who, "", nil)
			rows, _ := r.json["data"].([]any)
			if r.code != 200 || len(rows) != 3 {
				t.Fatalf("%s queue: %d %s", who, r.code, r.body)
			}
			for _, x := range rows {
				m := x.(map[string]any)
				if m["reviewer_is_submitter"] != (who == "6") {
					t.Fatalf("%s: reviewer_is_submitter = %v", who, m["reviewer_is_submitter"])
				}
				if m["state"] != "PENDING_REVIEW" || m["review_provenance"] != "NOT_REVIEWED" {
					t.Fatalf("a queued submission is not reviewed or received: %v", m)
				}
				for _, g := range m["gates"].([]any) {
					if g.(map[string]any)["outcome"] == "SUCCESS" {
						t.Fatalf("%s: a decision gate is open: %v", who, g)
					}
				}
			}
			d := call(t, h, http.MethodGet, queue+"/"+rows[0].(map[string]any)["attempt_id"].(string), who, "", nil)
			acts, _ := d.json["allowed_actions"].(map[string]any)
			if d.code != 200 || len(acts) == 0 {
				t.Fatalf("%s detail: %d %s", who, d.code, d.body)
			}
			for k, v := range acts {
				if v.(map[string]any)["allowed"] != false {
					t.Fatalf("%s: %s allowed", who, k)
				}
			}
		}
		// No decision, posting, settlement, refund, retry or reconciliation route exists.
		attempt := e.q(t, `SELECT min(attempt_id::text) FROM payments.submitted_attempt`)
		for _, p := range []string{queue + "/" + attempt + "/approve", queue + "/" + attempt + "/reject", queue + "/" + attempt, queue,
			"/api/v1/payments/attempts/" + legacyAttempt + "/retry", "/api/v1/payments/attempts/" + legacyAttempt + "/refund",
			"/api/v1/invoices/" + inv1 + "/settle", "/api/v1/finance/reconciliation"} {
			for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
				r := call(t, h, m, p, "3", `{}`, map[string]string{"Idempotency-Key": "p07-no-command-0001"})
				if r.code != 404 && r.code != 405 {
					t.Fatalf("%s %s = %d (no financial command may exist)", m, p, r.code)
				}
			}
		}
	})

	if e.md5(t, p07Model...) != fp {
		t.Fatal("a workspace read changed billing, payments or finance state")
	}
	for path, keys := range map[string][]string{queue: {"invoice_id", "evidence_state"}, "/api/v1/payments/attempts": {"has_state_regression"}} {
		for _, k := range keys {
			if !slices.Contains(apiserver.ContractQuery[path], k) {
				t.Fatalf("allowlist %s lacks %s", path, k)
			}
		}
	}
}
