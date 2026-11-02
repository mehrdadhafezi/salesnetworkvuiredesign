package http

import (
	"context"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/command"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// WriteHandlers serves the Slice 11 contact outcome read and command. The
// command never runs SQL here and never calls a connector: it delegates to
// the cases command service; the legacy write-back is integration's.
type WriteHandlers struct {
	read  *authorization.Guard // flag sales.read_shadow
	write *authorization.Guard // flag cases.nonfinancial_write
	mode  rollout.Source
	svc   *command.Service
}

// NewWrite returns the handlers.
func NewWrite(read, write *authorization.Guard, mode rollout.Source, svc *command.Service) *WriteHandlers {
	return &WriteHandlers{read: read, write: write, mode: mode, svc: svc}
}

// ContactOutcomeWriteEnabled reports the Slice 11 flag (navigation hint).
func (h *WriteHandlers) ContactOutcomeWriteEnabled(ctx context.Context) bool {
	on, err := h.write.Enabled(ctx)
	return err == nil && on
}

// blockedCaseActions are legacy actions that exist but are unavailable.
func blockedCaseActions() []oapi.BlockedAction {
	return []oapi.BlockedAction{
		{Action: "record_duplicate_outcome", BlockedBy: []string{command.DecisionD29, command.DecisionD31}},
		{Action: "record_pre_invoice_outcome", BlockedBy: []string{command.DecisionADR01}},
		{Action: "issue_invoice", BlockedBy: []string{command.DecisionADR01}},
		{Action: "return_case", BlockedBy: []string{command.DecisionADR01, "F01", "OPD-03"}},
		{Action: "transfer_case", BlockedBy: []string{"OPD-10"}},
		{Action: "allocate_case", BlockedBy: []string{"POOL_OWNER_EVIDENCE_MISSING"}},
		{Action: "mark_ready", BlockedBy: []string{command.PathUnproven}},
	}
}

// GetCaseContactOutcomes reads the contact outcome facts and computes the
// record action with the command's own evaluator.
func (h *WriteHandlers) GetCaseContactOutcomes(ctx context.Context, req oapi.GetCaseContactOutcomesRequestObject) (oapi.GetCaseContactOutcomesResponseObject, error) {
	c, err := h.read.Check(ctx, identityapi.CapCaseRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetCaseContactOutcomes200JSONResponse{Body: oapi.ContactOutcomeEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetCaseContactOutcomes403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	v, err := h.svc.ContactOutcomes(ctx, req.CaseId.String())
	if errors.Is(err, command.ErrNotFound) {
		return oapi.GetCaseContactOutcomes404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.ContactOutcomeView{CaseId: req.CaseId, CaseVersion: v.Version, WorkflowType: v.Workflow, RolloutMode: oapi.RolloutMode(mode),
		History: []oapi.ContactOutcomeEvent{}, BlockedActions: blockedCaseActions()}
	if cur := v.Current; cur != nil {
		data.Current = &oapi.ContactOutcomeCurrent{Outcome: oapi.ContactOutcomeCurrentOutcome(cur.Outcome), Reason: cur.Reason, CallbackAt: cur.CallbackAt,
			LastCommandId: authzhttp.UUID(cur.LastCommandID), RecordedAt: cur.RecordedAt.UTC()}
	}
	for _, e := range v.History {
		data.History = append(data.History, oapi.ContactOutcomeEvent{EventId: authzhttp.UUID(e.EventID), CommandId: authzhttp.UUID(e.CommandID),
			Outcome: e.Outcome, Reason: e.Reason, CallbackAt: e.CallbackAt, PreviousOutcome: e.PreviousOutcome, BeforeVersion: e.BeforeVersion,
			AfterVersion: e.AfterVersion, ActorUserId: authzhttp.UUID(e.ActorUserID), CorrelationId: e.CorrelationID, OccurredAt: e.OccurredAt.UTC()})
	}
	// allowed_actions: the same evaluator as the command (flag + capability
	// + holder row scope). Disabled or denied → not allowed, with the reason.
	action := oapi.AllowedAction{Allowed: false}
	wc, err := h.write.Check(ctx, identityapi.CapContactOutcomeRecord)
	if err != nil {
		return nil, err
	}
	code := reasoncodes.FeatureDisabled
	if wc.Gate == authorization.GateDenied {
		code = reasoncodes.ActionDenied
	}
	if wc.Gate == authorization.GateAllowed && mode != rollout.Disabled {
		s := h.write.Subject(ctx)
		ok, why, err := h.svc.Eligible(ctx, req.CaseId.String(), command.Actor{Principal: s.PrincipalID, UserID: s.UserID})
		if err != nil {
			return nil, err
		}
		action.Allowed, code = ok, why
	}
	if !action.Allowed {
		r := oapi.ReasonCode(code)
		action.ReasonCode = &r
	}
	allowed := oapi.AllowedActions{string(identityapi.CapContactOutcomeRecord): action}
	return oapi.GetCaseContactOutcomes200JSONResponse{Body: oapi.ContactOutcomeEnvelope{Outcome: authzhttp.Outcome(result.Success),
		Reasons: authzhttp.Reasons(), CorrelationId: httpx.CorrelationID(ctx), Data: &data, AllowedActions: &allowed}}, nil
}

func commandResult(r command.Result) *oapi.CommandResult {
	cr := &oapi.CommandResult{RolloutMode: oapi.RolloutMode(r.Mode), Applied: r.Applied, WritebackState: oapi.CommandWritebackState(r.WritebackState),
		BlockedBy: r.BlockedBy}
	if cr.RolloutMode == "" {
		cr.RolloutMode = oapi.RolloutMode(rollout.Disabled)
	}
	if cr.BlockedBy == nil {
		cr.BlockedBy = []string{}
	}
	cr.CommandId = authzhttp.OptUUID(nonEmpty(r.CommandID))
	if r.CaseVersion > 0 {
		v := r.CaseVersion
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

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// RecordContactOutcome runs the command for the caller.
func (h *WriteHandlers) RecordContactOutcome(ctx context.Context, req oapi.RecordContactOutcomeRequestObject) (oapi.RecordContactOutcomeResponseObject, error) {
	c, err := h.write.Check(ctx, identityapi.CapContactOutcomeRecord)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		return oapi.RecordContactOutcome200JSONResponse{Body: authzhttp.CommandEnvelope(ctx, result.Blocked, []reasoncodes.Code{reasoncodes.FeatureDisabled},
			commandResult(command.Result{Mode: rollout.Disabled, WritebackState: command.WritebackNotRequested}))}, nil
	case authorization.GateDenied:
		return oapi.RecordContactOutcome403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	if req.Body == nil {
		return oapi.RecordContactOutcome400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	version, ok := authzhttp.IfMatchVersion(req.Params.IfMatch)
	if !ok {
		return oapi.RecordContactOutcome400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "If-Match must carry the case version")}, nil
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	s := h.write.Subject(ctx)
	r, err := h.svc.RecordContactOutcome(ctx, mode, command.Command{CaseID: req.CaseId.String(), Outcome: string(req.Body.Outcome), Reason: req.Body.Reason,
		CallbackAt: req.Body.CallbackAt, ExpectedVersion: version, IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx),
		Actor: command.Actor{Principal: s.PrincipalID, UserID: s.UserID}})
	switch {
	case errors.Is(err, command.ErrInvalid):
		return oapi.RecordContactOutcome400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, command.ErrNotFound):
		return oapi.RecordContactOutcome404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	env := authzhttp.CommandEnvelope(ctx, r.Status, r.Reasons, commandResult(r))
	switch r.Status {
	case result.Unauthorized:
		return oapi.RecordContactOutcome403JSONResponse{DeniedJSONResponse: oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: env.Outcome,
			Reasons: env.Reasons, CorrelationId: env.CorrelationId}}}, nil
	case result.Conflict:
		return oapi.RecordContactOutcome409JSONResponse{Body: env}, nil
	}
	return oapi.RecordContactOutcome200JSONResponse{Body: env}, nil
}
