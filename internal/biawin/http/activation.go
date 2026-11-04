package http

// Slice 14 SHADOW activation decisions (GET only). D-35: only SHADOW
// decisions exist; this read never activates, decides or records anything.

import (
	"context"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/activation"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// ActivationHandlers serves the activation decision read.
type ActivationHandlers struct {
	guard *authorization.Guard
	svc   *activation.Service
}

// NewActivation returns the handlers.
func NewActivation(guard *authorization.Guard, svc *activation.Service) *ActivationHandlers {
	return &ActivationHandlers{guard: guard, svc: svc}
}

// ListBiawinActivationDecisions lists SHADOW activation decisions.
func (h *ActivationHandlers) ListBiawinActivationDecisions(ctx context.Context, req oapi.ListBiawinActivationDecisionsRequestObject) (oapi.ListBiawinActivationDecisionsResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapActivationRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListBiawinActivationDecisions200JSONResponse{Body: oapi.ActivationListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListBiawinActivationDecisions403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	var decision *string
	if req.Params.Decision != nil {
		d := string(*req.Params.Decision)
		decision = &d
	}
	limit := 0
	if req.Params.Limit != nil {
		limit = int(*req.Params.Limit)
	}
	rows, err := h.svc.Decisions(ctx, decision, limit)
	if err != nil {
		return nil, err
	}
	items := []oapi.ActivationDecision{}
	for _, d := range rows {
		reasons := oapi.Reasons{}
		for _, r := range d.Reasons {
			reasons = append(reasons, oapi.ReasonCode(r))
		}
		blocked := d.BlockedBy
		if blocked == nil {
			blocked = []string{}
		}
		items = append(items, oapi.ActivationDecision{ActivationId: authzhttp.UUID(d.ActivationID), BusinessKey: d.BusinessKey, InvoiceId: authzhttp.UUID(d.InvoiceID),
			InvoiceRef: d.InvoiceRef, CustomerLinkState: oapi.ActivationDecisionCustomerLinkState(d.LinkState), CustomerId: authzhttp.OptUUID(d.CustomerID),
			ExistingCustomer: d.Existing, Route: oapi.ActivationDecisionRoute(d.Route), ProductKey: d.ProductKey, ProductVersion: d.ProductVersion,
			Decision: oapi.ActivationDecisionDecision(d.Decision), Reasons: reasons, BlockedBy: blocked, SubscriptionId: authzhttp.OptUUID(d.SubscriptionID),
			ItemCount: d.ItemCount, RolloutMode: oapi.ActivationDecisionRolloutMode(d.Mode), CorrelationId: d.Correlation, MessageId: authzhttp.UUID(d.MessageID),
			DecidedAt: d.DecidedAt.UTC()})
	}
	return oapi.ListBiawinActivationDecisions200JSONResponse{Body: oapi.ActivationListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx), Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
