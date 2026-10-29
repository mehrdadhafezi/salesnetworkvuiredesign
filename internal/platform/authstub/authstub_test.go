package authstub

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseEnvironmentFailsClosed(t *testing.T) {
	cases := map[string]Environment{"": Production, "production": Production, "development": Development, "test": Test}
	for in, want := range cases {
		got, err := ParseEnvironment(in)
		if err != nil || got != want {
			t.Fatalf("ParseEnvironment(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"dev", "prod", "Production", "staging", "local"} {
		got, err := ParseEnvironment(bad)
		if !errors.Is(err, ErrUnknownEnvironment) || got != Production {
			t.Fatalf("ParseEnvironment(%q) = %q, %v; unknown must fail and default to production", bad, got, err)
		}
	}
}

func TestStubRefusedOutsideDevAndTest(t *testing.T) {
	for _, env := range []Environment{Production, "", "staging"} {
		if s, err := New(env); !errors.Is(err, ErrProductionLockout) || s != nil {
			t.Fatalf("New(%q) = %v, %v; must be locked out", env, s, err)
		}
	}
	for _, env := range []Environment{Development, Test} {
		if _, err := New(env); err != nil {
			t.Fatalf("New(%q): %v", env, err)
		}
	}
}

func TestSyntheticIdentities(t *testing.T) {
	ids := Identities()
	if strings.Join(ids, ",") != "dev-user-001,dev-user-002" {
		t.Fatalf("identities = %v", ids)
	}
	s, _ := New(Test)
	for _, id := range ids {
		p, ok, err := s.Resolve(id)
		if err != nil || !ok || p.ID != id || !strings.HasPrefix(p.DisplayName, "Dev ") {
			t.Fatalf("Resolve(%s) = %+v, %v, %v", id, p, ok, err)
		}
		for _, h := range p.CapabilityHints {
			if !strings.HasPrefix(h, "dev.") {
				t.Fatalf("hint %q must be a synthetic dev.* hint", h)
			}
		}
	}
	if _, ok, err := s.Resolve(""); ok || err != nil {
		t.Fatal("empty header must be anonymous")
	}
	if _, _, err := s.Resolve("admin"); !errors.Is(err, ErrUnknownIdentity) {
		t.Fatalf("unknown identity err = %v", err)
	}
	// Mutating a returned principal never changes the registry.
	p, _, _ := s.Resolve("dev-user-001")
	p.CapabilityHints[0] = "tampered"
	again, _, _ := s.Resolve("dev-user-001")
	if again.CapabilityHints[0] == "tampered" {
		t.Fatal("registry must not be mutable through a principal")
	}
}

func TestMiddleware(t *testing.T) {
	s, _ := New(Development)
	var seen string
	h := s.Middleware(func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadRequest)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := FromContext(r.Context()); ok {
			seen = p.ID
		} else {
			seen = "anonymous"
		}
	}))
	for header, want := range map[string]string{"": "anonymous", "dev-user-002": "dev-user-002"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set(Header, header)
		}
		h.ServeHTTP(httptest.NewRecorder(), req)
		if seen != want {
			t.Fatalf("header %q → %q", header, seen)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(Header, "real.person")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown identity status = %d", rec.Code)
	}
}
