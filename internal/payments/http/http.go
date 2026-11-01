// Package http adapts the payments read service to the generated OpenAPI
// strict server. Handlers guard (flag + evaluator), call the read service
// and map rows to contract types. No SQL, no writes, no import command
// (lint). Gateway references are never exposed (only whether a hashed one
// exists); receipts are reference metadata only (no URL, path or file).
// Legacy 'approved' stays raw beside the explicit state and provenance.
package http

import (
	"context"
	"errors"
	"strconv"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the payment attempt and receipt read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
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

func source(p store.Provenance) oapi.BillingSource {
	return oapi.BillingSource{SourceSystem: p.System, SourceSpec: p.Spec, SourceKey: p.Key, FirstBatchId: authzhttp.UUID(p.FirstBatch),
		LastBatchId: authzhttp.UUID(p.LastBatch), Presence: oapi.Presence(p.Presence), FirstSeenAt: p.FirstSeen, LastSeenAt: p.LastSeen}
}

func attempt(a store.Attempt) oapi.PaymentAttempt {
	amt := oapi.BillingAmount{Raw: a.AmountRaw, Representation: oapi.AmountRepresentation(a.AmountRepr)}
	if a.AmountRepr == domain.Representable && a.Amount != nil && a.Unit != nil {
		amt.Money = &oapi.Money{Amount: strconv.FormatInt(*a.Amount, 10), Unit: oapi.Unit(*a.Unit)}
	}
	out := oapi.PaymentAttempt{AttemptId: authzhttp.UUID(a.AttemptID), InvoiceId: uuidp(a.InvoiceID),
		InvoiceLinkState: oapi.PaymentAttemptInvoiceLinkState(a.InvoiceLink), StageNoRaw: a.StageNoRaw, Channel: oapi.PaymentChannel(a.Channel),
		PayMethodRaw: a.PayMethodRaw, PaymentSourceRaw: a.PaymentSourceRaw, Amount: amt, StateNamespace: a.StateNamespace, StateRaw: a.StateRaw,
		StateMapping: oapi.TargetMapping(a.StateMapping), ApprovalProvenance: oapi.ApprovalProvenance(a.ApprovalProvenance),
		UploadedByTypeRaw: a.UploadedByTypeRaw, CreatedAtRaw: a.CreatedAtRaw, UpdatedAtRaw: a.UpdatedAtRaw, Version: a.Version, Source: source(a.Source)}
	if a.StateTarget != nil {
		s := oapi.PaymentAttemptState(*a.StateTarget)
		out.State = &s
	}
	if a.UploadedByKey != nil {
		out.UploadedBy = &oapi.UserRef{LegacyKey: *a.UploadedByKey, UserId: uuidp(a.UploadedByUser)}
	}
	if g := a.Gateway; g != nil {
		ge := oapi.GatewayEvidence{ProviderRaw: g.ProviderRaw, HasAuthorityReference: g.HasAuthority, HasGatewayReference: g.HasReference,
			StateRaw: g.StateRaw, StateMapping: oapi.TargetMapping(g.StateMapping)}
		if g.StateTarget != nil {
			s := oapi.GatewayEvidenceState(*g.StateTarget)
			ge.State = &s
		}
		out.Gateway = &ge
	}
	return out
}

func receipt(r store.Receipt) oapi.ReceiptReference {
	return oapi.ReceiptReference{ReceiptId: authzhttp.UUID(r.ReceiptID), InvoiceId: uuidp(r.InvoiceID), OwnerKind: oapi.ReceiptReferenceOwnerKind(r.OwnerKind),
		StageNoRaw: r.StageNoRaw, RefIndex: r.RefIndex, HasReference: r.HasReference, Host: r.Host, Extension: r.Extension,
		State: oapi.ReceiptState(r.State), Source: source(r.Source)}
}

type gate[R any] struct {
	disabled func(o oapi.Outcome, r []oapi.ReasonCode) R
	denied   func(d oapi.DeniedJSONResponse) R
}

func guardOp[R any](ctx context.Context, h *Handlers, gt gate[R]) (authorization.Check, *R, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPaymentAttemptRead)
	if err != nil {
		return c, nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		res := gt.disabled(o, r)
		return c, &res, nil
	case authorization.GateDenied:
		res := gt.denied(authzhttp.Denied(ctx, c.Decision))
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

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

func attempts(rows []store.Attempt) *[]oapi.PaymentAttempt {
	items := make([]oapi.PaymentAttempt, 0, len(rows))
	for _, a := range rows {
		items = append(items, attempt(a))
	}
	return &items
}

// ListPaymentAttempts lists attempts with allowlisted filters.
func (h *Handlers) ListPaymentAttempts(ctx context.Context, req oapi.ListPaymentAttemptsRequestObject) (oapi.ListPaymentAttemptsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, gate[oapi.ListPaymentAttemptsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListPaymentAttemptsResponseObject {
			return oapi.ListPaymentAttempts200JSONResponse{Body: oapi.PaymentAttemptListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListPaymentAttemptsResponseObject {
			return oapi.ListPaymentAttempts403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	p := req.Params
	res, err := h.q.ListAttempts(ctx, store.AttemptFilter{Channel: strp(p.Channel), StateMapping: strp(p.StateMapping), StateTarget: strp(p.State),
		Provenance: strp(p.ApprovalProvenance), Presence: strp(p.Presence), InvoiceLink: strp(p.InvoiceLinkState)}, p.Cursor, p.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListPaymentAttempts400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	o, r := ok()
	return oapi.ListPaymentAttempts200JSONResponse{Body: oapi.PaymentAttemptListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: attempts(res.Attempts), Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListInvoicePaymentAttempts pages the attempts of one invoice. An invoice
// with no attempt (or unknown to payments) yields an empty page.
func (h *Handlers) ListInvoicePaymentAttempts(ctx context.Context, req oapi.ListInvoicePaymentAttemptsRequestObject) (oapi.ListInvoicePaymentAttemptsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, gate[oapi.ListInvoicePaymentAttemptsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListInvoicePaymentAttemptsResponseObject {
			return oapi.ListInvoicePaymentAttempts200JSONResponse{Body: oapi.PaymentAttemptListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListInvoicePaymentAttemptsResponseObject {
			return oapi.ListInvoicePaymentAttempts403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	id := req.InvoiceId.String()
	res, err := h.q.ListAttempts(ctx, store.AttemptFilter{InvoiceID: &id}, req.Params.Cursor, req.Params.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListInvoicePaymentAttempts400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	o, r := ok()
	return oapi.ListInvoicePaymentAttempts200JSONResponse{Body: oapi.PaymentAttemptListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: attempts(res.Attempts), Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetPaymentAttempt returns one attempt.
func (h *Handlers) GetPaymentAttempt(ctx context.Context, req oapi.GetPaymentAttemptRequestObject) (oapi.GetPaymentAttemptResponseObject, error) {
	c, stop, err := guardOp(ctx, h, gate[oapi.GetPaymentAttemptResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetPaymentAttemptResponseObject {
			return oapi.GetPaymentAttempt200JSONResponse{Body: oapi.PaymentAttemptEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetPaymentAttemptResponseObject {
			return oapi.GetPaymentAttempt403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	a, err := h.q.GetAttempt(ctx, req.AttemptId.String())
	if errors.Is(err, query.ErrNotFound) {
		return oapi.GetPaymentAttempt404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	d := attempt(a)
	o, r := ok()
	return oapi.GetPaymentAttempt200JSONResponse{Body: oapi.PaymentAttemptEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Data: &d,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListPaymentAttemptHistory pages an attempt's observed state changes.
func (h *Handlers) ListPaymentAttemptHistory(ctx context.Context, req oapi.ListPaymentAttemptHistoryRequestObject) (oapi.ListPaymentAttemptHistoryResponseObject, error) {
	c, stop, err := guardOp(ctx, h, gate[oapi.ListPaymentAttemptHistoryResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListPaymentAttemptHistoryResponseObject {
			return oapi.ListPaymentAttemptHistory200JSONResponse{Body: oapi.AttemptHistoryListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListPaymentAttemptHistoryResponseObject {
			return oapi.ListPaymentAttemptHistory403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	res, err := h.q.History(ctx, req.AttemptId.String(), req.Params.Cursor, req.Params.Limit)
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.ListPaymentAttemptHistory400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.ListPaymentAttemptHistory404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	items := make([]oapi.AttemptHistoryEntry, 0, len(res.Entries))
	for _, e := range res.Entries {
		items = append(items, oapi.AttemptHistoryEntry{HistoryId: authzhttp.UUID(e.HistoryID), OldStateRaw: e.OldStateRaw, NewStateRaw: e.NewRaw,
			Regression: e.Regression, BatchId: authzhttp.UUID(e.BatchID), RecordedAt: e.RecordedAt})
	}
	o, r := ok()
	return oapi.ListPaymentAttemptHistory200JSONResponse{Body: oapi.AttemptHistoryListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListInvoiceReceipts pages an invoice's receipt references.
func (h *Handlers) ListInvoiceReceipts(ctx context.Context, req oapi.ListInvoiceReceiptsRequestObject) (oapi.ListInvoiceReceiptsResponseObject, error) {
	c, stop, err := guardOp(ctx, h, gate[oapi.ListInvoiceReceiptsResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.ListInvoiceReceiptsResponseObject {
			return oapi.ListInvoiceReceipts200JSONResponse{Body: oapi.ReceiptReferenceListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.ListInvoiceReceiptsResponseObject {
			return oapi.ListInvoiceReceipts403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	res, err := h.q.ReceiptsOf(ctx, req.InvoiceId.String(), req.Params.Cursor, req.Params.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListInvoiceReceipts400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.ReceiptReference, 0, len(res.Receipts))
	for _, x := range res.Receipts {
		items = append(items, receipt(x))
	}
	o, r := ok()
	return oapi.ListInvoiceReceipts200JSONResponse{Body: oapi.ReceiptReferenceListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetReceiptReference returns one receipt reference.
func (h *Handlers) GetReceiptReference(ctx context.Context, req oapi.GetReceiptReferenceRequestObject) (oapi.GetReceiptReferenceResponseObject, error) {
	c, stop, err := guardOp(ctx, h, gate[oapi.GetReceiptReferenceResponseObject]{
		disabled: func(o oapi.Outcome, r []oapi.ReasonCode) oapi.GetReceiptReferenceResponseObject {
			return oapi.GetReceiptReference200JSONResponse{Body: oapi.ReceiptReferenceEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}
		},
		denied: func(d oapi.DeniedJSONResponse) oapi.GetReceiptReferenceResponseObject {
			return oapi.GetReceiptReference403JSONResponse{DeniedJSONResponse: d}
		},
	})
	if err != nil || stop != nil {
		return deref(stop), err
	}
	x, err := h.q.GetReceipt(ctx, req.ReceiptId.String())
	if errors.Is(err, query.ErrNotFound) {
		return oapi.GetReceiptReference404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	if err != nil {
		return nil, err
	}
	d := receipt(x)
	o, r := ok()
	return oapi.GetReceiptReference200JSONResponse{Body: oapi.ReceiptReferenceEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Data: &d,
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
