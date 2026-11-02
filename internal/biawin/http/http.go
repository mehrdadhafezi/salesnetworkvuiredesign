// Package http adapts the biawin READ shadow service to the generated
// OpenAPI strict server. Handlers guard (flag biawin_fulfillment.read_shadow
// + evaluator + ORGANIZATION scope), call the read service and map results.
// GET only: no SQL, no import, no activation, no provider call and no write
// of any kind (architecture lint). Every answer repeats the SHADOW notice
// and names the one run it came from.
package http

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/query"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Notice is the fixed SHADOW notice of every Biawin / fulfillment answer.
const Notice = oapi.SHADOWImportedLegacySubscriptionAndFulfillmentEvidenceForArchitectureValidationAndParityOnlyNotLiveProviderStateNoActivationProviderShippingOrWorkflowAuthority

// Handlers serves the biawin read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

// BiawinEnabled reports the Slice 10 flag (session navigation hint).
func (h *Handlers) BiawinEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func corr(ctx context.Context) string { return httpx.CorrelationID(ctx) }

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

// guardOp checks flag + capability and the evaluator scope. Only
// ORGANIZATION is accepted; anything narrower is denied, never broadened
// (no subscription row-scope resolver exists).
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

func link(state, basis string, id, key *string) oapi.BiawinLink {
	l := oapi.BiawinLink{State: oapi.LinkState(state), Id: authzhttp.OptUUID(id), LegacyKey: key}
	if basis != "" {
		l.Basis = &basis
	}
	return l
}

func subOf(s query.Subscription) oapi.BiawinSubscription {
	return oapi.BiawinSubscription{SubscriptionId: authzhttp.UUID(s.SubscriptionID), SourceSpec: s.SourceSpec, SourceKey: s.SourceKey,
		Provider: s.Provider, SourceRefRaw: s.SourceRefRaw, SourceTypeRaw: s.SourceTypeRaw, ProductKey: s.ProductKey,
		ProductTitleSnapshot: s.ProductTitleSnapshot, ProductVersionState: oapi.BiawinSubscriptionProductVersionState(s.ProductVersionState),
		Customer:            link(s.CustomerLinkState, s.CustomerLinkBasis, s.CustomerID, s.CustomerAccountKey),
		Invoice:             link(s.InvoiceLinkState, "", s.InvoiceID, s.InvoiceKey),
		FirstPaymentInvoice: link(s.FirstPaymentLinkState, "", s.FirstPaymentInvoiceID, s.FirstPaymentInvoiceKey),
		Case:                link(s.CaseLinkState, s.CaseLinkBasis, s.CaseID, s.DotCaseKey),
		OriginUserKey:       s.OriginUserKey, OriginUserId: authzhttp.OptUUID(s.OriginUserID), StateNamespace: s.StateNamespace, StateRaw: s.StateRaw,
		StateMapped: oapi.SubscriptionState(s.StateMapped), ActivationEvidence: oapi.ActivationEvidence(s.ActivationEvidence),
		ActivatedAtRaw: s.ActivatedAtRaw, CreatedAtRaw: s.CreatedAtRaw, UpdatedAtRaw: s.UpdatedAtRaw, ItemCount: s.ItemCount}
}

func isInvalid(err error) bool  { return errors.Is(err, query.ErrInvalid) }
func isNoRun(err error) bool    { return errors.Is(err, query.ErrNoRun) }
func isNotFound(err error) bool { return errors.Is(err, query.ErrNotFound) }

// ListBiawinSubscriptions lists the subscriptions of one run.
func (h *Handlers) ListBiawinSubscriptions(ctx context.Context, req oapi.ListBiawinSubscriptionsRequestObject) (oapi.ListBiawinSubscriptionsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapBiawinSubscriptionRead, gate[oapi.ListBiawinSubscriptionsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListBiawinSubscriptionsResponseObject {
			return oapi.ListBiawinSubscriptions200JSONResponse{Body: oapi.SubscriptionListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListBiawinSubscriptionsResponseObject {
			return oapi.ListBiawinSubscriptions403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	f := query.Filter{RunID: idOf(p.RunId), State: strOf(p.State), CustomerLinkState: strOf(p.CustomerLinkState),
		InvoiceLinkState: strOf(p.InvoiceLinkState), ActivationEvidence: strOf(p.ActivationEvidence), ProductKey: p.Product}
	run, subs, sum, next, err := h.q.Subscriptions(ctx, f, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListBiawinSubscriptions400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListBiawinSubscriptions200JSONResponse{Body: oapi.SubscriptionListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.BiawinSubscription{}
	for _, s := range subs {
		items = append(items, subOf(s))
	}
	counts := []oapi.StateCount{}
	for _, x := range sum {
		counts = append(counts, oapi.StateCount{Key: x.Key, Count: x.N})
	}
	o, r := outcomeOf(&run)
	kv := map[string]*string{"state": f.State, "customer_link_state": f.CustomerLinkState, "invoice_link_state": f.InvoiceLinkState,
		"activation_evidence": f.ActivationEvidence, "product": f.ProductKey}
	return oapi.ListBiawinSubscriptions200JSONResponse{Body: oapi.SubscriptionListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: metaOf(&run, kv), Summary: &counts, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetBiawinSubscription returns one subscription of one run with its history.
func (h *Handlers) GetBiawinSubscription(ctx context.Context, req oapi.GetBiawinSubscriptionRequestObject) (oapi.GetBiawinSubscriptionResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapBiawinSubscriptionRead, gate[oapi.GetBiawinSubscriptionResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetBiawinSubscriptionResponseObject {
			return oapi.GetBiawinSubscription200JSONResponse{Body: oapi.SubscriptionEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetBiawinSubscriptionResponseObject {
			return oapi.GetBiawinSubscription403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, d, err := h.q.SubscriptionByID(ctx, idOf(req.Params.RunId), req.SubscriptionId.String())
	switch {
	case isInvalid(err):
		return oapi.GetBiawinSubscription400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err), isNotFound(err):
		return oapi.GetBiawinSubscription404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	sub := subOf(d.Subscription)
	hist := []oapi.SubscriptionEvent{}
	for _, e := range d.Events {
		hist = append(hist, oapi.SubscriptionEvent{SourceKey: e.SourceKey, EventClass: oapi.SubscriptionEventEventClass(e.EventClass),
			EventKeyRaw: e.EventKeyRaw, ItemId: authzhttp.OptUUID(e.ItemID), ActorKey: e.ActorKey, ActorUserId: authzhttp.OptUUID(e.ActorUserID),
			OccurredAtRaw: e.OccurredAtRaw})
	}
	o, r := outcomeOf(&run)
	return oapi.GetBiawinSubscription200JSONResponse{Body: oapi.SubscriptionEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: metaOf(&run, nil), Subscription: &sub, History: &hist, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListBiawinSubscriptionItems lists the items of one subscription of one run.
func (h *Handlers) ListBiawinSubscriptionItems(ctx context.Context, req oapi.ListBiawinSubscriptionItemsRequestObject) (oapi.ListBiawinSubscriptionItemsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapBiawinSubscriptionRead, gate[oapi.ListBiawinSubscriptionItemsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListBiawinSubscriptionItemsResponseObject {
			return oapi.ListBiawinSubscriptionItems200JSONResponse{Body: oapi.SubscriptionItemListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListBiawinSubscriptionItemsResponseObject {
			return oapi.ListBiawinSubscriptionItems403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	run, its, total, next, err := h.q.Items(ctx, idOf(p.RunId), req.SubscriptionId.String(), query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListBiawinSubscriptionItems400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNoRun(err), isNotFound(err):
		return oapi.ListBiawinSubscriptionItems404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.SubscriptionItem{}
	for _, it := range its {
		x := oapi.SubscriptionItem{ItemId: authzhttp.UUID(it.ItemID), SubscriptionId: authzhttp.UUID(it.SubscriptionID), SourceKey: it.SourceKey,
			State: oapi.SubscriptionItemState(it.StateMapped), ServiceKey: it.ServiceKey, ServiceTitleSnapshot: it.ServiceTitleSnapshot,
			CreatedAtRaw: it.CreatedAtRaw, InvoiceLinks: []oapi.ItemInvoiceLink{}}
		for _, l := range it.InvoiceLinks {
			x.InvoiceLinks = append(x.InvoiceLinks, oapi.ItemInvoiceLink{SourceKey: l.SourceKey, State: oapi.ItemInvoiceLinkState(l.State),
				InvoiceId: authzhttp.OptUUID(l.InvoiceID), InvoiceKey: l.InvoiceKey, StatusRaw: l.StatusRaw, VerifiedAtRaw: l.Verified})
		}
		items = append(items, x)
	}
	o, r := outcomeOf(&run)
	return oapi.ListBiawinSubscriptionItems200JSONResponse{Body: oapi.SubscriptionItemListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: metaOf(&run, nil), ItemCount: &total, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetBiawinParity returns the latest (or a named) parity check of one run.
func (h *Handlers) GetBiawinParity(ctx context.Context, req oapi.GetBiawinParityRequestObject) (oapi.GetBiawinParityResponseObject, error) {
	c, stop, err := guardOp(ctx, h, identityapi.CapBiawinParityRead, gate[oapi.GetBiawinParityResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetBiawinParityResponseObject {
			return oapi.GetBiawinParity200JSONResponse{Body: oapi.ParityEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetBiawinParityResponseObject {
			return oapi.GetBiawinParity403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	res := strOf(p.Result)
	run, pc, next, err := h.q.Parity(ctx, idOf(p.RunId), idOf(p.CheckId), res, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.GetBiawinParity400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNotFound(err):
		return oapi.GetBiawinParity404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.GetBiawinParity200JSONResponse{Body: oapi.ParityEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	meta := metaOf(&run, map[string]*string{"check_id": idOf(p.CheckId), "result": res})
	if pc == nil {
		// A run without a recorded parity check: UNKNOWN, never MATCH or zero.
		return oapi.GetBiawinParity200JSONResponse{Body: oapi.ParityEnvelope{Outcome: authzhttp.Outcome(result.Unknown),
			Reasons: authzhttp.Reasons(reasoncodes.OutcomeUnknown), CorrelationId: corr(ctx), Notice: Notice, Meta: meta,
			AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
	}
	chk := oapi.ParityCheck{CheckId: authzhttp.UUID(pc.CheckID), RunId: authzhttp.UUID(pc.RunID), EvidenceSnapshotId: pc.EvidenceSnapshotID,
		EvidenceCutoff: pc.EvidenceCutoff.UTC(), EvidenceStatus: oapi.BiawinCoverage(pc.EvidenceStatus), CheckedAt: pc.CheckedAt.UTC(),
		Outcome: oapi.ParityOutcome(pc.Outcome), Reasons: reasonsOf(pc.Reasons), LegacyCount: pc.LegacyCount, TargetCount: pc.TargetCount,
		Matched: pc.Matched, MissingTarget: pc.MissingTarget, ExtraTarget: pc.ExtraTarget, Unknown: pc.Unknown, Conflict: pc.Conflict,
		NotRecoverable: pc.NotRecoverable, CutoffMismatch: pc.CutoffMismatch, Rows: []oapi.ParityRow{}}
	for _, x := range pc.Rows {
		chk.Rows = append(chk.Rows, oapi.ParityRow{SourceKey: x.SourceKey, Result: oapi.ParityResult(x.Result), LegacyClass: x.LegacyClass, TargetState: x.TargetState})
	}
	o, r := outcomeOf(&run)
	return oapi.GetBiawinParity200JSONResponse{Body: oapi.ParityEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Meta: meta, Check: &chk, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
