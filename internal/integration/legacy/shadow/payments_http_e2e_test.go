package shadow_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	financehttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

type paymentsSurface struct {
	*paymentshttp.WriteHandlers
	*financehttp.ReviewHandlers
}

func (e *paymentsEnv) setFlags(t *testing.T, kv map[string]bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-03T08:30:00Z")
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

func (e *paymentsEnv) server(t *testing.T, subs testSubjects) http.Handler {
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
	h, err := apiserver.NewHandlerWithSurfaces(cfg, e.clock, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{
		Payments: paymentsSurface{paymentshttp.NewWrite(guard("payments.manual_write"), mode, e.svc), financehttp.NewReview(guard("finance.review_read"), e.svc)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, r.Header.Get(actorHeader))))
	})
}

func dataOf(r wresp) map[string]any {
	d, _ := r.json["data"].(map[string]any)
	return d
}

// The payments surface over HTTP: flags, capabilities, row scope,
// allowed_actions from the command evaluator, idempotency, 409 on stale
// versions, per-item outcomes, and the read-only Finance review queue with
// its D-43 and maker / checker gate. No decision route exists.
func TestPaymentsHTTP(t *testing.T) {
	e := setupPayments(t)
	payCaps := []identityapi.Capability{identityapi.CapPaymentUploadStage, identityapi.CapPaymentBatchSubmit, identityapi.CapPaymentSubmissionRead,
		identityapi.CapPaymentEvidenceRead, identityapi.CapFinanceReviewRead}
	subs := testSubjects{users: e.users, caps: map[string][]identityapi.Capability{
		"6": payCaps, "7": payCaps, "2": payCaps,
		"1": {identityapi.CapFinanceReviewRead},
	}}
	h := e.server(t, subs)
	inv := e.invoices["3001"]
	upload := func(actor string, body []byte) wresp {
		return call(t, h, http.MethodPost, "/api/v1/payments/uploads", actor, string(body), map[string]string{"Content-Type": "application/octet-stream"})
	}
	// Flags off: BLOCKED / FEATURE_DISABLED, nothing stored.
	if r := upload("6", receipt(61)); r.code != 200 || r.json["outcome"] != "BLOCKED" || e.count(t, `SELECT count(*) FROM payments.upload_object`) != 0 {
		t.Fatalf("flag off upload: %d %s", r.code, r.body)
	}
	// payments.manual_write alone is SHADOW_COMPARE: BLOCKED for payments.
	e.setFlags(t, map[string]bool{"payments.manual_write": true})
	if r := upload("6", receipt(61)); r.json["outcome"] != "BLOCKED" || !strings.Contains(r.body, "SHADOW_COMPARE_UNSUPPORTED") {
		t.Fatalf("shadow upload: %s", r.body)
	}
	e.setFlags(t, map[string]bool{"payments.manual_writeback": true, "finance.review_read": true})
	// No capability → 403; an HTML body is 400.
	if r := upload("9", receipt(61)); r.code != 403 {
		t.Fatalf("no capability: %d", r.code)
	}
	if r := upload("6", []byte("<html></html>")); r.code != 400 {
		t.Fatalf("html: %d %s", r.code, r.body)
	}
	up := upload("6", receipt(61))
	if up.code != 200 || up.json["outcome"] != "SUCCESS" {
		t.Fatalf("upload: %d %s", up.code, up.body)
	}
	ud := dataOf(up)
	uploadID, sum := ud["upload_id"].(string), ud["content_sha256"].(string)

	// Submission context: allowed_actions from the same evaluator.
	ctxView := call(t, h, http.MethodGet, "/api/v1/payments/invoices/"+inv+"/submission", "6", "", nil)
	allowed := ctxView.json["allowed_actions"].(map[string]any)["payments.batch.submit"].(map[string]any)
	if ctxView.code != 200 || allowed["allowed"] != true || dataOf(ctxView)["available_amount"] != float64(1000) ||
		dataOf(ctxView)["collection_version"] != float64(0) || !strings.Contains(ctxView.body, `"issue_next_payment_stage"`) {
		t.Fatalf("context: %s", ctxView.body)
	}
	for _, who := range []string{"7", "2"} { // another seller; the manager above
		v := call(t, h, http.MethodGet, "/api/v1/payments/invoices/"+inv+"/submission", who, "", nil)
		a := v.json["allowed_actions"].(map[string]any)["payments.batch.submit"].(map[string]any)
		if a["allowed"] != false || a["reason_code"] != "ROW_SCOPE_DENIED" || len(dataOf(v)["attempts"].([]any)) != 0 {
			t.Fatalf("context for %s: %s", who, v.body)
		}
	}
	body := fmt.Sprintf(`{"stage_no":1,"entry_mode":"RECEIPT","items":[{"item_key":"a","amount":400,"unit":"IRT","receipts":[{"upload_id":"%s","content_sha256":"%s"}]}]}`,
		uploadID, sum)
	hdr := func(key, version string) map[string]string {
		return map[string]string{"Idempotency-Key": key, "If-Match": version}
	}
	batchPath := "/api/v1/payments/invoices/" + inv + "/batches"
	// Another seller → 403 (UNAUTHORIZED); nothing committed.
	if r := call(t, h, http.MethodPost, batchPath, "7", body, hdr("http-key-000000001", "0")); r.code != 403 {
		t.Fatalf("other seller: %d %s", r.code, r.body)
	}
	// Stale version → 409 with per-item outcomes and the current version.
	if r := call(t, h, http.MethodPost, batchPath, "6", body, hdr("http-key-000000002", "3")); r.code != 409 ||
		dataOf(r)["current_version"] != float64(0) {
		t.Fatalf("stale: %d %s", r.code, r.body)
	}
	if r := call(t, h, http.MethodPost, batchPath, "6", body, hdr("http-key-000000003", "-1")); r.code != 400 {
		t.Fatalf("bad If-Match: %d", r.code)
	}
	ok := call(t, h, http.MethodPost, batchPath, "6", body, hdr("http-key-000000004", "0"))
	d := dataOf(ok)
	if ok.code != 200 || ok.json["outcome"] != "SUCCESS" || d["applied"] != true || d["collection_version"] != float64(1) ||
		d["writeback_state"] != "AWAITING_EVIDENCE_FINALISATION" || d["items"].([]any)[0].(map[string]any)["committed"] != true {
		t.Fatalf("batch: %d %s", ok.code, ok.body)
	}
	batchID := d["batch_id"].(string)
	attemptID := d["items"].([]any)[0].(map[string]any)["attempt_id"].(string)
	// Replay → same result; another body with the key → 409.
	if r := call(t, h, http.MethodPost, batchPath, "6", body, hdr("http-key-000000004", "0")); r.code != 200 || dataOf(r)["replayed"] != true ||
		dataOf(r)["batch_id"] != batchID {
		t.Fatalf("replay: %s", r.body)
	}
	if r := call(t, h, http.MethodPost, batchPath, "6", strings.Replace(body, `"amount":400`, `"amount":300`, 1), hdr("http-key-000000004", "0")); r.code != 409 {
		t.Fatalf("idempotency conflict: %d", r.code)
	}
	// Batch read: the submitter only.
	if r := call(t, h, http.MethodGet, "/api/v1/payments/batches/"+batchID, "6", "", nil); r.code != 200 || len(dataOf(r)["attempts"].([]any)) != 1 {
		t.Fatalf("batch read: %d %s", r.code, r.body)
	}
	if r := call(t, h, http.MethodGet, "/api/v1/payments/batches/"+batchID, "7", "", nil); r.code != 404 {
		t.Fatalf("another seller read the batch: %d", r.code)
	}

	// Finance review queue: evidence not READY yet → D-43 gate; the
	// submitter is never an independent reviewer; no decision is allowed.
	q := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts", "1", "", nil)
	items, _ := q.json["data"].([]any)
	if q.code != 200 || len(items) != 1 {
		t.Fatalf("queue: %d %s", q.code, q.body)
	}
	first := items[0].(map[string]any)
	if first["evidence_ready"] != false || first["reviewer_is_submitter"] != false || !strings.Contains(q.body, `"D-43"`) || !strings.Contains(q.body, `"D-41"`) {
		t.Fatalf("D-43 gate: %s", q.body)
	}
	if _, err := e.svc.Finalise(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	a := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts/"+attemptID, "1", "", nil)
	ad := dataOf(a)
	if a.code != 200 || ad["evidence_ready"] != true || strings.Contains(a.body, `"D-43"`) || !strings.Contains(a.body, `"OPD-08"`) {
		t.Fatalf("ready attempt: %s", a.body)
	}
	for _, v := range a.json["allowed_actions"].(map[string]any) {
		if v.(map[string]any)["allowed"] != false {
			t.Fatalf("a Finance decision was allowed: %s", a.body)
		}
	}
	// The submitter (with review read) sees itself as the maker.
	if m := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts/"+attemptID, "6", "", nil); dataOf(m)["reviewer_is_submitter"] != true ||
		!strings.Contains(m.body, `"UNAUTHORIZED"`) {
		t.Fatalf("maker as checker: %s", m.body)
	}
	if r := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts", "9", "", nil); r.code != 403 {
		t.Fatalf("queue without capability: %d", r.code)
	}
	// No decision, posting, refund, next-stage, gateway or invoice route.
	for _, p := range []string{"/api/v1/finance/review/attempts/" + attemptID + "/decisions", "/api/v1/finance/review/attempts/" + attemptID,
		"/api/v1/payments/attempts/" + attemptID + "/approve", "/api/v1/billing/invoices/" + inv + "/next-stage", "/api/v1/payments/gateway/callback",
		"/api/v1/billing/invoices", "/api/v1/finance/ledger/entries"} {
		if r := call(t, h, http.MethodPost, p, "1", `{}`, hdr("http-key-000000009", "1")); r.code < 400 {
			t.Fatalf("POST %s = %d", p, r.code)
		}
	}
	// Evidence diagnostics read.
	if r := call(t, h, http.MethodGet, "/api/v1/payments/evidence", "6", "", nil); r.code != 200 ||
		dataOf(r)["receipts_by_state"].(map[string]any)["READY"] != float64(1) {
		t.Fatalf("evidence: %d %s", r.code, r.body)
	}
	// Rollback: the write flag off → BLOCKED; history stays readable.
	e.setFlags(t, map[string]bool{"payments.manual_write": false})
	if r := call(t, h, http.MethodPost, batchPath, "6", body, hdr("http-key-000000010", "1")); r.json["outcome"] != "BLOCKED" ||
		dataOf(r)["applied"] != false {
		t.Fatalf("rollback: %s", r.body)
	}
	if e.count(t, `SELECT count(*) FROM payments.submitted_attempt`) != 1 || e.count(t, `SELECT count(*) FROM payments.receipt_state_history`) < 3 {
		t.Fatal("history must survive a rollback")
	}
}
