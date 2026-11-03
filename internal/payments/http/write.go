package http

import (
	"context"
	"errors"
	"io"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// WriteHandlers serves the Slice 12 payments surface: receipt upload, the
// submission context, the manual payment batch, the caller's batch read
// and evidence diagnostics. Handlers never run SQL, never call a connector
// or a gateway and never decide policy: they delegate to the payments
// command service, which re-evaluates everything at commit.
type WriteHandlers struct {
	guard *authorization.Guard // flag payments.manual_write
	mode  rollout.Source
	svc   *command.Service
}

// NewWrite returns the handlers.
func NewWrite(guard *authorization.Guard, mode rollout.Source, svc *command.Service) *WriteHandlers {
	return &WriteHandlers{guard: guard, mode: mode, svc: svc}
}

// PaymentSubmissionEnabled reports the Slice 12 flag (navigation hint).
func (h *WriteHandlers) PaymentSubmissionEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

// BlockedPaymentActions are legacy financial actions that exist but are
// unavailable in the target, with the decisions blocking them.
func BlockedPaymentActions() []oapi.BlockedAction {
	return []oapi.BlockedAction{
		{Action: "submit_payment_info_without_receipt", BlockedBy: []string{command.DecisionD43}},
		{Action: "submit_payment_for_another_seller", BlockedBy: []string{command.DecisionOPD10}},
		{Action: "pay_online_gateway", BlockedBy: []string{"CUSTOMER_CHANNEL_NOT_BUILT", command.DecisionD25, command.DecisionD26}},
		{Action: "finance_approve_payment", BlockedBy: []string{"D-41", command.DecisionOPD08, "OPD-09", command.DecisionD43}},
		{Action: "finance_reject_payment", BlockedBy: []string{command.DecisionOPD08, "BILLING_REJECTION_TRANSITION_UNDEFINED"}},
		{Action: "issue_next_payment_stage", BlockedBy: []string{"D-32", "BILLING_STAGE_LIFECYCLE_UNKNOWN"}},
		{Action: "operations_upgrade_payment", BlockedBy: []string{"D-33"}},
		{Action: "refund_payment", BlockedBy: []string{"OPD-04"}},
		{Action: "issue_invoice", BlockedBy: []string{"ADR-01"}},
		{Action: "return_case", BlockedBy: []string{"ADR-01", "F01", "OPD-03"}},
	}
}

func (h *WriteHandlers) actor(ctx context.Context) command.Actor {
	s := h.guard.Subject(ctx)
	return command.Actor{Principal: s.PrincipalID, UserID: s.UserID}
}

func strs(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func unit(u string) *oapi.WriteUnit {
	if u == "" {
		return nil
	}
	w := oapi.WriteUnit(u)
	return &w
}

// StagePaymentUpload stages one receipt file.
func (h *WriteHandlers) StagePaymentUpload(ctx context.Context, req oapi.StagePaymentUploadRequestObject) (oapi.StagePaymentUploadResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPaymentUploadStage)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.StagePaymentUpload200JSONResponse{Body: oapi.PaymentUploadEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
			Data: &oapi.PaymentUploadResult{RolloutMode: oapi.RolloutModeTARGETWRITEDISABLED, BlockedBy: []string{}}}}, nil
	case authorization.GateDenied:
		return oapi.StagePaymentUpload403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	if req.Body == nil {
		return oapi.StagePaymentUpload400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "file body required")}, nil
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, command.MaxUploadBytes+1))
	if err != nil {
		return oapi.StagePaymentUpload413ApplicationProblemPlusJSONResponse{Body: httpx.ProblemBodyTooLarge.BodyCtx(ctx)}, nil
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.svc.Upload(ctx, mode, h.actor(ctx), body)
	switch {
	case errors.Is(err, command.ErrInvalid):
		return oapi.StagePaymentUpload400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case err != nil:
		return nil, err
	}
	env := oapi.PaymentUploadEnvelope{Outcome: authzhttp.Outcome(r.Status), Reasons: authzhttp.Reasons(r.Reasons...), CorrelationId: httpx.CorrelationID(ctx)}
	data := oapi.PaymentUploadResult{RolloutMode: oapi.RolloutMode(r.Mode), Existing: r.Existing, BlockedBy: strs(r.BlockedBy), ExpiresAt: r.ExpiresAt}
	if r.UploadID != "" {
		id := authzhttp.UUID(r.UploadID)
		sum, size, media, state := r.ContentSHA256, r.SizeBytes, oapi.PaymentUploadResultMediaType(r.MediaType), oapi.UploadState(r.State)
		data.UploadId, data.ContentSha256, data.SizeBytes, data.MediaType, data.State = &id, &sum, &size, &media, &state
	}
	env.Data = &data
	if r.Status == result.Unauthorized {
		return oapi.StagePaymentUpload403JSONResponse{DeniedJSONResponse: oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: env.Outcome,
			Reasons: env.Reasons, CorrelationId: env.CorrelationId}}}, nil
	}
	return oapi.StagePaymentUpload200JSONResponse{Body: env}, nil
}

func attemptsOf(as []command.AttemptView) []oapi.SubmittedAttempt {
	out := make([]oapi.SubmittedAttempt, 0, len(as))
	for _, a := range as {
		v := oapi.SubmittedAttempt{AttemptId: authzhttp.UUID(a.AttemptID), BatchId: authzhttp.UUID(a.BatchID), ItemKey: a.ItemKey, Amount: a.Amount,
			Unit: oapi.WriteUnit(a.Unit), State: oapi.SubmittedAttemptState(a.State), ReviewProvenance: oapi.SubmittedAttemptReviewProvenance(a.ReviewProvenance),
			EvidenceState: oapi.EvidenceState(a.EvidenceState), SubmittedAt: a.SubmittedAt, Receipts: []oapi.PaymentReceipt{}}
		for _, r := range a.Receipts {
			v.Receipts = append(v.Receipts, oapi.PaymentReceipt{ReceiptId: authzhttp.UUID(r.ReceiptID), UploadId: authzhttp.UUID(r.UploadID),
				State: oapi.EvidenceReceiptState(r.State), MediaType: r.MediaType, SizeBytes: r.SizeBytes, UpdatedAt: r.UpdatedAt})
		}
		out = append(out, v)
	}
	return out
}

// GetPaymentSubmission reads the submission context with allowed_actions
// from the command's own evaluator.
func (h *WriteHandlers) GetPaymentSubmission(ctx context.Context, req oapi.GetPaymentSubmissionRequestObject) (oapi.GetPaymentSubmissionResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPaymentSubmissionRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetPaymentSubmission200JSONResponse{Body: oapi.PaymentSubmissionEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetPaymentSubmission403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	actor := h.actor(ctx)
	v, found, err := h.svc.Context(ctx, req.InvoiceId.String(), actor)
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.GetPaymentSubmission404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.PaymentSubmissionContext{InvoiceId: req.InvoiceId, StageNo: v.StageNo, Unit: nil, DueAmount: v.Due, RemainingAmount: v.Remaining,
		PendingOrAcceptedAmount: v.Used, AvailableAmount: v.Available, CollectionVersion: v.CollectionVersion, Eligible: v.Eligible,
		EvaluationStatus: oapi.Outcome(v.Status), Reasons: authzhttp.Reasons(v.Reasons...), BlockedBy: strs(v.BlockedBy), RolloutMode: oapi.RolloutMode(mode),
		Attempts: attemptsOf(v.Attempts), BlockedActions: BlockedPaymentActions()}
	if v.Unit != nil {
		data.Unit = unit(*v.Unit)
	}
	// allowed_actions: flag + capability + rollout mode + the command's own
	// eligibility evaluation. Never inferred from a role or the hierarchy.
	action := oapi.AllowedAction{Allowed: false}
	wc, err := h.guard.Check(ctx, identityapi.CapPaymentBatchSubmit)
	if err != nil {
		return nil, err
	}
	code := reasoncodes.FeatureDisabled
	switch {
	case wc.Gate == authorization.GateDenied:
		code = reasoncodes.ActionDenied
	case wc.Gate == authorization.GateAllowed && mode == rollout.Shadow:
		code = reasoncodes.PolicyUnresolved
	case wc.Gate == authorization.GateAllowed && mode == rollout.Controlled:
		action.Allowed = v.Eligible
		if !v.Eligible && len(v.Reasons) > 0 {
			code = v.Reasons[0]
		}
	}
	if !action.Allowed {
		r := oapi.ReasonCode(code)
		action.ReasonCode = &r
	}
	allowed := oapi.AllowedActions{string(identityapi.CapPaymentBatchSubmit): action}
	return oapi.GetPaymentSubmission200JSONResponse{Body: oapi.PaymentSubmissionEnvelope{Outcome: authzhttp.Outcome(result.Success),
		Reasons: authzhttp.Reasons(), CorrelationId: httpx.CorrelationID(ctx), Data: &data, AllowedActions: &allowed}}, nil
}

func batchResult(r command.BatchResult, invoiceID string) *oapi.PaymentBatchResult {
	out := &oapi.PaymentBatchResult{RolloutMode: oapi.RolloutMode(r.Mode), Applied: r.Applied, InvoiceId: authzhttp.UUID(invoiceID), StageNo: r.StageNo,
		TotalAmount: r.TotalAmount, Unit: unit(r.Unit), Items: []oapi.PaymentBatchItemOutcome{}, BlockedBy: strs(r.BlockedBy),
		WritebackState: oapi.PaymentBatchResultWritebackState(r.WritebackState)}
	if out.RolloutMode == "" {
		out.RolloutMode = oapi.RolloutModeTARGETWRITEDISABLED
	}
	if r.BatchID != "" {
		id := authzhttp.UUID(r.BatchID)
		out.BatchId = &id
	}
	if r.CollectionVersion > 0 {
		v := r.CollectionVersion
		out.CollectionVersion = &v
	}
	if r.Status == result.Conflict {
		v := r.CurrentVersion
		out.CurrentVersion = &v
	}
	if r.EvidenceState != "" {
		e := oapi.EvidenceState(r.EvidenceState)
		out.EvidenceState = &e
	}
	if r.Replayed {
		t := true
		out.Replayed = &t
	}
	for _, it := range r.Items {
		o := oapi.PaymentBatchItemOutcome{ItemKey: it.ItemKey, Outcome: authzhttp.Outcome(it.Status), Reasons: authzhttp.Reasons(it.Reasons...),
			BlockedBy: strs(it.BlockedBy), Valid: it.Valid, Committed: it.Committed, Amount: it.Amount, Unit: oapi.WriteUnit(it.Unit),
			Receipts: []oapi.PaymentReceiptOutcome{}}
		if it.AttemptID != "" {
			id := authzhttp.UUID(it.AttemptID)
			o.AttemptId = &id
		}
		for _, rc := range it.Receipts {
			o.Receipts = append(o.Receipts, oapi.PaymentReceiptOutcome{ReceiptId: authzhttp.UUID(rc.ReceiptID), UploadId: authzhttp.UUID(rc.UploadID),
				State: oapi.EvidenceReceiptState(rc.State)})
		}
		out.Items = append(out.Items, o)
	}
	return out
}

// SubmitPaymentBatch runs the command for the caller.
func (h *WriteHandlers) SubmitPaymentBatch(ctx context.Context, req oapi.SubmitPaymentBatchRequestObject) (oapi.SubmitPaymentBatchResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPaymentBatchSubmit)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.SubmitPaymentBatch200JSONResponse{Body: oapi.PaymentBatchEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
			Data: batchResult(command.BatchResult{Mode: rollout.Disabled, WritebackState: command.WritebackNotRequested}, req.InvoiceId.String())}}, nil
	case authorization.GateDenied:
		return oapi.SubmitPaymentBatch403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	if req.Body == nil {
		return oapi.SubmitPaymentBatch400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, "body required")}, nil
	}
	version, ok := authzhttp.IfMatchCollectionVersion(req.Params.IfMatch)
	if !ok {
		return oapi.SubmitPaymentBatch400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx,
			"If-Match must carry the collection version")}, nil
	}
	mode, err := h.mode(ctx)
	if err != nil {
		return nil, err
	}
	cmd := command.BatchCommand{InvoiceID: req.InvoiceId.String(), StageNo: req.Body.StageNo, EntryMode: string(req.Body.EntryMode),
		ExpectedVersion: version, IdempotencyKey: req.Params.IdempotencyKey, CorrelationID: httpx.CorrelationID(ctx), Actor: h.actor(ctx)}
	for _, it := range req.Body.Items {
		i := command.Item{ItemKey: it.ItemKey, Amount: it.Amount, Unit: string(it.Unit)}
		for _, r := range it.Receipts {
			i.Receipts = append(i.Receipts, command.ReceiptRef{UploadID: r.UploadId.String(), ContentSHA256: r.ContentSha256})
		}
		cmd.Items = append(cmd.Items, i)
	}
	r, err := h.svc.SubmitBatch(ctx, mode, cmd)
	switch {
	case errors.Is(err, command.ErrInvalid):
		return oapi.SubmitPaymentBatch400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, command.ErrNotFound):
		return oapi.SubmitPaymentBatch404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	env := oapi.PaymentBatchEnvelope{Outcome: authzhttp.Outcome(r.Status), Reasons: authzhttp.Reasons(r.Reasons...), CorrelationId: httpx.CorrelationID(ctx),
		Data: batchResult(r, cmd.InvoiceID)}
	switch r.Status {
	case result.Unauthorized:
		return oapi.SubmitPaymentBatch403JSONResponse{DeniedJSONResponse: oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: env.Outcome,
			Reasons: env.Reasons, CorrelationId: env.CorrelationId}}}, nil
	case result.Conflict:
		return oapi.SubmitPaymentBatch409JSONResponse{Body: env}, nil
	}
	return oapi.SubmitPaymentBatch200JSONResponse{Body: env}, nil
}

// GetPaymentBatch reads one of the caller's committed batches.
func (h *WriteHandlers) GetPaymentBatch(ctx context.Context, req oapi.GetPaymentBatchRequestObject) (oapi.GetPaymentBatchResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPaymentSubmissionRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetPaymentBatch200JSONResponse{Body: oapi.PaymentBatchReadEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetPaymentBatch403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	b, found, err := h.svc.Batch(ctx, req.BatchId.String(), h.actor(ctx))
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.GetPaymentBatch404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	data := oapi.PaymentBatch{BatchId: authzhttp.UUID(b.BatchID), InvoiceId: authzhttp.UUID(b.InvoiceID), StageNo: b.StageNo, ItemCount: b.ItemCount,
		TotalAmount: b.TotalAmount, Unit: oapi.WriteUnit(b.Unit), CollectionVersionBefore: b.BeforeVersion, CollectionVersionAfter: b.AfterVersion,
		CorrelationId: b.CorrelationID, CreatedAt: b.CreatedAt, Attempts: attemptsOf(b.Attempts)}
	return oapi.GetPaymentBatch200JSONResponse{Body: oapi.PaymentBatchReadEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &data}}, nil
}

// GetPaymentEvidence reads the evidence diagnostics.
func (h *WriteHandlers) GetPaymentEvidence(ctx context.Context, _ oapi.GetPaymentEvidenceRequestObject) (oapi.GetPaymentEvidenceResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPaymentEvidenceRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetPaymentEvidence200JSONResponse{Body: oapi.EvidenceDiagnosticsEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetPaymentEvidence403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	ev, err := h.svc.Evidence(ctx, 200)
	if err != nil {
		return nil, err
	}
	data := oapi.EvidenceDiagnostics{ReceiptsByState: ev.Receipts, UploadsByState: ev.Uploads, Reconciliations: []oapi.EvidenceReconciliation{}}
	for _, r := range ev.Reconciliations {
		data.Reconciliations = append(data.Reconciliations, oapi.EvidenceReconciliation{ReconciliationId: authzhttp.UUID(r.ReconciliationID),
			ReceiptId: authzhttp.UUID(r.ReceiptID), AttemptId: authzhttp.UUID(r.AttemptID), Kind: oapi.EvidenceReconciliationKind(r.Kind),
			Reason: oapi.EvidenceReconciliationReason(r.Reason), DomainOwner: oapi.EvidenceReconciliationDomainOwner(r.DomainOwner),
			NextSafeAction: oapi.EvidenceReconciliationNextSafeAction(r.NextSafeAction), ReceiptState: oapi.EvidenceReceiptState(r.ReceiptState),
			OpenedAt: r.OpenedAt})
	}
	return oapi.GetPaymentEvidence200JSONResponse{Body: oapi.EvidenceDiagnosticsEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &data}}, nil
}
