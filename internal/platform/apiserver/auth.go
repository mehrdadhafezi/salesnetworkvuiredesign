package apiserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// EnvAllowedOrigins lists the exact browser origins allowed to send unsafe
// requests when R1 sessions are configured (comma separated). Required in
// production; each must be https.
const EnvAllowedOrigins = "CRM_AUTH_ALLOWED_ORIGINS"

// ErrAuthConflict: the dev stub and server-side sessions are mutually
// exclusive, so a synthetic principal can never coexist with a real one.
var ErrAuthConflict = errors.New("apiserver: the dev auth stub and R1 sessions cannot both be enabled")

// AuthAPI is the R1 authentication surface (identity, composed by
// cmd/crm-api). Without one, login is BLOCKED / FEATURE_DISABLED and every
// request is anonymous.
type AuthAPI interface {
	LoginWithPassword(ctx context.Context, request oapi.LoginWithPasswordRequestObject) (oapi.LoginWithPasswordResponseObject, error)
	Logout(ctx context.Context, request oapi.LogoutRequestObject) (oapi.LogoutResponseObject, error)
	LogoutAll(ctx context.Context, request oapi.LogoutAllRequestObject) (oapi.LogoutAllResponseObject, error)
	// LoginEnabled reports the identity.session_auth flag (navigation hint).
	LoginEnabled(ctx context.Context) bool
	websession.Authenticator
}

type unconfiguredAuth struct{}

func (unconfiguredAuth) LoginWithPassword(ctx context.Context, _ oapi.LoginWithPasswordRequestObject) (oapi.LoginWithPasswordResponseObject, error) {
	return oapi.LoginWithPassword200JSONResponse{Body: oapi.AuthEnvelope{Outcome: oapi.Outcome(result.Blocked.String()),
		Reasons: []oapi.ReasonCode{oapi.ReasonCode(reasoncodes.FeatureDisabled)}, CorrelationId: httpx.CorrelationID(ctx)}}, nil
}

func (unconfiguredAuth) Logout(ctx context.Context, _ oapi.LogoutRequestObject) (oapi.LogoutResponseObject, error) {
	clear := websession.ClearCookie().String()
	return oapi.Logout200JSONResponse{Body: oapi.AuthEnvelope{Outcome: oapi.Outcome(result.Success.String()), Reasons: []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx)}, Headers: oapi.Logout200ResponseHeaders{SetCookie: &clear}}, nil
}

func (unconfiguredAuth) LogoutAll(ctx context.Context, _ oapi.LogoutAllRequestObject) (oapi.LogoutAllResponseObject, error) {
	return oapi.LogoutAll401JSONResponse{Body: oapi.AuthEnvelope{Outcome: oapi.Outcome(result.Unauthorized.String()),
		Reasons: []oapi.ReasonCode{oapi.ReasonCode(reasoncodes.ActionDenied)}, CorrelationId: httpx.CorrelationID(ctx)}}, nil
}

func (unconfiguredAuth) LoginEnabled(context.Context) bool { return false }

func (unconfiguredAuth) Authenticate(context.Context, string) (websession.Principal, bool, error) {
	return websession.Principal{}, false, nil
}

func (unconfiguredAuth) RecordCSRFRejection(context.Context, websession.Principal) error { return nil }

// ParseAllowedOrigins validates CRM_AUTH_ALLOWED_ORIGINS. Production
// accepts only https origins; development and test also accept http
// loopback origins.
func ParseAllowedOrigins(raw string, env authstub.Environment) ([]string, error) {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		u, err := url.Parse(o)
		if err != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || o != u.Scheme+"://"+u.Host {
			return nil, fmt.Errorf("%w: %s entry %q is not an exact origin", ErrInvalidConfig, EnvAllowedOrigins, o)
		}
		loopback := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
		switch {
		case u.Scheme == "https":
		case u.Scheme == "http" && loopback && env != authstub.Production:
		default:
			return nil, fmt.Errorf("%w: %s entry %q must be https", ErrInvalidConfig, EnvAllowedOrigins, o)
		}
		out = append(out, o)
	}
	return out, nil
}

// sessionAuth resolves the session cookie on every request. A valid
// session puts a websession.Principal on the context; an invalid cookie is
// cleared in the response and the request continues anonymously. It never
// reads the dev stub header.
func sessionAuth(auth websession.Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			token, present := websession.TokenFrom(r)
			if !present {
				next.ServeHTTP(w, r.WithContext(websession.WithState(ctx, websession.StateNone)))
				return
			}
			p, ok, err := auth.Authenticate(ctx, token)
			if err != nil {
				// Never authenticate on error; the token is never logged.
				logger.Error("session lookup failed", "correlation_id", httpx.CorrelationID(ctx), "error", err)
			}
			if !ok || err != nil {
				w.Header().Add("Set-Cookie", websession.ClearCookie().String())
				next.ServeHTTP(w, r.WithContext(websession.WithState(ctx, websession.StateInvalid)))
				return
			}
			ctx = websession.WithState(websession.WithPrincipal(ctx, p), websession.StateActive)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// loginPath is the one unsafe route that runs before a CSRF token exists.
const loginPath = "/api/v1/auth/login"

func unsafeMethod(m string) bool {
	return m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions
}

// crossSiteGuard protects every unsafe request:
//  1. Sec-Fetch-Site: cross-site is refused;
//  2. a present Origin must be one of the allowed origins (or, when none is
//     configured outside production, the request's own host);
//  3. with an authenticated session, X-CSRF-Token must equal the
//     session-bound token (login is exempt: it has no token yet and is
//     covered by 1, 2 and SameSite=Strict).
//
// Anonymous unsafe requests carry no ambient authority and are left to the
// handlers (which deny them).
func crossSiteGuard(allowed []string, auth websession.Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !unsafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			reject := func(why string) {
				logger.Warn("cross-site request rejected", "correlation_id", httpx.CorrelationID(r.Context()), "check", why, "path", r.URL.Path)
				httpx.WriteProblem(w, r, httpx.ProblemCrossSite)
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				reject("sec-fetch-site")
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				own := ""
				if r.Host != "" {
					own = "://" + r.Host
				}
				if !slices.Contains(allowed, origin) && !(len(allowed) == 0 && own != "" && strings.HasSuffix(origin, own)) {
					reject("origin")
					return
				}
			}
			if p, ok := websession.FromContext(r.Context()); ok && r.URL.Path != loginPath {
				if !websession.CSRFValid(p.CSRF, r.Header.Get(websession.CSRFHeader)) {
					if err := auth.RecordCSRFRejection(r.Context(), p); err != nil {
						logger.Error("csrf rejection not recorded", "correlation_id", httpx.CorrelationID(r.Context()), "error", err)
					}
					reject("csrf-token")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
