// Package http adapts the commission READ shadow service to the generated
// OpenAPI strict server. Handlers guard (flag commission.read_shadow +
// evaluator), call the read service and map STORED results. GET only: no
// SQL, no calculation, no recalculation, no import, no approval, no posting
// and no write of any kind (architecture lint). Every answer repeats the
// SHADOW notice; posting_state is always BLOCKED_NOT_AUTHORIZED.
package http

import (
	"context"
	"encoding/json"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/query"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Notice is the fixed SHADOW notice of every commission answer.
const Notice = oapi.SHADOWOnlyNoCommissionWasCalculatedApprovedPostedOrCreditedByThisReadLegacyRemainsTheSystemOfRecord

var _ = func() bool {
	if string(Notice) != api.Notice {
		panic("commission: OpenAPI notice and api.Notice differ")
	}
	return true
}()

// Handlers serves the commission read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

// CommissionEnabled reports the Slice 13 flag (session navigation hint).
func (h *Handlers) CommissionEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func corr(ctx context.Context) string { return httpx.CorrelationID(ctx) }

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

// guardOp checks flag + capability. Organization reads need ORGANIZATION
// scope (never broadened); the own read is row-filtered by the caller.
func guardOp[R any](ctx context.Context, h *Handlers, cp identityapi.Capability, own bool, g gate[R]) (authorization.Check, *R, error) {
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
	if !own && c.Decision.Scope != authorization.ScopeOrganization {
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

// outcomeOf: the run's technical outcome; no run is UNKNOWN (never zero).
func outcomeOf(run *api.Run) (oapi.Outcome, []oapi.ReasonCode) {
	if run == nil {
		return authzhttp.Outcome(result.Unknown), authzhttp.Reasons(reasoncodes.SourceUnavailable)
	}
	status := result.Success
	if run.Outcome == "PARTIAL" {
		status = result.Partial
	}
	cs, _ := reasoncodes.Normalize(codes(run.Reasons))
	return authzhttp.Outcome(status), authzhttp.Reasons(cs...)
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

func limitOf(l *oapi.Limit) int {
	if l == nil {
		return 0
	}
	return int(*l)
}

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

func runOf(r api.Run) oapi.CommissionRun {
	out := oapi.CommissionRun{RecordKind: oapi.SHADOW, RunId: authzhttp.UUID(r.RunID), RunSeq: r.RunSeq, State: oapi.CommissionRunState(r.State),
		NormalizationVersion: r.NormalizationVersion, Rounding: oapi.CommissionRunRounding(r.Rounding), SnapshotId: r.SnapshotID,
		SourceCutoff: r.Cutoff.UTC(), CompletedAt: r.CompletedAt.UTC(), Outcome: oapi.CommissionRunOutcome(r.Outcome), Reasons: oapi.Reasons{},
		Counts: r.Counts, Sources: []oapi.CommissionRunSource{}}
	if out.Counts == nil {
		out.Counts = map[string]int64{}
	}
	for _, c := range codes(r.Reasons) {
		out.Reasons = append(out.Reasons, oapi.ReasonCode(c))
	}
	for _, k := range sortedKeys(r.Sources) {
		out.Sources = append(out.Sources, oapi.CommissionRunSource{Source: k, Status: oapi.CommissionSourceStatus(r.Sources[k])})
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func isInvalid(err error) bool  { return errors.Is(err, query.ErrInvalid) }
func isNoRun(err error) bool    { return errors.Is(err, query.ErrNoRun) }
func isNotFound(err error) bool { return errors.Is(err, query.ErrNotFound) }

// GetCommissionOverview returns the overview of one stored run.
func (h *Handlers) GetCommissionOverview(ctx context.Context, req oapi.GetCommissionOverviewRequestObject) (oapi.GetCommissionOverviewResponseObject, error) {
	type R = oapi.GetCommissionOverviewResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionShadowRead, false, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.GetCommissionOverview200JSONResponse{Body: oapi.CommissionOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.GetCommissionOverview403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	ov, err := h.q.Overview(ctx, idOf(req.Params.RunId))
	decisions := []oapi.CommissionDecision{}
	for _, d := range api.Decisions() {
		decisions = append(decisions, oapi.CommissionDecision{Id: d.ID, State: oapi.CommissionDecisionState(d.State), Effect: d.Effect})
	}
	body := oapi.CommissionOverview{Decisions: decisions, Findings: map[string]int64{}, PostingState: oapi.BLOCKEDNOTAUTHORIZED}
	switch {
	case isInvalid(err):
		return oapi.GetCommissionOverview400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNotFound(err):
		return oapi.GetCommissionOverview404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.GetCommissionOverview200JSONResponse{Body: oapi.CommissionOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
			Overview: &body, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
	case err != nil:
		return nil, err
	}
	run := runOf(*ov.Run)
	rows, err := h.q.SourceRows(ctx, ov.Run.RunID)
	if err != nil {
		return nil, err
	}
	for i := range run.Sources {
		run.Sources[i].RowsRead = rows[run.Sources[i].Source]
	}
	body.Run = &run
	body.Findings = ov.Findings
	if p := ov.Parity; p != nil {
		body.Parity = &oapi.CommissionParitySummary{Cohort: p.Cohort, WindowFromRaw: p.WindowFrom, WindowToRaw: p.WindowTo, LegacyRows: p.LegacyRows,
			ComparedRows: p.ComparedRows, Coverage: oapi.CommissionCoverage(p.Coverage), Classes: p.Classes, Engines: p.Engines,
			LegacyTotals: p.LegacyTotals, TargetTotals: p.TargetTotals, LegacyAmountMissing: p.LegacyMissing, TargetAmountMissing: p.TargetMissing,
			Reasons: p.Reasons}
	}
	o, r := outcomeOf(ov.Run)
	return oapi.GetCommissionOverview200JSONResponse{Body: oapi.CommissionOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Overview: &body, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCommissionRuns lists the stored shadow runs.
func (h *Handlers) ListCommissionRuns(ctx context.Context, req oapi.ListCommissionRunsRequestObject) (oapi.ListCommissionRunsResponseObject, error) {
	type R = oapi.ListCommissionRunsResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionShadowRead, false, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.ListCommissionRuns200JSONResponse{Body: oapi.CommissionRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.ListCommissionRuns403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	runs, err := h.q.Runs(ctx, limitOf(req.Params.Limit))
	if err != nil {
		return nil, err
	}
	items := []oapi.CommissionRun{}
	for _, r := range runs {
		items = append(items, runOf(r))
	}
	o, r := authzhttp.Outcome(result.Success), []oapi.ReasonCode{}
	if len(runs) == 0 {
		o, r = outcomeOf(nil)
	}
	return oapi.ListCommissionRuns200JSONResponse{Body: oapi.CommissionRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCommissionCalculations lists stored calculations of one run.
func (h *Handlers) ListCommissionCalculations(ctx context.Context, req oapi.ListCommissionCalculationsRequestObject) (oapi.ListCommissionCalculationsResponseObject, error) {
	type R = oapi.ListCommissionCalculationsResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionShadowRead, false, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.ListCommissionCalculations200JSONResponse{Body: oapi.CommissionCalculationListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.ListCommissionCalculations403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	run, calcs, next, err := h.q.Calculations(ctx, idOf(p.RunId), strOf(p.ParityClass), strOf(p.Engine), query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListCommissionCalculations400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNotFound(err):
		return oapi.ListCommissionCalculations404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListCommissionCalculations200JSONResponse{Body: oapi.CommissionCalculationListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.CommissionCalculation{}
	for _, x := range calcs {
		fields := []oapi.CommissionParityField{}
		_ = json.Unmarshal(x.ParityFields, &fields)
		if x.Blocks == nil {
			x.Blocks = []string{}
		}
		if x.Reasons == nil {
			x.Reasons = []string{}
		}
		items = append(items, oapi.CommissionCalculation{CalcId: authzhttp.UUID(x.CalcID), SourceKey: x.SourceKey, EntitlementKey: x.EntitlementKey,
			Engine: oapi.CommissionEngine(x.Engine), FunctionId: x.FunctionID, PolicyKey: x.PolicyKey, PolicyVersion: x.PolicyVersion, EventRef: x.EventRef,
			InputSha256: x.InputSHA, ShareFacet: x.ShareFacet, RecipientState: oapi.CommissionCalculationRecipientState(x.RecipientState),
			RecipientKey: x.RecipientKey, RecipientUserId: authzhttp.OptUUID(x.RecipientUserID), Base: x.Base, RatePercent: x.Rate, Amount: x.Amount,
			LegacyAmount: x.LegacyAmount, Unit: oapi.CommissionCalculationUnit(x.Unit), Eligibility: oapi.CommissionCalculationEligibility(x.Eligibility),
			Coverage: oapi.CommissionCoverage(x.Coverage), Blocks: x.Blocks, ParityClass: oapi.CommissionParityClass(x.ParityClass), ParityFields: fields,
			Reasons: x.Reasons})
	}
	rr := runOf(run)
	o, r := outcomeOf(&run)
	return oapi.ListCommissionCalculations200JSONResponse{Body: oapi.CommissionCalculationListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Run: &rr, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCommissionFindings lists reconciliation findings of one run.
func (h *Handlers) ListCommissionFindings(ctx context.Context, req oapi.ListCommissionFindingsRequestObject) (oapi.ListCommissionFindingsResponseObject, error) {
	type R = oapi.ListCommissionFindingsResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionShadowRead, false, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.ListCommissionFindings200JSONResponse{Body: oapi.CommissionFindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.ListCommissionFindings403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	run, fs, next, err := h.q.Findings(ctx, idOf(p.RunId), strOf(p.Kind), query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListCommissionFindings400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNotFound(err):
		return oapi.ListCommissionFindings404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListCommissionFindings200JSONResponse{Body: oapi.CommissionFindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.CommissionFinding{}
	for _, f := range fs {
		items = append(items, oapi.CommissionFinding{FindingId: authzhttp.UUID(f.FindingID), Kind: f.Kind, Severity: oapi.CommissionFindingSeverity(f.Severity),
			SourceKey: f.SourceKey, EntitlementKey: f.EntKey, Detail: f.Detail})
	}
	rr := runOf(run)
	o, r := outcomeOf(&run)
	return oapi.ListCommissionFindings200JSONResponse{Body: oapi.CommissionFindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Run: &rr, Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCommissionPolicies lists observed policy versions.
func (h *Handlers) ListCommissionPolicies(ctx context.Context, req oapi.ListCommissionPoliciesRequestObject) (oapi.ListCommissionPoliciesResponseObject, error) {
	type R = oapi.ListCommissionPoliciesResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionPolicyRead, false, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.ListCommissionPolicies200JSONResponse{Body: oapi.CommissionPolicyListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.ListCommissionPolicies403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	ps, next, err := h.q.Policies(ctx, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListCommissionPolicies400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.CommissionPolicyVersion{}
	for _, v := range ps {
		items = append(items, oapi.CommissionPolicyVersion{PolicyKey: v.PolicyKey, Version: v.Version, Purpose: v.Purpose, ShareFacet: v.ShareFacet,
			FunctionId: v.FunctionID, RatePercent: v.Rate, Fixed: v.Fixed, Unit: oapi.CommissionPolicyVersionUnit(v.Unit), EffectiveFrom: v.EffectiveFrom,
			EffectiveTo: v.EffectiveTo, Evidence: oapi.CommissionPolicyVersionEvidence(v.Evidence),
			ApprovalState: oapi.CommissionPolicyVersionApprovalState(v.ApprovalState), Provenance: v.Provenance, SourceRef: v.SourceRef,
			FirstRunId: authzhttp.UUID(v.FirstRunID)})
	}
	o, r := authzhttp.Outcome(result.Success), []oapi.ReasonCode{}
	if len(ps) == 0 && p.Cursor == nil {
		o, r = outcomeOf(nil)
	}
	return oapi.ListCommissionPolicies200JSONResponse{Body: oapi.CommissionPolicyListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Items: &items, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCommissionHistoricalRuns lists the legacy dry runs of one run.
func (h *Handlers) ListCommissionHistoricalRuns(ctx context.Context, req oapi.ListCommissionHistoricalRunsRequestObject) (oapi.ListCommissionHistoricalRunsResponseObject, error) {
	type R = oapi.ListCommissionHistoricalRunsResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionShadowRead, false, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.ListCommissionHistoricalRuns200JSONResponse{Body: oapi.CommissionHistoricalRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.ListCommissionHistoricalRuns403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	run, hs, err := h.q.HistoricalRuns(ctx, idOf(req.Params.RunId))
	switch {
	case isInvalid(err):
		return oapi.ListCommissionHistoricalRuns400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNotFound(err):
		return oapi.ListCommissionHistoricalRuns404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListCommissionHistoricalRuns200JSONResponse{Body: oapi.CommissionHistoricalRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.CommissionHistoricalRun{}
	for _, x := range hs {
		items = append(items, oapi.CommissionHistoricalRun{SourceKey: x.SourceKey, RunCode: x.RunCode, WindowFromRaw: x.WindowFrom, WindowToRaw: x.WindowTo,
			ApprovalRaw: x.ApprovalRaw, State: oapi.CommissionHistoricalRunState(x.State), ApprovedByKey: x.ApprovedByKey, ApprovedAtRaw: x.ApprovedAtRaw,
			RejectedByKey: x.RejectedByKey, RejectedAtRaw: x.RejectedAtRaw, RejectedThenApproved: x.RejectedThenApproved, ItemCount: x.ItemCount,
			PostedItems: x.PostedItems, FailedItems: x.FailedItems})
	}
	rr := runOf(run)
	o, r := outcomeOf(&run)
	return oapi.ListCommissionHistoricalRuns200JSONResponse{Body: oapi.CommissionHistoricalRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx),
		Notice: Notice, Run: &rr, Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListMyCommissionEarnings lists the caller's own historical credits. The
// row filter is the authenticated user only; nothing broadens it.
func (h *Handlers) ListMyCommissionEarnings(ctx context.Context, req oapi.ListMyCommissionEarningsRequestObject) (oapi.ListMyCommissionEarningsResponseObject, error) {
	type R = oapi.ListMyCommissionEarningsResponseObject
	c, stop, err := guardOp(ctx, h, identityapi.CapCommissionOwnRead, true, gate[R]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) R {
			return oapi.ListMyCommissionEarnings200JSONResponse{Body: oapi.CommissionEarningListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}
		},
		denied: func(d oapi.DeniedJSONResponse) R {
			return oapi.ListMyCommissionEarnings403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	s := h.guard.Subject(ctx)
	if s.UserID == "" {
		// No resolved identity user: own scope cannot be proven (never "all").
		d := c.Decision
		code := reasoncodes.RowScopeDenied
		d.Allowed, d.Outcome, d.Reason = false, result.Unauthorized, &code
		return oapi.ListMyCommissionEarnings403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, d)}, nil
	}
	p := req.Params
	run, es, totals, next, err := h.q.Earnings(ctx, idOf(p.RunId), s.UserID, query.Page{Cursor: p.Cursor, Limit: limitOf(p.Limit)})
	switch {
	case isInvalid(err):
		return oapi.ListMyCommissionEarnings400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case isNotFound(err):
		return oapi.ListMyCommissionEarnings404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case isNoRun(err):
		o, r := outcomeOf(nil)
		return oapi.ListMyCommissionEarnings200JSONResponse{Body: oapi.CommissionEarningListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice}}, nil
	case err != nil:
		return nil, err
	}
	items := []oapi.CommissionEarning{}
	for _, e := range es {
		items = append(items, oapi.CommissionEarning{SourceKey: e.SourceKey, Engine: oapi.CommissionEngine(e.Engine), Direction: oapi.CommissionEarningDirection(e.Direction),
			TypeRaw: e.TypeRaw, InvoiceId: authzhttp.OptUUID(e.InvoiceID), StageRaw: e.Stage, PurposeRaw: e.Purpose, ShareFacetRaw: e.ShareFacet, Amount: e.Amount,
			Unit: oapi.CommissionEarningUnit(e.Unit), Reversed: e.Reversed, OccurredAtRaw: e.OccurredAtRaw, EntitlementKey: e.EntitlementKey})
	}
	rr := runOf(run)
	o, r := outcomeOf(&run)
	return oapi.ListMyCommissionEarnings200JSONResponse{Body: oapi.CommissionEarningListEnvelope{Outcome: o, Reasons: r, CorrelationId: corr(ctx), Notice: Notice,
		Run: &rr, Items: &items, Totals: &totals, Page: page(next), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
