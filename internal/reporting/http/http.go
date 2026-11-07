// Package http adapts the reporting read service to the generated OpenAPI
// strict server. Handlers guard (flag reporting.read_shadow + evaluator),
// build the ONE shared query spec from allowlisted parameters, call the read
// service and map results. No SQL, no projection rebuild, no write (lint).
//
// Invoice Review is a QUERY SURFACE ONLY (D-24: no target UI; D-23: no
// export of any kind). Every Invoice Review read is audited FIRST through the
// audit module in its own transaction — never inside the read-only query
// transaction — and when the audit cannot be written the read fails closed
// (D-45 safe default): no row leaves the server.
package http

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/audit"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/query"
)

// ReadAuditor records one sensitive read in the audit module's own
// transaction. A nil auditor, or any error, fails the read closed.
type ReadAuditor func(ctx context.Context, e audit.Event) error

// Handlers serves the reporting read operations.
type Handlers struct {
	guard   *authorization.Guard
	q       *query.Service
	auditor ReadAuditor
	clock   timeutil.Clock
}

// New returns the handlers. auditor may be nil: Invoice Review then fails
// closed; aggregate reporting and MIS are unaffected.
func New(guard *authorization.Guard, q *query.Service, auditor ReadAuditor, clock timeutil.Clock) *Handlers {
	return &Handlers{guard: guard, q: q, auditor: auditor, clock: clock}
}

// ReportingEnabled reports the Slice 8 flag (session navigation hint).
func (h *Handlers) ReportingEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func corr(ctx context.Context) string { return httpx.CorrelationID(ctx) }

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

func money(amount *int64, unit string) *oapi.Money {
	if amount == nil || (unit != "IRT" && unit != "IRR") {
		return nil
	}
	return &oapi.Money{Amount: strconv.FormatInt(*amount, 10), Unit: oapi.Unit(unit)}
}

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

// guardOp checks flag + capability and the evaluator scope. Only
// ORGANIZATION has frozen authority; anything narrower is denied rather than
// broadened (no row-scope resolver exists for reporting yet).
func guardOp[R any](ctx context.Context, h *Handlers, cp identityapi.Capability, g gate[R]) (authorization.Check, *R, error) {
	c, err := h.guard.Check(ctx, cp)
	if err != nil {
		return c, nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		res := g.disabled(o, r)
		return c, &res, nil
	case authorization.GateDenied:
		res := g.denied(authzhttp.Denied(ctx, c.Decision))
		return c, &res, nil
	}
	if c.Decision.Scope != authorization.ScopeOrganization {
		d := c.Decision
		code := reasoncodes.RowScopeDenied
		d.Allowed, d.Outcome, d.Reason = false, result.Unauthorized, &code
		res := g.denied(authzhttp.Denied(ctx, d))
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

// specOf builds the shared query spec from typed parameters (nil = unset).
func specOf(report string, scope *string, tb *oapi.ReportTimeBasis, from, to, asOf *time.Time, runID *openapi_types.UUID,
	filters map[string]*string) query.Spec {
	s := query.Spec{Report: report, Filters: map[string]string{}, From: from, To: to, AsOf: asOf}
	if scope != nil {
		s.Scope = *scope
	}
	if tb != nil {
		s.TimeBasis = domain.TimeBasis(*tb)
	}
	if runID != nil {
		v := runID.String()
		s.RunID = &v
	}
	for k, v := range filters {
		if v != nil {
			s.Filters[k] = *v
		}
	}
	return s
}

func boolp(b *bool) *string {
	if b == nil {
		return nil
	}
	v := strconv.FormatBool(*b)
	return &v
}

func runOf(r *query.Run) *oapi.ProjectionRun {
	if r == nil {
		return nil
	}
	out := oapi.ProjectionRun{RunId: authzhttp.UUID(r.RunID), ProjectionVersion: r.ProjectionVersion, SourceCutoff: r.SourceCutoff,
		GeneratedAt: r.CompletedAt, Outcome: oapi.ProjectionRunOutcome(r.Outcome), Sources: []oapi.SourceContribution{}}
	names := make([]string, 0, len(r.Sources))
	for n := range r.Sources {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		s := r.Sources[n]
		out.Sources = append(out.Sources, oapi.SourceContribution{Source: n, Status: oapi.CoverageState(s.Status), RowsRead: s.RowsRead,
			ReadAt: s.ReadAt, Reason: s.Reason})
	}
	return &out
}

func metaOf(res query.Result) oapi.ReportMeta {
	q := oapi.ReportQuery{Report: oapi.ReportQueryReport(res.Spec.Report), Scope: oapi.ReportQueryScope(res.Spec.Scope),
		TimeBasis: oapi.ReportTimeBasis(res.Spec.TimeBasis), From: res.Spec.From, To: res.Spec.To, AsOf: res.Spec.AsOf,
		RunId: uuidp(res.Spec.RunID), Filters: res.Spec.Filters}
	if q.Filters == nil {
		q.Filters = map[string]string{}
	}
	m := oapi.ReportMeta{Query: q, Available: res.Available, Run: runOf(res.Run), Freshness: oapi.Freshness(res.Freshness),
		Coverage: oapi.CoverageState(res.Coverage), SourceCoverage: map[string]oapi.CoverageState{}}
	for k, v := range res.SourceCoverage {
		m.SourceCoverage[k] = oapi.CoverageState(v)
	}
	return m
}

func summaryOf(s *query.Summary) *oapi.ReportSummary {
	if s == nil {
		return nil
	}
	out := oapi.ReportSummary{Total: s.Total, Counts: map[string]oapi.CountMap{}}
	for k, v := range s.Counts {
		out.Counts[k] = oapi.CountMap(v)
	}
	if s.Amounts != nil {
		a := []oapi.UnitAmount{}
		for _, x := range s.Amounts {
			a = append(a, oapi.UnitAmount{Unit: oapi.UnitAmountUnit(x.Unit), Money: money(x.Sum, x.Unit), Rows: x.Rows, Overflow: x.Overflow})
		}
		out.Amounts = &a
		u, ab := s.Unrepresentable, s.Absent
		out.UnrepresentableAmounts, out.AbsentAmounts = &u, &ab
	}
	if s.UnitClasses != nil {
		uc := map[string]oapi.CountMap{}
		for k, v := range s.UnitClasses {
			uc[k] = oapi.CountMap(v)
		}
		out.UnitClasses = &uc
	}
	return &out
}

// outcomeOf: UNKNOWN when nothing answers, PARTIAL when coverage is not
// complete, SUCCESS otherwise; STALE is a reason, never hidden.
func outcomeOf(res query.Result) (oapi.Outcome, []oapi.ReasonCode) {
	var reasons []reasoncodes.Code
	st := result.Success
	switch {
	case !res.Available:
		st, reasons = result.Unknown, append(reasons, reasoncodes.SourceUnavailable)
	case res.Coverage != domain.Available:
		st, reasons = result.Partial, append(reasons, reasoncodes.CoverageIncomplete)
	}
	if res.Freshness == domain.Stale {
		reasons = append(reasons, reasoncodes.StaleState)
	}
	return authzhttp.Outcome(st), authzhttp.Reasons(reasons...)
}

func caseRow(x query.CaseRow) oapi.CaseFactRow {
	return oapi.CaseFactRow{CaseId: authzhttp.UUID(x.CaseID), WorkflowType: x.Workflow, StateNamespace: x.StateNamespace, StateRaw: x.StateRaw,
		StateMapping: x.StateMapping, CustomerLinkState: oapi.ResolutionState(x.CustomerLinkState), Presence: oapi.Presence(x.Presence),
		SourceSpec: x.SourceSpec, SourceKey: x.SourceKey, AliasCount: x.AliasCount, FirstSeenAt: x.FirstSeenAt,
		OriginalOwnerResolution: x.OriginalOwnerResolution, OriginalOwnerKey: x.OriginalOwnerKey, OriginalSellerResolution: x.OriginalSellerResolution,
		OriginalSellerKey: x.OriginalSellerKey, CreditOwnerResolution: x.CreditOwnerResolution}
}

func custodyRow(x query.CustodyRow) oapi.CustodyFactRow {
	return oapi.CustodyFactRow{CaseId: authzhttp.UUID(x.CaseID), Derivation: x.Derivation, Parity: x.Parity, DerivedHolderKey: x.DerivedKey,
		LegacyHolderKey: x.LegacyKey, EventCount: x.EventCount, FindingCount: x.FindingCount, Presence: oapi.Presence(x.Presence)}
}

func cell(c query.StateCell) oapi.StateCell { return oapi.StateCell{Raw: c.Raw, Mapping: c.Mapping} }

func invoiceRow(x query.InvoiceRow) oapi.InvoiceFactRow {
	out := oapi.InvoiceFactRow{InvoiceId: authzhttp.UUID(x.InvoiceID), SourceSpec: x.SourceSpec, SourceKey: x.SourceKey, Presence: oapi.Presence(x.Presence),
		CustomerLinkState: oapi.ResolutionState(x.CustomerLinkState), CaseLinkState: oapi.ResolutionState(x.CaseLinkState), Main: cell(x.Main),
		Invoice: cell(x.Invoice), Payment: cell(x.Payment), Workflow: cell(x.Workflow), Return: cell(x.Return), UnknownStateCount: x.UnknownStates,
		ItemCount: x.ItemCount, StageCount: x.StageCount, ItemsComplete: x.ItemsComplete, StagesComplete: x.StagesComplete, FirstSeenAt: x.FirstSeenAt,
		Units: []oapi.InvoiceUnitRow{}}
	for _, u := range x.Units {
		out.Units = append(out.Units, oapi.InvoiceUnitRow{Unit: oapi.InvoiceUnitRowUnit(u.Unit), DeclaredFinal: money(u.DeclaredFinal, u.Unit),
			DeclaredPaymentTotal: money(u.DeclaredPayment, u.Unit), ItemSum: money(u.ItemSum, u.Unit), ItemState: u.ItemState,
			StageSum: money(u.StageSum, u.Unit), StageState: u.StageState, PaymentSum: money(u.PaymentSum, u.Unit), PaymentState: u.PaymentState,
			Class: u.Class})
	}
	return out
}

func paymentRow(x query.PaymentRow) oapi.PaymentFactRow {
	out := oapi.PaymentFactRow{AttemptId: authzhttp.UUID(x.AttemptID), InvoiceId: uuidp(x.InvoiceID),
		InvoiceLinkState: oapi.PaymentFactRowInvoiceLinkState(x.InvoiceLinkState), Channel: x.Channel, StateRaw: x.StateRaw, StateMapping: x.StateMapping,
		State: x.StateTarget, ApprovalProvenance: x.Provenance, AmountRaw: x.AmountRaw, AmountRepresentation: x.AmountRepr, GatewayState: x.GatewayState,
		Presence: oapi.Presence(x.Presence), FirstSeenAt: x.FirstSeenAt}
	if x.Unit != nil && x.AmountRepr == "REPRESENTABLE" {
		out.Money = money(x.Amount, *x.Unit)
	}
	return out
}

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

func isInvalid(err error) bool { return errors.Is(err, query.ErrInvalidQuery) }

// ---- reports ---------------------------------------------------------------

// GetCaseReport answers the cases report (summary + page, same spec).
func (h *Handlers) GetCaseReport(ctx context.Context, req oapi.GetCaseReportRequestObject) (oapi.GetCaseReportResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetCaseReportResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetCaseReportResponseObject {
			return oapi.GetCaseReport200JSONResponse{Body: oapi.CaseReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetCaseReportResponseObject {
			return oapi.GetCaseReport403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportCases, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, map[string]*string{
		"presence": strp(p.Presence), "workflow_type": strp(p.WorkflowType), "state_mapping": strp(p.StateMapping),
		"customer_link_state": strp(p.CustomerLinkState), "original_owner_resolution": strp(p.OriginalOwnerResolution)})
	res, pg, err := h.q.Report(ctx, spec, p.Cursor, p.Limit)
	if isInvalid(err) {
		return oapi.GetCaseReport400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.CaseFactRow{}
	for _, r := range pg.Rows {
		items = append(items, caseRow(r.(query.CaseRow)))
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	return oapi.GetCaseReport200JSONResponse{Body: oapi.CaseReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m,
		Summary: summaryOf(res.Summary), Items: &items, Page: page(pg.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetCaseReportRow answers one case row and its membership in the spec.
func (h *Handlers) GetCaseReportRow(ctx context.Context, req oapi.GetCaseReportRowRequestObject) (oapi.GetCaseReportRowResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetCaseReportRowResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetCaseReportRowResponseObject {
			return oapi.GetCaseReportRow200JSONResponse{Body: oapi.CaseDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetCaseReportRowResponseObject {
			return oapi.GetCaseReportRow403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportCases, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, map[string]*string{
		"presence": strp(p.Presence), "workflow_type": strp(p.WorkflowType), "state_mapping": strp(p.StateMapping),
		"customer_link_state": strp(p.CustomerLinkState), "original_owner_resolution": strp(p.OriginalOwnerResolution)})
	res, row, member, err := h.q.Detail(ctx, spec, req.CaseId.String())
	switch {
	case isInvalid(err):
		return oapi.GetCaseReportRow400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetCaseReportRow404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	env := oapi.CaseDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m, AllowedActions: authzhttp.AllowedActions(c.Decisions)}
	if row != nil {
		d := caseRow(row.(query.CaseRow))
		env.Data, env.Member = &d, &member
	}
	return oapi.GetCaseReportRow200JSONResponse{Body: env}, nil
}

// GetCustodyReport answers the custody report.
func (h *Handlers) GetCustodyReport(ctx context.Context, req oapi.GetCustodyReportRequestObject) (oapi.GetCustodyReportResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetCustodyReportResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetCustodyReportResponseObject {
			return oapi.GetCustodyReport200JSONResponse{Body: oapi.CustodyReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetCustodyReportResponseObject {
			return oapi.GetCustodyReport403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportCustody, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, map[string]*string{
		"presence": strp(p.Presence), "parity": strp(p.Parity), "derivation": strp(p.Derivation)})
	res, pg, err := h.q.Report(ctx, spec, p.Cursor, p.Limit)
	if isInvalid(err) {
		return oapi.GetCustodyReport400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.CustodyFactRow{}
	for _, r := range pg.Rows {
		items = append(items, custodyRow(r.(query.CustodyRow)))
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	return oapi.GetCustodyReport200JSONResponse{Body: oapi.CustodyReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m,
		Summary: summaryOf(res.Summary), Items: &items, Page: page(pg.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetCustodyReportRow answers one custody row and its membership.
func (h *Handlers) GetCustodyReportRow(ctx context.Context, req oapi.GetCustodyReportRowRequestObject) (oapi.GetCustodyReportRowResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetCustodyReportRowResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetCustodyReportRowResponseObject {
			return oapi.GetCustodyReportRow200JSONResponse{Body: oapi.CustodyDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetCustodyReportRowResponseObject {
			return oapi.GetCustodyReportRow403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportCustody, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, map[string]*string{
		"presence": strp(p.Presence), "parity": strp(p.Parity), "derivation": strp(p.Derivation)})
	res, row, member, err := h.q.Detail(ctx, spec, req.CaseId.String())
	switch {
	case isInvalid(err):
		return oapi.GetCustodyReportRow400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetCustodyReportRow404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	env := oapi.CustodyDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m, AllowedActions: authzhttp.AllowedActions(c.Decisions)}
	if row != nil {
		d := custodyRow(row.(query.CustodyRow))
		env.Data, env.Member = &d, &member
	}
	return oapi.GetCustodyReportRow200JSONResponse{Body: env}, nil
}

func invoiceFilters(presence, cust, cs *string, unknown *bool, mainMapping, class, unit *string) map[string]*string {
	return map[string]*string{"presence": presence, "customer_link_state": cust, "case_link_state": cs, "has_unknown_status": boolp(unknown),
		"main_mapping": mainMapping, "total_class": class, "unit": unit}
}

// GetInvoiceReport answers the invoices report.
func (h *Handlers) GetInvoiceReport(ctx context.Context, req oapi.GetInvoiceReportRequestObject) (oapi.GetInvoiceReportResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetInvoiceReportResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetInvoiceReportResponseObject {
			return oapi.GetInvoiceReport200JSONResponse{Body: oapi.InvoiceReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetInvoiceReportResponseObject {
			return oapi.GetInvoiceReport403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportInvoices, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, invoiceFilters(strp(p.Presence),
		strp(p.CustomerLinkState), strp(p.CaseLinkState), p.HasUnknownStatus, strp(p.MainMapping), strp(p.TotalClass), strp(p.Unit)))
	res, pg, err := h.q.Report(ctx, spec, p.Cursor, p.Limit)
	if isInvalid(err) {
		return oapi.GetInvoiceReport400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.InvoiceFactRow{}
	for _, r := range pg.Rows {
		items = append(items, invoiceRow(r.(query.InvoiceRow)))
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	return oapi.GetInvoiceReport200JSONResponse{Body: oapi.InvoiceReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m,
		Summary: summaryOf(res.Summary), Items: &items, Page: page(pg.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetInvoiceReportRow answers one invoice row and its membership.
func (h *Handlers) GetInvoiceReportRow(ctx context.Context, req oapi.GetInvoiceReportRowRequestObject) (oapi.GetInvoiceReportRowResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetInvoiceReportRowResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetInvoiceReportRowResponseObject {
			return oapi.GetInvoiceReportRow200JSONResponse{Body: oapi.InvoiceDetailFactEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetInvoiceReportRowResponseObject {
			return oapi.GetInvoiceReportRow403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportInvoices, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, invoiceFilters(strp(p.Presence),
		strp(p.CustomerLinkState), strp(p.CaseLinkState), p.HasUnknownStatus, strp(p.MainMapping), strp(p.TotalClass), strp(p.Unit)))
	res, row, member, err := h.q.Detail(ctx, spec, req.InvoiceId.String())
	switch {
	case isInvalid(err):
		return oapi.GetInvoiceReportRow400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetInvoiceReportRow404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	env := oapi.InvoiceDetailFactEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m, AllowedActions: authzhttp.AllowedActions(c.Decisions)}
	if row != nil {
		d := invoiceRow(row.(query.InvoiceRow))
		env.Data, env.Member = &d, &member
	}
	return oapi.GetInvoiceReportRow200JSONResponse{Body: env}, nil
}

func paymentFilters(presence, channel, state, mapping, prov, link, unit *string) map[string]*string {
	return map[string]*string{"presence": presence, "channel": channel, "state": state, "state_mapping": mapping, "approval_provenance": prov,
		"invoice_link_state": link, "unit": unit}
}

// GetPaymentReport answers the payments report.
func (h *Handlers) GetPaymentReport(ctx context.Context, req oapi.GetPaymentReportRequestObject) (oapi.GetPaymentReportResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetPaymentReportResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetPaymentReportResponseObject {
			return oapi.GetPaymentReport200JSONResponse{Body: oapi.PaymentReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetPaymentReportResponseObject {
			return oapi.GetPaymentReport403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportPayments, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, paymentFilters(strp(p.Presence), strp(p.Channel),
		strp(p.State), strp(p.StateMapping), strp(p.ApprovalProvenance), strp(p.InvoiceLinkState), strp(p.Unit)))
	res, pg, err := h.q.Report(ctx, spec, p.Cursor, p.Limit)
	if isInvalid(err) {
		return oapi.GetPaymentReport400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.PaymentFactRow{}
	for _, r := range pg.Rows {
		items = append(items, paymentRow(r.(query.PaymentRow)))
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	return oapi.GetPaymentReport200JSONResponse{Body: oapi.PaymentReportEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m,
		Summary: summaryOf(res.Summary), Items: &items, Page: page(pg.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetPaymentReportRow answers one payment row and its membership.
func (h *Handlers) GetPaymentReportRow(ctx context.Context, req oapi.GetPaymentReportRowRequestObject) (oapi.GetPaymentReportRowResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetPaymentReportRowResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetPaymentReportRowResponseObject {
			return oapi.GetPaymentReportRow200JSONResponse{Body: oapi.PaymentDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetPaymentReportRowResponseObject {
			return oapi.GetPaymentReportRow403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportPayments, strp(p.Scope), p.TimeBasis, p.From, p.To, p.AsOf, p.RunId, paymentFilters(strp(p.Presence), strp(p.Channel),
		strp(p.State), strp(p.StateMapping), strp(p.ApprovalProvenance), strp(p.InvoiceLinkState), strp(p.Unit)))
	res, row, member, err := h.q.Detail(ctx, spec, req.AttemptId.String())
	switch {
	case isInvalid(err):
		return oapi.GetPaymentReportRow400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetPaymentReportRow404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	env := oapi.PaymentDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m, AllowedActions: authzhttp.AllowedActions(c.Decisions)}
	if row != nil {
		d := paymentRow(row.(query.PaymentRow))
		env.Data, env.Member = &d, &member
	}
	return oapi.GetPaymentReportRow200JSONResponse{Body: env}, nil
}

// ---- metrics -----------------------------------------------------------------

func contractOf(c domain.Contract) oapi.MetricContract {
	status, reason := c.Status()
	out := oapi.MetricContract{MetricId: c.ID, Version: c.Version, Name: c.Name, Grain: c.Grain, Source: c.Source, Cohort: c.Cohort,
		TimeBases: []oapi.ReportTimeBasis{}, Scope: c.Scope, Formula: c.Formula, Unit: c.Unit, FreshnessSlaSeconds: int64(c.FreshnessSLA / time.Second),
		Coverage: c.Coverage, Status: oapi.MetricContractStatus(status)}
	for _, b := range c.TimeBases {
		out.TimeBases = append(out.TimeBases, oapi.ReportTimeBasis(b))
	}
	if reason != "" {
		out.StatusReason = &reason
	}
	if c.Finding != "" {
		f := c.Finding
		out.Finding = &f
	}
	return out
}

func metricOf(mv query.MetricValue) oapi.MetricValue {
	out := oapi.MetricValue{Contract: contractOf(mv.Contract), ValueState: oapi.MetricValueValueState(mv.ValueState), Meta: metaOf(mv.Result),
		Summary: summaryOf(mv.Result.Summary)}
	if mv.Result.Spec.Report == "" {
		// NOT_FINAL metrics carry no query at all: nothing was evaluated.
		out.Meta.Query = oapi.ReportQuery{Report: oapi.ReportQueryReport(mv.Contract.Report), Scope: oapi.ReportQueryScope(query.ScopeOrganization),
			TimeBasis: oapi.ReportTimeBasis(domain.CurrentSnapshot), Filters: map[string]string{}}
		out.Meta.Freshness, out.Meta.Coverage = oapi.Freshness(domain.Unknown), oapi.CoverageState(domain.Unknown)
		out.Meta.SourceCoverage = map[string]oapi.CoverageState{}
	}
	if mv.Reason != "" {
		r := mv.Reason
		out.Reason = &r
	}
	return out
}

// evaluate is the list/overview evaluation: event-range-only metrics need a
// range and are reported UNKNOWN with the reason instead of a value.
func (h *Handlers) evaluate(ctx context.Context, c domain.Contract) (oapi.MetricValue, error) {
	if st, _ := c.Status(); st == "FINAL" && !c.Allows(domain.CurrentSnapshot) {
		mv := query.MetricValue{Contract: c, Status: "FINAL", ValueState: "UNKNOWN",
			Reason: "event-range metric: evaluate with time_basis=EVENT_RANGE and from/to"}
		return metricOf(mv), nil
	}
	mv, err := h.q.Metric(ctx, c.ID, "", nil, nil, nil, nil)
	if err != nil {
		return oapi.MetricValue{}, err
	}
	return metricOf(mv), nil
}

// ListReportingMetrics lists every contract with its current value.
func (h *Handlers) ListReportingMetrics(ctx context.Context, _ oapi.ListReportingMetricsRequestObject) (oapi.ListReportingMetricsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.ListReportingMetricsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListReportingMetricsResponseObject {
			return oapi.ListReportingMetrics200JSONResponse{Body: oapi.MetricListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListReportingMetricsResponseObject {
			return oapi.ListReportingMetrics403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	items := []oapi.MetricValue{}
	for _, ct := range domain.Registry() {
		mv, err := h.evaluate(ctx, ct)
		if err != nil {
			return nil, err
		}
		items = append(items, mv)
	}
	return oapi.ListReportingMetrics200JSONResponse{Body: oapi.MetricListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: corr(ctx), Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetReportingMetric evaluates one metric in an explicit time basis.
func (h *Handlers) GetReportingMetric(ctx context.Context, req oapi.GetReportingMetricRequestObject) (oapi.GetReportingMetricResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMetricsRead, gate[oapi.GetReportingMetricResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetReportingMetricResponseObject {
			return oapi.GetReportingMetric200JSONResponse{Body: oapi.MetricEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetReportingMetricResponseObject {
			return oapi.GetReportingMetric403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	var tb domain.TimeBasis
	if p.TimeBasis != nil {
		tb = domain.TimeBasis(*p.TimeBasis)
	}
	var run *string
	if p.RunId != nil {
		v := p.RunId.String()
		run = &v
	}
	mv, err := h.q.Metric(ctx, req.MetricId, tb, p.From, p.To, p.AsOf, run)
	switch {
	case isInvalid(err):
		return oapi.GetReportingMetric400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetReportingMetric404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	d := metricOf(mv)
	o, r := outcomeOf(mv.Result)
	if mv.ValueState == "NOT_EVALUATED" {
		o, r = authzhttp.Outcome(result.Blocked), authzhttp.Reasons(reasoncodes.PolicyUnresolved)
	}
	return oapi.GetReportingMetric200JSONResponse{Body: oapi.MetricEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Data: &d,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ---- MIS -------------------------------------------------------------------

// GetMisOverview is the read-only Today view: proven contracts only.
func (h *Handlers) GetMisOverview(ctx context.Context, _ oapi.GetMisOverviewRequestObject) (oapi.GetMisOverviewResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMISRead, gate[oapi.GetMisOverviewResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetMisOverviewResponseObject {
			return oapi.GetMisOverview200JSONResponse{Body: oapi.MisOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetMisOverviewResponseObject {
			return oapi.GetMisOverview403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, err := h.q.LatestRun(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.MisOverview{Run: runOf(run), Metrics: []oapi.MetricValue{}, Deferred: []oapi.MetricContract{}}
	data.Freshness, data.Coverage = runHealth(run, h.clock)
	for _, ct := range domain.Registry() {
		if st, _ := ct.Status(); st != "FINAL" {
			data.Deferred = append(data.Deferred, contractOf(ct))
			continue
		}
		if !ct.Allows(domain.CurrentSnapshot) {
			continue
		}
		mv, err := h.evaluate(ctx, ct)
		if err != nil {
			return nil, err
		}
		data.Metrics = append(data.Metrics, mv)
	}
	o, r := healthOutcome(run, data.Freshness, data.Coverage)
	return oapi.GetMisOverview200JSONResponse{Body: oapi.MisOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Data: &data,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func runHealth(run *query.Run, clock timeutil.Clock) (oapi.Freshness, oapi.CoverageState) {
	if run == nil {
		return oapi.Freshness(domain.Unknown), oapi.CoverageState(domain.Unknown)
	}
	var states []string
	for _, s := range run.Sources {
		states = append(states, s.Status)
	}
	return oapi.Freshness(domain.FreshnessOf(&run.CompletedAt, clock.Now().Time(), query.ReportFreshnessSLA)),
		oapi.CoverageState(domain.WorstCoverage(states...))
}

func healthOutcome(run *query.Run, f oapi.Freshness, cov oapi.CoverageState) (oapi.Outcome, []oapi.ReasonCode) {
	var reasons []reasoncodes.Code
	st := result.Success
	switch {
	case run == nil:
		st, reasons = result.Unknown, append(reasons, reasoncodes.SourceUnavailable)
	case cov != oapi.CoverageState(domain.Available):
		st, reasons = result.Partial, append(reasons, reasoncodes.CoverageIncomplete)
	}
	if f == oapi.Freshness(domain.Stale) {
		reasons = append(reasons, reasoncodes.StaleState)
	}
	return authzhttp.Outcome(st), authzhttp.Reasons(reasons...)
}

func i64(v int64) *int64 { return &v }

// GetMisSourceQuality reports valid / duplicate candidate / invalid / unknown
// / conflict per published source at an explicit grain. A class that does
// not apply is null (not zero); a missing source has no counts at all.
func (h *Handlers) GetMisSourceQuality(ctx context.Context, _ oapi.GetMisSourceQualityRequestObject) (oapi.GetMisSourceQualityResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMISRead, gate[oapi.GetMisSourceQualityResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetMisSourceQualityResponseObject {
			return oapi.GetMisSourceQuality200JSONResponse{Body: oapi.SourceQualityEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetMisSourceQualityResponseObject {
			return oapi.GetMisSourceQuality403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, err := h.q.LatestRun(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.SourceQuality{Run: runOf(run), Rows: []oapi.SourceQualityRow{}}
	data.Freshness, _ = runHealth(run, h.clock)
	if run == nil {
		o, r := healthOutcome(nil, data.Freshness, oapi.CoverageState(domain.Unknown))
		return oapi.GetMisSourceQuality200JSONResponse{Body: oapi.SourceQualityEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Data: &data,
			AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
	}
	pinned := run.RunID
	summary := func(report string, filters map[string]string) (*query.Summary, error) {
		one := 1
		res, _, err := h.q.Report(ctx, query.Spec{Report: report, RunID: &pinned, Filters: filters}, nil, &one)
		return res.Summary, err
	}
	status := func(src string) string { return run.Sources[src].Status }
	present := map[string]string{"presence": "PRESENT"}
	add := func(src, grain, note string, fill func(*oapi.SourceQualityRow) error) error {
		row := oapi.SourceQualityRow{Source: src, Grain: grain, Coverage: oapi.CoverageState(status(src)), Note: note}
		if s := status(src); s == domain.Available || s == domain.Partial {
			if err := fill(&row); err != nil {
				return err
			}
		}
		data.Rows = append(data.Rows, row)
		return nil
	}
	steps := []func() error{
		func() error {
			return add("cases", "Case (one per staged source row)", "valid = mapped source state; unknown = unmapped raw state; conflict = customer link CONFLICT. Aliases are not duplicates (OPD-01).",
				func(r *oapi.SourceQualityRow) error {
					s, err := summary(domain.ReportCases, present)
					if err != nil {
						return err
					}
					r.Valid, r.Unknown, r.Conflict = i64(s.Counts["state_mapping"]["KNOWN"]), i64(s.Counts["state_mapping"]["UNKNOWN"]),
						i64(s.Counts["customer_link_state"]["CONFLICT"])
					return nil
				})
		},
		func() error {
			return add("custody", "Case custody projection", "valid = MATCH; conflict = CONFLICT or MULTIPLE_CURRENT; unknown = TARGET_UNKNOWN or LEGACY_UNKNOWN; NOT_RECOVERABLE and HISTORY_GAP are reported in the custody report.",
				func(r *oapi.SourceQualityRow) error {
					s, err := summary(domain.ReportCustody, present)
					if err != nil {
						return err
					}
					p := s.Counts["parity"]
					r.Valid, r.Conflict, r.Unknown = i64(p["MATCH"]), i64(p["CONFLICT"]+p["MULTIPLE_CURRENT"]), i64(p["TARGET_UNKNOWN"]+p["LEGACY_UNKNOWN"])
					return nil
				})
		},
		func() error {
			return add("billing", "Invoice", "valid = no unknown raw state; unknown = at least one unknown invoice or stage state; conflict = customer link CONFLICT.",
				func(r *oapi.SourceQualityRow) error {
					s, err := summary(domain.ReportInvoices, map[string]string{"presence": "PRESENT", "has_unknown_status": "false"})
					if err != nil {
						return err
					}
					u, err := summary(domain.ReportInvoices, map[string]string{"presence": "PRESENT", "has_unknown_status": "true"})
					if err != nil {
						return err
					}
					r.Valid, r.Unknown = i64(s.Total), i64(u.Total)
					r.Conflict = i64(s.Counts["customer_link_state"]["CONFLICT"] + u.Counts["customer_link_state"]["CONFLICT"])
					return nil
				})
		},
		func() error {
			return add("payments", "Payment attempt", "valid = explicitly mapped state; unknown = unmapped raw state (EMPTY counted in the payments report).",
				func(r *oapi.SourceQualityRow) error {
					s, err := summary(domain.ReportPayments, present)
					if err != nil {
						return err
					}
					r.Valid, r.Unknown = i64(s.Counts["state_mapping"]["MAPPED"]), i64(s.Counts["state_mapping"]["UNKNOWN"])
					return nil
				})
		},
		func() error {
			return add("customers", "Customer (one per issued-to subject)",
				"duplicate candidates share only a phone: they are NEVER merged and never one customer (Gate-0); valid = RESOLVED; invalid = NOT_RECOVERABLE.",
				func(r *oapi.SourceQualityRow) error {
					a, err := h.q.Aggregates(ctx, run.RunID, "customers")
					if err != nil {
						return err
					}
					r.Valid, r.DuplicateCandidates, r.Invalid = i64(a["state_RESOLVED"]), i64(a["phone_only_candidates"]), i64(a["state_NOT_RECOVERABLE"])
					r.Unknown, r.Conflict = i64(a["state_UNKNOWN"]), i64(a["state_CONFLICT"])
					return nil
				})
		},
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	_, cov := runHealth(run, h.clock)
	o, r := healthOutcome(run, data.Freshness, cov)
	return oapi.GetMisSourceQuality200JSONResponse{Body: oapi.SourceQualityEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Data: &data,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetMisReconciliation lists projection runs with source coverage, the
// honest F02/F03 metrics and the metric contracts.
func (h *Handlers) GetMisReconciliation(ctx context.Context, req oapi.GetMisReconciliationRequestObject) (oapi.GetMisReconciliationResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapReportingMISRead, gate[oapi.GetMisReconciliationResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetMisReconciliationResponseObject {
			return oapi.GetMisReconciliation200JSONResponse{Body: oapi.MisReconciliationEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetMisReconciliationResponseObject {
			return oapi.GetMisReconciliation403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	runs, next, err := h.q.Runs(ctx, req.Params.Cursor, req.Params.Limit)
	if isInvalid(err) {
		return oapi.GetMisReconciliation400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	data := oapi.MisReconciliation{Runs: []oapi.ProjectionRun{}, Contracts: []oapi.MetricContract{}}
	for i := range runs {
		data.Runs = append(data.Runs, *runOf(&runs[i]))
	}
	for _, ct := range domain.Registry() {
		data.Contracts = append(data.Contracts, contractOf(ct))
	}
	for id, dst := range map[string]*oapi.MetricValue{"invoices.open_preinvoices": &data.F02, "cases.staged_total": &data.F03TotalCases,
		"cases.legacy_leads_only": &data.F03LegacyLeadsOnly} {
		ct, _ := domain.Lookup(id)
		if *dst, err = h.evaluate(ctx, ct); err != nil {
			return nil, err
		}
	}
	return oapi.GetMisReconciliation200JSONResponse{Body: oapi.MisReconciliationEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: corr(ctx), Data: &data, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ---- Invoice Review (QUERY SURFACE ONLY; audited; fail closed) ---------------

// ErrAuditUnavailable reports that the sensitive-read audit could not be
// recorded; the read is refused (D-45 safe default).
var ErrAuditUnavailable = errors.New("reporting: sensitive-read audit unavailable")

func (h *Handlers) auditRead(ctx context.Context, subject refs.NativeRef, meta map[string]string) error {
	if h.auditor == nil {
		return ErrAuditUnavailable
	}
	id, err := audit.NewEventID()
	if err != nil {
		return ErrAuditUnavailable
	}
	if p, ok := authstub.FromContext(ctx); ok {
		meta["principal"] = p.ID // DEV/TEST synthetic principal only
	}
	// R1: the authenticated actor of a real session (identifiers only; no
	// token or CSRF value is ever recorded).
	if p, ok := websession.FromContext(ctx); ok {
		meta["user_id"] = p.UserID
		meta["session_id"] = p.SessionID
	}
	c := corr(ctx)
	e := audit.Event{ID: id, Subject: subject, Action: "reporting.invoice_review.read", OccurredAt: h.clock.Now(), Metadata: meta}
	if c != "" {
		e.CorrelationID = &c
	}
	if err := h.auditor(ctx, e); err != nil {
		return ErrAuditUnavailable
	}
	return nil
}

// listSubject is the audit subject of a query read: an EVENT reference with a
// decimal id (refs ids are decimal native ids, never UUIDs).
func listSubject(clock timeutil.Clock) (refs.NativeRef, error) {
	return refs.NewNativeRef(refs.DomainEvent, strconv.FormatInt(clock.Now().Time().UnixNano(), 10))
}

func failClosed() (oapi.Outcome, []oapi.ReasonCode) {
	return authzhttp.Outcome(result.Blocked), authzhttp.Reasons(reasoncodes.PolicyUnresolved)
}

// ListInvoiceReview is the audited Invoice Review query (no export).
func (h *Handlers) ListInvoiceReview(ctx context.Context, req oapi.ListInvoiceReviewRequestObject) (oapi.ListInvoiceReviewResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapInvoiceReviewRead, gate[oapi.ListInvoiceReviewResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListInvoiceReviewResponseObject {
			return oapi.ListInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListInvoiceReviewResponseObject {
			return oapi.ListInvoiceReview403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	spec := specOf(domain.ReportInvoices, nil, nil, nil, nil, nil, nil, invoiceFilters(strp(p.Presence), strp(p.CustomerLinkState),
		strp(p.CaseLinkState), p.HasUnknownStatus, nil, strp(p.TotalClass), strp(p.Unit)))
	if err := spec.Validate(); err != nil {
		return oapi.ListInvoiceReview400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	res, pg, err := h.q.Report(ctx, spec, p.Cursor, p.Limit)
	if isInvalid(err) {
		return oapi.ListInvoiceReview400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	// The read above ran in its read-only transaction; nothing leaves the
	// server unless the audit record is written first (D-45 fail closed).
	subject, err := listSubject(h.clock)
	if err != nil {
		return nil, err
	}
	meta := map[string]string{"surface": "invoice_review.list", "rows": strconv.Itoa(len(pg.Rows))}
	if res.Run != nil {
		meta["run_id"] = res.Run.RunID
	}
	for k, v := range spec.Filters {
		meta["filter_"+k] = v
	}
	if err := h.auditRead(ctx, subject, meta); err != nil {
		o, r := failClosed()
		return oapi.ListInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}, nil
	}
	items := []oapi.InvoiceFactRow{}
	for _, r := range pg.Rows {
		items = append(items, invoiceRow(r.(query.InvoiceRow)))
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	return oapi.ListInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m,
		Summary: summaryOf(res.Summary), Items: &items, Page: page(pg.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetInvoiceReview is one audited Invoice Review row.
func (h *Handlers) GetInvoiceReview(ctx context.Context, req oapi.GetInvoiceReviewRequestObject) (oapi.GetInvoiceReviewResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapInvoiceReviewRead, gate[oapi.GetInvoiceReviewResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetInvoiceReviewResponseObject {
			return oapi.GetInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetInvoiceReviewResponseObject {
			return oapi.GetInvoiceReview403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	id := req.InvoiceId.String()
	res, row, _, err := h.q.Detail(ctx, query.Spec{Report: domain.ReportInvoices}, id)
	switch {
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetInvoiceReview404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	// Audit subject: the invoice's native (legacy) reference; the target id
	// and run travel as metadata. No audit → no row (D-45 fail closed).
	subject, err := listSubject(h.clock)
	meta := map[string]string{"surface": "invoice_review.detail", "invoice_id": id}
	if row != nil {
		inv := row.(query.InvoiceRow)
		if ref, rerr := refs.NewNativeRef(refs.DomainInvoice, inv.SourceKey); rerr == nil {
			subject, err = ref, nil
		}
		meta["source_spec"] = inv.SourceSpec
	}
	if err != nil {
		return nil, err
	}
	if res.Run != nil {
		meta["run_id"] = res.Run.RunID
	}
	if err := h.auditRead(ctx, subject, meta); err != nil {
		o, r := failClosed()
		return oapi.GetInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx)}}, nil
	}
	o, r := outcomeOf(res)
	m := metaOf(res)
	env := oapi.InvoiceReviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Meta: &m, AllowedActions: authzhttp.AllowedActions(c.Decisions)}
	if row != nil {
		d := invoiceRow(row.(query.InvoiceRow))
		env.Data = &d
	}
	return oapi.GetInvoiceReview200JSONResponse{Body: env}, nil
}
