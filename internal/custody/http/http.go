// Package http adapts the custody read service to the generated OpenAPI
// strict server. Handlers guard (flag + evaluator), call the read service
// and map rows to contract types. No SQL, no writes, no import command,
// and no action endpoint of any kind (lint; contract).
package http

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/store"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the custody read operations and the parity report.
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

func userRef(key, user *string) *oapi.UserRef {
	if key == nil {
		return nil
	}
	return &oapi.UserRef{LegacyKey: *key, UserId: uuidp(user)}
}

func assignment(a store.Assignment) oapi.CustodyAssignment {
	return oapi.CustodyAssignment{CaseId: authzhttp.UUID(a.CaseID), Derivation: oapi.Derivation(a.Derivation),
		DerivedHolder: userRef(a.DerivedKey, a.DerivedUserID), DerivedPositionRaw: a.DerivedPos, LegacyHolder: userRef(a.LegacyKey, a.LegacyUserID),
		LegacyPositionRaw: a.LegacyPos, LegacySourceSpec: a.LegacySpec, Parity: oapi.Parity(a.Parity), EventCount: a.EventCount,
		FindingCount: a.FindingCount, Presence: oapi.Presence(a.Presence), RefreshedAt: a.RefreshedAt}
}

func blocked(ctx context.Context, h *Handlers, cp identityapi.Capability) (authorization.Check, bool, *oapi.DeniedJSONResponse, error) {
	c, err := h.guard.Check(ctx, cp)
	if err != nil {
		return c, false, nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		return c, true, nil, nil
	case authorization.GateDenied:
		d := authzhttp.Denied(ctx, c.Decision)
		return c, false, &d, nil
	}
	return c, false, nil, nil
}

// ListCustodyAssignments lists derived custody projections.
func (h *Handlers) ListCustodyAssignments(ctx context.Context, req oapi.ListCustodyAssignmentsRequestObject) (oapi.ListCustodyAssignmentsResponseObject, error) {
	c, off, denied, err := blocked(ctx, h, identityapi.CapCustodyRead)
	switch {
	case err != nil:
		return nil, err
	case off:
		o, r := authzhttp.Disabled()
		return oapi.ListCustodyAssignments200JSONResponse{Body: oapi.CustodyAssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case denied != nil:
		return oapi.ListCustodyAssignments403JSONResponse{DeniedJSONResponse: *denied}, nil
	}
	p := req.Params
	res, err := h.q.ListAssignments(ctx, store.AssignmentFilter{Parity: strp(p.Parity), Derivation: strp(p.Derivation), Presence: strp(p.Presence)},
		p.Cursor, p.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListCustodyAssignments400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.CustodyAssignment, 0, len(res.Assignments))
	for _, a := range res.Assignments {
		items = append(items, assignment(a))
	}
	o, r := ok()
	return oapi.ListCustodyAssignments200JSONResponse{Body: oapi.CustodyAssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: &oapi.PageInfo{HasMore: res.NextCursor != nil, NextCursor: res.NextCursor}, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetCaseCustody returns the projection, append-only events and findings.
func (h *Handlers) GetCaseCustody(ctx context.Context, req oapi.GetCaseCustodyRequestObject) (oapi.GetCaseCustodyResponseObject, error) {
	c, off, denied, err := blocked(ctx, h, identityapi.CapCustodyRead)
	switch {
	case err != nil:
		return nil, err
	case off:
		o, r := authzhttp.Disabled()
		return oapi.GetCaseCustody200JSONResponse{Body: oapi.CaseCustodyEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case denied != nil:
		return oapi.GetCaseCustody403JSONResponse{DeniedJSONResponse: *denied}, nil
	}
	cc, err := h.q.CaseCustody(ctx, req.CaseId.String())
	switch {
	case errors.Is(err, query.ErrInvalidQuery):
		return oapi.GetCaseCustody400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case errors.Is(err, query.ErrNotFound):
		return oapi.GetCaseCustody404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	case err != nil:
		return nil, err
	}
	data := oapi.CaseCustody{Assignment: assignment(cc.Assignment), Events: []oapi.CustodyEvent{}, Findings: []oapi.CustodyFinding{}}
	for _, e := range cc.Events {
		data.Events = append(data.Events, oapi.CustodyEvent{EventId: authzhttp.UUID(e.EventID), SourceSpec: e.SourceSpec, SourceKey: e.SourceKey,
			LegacyAction: e.Action, Kind: oapi.EventKind(e.Kind), Context: oapi.EventContext(e.Context), Actor: userRef(e.ActorKey, e.ActorUserID),
			From: userRef(e.FromKey, e.FromUserID), To: userRef(e.ToKey, e.ToUserID), FromPositionRaw: e.FromPos, ToPositionRaw: e.ToPos,
			OccurredRaw: e.OccurredRaw, RecordedAt: e.RecordedAt})
	}
	for _, f := range cc.Findings {
		data.Findings = append(data.Findings, oapi.CustodyFinding{FindingId: authzhttp.UUID(f.FindingID), Kind: oapi.CustodyFindingKind(f.Kind),
			Detail: f.Detail, EventId: uuidp(f.EventID), Presence: oapi.Presence(f.Presence), RecordedAt: f.RecordedAt})
	}
	o, r := ok()
	return oapi.GetCaseCustody200JSONResponse{Body: oapi.CaseCustodyEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ParityNotes states what the parity report cannot claim (four owners,
// GATE-0 INV-004; history recoverability).
var ParityNotes = []string{
	"current-holder parity compares the legacy holder of record with the holder derived from append-only custody events",
	"legacy leads keep no assignment log: their custody is TARGET_UNKNOWN or NOT_RECOVERABLE by design, never assumed",
	"the legacy holder of record is also used for payment-workflow routing (PAYMENT_WORKFLOW events): there Current Custody and Next Actor are not separable from legacy evidence",
	"Original Owner/Seller are frozen attribution snapshots; Credit Owner is not evaluated in Slice 6; Next Actor is not modelled",
	"full parity is claimed only when every present projection is MATCH",
}

// GetCustodyParityReport returns the custody parity report.
func (h *Handlers) GetCustodyParityReport(ctx context.Context, _ oapi.GetCustodyParityReportRequestObject) (oapi.GetCustodyParityReportResponseObject, error) {
	c, off, denied, err := blocked(ctx, h, identityapi.CapSalesReportRead)
	switch {
	case err != nil:
		return nil, err
	case off:
		o, r := authzhttp.Disabled()
		return oapi.GetCustodyParityReport200JSONResponse{Body: oapi.CustodyParityReportEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case denied != nil:
		return oapi.GetCustodyParityReport403JSONResponse{DeniedJSONResponse: *denied}, nil
	}
	p, err := h.q.ParityReport(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.CustodyParityReport{AssignmentsPresent: p.Assignments, CurrentHolderParity: oapi.CountMap(p.ByParity),
		DerivationStates: oapi.CountMap(p.ByDerivation), FindingsByKind: oapi.CountMap(p.FindingsByKind), CustodyEvents: p.Events,
		CustodyEventsByKind: oapi.CountMap(p.EventsByKind), CasesWithCustodyEvents: p.CasesWithEvents, UnknownLegacyActions: oapi.CountMap(p.UnknownActions),
		FullParity: p.FullParity, Notes: append([]string(nil), ParityNotes...)}
	o, r := ok()
	return oapi.GetCustodyParityReport200JSONResponse{Body: oapi.CustodyParityReportEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
