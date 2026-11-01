package imports_test

import (
	"context"
	"crypto/sha256"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/imports"
	paymentsmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// All data is SYNTHETIC.
const (
	b1      = "11111111-1111-4111-8111-111111111111"
	b2      = "22222222-2222-4222-8222-222222222222"
	b3      = "33333333-3333-4333-8333-333333333333"
	invSpec = "synthetic.invoice.v2"
	paySpec = "synthetic.payment.v1"
	stgSpec = "synthetic.invoice_payment_stage.v1"
	inv1    = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1"
	inv2    = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2"
)

// billingStub is the published billing read surface; invoice key 3 is
// unknown to billing.
type billingStub struct {
	calls int
	extra map[string]string // invoice keys resolved later
}

func (b *billingStub) ResolveInvoices(_ context.Context, _, spec string, keys []string) (map[string]billingapi.SourceInvoice, error) {
	b.calls++
	out := map[string]billingapi.SourceInvoice{}
	for _, k := range keys {
		switch {
		case spec == invSpec && k == "1":
			out[k] = billingapi.SourceInvoice{InvoiceID: inv1, Present: true}
		case spec == invSpec && k == "2":
			out[k] = billingapi.SourceInvoice{InvoiceID: inv2, Present: true}
		case spec == invSpec && b.extra[k] != "":
			out[k] = billingapi.SourceInvoice{InvoiceID: b.extra[k], Present: true}
		}
	}
	return out, nil
}

func (b *billingStub) RowProvenance(context.Context, string, string, []string) (map[string]billingapi.RowProvenance, error) {
	return nil, nil
}

func (b *billingStub) Coverage(context.Context) (billingapi.Coverage, error) {
	return billingapi.Coverage{}, nil
}

type env struct {
	*pgtest.Env
	admin   *pgx.Conn
	cmd     *imports.Command
	q       *query.Service
	billing *billingStub
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, paymentsmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:00:00Z")
	r := e.Runner(t, payments.Module)
	stub := &billingStub{extra: map[string]string{}}
	return &env{Env: e, admin: admin, cmd: imports.New(r, stub, timeutil.FixedClock{At: at}), q: query.New(r), billing: stub}
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func (e *env) fingerprint(t *testing.T) string {
	t.Helper()
	var s string
	if err := e.admin.QueryRow(context.Background(), `SELECT md5(
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY attempt_id), '') FROM payments.payment_attempt t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY attempt_id), '') FROM payments.gateway_attempt t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY history_id), '') FROM payments.attempt_state_history t) ||
		(SELECT COALESCE(string_agg(t::text, '|' ORDER BY receipt_id), '') FROM payments.receipt t))`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func src(batch, spec, key, content string) api.SourceRef {
	sum := sha256.Sum256([]byte(spec + "/" + key + "/" + content))
	return api.SourceRef{System: "wp", Spec: spec, Key: key, BatchID: batch, RowSHA256: sum[:]}
}

func str(s string) *string { return &s }

func hash(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }

// attempt mirrors the connector's explicit mapping table for the test.
func attempt(batch, key, inv, raw, amount string, gateway bool) api.AttemptRecord {
	targets := map[string]string{"pending": "PENDING", "approved": "ACCEPTED", "rejected": "REJECTED", "cancelled": "CANCELLED",
		"reconcile_error": "RECONCILIATION_REQUIRED"}
	a := api.AttemptRecord{Source: src(batch, paySpec, key, raw+amount), Invoice: api.RowRef{Spec: invSpec, Key: inv},
		StageNoRaw: str("1"), Channel: api.ChannelManualCard, PayMethodRaw: str("card"),
		Amount: api.RawMoney{Raw: str(amount), Unit: "IRT"}, StateNamespace: "legacy_payment.status", StateRaw: str(raw),
		ApprovalProvenance: api.ProvNotApproved}
	if t, ok := targets[raw]; ok {
		a.StateTarget = str(t)
	}
	if gateway {
		a.Channel, a.PayMethodRaw = api.ChannelGateway, str("gateway")
		a.Gateway = &api.GatewayRecord{ProviderRaw: str("zibal"), AuthoritySHA256: hash("authority-" + key), StateRaw: str(raw)}
	}
	if raw == "approved" {
		a.ApprovalProvenance = api.ProvLegacyManualApproval
		if gateway {
			a.ApprovalProvenance = api.ProvGatewayAuto
		}
	}
	return a
}

func receipt(batch, key, inv string, idx int) api.ReceiptRecord {
	return api.ReceiptRecord{Source: src(batch, stgSpec, key, "r"), OwnerKind: "STAGE", Invoice: api.RowRef{Spec: invSpec, Key: inv},
		StageNoRaw: str("1"), RefIndex: idx, RefSHA256: hash("ref-" + key + strconv.Itoa(idx)), Host: str("files.example.test"),
		Extension: str("jpg"), State: api.ReceiptReferenceOnly}
}

func base(batch string) api.PaymentsImport {
	return api.PaymentsImport{BatchID: batch, AttemptsComplete: true, ReceiptSourcesComplete: true,
		Attempts: []api.AttemptRecord{
			attempt(batch, "1", "1", "approved", "600", true),       // gateway auto-approval
			attempt(batch, "2", "1", "approved", "400.00", false),   // legacy manual approval
			attempt(batch, "3", "1", "reconcile_error", "50", true), // visible, not counted
			attempt(batch, "4", "2", "verified", "70", true),        // ambiguous → UNKNOWN
			attempt(batch, "5", "2", "rejected", "70", false),       // not counted
			attempt(batch, "6", "3", "pending", "10", false),        // invoice unknown to billing
		},
		Receipts: []api.ReceiptRecord{receipt(batch, "1", "1", 0), receipt(batch, "1", "1", 1)},
	}
}

func (e *env) run(t *testing.T, in api.PaymentsImport) api.ImportReport {
	t.Helper()
	in.SnapshotID, in.SourceSystem = "synthetic-payments-01", "wp"
	rep, err := e.cmd.ImportPayments(context.Background(), in)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return rep
}

// Raw states, provenance, channel and gateway evidence stay separate;
// approved is never Finance-reviewed; reconcile_error stays visible.
func TestAttemptSemantics(t *testing.T) {
	e := setup(t)
	rep := e.run(t, base(b1))
	if rep.Outcome != result.Partial || rep.Counts["attempts_invoice_unresolved"] != 1 {
		t.Fatalf("an unresolved invoice must be PARTIAL: %+v", rep)
	}
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'approved' AND channel = 'GATEWAY' AND approval_provenance = 'GATEWAY_AUTO'`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'approved' AND channel = 'MANUAL_CARD' AND approval_provenance = 'LEGACY_MANUAL_APPROVAL'`) != 1 {
		t.Fatal("approval provenance must be explicit")
	}
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'reconcile_error' AND state_target = 'RECONCILIATION_REQUIRED'`) != 1 {
		t.Fatal("reconcile_error must stay visible")
	}
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE state_raw = 'verified' AND state_mapping = 'UNKNOWN' AND state_target IS NULL`) != 1 {
		t.Fatal("ambiguous raw state must be UNKNOWN")
	}
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE invoice_link_state = 'UNKNOWN' AND invoice_id IS NULL`) != 1 {
		t.Fatal("unresolved invoice must stay UNKNOWN")
	}
	if e.count(t, `SELECT count(*) FROM payments.gateway_attempt WHERE octet_length(authority_sha256) = 32`) != 3 {
		t.Fatal("gateway authority stored as hash only")
	}
	cov, err := e.q.Coverage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cov.ApprovedRaw != 2 || cov.ReconcileErrorRaw != 1 || cov.UnknownRaw["verified"] != 1 || cov.ByProvenance["GATEWAY_AUTO"] != 1 ||
		cov.WithoutInvoice != 1 || cov.ReceiptsReferenced != 2 {
		t.Fatalf("coverage: %+v", cov)
	}
	// Contract: GATEWAY_AUTO requires the gateway channel; provenance only on ACCEPTED.
	bad := base(b2)
	bad.Attempts[1].ApprovalProvenance = api.ProvGatewayAuto
	if err := bad.Validate(); err == nil {
		t.Fatal("GATEWAY_AUTO on a manual attempt must be rejected")
	}
	bad = base(b2)
	bad.Attempts[4].ApprovalProvenance = api.ProvLegacyManualApproval
	if err := bad.Validate(); err == nil {
		t.Fatal("approval provenance on a non-accepted attempt must be rejected")
	}
}

// Payment sums count only explicitly ACCEPTED attempts; UNKNOWN attempts or
// an incomplete source make the invoice INCOMPLETE (never zero).
func TestAcceptedSums(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	sums, err := e.q.AcceptedSums(context.Background(), []string{inv1, inv2})
	if err != nil {
		t.Fatal(err)
	}
	p1 := sums[inv1]
	if !p1.Complete || p1.Units["IRT"].State != "COMPLETE" || *p1.Units["IRT"].Amount != 1000 || p1.Units["IRT"].Accepted != 2 {
		t.Fatalf("accepted sum: %+v", p1)
	}
	p2 := sums[inv2]
	if p2.Complete || p2.Unclassified != 1 || len(p2.Units) != 0 {
		t.Fatalf("unknown attempt must make the invoice incomplete: %+v", p2)
	}
	in := base(b2)
	in.AttemptsComplete = false
	e.run(t, in)
	sums, _ = e.q.AcceptedSums(context.Background(), []string{inv1})
	if sums[inv1].Complete || sums[inv1].Units["IRT"].State != "INCOMPLETE" {
		t.Fatalf("incomplete source: %+v", sums[inv1])
	}
}

// Replay is idempotent; a changed state appends history (regression when an
// accepted attempt is no longer accepted); history is append-only; only a
// COMPLETE source marks absence; nothing is deleted.
func TestReplayHistoryAbsence(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	fp := e.fingerprint(t)
	if r := e.run(t, base(b1)); !r.Replay || e.fingerprint(t) != fp {
		t.Fatalf("replay changed state: %+v", r)
	}
	in := base(b2)
	in.Attempts[1] = attempt(b2, "2", "1", "rejected", "400.00", false)
	in.Attempts = in.Attempts[:5] // attempt 6 gone
	in.Receipts = in.Receipts[:1]
	r := e.run(t, in)
	if r.Counts["attempts_state_regression"] != 1 || r.Counts["attempts_marked_absent"] != 1 || r.Counts["receipts_marked_absent"] != 1 {
		t.Fatalf("refresh/absence: %+v", r)
	}
	if e.count(t, `SELECT count(*) FROM payments.attempt_state_history WHERE old_state_raw = 'approved' AND new_state_raw = 'rejected' AND regression`) != 1 ||
		e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE source_key = '2' AND version = 2`) != 1 {
		t.Fatal("state change must be recorded and versioned")
	}
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt`) != 6 || e.count(t, `SELECT count(*) FROM payments.receipt`) != 2 {
		t.Fatal("nothing may be deleted")
	}
	ctx := context.Background()
	if _, err := e.admin.Exec(ctx, `UPDATE payments.attempt_state_history SET regression = false`); err == nil {
		t.Fatal("history must be append-only (UPDATE)")
	}
	if _, err := e.admin.Exec(ctx, `DELETE FROM payments.attempt_state_history`); err == nil {
		t.Fatal("history must be append-only (DELETE)")
	}
	// Incomplete source: no absence.
	in3 := base(b3)
	in3.AttemptsComplete, in3.ReceiptSourcesComplete = false, false
	in3.Attempts, in3.Receipts = in3.Attempts[:1], nil
	before := e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE presence = 'ABSENT_IN_LATEST'`)
	if r := e.run(t, in3); r.Outcome != result.Partial || e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE presence = 'ABSENT_IN_LATEST'`) != before {
		t.Fatalf("incomplete source must not mark absence: %+v", r)
	}
}

// Attempts, history and receipts paginate beyond one page.
func TestPagination(t *testing.T) {
	e := setup(t)
	in := base(b1)
	for i := 10; i < 17; i++ {
		in.Attempts = append(in.Attempts, attempt(b1, strconv.Itoa(i), "1", "pending", "1", false))
		in.Receipts = append(in.Receipts, receipt(b1, "2", "1", i))
	}
	e.run(t, in)
	two := 2
	n := 0
	var cur *string
	for {
		p, err := e.q.ListAttempts(context.Background(), store.AttemptFilter{}, cur, &two)
		if err != nil {
			t.Fatal(err)
		}
		n += len(p.Attempts)
		if cur = p.NextCursor; cur == nil {
			break
		}
	}
	if n != 13 {
		t.Fatalf("attempts seen %d", n)
	}
	n = 0
	for {
		p, err := e.q.ReceiptsOf(context.Background(), inv1, cur, &two)
		if err != nil {
			t.Fatal(err)
		}
		n += len(p.Receipts)
		if cur = p.NextCursor; cur == nil {
			break
		}
	}
	if n != 9 {
		t.Fatalf("receipts seen %d", n)
	}
	gw := api.ChannelGateway
	p, err := e.q.ListAttempts(context.Background(), store.AttemptFilter{Channel: &gw}, nil, nil)
	if err != nil || len(p.Attempts) != 3 || p.Attempts[0].Gateway == nil {
		t.Fatalf("channel filter: %v", err)
	}
	bad := "PHONE"
	if _, err := e.q.ListAttempts(context.Background(), store.AttemptFilter{Channel: &bad}, nil, nil); err == nil {
		t.Fatal("non-allowlisted filter value must be rejected")
	}
}

func (b *billingStub) TotalClasses(context.Context, []string, map[string]billingapi.PaymentEvidence) (map[string]map[string]string, error) {
	return nil, nil
}

// Projections refresh independently of the source checksum: an attempt
// whose invoice is imported later gains its link on replay, and a receipt
// whose stage moved to another invoice follows it with the same reference.
func TestReplayRefreshesLinksAndReceiptOwnership(t *testing.T) {
	e := setup(t)
	e.run(t, base(b1))
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE source_key = '6' AND invoice_link_state = 'UNKNOWN'`) != 1 {
		t.Fatal("precondition: attempt 6 unresolved")
	}
	inv3 := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa3"
	e.billing.extra["3"] = inv3
	in := base(b1)
	in.Receipts[0].Invoice = api.RowRef{Spec: invSpec, Key: "2"}
	in.Receipts[0].StageNoRaw = str("2")
	r := e.run(t, in)
	if r.Counts["attempts_refreshed"] != 1 || r.Counts["receipts_refreshed"] != 1 || r.Outcome != result.Success {
		t.Fatalf("refresh: %+v", r)
	}
	if e.count(t, `SELECT count(*) FROM payments.payment_attempt WHERE source_key = '6' AND invoice_id = $1 AND invoice_link_state = 'RESOLVED'`, inv3) != 1 {
		t.Fatal("attempt link must be repaired on replay")
	}
	if e.count(t, `SELECT count(*) FROM payments.receipt WHERE ref_index = 0 AND invoice_id = $1 AND stage_no_raw = '2'`, inv2) != 1 {
		t.Fatal("receipt ownership must follow the source row")
	}
	if e.count(t, `SELECT count(*) FROM payments.attempt_state_history`) != 0 {
		t.Fatal("a link refresh is not a state change")
	}
	fp := e.fingerprint(t)
	if r := e.run(t, in); !r.Replay || e.fingerprint(t) != fp {
		t.Fatalf("second replay must change nothing: %+v", r)
	}
}
