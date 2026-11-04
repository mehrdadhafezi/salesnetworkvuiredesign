package activation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Decision is one stored activation decision.
type Decision struct {
	ActivationID, BusinessKey, InvoiceID, LinkState, Route, ProductKey, Decision, Mode, Correlation, MessageID string
	InvoiceRef, CustomerID, SubscriptionID                                                                     *string
	Existing                                                                                                   bool
	ProductVersion, ItemCount                                                                                  int
	Reasons, BlockedBy                                                                                         []string
	DecidedAt                                                                                                  time.Time
}

// Decisions lists activation decisions, newest first (READ ONLY).
func (s *Service) Decisions(ctx context.Context, decision *string, limit int) ([]Decision, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	out := []Decision{}
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT activation_id::text, business_key, invoice_id::text, customer_link_state, route, product_key, decision, rollout_mode,
			correlation_id, message_id::text, invoice_ref, customer_id::text, subscription_id::text, existing_customer, product_version, item_count, reasons,
			blocked_by, decided_at FROM biawin.activation WHERE ($1::text IS NULL OR decision = $1) ORDER BY decided_at DESC, activation_id LIMIT $2`, decision, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d Decision
			var r, b []byte
			if err := rows.Scan(&d.ActivationID, &d.BusinessKey, &d.InvoiceID, &d.LinkState, &d.Route, &d.ProductKey, &d.Decision, &d.Mode, &d.Correlation,
				&d.MessageID, &d.InvoiceRef, &d.CustomerID, &d.SubscriptionID, &d.Existing, &d.ProductVersion, &d.ItemCount, &r, &b, &d.DecidedAt); err != nil {
				return err
			}
			_ = json.Unmarshal(r, &d.Reasons)
			_ = json.Unmarshal(b, &d.BlockedBy)
			out = append(out, d)
		}
		return rows.Err()
	})
	return out, err
}
