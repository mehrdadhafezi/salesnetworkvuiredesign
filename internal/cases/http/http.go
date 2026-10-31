// Package http adapts the cases read service to the generated OpenAPI strict
// server. Handlers guard (flag + evaluator), call the read service and map
// rows to contract types. No SQL, no writes, no import command (lint).
//
// The coverage report carries the F02/F03 labels exactly as the frozen
// authorities define them; a metric that cannot be evaluated in Slice 6 is
// NOT_EVALUATED with a null count — never zero.
package http

import (
	"context"
	"errors"
	"strconv"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/store"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the cases read operations and the coverage report.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

// SalesEnabled reports the Slice 6 flag (session navigation hint).
func (h *Handlers) SalesEnabled(ctx context.Context) bool {
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

func userRef(key, user *string) *oapi.UserRef {
	if key == nil {
		return nil
	}
	return &oapi.UserRef{LegacyKey: *key, UserId: uuidp(user)}
}

func caseSummary(c store.Case) oapi.CaseSummary {
	return oapi.CaseSummary{CaseId: authzhttp.UUID(c.CaseID), WorkflowType: oapi.WorkflowType(c.Workflow), StateNamespace: c.StateNamespace,
		StateRaw: c.StateRaw, StateMapping: oapi.StateMapping(c.StateMapping), CustomerId: uuidp(c.CustomerID),
		CustomerLinkState: oapi.ResolutionState(c.CustomerLinkState), CustomerLinkReason: c.CustomerLinkReason, Presence: oapi.Presence(c.Presence),
		Source: oapi.CaseSource{SourceSystem: c.Source.System, SourceSpec: c.Source.Spec, SourceKey: c.Source.Key, ChannelRaw: c.Source.ChannelRaw,
			OriginRefRaw: c.Source.OriginRefRaw, FirstBatchId: authzhttp.UUID(c.Source.FirstBatchID), LastBatchId: authzhttp.UUID(c.Source.LastBatch),
			Presence: oapi.Presence(c.Source.Presence), FirstSeenAt: c.Source.FirstSeen, LastSeenAt: c.Source.LastSeen},
		AliasCount: c.AliasCount, HistoryCount: c.HistoryCount, ImportedAt: c.ImportedAt, RefreshedAt: c.RefreshedAt}
}

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

// guard runs flag + evaluator and returns a terminal response when blocked.
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
	return c, nil, nil
}

// ListCases lists shadow Cases.
func (h *Handlers) ListCases(ctx context.Context, req oapi.ListCasesRequestObject) (oapi.ListCasesResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapCaseRead, gate[oapi.ListCasesResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListCasesResponseObject {
			return oapi.ListCases200JSONResponse{Body: oapi.CaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListCasesResponseObject {
			return oapi.ListCases403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	res, err := h.q.ListCases(ctx, store.CaseFilter{Workflow: strp(p.WorkflowType), StateMapping: strp(p.StateMapping),
		CustomerLinkState: strp(p.CustomerLinkState), Presence: strp(p.Presence)}, p.Cursor, p.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListCases400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.CaseSummary, 0, len(res.Cases))
	for _, x := range res.Cases {
		items = append(items, caseSummary(x))
	}
	o, r := ok()
	return oapi.ListCases200JSONResponse{Body: oapi.CaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Items: &items,
		Page: &oapi.PageInfo{HasMore: res.NextCursor != nil, NextCursor: res.NextCursor}, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func deref[R any](p *R) R {
	var zero R
	if p == nil {
		return zero
	}
	return *p
}

// GetCase returns one Case with lead facts and explicit links.
func (h *Handlers) GetCase(ctx context.Context, req oapi.GetCaseRequestObject) (oapi.GetCaseResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapCaseRead, gate[oapi.GetCaseResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetCaseResponseObject {
			return oapi.GetCase200JSONResponse{Body: oapi.CaseEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetCaseResponseObject {
			return oapi.GetCase403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	d, err := h.q.GetCase(ctx, req.CaseId.String())
	if errors.Is(err, query.ErrNotFound) {
		return oapi.GetCase404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	data := oapi.CaseDetail{Case: caseSummary(d.Case), Aliases: []oapi.CaseAlias{}}
	if l := d.Lead; l != nil {
		data.Lead = &oapi.LeadFacts{StatusRaw: l.StatusRaw, LeadStatusRaw: l.LeadStatusRaw, DestinationPanelRaw: l.DestinationPanelRaw,
			ImportedAtRaw: l.ImportedAtRaw, AssignedAtRaw: l.AssignedAtRaw, UpdatedAtRaw: l.UpdatedAtRaw, Presence: oapi.Presence(l.Presence)}
	}
	for _, a := range d.Aliases {
		data.Aliases = append(data.Aliases, oapi.CaseAlias{AliasId: authzhttp.UUID(a.AliasID), CaseId: authzhttp.UUID(a.CaseID),
			RelatedCaseId: uuidp(a.RelatedCaseID), RelatedSourceSpec: a.RelatedSpec, RelatedSourceKey: a.RelatedKey,
			Relation: oapi.CaseAliasRelation(a.Relation), Resolution: oapi.ResolutionState(a.Resolution), Basis: a.Basis,
			Presence: oapi.Presence(a.Presence), RecordedAt: a.RecordedAt})
	}
	o, r := ok()
	return oapi.GetCase200JSONResponse{Body: oapi.CaseEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Data: &data,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCaseHistory pages a Case's append-only state history.
func (h *Handlers) ListCaseHistory(ctx context.Context, req oapi.ListCaseHistoryRequestObject) (oapi.ListCaseHistoryResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapCaseRead, gate[oapi.ListCaseHistoryResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListCaseHistoryResponseObject {
			return oapi.ListCaseHistory200JSONResponse{Body: oapi.StateHistoryListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListCaseHistoryResponseObject {
			return oapi.ListCaseHistory403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	res, err := h.q.History(ctx, req.CaseId.String(), req.Params.Cursor, req.Params.Limit)
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.ListCaseHistory400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.ListCaseHistory404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := make([]oapi.StateHistoryEntry, 0, len(res.Rows))
	for _, x := range res.Rows {
		items = append(items, oapi.StateHistoryEntry{HistoryId: authzhttp.UUID(x.HistoryID), SourceSpec: x.SourceSpec, SourceKey: x.SourceKey,
			OldStateRaw: x.OldRaw, NewStateRaw: x.NewRaw, Actor: userRef(x.ActorKey, x.ActorUserID), OccurredRaw: x.OccurredRaw, RecordedAt: x.RecordedAt})
	}
	o, r := ok()
	return oapi.ListCaseHistory200JSONResponse{Body: oapi.StateHistoryListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: &oapi.PageInfo{HasMore: res.NextCursor != nil, NextCursor: res.NextCursor}, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCaseAttribution lists the immutable attribution snapshots of a Case.
func (h *Handlers) ListCaseAttribution(ctx context.Context, req oapi.ListCaseAttributionRequestObject) (oapi.ListCaseAttributionResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapCaseRead, gate[oapi.ListCaseAttributionResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListCaseAttributionResponseObject {
			return oapi.ListCaseAttribution200JSONResponse{Body: oapi.AttributionListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListCaseAttributionResponseObject {
			return oapi.ListCaseAttribution403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	rows, err := h.q.Attribution(ctx, req.CaseId.String())
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.ListCaseAttribution400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.ListCaseAttribution404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := make([]oapi.AttributionSnapshot, 0, len(rows))
	for _, s := range rows {
		items = append(items, oapi.AttributionSnapshot{SnapshotId: authzhttp.UUID(s.SnapshotID), Facet: oapi.Facet(s.Facet),
			Resolution: oapi.ResolutionState(s.Resolution), Subject: userRef(s.SubjectKey, s.SubjectUserID), PositionRaw: s.PositionRaw,
			Basis: s.Basis, SourceTimeRaw: s.SourceTimeRaw, EvidenceSpec: s.EvidenceSpec, EvidenceKey: s.EvidenceKey,
			CapturedBatchId: authzhttp.UUID(s.CapturedBatchID), CapturedAt: s.CapturedAt})
	}
	o, r := ok()
	return oapi.ListCaseAttribution200JSONResponse{Body: oapi.AttributionListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// Authoritative definitions (quoted from the frozen sources; not invented).
const (
	authorityF03 = "CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md F03; GATE-0-PRODUCT-INVARIANTS.md metric table (Total Cases, Legacy Leads Only); CRM-IMPLEMENTATION-FOUNDATION-PLAN.md F03"
	authorityF02 = "CRM-PRODUCT-ROLE-ARCHITECTURE-AUDIT.fa.md F02; GATE-0-PRODUCT-INVARIANTS.md metric table (Open Pre-invoices); CRM-IMPLEMENTATION-FOUNDATION-PLAN.md F02"
	defLeadsOnly = "Legacy lead fact only: distinct legacy lead ids from the legacy lead table, explicitly labelled; it is not Total Cases and must never be named the whole CRM (GATE-0)."
	defTotal     = "Total Cases: distinct Cases across the declared sources (legacy + V4 and other named sources), deduplicated only by PROVEN aliases, with the unresolved count separate (GATE-0)."
	defF02       = "Open Pre-invoices: distinct invoices whose current normalized state is pre_invoice without completion/cancel, at a current snapshot; the legacy predicate omitted pre_invoice (F02)."
)

func i64(n int64) *int64 { return &n }

// CoverageMetrics builds the honest F02/F03 metrics from the coverage.
func CoverageMetrics(cov store.CoverageView) (leads, total, f02 oapi.HonestMetric) {
	leads = oapi.HonestMetric{Code: "F03", Label: "Legacy Leads Only (not Total Cases)", Status: "EVALUATED", Count: i64(cov.LegacyLeadsOnly),
		Definition: defLeadsOnly, Authority: authorityF03, Coverage: "legacy lead source as staged", Reason: "counted from present legacy lead facts"}
	unresolved := cov.AliasesRelated + cov.AliasesUnknown
	total = oapi.HonestMetric{Code: "F03", Label: "Total Cases (staged sources only; not a proven cross-source total)", Status: "PARTIAL",
		Count: i64(cov.CasesPresent), Definition: defTotal, Authority: authorityF03,
		Coverage: "INCOMPLETE: only the legacy lead and distribution item sources are staged in Slice 6",
		Reason:   "no alias is proven (OPD-01 open), so nothing is deduplicated; " + strconv.FormatInt(unresolved, 10) + " explicit links remain unresolved"}
	f02 = oapi.HonestMetric{Code: "F02", Label: "Open Pre-invoices", Status: "NOT_EVALUATED", Count: nil, Definition: defF02, Authority: authorityF02,
		Coverage: "NONE: invoice state is not staged in Slice 6 (sales read only; billing is a later slice)",
		Reason:   "F02 is an invoice-grain predicate; Slice 6 holds no invoice state, so no count is claimed (null is not zero)"}
	return leads, total, f02
}

// GetSalesCoverageReport returns the source coverage report.
func (h *Handlers) GetSalesCoverageReport(ctx context.Context, _ oapi.GetSalesCoverageReportRequestObject) (oapi.GetSalesCoverageReportResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapSalesReportRead, gate[oapi.GetSalesCoverageReportResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetSalesCoverageReportResponseObject {
			return oapi.GetSalesCoverageReport200JSONResponse{Body: oapi.SalesCoverageReportEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetSalesCoverageReportResponseObject {
			return oapi.GetSalesCoverageReport403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	cov, err := h.q.CoverageReport(ctx)
	if err != nil {
		return nil, err
	}
	view := store.CoverageView{LegacyLeadsOnly: cov.LegacyLeadsOnly, CasesPresent: cov.CasesPresent,
		AliasesRelated: cov.AliasesByResolution["RELATED"], AliasesUnknown: cov.AliasesByResolution["UNKNOWN"]}
	leads, total, f02 := CoverageMetrics(view)
	attr := map[string]oapi.CountMap{}
	for f, m := range cov.AttributionByFacet {
		attr[f] = oapi.CountMap(m)
	}
	data := oapi.SalesCoverageReport{CasesTotal: cov.CasesTotal, CasesPresent: cov.CasesPresent, CasesBySourceSpec: oapi.CountMap(cov.CasesBySource),
		CasesByWorkflowType: oapi.CountMap(cov.CasesByWorkflow), CasesWithUnknownStateMapping: cov.UnknownStates, SourcelessCases: cov.SourcelessCases,
		AliasesByResolution: oapi.CountMap(cov.AliasesByResolution), StatusHistoryRows: cov.StatusHistoryRows, CasesWithStatusHistory: cov.CasesWithHistory,
		AttributionByFacetAndResolution: attr, CustomerLinkStates: oapi.CountMap(cov.CustomerLinks), LegacyLeadsOnly: leads, TotalCases: total, F02: f02,
		SourceCoverage: map[string]string{
			"legacy_leads":                  "STAGED (as declared by the import batch)",
			"distribution_items":            "STAGED (as declared by the import batch)",
			"legacy_seller_flow_states":     "UNKNOWN (not staged in Slice 6)",
			"dot_conversion_cases":          "UNKNOWN (not staged in Slice 6)",
			"mis_source_rows_and_plans":     "UNKNOWN (not staged in Slice 6)",
			"invoice_state_for_f02":         "NOT_STAGED (billing is a later slice)",
			"customer_references_for_cases": "NONE: no explicit customer reference in staged sales sources; customer linkage stays UNKNOWN",
		}}
	o, r := ok()
	return oapi.GetSalesCoverageReport200JSONResponse{Body: oapi.SalesCoverageReportEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
