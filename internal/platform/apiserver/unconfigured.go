package apiserver

import (
	"context"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// unconfigured is the Slice 4 read surface when no module is wired (no
// read databases configured): every operation is BLOCKED with
// FEATURE_DISABLED, never an empty success, and nothing is evaluated as
// allowed.
type unconfigured struct{}

var _ ReadAPI = unconfigured{}

func blocked(ctx context.Context) (oapi.Outcome, []oapi.ReasonCode, string) {
	return oapi.Outcome(result.Blocked.String()), []oapi.ReasonCode{oapi.ReasonCode(reasoncodes.FeatureDisabled)}, httpx.CorrelationID(ctx)
}

func (unconfigured) HRReadEnabled(context.Context) bool { return false }

func (unconfigured) GetEffectiveAccess(ctx context.Context, _ oapi.GetEffectiveAccessRequestObject) (oapi.GetEffectiveAccessResponseObject, error) {
	data := oapi.EffectiveAccess{Decisions: []oapi.AccessDecision{}}
	reason := oapi.ReasonCode(reasoncodes.FeatureDisabled)
	for _, c := range []oapi.Capability{oapi.IdentityUserRead, oapi.OrgPersonRead, oapi.OrgEmploymentRead, oapi.OrgPositionRead, oapi.OrgUnitRead, oapi.OrgAssignmentRead} {
		data.Decisions = append(data.Decisions, oapi.AccessDecision{Capability: c, Outcome: oapi.Outcome(result.Blocked.String()),
			ReasonCode: &reason, Scope: oapi.AccessDecisionScopeNONE, MatchedRoles: []string{}})
	}
	return oapi.GetEffectiveAccess200JSONResponse{Body: oapi.EffectiveAccessEnvelope{Outcome: oapi.Outcome(result.Success.String()),
		Reasons: []oapi.ReasonCode{}, CorrelationId: httpx.CorrelationID(ctx), Data: data}}, nil
}

func (unconfigured) ListIdentityUsers(ctx context.Context, _ oapi.ListIdentityUsersRequestObject) (oapi.ListIdentityUsersResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListIdentityUsers200JSONResponse{Body: oapi.UserListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) GetIdentityUser(ctx context.Context, _ oapi.GetIdentityUserRequestObject) (oapi.GetIdentityUserResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetIdentityUser200JSONResponse{Body: oapi.UserEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationPersons(ctx context.Context, _ oapi.ListOrganizationPersonsRequestObject) (oapi.ListOrganizationPersonsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationPersons200JSONResponse{Body: oapi.PersonListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) GetOrganizationPerson(ctx context.Context, _ oapi.GetOrganizationPersonRequestObject) (oapi.GetOrganizationPersonResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetOrganizationPerson200JSONResponse{Body: oapi.PersonEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationPositions(ctx context.Context, _ oapi.ListOrganizationPositionsRequestObject) (oapi.ListOrganizationPositionsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationPositions200JSONResponse{Body: oapi.LookupListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationLevels(ctx context.Context, _ oapi.ListOrganizationLevelsRequestObject) (oapi.ListOrganizationLevelsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationLevels200JSONResponse{Body: oapi.LookupListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationOrgUnits(ctx context.Context, _ oapi.ListOrganizationOrgUnitsRequestObject) (oapi.ListOrganizationOrgUnitsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationOrgUnits200JSONResponse{Body: oapi.OrgUnitListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationAssignments(ctx context.Context, _ oapi.ListOrganizationAssignmentsRequestObject) (oapi.ListOrganizationAssignmentsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationAssignments200JSONResponse{Body: oapi.AssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationHierarchyFindings(ctx context.Context, _ oapi.ListOrganizationHierarchyFindingsRequestObject) (oapi.ListOrganizationHierarchyFindingsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationHierarchyFindings200JSONResponse{Body: oapi.FindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}
