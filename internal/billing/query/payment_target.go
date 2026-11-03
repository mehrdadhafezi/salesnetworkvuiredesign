package query

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

var _ api.PaymentTargetReader = (*Service)(nil)

// PaymentTarget reads the payment-target evidence of one invoice in a
// billing-owned READ ONLY transaction.
func (s *Service) PaymentTarget(ctx context.Context, invoiceID string) (api.PaymentTarget, bool, error) {
	if !uuidPattern.MatchString(invoiceID) {
		return api.PaymentTarget{}, false, nil
	}
	var t api.PaymentTarget
	found := false
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var presence string
		err := tx.QueryRow(ctx, `SELECT invoice_id::text, source_system, source_spec, source_key, presence, seller_user_id::text, issued_by_user_id::text, current_stage_raw,
			payment_plan_raw, stages_source_complete, version FROM billing.invoice WHERE invoice_id = $1`, invoiceID).
			Scan(&t.InvoiceID, &t.SourceSystem, &t.SourceSpec, &t.SourceKey, &presence, &t.SellerUserID, &t.IssuedByUserID, &t.CurrentStageRaw, &t.PaymentPlanRaw, &t.StagesComplete, &t.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found, t.Present = true, presence == "PRESENT"
		var mapping string
		err = tx.QueryRow(ctx, `SELECT raw_value, mapping FROM billing.invoice_state_evidence WHERE invoice_id = $1 AND field = 'MAIN'`, invoiceID).
			Scan(&t.MainStatusRaw, &mapping)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		t.MainStatusMapping = api.Vocabulary(mapping)
		if errors.Is(err, pgx.ErrNoRows) {
			t.MainStatusMapping = api.EmptyValue
		}
		var amount *int64
		var unit *string
		var repr string
		err = tx.QueryRow(ctx, `SELECT amount, unit, representation FROM billing.amount_evidence WHERE invoice_id = $1 AND kind = 'REMAINING_AMOUNT'`,
			invoiceID).Scan(&amount, &unit, &repr)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err == nil && repr == "REPRESENTABLE" && amount != nil && unit != nil {
			t.Remaining = api.Money{Amount: *amount, Unit: *unit, Representable: true}
		}
		if t.CurrentStageRaw == nil {
			return nil
		}
		no, convErr := strconv.Atoi(*t.CurrentStageRaw)
		if convErr != nil || no < 1 {
			return nil
		}
		rows, err := tx.Query(ctx, `SELECT stage_id::text, state_raw, state_mapping, state_target, requested_amount, unit, requested_repr
			FROM billing.payment_stage WHERE invoice_id = $1 AND presence = 'PRESENT' AND stage_no_raw = $2 ORDER BY stage_id`, invoiceID, strconv.Itoa(no))
		if err != nil {
			return err
		}
		defer rows.Close()
		n := 0
		for rows.Next() {
			st := api.PaymentTargetStage{StageNo: no}
			var amt *int64
			var u *string
			var rp string
			if err := rows.Scan(&st.StageID, &st.StateRaw, &st.StateMapping, &st.StateTarget, &amt, &u, &rp); err != nil {
				return err
			}
			if rp == "REPRESENTABLE" && amt != nil && u != nil {
				st.Requested = api.Money{Amount: *amt, Unit: *u, Representable: true}
			}
			n++
			if n == 1 {
				t.Stage = &st
			}
		}
		if n > 1 {
			t.Stage, t.StageAmbiguous = nil, true
		}
		return rows.Err()
	})
	return t, found, err
}
