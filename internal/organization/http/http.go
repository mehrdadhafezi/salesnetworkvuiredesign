// Package http adapts the organization read service to the generated
// OpenAPI strict server. Handlers guard (flag + evaluator), call the read
// service and map rows to contract types with per-capability field masking.
// They hold no SQL, write nothing and never reach an import command
// (architecture lint).
package http

import (
	"context"
	"errors"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the organization read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
	kinds map[string]bool
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	kinds := map[string]bool{}
	for _, k := range domain.FindingKinds() {
		kinds[k] = true
	}
	return &Handlers{guard: guard, q: q, kinds: kinds}
}

func ok() (oapi.Outcome, []oapi.ReasonCode) {
	return authzhttp.Outcome(result.Success), authzhttp.Reasons()
}

func page(next *string) *oapi.PageInfo { return &oapi.PageInfo{HasMore: next != nil, NextCursor: next} }

func str(p *oapi.NativeID) *string {
	if p == nil {
		return nil
	}
	s := p.String()
	return &s
}

func date(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

func int32p(p *int32) *int {
	if p == nil {
		return nil
	}
	n := int(*p)
	return &n
}

func personRef(r *store.PersonRef) *oapi.PersonRef {
	if r == nil {
		return nil
	}
	return &oapi.PersonRef{PersonId: authzhttp.UUID(r.PersonID), EmploymentId: authzhttp.UUID(r.EmploymentID), DisplayName: r.DisplayName}
}

func lookupRef(r *store.LookupRef) *oapi.LookupRef {
	if r == nil {
		return nil
	}
	return &oapi.LookupRef{Id: authzhttp.UUID(r.ID), Slug: r.Slug, Label: r.Label}
}

func unitRef(r *store.UnitRef) *oapi.UnitRef {
	if r == nil {
		return nil
	}
	return &oapi.UnitRef{Id: authzhttp.UUID(r.ID), Kind: oapi.OrgUnitKind(r.Kind), Label: r.Label}
}

func refState(s string) *oapi.RefState { st := oapi.RefState(s); return &st }

func findingKinds(ks []string) []oapi.FindingKind {
	out := make([]oapi.FindingKind, 0, len(ks))
	for _, k := range ks {
		out = append(out, oapi.FindingKind(k))
	}
	return out
}

// person maps one person with field masking by the evaluated capabilities.
func person(p store.Person, c authorization.Check) oapi.PersonSummary {
	out := oapi.PersonSummary{PersonId: authzhttp.UUID(p.PersonID), DisplayName: p.DisplayName, FindingCount: int(p.FindingCount),
		Provenance: authzhttp.Provenance(p.Source)}
	if c.Can(identityapi.CapEmploymentRead) {
		e := p.Employment
		emp := oapi.Employment{EmploymentId: authzhttp.UUID(e.EmploymentID), Status: oapi.EmploymentStatus(e.Status), StatusRaw: e.StatusRaw,
			TypeRaw: e.TypeRaw, EmployeeCode: e.EmployeeCode, HireDate: date(e.HireDate), HireDateRaw: e.HireDateRaw,
			TerminationDate: date(e.TerminationDate), TerminationDateRaw: e.TerminationDateRaw, IsActive: e.IsActive,
			Provenance: authzhttp.Provenance(e.Source)}
		emp.User.UserId, emp.User.LinkState = authzhttp.OptUUID(e.UserID), oapi.UserLinkState(e.UserLink)
		if c.Can(identityapi.CapPositionRead) {
			emp.Position, emp.PositionState = lookupRef(e.Position), refState(e.PositionState)
			emp.Level, emp.LevelState = lookupRef(e.Level), refState(e.LevelState)
		}
		if c.Can(identityapi.CapUnitRead) {
			emp.Department, emp.DepartmentState = unitRef(e.Department), refState(e.DepartmentState)
			emp.Unit, emp.UnitState = unitRef(e.Unit), refState(e.UnitState)
			emp.Team, emp.TeamState = unitRef(e.Team), refState(e.TeamState)
		}
		out.Employment = &emp
	}
	if c.Can(identityapi.CapAssignmentRead) {
		ms := oapi.ManagerState(p.ManagerState)
		out.ManagerState, out.Manager = &ms, personRef(p.Manager)
	}
	return out
}

func assignment(a store.Assignment) oapi.Assignment {
	out := oapi.Assignment{AssignmentId: authzhttp.UUID(a.AssignmentID),
		Child:            oapi.AssignmentEnd{State: oapi.AssignmentEndState(a.ChildState), Person: personRef(a.Child)},
		Parent:           oapi.AssignmentEnd{State: oapi.AssignmentEndState(a.ParentState), Person: personRef(a.Parent)},
		RelationshipKind: oapi.AssignmentRelationshipKind(a.Kind), RelationshipRaw: a.KindRaw, EffectiveFromRaw: a.EffectiveFromRaw,
		EffectiveToRaw: a.EffectiveToRaw, IntervalZone: oapi.AssignmentIntervalZoneUNKNOWN, IsCurrent: a.IsCurrent, CurrentRaw: a.CurrentRaw,
		FindingKinds: findingKinds(a.FindingKinds), Provenance: authzhttp.Provenance(a.Source)}
	return out
}

func finding(f store.Finding) oapi.Finding {
	return oapi.Finding{FindingId: authzhttp.UUID(f.FindingID), Kind: oapi.FindingKind(f.Kind), Severity: oapi.FindingSeverity(f.Severity),
		SubjectType: oapi.FindingSubject(f.SubjectType), SubjectId: authzhttp.OptUUID(f.SubjectID), SubjectSourceKey: f.SubjectKey,
		RelatedId: authzhttp.OptUUID(f.RelatedID), RelatedSourceKey: f.RelatedKey, Detail: f.Detail}
}

func lookup(l store.Lookup) oapi.Lookup {
	return oapi.Lookup{Id: authzhttp.UUID(l.ID), Slug: l.Slug, Label: l.Label, Description: l.Description, SortOrder: int32p(l.SortOrder),
		IsActive: l.IsActive, IsSystem: l.IsSystem, Provenance: authzhttp.Provenance(l.Source)}
}

// ListOrganizationPersons lists persons.
func (h *Handlers) ListOrganizationPersons(ctx context.Context, req oapi.ListOrganizationPersonsRequestObject) (oapi.ListOrganizationPersonsResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPersonRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListOrganizationPersons200JSONResponse{Body: oapi.PersonListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListOrganizationPersons403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	pr := req.Params
	p := query.PersonsParams{Cursor: pr.Cursor, Limit: pr.Limit}
	p.Filter = store.PersonFilter{Query: pr.Q, PositionID: str(pr.PositionId), LevelID: str(pr.LevelId), OrgUnitID: str(pr.OrgUnitId),
		IsActive: pr.IsActive, HasFindings: pr.HasFindings}
	if pr.EmploymentStatus != nil {
		s := string(*pr.EmploymentStatus)
		p.Filter.Status = &s
	}
	if pr.Sort != nil {
		p.Sort = string(*pr.Sort)
	}
	// Filters on masked fields are refused rather than silently answering.
	if (p.Filter.Status != nil || p.Filter.IsActive != nil) && !c.Can(identityapi.CapEmploymentRead) ||
		(p.Filter.PositionID != nil || p.Filter.LevelID != nil) && !c.Can(identityapi.CapPositionRead) ||
		p.Filter.OrgUnitID != nil && !c.Can(identityapi.CapUnitRead) || p.Filter.HasFindings != nil && !c.Can(identityapi.CapAssignmentRead) {
		return oapi.ListOrganizationPersons403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, authorization.Decision{})}, nil
	}
	res, err := h.q.ListPersons(ctx, p)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListOrganizationPersons400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.PersonSummary, 0, len(res.Persons))
	for _, x := range res.Persons {
		items = append(items, person(x, c))
	}
	o, r := ok()
	return oapi.ListOrganizationPersons200JSONResponse{Body: oapi.PersonListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetOrganizationPerson returns one person.
func (h *Handlers) GetOrganizationPerson(ctx context.Context, req oapi.GetOrganizationPersonRequestObject) (oapi.GetOrganizationPersonResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPersonRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetOrganizationPerson200JSONResponse{Body: oapi.PersonEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetOrganizationPerson403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	d, found, err := h.q.GetPerson(ctx, req.PersonId.String())
	if err != nil {
		return nil, err
	}
	if !found {
		return oapi.GetOrganizationPerson404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	data := oapi.PersonDetail{Person: person(d.Person, c), Assignments: []oapi.Assignment{}, DirectReports: []oapi.PersonRef{},
		History: []oapi.HistoryEntry{}, Findings: []oapi.Finding{}}
	if c.Can(identityapi.CapAssignmentRead) {
		for _, a := range d.Assignments {
			data.Assignments = append(data.Assignments, assignment(a))
		}
		for _, r := range d.DirectReports {
			data.DirectReports = append(data.DirectReports, *personRef(&r))
		}
		for _, hh := range d.History {
			data.History = append(data.History, oapi.HistoryEntry{HistoryId: authzhttp.UUID(hh.HistoryID), OldParent: personRef(hh.OldParent),
				NewParent: personRef(hh.NewParent), ActionRaw: hh.ActionRaw, RecordedRaw: hh.RecordedRaw, Provenance: authzhttp.Provenance(hh.Source)})
		}
		for _, f := range d.Findings {
			data.Findings = append(data.Findings, finding(f))
		}
	}
	o, r := ok()
	return oapi.GetOrganizationPerson200JSONResponse{Body: oapi.PersonEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

func (h *Handlers) lookups(ctx context.Context, which string, cursor *string, limit *int) (oapi.LookupListEnvelope, int, error) {
	c, err := h.guard.Check(ctx, identityapi.CapPositionRead)
	if err != nil {
		return oapi.LookupListEnvelope{}, 0, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.LookupListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}, 200, nil
	case authorization.GateDenied:
		return oapi.LookupListEnvelope{}, 403, nil
	}
	res, err := h.q.ListLookups(ctx, which, cursor, limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.LookupListEnvelope{}, 400, err
	}
	if err != nil {
		return oapi.LookupListEnvelope{}, 0, err
	}
	items := make([]oapi.Lookup, 0, len(res.Items))
	for _, l := range res.Items {
		items = append(items, lookup(l))
	}
	o, r := ok()
	return oapi.LookupListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx), Items: &items, Page: page(res.NextCursor),
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}, 200, nil
}

// ListOrganizationPositions lists positions.
func (h *Handlers) ListOrganizationPositions(ctx context.Context, req oapi.ListOrganizationPositionsRequestObject) (oapi.ListOrganizationPositionsResponseObject, error) {
	env, code, err := h.lookups(ctx, "position", req.Params.Cursor, req.Params.Limit)
	switch {
	case code == 400:
		return oapi.ListOrganizationPositions400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case err != nil:
		return nil, err
	case code == 403:
		return oapi.ListOrganizationPositions403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, authorization.Decision{})}, nil
	}
	return oapi.ListOrganizationPositions200JSONResponse{Body: env}, nil
}

// ListOrganizationLevels lists levels.
func (h *Handlers) ListOrganizationLevels(ctx context.Context, req oapi.ListOrganizationLevelsRequestObject) (oapi.ListOrganizationLevelsResponseObject, error) {
	env, code, err := h.lookups(ctx, "level", req.Params.Cursor, req.Params.Limit)
	switch {
	case code == 400:
		return oapi.ListOrganizationLevels400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	case err != nil:
		return nil, err
	case code == 403:
		return oapi.ListOrganizationLevels403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, authorization.Decision{})}, nil
	}
	return oapi.ListOrganizationLevels200JSONResponse{Body: env}, nil
}

// ListOrganizationOrgUnits lists the org unit tree.
func (h *Handlers) ListOrganizationOrgUnits(ctx context.Context, req oapi.ListOrganizationOrgUnitsRequestObject) (oapi.ListOrganizationOrgUnitsResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapUnitRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListOrganizationOrgUnits200JSONResponse{Body: oapi.OrgUnitListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListOrganizationOrgUnits403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	var kind *string
	if req.Params.Kind != nil {
		k := string(*req.Params.Kind)
		kind = &k
	}
	res, err := h.q.ListOrgUnits(ctx, kind, req.Params.Cursor, req.Params.Limit)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListOrganizationOrgUnits400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.OrgUnit, 0, len(res.Units))
	for _, u := range res.Units {
		items = append(items, oapi.OrgUnit{OrgUnitId: authzhttp.UUID(u.OrgUnitID), Kind: oapi.OrgUnitKind(u.Kind), Slug: u.Slug, Label: u.Label,
			SortOrder: int32p(u.SortOrder), IsActive: u.IsActive, ParentId: authzhttp.OptUUID(u.ParentID), ParentState: oapi.OrgUnitParentState(u.ParentState),
			MemberCount: int(u.MemberCount), FindingKinds: findingKinds(u.FindingKinds), Provenance: authzhttp.Provenance(u.Source)})
	}
	o, r := ok()
	return oapi.ListOrganizationOrgUnits200JSONResponse{Body: oapi.OrgUnitListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListOrganizationAssignments lists reporting assignments.
func (h *Handlers) ListOrganizationAssignments(ctx context.Context, req oapi.ListOrganizationAssignmentsRequestObject) (oapi.ListOrganizationAssignmentsResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapAssignmentRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListOrganizationAssignments200JSONResponse{Body: oapi.AssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListOrganizationAssignments403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	pr := req.Params
	f := store.AssignmentFilter{PersonID: str(pr.PersonId), Current: pr.IsCurrent, HasFindings: pr.HasFindings}
	if pr.Relation != nil {
		s := string(*pr.Relation)
		f.Role = &s
	}
	if pr.RelationshipKind != nil {
		s := string(*pr.RelationshipKind)
		f.Kind = &s
	}
	res, err := h.q.ListAssignments(ctx, query.AssignmentsParams{Filter: f, Cursor: pr.Cursor, Limit: pr.Limit})
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListOrganizationAssignments400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.Assignment, 0, len(res.Assignments))
	for _, a := range res.Assignments {
		items = append(items, assignment(a))
	}
	o, r := ok()
	return oapi.ListOrganizationAssignments200JSONResponse{Body: oapi.AssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// ListOrganizationHierarchyFindings lists hierarchy findings.
func (h *Handlers) ListOrganizationHierarchyFindings(ctx context.Context, req oapi.ListOrganizationHierarchyFindingsRequestObject) (oapi.ListOrganizationHierarchyFindingsResponseObject, error) {
	c, err := h.guard.Check(ctx, identityapi.CapAssignmentRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListOrganizationHierarchyFindings200JSONResponse{Body: oapi.FindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListOrganizationHierarchyFindings403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	pr := req.Params
	f := store.FindingFilter{PersonID: str(pr.PersonId)}
	if pr.Kind != nil {
		s := string(*pr.Kind)
		f.Kind = &s
	}
	if pr.Severity != nil {
		s := string(*pr.Severity)
		f.Severity = &s
	}
	if pr.SubjectType != nil {
		s := string(*pr.SubjectType)
		f.SubjectType = &s
	}
	res, err := h.q.ListFindings(ctx, query.FindingsParams{Filter: f, Cursor: pr.Cursor, Limit: pr.Limit}, h.kinds)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListOrganizationHierarchyFindings400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.Finding, 0, len(res.Findings))
	for _, x := range res.Findings {
		items = append(items, finding(x))
	}
	o, r := ok()
	return oapi.ListOrganizationHierarchyFindings200JSONResponse{Body: oapi.FindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx),
		Items: &items, Page: page(res.NextCursor), AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
