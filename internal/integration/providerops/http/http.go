// Package http adapts the provider operation ledger to the OpenAPI server:
// a GET list (UNKNOWN is never shown as delivered; connector is always
// NOT_CONFIGURED) and the human reconciliation command. Nothing here
// dispatches to a provider or knows a provider: no port is reachable from
// the API server.
package http

import (
	"context"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the provider operation surface.
type Handlers struct {
	read      *authorization.Guard // flag fulfillment.work_read
	reconcile *authorization.Guard // flag integration.provider_reconcile
	svc       *providerops.Service
}

// New returns the handlers.
func New(read, reconcile *authorization.Guard, svc *providerops.Service) *Handlers {
	return &Handlers{read: read, reconcile: reconcile, svc: svc}
}

func opOf(o providerops.Operation) oapi.ProviderOperation {
	return oapi.ProviderOperation{OperationId: authzhttp.UUID(o.OperationID), Provider: oapi.ProviderOperationProvider(o.Provider),
		OpType: oapi.ProviderOperationOpType(o.OpType), SubjectCaseId: authzhttp.OptUUID(o.SubjectCaseID), Reference: o.Reference,
		State: oapi.ProviderOperationState(o.State), BlockedBy: o.BlockedBy, Attempts: o.Attempts, ExternalRef: o.ExternalRef, ReconciliationOwner: o.Owner,
		Version: o.Version, CorrelationId: o.CorrelationID, UpdatedAt: o.UpdatedAt.UTC()}
}

// ListProviderOperations lists external operations.
func (h *Handlers) ListProviderOperations(ctx context.Context, req oapi.ListProviderOperationsRequestObject) (oapi.ListProviderOperationsResponseObject, error) {
	c, err := h.read.Check(ctx, identityapi.CapProviderOperationRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListProviderOperations200JSONResponse{Body: oapi.ProviderOperationListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListProviderOperations403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	var state *string
	if req.Params.State != nil {
		s := string(*req.Params.State)
		state = &s
	}
	limit := 0
	if req.Params.Limit != nil {
		limit = int(*req.Params.Limit)
	}
	ops, err := h.svc.List(ctx, state, limit)
	if err != nil {
		return nil, err
	}
	items := []oapi.ProviderOperation{}
	for _, o := range ops {
		items = append(items, opOf(o))
	}
	nc := oapi.NOTCONFIGURED
	return oapi.ListProviderOperations200JSONResponse{Body: oapi.ProviderOperationListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx), Connector: &nc, Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ResolveProviderOperation records human reconciliation evidence.
func (h *Handlers) ResolveProviderOperation(ctx context.Context, req oapi.ResolveProviderOperationRequestObject) (oapi.ResolveProviderOperationResponseObject, error) {
	c, err := h.reconcile.Check(ctx, identityapi.CapProviderOperationRecon)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ResolveProviderOperation200JSONResponse{Body: oapi.ProviderOperationEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ResolveProviderOperation403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	if req.Body == nil || len(req.Params.IdempotencyKey) < 8 {
		return oapi.ResolveProviderOperation400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body and Idempotency-Key required")}, nil
	}
	version, ok := authzhttp.IfMatchVersion(req.Params.IfMatch)
	if !ok {
		return oapi.ResolveProviderOperation400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "If-Match must carry the operation version")}, nil
	}
	s := h.reconcile.Subject(ctx)
	if s.UserID == "" {
		d := c.Decision
		code := reasoncodes.RowScopeDenied
		d.Allowed, d.Outcome, d.Reason = false, result.Unauthorized, &code
		return oapi.ResolveProviderOperation403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, d)}, nil
	}
	ext := ""
	if req.Body.ExternalRef != nil {
		ext = *req.Body.ExternalRef
	}
	op, err := h.svc.Resolve(ctx, providerops.Resolution{OperationID: req.OperationId.String(), Present: req.Body.Present, ExternalRef: ext,
		EvidenceNote: req.Body.EvidenceNote, ExpectedVersion: version, ActorUserID: s.UserID})
	switch {
	case errors.Is(err, providerops.ErrInvalid):
		return oapi.ResolveProviderOperation400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, providerops.ErrNotFound):
		return oapi.ResolveProviderOperation404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case errors.Is(err, providerops.ErrState):
		return oapi.ResolveProviderOperation409JSONResponse{Body: oapi.ProviderOperationEnvelope{Outcome: authzhttp.Outcome(result.Conflict),
			Reasons: authzhttp.Reasons(reasoncodes.StaleState), CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case err != nil:
		return nil, err
	}
	o := opOf(op)
	return oapi.ResolveProviderOperation200JSONResponse{Body: oapi.ProviderOperationEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx), Operation: &o}}, nil
}
