package authorization

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

type loader struct {
	grants []identityapi.Grant
	err    error
	calls  int
}

func (l *loader) ActiveGrants(_ context.Context, _ string) ([]identityapi.Grant, error) {
	l.calls++
	return l.grants, l.err
}

const (
	userA = "11111111-1111-4111-8111-111111111111"
	userB = "22222222-2222-4222-8222-222222222222"
)

func sessionCtx(user string) context.Context {
	return websession.WithPrincipal(context.Background(), websession.Principal{SessionID: "33333333-3333-4333-8333-333333333333", UserID: user})
}

// No session → anonymous; the dev stub principal is never read.
func TestSessionSubjectsAnonymousAndNoDevFallback(t *testing.T) {
	l := &loader{}
	r := SessionSubjects{Grants: l}
	if s := r.Subject(context.Background()); s.PrincipalID != "" || s.UserID != "" || l.calls != 0 {
		t.Fatalf("no session: %+v", s)
	}
	// A synthetic dev principal on the context (as the stub would set) is ignored.
	p, _, err := mustStub(t).Resolve("dev-user-001")
	if err != nil {
		t.Fatal(err)
	}
	ctx := withStubPrincipal(t, p)
	if s := r.Subject(ctx); s.PrincipalID != "" || len(s.Grants) != 0 {
		t.Fatalf("dev principal leaked into the production resolver: %+v", s)
	}
	for _, c := range identityapi.Capabilities() {
		if d := Evaluate(Runtime, r.Subject(ctx), Request{Capability: c.Key}); d.Allowed {
			t.Fatalf("%s allowed without a session", c.Key)
		}
	}
}

// Authentication establishes identity, not authority.
func TestSessionSubjectsIdentityNotAuthority(t *testing.T) {
	r := SessionSubjects{Grants: &loader{}}
	s := r.Subject(sessionCtx(userA))
	if s.PrincipalID != userA || s.UserID != userA || len(s.Grants) != 0 {
		t.Fatalf("subject: %+v", s)
	}
	for _, c := range identityapi.Capabilities() {
		if d := Evaluate(Runtime, s, Request{Capability: c.Key}); d.Allowed || d.Outcome != result.Unauthorized {
			t.Fatalf("%s: an authenticated user without ACTIVE grants must be UNAUTHORIZED, got %+v", c.Key, d)
		}
	}
}

// Only the user's own ACTIVE, non-synthetic role grants count; HR authority
// never expands into Sales, Finance, commission or operations.
func TestSessionSubjectsGrantFiltering(t *testing.T) {
	l := &loader{grants: []identityapi.Grant{
		{UserID: userA, RoleKey: role("hr"), Status: identityapi.GrantActive, Basis: "TEST_GRANT"},
		// A loader bug must not smuggle these in:
		{UserID: userA, Status: identityapi.GrantActive, Basis: DevGrantBasis, Capabilities: []identityapi.Capability{identityapi.CapFinanceReviewRead}},
		{UserID: userA, RoleKey: role("finance"), Status: identityapi.GrantShadow, Basis: "LEGACY_PROFILE_ROLE"},
		{UserID: userB, RoleKey: role("finance"), Status: identityapi.GrantActive, Basis: "TEST_GRANT"},
		{UserID: userA, RoleKey: role("hr"), Status: identityapi.GrantActive, Basis: "TEST_GRANT", Capabilities: []identityapi.Capability{identityapi.CapPaymentBatchSubmit}},
	}}
	s := SessionSubjects{Grants: l}.Subject(sessionCtx(userA))
	if len(s.Grants) != 2 {
		t.Fatalf("grants kept: %+v", s.Grants)
	}
	for _, g := range s.Grants {
		if g.Basis == DevGrantBasis || len(g.Capabilities) != 0 || g.UserID != userA {
			t.Fatalf("grant kept: %+v", g)
		}
	}
	allowed := map[identityapi.Capability]bool{}
	for c, d := range EvaluateAll(Runtime, s) {
		if d.Allowed {
			allowed[c] = true
		}
	}
	for c := range allowed {
		if !strings.HasPrefix(string(c), "org.") && !strings.HasPrefix(string(c), "identity.") {
			t.Fatalf("HR grant reached %s", c)
		}
	}
	if !allowed[identityapi.CapPersonRead] {
		t.Fatal("HR read capability missing")
	}
	for _, c := range []identityapi.Capability{identityapi.CapFinanceReviewRead, identityapi.CapFinanceLedgerRead, identityapi.CapPaymentBatchSubmit,
		identityapi.CapCaseRead, identityapi.CapContactOutcomeRecord, identityapi.CapCommissionShadowRead, identityapi.CapWorkAssign, identityapi.CapHRRequestReview} {
		if allowed[c] {
			t.Fatalf("HR expanded into %s", c)
		}
	}
}

// A grant loading failure never authenticates authority: it fails closed.
func TestSessionSubjectsLoaderFailureFailsClosed(t *testing.T) {
	s := SessionSubjects{Grants: &loader{err: errors.New("db down")}}.Subject(sessionCtx(userA))
	if d := Evaluate(Runtime, s, Request{Capability: identityapi.CapPersonRead}); d.Allowed || d.Outcome != result.Blocked {
		t.Fatalf("loader failure: %+v", d)
	}
}

// Finance approval and posting are not capabilities anyone can hold: no
// approve / reject / post / credit / settle / refund capability exists.
func TestNoFinancialExecutionCapability(t *testing.T) {
	for _, c := range identityapi.Capabilities() {
		k := string(c.Key)
		for _, verb := range []string{"approve", "reject", "post", "credit", "settle", "refund", "charge", "activate"} {
			if strings.Contains(k, "."+verb) {
				t.Fatalf("financial execution capability %s exists", k)
			}
		}
	}
}

func mustStub(t *testing.T) *authstub.Stub {
	t.Helper()
	s, err := authstub.New(authstub.Test)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// withStubPrincipal returns a context carrying the dev stub's principal,
// exactly as the stub middleware would attach it.
func withStubPrincipal(t *testing.T, p authstub.Principal) context.Context {
	t.Helper()
	var got context.Context
	h := mustStub(t).Middleware(func(http.ResponseWriter, *http.Request, error) { t.Fatal("stub refused") })(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.Context() }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(authstub.Header, p.ID)
	h.ServeHTTP(httptest.NewRecorder(), req)
	if q, ok := authstub.FromContext(got); !ok || q.ID != p.ID {
		t.Fatal("stub principal not attached")
	}
	return got
}
