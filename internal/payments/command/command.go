// Package command holds the payments module's Slice 12 controlled financial
// writes (Architecture V1 §15):
//
//   - Upload: stage one receipt file into durable quarantine storage; the
//     upload row is written only after the bytes are durable;
//   - SubmitBatch: one owning-seller manual (receipt-mode) payment batch for
//     one invoice's current stage, all-or-nothing in ONE payments
//     transaction, with per-item outcomes;
//   - Finaliser: post-commit evidence finalisation, evidence reconciliation
//     and unbound-upload expiry (D-44 safe default);
//   - Gateway: the synthetic-only gateway attempt flow (no route).
//
// Every decision that gates a write is evaluated inside the owner
// transaction after the per-stage collection account row is locked
// (apply-time recheck). The payments module writes only its own schema; it
// reads billing through billing's published PaymentTargetReader and never
// changes an invoice, a stage, the ledger, custody, a case, a customer, a
// subscription or a commission. A refused command commits nothing.
//
// Decisions are never resolved here: info-only submission (D-43), Finance
// review (D-41, D-43, OPD-08), assist submission (OPD-10), rounding and unit
// conversion (OPD-06) and the customer channel (§15.10, D-26) stay BLOCKED.
package command

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"

	"github.com/jackc/pgx/v5"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/objectstore"
)

// Limits from the 2.0.160 receipt upload contract (SN_Helpers::upload_receipt).
const (
	MaxUploadBytes = 5 << 20
	MaxBatchBytes  = 20 << 20
	MaxItems       = 10
	MaxReceipts    = 10
)

// Decision and policy ids named by refusals.
const (
	DecisionD43       = "D-43"
	DecisionOPD06     = "OPD-06"
	DecisionOPD08     = "OPD-08"
	DecisionOPD10     = "OPD-10"
	DecisionD25       = "D-25"
	DecisionD26       = "D-26"
	ShadowUnsupported = "SHADOW_COMPARE_UNSUPPORTED"
)

var (
	// ErrInvalid reports a malformed command (400).
	ErrInvalid = errors.New("payments/command: invalid command")
	// ErrNotFound reports an unknown invoice, batch or upload (404).
	ErrNotFound = errors.New("payments/command: not found")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	keyPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
	itemKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Actor is who issues a command.
type Actor struct {
	Principal string
	UserID    string
}

// Service runs the payments commands.
type Service struct {
	runner  *db.Runner
	store   objectstore.Store
	targets billingapi.PaymentTargetReader
	clock   timeutil.Clock
}

// New returns the service over a payments-owned runner.
func New(runner *db.Runner, store objectstore.Store, targets billingapi.PaymentTargetReader, clock timeutil.Clock) (*Service, error) {
	if runner == nil || runner.Module() != payments.Module || store == nil || targets == nil || clock == nil {
		return nil, errors.New("payments/command: runner must be owned by payments; store, billing targets and clock are required")
	}
	return &Service{runner: runner, store: store, targets: targets, clock: clock}, nil
}

// Stage states that accept a new manual payment (2.0.160: a stage that is
// paid / approved refuses; pending review and rejected stages accept a new
// transaction). CANCELLED and unmapped states refuse.
var payableStage = map[string]bool{"PENDING": true, "AWAITING_REVIEW": true, "REJECTED": true}

// Legacy invoice main statuses that refuse a payment (2.0.160
// sn_save_invoice_manual_payment).
var terminalInvoice = map[string]bool{"paid": true, "approved": true, "cancelled": true, "payment_archived": true}

// Eligibility is the server's evaluation of whether the actor may submit a
// manual payment for the invoice's current stage now. The same evaluation
// gates the command (inside its transaction) and produces allowed_actions.
type Eligibility struct {
	Status    result.Status
	Reasons   []reasoncodes.Code
	BlockedBy []string
	Target    billingapi.PaymentTarget
	StageNo   int
	StageID   string
	Unit      string
	Due       int64
	Remaining int64
	Used      int64
	Available int64
	// Known is true when Due/Remaining/Used/Available were evaluated.
	Known bool
}

func refusal(status result.Status, code reasoncodes.Code, blockedBy ...string) Eligibility {
	return Eligibility{Status: status, Reasons: []reasoncodes.Code{code}, BlockedBy: blockedBy}
}

// evaluate decides eligibility inside tx (a payments transaction). found is
// false for an unknown invoice.
func (s *Service) evaluate(ctx context.Context, tx db.Tx, invoiceID string, actor Actor) (Eligibility, bool, error) {
	t, ok, err := s.targets.PaymentTarget(ctx, invoiceID)
	if err != nil || !ok {
		return Eligibility{}, ok, err
	}
	e, err := s.evaluateTarget(ctx, tx, t, actor)
	e.Target = t
	return e, true, err
}

func (s *Service) evaluateTarget(ctx context.Context, tx db.Tx, t billingapi.PaymentTarget, actor Actor) (Eligibility, error) {
	if !t.Present {
		return refusal(result.Blocked, reasoncodes.StaleState), nil
	}
	if actor.UserID == "" {
		return refusal(result.Unauthorized, reasoncodes.RowScopeDenied), nil
	}
	// Row scope (2.0.160): the owning seller of the invoice. A conversion
	// invoice is payable by the seller only when the seller issued it; the
	// target cannot tell conversion invoices apart, so the issuer must be the
	// seller (fail closed). Hierarchy, Finance and administrators never
	// substitute (OPD-10, OPD-08).
	if t.SellerUserID == nil || t.IssuedByUserID == nil {
		return refusal(result.Blocked, reasoncodes.DependencyUnknown), nil
	}
	if *t.SellerUserID != actor.UserID || *t.IssuedByUserID != actor.UserID {
		return refusal(result.Unauthorized, reasoncodes.RowScopeDenied, DecisionOPD10), nil
	}
	if !t.StagesComplete {
		return refusal(result.Blocked, reasoncodes.CoverageIncomplete), nil
	}
	if t.MainStatusMapping != billingapi.KnownVocabulary || t.MainStatusRaw == nil {
		return refusal(result.Blocked, reasoncodes.DependencyUnknown), nil
	}
	if terminalInvoice[*t.MainStatusRaw] {
		return refusal(result.Blocked, reasoncodes.ActionDenied), nil
	}
	if t.StageAmbiguous {
		return refusal(result.Blocked, reasoncodes.LinkageConflict), nil
	}
	if t.Stage == nil {
		return refusal(result.Blocked, reasoncodes.DependencyUnknown), nil
	}
	st := *t.Stage
	if st.StateMapping != "MAPPED" || st.StateTarget == nil {
		return refusal(result.Blocked, reasoncodes.DependencyUnknown), nil
	}
	if !payableStage[*st.StateTarget] {
		return refusal(result.Blocked, reasoncodes.ActionDenied), nil
	}
	if !st.Requested.Representable || !t.Remaining.Representable {
		return refusal(result.Blocked, reasoncodes.CoverageIncomplete), nil
	}
	if st.Requested.Unit != t.Remaining.Unit {
		return refusal(result.Blocked, reasoncodes.PolicyUnresolved, DecisionOPD06), nil
	}
	e := Eligibility{StageNo: st.StageNo, StageID: st.StageID, Unit: st.Requested.Unit, Due: st.Requested.Amount, Remaining: t.Remaining.Amount}
	if e.Due <= 0 || e.Due > e.Remaining {
		e.Status, e.Reasons = result.Blocked, []reasoncodes.Code{reasoncodes.ActionDenied}
		return e, nil
	}
	used, code, blocked, err := s.used(ctx, tx, t.InvoiceID, st.StageNo, e.Unit)
	if err != nil {
		return Eligibility{}, err
	}
	if code != "" {
		e.Status, e.Reasons, e.BlockedBy = result.Blocked, []reasoncodes.Code{code}, blocked
		return e, nil
	}
	e.Used, e.Available, e.Known = used, e.Due-used, true
	if e.Available <= 0 {
		e.Status, e.Reasons = result.Blocked, []reasoncodes.Code{reasoncodes.ActionDenied}
		return e, nil
	}
	e.Status, e.Reasons = result.Success, []reasoncodes.Code{}
	return e, nil
}

// used sums the stage's pending and accepted attempts from payments' own
// data (Architecture V1 §15.2 rule 3): target submissions plus imported
// legacy attempts. Anything it cannot classify exactly blocks (fail closed):
// an unlinked imported attempt, an unmapped state, a row without a stage, an
// unrepresentable amount or another unit (no conversion, OPD-06).
func (s *Service) used(ctx context.Context, tx db.Tx, invoiceID string, stageNo int, unit string) (int64, reasoncodes.Code, []string, error) {
	var unlinked int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM payments.payment_attempt WHERE invoice_link_state <> 'RESOLVED'`).Scan(&unlinked); err != nil {
		return 0, "", nil, err
	}
	if unlinked > 0 {
		return 0, reasoncodes.CoverageIncomplete, nil, nil
	}
	var total int64
	rows, err := tx.Query(ctx, `SELECT unit, sum(amount) FROM payments.submitted_attempt WHERE invoice_id = $1 AND stage_no = $2 GROUP BY unit`,
		invoiceID, stageNo)
	if err != nil {
		return 0, "", nil, err
	}
	for rows.Next() {
		var u string
		var sum int64
		if err := rows.Scan(&u, &sum); err != nil {
			rows.Close()
			return 0, "", nil, err
		}
		if u != unit {
			rows.Close()
			return 0, reasoncodes.PolicyUnresolved, []string{DecisionOPD06}, nil
		}
		total += sum
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, "", nil, err
	}
	// Target gateway requests (synthetic-only in Slice 12): a VERIFIED one is
	// an accepted payment; an outcome that is not known blocks (reconcile
	// first); a redirect without a result is not a registered payment
	// (2.0.160 summary semantics).
	grows, err := tx.Query(ctx, `SELECT state, amount, unit FROM payments.gateway_request WHERE invoice_id = $1 AND stage_no = $2`, invoiceID, stageNo)
	if err != nil {
		return 0, "", nil, err
	}
	for grows.Next() {
		var st, u string
		var amount int64
		if err := grows.Scan(&st, &amount, &u); err != nil {
			grows.Close()
			return 0, "", nil, err
		}
		switch st {
		case "VERIFIED":
			if u != unit {
				grows.Close()
				return 0, reasoncodes.PolicyUnresolved, []string{DecisionOPD06}, nil
			}
			total += amount
		case "UNKNOWN", "RECONCILIATION_REQUIRED":
			grows.Close()
			return 0, reasoncodes.OutcomeUnknown, nil, nil
		}
	}
	grows.Close()
	if err := grows.Err(); err != nil {
		return 0, "", nil, err
	}
	stage := fmt.Sprint(stageNo)
	rows, err = tx.Query(ctx, `SELECT stage_no_raw, channel, state_mapping, state_target, amount, unit FROM payments.payment_attempt
		WHERE invoice_id = $1 AND presence = 'PRESENT' AND (stage_no_raw IS NULL OR stage_no_raw = $2)`, invoiceID, stage)
	if err != nil {
		return 0, "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			stageRaw, target, u *string
			channel, mapping    string
			amount              *int64
		)
		if err := rows.Scan(&stageRaw, &channel, &mapping, &target, &amount, &u); err != nil {
			return 0, "", nil, err
		}
		if mapping != "MAPPED" || target == nil {
			return 0, reasoncodes.DependencyUnknown, nil, nil
		}
		switch *target {
		case "REJECTED", "CANCELLED":
			continue
		case "RECONCILIATION_REQUIRED":
			return 0, reasoncodes.DependencyUnknown, nil, nil
		case "PENDING":
			// An unfinished gateway request is not a registered transfer
			// (2.0.160 SN_Payment_Transactions::summary).
			if channel == "GATEWAY" {
				continue
			}
		}
		if stageRaw == nil {
			return 0, reasoncodes.DependencyUnknown, nil, nil
		}
		if amount == nil || u == nil {
			return 0, reasoncodes.CoverageIncomplete, nil, nil
		}
		if *u != unit {
			return 0, reasoncodes.PolicyUnresolved, []string{DecisionOPD06}, nil
		}
		total += *amount
	}
	return total, "", nil, rows.Err()
}

// lockAccount creates (if absent) and locks the per (invoice, stage)
// collection account; it returns the current version.
func lockAccount(ctx context.Context, tx db.Tx, invoiceID string, stageNo int, at any) (int64, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO payments.collection_account (invoice_id, stage_no, version, created_at, updated_at)
		VALUES ($1, $2, 0, $3, $3) ON CONFLICT (invoice_id, stage_no) DO NOTHING`, invoiceID, stageNo, at); err != nil {
		return 0, err
	}
	var v int64
	err := tx.QueryRow(ctx, `SELECT version FROM payments.collection_account WHERE invoice_id = $1 AND stage_no = $2 FOR UPDATE`,
		invoiceID, stageNo).Scan(&v)
	return v, err
}

// accountVersion reads a collection account version without locking (0
// when it does not exist yet).
func accountVersion(ctx context.Context, tx db.Tx, invoiceID string, stageNo int) (int64, error) {
	var v int64
	err := tx.QueryRow(ctx, `SELECT version FROM payments.collection_account WHERE invoice_id = $1 AND stage_no = $2`, invoiceID, stageNo).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return v, err
}

// severity orders refusals for a batch-level status.
var severity = map[result.Status]int{result.Unauthorized: 3, result.Conflict: 2, result.Blocked: 1}

func uniqueCodes(in []reasoncodes.Code) []reasoncodes.Code {
	seen := map[reasoncodes.Code]bool{}
	out := []reasoncodes.Code{}
	for _, c := range in {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, c := range in {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

func hexOf(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, 2*len(b))
	for i, v := range b {
		out[2*i], out[2*i+1] = h[v>>4], h[v&15]
	}
	return string(out)
}
