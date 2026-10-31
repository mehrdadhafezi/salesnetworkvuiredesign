// Package http adapts the customers read service to the generated OpenAPI
// strict server. Handlers guard (flag + evaluator), call the read service
// and map rows to contract types. Contact and identifier values are masked
// by the store; normalized values, names and staging rows never leave. No
// SQL, no writes, no import command (architecture lint).
package http

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/store"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the customers read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

// CustomersEnabled reports the Slice 5 flag (session navigation hint).
func (h *Handlers) CustomersEnabled(ctx context.Context) bool {
	on, err := h.guard.Enabled(ctx)
	return err == nil && on
}

func ok() (oapi.Outcome, []oapi.ReasonCode) {
	return authzhttp.Outcome(result.Success), authzhttp.Reasons()
}

func pageInfo(next *string) *oapi.PageInfo {
	return &oapi.PageInfo{HasMore: next != nil, NextCursor: next}
}

func prov(p store.Provenance) oapi.Provenance {
	return authzhttp.Provenance(identityapi.Provenance{System: p.System, Spec: p.Spec, Key: p.Key, BatchID: p.BatchID,
		ImportedAt: p.ImportedAt, RefreshedAt: p.RefreshedAt, Presence: p.Presence})
}

func customer(c store.Customer) oapi.CustomerSummary {
	return oapi.CustomerSummary{CustomerId: authzhttp.UUID(c.CustomerID), SubjectKind: oapi.CustomerSummarySubjectKind(c.SubjectKind),
		ResolutionState: oapi.ResolutionState(c.State), ResolutionReason: c.StateReason, NameMasked: c.NameMasked, ObservedRaw: c.ObservedRaw,
		ContactCount: int(c.ContactCount), IdentifierCount: int(c.IdentifierCount), EvidenceCount: int(c.EvidenceCount),
		OpenCaseCount: int(c.OpenCaseCount), ConflictCaseCount: int(c.ConflictCases), Provenance: prov(c.Source)}
}

func mergeCase(c store.Case) oapi.MergeCase {
	members := make([]openapi_types.UUID, 0, len(c.Members))
	for _, m := range c.Members {
		members = append(members, authzhttp.UUID(m))
	}
	kinds := make([]oapi.EvidenceKind, 0, len(c.EvidenceKinds))
	for _, k := range c.EvidenceKinds {
		kinds = append(kinds, oapi.EvidenceKind(k))
	}
	return oapi.MergeCase{MergeCaseId: authzhttp.UUID(c.ID), Reason: oapi.CaseReason(c.Reason), State: oapi.ResolutionState(c.State),
		StateReason: c.StateReason, MaxStrength: oapi.Strength(c.MaxStrength), MemberCustomerIds: members, EvidenceKinds: kinds,
		Presence: oapi.Presence(c.Presence), CreatedAt: c.CreatedAt, RefreshedAt: c.RefreshedAt}
}

func strp[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func idp(v *openapi_types.UUID) *string {
	if v == nil {
		return nil
	}
	s := v.String()
	return &s
}

// ListCustomers lists shadow customers.
func (h *Handlers) ListCustomers(ctx context.Context, req oapi.ListCustomersRequestObject) (oapi.ListCustomersResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapCustomerRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListCustomers200JSONResponse{Body: oapi.CustomerListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListCustomers403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	p := req.Params
	res, err := h.q.ListCustomers(ctx, strp(p.State), p.HasConflict, strp(p.Presence), p.Cursor, p.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListCustomers400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.CustomerSummary, 0, len(res.Customers))
	for _, x := range res.Customers {
		items = append(items, customer(x))
	}
	o, r := ok()
	return oapi.ListCustomers200JSONResponse{Body: oapi.CustomerListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: pageInfo(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetCustomer returns one customer with identifiers and candidate cases.
func (h *Handlers) GetCustomer(ctx context.Context, req oapi.GetCustomerRequestObject) (oapi.GetCustomerResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapCustomerRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetCustomer200JSONResponse{Body: oapi.CustomerEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetCustomer403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	d, found, err := h.q.GetCustomer(ctx, req.CustomerId.String())
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.GetCustomer404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	data := oapi.CustomerDetail{Customer: customer(d.Customer), Identifiers: []oapi.Identifier{}, MergeCases: []oapi.MergeCase{}}
	for _, i := range d.Identifiers {
		data.Identifiers = append(data.Identifiers, oapi.Identifier{IdentifierId: authzhttp.UUID(i.ID), Scheme: oapi.IdentifierScheme(i.Scheme),
			MaskedValue: i.Masked, Derivation: oapi.IdentifierDerivation(i.Derivation), Status: oapi.IdentifierStatus(i.Status), Provenance: prov(i.Source)})
	}
	// Candidate cases appear only when the queue capability is also allowed.
	if c.Can(identityapi.CapMergeCaseRead) {
		for _, k := range d.Cases {
			data.MergeCases = append(data.MergeCases, mergeCase(k))
		}
	}
	o, r := ok()
	return oapi.GetCustomer200JSONResponse{Body: oapi.CustomerEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCustomerContactPoints lists masked contact points.
func (h *Handlers) ListCustomerContactPoints(ctx context.Context, req oapi.ListCustomerContactPointsRequestObject) (oapi.ListCustomerContactPointsResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapCustomerRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListCustomerContactPoints200JSONResponse{Body: oapi.ContactPointListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListCustomerContactPoints403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	rows, found, err := h.q.Contacts(ctx, req.CustomerId.String())
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.ListCustomerContactPoints404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	items := make([]oapi.ContactPoint, 0, len(rows))
	for _, v := range rows {
		items = append(items, oapi.ContactPoint{ContactPointId: authzhttp.UUID(v.ID), Kind: oapi.ContactPointKind(v.Kind), Role: oapi.ContactPointRole(v.Role),
			MaskedValue: v.Masked, Normalization: oapi.ContactPointNormalization(v.Normalization), SharedWithCustomers: int(v.SharedWith),
			ValidFromRaw: v.ValidFromRaw, ValidToRaw: v.ValidToRaw, ObservedRaw: v.ObservedRaw, Provenance: prov(v.Source)})
	}
	o, r := ok()
	return oapi.ListCustomerContactPoints200JSONResponse{Body: oapi.ContactPointListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListCustomerEvidence lists evidence about a customer.
func (h *Handlers) ListCustomerEvidence(ctx context.Context, req oapi.ListCustomerEvidenceRequestObject) (oapi.ListCustomerEvidenceResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapCustomerRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListCustomerEvidence200JSONResponse{Body: oapi.EvidenceListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListCustomerEvidence403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	res, found, err := h.q.Evidence(ctx, req.CustomerId.String(), strp(req.Params.Strength), req.Params.Cursor, req.Params.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListCustomerEvidence400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.ListCustomerEvidence404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	items := make([]oapi.Evidence, 0, len(res.Evidence))
	for _, v := range res.Evidence {
		items = append(items, oapi.Evidence{EvidenceId: authzhttp.UUID(v.ID), Kind: oapi.EvidenceKind(v.Kind), Strength: oapi.Strength(v.Strength),
			Polarity: oapi.EvidencePolarity(v.Polarity), Origin: oapi.EvidenceOrigin(v.Origin), Reference: v.Reference, ObservedRaw: v.ObservedRaw,
			SubjectCount: int(v.SubjectCount), SourceSpec: v.Spec, Presence: oapi.Presence(v.Presence), RecordedAt: v.RecordedAt})
	}
	o, r := ok()
	return oapi.ListCustomerEvidence200JSONResponse{Body: oapi.EvidenceListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: pageInfo(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListMergeCases lists the read-only candidate queue.
func (h *Handlers) ListMergeCases(ctx context.Context, req oapi.ListMergeCasesRequestObject) (oapi.ListMergeCasesResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapMergeCaseRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListMergeCases200JSONResponse{Body: oapi.MergeCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListMergeCases403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	p := req.Params
	res, err := h.q.ListCases(ctx, store.CaseFilter{State: strp(p.State), Strength: strp(p.EvidenceStrength), Reason: strp(p.Reason),
		CustomerID: idp(p.CustomerId)}, p.Cursor, p.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListMergeCases400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.MergeCase, 0, len(res.Cases))
	for _, k := range res.Cases {
		items = append(items, mergeCase(k))
	}
	o, r := ok()
	return oapi.ListMergeCases200JSONResponse{Body: oapi.MergeCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: pageInfo(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetMergeCase returns one candidate.
func (h *Handlers) GetMergeCase(ctx context.Context, req oapi.GetMergeCaseRequestObject) (oapi.GetMergeCaseResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapMergeCaseRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetMergeCase200JSONResponse{Body: oapi.MergeCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetMergeCase403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	k, found, err := h.q.GetCase(ctx, req.MergeCaseId.String())
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.GetMergeCase404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	data := mergeCase(k)
	o, r := ok()
	return oapi.GetMergeCase200JSONResponse{Body: oapi.MergeCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func ints(m map[string]int64) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = int(v)
	}
	return out
}

// GetCustomerResolutionReport returns the resolution projection report.
func (h *Handlers) GetCustomerResolutionReport(ctx context.Context, _ oapi.GetCustomerResolutionReportRequestObject) (oapi.GetCustomerResolutionReportResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapIdentityReportRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetCustomerResolutionReport200JSONResponse{Body: oapi.ResolutionReportEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetCustomerResolutionReport403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	rep, err := h.q.ResolutionReport(ctx)
	if err != nil {
		return nil, err
	}
	data := oapi.ResolutionReport{Customers: int(rep.Customers), CustomersPresent: int(rep.CustomersPresent), SourceSubjectsPresent: int(rep.SubjectsPresent),
		CustomersByState: ints(rep.ByState), MergeCasesByState: ints(rep.CasesByState), MergeCasesByReason: ints(rep.CasesByReason),
		PhoneOnlyCandidateCount: int(rep.PhoneOnlyCandidates), OverlappingPhoneConflictCount: int(rep.OverlappingPhoneConflict),
		DuplicateExternalIdConflictCount: int(rep.DuplicateExternalID), StrongEvidenceCount: int(rep.StrongEvidence),
		WeakOnlyCustomerCount: int(rep.WeakOnlyCustomers), UnresolvedCount: int(rep.Unresolved), PhoneOnlyResolvedCount: int(rep.PhoneOnlyResolved),
		PhoneOnlyMergeCount: int(rep.PhoneOnlyMerges), MergeCount: int(rep.Merges)}
	// The invariants are part of the outcome: a violation is never SUCCESS.
	o, r := ok()
	if rep.PhoneOnlyResolved != 0 || rep.Merges != 0 || rep.PhoneOnlyMerges != 0 {
		o, r = authzhttp.Outcome(result.Blocked), authzhttp.Reasons(reasoncodes.LinkageConflict)
	}
	return oapi.GetCustomerResolutionReport200JSONResponse{Body: oapi.ResolutionReportEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
