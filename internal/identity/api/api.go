// Package api is the published surface of the identity module (Architecture
// V1 §21): the capability vocabulary and role bundles, the shadow import
// command contract, and the read/resolve interfaces other modules may call.
// Other modules import this package only; identity's store and command
// implementations stay private.
//
// Nothing here is authentication. Slice 4 has no credentials, sessions or
// login; user accounts are a read/shadow model and every role grant is a
// SHADOW grant that never authorises a runtime request.
package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// Capability is a module-qualified action, "<module>.<object>.<action>"
// (Architecture V1 §18). Slice 4 defines only the read capabilities the HR
// read surfaces need; later slices add their own.
type Capability string

// Slice 4 read capabilities. The "org." prefix follows the architecture's
// own example ("org.assignment.apply"); there is no second taxonomy.
const (
	CapUserRead       Capability = "identity.user.read"
	CapPersonRead     Capability = "org.person.read"
	CapEmploymentRead Capability = "org.employment.read"
	CapPositionRead   Capability = "org.position.read"
	CapUnitRead       Capability = "org.unit.read"
	CapAssignmentRead Capability = "org.assignment.read"
)

// CapabilityDef is one vocabulary entry.
type CapabilityDef struct {
	Key         Capability
	Description string
}

var vocabulary = []CapabilityDef{
	{CapUserRead, "Read user accounts (shadow identity model)"},
	{CapPersonRead, "Read persons (HR identity)"},
	{CapEmploymentRead, "Read employment lifecycle attributes"},
	{CapPositionRead, "Read positions and levels"},
	{CapUnitRead, "Read the organisation unit tree"},
	{CapAssignmentRead, "Read reporting assignments, their history and hierarchy findings"},
}

// Capabilities returns the Slice 4 vocabulary in a stable order.
func Capabilities() []CapabilityDef { return append([]CapabilityDef(nil), vocabulary...) }

// KnownCapability reports whether c is in the vocabulary.
func KnownCapability(c Capability) bool {
	for _, d := range vocabulary {
		if d.Key == c {
			return true
		}
	}
	return false
}

// Role is a named capability bundle (Architecture V1 §18). Target roles are
// the primary operational roles of §4A; they are not WordPress role names.
type Role string

// Target roles (Architecture V1 §4A).
const (
	RoleSeller           Role = "seller"
	RoleSupervisor       Role = "supervisor"
	RoleSeniorSupervisor Role = "senior_supervisor"
	RoleSalesManager     Role = "sales_manager"
	RoleSalesDeputy      Role = "sales_deputy"
	RoleMIS              Role = "mis"
	RoleHR               Role = "hr"
	RoleFinance          Role = "finance"
	RoleInvoiceReviewer  Role = "invoice_reviewer"
)

// RoleDef is one role with its Slice 4 bundle.
type RoleDef struct {
	Key          Role
	Label        string
	Capabilities []Capability
}

// Only the HR role carries Slice 4 (HR read) capabilities. Every other role
// exists so legacy role evidence can be mapped, but grants nothing in this
// slice: its capabilities belong to later slices, not to guesses here.
var roles = []RoleDef{
	{RoleSeller, "Seller", nil},
	{RoleSupervisor, "Supervisor", nil},
	{RoleSeniorSupervisor, "Senior Supervisor", nil},
	{RoleSalesManager, "Sales Manager", nil},
	{RoleSalesDeputy, "Sales Deputy", nil},
	{RoleMIS, "MIS", nil},
	{RoleHR, "HR", []Capability{CapUserRead, CapPersonRead, CapEmploymentRead, CapPositionRead, CapUnitRead, CapAssignmentRead}},
	{RoleFinance, "Finance", nil},
	{RoleInvoiceReviewer, "Invoice Reviewer", nil},
}

// Roles returns every target role with its bundle, in a stable order.
func Roles() []RoleDef {
	out := make([]RoleDef, len(roles))
	for i, r := range roles {
		out[i] = RoleDef{Key: r.Key, Label: r.Label, Capabilities: append([]Capability(nil), r.Capabilities...)}
	}
	return out
}

// LookupRole returns the role definition for key.
func LookupRole(key Role) (RoleDef, bool) {
	for _, r := range Roles() {
		if r.Key == key {
			return r, true
		}
	}
	return RoleDef{}, false
}

// --- provenance ---------------------------------------------------------

var (
	// ErrInvalidImport reports an import command that breaks its contract.
	ErrInvalidImport = errors.New("identity: invalid import")

	keyPattern  = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	specPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*\.v[1-9][0-9]*$`)
	sysPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,15}$`)
)

// SourceRef is the provenance of one imported legacy row: source system,
// extract spec, legacy key, staging batch and the staged row checksum. The
// legacy key is provenance, never the target identity.
type SourceRef struct {
	System    string
	Spec      string
	Key       string
	BatchID   string
	RowSHA256 []byte
}

// Validate checks the reference shape.
func (s SourceRef) Validate() error {
	switch {
	case !sysPattern.MatchString(s.System):
		return fmt.Errorf("%w: source system %q", ErrInvalidImport, s.System)
	case !specPattern.MatchString(s.Spec):
		return fmt.Errorf("%w: source spec %q", ErrInvalidImport, s.Spec)
	case !keyPattern.MatchString(s.Key):
		return fmt.Errorf("%w: source key %q", ErrInvalidImport, s.Key)
	case !uuidPattern.MatchString(s.BatchID):
		return fmt.Errorf("%w: batch id %q", ErrInvalidImport, s.BatchID)
	case len(s.RowSHA256) != 32:
		return fmt.Errorf("%w: row checksum length %d", ErrInvalidImport, len(s.RowSHA256))
	}
	return nil
}

// ValidKey reports whether k is a canonical legacy key.
func ValidKey(k string) bool { return keyPattern.MatchString(k) }

// Provenance is the provenance of a stored target row, as read back.
type Provenance struct {
	System      string
	Spec        string
	Key         string
	BatchID     string
	ImportedAt  time.Time
	RefreshedAt time.Time
	Presence    string // PRESENT | ABSENT_IN_LATEST
}

// --- import command ---------------------------------------------------------

// UserRecord is one legacy account. Login and display name are evidence
// only; neither is ever a key.
type UserRecord struct {
	Source        SourceRef
	Login         *string
	DisplayName   *string
	RegisteredRaw *string
}

// RoleEvidence is one raw legacy role label attached to a user. RoleKey is
// the target role the label maps to, or nil when it is not mapped. It is
// materialised as a SHADOW grant only.
type RoleEvidence struct {
	Evidence    SourceRef
	UserKey     string
	LegacyValue string
	RoleKey     *Role
}

// IdentityImport is one identity import command for one staged batch.
type IdentityImport struct {
	BatchID    string
	SnapshotID string
	// SourceSystem is the legacy system of every row (and of the absence
	// update when an entity is COMPLETE but empty).
	SourceSystem string
	// UsersComplete marks the staged user entity COMPLETE: only then may
	// previously imported accounts missing from this batch be marked
	// ABSENT_IN_LATEST (never deleted).
	UsersComplete    bool
	EvidenceComplete bool
	Users            []UserRecord
	RoleEvidence     []RoleEvidence
}

// Validate checks the command.
func (in IdentityImport) Validate() error {
	if !uuidPattern.MatchString(in.BatchID) || in.SnapshotID == "" || !sysPattern.MatchString(in.SourceSystem) {
		return fmt.Errorf("%w: batch/snapshot/source system", ErrInvalidImport)
	}
	seen := map[string]bool{}
	for _, u := range in.Users {
		if err := u.Source.Validate(); err != nil {
			return err
		}
		if u.Source.BatchID != in.BatchID || u.Source.System != in.SourceSystem {
			return fmt.Errorf("%w: user row from another batch or system", ErrInvalidImport)
		}
		if seen[u.Source.System+"/"+u.Source.Key] {
			return fmt.Errorf("%w: duplicate user key %s", ErrInvalidImport, u.Source.Key)
		}
		seen[u.Source.System+"/"+u.Source.Key] = true
	}
	evidence := map[string]bool{}
	for _, e := range in.RoleEvidence {
		if err := e.Evidence.Validate(); err != nil {
			return err
		}
		if e.Evidence.BatchID != in.BatchID || e.Evidence.System != in.SourceSystem {
			return fmt.Errorf("%w: role evidence from another batch or system", ErrInvalidImport)
		}
		if !ValidKey(e.UserKey) || e.LegacyValue == "" || len(e.LegacyValue) > 120 {
			return fmt.Errorf("%w: role evidence %s", ErrInvalidImport, e.Evidence.Key)
		}
		if e.RoleKey != nil {
			if _, ok := LookupRole(*e.RoleKey); !ok {
				return fmt.Errorf("%w: unknown target role %q", ErrInvalidImport, *e.RoleKey)
			}
		}
		if evidence[e.Evidence.Key] {
			return fmt.Errorf("%w: duplicate role evidence %s", ErrInvalidImport, e.Evidence.Key)
		}
		evidence[e.Evidence.Key] = true
	}
	return nil
}

// Issue is one item the command could not resolve. It carries keys and a
// reason, never row values.
type Issue struct {
	Kind      string // e.g. ROLE_EVIDENCE_USER_MISSING
	SourceKey string
	Detail    string
}

// ImportReport is the command result.
type ImportReport struct {
	Outcome result.Status
	Replay  bool
	RunID   string
	Counts  map[string]int64
	Issues  []Issue
}

// Importer is the identity import command (implemented by identity/imports).
type Importer interface {
	ImportIdentity(ctx context.Context, in IdentityImport) (ImportReport, error)
}

// --- read / resolve -----------------------------------------------------------

// UserResolver resolves legacy account keys to native user ids (published
// identity reference). Missing keys are absent from the result.
type UserResolver interface {
	ResolveUsers(ctx context.Context, system string, keys []string) (map[string]string, error)
}

// GrantStatus is the lifecycle of a role grant. Slice 4 materialises SHADOW
// grants only; ACTIVE exists for runtime subjects outside the import path
// (synthetic dev/test subjects) and future identity commands.
type GrantStatus string

const (
	GrantShadow GrantStatus = "SHADOW"
	GrantActive GrantStatus = "ACTIVE"
)

// Grant is one role grant as the authorization module reads it.
type Grant struct {
	UserID      string
	RoleKey     *Role  // nil: legacy label not mapped to a target role
	LegacyValue string // raw evidence
	Status      GrantStatus
	Basis       string
}

// GrantReader lists stored grants (read-only).
type GrantReader interface {
	Grants(ctx context.Context) ([]Grant, error)
}

// SortedCapabilities returns caps sorted, deduplicated.
func SortedCapabilities(caps []Capability) []Capability {
	m := map[Capability]bool{}
	for _, c := range caps {
		m[c] = true
	}
	out := make([]Capability, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Reader is the published identity read surface used by reconciliation and
// the authorization diff.
type Reader interface {
	UserResolver
	GrantReader
	Counts(ctx context.Context) (map[string]int64, error)
}
