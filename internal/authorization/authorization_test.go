package authorization

import (
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

func role(r identityapi.Role) *identityapi.Role { return &r }

func grant(r *identityapi.Role, st identityapi.GrantStatus) identityapi.Grant {
	return identityapi.Grant{UserID: "u-1", RoleKey: r, LegacyValue: "evidence", Status: st, Basis: "TEST"}
}

func str(s string) *string { return &s }

func TestAllowedOnlyByExplicitCapabilityGrant(t *testing.T) {
	s := Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), identityapi.GrantActive)}}
	for _, c := range identityapi.Capabilities() {
		d := Evaluate(Runtime, s, Request{Capability: c.Key})
		if !d.Allowed || d.Outcome != result.Success || d.Scope != ScopeOrganization || len(d.Matched) != 1 || d.Matched[0].Role != identityapi.RoleHR {
			t.Fatalf("%s: %+v", c.Key, d)
		}
	}
}

func TestDeniedWhenAbsent(t *testing.T) {
	cases := map[string]Subject{
		"anonymous":         {Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), identityapi.GrantActive)}},
		"no grants":         {PrincipalID: "p-1"},
		"role without cap":  {PrincipalID: "p-1", Grants: []identityapi.Grant{grant(role(identityapi.RoleSalesManager), identityapi.GrantActive)}},
		"unmapped role":     {PrincipalID: "p-1", Grants: []identityapi.Grant{grant(nil, identityapi.GrantActive)}},
		"unknown role":      {PrincipalID: "p-1", Grants: []identityapi.Grant{grant(role("administrator"), identityapi.GrantActive)}},
		"shadow at runtime": {PrincipalID: "p-1", Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), identityapi.GrantShadow)}},
		"unknown status":    {PrincipalID: "p-1", Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), "PENDING")}},
	}
	for name, s := range cases {
		d := Evaluate(Runtime, s, Request{Capability: identityapi.CapPersonRead})
		if d.Allowed || d.Outcome != result.Unauthorized || d.Reason == nil || *d.Reason != reasoncodes.ActionDenied || d.Scope != ScopeNone {
			t.Fatalf("%s: %+v", name, d)
		}
	}
}

// Merge-blocking security invariant: organisation context can never escalate.
func TestHierarchyAndPositionNeverGrant(t *testing.T) {
	org := &OrgContext{PositionSlug: str("hr"), LevelSlug: str("senior"), IsManager: true, SubordinateCount: 40, OrgUnitIDs: []string{"root"}}
	for _, grants := range [][]identityapi.Grant{nil, {grant(role(identityapi.RoleSalesManager), identityapi.GrantActive)}, {grant(nil, identityapi.GrantActive)}} {
		s := Subject{PrincipalID: "p-mgr", Grants: grants, Org: org}
		for _, mode := range []Mode{Runtime, Shadow} {
			for c, d := range EvaluateAll(mode, s) {
				if d.Allowed {
					t.Fatalf("manager/hr-position subject without capability was allowed %s: %+v", c, d)
				}
			}
		}
	}
	// And context never widens an existing grant's scope.
	s := Subject{PrincipalID: "p-hr", Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), identityapi.GrantActive)}}
	with := s
	with.Org = org
	for _, c := range identityapi.Capabilities() {
		a, b := Evaluate(Runtime, s, Request{Capability: c.Key}), Evaluate(Runtime, with, Request{Capability: c.Key})
		if a.Allowed != b.Allowed || a.Scope != b.Scope {
			t.Fatalf("org context changed the decision for %s", c.Key)
		}
	}
}

func TestUnknownAndConflictFailClosed(t *testing.T) {
	s := Subject{PrincipalID: "p-1", Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), identityapi.GrantActive)}}
	d := Evaluate(Runtime, s, Request{Capability: "org.person.delete"})
	if d.Allowed || d.Outcome != result.Blocked || *d.Reason != reasoncodes.PolicyUnresolved {
		t.Fatalf("unknown capability: %+v", d)
	}
	s.EvidenceConflict = true
	d = Evaluate(Shadow, s, Request{Capability: identityapi.CapPersonRead})
	if d.Allowed || d.Outcome != result.Blocked || *d.Reason != reasoncodes.LinkageConflict {
		t.Fatalf("conflict: %+v", d)
	}
}

func TestShadowModeCountsShadowGrants(t *testing.T) {
	s := Subject{PrincipalID: "u-1", Grants: []identityapi.Grant{grant(role(identityapi.RoleHR), identityapi.GrantShadow)}}
	if d := Evaluate(Shadow, s, Request{Capability: identityapi.CapUnitRead}); !d.Allowed {
		t.Fatalf("shadow: %+v", d)
	}
	if d := Evaluate(Runtime, s, Request{Capability: identityapi.CapUnitRead}); d.Allowed {
		t.Fatal("a SHADOW grant authorised a runtime request")
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		l       LegacyVerdict
		t       TargetVerdict
		unmap   bool
		want    DiffClass
		secured bool
	}{
		{LegacyAllowed, TargetAllowed, false, DiffMatch, false},
		{LegacyDenied, TargetDenied, false, DiffMatch, false},
		{LegacyAllowed, TargetDenied, false, DiffTargetDeniesLegacyAllows, false},
		{LegacyDenied, TargetAllowed, false, DiffTargetAllowsLegacyDenies, true},
		{LegacyUnknown, TargetDenied, false, DiffUnknownLegacy, false},
		{LegacyUnknown, TargetAllowed, false, DiffUnknownLegacy, true},
		{LegacyUnknown, TargetDenied, true, DiffUnmappedRole, false},
		{LegacyAllowed, TargetDenied, true, DiffUnmappedRole, false},
		{LegacyAllowed, TargetUnknown, false, DiffUnknownTarget, false},
		{LegacyConflict, TargetAllowed, false, DiffConflict, true},
		{LegacyAllowed, TargetConflict, false, DiffConflict, false},
	}
	for _, c := range cases {
		got, sec := Classify(c.l, c.t, c.unmap)
		if got != c.want || sec != c.secured {
			t.Fatalf("Classify(%s,%s,%v) = %s,%v want %s,%v", c.l, c.t, c.unmap, got, sec, c.want, c.secured)
		}
	}
	if TargetVerdictOf(Evaluate(Shadow, Subject{PrincipalID: "x", EvidenceConflict: true}, Request{Capability: identityapi.CapPersonRead})) != TargetConflict {
		t.Fatal("conflict verdict")
	}
}

func TestVocabularyShape(t *testing.T) {
	seen := map[identityapi.Capability]bool{}
	for _, c := range identityapi.Capabilities() {
		if seen[c.Key] {
			t.Fatalf("duplicate capability %s", c.Key)
		}
		seen[c.Key] = true
	}
	for _, r := range identityapi.Roles() {
		for _, c := range r.Capabilities {
			if !seen[c] {
				t.Fatalf("role %s bundles unknown capability %s", r.Key, c)
			}
		}
		if r.Key != identityapi.RoleHR && len(r.Capabilities) != 0 {
			t.Fatalf("role %s carries Slice 4 capabilities without evidence", r.Key)
		}
	}
}
