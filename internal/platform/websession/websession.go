// Package websession is the HTTP-level session vocabulary of R1: the session
// cookie, the authenticated principal carried on the request context, and the
// session-bound CSRF token. It knows no business module and stores nothing;
// the identity module owns sessions and implements Authenticator.
//
// Controls:
//   - the session token is 256 bits from crypto/rand, base64url, opaque;
//     only identity stores it, and only as a SHA-256 digest;
//   - the cookie is "__Host-crm_session": Secure, HttpOnly, SameSite=Strict,
//     Path=/, no Domain (the __Host- prefix makes browsers enforce all three
//     of Secure, Path=/ and no Domain);
//   - the CSRF token is HMAC-SHA256(session token, "crm-csrf-v1"): bound to
//     the session, recomputed by the server on every request, never stored,
//     and unforgeable without the HttpOnly cookie;
//   - this package never logs, and its errors never contain, a token.
package websession

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"
)

// CookieName is the session cookie (the __Host- prefix is enforced by browsers).
const CookieName = "__Host-crm_session"

// CSRFHeader carries the CSRF token on unsafe requests.
const CSRFHeader = "X-CSRF-Token"

// TokenBytes is the random size of a session token.
const TokenBytes = 32

var enc = base64.RawURLEncoding

// NewToken returns a fresh opaque session token.
func NewToken() (string, error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return enc.EncodeToString(b), nil
}

// Digest returns the stored representation of a token, and false when the
// token is not a well-formed session token (a malformed or foreign value is
// never looked up).
func Digest(token string) ([32]byte, bool) {
	if len(token) != enc.EncodedLen(TokenBytes) {
		return [32]byte{}, false
	}
	raw, err := enc.DecodeString(token)
	if err != nil || len(raw) != TokenBytes {
		return [32]byte{}, false
	}
	return sha256.Sum256(raw), true
}

// CSRFToken derives the CSRF token bound to a session token.
func CSRFToken(sessionToken string) string {
	m := hmac.New(sha256.New, []byte(sessionToken))
	m.Write([]byte("crm-csrf-v1"))
	return enc.EncodeToString(m.Sum(nil))
}

// CSRFValid compares a presented CSRF token with the expected one in
// constant time.
func CSRFValid(expected, presented string) bool {
	if expected == "" || presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(presented)) == 1
}

// Principal is an authenticated session on the request context.
type Principal struct {
	SessionID         string
	UserID            string
	DisplayName       string
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	// CSRF is the expected CSRF token for this request's session.
	CSRF string
}

type principalKey struct{}
type stateKey struct{}

// State is what the session middleware concluded about a request.
type State string

const (
	// StateNone: no session cookie was presented.
	StateNone State = "none"
	// StateActive: a valid session was presented.
	StateActive State = "active"
	// StateInvalid: a cookie was presented but is malformed, unknown,
	// expired or revoked (deliberately not distinguished to the client).
	StateInvalid State = "invalid"
)

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// FromContext returns the request's session principal, if any.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// WithState records the middleware's conclusion.
func WithState(ctx context.Context, s State) context.Context {
	return context.WithValue(ctx, stateKey{}, s)
}

// StateFrom returns the recorded state (StateNone when absent).
func StateFrom(ctx context.Context) State {
	if s, ok := ctx.Value(stateKey{}).(State); ok {
		return s
	}
	return StateNone
}

// Cookie returns the session cookie for token, expiring with the session.
func Cookie(token string, absoluteExpiry, now time.Time) *http.Cookie {
	maxAge := int(absoluteExpiry.Sub(now).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	return &http.Cookie{Name: CookieName, Value: token, Path: "/", MaxAge: maxAge, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}

// ClearCookie removes the session cookie in the browser.
func ClearCookie() *http.Cookie {
	return &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}

// TokenFrom returns the presented session token, if any. Only the first
// cookie of that name counts.
func TokenFrom(r *http.Request) (string, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

// Authenticator resolves a presented session token. It is implemented by
// the identity module and composed by cmd/crm-api. ok=false means the token
// is not an active session (malformed, unknown, expired, revoked, disabled
// credential, absent account, or session authentication switched off).
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (p Principal, ok bool, err error)
	// RecordCSRFRejection appends a CSRF_REJECTED event for an
	// authenticated request that failed the CSRF check.
	RecordCSRFRejection(ctx context.Context, p Principal) error
}
