package http

import (
	"context"
	"errors"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/hrrequest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// HRRequestHandlers serves the Slice 11 HR change request reads and
// commands. No SQL here; the command service re-checks everything at commit.
type HRRequestHandlers struct {
	guard *authorization.Guard // flag organization.hr_request_write
	mode  rollout.Source
	svc   *hrrequest.Service
}

// NewHRRequests returns the handlers.
func NewHRRequests(guard *authorization.Guard, mode rollout.Source, svc *hrrequest.Service) *HRRequestHandlers {
	return &HRRequestHandlers{guard: guard, mode: mode, svc: svc}
}

// HRRequestWriteEnabled reports the Slice 11 flag (navigation hint).
func (h *HRRequestHandlers) HRRequestWriteEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func requestOf(q hrrequest.Request, c authorization.Check, actorUserID string) oapi.HRRequest {
	out := oapi.HRRequest{RequestId: authzhttp.UUID(q.RequestID), RequestType: oapi.HRRequestType(q.RequestType), Status: oapi.HRRequestStatus(q.Status),
		SubjectUserId: authzhttp.UUID(q.SubjectUserID), SubjectParentUserId: authzhttp.OptUUID(q.SubjectParentUserID),
		TargetParentUserId: authzhttp.OptUUID(q.TargetParentUserID), RequestedByUserId: authzhttp.UUID(q.RequestedByUserID),
		CurrentReviewerUserId: authzhttp.OptUUID(q.CurrentReviewerUserID), Reason: q.Reason, ReviewerNotes: q.ReviewerNotes, Version: q.Version,
		CreatedAt: q.CreatedAt.UTC(), UpdatedAt: q.UpdatedAt.UTC(), BlockedActions: []oapi.BlockedAction{}}
	blocked := hrrequest.Blocked(q)
	keys := make([]string, 0, len(blocked))
	for k := range blocked {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out.BlockedActions = append(out.BlockedActions, oapi.BlockedAction{Action: k, BlockedBy: blocked[k]})
	}
	// Step decisions: capability (same evaluator) AND the request's own
	// current-reviewer rule; a pending_hr request offers no decision.
	decide := oapi.AllowedAction{Allowed: c.Can(identityapi.CapHRRequestReview) && hrrequest.CanDecide(q, actorUserID)}
	if !decide.Allowed {
		code := oapi.ReasonCode(reasoncodes.RowScopeDenied)
		if !c.Can(identityapi.CapHRRequestReview) {
			code = oapi.ReasonCode(reasoncodes.ActionDenied)
		} else if q.Status == "pending_hr" {
			code = oapi.ReasonCode(reasoncodes.PolicyUnresolved)
		}
		decide.ReasonCode = &code
	}
	actions := oapi.AllowedActions{"approve_step": decide, "reject": decide}
	out.AllowedActions = &actions
	return out
}

func (h *HRRequestHandlers) readGate(ctx context.Context) (authorization.Check, *oapi.Outcome, []oapi.ReasonCode, error) {
	c, err := h.guard.Check(ctx, identityapi.CapHRRequestRead)
	if err != nil {
		return c, nil, nil, err
	}
	if c.Gate == authorization.GateDisabled {
		o, r := authzhttp.Disabled()
		return c, &o, r, nil
	}
	return c, nil, nil, nil
}

// ListHRRequests lists the caller's requests (requester / reviewer / actor).
func (h *HRRequestHandlers) ListHRRequests(ctx context.Context, req oapi.ListHRRequestsRequestObject) (oapi.ListHRRequestsResponseObject, error) {
	c, o, r, err := h.readGate(ctx)
	if err != nil {
		return nil, err
	}
	if o != nil {
		return oapi.ListHRRequests200JSONResponse{Body: oapi.HRRequestListEnvelope{Outcome: *o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	}
	if c.Gate == authorization.GateDenied {
		return oapi.ListHRRequests403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	limit := 50
	if req.Params.Limit != nil {
		limit = int(*req.Params.Limit)
	}
	s := h.guard.Subject(ctx)
	rows, next, err := h.svc.List(ctx, s.UserID, req.Params.Cursor, limit)
	if errors.Is(err, hrrequest.ErrInvalid) {
		return oapi.ListHRRequests400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	items := []oapi.HRRequest{}
	for _, q := range rows {
		items = append(items, requestOf(q, c, s.UserID))
	}
	rm := oapi.RolloutMode(mode)
	create := oapi.AllowedAction{Allowed: c.Can(identityapi.CapHRRequestCreate) && mode != rollout.Disabled && s.UserID != ""}
	if !create.Allowed {
		code := oapi.ReasonCode(reasoncodes.ActionDenied)
		create.ReasonCode = &code
	}
	actions := oapi.AllowedActions{string(identityapi.CapHRRequestCreate): create}
	return oapi.ListHRRequests200JSONResponse{Body: oapi.HRRequestListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), RolloutMode: &rm, Items: &items, Page: &oapi.PageInfo{HasMore: next != nil, NextCursor: next},
		AllowedActions: &actions}}, nil
}

// GetHRRequest returns one visible request with its history.
func (h *HRRequestHandlers) GetHRRequest(ctx context.Context, req oapi.GetHRRequestRequestObject) (oapi.GetHRRequestResponseObject, error) {
	c, o, r, err := h.readGate(ctx)
	if err != nil {
		return nil, err
	}
	if o != nil {
		return oapi.GetHRRequest200JSONResponse{Body: oapi.HRRequestEnvelope{Outcome: *o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	}
	if c.Gate == authorization.GateDenied {
		return oapi.GetHRRequest403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	s := h.guard.Subject(ctx)
	q, hist, err := h.svc.Get(ctx, s.UserID, req.RequestId.String())
	switch {
	case errors.Is(err, hrrequest.ErrNotFound):
		return oapi.GetHRRequest404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case errors.Is(err, hrrequest.ErrInvalid):
		return oapi.GetHRRequest400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case err != nil:
		return nil, err
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	rq := requestOf(q, c, s.UserID)
	events := []oapi.HRRequestEvent{}
	for _, e := range hist {
		events = append(events, oapi.HRRequestEvent{EventId: authzhttp.UUID(e.EventID), CommandId: authzhttp.UUID(e.CommandID),
			Action: oapi.HRRequestEventAction(e.Action), FromStatus: e.FromStatus, ToStatus: e.ToStatus, NextReviewerUserId: authzhttp.OptUUID(e.NextReviewerUserID),
			Notes: e.Notes, AfterVersion: e.AfterVersion, ActorUserId: authzhttp.UUID(e.ActorUserID), CorrelationId: e.CorrelationID, OccurredAt: e.OccurredAt.UTC()})
	}
	rm := oapi.RolloutMode(mode)
	return oapi.GetHRRequest200JSONResponse{Body: oapi.HRRequestEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), RolloutMode: &rm, Request: &rq, History: &events}}, nil
}

func hrResult(r hrrequest.Result) *oapi.CommandResult {
	cr := &oapi.CommandResult{RolloutMode: oapi.RolloutMode(r.Mode), Applied: r.Applied, WritebackState: oapi.CommandWritebackState(r.WritebackState),
		BlockedBy: r.BlockedBy, CommandId: authzhttp.OptUUID(opt(r.CommandID)), RequestId: authzhttp.OptUUID(opt(r.RequestID)), RequestStatus: opt(r.RequestStatus)}
	if cr.RolloutMode == "" {
		cr.RolloutMode = oapi.RolloutMode(rollout.Disabled)
	}
	if cr.BlockedBy == nil {
		cr.BlockedBy = []string{}
	}
	if r.Version > 0 {
		v := r.Version
		cr.Version = &v
	}
	if r.CurrentVersion > 0 {
		v := r.CurrentVersion
		cr.CurrentVersion = &v
	}
	if r.Replayed {
		t := true
		cr.Replayed = &t
	}
	return cr
}

func opt(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *HRRequestHandlers) commandGate(ctx context.Context, cp identityapi.Capability) (authorization.Check, rollout.Mode, error) {
	c, err := h.guard.Check(ctx, cp)
	if err != nil {
		return c, rollout.Disabled, err
	}
	mode, err := h.mode(ctx)
	return c, mode, err
}

// CreateHRRequest runs CreateHRRequest for the caller.
func (h *HRRequestHandlers) CreateHRRequest(ctx context.Context, req oapi.CreateHRRequestRequestObject) (oapi.CreateHRRequestResponseObject, error) {
	c, mode, err := h.commandGate(ctx, identityapi.CapHRRequestCreate)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		return oapi.CreateHRRequest200JSONResponse{Body: authzhttp.CommandEnvelope(ctx, result.Blocked, []reasoncodes.Code{reasoncodes.FeatureDisabled},
			hrResult(hrrequest.Result{Mode: rollout.Disabled, WritebackState: hrrequest.WritebackNotRequested}))}, nil
	case authorization.GateDenied:
		return oapi.CreateHRRequest403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	if req.Body == nil {
		return oapi.CreateHRRequest400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	var target *string
	if req.Body.TargetParentUserId != nil {
		t := req.Body.TargetParentUserId.String()
		target = &t
	}
	s := h.guard.Subject(ctx)
	r, err := h.svc.CreateRequest(ctx, mode, hrrequest.Create{RequestType: string(req.Body.RequestType), SubjectUserID: req.Body.SubjectUserId.String(),
		TargetParentUserID: target, Reason: req.Body.Reason, IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx),
		Actor: hrrequest.Actor{Principal: s.PrincipalID, UserID: s.UserID}})
	if errors.Is(err, hrrequest.ErrInvalid) {
		return oapi.CreateHRRequest400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	env := authzhttp.CommandEnvelope(ctx, r.Status, r.Reasons, hrResult(r))
	switch r.Status {
	case result.Unauthorized:
		return oapi.CreateHRRequest403JSONResponse{DeniedJSONResponse: oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: env.Outcome,
			Reasons: env.Reasons, CorrelationId: env.CorrelationId}}}, nil
	case result.Conflict:
		return oapi.CreateHRRequest409JSONResponse{Body: env}, nil
	}
	return oapi.CreateHRRequest200JSONResponse{Body: env}, nil
}

// DecideHRRequest runs DecideHRRequest for the caller.
func (h *HRRequestHandlers) DecideHRRequest(ctx context.Context, req oapi.DecideHRRequestRequestObject) (oapi.DecideHRRequestResponseObject, error) {
	c, mode, err := h.commandGate(ctx, identityapi.CapHRRequestReview)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		return oapi.DecideHRRequest200JSONResponse{Body: authzhttp.CommandEnvelope(ctx, result.Blocked, []reasoncodes.Code{reasoncodes.FeatureDisabled},
			hrResult(hrrequest.Result{Mode: rollout.Disabled, WritebackState: hrrequest.WritebackNotRequested}))}, nil
	case authorization.GateDenied:
		return oapi.DecideHRRequest403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	if req.Body == nil {
		return oapi.DecideHRRequest400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	version, ok := authzhttp.IfMatchVersion(req.Params.IfMatch)
	if !ok {
		return oapi.DecideHRRequest400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "If-Match must carry the request version")}, nil
	}
	s := h.guard.Subject(ctx)
	r, err := h.svc.DecideRequest(ctx, mode, hrrequest.Decide{RequestID: req.RequestId.String(), Decision: string(req.Body.Decision), Notes: req.Body.Notes,
		ExpectedVersion: version, IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx),
		Actor: hrrequest.Actor{Principal: s.PrincipalID, UserID: s.UserID}})
	switch {
	case errors.Is(err, hrrequest.ErrInvalid):
		return oapi.DecideHRRequest400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, hrrequest.ErrNotFound):
		return oapi.DecideHRRequest404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	env := authzhttp.CommandEnvelope(ctx, r.Status, r.Reasons, hrResult(r))
	switch r.Status {
	case result.Unauthorized:
		return oapi.DecideHRRequest403JSONResponse{DeniedJSONResponse: oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: env.Outcome,
			Reasons: env.Reasons, CorrelationId: env.CorrelationId}}}, nil
	case result.Conflict:
		return oapi.DecideHRRequest409JSONResponse{Body: env}, nil
	}
	return oapi.DecideHRRequest200JSONResponse{Body: env}, nil
}
