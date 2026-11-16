package main

// RM-ACCESS-01 real API + disposable PostgreSQL regression: the production
// composition (NewReadAPI, NewCustomersAPI, the flag store and apiserver)
// with the DEV/TEST stub's synthetic subjects. Nothing here creates a grant:
// dev-user-001's synthetic capabilities live in code (authorization.
// DevSubjects) and are refused in production (case K).

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

type accessStack struct {
	h     http.Handler
	flags *db.Runner
	admin *pgx.Conn
}

func newAccessStack(t *testing.T) *accessStack {
	t.Helper()
	srcs := []migrations.Source{identitymigrations.Source(), organizationmigrations.Source(), customersmigrations.Source()}
	e := pgtest.NewWith(t, srcs...)
	ctx := context.Background()
	for _, s := range srcs {
		if _, err := migrations.Up(ctx, e.DSN, s); err != nil {
			t.Fatalf("migrations: %v", err)
		}
	}
	flags := e.Runner(t, owner.FeatureFlags)
	read, err := NewReadAPI(e.Runner(t, identity.Module), e.Runner(t, organization.Module), flags, true)
	if err != nil {
		t.Fatal(err)
	}
	cust, err := NewCustomersAPI(e.Runner(t, customers.Module), flags, true)
	if err != nil {
		t.Fatal(err)
	}
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Test, true
	at, _ := timeutil.ParseRFC3339("2026-11-16T10:00:00Z")
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{Read: read, Customers: cust})
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	return &accessStack{h: h, flags: flags, admin: admin}
}

func (s *accessStack) set(t *testing.T, key string, on bool) {
	t.Helper()
	at, _ := timeutil.ParseRFC3339("2026-11-16T09:00:00Z")
	if err := s.flags.InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error {
		return featureflags.Set(ctx, tx, featureflags.Flag{Key: key, Enabled: on, UpdatedAt: at})
	}); err != nil {
		t.Fatal(err)
	}
}

type decision struct {
	Capability   string   `json:"capability"`
	Allowed      bool     `json:"allowed"`
	Outcome      string   `json:"outcome"`
	ReasonCode   *string  `json:"reason_code"`
	Scope        string   `json:"scope"`
	MatchedRoles []string `json:"matched_roles"`
}

type effective struct {
	Authenticated bool       `json:"authenticated"`
	HRReadEnabled bool       `json:"hr_read_enabled"`
	Decisions     []decision `json:"decisions"`
}

func (s *accessStack) get(t *testing.T, path, user string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if user != "" {
		req.Header.Set(authstub.Header, user)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	var m map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	return rec.Code, m, rec.Body.String()
}

func (s *accessStack) effective(t *testing.T, user string) (effective, map[string]decision) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/authorization/effective-access", nil)
	if user != "" {
		req.Header.Set(authstub.Header, user)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("effective-access = %d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Outcome string    `json:"outcome"`
		Data    effective `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Outcome != "SUCCESS" || len(env.Data.Decisions) != 43 {
		t.Fatalf("effective-access envelope: %s", rec.Body.String())
	}
	by := map[string]decision{}
	for _, d := range env.Data.Decisions {
		by[d.Capability] = d
	}
	return env.Data, by
}

func want(t *testing.T, label string, d decision, allowed bool, outcome, reason string) {
	t.Helper()
	r := ""
	if d.ReasonCode != nil {
		r = *d.ReasonCode
	}
	if d.Allowed != allowed || d.Outcome != outcome || r != reason {
		t.Fatalf("%s %s: allowed=%v outcome=%s reason=%s; want %v %s %s", label, d.Capability, d.Allowed, d.Outcome, r, allowed, outcome, reason)
	}
}

// fingerprint is the role / grant / capability catalog state (N: no mutation).
func (s *accessStack) fingerprint(t *testing.T) string {
	t.Helper()
	var out string
	err := s.admin.QueryRow(context.Background(), `SELECT
	  (SELECT count(*) FROM identity.role_grant)::text || '/' ||
	  (SELECT coalesce(md5(string_agg(g::text, ',' ORDER BY g::text)), '') FROM identity.role_grant g) || '/' ||
	  (SELECT coalesce(md5(string_agg(r::text, ',' ORDER BY r::text)), '') FROM identity.role_capability r) || '/' ||
	  (SELECT count(*) FROM identity.capability)::text`).Scan(&out)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEffectiveAccessPerCapabilityAvailabilityPostgres(t *testing.T) {
	s := newAccessStack(t)
	before := s.fingerprint(t)
	hr := identityapi.HRReadCapabilities()
	const customerRead = string(identityapi.CapCustomerRead)

	// Default: every flag off. Nothing is allowed for anyone.
	for _, u := range []string{"", "dev-user-001", "dev-user-002"} {
		ea, ds := s.effective(t, u)
		if ea.HRReadEnabled || ea.Authenticated != (u != "") {
			t.Fatalf("default %q: %+v", u, ea)
		}
		for c, d := range ds {
			want(t, "default "+u, d, false, "BLOCKED", "FEATURE_DISABLED")
			_ = c
		}
	}

	// B + C + F. HR off, customers on.
	s.set(t, CustomersFlag, true)
	ea, ds := s.effective(t, "dev-user-001")
	if ea.HRReadEnabled {
		t.Fatal("hr_read_enabled with the HR flag off")
	}
	// B: the synthetic customer grant is no longer globally blocked by HR.
	want(t, "B", ds[customerRead], true, "SUCCESS", "")
	if ds[customerRead].Scope != "ORGANIZATION" || len(ds[customerRead].MatchedRoles) != 1 || ds[customerRead].MatchedRoles[0] != "synthetic_dev_stub" {
		t.Fatalf("B: %+v", ds[customerRead])
	}
	// The customers endpoint is governed by its own flag and guard.
	if code, m, body := s.get(t, "/api/v1/customers", "dev-user-001"); code != 200 || m["outcome"] != "SUCCESS" {
		t.Fatalf("B customers endpoint: %d %s", code, body)
	}
	// C: HR capabilities held through the HR role stay BLOCKED, endpoint too.
	for _, c := range hr {
		want(t, "C", ds[string(c.Key)], false, "BLOCKED", "FEATURE_DISABLED")
	}
	if code, m, body := s.get(t, "/api/v1/organization/persons", "dev-user-001"); code != 200 || m["outcome"] != "BLOCKED" || m["items"] != nil {
		t.Fatalf("C HR endpoint: %d %s", code, body)
	}
	// F: every non-customer, non-HR capability (own feature off) stays BLOCKED.
	allowed := 0
	for c, d := range ds {
		if d.Allowed {
			allowed++
			if c != customerRead && c != string(identityapi.CapMergeCaseRead) && c != string(identityapi.CapIdentityReportRead) {
				t.Fatalf("F: %s allowed with its feature off", c)
			}
		}
	}
	if allowed != 3 {
		t.Fatalf("F: %d allowed, want the 3 customer reads", allowed)
	}
	// A. HR off, non-HR capability not granted: denied, never allowed.
	_, viewer := s.effective(t, "dev-user-002")
	want(t, "A", viewer[customerRead], false, "UNAUTHORIZED", "ACTION_DENIED")
	if code, m, body := s.get(t, "/api/v1/customers", "dev-user-002"); code != 403 || m["outcome"] != "UNAUTHORIZED" {
		t.Fatalf("A customers endpoint: %d %s", code, body)
	}
	// H. Anonymous: authenticated false, nothing allowed, endpoint refused.
	anon, ads := s.effective(t, "")
	if anon.Authenticated {
		t.Fatal("H: anonymous authenticated")
	}
	want(t, "H", ads[customerRead], false, "UNAUTHORIZED", "ACTION_DENIED")
	if code, m, body := s.get(t, "/api/v1/customers", ""); code != 403 || m["outcome"] != "UNAUTHORIZED" {
		t.Fatalf("H customers endpoint: %d %s", code, body)
	}

	// D + E. HR on.
	s.set(t, HRReadFlag, true)
	_, viewer = s.effective(t, "dev-user-002")
	for _, c := range hr {
		want(t, "D", viewer[string(c.Key)], false, "UNAUTHORIZED", "ACTION_DENIED")
	}
	if code, m, body := s.get(t, "/api/v1/organization/persons", "dev-user-002"); code != 403 || m["outcome"] != "UNAUTHORIZED" {
		t.Fatalf("D HR endpoint: %d %s", code, body)
	}
	ea, ds = s.effective(t, "dev-user-001")
	if !ea.HRReadEnabled {
		t.Fatal("E: hr_read_enabled")
	}
	for _, c := range hr {
		want(t, "E", ds[string(c.Key)], true, "SUCCESS", "")
	}
	if code, m, body := s.get(t, "/api/v1/organization/persons", "dev-user-001"); code != 200 || m["outcome"] != "SUCCESS" {
		t.Fatalf("E HR endpoint: %d %s", code, body)
	}
	// HR on enables nothing outside HR + customers (whose own flag is on).
	if n := countAllowed(ds); n != 6+3 {
		t.Fatalf("E: %d allowed", n)
	}

	// F (reverse). Customers off, HR on: customer reads BLOCKED again.
	s.set(t, CustomersFlag, false)
	_, ds = s.effective(t, "dev-user-001")
	want(t, "F", ds[customerRead], false, "BLOCKED", "FEATURE_DISABLED")
	if code, m, body := s.get(t, "/api/v1/customers", "dev-user-001"); code != 200 || m["outcome"] != "BLOCKED" {
		t.Fatalf("F customers endpoint: %d %s", code, body)
	}

	// O. Turning a non-HR module's flag on never makes a write, review, posting
	// or reconciliation capability allowed through another module's flag.
	for _, c := range []identityapi.Capability{identityapi.CapPaymentBatchSubmit, identityapi.CapWorkAssign, identityapi.CapWorkExecute,
		identityapi.CapProviderOperationRecon, identityapi.CapHRRequestReview, identityapi.CapContactOutcomeRecord, identityapi.CapFinanceReviewRead} {
		want(t, "O", ds[string(c)], false, "BLOCKED", "FEATURE_DISABLED")
	}

	// N. No role, grant or capability catalog row was written.
	if after := s.fingerprint(t); after != before {
		t.Fatalf("N: role/grant state changed: %s -> %s", before, after)
	}
}

func countAllowed(ds map[string]decision) int {
	n := 0
	for _, d := range ds {
		if d.Allowed {
			n++
		}
	}
	return n
}

// K. The synthetic DEV/TEST subjects behind every allowed decision above are
// refused in production: the server will not start with dev auth there.
func TestDevAuthRefusedInProductionForAccess(t *testing.T) {
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Production, true
	at, _ := timeutil.ParseRFC3339("2026-11-16T10:00:00Z")
	if _, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)), apiserver.Surfaces{}); err == nil {
		t.Fatal("dev auth accepted in production")
	}
}
