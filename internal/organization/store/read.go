package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Provenance is the read-back provenance of a row.
type Provenance = identityapi.Provenance

// LookupRef is a compact position/level reference.
type LookupRef struct {
	ID    string
	Slug  *string
	Label *string
}

// UnitRef is a compact org unit reference.
type UnitRef struct {
	ID    string
	Kind  string
	Label *string
}

// PersonRef is a compact person reference (for managers and reports).
type PersonRef struct {
	PersonID     string
	EmploymentID string
	DisplayName  *string
}

// Employment is the employment read model.
type Employment struct {
	EmploymentID       string
	Status             string
	StatusRaw          *string
	TypeRaw            *string
	EmployeeCode       *string
	HireDate           *time.Time
	HireDateRaw        *string
	TerminationDate    *time.Time
	TerminationDateRaw *string
	IsActive           *bool
	UserID             *string
	UserLink           string
	Position           *LookupRef
	PositionState      string
	Level              *LookupRef
	LevelState         string
	Department         *UnitRef
	DepartmentState    string
	Unit               *UnitRef
	UnitState          string
	Team               *UnitRef
	TeamState          string
	Source             Provenance
}

// Person is one person row with its employment summary.
type Person struct {
	PersonID     string
	DisplayName  *string
	Source       Provenance
	Employment   Employment
	ManagerState string // NONE | RESOLVED | MULTIPLE | MISSING
	Manager      *PersonRef
	FindingCount int64
}

// PersonFilter is the allowlisted persons filter set.
type PersonFilter struct {
	Query       *string
	Status      *string
	PositionID  *string
	LevelID     *string
	OrgUnitID   *string
	IsActive    *bool
	HasFindings *bool
}

// Sort is an allowlisted sort: Expr is a constant SQL expression.
type Sort struct {
	Expr string
	Desc bool
}

// PersonSorts maps public sort keys to constant expressions.
var PersonSorts = map[string]Sort{
	"display_name":       {Expr: "COALESCE(p.display_name, '')"},
	"-display_name":      {Expr: "COALESCE(p.display_name, '')", Desc: true},
	"employee_code":      {Expr: "COALESCE(e.employee_code, '')"},
	"-employee_code":     {Expr: "COALESCE(e.employee_code, '')", Desc: true},
	"employment_status":  {Expr: "e.employment_status"},
	"-employment_status": {Expr: "e.employment_status", Desc: true},
}

// EmploymentStatuses is the allowlist of the status filter.
var EmploymentStatuses = map[string]bool{"ACTIVE": true, "INACTIVE": true, "SUSPENDED": true, "RESIGNED": true,
	"TERMINATED": true, "PROBATION": true, "UNMAPPED": true, "NOT_RECORDED": true}

const personSelect = `SELECT p.person_id::text, p.display_name, p.source_system, p.source_spec, p.source_key,
	p.source_batch_id::text, p.imported_at, p.refreshed_at, p.presence,
	e.employment_id::text, e.employment_status, e.employment_status_raw, e.employment_type_raw, e.employee_code,
	e.hire_date, e.hire_date_raw, e.termination_date, e.termination_date_raw, e.is_active, e.user_id::text, e.user_link_state,
	pos.position_id::text, pos.slug, pos.label, e.position_ref_state,
	lv.level_id::text, lv.slug, lv.label, e.level_ref_state,
	d.org_unit_id::text, d.label, e.department_ref_state,
	u.org_unit_id::text, u.label, e.unit_ref_state,
	tm.org_unit_id::text, tm.label, e.team_ref_state,
	e.source_system, e.source_spec, e.source_key, e.source_batch_id::text, e.imported_at, e.refreshed_at, e.presence,
	cm.n, mp.person_id::text, me.employment_id::text, mp.display_name, cm.missing, fc.n
FROM organization.person p
JOIN organization.employment e ON e.person_id = p.person_id
LEFT JOIN organization.position pos ON pos.position_id = e.position_id
LEFT JOIN organization.level lv ON lv.level_id = e.level_id
LEFT JOIN organization.org_unit d ON d.org_unit_id = e.department_id
LEFT JOIN organization.org_unit u ON u.org_unit_id = e.unit_id
LEFT JOIN organization.org_unit tm ON tm.org_unit_id = e.team_id
LEFT JOIN LATERAL (
	SELECT count(*) AS n, (array_agg(a.parent_employment_id))[1] AS parent_id, bool_or(a.parent_ref_state = 'MISSING') AS missing
	FROM organization.org_assignment a
	WHERE a.child_employment_id = e.employment_id AND a.is_current AND a.relationship_kind = 'REPORTS_TO' AND a.presence = 'PRESENT'
) cm ON true
LEFT JOIN organization.employment me ON me.employment_id = cm.parent_id AND cm.n = 1
LEFT JOIN organization.person mp ON mp.person_id = me.person_id
LEFT JOIN LATERAL (
	SELECT count(*) AS n FROM organization.hierarchy_finding f
	WHERE f.subject_id = e.employment_id
	   OR f.subject_id IN (SELECT a2.assignment_id FROM organization.org_assignment a2 WHERE a2.child_employment_id = e.employment_id)
) fc ON true`

func scanPerson(row pgx.Row) (Person, error) {
	var (
		p                                     Person
		e                                     = &p.Employment
		posID, posSlug, posLabel              *string
		lvID, lvSlug, lvLabel                 *string
		dID, dLabel, uID, uLabel, tID, tLabel *string
		hire, term                            *time.Time
		mgrPerson, mgrEmp, mgrName            *string
		mgrN                                  int64
		mgrMissing                            *bool
	)
	err := row.Scan(&p.PersonID, &p.DisplayName, &p.Source.System, &p.Source.Spec, &p.Source.Key, &p.Source.BatchID,
		&p.Source.ImportedAt, &p.Source.RefreshedAt, &p.Source.Presence,
		&e.EmploymentID, &e.Status, &e.StatusRaw, &e.TypeRaw, &e.EmployeeCode, &hire, &e.HireDateRaw, &term, &e.TerminationDateRaw,
		&e.IsActive, &e.UserID, &e.UserLink,
		&posID, &posSlug, &posLabel, &e.PositionState, &lvID, &lvSlug, &lvLabel, &e.LevelState,
		&dID, &dLabel, &e.DepartmentState, &uID, &uLabel, &e.UnitState, &tID, &tLabel, &e.TeamState,
		&e.Source.System, &e.Source.Spec, &e.Source.Key, &e.Source.BatchID, &e.Source.ImportedAt, &e.Source.RefreshedAt, &e.Source.Presence,
		&mgrN, &mgrPerson, &mgrEmp, &mgrName, &mgrMissing, &p.FindingCount)
	if err != nil {
		return p, err
	}
	p.Source.ImportedAt, p.Source.RefreshedAt = p.Source.ImportedAt.UTC(), p.Source.RefreshedAt.UTC()
	e.Source.ImportedAt, e.Source.RefreshedAt = e.Source.ImportedAt.UTC(), e.Source.RefreshedAt.UTC()
	e.HireDate, e.TerminationDate = hire, term
	if posID != nil {
		e.Position = &LookupRef{ID: *posID, Slug: posSlug, Label: posLabel}
	}
	if lvID != nil {
		e.Level = &LookupRef{ID: *lvID, Slug: lvSlug, Label: lvLabel}
	}
	if dID != nil {
		e.Department = &UnitRef{ID: *dID, Kind: "DEPARTMENT", Label: dLabel}
	}
	if uID != nil {
		e.Unit = &UnitRef{ID: *uID, Kind: "UNIT", Label: uLabel}
	}
	if tID != nil {
		e.Team = &UnitRef{ID: *tID, Kind: "TEAM", Label: tLabel}
	}
	switch {
	case mgrN == 0:
		p.ManagerState = "NONE"
	case mgrN > 1:
		p.ManagerState = "MULTIPLE"
	case mgrPerson != nil:
		p.ManagerState = "RESOLVED"
		p.Manager = &PersonRef{PersonID: *mgrPerson, EmploymentID: *mgrEmp, DisplayName: mgrName}
	case mgrMissing != nil && *mgrMissing:
		p.ManagerState = "MISSING"
	default:
		p.ManagerState = "NONE"
	}
	return p, nil
}

// ListPersons returns one keyset page (limit+1 rows signal more).
func ListPersons(ctx context.Context, tx db.Tx, f PersonFilter, s Sort, afterValue, afterID *string, limit int) ([]Person, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	op, dir := ">", "ASC"
	if s.Desc {
		op, dir = "<", "DESC"
	}
	q := personSelect + `
WHERE ($1::text IS NULL OR p.display_name ILIKE '%' || $1 || '%' OR e.employee_code ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR e.employment_status = $2)
  AND ($3::uuid IS NULL OR e.position_id = $3)
  AND ($4::uuid IS NULL OR e.level_id = $4)
  AND ($5::uuid IS NULL OR $5 IN (e.department_id, e.unit_id, e.team_id))
  AND ($6::boolean IS NULL OR e.is_active = $6)
  AND ($7::boolean IS NULL OR (fc.n > 0) = $7)
  AND ($8::text IS NULL OR (` + s.Expr + `, p.person_id) ` + op + ` ($8, $9::uuid))
ORDER BY ` + s.Expr + ` ` + dir + `, p.person_id ` + dir + ` LIMIT $10`
	var query *string
	if f.Query != nil {
		v := escapeLike(*f.Query)
		query = &v
	}
	rows, err := tx.Query(ctx, q, query, f.Status, f.PositionID, f.LevelID, f.OrgUnitID, f.IsActive, f.HasFindings, afterValue, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SortValue returns the keyset value of p for the public sort key.
func SortValue(p Person, sort string) string {
	switch sort {
	case "employee_code", "-employee_code":
		if p.Employment.EmployeeCode != nil {
			return *p.Employment.EmployeeCode
		}
		return ""
	case "employment_status", "-employment_status":
		return p.Employment.Status
	}
	if p.DisplayName != nil {
		return *p.DisplayName
	}
	return ""
}

// GetPerson returns one person.
func GetPerson(ctx context.Context, tx db.Tx, id string) (Person, bool, error) {
	if err := ReadTx(tx); err != nil {
		return Person{}, false, err
	}
	p, err := scanPerson(tx.QueryRow(ctx, personSelect+` WHERE p.person_id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Person{}, false, nil
	}
	return p, err == nil, err
}

// Assignment is one reporting assignment read model.
type Assignment struct {
	AssignmentID     string
	Child            *PersonRef
	ChildState       string
	Parent           *PersonRef
	ParentState      string
	Kind             string
	KindRaw          *string
	EffectiveFromRaw *string
	EffectiveToRaw   *string
	IsCurrent        *bool
	CurrentRaw       *string
	Source           Provenance
	FindingKinds     []string
}

// AssignmentFilter is the allowlisted assignment filter set.
type AssignmentFilter struct {
	PersonID    *string // child or parent person
	Role        *string // CHILD | PARENT (with PersonID)
	Current     *bool
	Kind        *string
	HasFindings *bool
}

const assignmentSelect = `SELECT a.assignment_id::text,
	cp.person_id::text, a.child_employment_id::text, cp.display_name, a.child_ref_state,
	pp.person_id::text, a.parent_employment_id::text, pp.display_name, a.parent_ref_state,
	a.relationship_kind, a.relationship_raw, a.effective_from_raw, a.effective_to_raw, a.is_current, a.current_raw,
	a.source_system, a.source_spec, a.source_key, a.source_batch_id::text, a.imported_at, a.refreshed_at, a.presence,
	COALESCE((SELECT array_agg(DISTINCT f.kind ORDER BY f.kind) FROM organization.hierarchy_finding f WHERE f.subject_id = a.assignment_id), '{}')
FROM organization.org_assignment a
LEFT JOIN organization.employment ce ON ce.employment_id = a.child_employment_id
LEFT JOIN organization.person cp ON cp.person_id = ce.person_id
LEFT JOIN organization.employment pe ON pe.employment_id = a.parent_employment_id
LEFT JOIN organization.person pp ON pp.person_id = pe.person_id`

func scanAssignment(row pgx.Row) (Assignment, error) {
	var (
		a                    Assignment
		cPerson, cEmp, cName *string
		pPerson, pEmp, pName *string
	)
	err := row.Scan(&a.AssignmentID, &cPerson, &cEmp, &cName, &a.ChildState, &pPerson, &pEmp, &pName, &a.ParentState,
		&a.Kind, &a.KindRaw, &a.EffectiveFromRaw, &a.EffectiveToRaw, &a.IsCurrent, &a.CurrentRaw,
		&a.Source.System, &a.Source.Spec, &a.Source.Key, &a.Source.BatchID, &a.Source.ImportedAt, &a.Source.RefreshedAt, &a.Source.Presence,
		&a.FindingKinds)
	a.Source.ImportedAt, a.Source.RefreshedAt = a.Source.ImportedAt.UTC(), a.Source.RefreshedAt.UTC()
	if cPerson != nil {
		a.Child = &PersonRef{PersonID: *cPerson, EmploymentID: *cEmp, DisplayName: cName}
	}
	if pPerson != nil {
		a.Parent = &PersonRef{PersonID: *pPerson, EmploymentID: *pEmp, DisplayName: pName}
	}
	return a, err
}

// ListAssignments returns one keyset page ordered by assignment id.
func ListAssignments(ctx context.Context, tx db.Tx, f AssignmentFilter, afterID *string, limit int) ([]Assignment, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	q := assignmentSelect + `
WHERE ($1::uuid IS NULL OR (COALESCE($2::text, 'ANY') IN ('ANY', 'CHILD') AND cp.person_id = $1)
                        OR (COALESCE($2::text, 'ANY') IN ('ANY', 'PARENT') AND pp.person_id = $1))
  AND ($3::boolean IS NULL OR a.is_current = $3)
  AND ($4::text IS NULL OR a.relationship_kind = $4)
  AND ($5::boolean IS NULL OR EXISTS (SELECT 1 FROM organization.hierarchy_finding f WHERE f.subject_id = a.assignment_id) = $5)
  AND ($6::uuid IS NULL OR a.assignment_id > $6)
ORDER BY a.assignment_id LIMIT $7`
	rows, err := tx.Query(ctx, q, f.PersonID, f.Role, f.Current, f.Kind, f.HasFindings, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// History is one append-only history row (free text excluded).
type History struct {
	HistoryID   string
	OldParent   *PersonRef
	NewParent   *PersonRef
	ActionRaw   *string
	RecordedRaw *string
	Source      Provenance
}

// HistoryForEmployment lists history rows whose child is the employment's
// legacy profile key, in recorded order.
func HistoryForEmployment(ctx context.Context, tx db.Tx, employmentID string) ([]History, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT h.history_id::text, op.person_id::text, oe.employment_id::text, op.display_name,
		np.person_id::text, ne.employment_id::text, np.display_name, h.action_raw, h.recorded_raw,
		h.source_system, h.source_spec, h.source_key, h.source_batch_id::text, h.imported_at, h.imported_at
		FROM organization.org_assignment_history h
		JOIN organization.employment e ON e.employment_id = $1 AND h.child_source_key = e.source_key AND h.source_system = e.source_system
		LEFT JOIN organization.employment oe ON oe.source_key = h.old_parent_source_key AND oe.source_system = h.source_system
		LEFT JOIN organization.person op ON op.person_id = oe.person_id
		LEFT JOIN organization.employment ne ON ne.source_key = h.new_parent_source_key AND ne.source_system = h.source_system
		LEFT JOIN organization.person np ON np.person_id = ne.person_id
		ORDER BY h.recorded_raw NULLS FIRST, length(h.source_key), h.source_key`, employmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []History
	for rows.Next() {
		var (
			h          History
			op, oe, on *string
			np, ne, nn *string
		)
		if err := rows.Scan(&h.HistoryID, &op, &oe, &on, &np, &ne, &nn, &h.ActionRaw, &h.RecordedRaw,
			&h.Source.System, &h.Source.Spec, &h.Source.Key, &h.Source.BatchID, &h.Source.ImportedAt, &h.Source.RefreshedAt); err != nil {
			return nil, err
		}
		h.Source.Presence = "PRESENT"
		h.Source.ImportedAt, h.Source.RefreshedAt = h.Source.ImportedAt.UTC(), h.Source.RefreshedAt.UTC()
		if op != nil {
			h.OldParent = &PersonRef{PersonID: *op, EmploymentID: *oe, DisplayName: on}
		}
		if np != nil {
			h.NewParent = &PersonRef{PersonID: *np, EmploymentID: *ne, DisplayName: nn}
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Finding is one stored hierarchy finding.
type Finding struct {
	FindingID   string
	Kind        string
	Severity    string
	SubjectType string
	SubjectID   *string
	SubjectKey  *string
	RelatedID   *string
	RelatedKey  *string
	Detail      string
}

// FindingFilter is the allowlisted findings filter set.
type FindingFilter struct {
	Kind        *string
	Severity    *string
	SubjectType *string
	PersonID    *string // findings about the person's employment, assignments or history
}

// ListFindings returns one keyset page ordered by (kind, finding id).
func ListFindings(ctx context.Context, tx db.Tx, f FindingFilter, afterKind, afterID *string, limit int) ([]Finding, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT f.finding_id::text, f.kind, f.severity, f.subject_type, f.subject_id::text, f.subject_source_key,
		f.related_id::text, f.related_source_key, f.detail
		FROM organization.hierarchy_finding f
		WHERE ($1::text IS NULL OR f.kind = $1)
		  AND ($2::text IS NULL OR f.severity = $2)
		  AND ($3::text IS NULL OR f.subject_type = $3)
		  AND ($4::uuid IS NULL OR f.subject_id IN (
				SELECT e.employment_id FROM organization.employment e WHERE e.person_id = $4
				UNION ALL SELECT a.assignment_id FROM organization.org_assignment a JOIN organization.employment e ON e.employment_id = a.child_employment_id WHERE e.person_id = $4
				UNION ALL SELECT h.history_id FROM organization.org_assignment_history h JOIN organization.employment e ON e.source_key = h.child_source_key AND e.source_system = h.source_system WHERE e.person_id = $4))
		  AND ($5::text IS NULL OR (f.kind, f.finding_id) > ($5, $6::uuid))
		ORDER BY f.kind, f.finding_id LIMIT $7`, f.Kind, f.Severity, f.SubjectType, f.PersonID, afterKind, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Finding
	for rows.Next() {
		var x Finding
		if err := rows.Scan(&x.FindingID, &x.Kind, &x.Severity, &x.SubjectType, &x.SubjectID, &x.SubjectKey, &x.RelatedID, &x.RelatedKey, &x.Detail); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// OrgUnit is one org unit read model.
type OrgUnit struct {
	OrgUnitID    string
	Kind         string
	Slug         *string
	Label        *string
	SortOrder    *int32
	IsActive     *bool
	ParentID     *string
	ParentState  string
	MemberCount  int64
	FindingKinds []string
	Source       Provenance
}

var kindRank = map[string]int{"DEPARTMENT": 1, "UNIT": 2, "TEAM": 3}

// KindRank returns the keyset rank of a unit kind.
func KindRank(kind string) int { return kindRank[kind] }

// ListOrgUnits returns one keyset page ordered by (kind rank, id).
func ListOrgUnits(ctx context.Context, tx db.Tx, kind *string, afterRank *int, afterID *string, limit int) ([]OrgUnit, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT o.org_unit_id::text, o.kind, o.slug, o.label, o.sort_order, o.is_active,
		o.parent_org_unit_id::text, o.parent_state,
		(SELECT count(*) FROM organization.employment e WHERE o.org_unit_id IN (e.department_id, e.unit_id, e.team_id) AND e.presence = 'PRESENT'),
		COALESCE((SELECT array_agg(DISTINCT f.kind ORDER BY f.kind) FROM organization.hierarchy_finding f WHERE f.subject_id = o.org_unit_id), '{}'),
		o.source_system, o.source_spec, o.source_key, o.source_batch_id::text, o.imported_at, o.refreshed_at, o.presence,
		CASE o.kind WHEN 'DEPARTMENT' THEN 1 WHEN 'UNIT' THEN 2 ELSE 3 END AS rank
		FROM organization.org_unit o
		WHERE ($1::text IS NULL OR o.kind = $1)
		  AND ($2::int IS NULL OR (CASE o.kind WHEN 'DEPARTMENT' THEN 1 WHEN 'UNIT' THEN 2 ELSE 3 END, o.org_unit_id) > ($2, $3::uuid))
		ORDER BY rank, o.org_unit_id LIMIT $4`, kind, afterRank, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrgUnit
	for rows.Next() {
		var (
			o    OrgUnit
			rank int
		)
		if err := rows.Scan(&o.OrgUnitID, &o.Kind, &o.Slug, &o.Label, &o.SortOrder, &o.IsActive, &o.ParentID, &o.ParentState,
			&o.MemberCount, &o.FindingKinds, &o.Source.System, &o.Source.Spec, &o.Source.Key, &o.Source.BatchID,
			&o.Source.ImportedAt, &o.Source.RefreshedAt, &o.Source.Presence, &rank); err != nil {
			return nil, err
		}
		o.Source.ImportedAt, o.Source.RefreshedAt = o.Source.ImportedAt.UTC(), o.Source.RefreshedAt.UTC()
		out = append(out, o)
	}
	return out, rows.Err()
}

// Lookup is one position or level read model.
type Lookup struct {
	ID          string
	Slug        *string
	Label       *string
	Description *string
	SortOrder   *int32
	IsActive    *bool
	IsSystem    *bool
	Source      Provenance
}

// ListLookups returns one keyset page of positions or levels ordered by
// (slug, id). which must be "position" or "level".
func ListLookups(ctx context.Context, tx db.Tx, which string, afterSlug, afterID *string, limit int) ([]Lookup, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	var t table
	switch which {
	case "position":
		t = positionTable
	case "level":
		t = levelTable
	default:
		return nil, fmt.Errorf("organization/store: unknown lookup %q", which)
	}
	rows, err := tx.Query(ctx, `SELECT `+t.id+`::text, slug, label, description, sort_order, is_active, is_system,
		source_system, source_spec, source_key, source_batch_id::text, imported_at, refreshed_at, presence
		FROM `+t.name+`
		WHERE ($1::text IS NULL OR (COALESCE(slug, ''), `+t.id+`) > ($1, $2::uuid))
		ORDER BY COALESCE(slug, ''), `+t.id+` LIMIT $3`, afterSlug, afterID, limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Lookup
	for rows.Next() {
		var l Lookup
		if err := rows.Scan(&l.ID, &l.Slug, &l.Label, &l.Description, &l.SortOrder, &l.IsActive, &l.IsSystem,
			&l.Source.System, &l.Source.Spec, &l.Source.Key, &l.Source.BatchID, &l.Source.ImportedAt, &l.Source.RefreshedAt, &l.Source.Presence); err != nil {
			return nil, err
		}
		l.Source.ImportedAt, l.Source.RefreshedAt = l.Source.ImportedAt.UTC(), l.Source.RefreshedAt.UTC()
		out = append(out, l)
	}
	return out, rows.Err()
}

// DirectReports lists the persons currently reporting to an employment.
func DirectReports(ctx context.Context, tx db.Tx, employmentID string) ([]PersonRef, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT p.person_id::text, e.employment_id::text, p.display_name
		FROM organization.org_assignment a
		JOIN organization.employment e ON e.employment_id = a.child_employment_id
		JOIN organization.person p ON p.person_id = e.person_id
		WHERE a.parent_employment_id = $1 AND a.is_current AND a.relationship_kind = 'REPORTS_TO' AND a.presence = 'PRESENT'
		ORDER BY COALESCE(p.display_name, ''), p.person_id`, employmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PersonRef
	for rows.Next() {
		var r PersonRef
		if err := rows.Scan(&r.PersonID, &r.EmploymentID, &r.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Counts returns reconciliation counts of the stored model (rows PRESENT in
// the latest import; history and findings are counted whole).
func Counts(ctx context.Context, tx db.Tx) (map[string]int64, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	out := map[string]int64{}
	queries := map[string]string{
		"persons":                       `SELECT count(*) FROM organization.person WHERE presence = 'PRESENT'`,
		"employments":                   `SELECT count(*) FROM organization.employment WHERE presence = 'PRESENT'`,
		"employments_linked":            `SELECT count(*) FROM organization.employment WHERE user_link_state = 'LINKED'`,
		"employments_shared_user":       `SELECT count(*) FROM organization.employment WHERE user_link_state = 'SHARED_USER'`,
		"employments_user_not_imported": `SELECT count(*) FROM organization.employment WHERE user_link_state = 'USER_NOT_IMPORTED'`,
		"employments_user_not_recorded": `SELECT count(*) FROM organization.employment WHERE user_link_state = 'USER_NOT_RECORDED'`,
		"employments_status_unmapped":   `SELECT count(*) FROM organization.employment WHERE employment_status = 'UNMAPPED'`,
		"positions":                     `SELECT count(*) FROM organization.position WHERE presence = 'PRESENT'`,
		"levels":                        `SELECT count(*) FROM organization.level WHERE presence = 'PRESENT'`,
		"departments":                   `SELECT count(*) FROM organization.org_unit WHERE kind = 'DEPARTMENT' AND presence = 'PRESENT'`,
		"units":                         `SELECT count(*) FROM organization.org_unit WHERE kind = 'UNIT' AND presence = 'PRESENT'`,
		"teams":                         `SELECT count(*) FROM organization.org_unit WHERE kind = 'TEAM' AND presence = 'PRESENT'`,
		"assignments":                   `SELECT count(*) FROM organization.org_assignment WHERE presence = 'PRESENT'`,
		"assignments_current":           `SELECT count(*) FROM organization.org_assignment WHERE is_current`,
		"assignment_history":            `SELECT count(*) FROM organization.org_assignment_history`,
		"position_role_defaults":        `SELECT count(*) FROM organization.position_role_default WHERE presence = 'PRESENT'`,
		"position_role_defaults_mapped": `SELECT count(*) FROM organization.position_role_default WHERE mapping_state = 'MAPPED'`,
		"hierarchy_findings":            `SELECT count(*) FROM organization.hierarchy_finding`,
	}
	for k, q := range queries {
		var n int64
		if err := tx.QueryRow(ctx, q).Scan(&n); err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		out[k] = n
	}
	rows, err := tx.Query(ctx, `SELECT kind, count(*) FROM organization.hierarchy_finding GROUP BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var n int64
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out["finding."+k] = n
	}
	return out, rows.Err()
}

// SubjectContexts returns, per linked user id, the organisation context the
// authorization diff passes (position slug and whether the employment has
// current reports). Context only: it is never authority.
func SubjectContexts(ctx context.Context, tx db.Tx) (map[string]api.SubjectContext, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT e.user_id::text, pos.slug, lv.slug,
		(SELECT count(*) FROM organization.org_assignment a WHERE a.parent_employment_id = e.employment_id AND a.is_current AND a.presence = 'PRESENT')
		FROM organization.employment e
		LEFT JOIN organization.position pos ON pos.position_id = e.position_id
		LEFT JOIN organization.level lv ON lv.level_id = e.level_id
		WHERE e.user_id IS NOT NULL AND e.presence = 'PRESENT'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]api.SubjectContext{}
	for rows.Next() {
		var (
			uid     string
			sc      api.SubjectContext
			reports int64
		)
		if err := rows.Scan(&uid, &sc.PositionSlug, &sc.LevelSlug, &reports); err != nil {
			return nil, err
		}
		sc.Reports = int(reports)
		sc.Employments = out[uid].Employments + 1
		out[uid] = sc
	}
	return out, rows.Err()
}

func escapeLike(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '%' || c == '_' || c == '\\' {
			b = append(b, '\\')
		}
		b = append(b, s[i])
	}
	return string(b)
}
