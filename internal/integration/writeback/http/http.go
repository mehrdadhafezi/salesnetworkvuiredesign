// Package http adapts the write-back diagnostics read service to the
// generated OpenAPI strict server: counts, states, attempts and
// reconciliation records only (no intent payloads, no PII, no SQL here). It
// has no fix, retry or resolve action: resolution is domain-owned.
package http

import (
	"context"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the write-back diagnostics.
type Handlers struct {
	guard *authorization.Guard // flag integration.writeback_read
	q     *writeback.Query
}

// New returns the handlers.
func New(guard *authorization.Guard, q *writeback.Query) *Handlers {
	return &Handlers{guard: guard, q: q}
}

// WritebackEnabled reports the diagnostics flag (navigation hint).
func (h *Handlers) WritebackEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

type gated struct {
	outcome *oapi.Outcome
	reasons []oapi.ReasonCode
	denied  *oapi.DeniedJSONResponse
}

func (h *Handlers) gate(ctx context.Context) (gated, error) {
	c, err := h.guard.Check(ctx, identityapi.CapWritebackRead)
	if err != nil {
		return gated{}, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return gated{outcome: &o, reasons: r}, nil
	case authorization.GateDenied:
		d := authzhttp.Denied(ctx, c.Decision)
		return gated{denied: &d}, nil
	}
	return gated{}, nil
}

func countsOf(cs []writeback.Count) []oapi.WritebackCount {
	out := []oapi.WritebackCount{}
	for _, c := range cs {
		out = append(out, oapi.WritebackCount{Key: c.Key, Count: c.N})
	}
	return out
}

// GetWritebackSummary returns the bounded counts.
func (h *Handlers) GetWritebackSummary(ctx context.Context, _ oapi.GetWritebackSummaryRequestObject) (oapi.GetWritebackSummaryResponseObject, error) {
	g, err := h.gate(ctx)
	if err != nil {
		return nil, err
	}
	if g.denied != nil {
		return oapi.GetWritebackSummary403JSONResponse{DeniedJSONResponse: *g.denied}, nil
	}
	if g.outcome != nil {
		return oapi.GetWritebackSummary200JSONResponse{Body: oapi.WritebackSummaryEnvelope{Outcome: *g.outcome, Reasons: g.reasons, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	}
	s, err := h.q.Summary(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.WritebackSummary{States: countsOf(s.States), Parity: countsOf(s.Parity), Reconciliations: countsOf(s.Reconciliations)}
	return oapi.GetWritebackSummary200JSONResponse{Body: oapi.WritebackSummaryEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &data}}, nil
}

func commandOf(c writeback.Command) oapi.WritebackCommand {
	out := oapi.WritebackCommand{CommandId: authzhttp.UUID(c.CommandID), SourceModule: oapi.WritebackCommandSourceModule(c.SourceModule),
		CommandType: c.CommandType, SubjectKind: oapi.WritebackCommandSubjectKind(c.SubjectKind), SubjectId: authzhttp.UUID(c.SubjectID),
		RolloutMode: oapi.WritebackCommandRolloutMode(c.Mode), State: oapi.WritebackState(c.State), Attempts: c.Attempts, LegacyRef: c.LegacyRef,
		ReceivedAt: c.Received.UTC(), UpdatedAt: c.Updated.UTC()}
	if c.Parity != nil {
		p := oapi.ParityResult11(*c.Parity)
		out.Parity = &p
	}
	return out
}

// ListWritebackCommands lists commands.
func (h *Handlers) ListWritebackCommands(ctx context.Context, req oapi.ListWritebackCommandsRequestObject) (oapi.ListWritebackCommandsResponseObject, error) {
	g, err := h.gate(ctx)
	if err != nil {
		return nil, err
	}
	if g.denied != nil {
		return oapi.ListWritebackCommands403JSONResponse{DeniedJSONResponse: *g.denied}, nil
	}
	if g.outcome != nil {
		return oapi.ListWritebackCommands200JSONResponse{Body: oapi.WritebackCommandListEnvelope{Outcome: *g.outcome, Reasons: g.reasons, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	}
	var state *string
	if req.Params.State != nil {
		s := string(*req.Params.State)
		state = &s
	}
	limit := 50
	if req.Params.Limit != nil {
		limit = int(*req.Params.Limit)
	}
	rows, next, err := h.q.List(ctx, state, req.Params.Cursor, limit)
	if errors.Is(err, writeback.ErrInvalid) {
		return oapi.ListWritebackCommands400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []oapi.WritebackCommand{}
	for _, c := range rows {
		items = append(items, commandOf(c))
	}
	return oapi.ListWritebackCommands200JSONResponse{Body: oapi.WritebackCommandListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Items: &items, Page: &oapi.PageInfo{HasMore: next != nil, NextCursor: next}}}, nil
}

// GetWritebackCommand returns one command with its attempts and records.
func (h *Handlers) GetWritebackCommand(ctx context.Context, req oapi.GetWritebackCommandRequestObject) (oapi.GetWritebackCommandResponseObject, error) {
	g, err := h.gate(ctx)
	if err != nil {
		return nil, err
	}
	if g.denied != nil {
		return oapi.GetWritebackCommand403JSONResponse{DeniedJSONResponse: *g.denied}, nil
	}
	if g.outcome != nil {
		return oapi.GetWritebackCommand200JSONResponse{Body: oapi.WritebackCommandEnvelope{Outcome: *g.outcome, Reasons: g.reasons, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	}
	c, err := h.q.Get(ctx, req.CommandId.String())
	switch {
	case errors.Is(err, writeback.ErrNotFound):
		return oapi.GetWritebackCommand404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case errors.Is(err, writeback.ErrInvalid):
		return oapi.GetWritebackCommand400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case err != nil:
		return nil, err
	}
	out := commandOf(c)
	log := []oapi.WritebackAttempt{}
	for _, a := range c.Log {
		w := oapi.WritebackAttempt{AttemptNo: a.No, Kind: oapi.WritebackAttemptKind(a.Kind), Outcome: oapi.Outcome(a.Outcome), Reasons: a.Reasons,
			ServiceActor: a.ServiceActor, LegacyRef: a.LegacyRef, StartedAt: a.Started.UTC(), FinishedAt: a.Finished.UTC()}
		if w.Reasons == nil {
			w.Reasons = []string{}
		}
		if a.Observation != nil {
			o := oapi.WritebackAttemptObservation(*a.Observation)
			w.Observation = &o
		}
		log = append(log, w)
	}
	recs := []oapi.WritebackReconciliation{}
	for _, r := range c.Reconciliations {
		recs = append(recs, oapi.WritebackReconciliation{Reason: oapi.WritebackReconciliationReason(r.Reason),
			NextSafeAction: oapi.WritebackReconciliationNextSafeAction(r.NextSafeAction), AttemptNo: r.AttemptNo, OpenedAt: r.OpenedAt.UTC()})
	}
	out.AttemptLog, out.Reconciliations = &log, &recs
	return oapi.GetWritebackCommand200JSONResponse{Body: oapi.WritebackCommandEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &out}}, nil
}
