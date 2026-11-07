package authorization

import (
	"context"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// GrantLoader loads the runtime (ACTIVE) role grants of one user. identity
// implements it; cmd/crm-api composes it.
type GrantLoader interface {
	ActiveGrants(ctx context.Context, userID string) ([]identityapi.Grant, error)
}

// SessionSubjects is the production subject resolver (R1). The subject is
// the identity user of an authenticated server-side session
// (websession.Principal, set only by the session middleware), and its only
// authority is that user's ACTIVE role grants.
//
// Invariants (tested):
//   - authentication establishes identity, never permission: no grant, no
//     capability (today no ACTIVE grant exists, so every capability denies);
//   - it never reads the development stub principal (authstub) and never
//     yields a synthetic DEV grant, even if a loader returned one;
//   - a grant of another user, or a non-ACTIVE grant, is dropped;
//   - a grant loading failure fails closed (EvidenceConflict → BLOCKED).
type SessionSubjects struct {
	Grants GrantLoader
}

// Subject implements SubjectResolver.
func (s SessionSubjects) Subject(ctx context.Context) Subject {
	p, ok := websession.FromContext(ctx)
	if !ok || p.UserID == "" || p.SessionID == "" {
		return Subject{}
	}
	subject := Subject{PrincipalID: p.UserID, UserID: p.UserID}
	if s.Grants == nil {
		return subject
	}
	grants, err := s.Grants.ActiveGrants(ctx, p.UserID)
	if err != nil {
		subject.EvidenceConflict = true
		return subject
	}
	for _, g := range grants {
		if g.UserID != p.UserID || g.Status != identityapi.GrantActive || g.Basis == DevGrantBasis {
			continue
		}
		g.Capabilities = nil // roles only; direct capability sets never come from data
		subject.Grants = append(subject.Grants, g)
	}
	return subject
}
