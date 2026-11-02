// Package http adapts the fulfillment READ shadow service to the generated
// OpenAPI strict server. Handlers guard (flag biawin_fulfillment.read_shadow
// + evaluator + ORGANIZATION scope), call the read service and map results.
// GET only: no SQL, no import, no assignment, no shipping or completion and no write
// of any kind (architecture lint). Every answer repeats the SHADOW notice
// and names the one run it came from.
package http

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/query"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Notice is the fixed SHADOW notice of every Biawin / fulfillment answer.
const Notice = oapi.SHADOWImportedLegacySubscriptionAndFulfillmentEvidenceForArchitectureValidationAndParityOnlyNotLiveProviderStateNoActivationProviderShippingOrWorkflowAuthority

// Handlers serves the fulfillment read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

func corr(ctx context.Context) string { return httpx.CorrelationID(ctx) }

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

// guardOp checks flag + capability and the evaluator scope. Only
// ORGANIZATION is accepted; anything narrower is denied, never broadened
// (no fulfillment row-scope resolver exists).
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

func codes(in []string) []reasoncodes.Code {
	var out []reasoncodes.Code
	for _, r := range in {
		if c, err := reasoncodes.ParseCode(r); err == nil {
			out = append(out, c)
		}
	}
	return out
}

// outcomeOf: the run's technical outcome; a STALE run is PARTIAL with
// STALE_STATE; no run is UNKNOWN (never an empty zero answer).
func outcomeOf(run *query.Run) (oapi.Outcome, []oapi.ReasonCode) {
	if run == nil {
		return authzhttp.Outcome(result.Unknown), authzhttp.Reasons(reasoncodes.SourceUnavailable)
	}
	status := result.Success
	cs := codes(run.Reasons)
	if run.Outcome == "PARTIAL" {
		status = result.Partial
	}
	if run.Freshness == "STALE" {
		status = result.Partial
		cs = append(cs, reasoncodes.StaleState)
	}
	cs, _ = reasoncodes.Normalize(cs)
	return authzhttp.Outcome(status), authzhttp.Reasons(cs...)
}

func reasonsOf(in []string) oapi.Reasons {
	out := oapi.Reasons{}
	for _, c := range codes(in) {
		out = append(out, oapi.ReasonCode(c))
	}
	return out
}

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

func limitOf(l *oapi.Limit) int {
	if l == nil {
		return 0
	}
	return int(*l)
}

func idOf(u *openapi_types.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

func strOf[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func runOf(r *query.Run) oapi.BiawinRun {
	out := oapi.BiawinRun{RecordKind: oapi.SHADOW, RunId: authzhttp.UUID(r.RunID), RunSeq: r.Seq, NormalizationVersion: r.NormalizationVersion,
		SnapshotId: r.SnapshotID, SourceCutoff: r.SourceCutoff.UTC(), StartedAt: r.Started.UTC(), CompletedAt: r.Completed.UTC(),
		Outcome: oapi.BiawinRunOutcome(r.Outcome), Reasons: reasonsOf(r.Reasons), Counts: r.Counts, Sources: []oapi.BiawinRunSource{},
		Freshness: oapi.BiawinFreshness(r.Freshness)}
	if out.Counts == nil {
		out.Counts = map[string]int64{}
	}
	for _, s := range r.Sources {
		out.Sources = append(out.Sources, oapi.BiawinRunSource{Source: s.Source, Status: oapi.BiawinCoverage(s.Status), RowsRead: s.Rows})
	}
	return out
}

func metaOf(r *query.Run, kv map[string]*string) *oapi.BiawinMeta {
	f := map[string]string{}
	for k, v := range kv {
		if v != nil {
			f[k] = *v
		}
	}
	return &oapi.BiawinMeta{RecordKind: oapi.SHADOW, Notice: Notice, Run: runOf(r), Query: oapi.BiawinQuery{RunId: authzhttp.UUID(r.RunID), Filters: f}}
}

func caseOf(c query.Case) oapi.FulfillmentCase {
	out := oapi.FulfillmentCase{CaseId: authzhttp.UUID(c.CaseID), Kind: oapi.FulfillmentKind(c.Kind), SourceSpec: c.SourceSpec, SourceKey: c.SourceKey,
		SubtypeRaw: c.SubtypeRaw, SubscriptionId: authzhttp.OptUUID(c.SubscriptionID), ItemId: authzhttp.OptUUID(c.ItemID),
		InvoiceLinkState: oapi.ItemInvoiceLinkState(c.InvoiceLinkState), InvoiceId: authzhttp.OptUUID(c.InvoiceID), InvoiceKey: c.InvoiceKey,
		StatusNamespace: c.StatusNamespace, StatusRaw: c.StatusRaw, StatusMapped: oapi.FulfillmentStatus(c.StatusMapped),
		ShippingStatusRaw: c.ShippingStatusRaw, ProviderStatusRaw: c.ProviderStatusRaw, ContactStatusRaw: c.ContactStatusRaw,
		CompletedAtRaw: c.CompletedAtRaw, CompletedByKey: c.CompletedByKey, CompletedByUserId: authzhttp.OptUUID(c.CompletedByUserID),
		CancelledAtRaw: c.CancelledAtRaw, ShippedAtRaw: c.ShippedAtRaw, DeliveredAtRaw: c.DeliveredAtRaw, StatusUpdatedAtRaw: c.StatusUpdatedAtRaw,
		CarrierRaw: c.CarrierRaw, TrackingRefSha256: c.TrackingRef, ProviderRefSha256: c.ProviderRef, ProviderRefScheme: c.ProviderRefScheme,
		CreatedAtRaw: c.CreatedAtRaw, UpdatedAtRaw: c.UpdatedAtRaw, Assignments: []oapi.FulfillmentAssignment{}}
	for _, a := range c.Assignments {
		out.Assignments = append(out.Assignments, oapi.FulfillmentAssignment{Role: oapi.FulfillmentAssignmentRole(a.Role), UserKey: a.UserKey,
			UserId: authzhttp.OptUUID(a.UserID), AssignedAtRaw: a.AssignedAtRaw})
	}
	return out
}

func isInvalid(err error) bool  { return errors.Is(err, query.ErrInvalid) }
func isNoRun(err error) bool    { return errors.Is(err, query.ErrNoRun) }
func isNotFound(err error) bool { return errors.Is(err, query.ErrNotFound) }

// ListFulfillmentCases lists the cases of one run.
func (h *Handlers) ListFulfillmentCases(ctx context.Context, req oapi.ListFulfillmentCasesRequestObject) (oapi.ListFulfillmentCasesResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFulfillmentCaseRead, gate[oapi.ListFulfillmentCasesResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFulfillmentCasesResponseObject {
			return oapi.ListFulfillmentCases200JSONResponse{Body: oapi.FulfillmentCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFulfillmentCasesResponseObject {
			return oapi.ListFulfillmentCases403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	f := query.Filter{RunID: idOf(p.RunId), Kind: strOf(p.Kind), Status: strOf(p.Status), SubscriptionID: idOf(p.SubscriptionId),
		ItemID: idOf(p.ItemId), AssigneeUserID: idOf(p.Assignee)}
	run, cs, sum, next, err := h.q.Cases(ctx, f, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListFulfillmentCases400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListFulfillmentCases200JSONResponse{Body: oapi.FulfillmentCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.FulfillmentCase{}
	for _, x := range cs {
		items = append(items, caseOf(x))
	}
	counts := []oapi.CaseCount{}
	for _, x := range sum {
		counts = append(counts, oapi.CaseCount{Kind: oapi.FulfillmentKind(x.Kind), Status: oapi.FulfillmentStatus(x.Status), Count: x.N})
	}
	o, r := outcomeOf(&run)
	kv := map[string]*string{"kind": f.Kind, "status": f.Status, "subscription_id": f.SubscriptionID, "item_id": f.ItemID, "assignee": f.AssigneeUserID}
	return oapi.ListFulfillmentCases200JSONResponse{Body: oapi.FulfillmentCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: metaOf(&run, kv), Summary: &counts, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetFulfillmentCase returns one case of one run.
func (h *Handlers) GetFulfillmentCase(ctx context.Context, req oapi.GetFulfillmentCaseRequestObject) (oapi.GetFulfillmentCaseResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFulfillmentCaseRead, gate[oapi.GetFulfillmentCaseResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetFulfillmentCaseResponseObject {
			return oapi.GetFulfillmentCase200JSONResponse{Body: oapi.FulfillmentCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetFulfillmentCaseResponseObject {
			return oapi.GetFulfillmentCase403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, d, err := h.q.CaseByID(ctx, idOf(req.Params.RunId), req.CaseId.String())
	switch {
	case isInvalid(err):
		return oapi.GetFulfillmentCase400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err), isNotFound(err):
		return oapi.GetFulfillmentCase404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	cs := caseOf(d.Case)
	ev := []oapi.ServiceEvidence{}
	for _, e := range d.Evidence {
		ev = append(ev, oapi.ServiceEvidence{SourceKey: e.SourceKey, OutcomeMapped: oapi.ServiceEvidenceOutcomeMapped(e.OutcomeMapped),
			FormDefinitionKey: e.FormDefinitionKey, SubmittedByKey: e.SubmittedByKey, SubmittedByUserId: authzhttp.OptUUID(e.SubmittedByUserID),
			SubmittedAtRaw: e.SubmittedAtRaw, OutcomeRaw: e.OutcomeRaw, DecidedByKey: e.DecidedByKey, DecidedAtRaw: e.DecidedAtRaw})
	}
	o, r := outcomeOf(&run)
	return oapi.GetFulfillmentCase200JSONResponse{Body: oapi.FulfillmentCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: metaOf(&run, nil), Case: &cs, Evidence: &ev, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListFulfillmentCaseHistory lists one case's append-only history of one run.
func (h *Handlers) ListFulfillmentCaseHistory(ctx context.Context, req oapi.ListFulfillmentCaseHistoryRequestObject) (oapi.ListFulfillmentCaseHistoryResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapFulfillmentCaseRead, gate[oapi.ListFulfillmentCaseHistoryResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListFulfillmentCaseHistoryResponseObject {
			return oapi.ListFulfillmentCaseHistory200JSONResponse{Body: oapi.FulfillmentCaseHistoryEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListFulfillmentCaseHistoryResponseObject {
			return oapi.ListFulfillmentCaseHistory403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	run, evs, next, err := h.q.History(ctx, idOf(p.RunId), req.CaseId.String(), query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListFulfillmentCaseHistory400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err), isNotFound(err):
		return oapi.ListFulfillmentCaseHistory404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.CaseEvent{}
	for _, e := range evs {
		items = append(items, oapi.CaseEvent{SourceSpec: e.SourceSpec, SourceKey: e.SourceKey, EventKeyRaw: e.EventKeyRaw, ActorKey: e.ActorKey,
			ActorUserId: authzhttp.OptUUID(e.ActorUserID), OldStatusRaw: e.OldStatusRaw, NewStatusRaw: e.NewStatusRaw, OccurredAtRaw: e.OccurredAtRaw})
	}
	o, r := outcomeOf(&run)
	return oapi.ListFulfillmentCaseHistory200JSONResponse{Body: oapi.FulfillmentCaseHistoryEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: metaOf(&run, nil), Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
