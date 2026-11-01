// Package http adapts the billing read service to the generated OpenAPI
// strict server. Handlers guard (flag + evaluator), call the read services
// and map rows to contract types. No SQL, no writes, no import command
// (lint). Payment sums and coverage come from the published payments read
// API; billing never reads payments tables.
//
// Nothing here names a canonical invoice status (TAD #20) or a canonical
// total (TAD #19); F02 is NOT_EVALUATED with a null count, never zero.
package http

import (
	"context"
	"errors"
	"sort"
	"strconv"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/store"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the invoice read operations and the reconciliation report.
type Handlers struct {
	guard    *authorization.Guard
	q        *query.Service
	payments paymentsapi.Reader
}

// New returns the handlers. payments is the published payments read API.
func New(guard *authorization.Guard, q *query.Service, payments paymentsapi.Reader) *Handlers {
	return &Handlers{guard: guard, q: q, payments: payments}
}

// BillingEnabled reports the Slice 7 flag (session navigation hint).
func (h *Handlers) BillingEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func ok() (oapi.Outcome, []oapi.ReasonCode) {
	return authzhttp.Outcome(result.Success), authzhttp.Reasons()
}

func uuidp(s *string) *openapi_types.UUID {
	if s == nil {
		return nil
	}
	u := authzhttp.UUID(*s)
	return &u
}

func strp[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

// UserRef maps a legacy user reference (nil = no reference).
func UserRef(key, user *string) *oapi.UserRef {
	if key == nil {
		return nil
	}
	return &oapi.UserRef{LegacyKey: *key, UserId: uuidp(user)}
}

// Money maps an exact amount in an explicit unit (nil when not representable).
func Money(amount *int64, unit *string) *oapi.Money {
	if amount == nil || unit == nil {
		return nil
	}
	return &oapi.Money{Amount: strconv.FormatInt(*amount, 10), Unit: oapi.Unit(*unit)}
}

// Amount maps raw text beside the money it represents.
func Amount(raw *string, amount *int64, unit *string, repr string) oapi.BillingAmount {
	a := oapi.BillingAmount{Raw: raw, Representation: oapi.AmountRepresentation(repr)}
	if repr == domain.Representable {
		a.Money = Money(amount, unit)
	}
	return a
}

// reprOf derives the representation of a value stored beside a sibling.
func reprOf(raw *string, amount *int64) string {
	switch {
	case amount != nil:
		return domain.Representable
	case raw == nil || *raw == "":
		return domain.Absent
	}
	return domain.Unrepresentable
}

// Source maps row provenance.
func Source(p store.Provenance) oapi.BillingSource {
	return oapi.BillingSource{SourceSystem: p.System, SourceSpec: p.Spec, SourceKey: p.Key, FirstBatchId: authzhttp.UUID(p.FirstBatch),
		LastBatchId: authzhttp.UUID(p.LastBatch), Presence: oapi.Presence(p.Presence), FirstSeenAt: p.FirstSeen, LastSeenAt: p.LastSeen}
}

func summary(v store.Invoice) oapi.InvoiceSummary {
	return oapi.InvoiceSummary{InvoiceId: authzhttp.UUID(v.InvoiceID), CodeRaw: v.CodeRaw, LifecycleState: oapi.InvoiceLifecycle(v.Lifecycle),
		Customer: oapi.BillingLink{Id: uuidp(v.CustomerID), State: oapi.ResolutionState(v.CustomerLink), Basis: v.CustomerBasis},
		Case:     oapi.BillingLink{Id: uuidp(v.CaseID), State: oapi.ResolutionState(v.CaseLink), Basis: v.CaseBasis},
		Seller:   UserRef(v.SellerKey, v.SellerUserID), IssuedBy: UserRef(v.IssuedByKey, v.IssuedByUserID),
		ItemsSourceComplete: v.ItemsComplete, StagesSourceComplete: v.StagesComplete, ItemCount: v.ItemCount, StageCount: v.StageCount,
		UnknownStateCount: v.UnknownStates, IssuedAtRaw: v.IssuedAtRaw, Source: Source(v.Source)}
}

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

func guardOp[R any](ctx context.Context, g *authorization.Guard, cp identityapi.Capability, gt gate[R]) (authorization.Check, *R, error) {
	c, err := g.Check(ctx, cp)
	if err != nil {
		return c, nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		res := gt.disabled(o, r)
		return c, &res, nil
	case authorization.GateDenied:
		res := gt.denied(authzhttp.Denied(ctx, c.Decision))
		return c, &res, nil
	}
	return c, nil, nil
}

func deref[R any](p *R) R {
	var zero R
	if p == nil {
		return zero
	}
	return *p
}

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

// ListInvoices lists shadow invoices.
func (h *Handlers) ListInvoices(ctx context.Context, req oapi.ListInvoicesRequestObject) (oapi.ListInvoicesResponseObject, error) {
	c, stop, err := guardOp(ctx, h.guard, identityapi.CapInvoiceRead, gate[oapi.ListInvoicesResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListInvoicesResponseObject {
			return oapi.ListInvoices200JSONResponse{Body: oapi.InvoiceListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListInvoicesResponseObject {
			return oapi.ListInvoices403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	res, err := h.q.ListInvoices(ctx, store.InvoiceFilter{Presence: strp(p.Presence), CustomerLink: strp(p.CustomerLinkState),
		CaseLink: strp(p.CaseLinkState), HasUnknownStatus: p.HasUnknownStatus}, p.Cursor, p.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListInvoices400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.InvoiceSummary, 0, len(res.Invoices))
	for _, v := range res.Invoices {
		items = append(items, summary(v))
	}
	o, r := ok()
	return oapi.ListInvoices200JSONResponse{Body: oapi.InvoiceListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Items: &items,
		Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetInvoice returns one invoice with its separate state and declared evidence.
func (h *Handlers) GetInvoice(ctx context.Context, req oapi.GetInvoiceRequestObject) (oapi.GetInvoiceResponseObject, error) {
	c, stop, err := guardOp(ctx, h.guard, identityapi.CapInvoiceRead, gate[oapi.GetInvoiceResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetInvoiceResponseObject {
			return oapi.GetInvoice200JSONResponse{Body: oapi.InvoiceEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetInvoiceResponseObject {
			return oapi.GetInvoice403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	v, err := h.q.GetInvoice(ctx, req.InvoiceId.String())
	if errors.Is(err, query.ErrNotFound) {
		return oapi.GetInvoice404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	d := oapi.InvoiceDetail{Invoice: summary(v), ProductRefRaw: v.ProductRefRaw, PayMethodRaw: v.PayMethodRaw, PaymentPlanRaw: v.PaymentPlanRaw,
		CurrentStageRaw: v.CurrentStageRaw, UpdatedAtRaw: v.UpdatedAtRaw, States: []oapi.InvoiceStateEvidence{}, Declared: []oapi.InvoiceDeclaredAmount{}}
	for _, s := range v.States {
		d.States = append(d.States, oapi.InvoiceStateEvidence{Field: oapi.InvoiceStateField(s.Field), Namespace: s.Namespace, Raw: s.Raw,
			Mapping: oapi.InvoiceVocabulary(s.Mapping)})
	}
	for _, a := range v.Declared {
		d.Declared = append(d.Declared, oapi.InvoiceDeclaredAmount{Kind: a.Kind, Amount: Amount(a.Raw, a.Amount, a.Unit, a.Representation)})
	}
	o, r := ok()
	return oapi.GetInvoice200JSONResponse{Body: oapi.InvoiceEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Data: &d,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListInvoiceItems pages an invoice's lines.
func (h *Handlers) ListInvoiceItems(ctx context.Context, req oapi.ListInvoiceItemsRequestObject) (oapi.ListInvoiceItemsResponseObject, error) {
	c, stop, err := guardOp(ctx, h.guard, identityapi.CapInvoiceRead, gate[oapi.ListInvoiceItemsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListInvoiceItemsResponseObject {
			return oapi.ListInvoiceItems200JSONResponse{Body: oapi.InvoiceLineListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListInvoiceItemsResponseObject {
			return oapi.ListInvoiceItems403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	res, err := h.q.Items(ctx, req.InvoiceId.String(), req.Params.Cursor, req.Params.Limit)
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.ListInvoiceItems400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.ListInvoiceItems404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := make([]oapi.InvoiceLine, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, oapi.InvoiceLine{ItemId: authzhttp.UUID(it.ItemID), ProductRefRaw: it.ProductRefRaw, TitleSnapshot: it.TitleSnapshot,
			KindRaw: it.KindRaw, QtyRaw: it.QtyRaw, UnitPrice: Amount(it.UnitPriceRaw, it.UnitPrice, it.Unit, reprOf(it.UnitPriceRaw, it.UnitPrice)),
			LineTotal: Amount(it.LineTotalRaw, it.LineTotal, it.Unit, it.LineRepr), IsFreeRaw: it.IsFreeRaw, CreatedAtRaw: it.CreatedAtRaw,
			Source: Source(it.Source)})
	}
	o, r := ok()
	return oapi.ListInvoiceItems200JSONResponse{Body: oapi.InvoiceLineListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListInvoiceStages pages an invoice's payment stages.
func (h *Handlers) ListInvoiceStages(ctx context.Context, req oapi.ListInvoiceStagesRequestObject) (oapi.ListInvoiceStagesResponseObject, error) {
	c, stop, err := guardOp(ctx, h.guard, identityapi.CapInvoiceRead, gate[oapi.ListInvoiceStagesResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListInvoiceStagesResponseObject {
			return oapi.ListInvoiceStages200JSONResponse{Body: oapi.PaymentStageListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListInvoiceStagesResponseObject {
			return oapi.ListInvoiceStages403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	res, err := h.q.Stages(ctx, req.InvoiceId.String(), req.Params.Cursor, req.Params.Limit)
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.ListInvoiceStages400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.ListInvoiceStages404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := make([]oapi.PaymentStage, 0, len(res.Stages))
	for _, s := range res.Stages {
		var st *oapi.PaymentStageState
		if s.StateTarget != nil {
			v := oapi.PaymentStageState(*s.StateTarget)
			st = &v
		}
		items = append(items, oapi.PaymentStage{StageId: authzhttp.UUID(s.StageID), StageNoRaw: s.StageNoRaw, StageTypeRaw: s.StageTypeRaw,
			Requested: Amount(s.RequestedRaw, s.Requested, s.Unit, s.RequestedRepr), StateNamespace: s.StateNamespace, StateRaw: s.StateRaw,
			StateMapping: oapi.TargetMapping(s.StateMapping), State: st, PayMethodRaw: s.PayMethodRaw, PaymentSourceRaw: s.PaymentSourceRaw,
			IssuedBy: UserRef(s.IssuedByKey, s.IssuedByUserID), ApprovedBy: UserRef(s.ApprovedByKey, s.ApprovedByUserID),
			ApprovedAtRaw: s.ApprovedAtRaw, PaidAtRaw: s.PaidAtRaw, CreatedAtRaw: s.CreatedAtRaw, Source: Source(s.Source)})
	}
	o, r := ok()
	return oapi.ListInvoiceStages200JSONResponse{Body: oapi.PaymentStageListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func sum(s domain.Sum) oapi.BillingSum {
	out := oapi.BillingSum{State: oapi.SumState(s.State), Rows: s.Rows}
	if s.Amount != nil && s.Unit != query.UnattributedUnit && s.State != domain.SumOverflow {
		u := s.Unit
		out.Money = Money(s.Amount, &u)
	}
	return out
}

// GetInvoiceTotals returns the per-unit total evidence. It needs both the
// invoice and the payment attempt capability (payment sums are payments
// evidence); either denial is a 403.
func (h *Handlers) GetInvoiceTotals(ctx context.Context, req oapi.GetInvoiceTotalsRequestObject) (oapi.GetInvoiceTotalsResponseObject, error) {
	g := gate[oapi.GetInvoiceTotalsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetInvoiceTotalsResponseObject {
			return oapi.GetInvoiceTotals200JSONResponse{Body: oapi.InvoiceTotalsEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetInvoiceTotalsResponseObject {
			return oapi.GetInvoiceTotals403JSONResponse{DeniedJSONResponse: d}
		},
	}
	c, stop, err := guardOp(ctx, h.guard, identityapi.CapInvoiceRead, g)
	if err != nil || stop != nil {
		return deref(stop), err
	}
	if _, stop, err := guardOp(ctx, h.guard, identityapi.CapPaymentAttemptRead, g); err != nil || stop != nil {
		return deref(stop), err
	}
	id := req.InvoiceId.String()
	sums, err := h.payments.AcceptedSums(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	p := sums[id]
	pay := map[string]query.PaymentSum{}
	for u, s := range p.Units {
		pay[u] = query.PaymentSum{Unit: u, Amount: s.Amount, State: s.State, Rows: s.Accepted}
	}
	ts, err := h.q.Totals(ctx, id, pay, p.Complete)
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.GetInvoiceTotals400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetInvoiceTotals404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	compared := make([]string, 0, len(query.ComparedDeclared))
	for k := range query.ComparedDeclared {
		compared = append(compared, k)
	}
	sort.Strings(compared)
	data := oapi.InvoiceTotals{InvoiceId: req.InvoiceId, ComparedDeclared: compared, PaymentEvidenceComplete: p.Complete,
		UnclassifiedAttempts: p.Unclassified, Units: []oapi.InvoiceUnitTotals{}}
	for _, t := range ts {
		u := oapi.InvoiceUnitTotals{Unit: oapi.InvoiceUnitTotalsUnit(t.Unit), ItemSum: sum(t.ItemSum), StageSum: sum(t.StageSum),
			PaymentSum: sum(t.PaymentSum), Class: oapi.TotalClass(t.Class)}
		kinds := make([]string, 0, len(t.Declared))
		for k := range t.Declared {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			unit := t.Unit
			if m := Money(t.Declared[k], &unit); m != nil {
				u.Declared = append(u.Declared, struct {
					Kind  string     `json:"kind"`
					Money oapi.Money `json:"money"`
				}{Kind: k, Money: *m})
			}
		}
		if u.Declared == nil {
			u.Declared = []struct {
				Kind  string     `json:"kind"`
				Money oapi.Money `json:"money"`
			}{}
		}
		data.Units = append(data.Units, u)
	}
	o, r := ok()
	return oapi.GetInvoiceTotals200JSONResponse{Body: oapi.InvoiceTotalsEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// Authoritative definition of F02 (quoted; not invented).
const (
	authorityF02 = "CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md F02; GATE-0-PRODUCT-INVARIANTS.md metric table (Open Pre-invoices); CRM-IMPLEMENTATION-FOUNDATION-PLAN.md F02"
	defF02       = "Open Pre-invoices: distinct invoices whose current normalized state is pre_invoice without completion/cancel, at a current snapshot; the legacy predicate omitted pre_invoice (F02)."
)

// F02 is the honest Open Pre-invoices metric of Slice 7.
func F02() oapi.HonestMetric {
	return oapi.HonestMetric{Code: "F02", Label: "Open Pre-invoices", Status: "NOT_EVALUATED", Count: nil, Definition: defF02, Authority: authorityF02,
		Coverage: "PARTIAL: raw invoice status fields are staged, but no canonical current invoice state exists",
		Reason:   "F02 needs the normalized current invoice state; its resolver is deferred (TAD #20), so no count is claimed (null is not zero)"}
}

// GetBillingReconciliationReport returns the billing and payments coverage.
func (h *Handlers) GetBillingReconciliationReport(ctx context.Context, _ oapi.GetBillingReconciliationReportRequestObject) (oapi.GetBillingReconciliationReportResponseObject, error) {
	c, stop, err := guardOp(ctx, h.guard, identityapi.CapBillingReportRead, gate[oapi.GetBillingReconciliationReportResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetBillingReconciliationReportResponseObject {
			return oapi.GetBillingReconciliationReport200JSONResponse{Body: oapi.BillingReconciliationReportEnvelope{Outcome: o, Reasons: r,
				CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetBillingReconciliationReportResponseObject {
			return oapi.GetBillingReconciliationReport403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	b, err := h.q.Coverage(ctx)
	if err != nil {
		return nil, err
	}
	p, err := h.payments.Coverage(ctx)
	if err != nil {
		return nil, err
	}
	nested := func(m map[string]map[string]int64) map[string]oapi.CountMap {
		out := map[string]oapi.CountMap{}
		for k, v := range m {
			out[k] = oapi.CountMap(v)
		}
		return out
	}
	data := oapi.BillingReconciliationReport{
		Billing: oapi.BillingCoverage{InvoicesPresent: b.Invoices, ItemsPresent: b.Items, StagesPresent: b.Stages, InvoicesWithoutItems: b.InvoicesNoItems,
			InvoicesWithoutStages: b.InvoicesNoStages, InvoiceStateVocabularyByField: nested(b.StateMappings), UnknownRawStates: nested(b.UnknownRawStates),
			StageStateMapping: oapi.CountMap(b.StageStates), UnknownStageRawStates: oapi.CountMap(b.UnknownStageRaw),
			CustomerLinkStates: oapi.CountMap(b.CustomerLinks), CaseLinkStates: oapi.CountMap(b.CaseLinks), UnrepresentableAmounts: b.Unrepresentable,
			InvoicesWithRawPreInvoiceValue: b.PreInvoiceRawSeen},
		Payments: oapi.PaymentsCoverage{AttemptsPresent: p.Attempts, AttemptsByChannel: oapi.CountMap(p.ByChannel),
			AttemptStateMapping: oapi.CountMap(p.ByStateMapping), AttemptStates: oapi.CountMap(p.ByTarget), UnknownRawAttemptStates: oapi.CountMap(p.UnknownRaw),
			ApprovedRawCount: p.ApprovedRaw, ReconcileErrorRawCount: p.ReconcileErrorRaw, ApprovalProvenance: oapi.CountMap(p.ByProvenance),
			GatewayAttempts: p.GatewayAttempts, GatewayStates: oapi.CountMap(p.GatewayStates), AttemptsWithoutInvoice: p.WithoutInvoice,
			UnrepresentableAmounts: p.Unrepresentable, AttemptStateRegressions: p.StateRegressions, ReceiptsReferenced: p.ReceiptsReferenced,
			ReceiptsUnknown: p.ReceiptsUnknown, ReceiptsMissing: p.ReceiptsMissing},
		F02: F02(),
		Notes: []string{
			"invoice lifecycle is UNKNOWN for every invoice: no canonical invoice status is claimed (TAD #20)",
			"each legacy invoice status field, payment stage, attempt, gateway and receipt state is reported separately",
			"amounts are exact int64 with an explicit unit; IRR and IRT are never combined or converted",
			"payment sums count only attempts explicitly mapped ACCEPTED; legacy 'approved' is not Finance-reviewed, settled or posted (D-25 open)",
			"receipts are reference metadata only; no file is read, so none is AVAILABLE",
			"per-batch provenance reconciliation runs offline (crm-legacy-import reconcile-billing)",
		}}
	o, r := ok()
	return oapi.GetBillingReconciliationReport200JSONResponse{Body: oapi.BillingReconciliationReportEnvelope{Outcome: o, Reasons: r,
		CorrelationId: httpx.CorrelationID(ctx), Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
