package http

import (
	"context"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/review"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	paymentsapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

// ReviewHandlers serves the Slice 12 read-only Finance review queue. There
// is no decision route: the gate of every attempt is computed by the
// finance review evaluator and shown, never acted on.
type ReviewHandlers struct {
	guard *authorization.Guard // flag finance.review_read
	src   paymentsapi.ReviewSource
}

// NewReview returns the handlers.
func NewReview(guard *authorization.Guard, src paymentsapi.ReviewSource) *ReviewHandlers {
	return &ReviewHandlers{guard: guard, src: src}
}

// FinanceReviewEnabled reports the Slice 12 flag (navigation hint).
func (h *ReviewHandlers) FinanceReviewEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func (h *ReviewHandlers) view(a paymentsapi.ReviewAttempt, reviewer string) oapi.ReviewAttempt {
	g := review.Evaluate(a, reviewer)
	out := oapi.ReviewAttempt{AttemptId: authzhttp.UUID(a.AttemptID), BatchId: authzhttp.UUID(a.BatchID), InvoiceId: authzhttp.UUID(a.InvoiceID),
		StageNo: a.StageNo, Amount: a.Amount, Unit: oapi.WriteUnit(a.Unit), State: oapi.ReviewAttemptState(a.State),
		ReviewProvenance: oapi.ReviewAttemptReviewProvenance(a.ReviewProvenance), EvidenceState: oapi.EvidenceState(a.EvidenceState),
		EvidenceReady: g.EvidenceReady, ReviewerIsSubmitter: g.ReviewerIsMaker, SubmittedByUserId: authzhttp.UUID(a.SubmittedByUserID),
		SubmittedAt: a.SubmittedAt, Receipts: []oapi.PaymentReceipt{}, Gates: []oapi.ReviewActionGate{}}
	for _, r := range a.Receipts {
		out.Receipts = append(out.Receipts, oapi.PaymentReceipt{ReceiptId: authzhttp.UUID(r.ReceiptID), UploadId: authzhttp.UUID(r.UploadID),
			State: oapi.EvidenceReceiptState(r.State), MediaType: r.MediaType, SizeBytes: r.SizeBytes, UpdatedAt: r.UpdatedAt})
	}
	for _, b := range g.Actions {
		out.Gates = append(out.Gates, oapi.ReviewActionGate{Action: oapi.ReviewActionGateAction(b.Action), Outcome: authzhttp.Outcome(b.Status),
			Reasons: authzhttp.Reasons(b.Reasons...), BlockedBy: b.BlockedBy})
	}
	return out
}

func (h *ReviewHandlers) gate(ctx context.Context) (authorization.Check, error) {
	return h.guard.Check(ctx, identityapi.CapFinanceReviewRead)
}

// ListFinanceReviewQueue lists attempts awaiting review with their gates.
func (h *ReviewHandlers) ListFinanceReviewQueue(ctx context.Context, req oapi.ListFinanceReviewQueueRequestObject) (oapi.ListFinanceReviewQueueResponseObject, error) {
	c, err := h.gate(ctx)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListFinanceReviewQueue200JSONResponse{Body: oapi.ReviewQueueEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListFinanceReviewQueue403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	limit := 50
	if req.Params.Limit != nil {
		limit = int(*req.Params.Limit)
	}
	var after *string
	if req.Params.Cursor != nil {
		cur, err := pagination.Decode(*req.Params.Cursor, "attempt_id")
		if err != nil {
			return oapi.ListFinanceReviewQueue400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "cursor")}, nil
		}
		after = &cur.ID
	}
	rows, err := h.src.ReviewQueue(ctx, after, limit)
	if err != nil {
		return nil, err
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		n, err := pagination.Encode(pagination.Cursor{Sort: "attempt_id", ID: rows[len(rows)-1].AttemptID})
		if err != nil {
			return nil, err
		}
		next = &n
	}
	reviewer := h.guard.Subject(ctx).UserID
	data := make([]oapi.ReviewAttempt, 0, len(rows))
	for _, a := range rows {
		data = append(data, h.view(a, reviewer))
	}
	return oapi.ListFinanceReviewQueue200JSONResponse{Body: oapi.ReviewQueueEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &data, Page: &oapi.PageInfo{HasMore: next != nil, NextCursor: next}}}, nil
}

// GetFinanceReviewAttempt reads one attempt and its gate. allowed_actions
// never allows a decision in Slice 12.
func (h *ReviewHandlers) GetFinanceReviewAttempt(ctx context.Context, req oapi.GetFinanceReviewAttemptRequestObject) (oapi.GetFinanceReviewAttemptResponseObject, error) {
	c, err := h.gate(ctx)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetFinanceReviewAttempt200JSONResponse{Body: oapi.ReviewAttemptEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetFinanceReviewAttempt403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	a, ok, err := h.src.ReviewAttempt(ctx, req.AttemptId.String())
	if err != nil {
		return nil, err
	}
	if !ok {
		return oapi.GetFinanceReviewAttempt404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	v := h.view(a, h.guard.Subject(ctx).UserID)
	allowed := oapi.AllowedActions{}
	for _, g := range v.Gates {
		code := oapi.ReasonCode(reasoncodes.PolicyUnresolved)
		if len(g.Reasons) > 0 {
			code = g.Reasons[0]
		}
		allowed[string(g.Action)] = oapi.AllowedAction{Allowed: false, ReasonCode: &code}
	}
	return oapi.GetFinanceReviewAttempt200JSONResponse{Body: oapi.ReviewAttemptEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &v, AllowedActions: &allowed}}, nil
}
