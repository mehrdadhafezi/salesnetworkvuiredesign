package http

// Slice 14 per-service operations surface. Reads are row-scoped to the
// services the caller CURRENTLY supervises or executes (catalog scope),
// never every service; commands go through fulfillment/work, which
// re-checks authority, scope, state and version inside its transaction.
// No handler has SQL, imports anything, activates, dispatches or touches
// money.

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	catalogapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// WorkHandlers serves the work-case operations.
type WorkHandlers struct {
	read    *authorization.Guard // flag fulfillment.work_read
	write   *authorization.Guard // flag fulfillment.work_write
	mode    rollout.Source       // family fulfillment.work
	svc     *work.Service
	catalog catalogapi.Reader
}

// NewWork returns the handlers.
func NewWork(read, write *authorization.Guard, mode rollout.Source, svc *work.Service, catalog catalogapi.Reader) *WorkHandlers {
	return &WorkHandlers{read: read, write: write, mode: mode, svc: svc, catalog: catalog}
}

// OperationsEnabled reports the Slice 14 read flag (navigation hint).
func (h *WorkHandlers) OperationsEnabled(ctx context.Context) bool {
	on, err := h.read.Enabled(ctx)
	return err == nil && on
}

func uuids(in []string) []openapi_types.UUID {
	out := []openapi_types.UUID{}
	for _, s := range in {
		out = append(out, authzhttp.UUID(s))
	}
	return out
}

func (h *WorkHandlers) caseOf(ctx context.Context, c work.CaseView, userID string) (oapi.WorkCase, error) {
	out := oapi.WorkCase{CaseId: authzhttp.UUID(c.CaseID), ItemId: authzhttp.UUID(c.ItemID), SubscriptionId: authzhttp.UUID(c.SubscriptionID),
		CustomerId: authzhttp.UUID(c.CustomerID), ServiceKey: c.ServiceKey, ServiceVersion: c.ServiceVersion, WorkflowKey: c.WorkflowKey,
		OriginMode: oapi.WorkCaseOriginMode(c.Origin), State: oapi.WorkCaseState(c.State), AssigneeUserId: authzhttp.OptUUID(c.Assignee), Version: c.Version,
		CreatedAt: c.CreatedAt.UTC(), UpdatedAt: c.UpdatedAt.UTC(), CurrentSupervisors: []openapi_types.UUID{}, FinancialBehaviour: oapi.WorkCaseFinancialBehaviour("NONE"),
		IntegrationProvider: oapi.WorkCaseIntegrationProvider("NONE"), ServiceTitle: c.ServiceKey}
	sv, ok, err := h.catalog.Service(ctx, c.ServiceKey, c.ServiceVersion)
	if err != nil {
		return out, err
	}
	if ok {
		out.ServiceTitle = sv.Title
		out.FinancialBehaviour = oapi.WorkCaseFinancialBehaviour(sv.FinancialBehaviour)
		out.IntegrationProvider = oapi.WorkCaseIntegrationProvider(sv.IntegrationProvider)
		reqs := make([]struct {
			Required bool   `json:"required"`
			Type     string `json:"type"`
		}, 0, len(sv.Evidence))
		for _, e := range sv.Evidence {
			reqs = append(reqs, struct {
				Required bool   `json:"required"`
				Type     string `json:"type"`
			}{e.Required, e.Type})
		}
		out.EvidenceRequirements = &reqs
		outs := []oapi.WorkCaseCompletionOutcomes{}
		for _, o := range sv.CompletionOutcomes {
			outs = append(outs, oapi.WorkCaseCompletionOutcomes(o))
		}
		out.CompletionOutcomes = &outs
	}
	sup, err := h.catalog.Supervisors(ctx, c.ServiceKey)
	if err != nil {
		return out, err
	}
	out.CurrentSupervisors = uuids(sup)
	if userID != "" {
		s, err := h.catalog.InScope(ctx, c.ServiceKey, userID, catalogapi.ScopeSupervisor)
		if err != nil {
			return out, err
		}
		e, err := h.catalog.InScope(ctx, c.ServiceKey, userID, catalogapi.ScopeExecutor)
		if err != nil {
			return out, err
		}
		out.Scope = &oapi.WorkScope{Supervisor: s, Executor: e}
	}
	return out, nil
}

// ListFulfillmentWorkCases lists the caller's per-service queue.
func (h *WorkHandlers) ListFulfillmentWorkCases(ctx context.Context, req oapi.ListFulfillmentWorkCasesRequestObject) (oapi.ListFulfillmentWorkCasesResponseObject, error) {
	c, err := h.read.Check(ctx, identityapi.CapWorkRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListFulfillmentWorkCases200JSONResponse{Body: oapi.WorkCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListFulfillmentWorkCases403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	s := h.read.Subject(ctx)
	if s.UserID == "" {
		d := c.Decision
		code := reasoncodes.RowScopeDenied
		d.Allowed, d.Outcome, d.Reason = false, result.Unauthorized, &code
		return oapi.ListFulfillmentWorkCases403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, d)}, nil
	}
	services, err := h.catalog.ServicesInScope(ctx, s.UserID)
	if err != nil {
		return nil, err
	}
	p := req.Params
	f := work.Filter{Services: services, ServiceKey: p.ServiceKey, Cursor: p.Cursor}
	if p.State != nil {
		st := string(*p.State)
		f.State = &st
	}
	if p.SubscriptionId != nil {
		id := p.SubscriptionId.String()
		f.SubscriptionID = &id
	}
	if p.Limit != nil {
		f.Limit = int(*p.Limit)
	}
	rows, next, err := h.svc.Queue(ctx, f)
	if errors.Is(err, work.ErrInvalid) {
		return oapi.ListFulfillmentWorkCases400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.WorkCase{}
	for _, r := range rows {
		wc, err := h.caseOf(ctx, r, s.UserID)
		if err != nil {
			return nil, err
		}
		items = append(items, wc)
	}
	return oapi.ListFulfillmentWorkCases200JSONResponse{Body: oapi.WorkCaseListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx), Items: &items, Page: &oapi.PageInfo{HasMore: next != nil, NextCursor: next},
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetFulfillmentWorkCase returns one case of a service in the caller's scope.
func (h *WorkHandlers) GetFulfillmentWorkCase(ctx context.Context, req oapi.GetFulfillmentWorkCaseRequestObject) (oapi.GetFulfillmentWorkCaseResponseObject, error) {
	c, err := h.read.Check(ctx, identityapi.CapWorkRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetFulfillmentWorkCase200JSONResponse{Body: oapi.WorkCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetFulfillmentWorkCase403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	s := h.read.Subject(ctx)
	cv, evs, evid, err := h.svc.Detail(ctx, req.CaseId.String())
	switch {
	case errors.Is(err, work.ErrNotFound):
		return oapi.GetFulfillmentWorkCase404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	// Row scope: a case of a service outside the caller's scope is not found
	// (no existence oracle).
	services, err := h.catalog.ServicesInScope(ctx, s.UserID)
	if err != nil {
		return nil, err
	}
	in := false
	for _, k := range services {
		in = in || k == cv.ServiceKey
	}
	if !in {
		return oapi.GetFulfillmentWorkCase404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	wc, err := h.caseOf(ctx, cv, s.UserID)
	if err != nil {
		return nil, err
	}
	hist := []oapi.WorkEvent{}
	for _, e := range evs {
		d := e.Detail
		if d == nil {
			d = map[string]any{}
		}
		hist = append(hist, oapi.WorkEvent{EventId: authzhttp.UUID(e.EventID), CommandId: authzhttp.OptUUID(e.CommandID), Kind: oapi.WorkEventKind(e.Kind),
			Detail: d, ActorUserId: authzhttp.OptUUID(e.Actor), Supervisors: uuids(e.Supervisors), BeforeVersion: e.Before, AfterVersion: e.After,
			CorrelationId: e.Correlation, OccurredAt: e.OccurredAt.UTC()})
	}
	ev := []oapi.WorkEvidence{}
	for _, v := range evid {
		ev = append(ev, oapi.WorkEvidence{EvidenceId: authzhttp.UUID(v.EvidenceID), EvidenceType: v.Type, ContentSha256: v.SHA, Label: v.Label,
			SubmittedBy: authzhttp.UUID(v.SubmittedBy), OccurredAt: v.OccurredAt.UTC()})
	}
	blocked := []oapi.BlockedCommand{}
	for _, name := range []string{"fulfillment.cancel_case", "fulfillment.reopen_case", "fulfillment.submit_review", "fulfillment.reassign_started",
		"biawin.wallet_charge", "biawin.activate_production"} {
		blocked = append(blocked, oapi.BlockedCommand{Command: name, BlockedBy: work.BlockedCommands()[name]})
	}
	return oapi.GetFulfillmentWorkCase200JSONResponse{Body: oapi.WorkCaseEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx), Case: &wc, History: &hist, Evidence: &ev, BlockedCommands: &blocked,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func workResult(r work.Result) *oapi.WorkCommandResult {
	out := &oapi.WorkCommandResult{RolloutMode: oapi.RolloutMode(r.Mode), Applied: r.Applied, BlockedBy: r.BlockedBy, Replayed: r.Replayed}
	if out.RolloutMode == "" {
		out.RolloutMode = oapi.RolloutMode(rollout.Disabled)
	}
	if out.BlockedBy == nil {
		out.BlockedBy = []string{}
	}
	if r.CommandID != "" {
		out.CommandId = authzhttp.OptUUID(&r.CommandID)
	}
	if r.CaseVersion > 0 {
		v := r.CaseVersion
		out.Version = &v
	}
	if r.Current > 0 {
		v := r.Current
		out.CurrentVersion = &v
	}
	if r.State != "" {
		st := r.State
		out.State = &st
	}
	return out
}

// outcomeKind is what a command handler answers.
type outcomeKind int

const (
	kindOK outcomeKind = iota
	kindDenied
	kindConflict
	kindBadRequest
	kindNotFound
)

type cmdOut struct {
	kind   outcomeKind
	env    oapi.WorkCommandEnvelope
	denied oapi.DeniedJSONResponse
	bad    oapi.ProblemApplicationProblemPlusJSONResponse
	nf     oapi.Problem
}

// run is the common command path: flag + capability, If-Match, rollout
// mode, the service, and the outcome mapping.
func (h *WorkHandlers) run(ctx context.Context, cap identityapi.Capability, ifMatch string, build func(work.Actor, int64) work.Command) (cmdOut, error) {
	c, err := h.write.Check(ctx, cap)
	if err != nil {
		return cmdOut{}, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		return cmdOut{kind: kindOK, env: oapi.WorkCommandEnvelope{Outcome: authzhttp.Outcome(result.Blocked), Reasons: authzhttp.Reasons(reasoncodes.FeatureDisabled),
			CorrelationId: httpx.CorrelationID(ctx), Command: workResult(work.Result{Mode: rollout.Disabled})}}, nil
	case authorization.GateDenied:
		return cmdOut{kind: kindDenied, denied: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	version, ok := authzhttp.IfMatchVersion(ifMatch)
	if !ok {
		return cmdOut{kind: kindBadRequest, bad: authzhttp.BadRequest(ctx, "If-Match must carry the case version")}, nil
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return cmdOut{}, err
	}
	s := h.write.Subject(ctx)
	r, err := h.svc.Execute(ctx, mode, build(work.Actor{Principal: s.PrincipalID, UserID: s.UserID}, version))
	switch {
	case errors.Is(err, work.ErrInvalid):
		return cmdOut{kind: kindBadRequest, bad: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, work.ErrNotFound):
		return cmdOut{kind: kindNotFound, nf: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return cmdOut{}, err
	}
	reasons := []oapi.ReasonCode{}
	for _, code := range r.Reasons {
		reasons = append(reasons, oapi.ReasonCode(code))
	}
	env := oapi.WorkCommandEnvelope{Outcome: authzhttp.Outcome(r.Status), Reasons: reasons, CorrelationId: httpx.CorrelationID(ctx), Command: workResult(r)}
	switch r.Status {
	case result.Unauthorized:
		return cmdOut{kind: kindDenied, denied: oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: env.Outcome, Reasons: env.Reasons,
			CorrelationId: env.CorrelationId}}}, nil
	case result.Conflict:
		return cmdOut{kind: kindConflict, env: env}, nil
	}
	return cmdOut{kind: kindOK, env: env}, nil
}

// AssignFulfillmentWorkCase assigns a case.
func (h *WorkHandlers) AssignFulfillmentWorkCase(ctx context.Context, req oapi.AssignFulfillmentWorkCaseRequestObject) (oapi.AssignFulfillmentWorkCaseResponseObject, error) {
	if req.Body == nil {
		return oapi.AssignFulfillmentWorkCase400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	o, err := h.run(ctx, identityapi.CapWorkAssign, req.Params.IfMatch, func(a work.Actor, v int64) work.Command {
		return work.Command{Type: work.TypeAssign, CaseID: req.CaseId.String(), AssigneeUserID: req.Body.AssigneeUserId.String(), ExpectedVersion: v,
			IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx), Actor: a}
	})
	if err != nil {
		return nil, err
	}
	switch o.kind {
	case kindDenied:
		return oapi.AssignFulfillmentWorkCase403JSONResponse{DeniedJSONResponse: o.denied}, nil
	case kindConflict:
		return oapi.AssignFulfillmentWorkCase409JSONResponse{Body: o.env}, nil
	case kindBadRequest:
		return oapi.AssignFulfillmentWorkCase400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: o.bad}, nil
	case kindNotFound:
		return oapi.AssignFulfillmentWorkCase404ApplicationProblemPlusJSONResponse{Body: o.nf}, nil
	}
	return oapi.AssignFulfillmentWorkCase200JSONResponse{Body: o.env}, nil
}

// RecordFulfillmentWorkAction records an operational action.
func (h *WorkHandlers) RecordFulfillmentWorkAction(ctx context.Context, req oapi.RecordFulfillmentWorkActionRequestObject) (oapi.RecordFulfillmentWorkActionResponseObject, error) {
	if req.Body == nil {
		return oapi.RecordFulfillmentWorkAction400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	o, err := h.run(ctx, identityapi.CapWorkExecute, req.Params.IfMatch, func(a work.Actor, v int64) work.Command {
		return work.Command{Type: work.TypeAction, CaseID: req.CaseId.String(), Action: string(req.Body.Action), FollowUpAt: req.Body.FollowUpAt,
			Note: req.Body.Note, ExpectedVersion: v, IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx), Actor: a}
	})
	if err != nil {
		return nil, err
	}
	switch o.kind {
	case kindDenied:
		return oapi.RecordFulfillmentWorkAction403JSONResponse{DeniedJSONResponse: o.denied}, nil
	case kindConflict:
		return oapi.RecordFulfillmentWorkAction409JSONResponse{Body: o.env}, nil
	case kindBadRequest:
		return oapi.RecordFulfillmentWorkAction400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: o.bad}, nil
	case kindNotFound:
		return oapi.RecordFulfillmentWorkAction404ApplicationProblemPlusJSONResponse{Body: o.nf}, nil
	}
	return oapi.RecordFulfillmentWorkAction200JSONResponse{Body: o.env}, nil
}

// AttachFulfillmentWorkEvidence attaches evidence metadata.
func (h *WorkHandlers) AttachFulfillmentWorkEvidence(ctx context.Context, req oapi.AttachFulfillmentWorkEvidenceRequestObject) (oapi.AttachFulfillmentWorkEvidenceResponseObject, error) {
	if req.Body == nil {
		return oapi.AttachFulfillmentWorkEvidence400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	o, err := h.run(ctx, identityapi.CapWorkExecute, req.Params.IfMatch, func(a work.Actor, v int64) work.Command {
		return work.Command{Type: work.TypeEvidence, CaseID: req.CaseId.String(), EvidenceType: req.Body.EvidenceType, ContentSHA: req.Body.ContentSha256,
			Label: req.Body.Label, ExpectedVersion: v, IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx), Actor: a}
	})
	if err != nil {
		return nil, err
	}
	switch o.kind {
	case kindDenied:
		return oapi.AttachFulfillmentWorkEvidence403JSONResponse{DeniedJSONResponse: o.denied}, nil
	case kindConflict:
		return oapi.AttachFulfillmentWorkEvidence409JSONResponse{Body: o.env}, nil
	case kindBadRequest:
		return oapi.AttachFulfillmentWorkEvidence400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: o.bad}, nil
	case kindNotFound:
		return oapi.AttachFulfillmentWorkEvidence404ApplicationProblemPlusJSONResponse{Body: o.nf}, nil
	}
	return oapi.AttachFulfillmentWorkEvidence200JSONResponse{Body: o.env}, nil
}

// CompleteFulfillmentWorkCase records the outcome of one service.
func (h *WorkHandlers) CompleteFulfillmentWorkCase(ctx context.Context, req oapi.CompleteFulfillmentWorkCaseRequestObject) (oapi.CompleteFulfillmentWorkCaseResponseObject, error) {
	if req.Body == nil {
		return oapi.CompleteFulfillmentWorkCase400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	o, err := h.run(ctx, identityapi.CapWorkExecute, req.Params.IfMatch, func(a work.Actor, v int64) work.Command {
		return work.Command{Type: work.TypeComplete, CaseID: req.CaseId.String(), Outcome: string(req.Body.Outcome), Reason: req.Body.Reason, ExpectedVersion: v,
			IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx), Actor: a}
	})
	if err != nil {
		return nil, err
	}
	switch o.kind {
	case kindDenied:
		return oapi.CompleteFulfillmentWorkCase403JSONResponse{DeniedJSONResponse: o.denied}, nil
	case kindConflict:
		return oapi.CompleteFulfillmentWorkCase409JSONResponse{Body: o.env}, nil
	case kindBadRequest:
		return oapi.CompleteFulfillmentWorkCase400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: o.bad}, nil
	case kindNotFound:
		return oapi.CompleteFulfillmentWorkCase404ApplicationProblemPlusJSONResponse{Body: o.nf}, nil
	}
	return oapi.CompleteFulfillmentWorkCase200JSONResponse{Body: o.env}, nil
}
