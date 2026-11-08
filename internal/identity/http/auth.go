package http

// R1 authentication handlers (apiserver.AuthAPI). They translate the
// identity session lifecycle to the contract and set or clear the session
// cookie. They never write a credential, never grant anything and never
// reveal why a login failed.

import (
	"context"
	"errors"

	"github.com/google/uuid"

	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/session"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// AuthHandlers serves login, logout and logout-all.
type AuthHandlers struct {
	svc   *session.Service
	clock timeutil.Clock
}

// NewAuth returns the handlers.
func NewAuth(svc *session.Service, clock timeutil.Clock) (*AuthHandlers, error) {
	if svc == nil || clock == nil {
		return nil, errors.New("identity http: session service and clock are required")
	}
	return &AuthHandlers{svc: svc, clock: clock}, nil
}

func sessionBody(p websession.Principal) (*oapi.AuthSession, error) {
	uid, err := uuid.Parse(p.UserID)
	if err != nil {
		return nil, err
	}
	return &oapi.AuthSession{UserId: uid, DisplayName: p.DisplayName, CsrfToken: p.CSRF, IdleExpiresAt: p.IdleExpiresAt.UTC(),
		AbsoluteExpiresAt: p.AbsoluteExpiresAt.UTC()}, nil
}

func envelope(ctx context.Context, s result.Status, codes ...reasoncodes.Code) oapi.AuthEnvelope {
	return oapi.AuthEnvelope{Outcome: authzhttp.Outcome(s), Reasons: authzhttp.Reasons(codes...), CorrelationId: httpx.CorrelationID(ctx)}
}

// LoginWithPassword verifies a credential and issues a session.
func (h *AuthHandlers) LoginWithPassword(ctx context.Context, req oapi.LoginWithPasswordRequestObject) (oapi.LoginWithPasswordResponseObject, error) {
	if req.Body == nil {
		return oapi.LoginWithPassword400ApplicationProblemPlusJSONResponse{ProblemApplicationProblemPlusJSONResponse: oapi.ProblemApplicationProblemPlusJSONResponse{Body: httpx.ProblemBadRequest.BodyCtx(ctx)}}, nil
	}
	replace := ""
	if p, ok := websession.FromContext(ctx); ok {
		replace = p.SessionID
	}
	r, err := h.svc.Login(ctx, session.Login{Name: req.Body.Login, Password: req.Body.Password, ReplaceSessionID: replace, CorrelationID: httpx.CorrelationID(ctx)})
	if err != nil {
		return nil, err
	}
	switch r.Status {
	case session.Disabled:
		return oapi.LoginWithPassword200JSONResponse{Body: envelope(ctx, result.Blocked, reasoncodes.FeatureDisabled)}, nil
	case session.Throttled:
		return oapi.LoginWithPassword429ApplicationProblemPlusJSONResponse{Body: httpx.ProblemThrottled.BodyCtx(ctx)}, nil
	case session.OK:
		body, err := sessionBody(r.Issued.Principal)
		if err != nil {
			return nil, err
		}
		env := envelope(ctx, result.Success)
		env.Reasons = []oapi.ReasonCode{}
		env.Session = body
		cookie := websession.Cookie(r.Issued.Token, r.Issued.Principal.AbsoluteExpiresAt, h.clock.Now().Time()).String()
		return oapi.LoginWithPassword200JSONResponse{Body: env, Headers: oapi.LoginWithPassword200ResponseHeaders{SetCookie: &cookie}}, nil
	default:
		// Wrong password, unknown login, disabled credential and absent
		// account are indistinguishable to the client.
		return oapi.LoginWithPassword401JSONResponse{Body: envelope(ctx, result.Unauthorized, reasoncodes.ActionDenied)}, nil
	}
}

// Logout revokes the presented session (idempotent) and clears the cookie.
func (h *AuthHandlers) Logout(ctx context.Context, _ oapi.LogoutRequestObject) (oapi.LogoutResponseObject, error) {
	if p, ok := websession.FromContext(ctx); ok {
		if _, err := h.svc.Logout(ctx, p, httpx.CorrelationID(ctx)); err != nil {
			return nil, err
		}
	}
	clear := websession.ClearCookie().String()
	env := envelope(ctx, result.Success)
	env.Reasons = []oapi.ReasonCode{}
	return oapi.Logout200JSONResponse{Body: env, Headers: oapi.Logout200ResponseHeaders{SetCookie: &clear}}, nil
}

// LogoutAll revokes every active session of the caller.
func (h *AuthHandlers) LogoutAll(ctx context.Context, _ oapi.LogoutAllRequestObject) (oapi.LogoutAllResponseObject, error) {
	p, ok := websession.FromContext(ctx)
	if !ok {
		return oapi.LogoutAll401JSONResponse{Body: envelope(ctx, result.Unauthorized, reasoncodes.ActionDenied)}, nil
	}
	if _, err := h.svc.LogoutAll(ctx, p, httpx.CorrelationID(ctx)); err != nil {
		return nil, err
	}
	clear := websession.ClearCookie().String()
	env := envelope(ctx, result.Success)
	env.Reasons = []oapi.ReasonCode{}
	return oapi.LogoutAll200JSONResponse{Body: env, Headers: oapi.LogoutAll200ResponseHeaders{SetCookie: &clear}}, nil
}

// LoginEnabled reports the identity.session_auth flag.
func (h *AuthHandlers) LoginEnabled(ctx context.Context) bool { return h.svc.LoginEnabled(ctx) }

// Authenticate implements websession.Authenticator.
func (h *AuthHandlers) Authenticate(ctx context.Context, token string) (websession.Principal, bool, error) {
	return h.svc.Authenticate(ctx, token)
}

// RecordCSRFRejection implements websession.Authenticator.
func (h *AuthHandlers) RecordCSRFRejection(ctx context.Context, p websession.Principal) error {
	return h.svc.RecordCSRFRejection(ctx, p)
}
