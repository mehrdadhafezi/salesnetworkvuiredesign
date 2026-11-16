package http_test

// RM-ACCESS-01 security regression matrix for the effective-access view
// (handler level; the real-database composition is covered in
// cmd/crm-api/access_pg_test.go). Every grant below is a synthetic test value
// built in memory: nothing is persisted and no production grant exists.

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

type resolver func(context.Context) authorization.Subject

func (r resolver) Subject(ctx context.Context) authorization.Subject { return r(ctx) }

func fixed(s authorization.Subject) authorization.SubjectResolver {
	return resolver(func(context.Context) authorization.Subject { return s })
}

func role(r identityapi.Role) *identityapi.Role { return &r }

func active(r identityapi.Role) identityapi.Grant {
	return identityapi.Grant{UserID: "u-1", RoleKey: role(r), LegacyValue: "test", Status: identityapi.GrantActive, Basis: "TEST"}
}

// synthetic is the dev-stub style direct grant (test-only; honoured only
// with the SYNTHETIC_DEV_STUB basis and ACTIVE status).
func synthetic(caps ...identityapi.Capability) identityapi.Grant {
	return identityapi.Grant{UserID: "u-1", LegacyValue: "synthetic", Status: identityapi.GrantActive, Basis: authorization.DevGrantBasis, Capabilities: caps}
}

// flags models the per-capability guard flags: each module key is on or off.
type flags map[string]bool

// moduleOf is a test-local grouping that mirrors the production table's
// shape (one flag per guarded surface) without importing cmd/crm-api.
func moduleOf(c identityapi.Capability) string {
	for _, h := range identityapi.HRReadCapabilities() {
		if h.Key == c {
			return "hr"
		}
	}
	switch c {
	case identityapi.CapCustomerRead, identityapi.CapMergeCaseRead, identityapi.CapIdentityReportRead:
		return "customers"
	case identityapi.CapCommissionShadowRead, identityapi.CapCommissionPolicyRead, identityapi.CapCommissionOwnRead:
		return "commission"
	}
	return "other"
}

func (f flags) availability() authorization.Availability {
	return func(context.Context) (map[identityapi.Capability]bool, error) {
		out := map[identityapi.Capability]bool{}
		for _, c := range identityapi.Capabilities() {
			out[c.Key] = f[moduleOf(c.Key)]
		}
		return out, nil
	}
}

func evaluate(t *testing.T, hrOn bool, s authorization.SubjectResolver, a authorization.Availability) (oapi.EffectiveAccess, error) {
	t.Helper()
	g, err := authorization.NewGuard(func(context.Context) (bool, error) { return hrOn, nil }, s)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := authzhttp.New(g, a).GetEffectiveAccess(context.Background(), oapi.GetEffectiveAccessRequestObject{})
	if err != nil {
		return oapi.EffectiveAccess{}, err
	}
	return resp.(oapi.GetEffectiveAccess200JSONResponse).Body.Data, nil
}

func must(t *testing.T, hrOn bool, s authorization.SubjectResolver, a authorization.Availability) map[oapi.Capability]oapi.AccessDecision {
	t.Helper()
	ea, err := evaluate(t, hrOn, s, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(ea.Decisions) != len(identityapi.Capabilities()) {
		t.Fatalf("decisions = %d, registry = %d", len(ea.Decisions), len(identityapi.Capabilities()))
	}
	out := map[oapi.Capability]oapi.AccessDecision{}
	for _, d := range ea.Decisions {
		out[d.Capability] = d
	}
	return out
}

func reason(d oapi.AccessDecision) string {
	if d.ReasonCode == nil {
		return ""
	}
	return string(*d.ReasonCode)
}

func expect(t *testing.T, name string, d oapi.AccessDecision, allowed bool, outcome, why, scope string) {
	t.Helper()
	if d.Allowed != allowed || string(d.Outcome) != outcome || reason(d) != why || string(d.Scope) != scope {
		t.Fatalf("%s %s: allowed=%v outcome=%s reason=%s scope=%s; want %v %s %s %s", name, d.Capability, d.Allowed, d.Outcome, reason(d), d.Scope,
			allowed, outcome, why, scope)
	}
}

const (
	cust = oapi.Capability(identityapi.CapCustomerRead)
	pers = oapi.Capability(identityapi.CapPersonRead)
	comm = oapi.Capability(identityapi.CapCommissionShadowRead)
)

var devAll = func() []identityapi.Capability {
	var out []identityapi.Capability
	for _, c := range identityapi.Capabilities() {
		out = append(out, c.Key)
	}
	return out
}()

// A. HR flag OFF, non-HR capability NOT granted: denied, never allowed.
func TestA_HROffNonHRNotGranted(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1"})
	ds := must(t, false, s, flags{"customers": true}.availability())
	expect(t, "A", ds[cust], false, "UNAUTHORIZED", "ACTION_DENIED", "NONE")
	for c, d := range ds {
		if d.Allowed {
			t.Fatalf("A: %s allowed without a grant", c)
		}
	}
}

// B. HR flag OFF, non-HR capability granted (synthetic ACTIVE grant), its own
// feature ON: no longer globally blocked by HR. Before RM-ACCESS-01 this was
// BLOCKED / FEATURE_DISABLED because the HR flag gated every capability.
func TestB_HROffNonHRGrantedIsNotBlockedByHR(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{synthetic(identityapi.CapCustomerRead)}})
	ds := must(t, false, s, flags{"customers": true}.availability())
	expect(t, "B", ds[cust], true, "SUCCESS", "", "ORGANIZATION")
	if !slices.Equal(ds[cust].MatchedRoles, []string{string(authorization.SyntheticDevRole)}) {
		t.Fatalf("B matched roles: %v", ds[cust].MatchedRoles)
	}
	// Only the granted capability: the grant is not widened to its module.
	expect(t, "B", ds[oapi.Capability(identityapi.CapMergeCaseRead)], false, "UNAUTHORIZED", "ACTION_DENIED", "NONE")
}

// C. HR flag OFF, HR capability granted: still BLOCKED (HR stays unavailable),
// even when every other module's feature is on.
func TestC_HROffHRGrantedStaysBlocked(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{active(identityapi.RoleHR)}})
	ds := must(t, false, s, flags{"customers": true, "commission": true, "other": true}.availability())
	for _, c := range identityapi.HRReadCapabilities() {
		expect(t, "C", ds[oapi.Capability(c.Key)], false, "BLOCKED", "FEATURE_DISABLED", "NONE")
	}
}

// D. HR flag ON, HR capability not granted: UNAUTHORIZED.
func TestD_HROnHRNotGranted(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{active(identityapi.RoleSalesManager)}})
	ds := must(t, true, s, flags{"hr": true}.availability())
	for _, c := range identityapi.HRReadCapabilities() {
		expect(t, "D", ds[oapi.Capability(c.Key)], false, "UNAUTHORIZED", "ACTION_DENIED", "NONE")
	}
}

// E. HR flag ON, HR capability granted: exactly the HR read set is allowed,
// and HR's flag enables nothing outside HR.
func TestE_HROnHRGranted(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{active(identityapi.RoleHR)}})
	ea, err := evaluate(t, true, s, flags{"hr": true}.availability())
	if err != nil {
		t.Fatal(err)
	}
	if !ea.HrReadEnabled || !ea.Authenticated {
		t.Fatalf("E: %+v", ea)
	}
	allowed := 0
	for _, d := range ea.Decisions {
		if d.Allowed {
			allowed++
			if moduleOf(identityapi.Capability(d.Capability)) != "hr" || !slices.Equal(d.MatchedRoles, []string{"hr"}) {
				t.Fatalf("E: %s allowed (%v)", d.Capability, d.MatchedRoles)
			}
		}
	}
	if allowed != len(identityapi.HRReadCapabilities()) {
		t.Fatalf("E: %d allowed", allowed)
	}
	// HR flag on never makes a non-HR capability available, even if held.
	s2 := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{synthetic(devAll...)}})
	ds := must(t, true, s2, flags{"hr": true}.availability())
	expect(t, "E", ds[cust], false, "BLOCKED", "FEATURE_DISABLED", "NONE")
	expect(t, "E", ds[comm], false, "BLOCKED", "FEATURE_DISABLED", "NONE")
}

// F. Non-HR feature OFF, non-HR capability granted: BLOCKED, whatever HR says.
func TestF_NonHRFeatureOffGranted(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{synthetic(devAll...)}})
	for _, hr := range []bool{false, true} {
		ds := must(t, hr, s, flags{"hr": hr, "customers": false, "commission": true}.availability())
		expect(t, "F", ds[cust], false, "BLOCKED", "FEATURE_DISABLED", "NONE")
		expect(t, "F", ds[comm], true, "SUCCESS", "", "ORGANIZATION")
	}
}

// G. No ACTIVE grant (shadow, unknown status, unmapped or unknown role,
// non-synthetic direct capabilities): nothing is allowed, every feature on.
func TestG_NoActiveGrant(t *testing.T) {
	all := flags{"hr": true, "customers": true, "commission": true, "other": true}.availability()
	for name, grants := range map[string][]identityapi.Grant{
		"none":            nil,
		"shadow":          {{UserID: "u-1", RoleKey: role(identityapi.RoleHR), LegacyValue: "x", Status: identityapi.GrantShadow, Basis: "LEGACY"}},
		"pending":         {{UserID: "u-1", RoleKey: role(identityapi.RoleHR), LegacyValue: "x", Status: "PENDING", Basis: "LEGACY"}},
		"unmapped role":   {{UserID: "u-1", LegacyValue: "x", Status: identityapi.GrantActive, Basis: "LEGACY"}},
		"unknown role":    {active("administrator")},
		"direct non-dev":  {{UserID: "u-1", LegacyValue: "x", Status: identityapi.GrantActive, Basis: "LEGACY", Capabilities: devAll}},
		"shadow dev":      {{UserID: "u-1", LegacyValue: "x", Status: identityapi.GrantShadow, Basis: authorization.DevGrantBasis, Capabilities: devAll}},
		"empty bundle":    {active(identityapi.RoleSeller), active(identityapi.RoleFinance), active(identityapi.RoleMIS)},
		"revoked/expired": {{UserID: "u-1", RoleKey: role(identityapi.RoleHR), LegacyValue: "x", Status: "REVOKED", Basis: "LEGACY"}},
	} {
		for c, d := range must(t, true, fixed(authorization.Subject{PrincipalID: "p-1", Grants: grants}), all) {
			if d.Allowed {
				t.Fatalf("G %s: %s allowed", name, c)
			}
		}
	}
}

// H. Anonymous (also: expired or revoked session, which resolves to the
// anonymous subject): no business permission, authenticated false.
func TestH_AnonymousNothingAllowed(t *testing.T) {
	all := flags{"hr": true, "customers": true, "commission": true, "other": true}.availability()
	ea, err := evaluate(t, true, fixed(authorization.Subject{Grants: []identityapi.Grant{active(identityapi.RoleHR), synthetic(devAll...)}}), all)
	if err != nil {
		t.Fatal(err)
	}
	if ea.Authenticated {
		t.Fatal("H: anonymous reported authenticated")
	}
	for _, d := range ea.Decisions {
		if d.Allowed || len(d.MatchedRoles) != 0 {
			t.Fatalf("H: %s %+v", d.Capability, d)
		}
	}
}

// I. Conflicting evidence and unavailable availability evidence fail closed.
func TestI_ConflictAndInvalidEvidenceFailClosed(t *testing.T) {
	all := flags{"hr": true, "customers": true, "commission": true, "other": true}.availability()
	s := fixed(authorization.Subject{PrincipalID: "p-1", EvidenceConflict: true, Grants: []identityapi.Grant{active(identityapi.RoleHR), synthetic(devAll...)}})
	for c, d := range must(t, true, s, all) {
		if d.Allowed || d.Outcome != "BLOCKED" || reason(d) != "LINKAGE_CONFLICT" {
			t.Fatalf("I conflict: %s %+v", c, d)
		}
	}
	held := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{synthetic(devAll...)}})
	// A flag store failure is an error (500), never a partial allowed view.
	if _, err := evaluate(t, true, held, func(context.Context) (map[identityapi.Capability]bool, error) {
		return nil, errors.New("flag store down")
	}); err == nil {
		t.Fatal("I: availability error swallowed")
	}
	// No availability source: every capability is unavailable.
	for c, d := range must(t, true, held, nil) {
		if d.Allowed || d.Outcome != "BLOCKED" || reason(d) != "FEATURE_DISABLED" {
			t.Fatalf("I nil availability: %s %+v", c, d)
		}
	}
	// A capability missing from the availability map is unavailable.
	partial := func(context.Context) (map[identityapi.Capability]bool, error) {
		return map[identityapi.Capability]bool{identityapi.CapCustomerRead: true}, nil
	}
	for c, d := range must(t, true, held, partial) {
		if d.Allowed != (c == cust) {
			t.Fatalf("I partial availability: %s %+v", c, d)
		}
	}
	// An availability entry for a capability outside the registry is ignored.
	extra := func(context.Context) (map[identityapi.Capability]bool, error) {
		return map[identityapi.Capability]bool{"org.assignment.apply": true}, nil
	}
	for c, d := range must(t, true, held, extra) {
		if d.Allowed || c == "org.assignment.apply" {
			t.Fatalf("I unknown capability: %s %+v", c, d)
		}
	}
}

// J. Multiple role grants: every matched role is reported, sorted, with no
// selection and no privilege beyond the union of the held bundles.
func TestJ_MultipleRolesNoInventedSelection(t *testing.T) {
	s := fixed(authorization.Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{
		active(identityapi.RoleSeller), active(identityapi.RoleHR), synthetic(identityapi.CapPersonRead)}})
	ds := must(t, true, s, flags{"hr": true, "customers": true, "commission": true, "other": true}.availability())
	if d := ds[pers]; !d.Allowed || !slices.Equal(d.MatchedRoles, []string{"hr", string(authorization.SyntheticDevRole)}) {
		t.Fatalf("J: %+v", d)
	}
	for c, d := range ds {
		if d.Allowed && moduleOf(identityapi.Capability(c)) != "hr" {
			t.Fatalf("J: %s escalated", c)
		}
	}
}

// L. The OpenAPI Capability enum is exactly the backend registry (both
// directions), so a future drift fails here. The enum values are read from
// the generated Go contract.
func TestL_ContractEnumEqualsRegistry(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	gen := filepath.Join(filepath.Dir(file), "..", "..", "platform", "httpx", "oapi", "oapi.gen.go")
	f, err := parser.ParseFile(token.NewFileSet(), gen, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var enum []string
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != "Capability" {
				continue
			}
			for _, v := range vs.Values {
				s, err := strconv.Unquote(v.(*ast.BasicLit).Value)
				if err != nil {
					t.Fatal(err)
				}
				enum = append(enum, s)
			}
		}
	}
	var registry []string
	for _, c := range identityapi.Capabilities() {
		registry = append(registry, string(c.Key))
		if !oapi.Capability(c.Key).Valid() {
			t.Errorf("registry capability %s is not in the OpenAPI enum", c.Key)
		}
	}
	slices.Sort(enum)
	slices.Sort(registry)
	if len(registry) != 43 || !slices.Equal(enum, registry) {
		t.Fatalf("contract enum (%d) != registry (%d)\nenum:     %v\nregistry: %v", len(enum), len(registry), enum, registry)
	}
}
