// Package http adapts the authorization evaluator to the generated OpenAPI
// strict server: the effective-access operation and the shared envelope
// helpers (outcome, reasons, allowed_actions) every Slice 4 read handler
// uses, so permission hints always come from the one evaluator.
package http

import (
	"context"
	"sort"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Outcome converts a status (never collapsing the seven).
func Outcome(s result.Status) oapi.Outcome {
	o, err := httpx.Outcome(s)
	if err != nil {
		return oapi.Outcome(result.Unknown.String())
	}
	return o
}

// Reasons converts reason codes.
func Reasons(codes ...reasoncodes.Code) []oapi.ReasonCode {
	out := make([]oapi.ReasonCode, 0, len(codes))
	for _, c := range codes {
		out = append(out, oapi.ReasonCode(c))
	}
	return out
}

// AllowedActions renders evaluator decisions as allowed_actions hints.
func AllowedActions(ds map[identityapi.Capability]authorization.Decision) *oapi.AllowedActions {
	out := oapi.AllowedActions{}
	for c, d := range ds {
		a := oapi.AllowedAction{Allowed: d.Allowed}
		if d.Reason != nil {
			r := oapi.ReasonCode(*d.Reason)
			a.ReasonCode = &r
		}
		out[string(c)] = a
	}
	return &out
}

// Denied is the UNAUTHORIZED envelope for a denied read.
func Denied(ctx context.Context, d authorization.Decision) oapi.DeniedJSONResponse {
	reasons := Reasons(reasoncodes.ActionDenied)
	if d.Reason != nil {
		reasons = Reasons(*d.Reason)
	}
	return oapi.DeniedJSONResponse{Body: oapi.ResultEnvelope{Outcome: Outcome(result.Unauthorized), Reasons: reasons, CorrelationId: httpx.CorrelationID(ctx)}}
}

// Disabled returns the BLOCKED / FEATURE_DISABLED outcome and reasons.
func Disabled() (oapi.Outcome, []oapi.ReasonCode) {
	return Outcome(result.Blocked), Reasons(reasoncodes.FeatureDisabled)
}

// BadRequest is the 400 problem body for an invalid query.
func BadRequest(ctx context.Context, detail string) oapi.ProblemApplicationProblemPlusJSONResponse {
	p := httpx.ProblemBadRequest
	p.Detail = detail
	return oapi.ProblemApplicationProblemPlusJSONResponse{Body: p.BodyCtx(ctx)}
}

// NotFound is the 404 problem body for an unknown id.
func NotFound(ctx context.Context) oapi.Problem { return httpx.ProblemNoRecord.BodyCtx(ctx) }

// Handlers serves the authorization operations.
type Handlers struct {
	guard     *authorization.Guard
	available authorization.Availability
}

// New returns the handlers. guard is the Slice 4 HR read guard: it resolves
// the caller and reports hr_read_enabled. available reports each
// capability's own feature availability (RM-ACCESS-01); nil makes every
// capability unavailable (fail closed).
func New(guard *authorization.Guard, available authorization.Availability) *Handlers {
	return &Handlers{guard: guard, available: available}
}

// GetEffectiveAccess evaluates every capability for the caller.
//
// Field meaning is unchanged; only the availability input is corrected
// (RM-ACCESS-01): allowed is true only when the caller holds the capability
// through a counting ACTIVE grant AND the feature that guards that
// capability is enabled. A capability whose own feature is off is BLOCKED /
// FEATURE_DISABLED with scope NONE. Before RM-ACCESS-01 the HR read flag
// (organization.hr_read_shadow) was applied to every capability, so one HR
// switch decided every module. hr_read_enabled still reports that HR flag.
func (h *Handlers) GetEffectiveAccess(ctx context.Context, _ oapi.GetEffectiveAccessRequestObject) (oapi.GetEffectiveAccessResponseObject, error) {
	hrEnabled, authenticated, ds, err := h.guard.Effective(ctx)
	if err != nil {
		return nil, err
	}
	avail := map[identityapi.Capability]bool{}
	if h.available != nil {
		if avail, err = h.available(ctx); err != nil {
			return nil, err
		}
	}
	data := oapi.EffectiveAccess{Authenticated: authenticated, HrReadEnabled: hrEnabled, Decisions: []oapi.AccessDecision{}}
	for _, c := range identityapi.Capabilities() {
		d := ds[c.Key]
		enabled := avail[c.Key]
		ad := oapi.AccessDecision{Capability: oapi.Capability(c.Key), Allowed: d.Allowed && enabled, Outcome: Outcome(d.Outcome),
			Scope: oapi.AccessDecisionScope(d.Scope), MatchedRoles: []string{}}
		if d.Reason != nil {
			r := oapi.ReasonCode(*d.Reason)
			ad.ReasonCode = &r
		}
		if !enabled {
			r := oapi.ReasonCode(reasoncodes.FeatureDisabled)
			ad.Outcome, ad.ReasonCode, ad.Scope = Outcome(result.Blocked), &r, oapi.AccessDecisionScope(authorization.ScopeNone)
		}
		for _, m := range d.Matched {
			ad.MatchedRoles = append(ad.MatchedRoles, string(m.Role))
		}
		sort.Strings(ad.MatchedRoles)
		data.Decisions = append(data.Decisions, ad)
	}
	return oapi.GetEffectiveAccess200JSONResponse{Body: oapi.EffectiveAccessEnvelope{
		Outcome: Outcome(result.Success), Reasons: Reasons(), CorrelationId: httpx.CorrelationID(ctx), Data: data}}, nil
}

// UUID converts a canonical native id read from the database.
func UUID(s string) openapi_types.UUID { return uuid.MustParse(s) }

// OptUUID converts an optional native id.
func OptUUID(s *string) *openapi_types.UUID {
	if s == nil {
		return nil
	}
	u := UUID(*s)
	return &u
}

// Provenance renders read-back provenance.
func Provenance(p identityapi.Provenance) oapi.Provenance {
	return oapi.Provenance{SourceSystem: p.System, SourceSpec: p.Spec, SourceKey: p.Key, BatchId: UUID(p.BatchID),
		ImportedAt: p.ImportedAt.UTC(), RefreshedAt: p.RefreshedAt.UTC(), Presence: oapi.Presence(p.Presence)}
}
