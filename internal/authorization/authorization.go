// Package authorization is the read evaluator of Slice 4 (Architecture V1
// §5, §18): Route → Module → Action → Row → Field, evaluated in code. It owns
// no business objects and, in Slice 4, no tables: role bundles and the
// capability vocabulary come from the published identity API, and the only
// policy is code-defined below. There is no rules language.
//
// Invariants (merge-blocking, tested):
//   - authority comes only from a role grant whose role bundle contains the
//     capability; nothing else ever allows;
//   - hierarchy, position, level and org-unit context never grant, and never
//     widen a grant (organization is scope context, not a permission engine);
//   - UNKNOWN never becomes allowed: an unknown capability, an unmapped or
//     unknown role, missing grants or conflicting evidence all deny;
//   - runtime evaluation counts only ACTIVE grants; SHADOW grants (imported
//     legacy evidence) count only in an explicit shadow evaluation used by
//     the capability diff report, never for a request.
package authorization

import (
	"sort"
	"strings"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Mode selects which grants count.
type Mode uint8

const (
	// Runtime counts ACTIVE grants only (every request).
	Runtime Mode = iota
	// Shadow also counts SHADOW grants. Used only for the offline capability
	// diff; never reachable from a request handler.
	Shadow
)

// OrgContext is organisation scope context about the subject. It is
// accepted so callers can pass it, and it is deliberately never a source of
// authority: Slice 4 has no frozen policy that derives permission from
// position or hierarchy (D-32 and D-42 remain open).
type OrgContext struct {
	PositionSlug     *string
	LevelSlug        *string
	IsManager        bool
	SubordinateCount int
	OrgUnitIDs       []string
}

// Subject is who is asking.
type Subject struct {
	// PrincipalID is empty for an anonymous request.
	PrincipalID string
	Grants      []identityapi.Grant
	// EvidenceConflict marks grant evidence the caller knows to be ambiguous
	// (e.g. one account linked to several legacy profiles). It fails closed.
	EvidenceConflict bool
	Org              *OrgContext
}

// Request is what is asked: one capability on an optional resource.
type Request struct {
	Capability identityapi.Capability
}

// Scope is the row scope a decision grants.
type Scope string

const (
	ScopeNone         Scope = "NONE"
	ScopeOrganization Scope = "ORGANIZATION"
)

// MatchedGrant names a grant that allowed a decision (safe to show).
type MatchedGrant struct {
	Role   identityapi.Role
	Basis  string
	Status identityapi.GrantStatus
}

// Decision is the server-authoritative result.
type Decision struct {
	Allowed bool
	// Outcome is SUCCESS when allowed, UNAUTHORIZED when the subject lacks
	// authority, BLOCKED when the policy itself cannot decide.
	Outcome       result.Status
	Reason        *reasoncodes.Code
	Module        string
	Matched       []MatchedGrant
	Scope         Scope
	IgnoredGrants int // unmapped, unknown-role or non-counting grants
}

func deny(out result.Status, code reasoncodes.Code, module string, ignored int) Decision {
	c := code
	return Decision{Outcome: out, Reason: &c, Module: module, Scope: ScopeNone, IgnoredGrants: ignored}
}

// Evaluate decides one request. It is pure and deterministic.
func Evaluate(mode Mode, s Subject, r Request) Decision {
	module, _, _ := strings.Cut(string(r.Capability), ".")
	if !identityapi.KnownCapability(r.Capability) {
		return deny(result.Blocked, reasoncodes.PolicyUnresolved, module, 0)
	}
	if s.PrincipalID == "" {
		return deny(result.Unauthorized, reasoncodes.ActionDenied, module, 0)
	}
	if s.EvidenceConflict {
		return deny(result.Blocked, reasoncodes.LinkageConflict, module, len(s.Grants))
	}
	var matched []MatchedGrant
	ignored := 0
	for _, g := range s.Grants {
		counts := g.Status == identityapi.GrantActive || (mode == Shadow && g.Status == identityapi.GrantShadow)
		// Synthetic DEV/TEST stub grants may carry a direct capability set;
		// any other grant's direct capabilities are ignored (roles only).
		if counts && g.Basis == DevGrantBasis && g.Status == identityapi.GrantActive && len(g.Capabilities) > 0 {
			hit := false
			for _, c := range g.Capabilities {
				if c == r.Capability {
					hit = true
				}
			}
			if hit {
				matched = append(matched, MatchedGrant{Role: SyntheticDevRole, Basis: g.Basis, Status: g.Status})
				continue
			}
		}
		if !counts || g.RoleKey == nil {
			ignored++
			continue
		}
		role, ok := identityapi.LookupRole(*g.RoleKey)
		if !ok {
			ignored++
			continue
		}
		for _, c := range role.Capabilities {
			if c == r.Capability {
				matched = append(matched, MatchedGrant{Role: role.Key, Basis: g.Basis, Status: g.Status})
				break
			}
		}
	}
	if len(matched) == 0 {
		return deny(result.Unauthorized, reasoncodes.ActionDenied, module, ignored)
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Role != matched[j].Role {
			return matched[i].Role < matched[j].Role
		}
		return matched[i].Basis < matched[j].Basis
	})
	// Slice 4 read capabilities are organisation-wide for a holder of the
	// capability (the HR role sees the organisation; HR spec §1, §17).
	// Sensitive fields are not part of any Slice 4 read model at all.
	return Decision{Allowed: true, Outcome: result.Success, Module: module, Matched: matched, Scope: ScopeOrganization, IgnoredGrants: ignored}
}

// EvaluateAll evaluates every vocabulary capability for s (allowed_actions
// and the effective-access view use the same evaluator).
func EvaluateAll(mode Mode, s Subject) map[identityapi.Capability]Decision {
	out := map[identityapi.Capability]Decision{}
	for _, c := range identityapi.Capabilities() {
		out[c.Key] = Evaluate(mode, s, Request{Capability: c.Key})
	}
	return out
}
