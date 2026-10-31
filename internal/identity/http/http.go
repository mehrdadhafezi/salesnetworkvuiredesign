// Package http adapts the identity read service to the generated OpenAPI
// strict server. Handlers only guard (flag + evaluator), call the read
// service and map rows to contract types: no SQL, no writes, no import
// command (architecture lint).
package http

import (
	"context"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// Handlers serves the identity read operations.
type Handlers struct {
	guard *authorization.Guard
	q     *query.Service
}

// New returns the handlers.
func New(guard *authorization.Guard, q *query.Service) *Handlers {
	return &Handlers{guard: guard, q: q}
}

func user(u store.User) oapi.UserSummary {
	return oapi.UserSummary{UserId: authzhttp.UUID(u.UserID), DisplayName: u.DisplayName,
		AccountState: oapi.UserSummaryAccountState(u.AccountState), LoginEnabled: oapi.UserSummaryLoginEnabled(u.LoginEnabled),
		Provenance: authzhttp.Provenance(u.Source)}
}

// ListIdentityUsers lists shadow accounts.
func (h *Handlers) ListIdentityUsers(ctx context.Context, req oapi.ListIdentityUsersRequestObject) (oapi.ListIdentityUsersResponseObject, error) {
	c, err := h.guard.Check(ctx, api.CapUserRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.ListIdentityUsers200JSONResponse{Body: oapi.UserListEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.ListIdentityUsers403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	p := query.ListParams{Cursor: req.Params.Cursor, Limit: req.Params.Limit}
	p.Filter.Query = req.Params.Q
	if req.Params.Presence != nil {
		s := string(*req.Params.Presence)
		p.Filter.Presence = &s
	}
	if req.Params.Sort != nil {
		p.Sort = string(*req.Params.Sort)
	}
	page, err := h.q.ListUsers(ctx, p)
	if errors.Is(err, query.ErrInvalidQuery) {
		return oapi.ListIdentityUsers400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: authzhttp.BadRequest(ctx, err.Error())}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]oapi.UserSummary, 0, len(page.Users))
	for _, u := range page.Users {
		items = append(items, user(u))
	}
	return oapi.ListIdentityUsers200JSONResponse{Body: oapi.UserListEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Items: &items, Page: &oapi.PageInfo{HasMore: page.NextCursor != nil, NextCursor: page.NextCursor},
		AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}

// GetIdentityUser returns one shadow account.
func (h *Handlers) GetIdentityUser(ctx context.Context, req oapi.GetIdentityUserRequestObject) (oapi.GetIdentityUserResponseObject, error) {
	c, err := h.guard.Check(ctx, api.CapUserRead)
	if err != nil {
		return nil, err
	}
	switch c.Gate {
	case authorization.GateDisabled:
		o, r := authzhttp.Disabled()
		return oapi.GetIdentityUser200JSONResponse{Body: oapi.UserEnvelope{Outcome: o, Reasons: r, CorrelationId: httpx.CorrelationID(ctx)}}, nil
	case authorization.GateDenied:
		return oapi.GetIdentityUser403JSONResponse{DeniedJSONResponse: authzhttp.Denied(ctx, c.Decision)}, nil
	}
	u, ok, err := h.q.GetUser(ctx, req.UserId.String())
	if err != nil {
		return nil, err
	}
	if !ok {
		return oapi.GetIdentityUser404ApplicationProblemPlusJSONResponse{Body: authzhttp.NotFound(ctx)}, nil
	}
	data := user(u)
	return oapi.GetIdentityUser200JSONResponse{Body: oapi.UserEnvelope{Outcome: authzhttp.Outcome(result.Success), Reasons: authzhttp.Reasons(),
		CorrelationId: httpx.CorrelationID(ctx), Data: &data, AllowedActions: authzhttp.AllowedActions(c.Decisions)}}, nil
}
