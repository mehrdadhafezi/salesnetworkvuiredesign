// Package api is the published surface of the organization module: the
// shadow import command contract. Records carry raw legacy evidence next to
// the few values the connector could classify; the organization module
// never sees legacy table names (they stay in internal/integration/legacy).
//
// Person and Employment are separate targets of one legacy profile row;
// User is an identity reference resolved through the published identity
// API (identity/api.UserResolver), never joined or written here.
package api

import (
	"context"
	"errors"
	"fmt"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ErrInvalidImport reports an import command that breaks its contract.
var ErrInvalidImport = errors.New("organization: invalid import")

// SourceRef is the provenance of one imported legacy row (shared shape).
type SourceRef = identityapi.SourceRef

// OrgUnitKind types the single organisation tree.
type OrgUnitKind string

const (
	Department OrgUnitKind = "DEPARTMENT"
	Unit       OrgUnitKind = "UNIT"
	Team       OrgUnitKind = "TEAM"
)

// ParentKind returns the kind a unit of kind k must hang under.
func (k OrgUnitKind) ParentKind() (OrgUnitKind, bool) {
	switch k {
	case Unit:
		return Department, true
	case Team:
		return Unit, true
	}
	return "", false
}

// EmploymentStatus is the target employment status. Only statuses evidenced
// in the official legacy code are mapped by the connector; any other raw
// value is UNMAPPED (raw kept), and a missing value is NOT_RECORDED.
type EmploymentStatus string

const (
	StatusActive      EmploymentStatus = "ACTIVE"
	StatusInactive    EmploymentStatus = "INACTIVE"
	StatusSuspended   EmploymentStatus = "SUSPENDED"
	StatusResigned    EmploymentStatus = "RESIGNED"
	StatusTerminated  EmploymentStatus = "TERMINATED"
	StatusProbation   EmploymentStatus = "PROBATION"
	StatusUnmapped    EmploymentStatus = "UNMAPPED"
	StatusNotRecorded EmploymentStatus = "NOT_RECORDED"
)

func (s EmploymentStatus) valid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusSuspended, StatusResigned, StatusTerminated, StatusProbation, StatusUnmapped, StatusNotRecorded:
		return true
	}
	return false
}

// RelationshipKind is the target relationship of an assignment.
type RelationshipKind string

const (
	ReportsTo            RelationshipKind = "REPORTS_TO"
	RelationshipUnmapped RelationshipKind = "UNMAPPED"
)

// LookupRecord is one position or level.
type LookupRecord struct {
	Source      SourceRef
	Slug        *string
	Label       *string
	Description *string
	SortOrder   *int
	IsActive    *bool
	IsSystem    *bool
	CreatedRaw  *string
	UpdatedRaw  *string
}

// OrgUnitRecord is one department, unit or team.
type OrgUnitRecord struct {
	Kind        OrgUnitKind
	Source      SourceRef
	ParentKey   *string // legacy key of the parent of ParentKind; nil = not recorded
	Slug        *string
	Label       *string
	Description *string
	SortOrder   *int
	IsActive    *bool
	CreatedRaw  *string
	UpdatedRaw  *string
}

// ProfileRecord is one legacy workforce profile: it is split into a Person
// and an Employment. Sensitive legacy columns are deliberately absent.
type ProfileRecord struct {
	Source             SourceRef
	UserKey            *string
	DisplayName        *string
	EmployeeCode       *string
	Status             EmploymentStatus
	StatusRaw          *string
	TypeRaw            *string
	HireDateRaw        *string
	TerminationDateRaw *string
	IsActive           *bool
	PositionKey        *string
	LevelKey           *string
	DepartmentKey      *string
	UnitKey            *string
	TeamKey            *string
	CreatedRaw         *string
	UpdatedRaw         *string
}

// AssignmentRecord is one reporting assignment between two profiles.
type AssignmentRecord struct {
	Source           SourceRef
	ChildKey         string
	ParentKey        *string
	Kind             RelationshipKind
	KindRaw          *string
	EffectiveFromRaw *string
	EffectiveToRaw   *string
	IsCurrent        *bool
	CurrentRaw       *string
	OriginRaw        *string
	RecordedByKey    *string
	CreatedRaw       *string
	UpdatedRaw       *string
}

// HistoryRecord is one append-only assignment history row.
type HistoryRecord struct {
	Source        SourceRef
	AssignmentKey *string
	ChildKey      *string
	OldParentKey  *string
	NewParentKey  *string
	ActorUserKey  *string
	ActionRaw     *string
	RecordedRaw   *string
}

// PositionRoleDefaultRecord is legacy evidence of the role a position
// suggests. It is never a grant.
type PositionRoleDefaultRecord struct {
	Source          SourceRef
	PositionKey     *string
	LegacyRole      *string
	RoleKey         *identityapi.Role
	DefaultLevelKey *string
	IsActive        *bool
}

// Entity names the parts of an organization import (completeness keys).
type Entity string

const (
	EntityPositions   Entity = "positions"
	EntityLevels      Entity = "levels"
	EntityDepartments Entity = "departments"
	EntityUnits       Entity = "units"
	EntityTeams       Entity = "teams"
	EntityProfiles    Entity = "profiles"
	EntityAssignments Entity = "assignments"
	EntityHistory     Entity = "history"
	EntityDefaults    Entity = "position_role_defaults"
)

// Entities lists every entity in import order.
func Entities() []Entity {
	return []Entity{EntityPositions, EntityLevels, EntityDepartments, EntityUnits, EntityTeams,
		EntityProfiles, EntityAssignments, EntityHistory, EntityDefaults}
}

// OrganizationImport is one organization import command for one batch.
type OrganizationImport struct {
	BatchID    string
	SnapshotID string
	// UserSystem is the source system of the account keys (identity refs).
	UserSystem  string
	Complete    map[Entity]bool
	Positions   []LookupRecord
	Levels      []LookupRecord
	OrgUnits    []OrgUnitRecord
	Profiles    []ProfileRecord
	Assignments []AssignmentRecord
	History     []HistoryRecord
	Defaults    []PositionRoleDefaultRecord
}

func validOptKey(k *string) bool { return k == nil || identityapi.ValidKey(*k) }

// Validate checks the command shape.
func (in OrganizationImport) Validate() error {
	check := func(s SourceRef) error {
		if err := s.Validate(); err != nil {
			return err
		}
		if s.BatchID != in.BatchID {
			return fmt.Errorf("%w: row from another batch", ErrInvalidImport)
		}
		return nil
	}
	if in.SnapshotID == "" || in.UserSystem == "" {
		return fmt.Errorf("%w: snapshot/user system", ErrInvalidImport)
	}
	for _, r := range append(append([]LookupRecord{}, in.Positions...), in.Levels...) {
		if err := check(r.Source); err != nil {
			return err
		}
	}
	for _, u := range in.OrgUnits {
		if err := check(u.Source); err != nil {
			return err
		}
		if _, hasParent := u.Kind.ParentKind(); u.Kind != Department && !hasParent {
			return fmt.Errorf("%w: org unit kind %q", ErrInvalidImport, u.Kind)
		}
		if (u.Kind == Department && u.ParentKey != nil) || !validOptKey(u.ParentKey) {
			return fmt.Errorf("%w: org unit parent %s", ErrInvalidImport, u.Source.Key)
		}
	}
	for _, p := range in.Profiles {
		if err := check(p.Source); err != nil {
			return err
		}
		if !p.Status.valid() {
			return fmt.Errorf("%w: employment status %q", ErrInvalidImport, p.Status)
		}
		for _, k := range []*string{p.PositionKey, p.LevelKey, p.DepartmentKey, p.UnitKey, p.TeamKey, p.UserKey} {
			if !validOptKey(k) {
				return fmt.Errorf("%w: profile reference %s", ErrInvalidImport, p.Source.Key)
			}
		}
	}
	for _, a := range in.Assignments {
		if err := check(a.Source); err != nil {
			return err
		}
		if !identityapi.ValidKey(a.ChildKey) || !validOptKey(a.ParentKey) || (a.Kind != ReportsTo && a.Kind != RelationshipUnmapped) {
			return fmt.Errorf("%w: assignment %s", ErrInvalidImport, a.Source.Key)
		}
	}
	for _, h := range in.History {
		if err := check(h.Source); err != nil {
			return err
		}
		for _, k := range []*string{h.AssignmentKey, h.ChildKey, h.OldParentKey, h.NewParentKey, h.ActorUserKey} {
			if !validOptKey(k) {
				return fmt.Errorf("%w: history %s", ErrInvalidImport, h.Source.Key)
			}
		}
	}
	for _, d := range in.Defaults {
		if err := check(d.Source); err != nil {
			return err
		}
		if !validOptKey(d.PositionKey) || !validOptKey(d.DefaultLevelKey) {
			return fmt.Errorf("%w: position role default %s", ErrInvalidImport, d.Source.Key)
		}
		if d.RoleKey != nil {
			if _, ok := identityapi.LookupRole(*d.RoleKey); !ok {
				return fmt.Errorf("%w: unknown target role %q", ErrInvalidImport, *d.RoleKey)
			}
		}
	}
	return nil
}

// ImportReport is the organization command result.
type ImportReport struct {
	Outcome  result.Status
	Replay   bool
	RunID    string
	Counts   map[string]int64
	Findings map[string]int64 // hierarchy finding counts by kind
	Issues   []identityapi.Issue
}

// Importer is the organization import command (organization/imports).
type Importer interface {
	ImportOrganization(ctx context.Context, in OrganizationImport) (ImportReport, error)
}

// SubjectContext is organisation context of one linked account: scope
// context for authorization, never authority.
type SubjectContext struct {
	PositionSlug *string
	LevelSlug    *string
	Reports      int // current direct reports
	Employments  int // employments linked to the account
}

// Reader is the published organization read surface used by reconciliation
// and the authorization diff.
type Reader interface {
	Counts(ctx context.Context) (map[string]int64, error)
	SubjectContexts(ctx context.Context) (map[string]SubjectContext, error)
}
