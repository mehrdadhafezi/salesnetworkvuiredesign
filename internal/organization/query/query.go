// Package query is the organization read service. Every call runs in a
// read-only, organization-owned transaction: a read can never create,
// backfill, repair or refresh anything (PostgreSQL rejects writes in a READ
// ONLY transaction). Filters and sorts are allowlisted; nothing from input
// reaches an SQL identifier.
package query

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pagination"
)

// ErrInvalidQuery reports a filter, sort, cursor or limit outside the allowlist.
var ErrInvalidQuery = errors.New("organization/query: invalid query")

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidQuery}, a...)...)
}

func checkID(name string, p *string) error {
	if p != nil && !uuidPattern.MatchString(*p) {
		return invalid("%s must be a native id", name)
	}
	return nil
}

// Service reads the organization model.
type Service struct{ runner *db.Runner }

var _ api.Reader = (*Service)(nil)

// New returns the read service over an organization-owned runner.
func New(runner *db.Runner) *Service { return &Service{runner: runner} }

func (s *Service) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// PersonsParams are the allowlisted list inputs.
type PersonsParams struct {
	Filter store.PersonFilter
	Sort   string
	Cursor *string
	Limit  *int
}

// PersonsPage is one page of persons.
type PersonsPage struct {
	Persons    []store.Person
	NextCursor *string
}

// ListPersons returns one page.
func (s *Service) ListPersons(ctx context.Context, p PersonsParams) (PersonsPage, error) {
	if p.Sort == "" {
		p.Sort = "display_name"
	}
	sort, ok := store.PersonSorts[p.Sort]
	if !ok {
		return PersonsPage{}, invalid("sort %q", p.Sort)
	}
	f := p.Filter
	if f.Query != nil && (len(*f.Query) == 0 || len(*f.Query) > 100) {
		return PersonsPage{}, invalid("q length")
	}
	if f.Status != nil && !store.EmploymentStatuses[*f.Status] {
		return PersonsPage{}, invalid("employment_status")
	}
	for name, v := range map[string]*string{"position_id": f.PositionID, "level_id": f.LevelID, "org_unit_id": f.OrgUnitID} {
		if err := checkID(name, v); err != nil {
			return PersonsPage{}, err
		}
	}
	limit, err := pagination.Limit(p.Limit)
	if err != nil {
		return PersonsPage{}, invalid("%v", err)
	}
	var afterV, afterID *string
	if p.Cursor != nil {
		c, err := pagination.Decode(*p.Cursor, p.Sort)
		if err != nil || !uuidPattern.MatchString(c.ID) {
			return PersonsPage{}, invalid("cursor")
		}
		afterV, afterID = &c.Value, &c.ID
	}
	var rows []store.Person
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListPersons(ctx, tx, f, sort, afterV, afterID, limit)
		return err
	}); err != nil {
		return PersonsPage{}, err
	}
	page := PersonsPage{Persons: rows}
	if len(rows) > limit {
		page.Persons = rows[:limit]
		last := page.Persons[limit-1]
		cur, err := pagination.Encode(pagination.Cursor{Sort: p.Sort, Value: store.SortValue(last, p.Sort), ID: last.PersonID})
		if err != nil {
			return PersonsPage{}, err
		}
		page.NextCursor = &cur
	}
	return page, nil
}

// PersonDetail is one person with its organisation context.
type PersonDetail struct {
	Person        store.Person
	Assignments   []store.Assignment // as child, current and past
	DirectReports []store.PersonRef
	History       []store.History
	Findings      []store.Finding
}

// GetPerson returns one person's detail.
func (s *Service) GetPerson(ctx context.Context, id string) (PersonDetail, bool, error) {
	if !uuidPattern.MatchString(id) {
		return PersonDetail{}, false, invalid("person id")
	}
	var (
		d  PersonDetail
		ok bool
	)
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if d.Person, ok, err = store.GetPerson(ctx, tx, id); err != nil || !ok {
			return err
		}
		child := "CHILD"
		if d.Assignments, err = store.ListAssignments(ctx, tx, store.AssignmentFilter{PersonID: &id, Role: &child}, nil, pagination.MaxLimit); err != nil {
			return err
		}
		if d.DirectReports, err = store.DirectReports(ctx, tx, d.Person.Employment.EmploymentID); err != nil {
			return err
		}
		if d.History, err = store.HistoryForEmployment(ctx, tx, d.Person.Employment.EmploymentID); err != nil {
			return err
		}
		d.Findings, err = store.ListFindings(ctx, tx, store.FindingFilter{PersonID: &id}, nil, nil, pagination.MaxLimit)
		return err
	})
	return d, ok, err
}

// AssignmentsParams are the allowlisted inputs.
type AssignmentsParams struct {
	Filter store.AssignmentFilter
	Cursor *string
	Limit  *int
}

// AssignmentsPage is one page of assignments.
type AssignmentsPage struct {
	Assignments []store.Assignment
	NextCursor  *string
}

const idSort = "id"

// ListAssignments returns one page.
func (s *Service) ListAssignments(ctx context.Context, p AssignmentsParams) (AssignmentsPage, error) {
	f := p.Filter
	if err := checkID("person_id", f.PersonID); err != nil {
		return AssignmentsPage{}, err
	}
	if f.Role != nil && (*f.Role != "CHILD" && *f.Role != "PARENT" || f.PersonID == nil) {
		return AssignmentsPage{}, invalid("relation requires person_id and is CHILD or PARENT")
	}
	if f.Kind != nil && *f.Kind != string(api.ReportsTo) && *f.Kind != string(api.RelationshipUnmapped) {
		return AssignmentsPage{}, invalid("relationship_kind")
	}
	limit, err := pagination.Limit(p.Limit)
	if err != nil {
		return AssignmentsPage{}, invalid("%v", err)
	}
	var after *string
	if p.Cursor != nil {
		c, err := pagination.Decode(*p.Cursor, idSort)
		if err != nil || !uuidPattern.MatchString(c.ID) {
			return AssignmentsPage{}, invalid("cursor")
		}
		after = &c.ID
	}
	var rows []store.Assignment
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListAssignments(ctx, tx, f, after, limit)
		return err
	}); err != nil {
		return AssignmentsPage{}, err
	}
	page := AssignmentsPage{Assignments: rows}
	if len(rows) > limit {
		page.Assignments = rows[:limit]
		cur, err := pagination.Encode(pagination.Cursor{Sort: idSort, ID: page.Assignments[limit-1].AssignmentID})
		if err != nil {
			return AssignmentsPage{}, err
		}
		page.NextCursor = &cur
	}
	return page, nil
}

// FindingsParams are the allowlisted inputs.
type FindingsParams struct {
	Filter store.FindingFilter
	Cursor *string
	Limit  *int
}

// FindingsPage is one page of findings.
type FindingsPage struct {
	Findings   []store.Finding
	NextCursor *string
}

// FindingKinds, severities and subject types allowlisted for filters.
var (
	FindingSeverities = map[string]bool{"CONFLICT": true, "UNKNOWN": true, "INFO": true}
	FindingSubjects   = map[string]bool{"ASSIGNMENT": true, "EMPLOYMENT": true, "ORG_UNIT": true, "HISTORY": true}
)

// ListFindings returns one page.
func (s *Service) ListFindings(ctx context.Context, p FindingsParams, kinds map[string]bool) (FindingsPage, error) {
	f := p.Filter
	if f.Kind != nil && !kinds[*f.Kind] {
		return FindingsPage{}, invalid("kind")
	}
	if f.Severity != nil && !FindingSeverities[*f.Severity] {
		return FindingsPage{}, invalid("severity")
	}
	if f.SubjectType != nil && !FindingSubjects[*f.SubjectType] {
		return FindingsPage{}, invalid("subject_type")
	}
	if err := checkID("person_id", f.PersonID); err != nil {
		return FindingsPage{}, err
	}
	limit, err := pagination.Limit(p.Limit)
	if err != nil {
		return FindingsPage{}, invalid("%v", err)
	}
	var afterK, afterID *string
	if p.Cursor != nil {
		c, err := pagination.Decode(*p.Cursor, "kind")
		if err != nil || !uuidPattern.MatchString(c.ID) || !kinds[c.Value] {
			return FindingsPage{}, invalid("cursor")
		}
		afterK, afterID = &c.Value, &c.ID
	}
	var rows []store.Finding
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListFindings(ctx, tx, f, afterK, afterID, limit)
		return err
	}); err != nil {
		return FindingsPage{}, err
	}
	page := FindingsPage{Findings: rows}
	if len(rows) > limit {
		page.Findings = rows[:limit]
		last := page.Findings[limit-1]
		cur, err := pagination.Encode(pagination.Cursor{Sort: "kind", Value: last.Kind, ID: last.FindingID})
		if err != nil {
			return FindingsPage{}, err
		}
		page.NextCursor = &cur
	}
	return page, nil
}

// OrgUnitsPage is one page of org units.
type OrgUnitsPage struct {
	Units      []store.OrgUnit
	NextCursor *string
}

// ListOrgUnits returns one page ordered by kind (department, unit, team).
func (s *Service) ListOrgUnits(ctx context.Context, kind, cursor *string, limitIn *int) (OrgUnitsPage, error) {
	if kind != nil && store.KindRank(*kind) == 0 {
		return OrgUnitsPage{}, invalid("kind")
	}
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return OrgUnitsPage{}, invalid("%v", err)
	}
	var afterRank *int
	var afterID *string
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "kind")
		r, convErr := strconv.Atoi(c.Value)
		if err != nil || convErr != nil || r < 1 || r > 3 || !uuidPattern.MatchString(c.ID) {
			return OrgUnitsPage{}, invalid("cursor")
		}
		afterRank, afterID = &r, &c.ID
	}
	var rows []store.OrgUnit
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListOrgUnits(ctx, tx, kind, afterRank, afterID, limit)
		return err
	}); err != nil {
		return OrgUnitsPage{}, err
	}
	page := OrgUnitsPage{Units: rows}
	if len(rows) > limit {
		page.Units = rows[:limit]
		last := page.Units[limit-1]
		cur, err := pagination.Encode(pagination.Cursor{Sort: "kind", Value: strconv.Itoa(store.KindRank(last.Kind)), ID: last.OrgUnitID})
		if err != nil {
			return OrgUnitsPage{}, err
		}
		page.NextCursor = &cur
	}
	return page, nil
}

// LookupsPage is one page of positions or levels.
type LookupsPage struct {
	Items      []store.Lookup
	NextCursor *string
}

// ListLookups returns one page of "position" or "level" rows.
func (s *Service) ListLookups(ctx context.Context, which string, cursor *string, limitIn *int) (LookupsPage, error) {
	limit, err := pagination.Limit(limitIn)
	if err != nil {
		return LookupsPage{}, invalid("%v", err)
	}
	var afterV, afterID *string
	if cursor != nil {
		c, err := pagination.Decode(*cursor, "slug")
		if err != nil || !uuidPattern.MatchString(c.ID) {
			return LookupsPage{}, invalid("cursor")
		}
		afterV, afterID = &c.Value, &c.ID
	}
	var rows []store.Lookup
	if err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		rows, err = store.ListLookups(ctx, tx, which, afterV, afterID, limit)
		return err
	}); err != nil {
		return LookupsPage{}, err
	}
	page := LookupsPage{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		last := page.Items[limit-1]
		v := ""
		if last.Slug != nil {
			v = *last.Slug
		}
		cur, err := pagination.Encode(pagination.Cursor{Sort: "slug", Value: v, ID: last.ID})
		if err != nil {
			return LookupsPage{}, err
		}
		page.NextCursor = &cur
	}
	return page, nil
}

// Counts implements api.Reader.
func (s *Service) Counts(ctx context.Context) (map[string]int64, error) {
	var out map[string]int64
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.Counts(ctx, tx)
		return err
	})
	return out, err
}

// SubjectContexts implements api.Reader.
func (s *Service) SubjectContexts(ctx context.Context) (map[string]api.SubjectContext, error) {
	var out map[string]api.SubjectContext
	err := s.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		out, err = store.SubjectContexts(ctx, tx)
		return err
	})
	return out, err
}
