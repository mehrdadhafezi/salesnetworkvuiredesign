package query

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/domain"
)

// CaseRow is one case fact. Ownership facets stay separate: current custody
// lives in the custody report; next actor is not modelled in Slice 8.
type CaseRow struct {
	CaseID, Workflow, StateNamespace string
	StateRaw                         *string
	StateMapping                     string
	CustomerLinkState                string
	Presence                         string
	SourceSpec, SourceKey            string
	AliasCount                       int64
	FirstSeenAt                      time.Time
	OriginalOwnerResolution          string
	OriginalOwnerKey                 *string
	OriginalSellerResolution         string
	OriginalSellerKey                *string
	CreditOwnerResolution            string
}

// CustodyRow is one derived custody projection fact.
type CustodyRow struct {
	CaseID, Derivation, Parity string
	DerivedKey, LegacyKey      *string
	EventCount                 int
	FindingCount               int64
	Presence                   string
}

// StateCell is one raw invoice state field with its vocabulary membership.
type StateCell struct {
	Raw     *string
	Mapping string
}

// UnitRow is one invoice's total evidence in one unit (no canonical total).
type UnitRow struct {
	Unit                                       string
	DeclaredFinal, DeclaredPayment             *int64
	ItemSum, StageSum, PaymentSum              *int64
	ItemState, StageState, PaymentState, Class string
}

// InvoiceRow is one invoice fact; every state field stays separate.
type InvoiceRow struct {
	InvoiceID, SourceSpec, SourceKey string
	CodeRaw                          *string
	Presence                         string
	CustomerLinkState, CaseLinkState string
	SellerKey                        *string
	Main, Invoice, Payment           StateCell
	Workflow, Return                 StateCell
	UnknownStates                    int64
	ItemCount, StageCount            int64
	ItemsComplete, StagesComplete    bool
	FirstSeenAt                      time.Time
	Units                            []UnitRow
}

// PaymentRow is one payment attempt fact.
type PaymentRow struct {
	AttemptID                 string
	InvoiceID                 *string
	InvoiceLinkState, Channel string
	StateRaw                  *string
	StateMapping              string
	StateTarget               *string
	Provenance                string
	AmountRaw                 *string
	Amount                    *int64
	Unit                      *string
	AmountRepr                string
	GatewayState              *string
	Presence                  string
	FirstSeenAt               time.Time
}

var selects = map[string]string{
	domain.ReportCases: `f.case_id::text, f.workflow_type, f.state_namespace, f.state_raw, f.state_mapping, f.customer_link_state, f.presence,
		f.source_spec, f.source_key, f.alias_count, f.first_seen_at, f.original_owner_resolution, f.original_owner_key,
		f.original_seller_resolution, f.original_seller_key, f.credit_owner_resolution`,
	domain.ReportCustody: `f.case_id::text, f.derivation, f.parity, f.derived_key, f.legacy_key, f.event_count, f.finding_count, f.presence`,
	domain.ReportInvoices: `f.invoice_id::text, f.source_spec, f.source_key, f.code_raw, f.presence, f.customer_link_state, f.case_link_state,
		f.seller_key, f.main_raw, f.main_mapping, f.invoice_raw, f.invoice_mapping, f.payment_raw, f.payment_mapping, f.workflow_raw,
		f.workflow_mapping, f.return_raw, f.return_mapping, f.unknown_state_count, f.item_count, f.stage_count, f.items_complete,
		f.stages_complete, f.first_seen_at`,
	domain.ReportPayments: `f.attempt_id::text, f.invoice_id::text, f.invoice_link_state, f.channel, f.state_raw, f.state_mapping, f.state_target,
		f.approval_provenance, f.amount_raw, f.amount, f.unit, f.amount_repr, f.gateway_state, f.presence, f.first_seen_at`,
}

func scanRow(report string, r pgx.Rows) (any, error) {
	switch report {
	case domain.ReportCases:
		var x CaseRow
		err := r.Scan(&x.CaseID, &x.Workflow, &x.StateNamespace, &x.StateRaw, &x.StateMapping, &x.CustomerLinkState, &x.Presence, &x.SourceSpec,
			&x.SourceKey, &x.AliasCount, &x.FirstSeenAt, &x.OriginalOwnerResolution, &x.OriginalOwnerKey, &x.OriginalSellerResolution,
			&x.OriginalSellerKey, &x.CreditOwnerResolution)
		return x, err
	case domain.ReportCustody:
		var x CustodyRow
		err := r.Scan(&x.CaseID, &x.Derivation, &x.Parity, &x.DerivedKey, &x.LegacyKey, &x.EventCount, &x.FindingCount, &x.Presence)
		return x, err
	case domain.ReportInvoices:
		var x InvoiceRow
		err := r.Scan(&x.InvoiceID, &x.SourceSpec, &x.SourceKey, &x.CodeRaw, &x.Presence, &x.CustomerLinkState, &x.CaseLinkState, &x.SellerKey,
			&x.Main.Raw, &x.Main.Mapping, &x.Invoice.Raw, &x.Invoice.Mapping, &x.Payment.Raw, &x.Payment.Mapping, &x.Workflow.Raw,
			&x.Workflow.Mapping, &x.Return.Raw, &x.Return.Mapping, &x.UnknownStates, &x.ItemCount, &x.StageCount, &x.ItemsComplete,
			&x.StagesComplete, &x.FirstSeenAt)
		return x, err
	default:
		var x PaymentRow
		err := r.Scan(&x.AttemptID, &x.InvoiceID, &x.InvoiceLinkState, &x.Channel, &x.StateRaw, &x.StateMapping, &x.StateTarget, &x.Provenance,
			&x.AmountRaw, &x.Amount, &x.Unit, &x.AmountRepr, &x.GatewayState, &x.Presence, &x.FirstSeenAt)
		return x, err
	}
}

func rowID(row any) string {
	switch x := row.(type) {
	case CaseRow:
		return x.CaseID
	case CustodyRow:
		return x.CaseID
	case InvoiceRow:
		return x.InvoiceID
	case PaymentRow:
		return x.AttemptID
	}
	return ""
}

// rowsOf pages the members of a spec (keyset on the native id), or reads one
// row by id (byID; membership is then checked separately).
func rowsOf(ctx context.Context, tx db.Tx, spec Spec, runID string, after *string, limit int, byID *string) ([]any, error) {
	def := reports[spec.Report]
	w, args := where(spec, runID)
	if after != nil {
		args = append(args, *after)
		w += fmt.Sprintf(" AND f.%s > $%d", def.id, len(args))
	}
	if byID != nil {
		args = append(args, *byID)
		w += fmt.Sprintf(" AND f.%s = $%d", def.id, len(args))
	}
	args = append(args, limit)
	rows, err := tx.Query(ctx, fmt.Sprintf("SELECT %s FROM %s f WHERE %s ORDER BY f.%s LIMIT $%d", selects[spec.Report], def.table, w, def.id, len(args)), args...)
	if err != nil {
		return nil, err
	}
	var out []any
	for rows.Next() {
		x, err := scanRow(spec.Report, rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if spec.Report == domain.ReportInvoices && len(out) > 0 {
		if err := attachUnits(ctx, tx, runID, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func attachUnits(ctx context.Context, tx db.Tx, runID string, rows []any) error {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.(InvoiceRow).InvoiceID
	}
	q, err := tx.Query(ctx, `SELECT invoice_id::text, unit, declared_final, declared_payment, item_sum, item_state, stage_sum, stage_state,
		payment_sum, payment_state, total_class FROM reporting.invoice_unit_fact WHERE run_id = $1 AND invoice_id = ANY($2::uuid[]) ORDER BY unit`,
		runID, ids)
	if err != nil {
		return err
	}
	defer q.Close()
	units := map[string][]UnitRow{}
	for q.Next() {
		var id string
		var u UnitRow
		if err := q.Scan(&id, &u.Unit, &u.DeclaredFinal, &u.DeclaredPayment, &u.ItemSum, &u.ItemState, &u.StageSum, &u.StageState,
			&u.PaymentSum, &u.PaymentState, &u.Class); err != nil {
			return err
		}
		units[id] = append(units[id], u)
	}
	if err := q.Err(); err != nil {
		return err
	}
	for i, r := range rows {
		x := r.(InvoiceRow)
		x.Units = units[x.InvoiceID]
		rows[i] = x
	}
	return nil
}
