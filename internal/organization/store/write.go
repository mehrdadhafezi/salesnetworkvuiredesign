// Package store holds the organization module's SQL. It is private to
// organization (architecture lint: foreign-store). Write functions require a
// writable organization-owned transaction (import path only); read
// functions require a read-only one.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrWrongOwner reports a transaction not owned by organization.
	ErrWrongOwner = errors.New("organization/store: transaction is not owned by organization")
	// ErrNotReadOnly reports a read path given a writable transaction.
	ErrNotReadOnly = errors.New("organization/store: read path requires a read-only transaction")
	// ErrReadOnly reports a write path given a read-only transaction.
	ErrReadOnly = errors.New("organization/store: write path requires a writable transaction")
)

// Change classifies what an upsert did.
type Change string

const (
	Inserted   Change = "inserted"
	Refreshed  Change = "refreshed"
	Reappeared Change = "reappeared"
	Rebatched  Change = "rebatched"
	Unchanged  Change = "unchanged"
)

func writable(tx db.Tx) error {
	if tx.Module() != organization.Module {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ReadTx checks a read path transaction.
func ReadTx(tx db.Tx) error {
	if tx.Module() != organization.Module {
		return ErrWrongOwner
	}
	if !tx.ReadOnly() {
		return ErrNotReadOnly
	}
	return nil
}

// LockImport serialises organization import commands.
func LockImport(ctx context.Context, tx db.Tx) error {
	if err := writable(tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('crm.organization.import'))`)
	return err
}

// table describes one shadow table with source provenance. Every name here
// is a constant of this package; no input reaches an identifier.
type table struct {
	name  string // schema-qualified
	id    string // primary key column
	kind  bool   // legacy keys are unique per kind (org_unit)
	attrs []string
}

var (
	positionTable = table{name: "organization.position", id: "position_id",
		attrs: []string{"slug", "label", "description", "sort_order", "is_active", "is_system", "source_created_raw", "source_updated_raw"}}
	levelTable = table{name: "organization.level", id: "level_id",
		attrs: []string{"slug", "label", "description", "sort_order", "is_active", "is_system", "source_created_raw", "source_updated_raw"}}
	orgUnitTable = table{name: "organization.org_unit", id: "org_unit_id", kind: true,
		attrs: []string{"parent_org_unit_id", "parent_source_key", "parent_state", "slug", "label", "description", "sort_order", "is_active", "source_created_raw", "source_updated_raw"}}
	personTable     = table{name: "organization.person", id: "person_id", attrs: []string{"display_name"}}
	employmentTable = table{name: "organization.employment", id: "employment_id",
		attrs: []string{"person_id", "user_id", "user_source_key", "user_link_state", "employee_code", "employment_status",
			"employment_status_raw", "employment_type_raw", "hire_date", "hire_date_raw", "termination_date", "termination_date_raw",
			"is_active", "position_id", "position_source_key", "position_ref_state", "level_id", "level_source_key", "level_ref_state",
			"department_id", "department_source_key", "department_ref_state", "unit_id", "unit_source_key", "unit_ref_state",
			"team_id", "team_source_key", "team_ref_state", "source_created_raw", "source_updated_raw"}}
	assignmentTable = table{name: "organization.org_assignment", id: "assignment_id",
		attrs: []string{"child_employment_id", "child_source_key", "child_ref_state", "parent_employment_id", "parent_source_key",
			"parent_ref_state", "relationship_kind", "relationship_raw", "effective_from_raw", "effective_to_raw", "is_current",
			"current_raw", "origin_raw", "recorded_by_source_key", "source_created_raw", "source_updated_raw"}}
	defaultTable = table{name: "organization.position_role_default", id: "default_id",
		attrs: []string{"position_id", "position_source_key", "role_key", "legacy_role_value", "mapping_state",
			"default_level_id", "default_level_source_key", "is_active"}}
)

func placeholders(from, n int) string {
	p := make([]string, n)
	for i := range p {
		p[i] = "$" + strconv.Itoa(from+i)
	}
	return strings.Join(p, ", ")
}

// upsert inserts or refreshes one row keyed by (system[, kind], key). A row
// whose checksum, presence and resolved attributes are unchanged is left
// alone (or only re-batched), so replaying a batch writes nothing.
func upsert(ctx context.Context, tx db.Tx, t table, kind string, src api.SourceRef, vals []any, now time.Time) (string, Change, error) {
	if err := writable(tx); err != nil {
		return "", "", err
	}
	if len(vals) != len(t.attrs) {
		return "", "", fmt.Errorf("organization/store: %s: %d values for %d columns", t.name, len(vals), len(t.attrs))
	}
	where := "source_system = $1 AND source_key = $2"
	keyArgs := []any{src.System, src.Key}
	if t.kind {
		where += " AND kind = $3"
		keyArgs = append(keyArgs, kind)
	}
	var id, batch, presence string
	err := tx.QueryRow(ctx, `SELECT `+t.id+`::text, source_batch_id::text, presence FROM `+t.name+` WHERE `+where+` FOR UPDATE`, keyArgs...).
		Scan(&id, &batch, &presence)
	if errors.Is(err, pgx.ErrNoRows) {
		if id, err = ids.NewUUID(); err != nil {
			return "", "", err
		}
		cols := []string{t.id, "source_system", "source_spec", "source_key", "first_batch_id", "source_batch_id",
			"source_row_sha256", "imported_at", "refreshed_at", "presence", "version"}
		args := []any{id, src.System, src.Spec, src.Key, src.BatchID, src.BatchID, src.RowSHA256, now, now, "PRESENT", int64(1)}
		if t.kind {
			cols = append(cols, "kind")
			args = append(args, kind)
		}
		cols = append(cols, t.attrs...)
		args = append(args, vals...)
		_, err = tx.Exec(ctx, `INSERT INTO `+t.name+` (`+strings.Join(cols, ", ")+`) VALUES (`+placeholders(1, len(args))+`)`, args...)
		return id, Inserted, err
	}
	if err != nil {
		return "", "", err
	}
	// Refresh only when content (checksum or resolved attributes) differs.
	set := []string{"source_spec = $2", "source_batch_id = $3", "source_row_sha256 = $4", "refreshed_at = $5", "presence = 'PRESENT'", "version = version + 1"}
	args := []any{id, src.Spec, src.BatchID, src.RowSHA256, now}
	cmpCols := []string{"source_row_sha256"}
	cmpVals := []string{"$4"}
	for i, c := range t.attrs {
		n := len(args) + 1
		set = append(set, fmt.Sprintf("%s = $%d", c, n))
		cmpCols = append(cmpCols, c)
		cmpVals = append(cmpVals, fmt.Sprintf("$%d", n))
		args = append(args, vals[i])
	}
	q := `UPDATE ` + t.name + ` SET ` + strings.Join(set, ", ") + ` WHERE ` + t.id + ` = $1 AND (` +
		strings.Join(cmpCols, ", ") + `) IS DISTINCT FROM (` + strings.Join(cmpVals, ", ") + `)`
	tag, err := tx.Exec(ctx, q, args...)
	if err != nil {
		return "", "", err
	}
	if tag.RowsAffected() > 0 {
		if presence != "PRESENT" {
			return id, Reappeared, nil
		}
		return id, Refreshed, nil
	}
	switch {
	case presence != "PRESENT":
		_, err = tx.Exec(ctx, `UPDATE `+t.name+` SET presence = 'PRESENT', source_batch_id = $2, refreshed_at = $3, version = version + 1 WHERE `+t.id+` = $1`, id, src.BatchID, now)
		return id, Reappeared, err
	case batch != src.BatchID:
		_, err = tx.Exec(ctx, `UPDATE `+t.name+` SET source_batch_id = $2, refreshed_at = $3 WHERE `+t.id+` = $1`, id, src.BatchID, now)
		return id, Rebatched, err
	}
	return id, Unchanged, nil
}

func markAbsent(ctx context.Context, tx db.Tx, t table, kind string, system string, present []string, now time.Time) (int64, error) {
	if err := writable(tx); err != nil {
		return 0, err
	}
	q := `UPDATE ` + t.name + ` SET presence = 'ABSENT_IN_LATEST', refreshed_at = $3, version = version + 1
		WHERE source_system = $1 AND presence = 'PRESENT' AND NOT (source_key = ANY($2))`
	args := []any{system, present, now}
	if t.kind {
		q += ` AND kind = $4`
		args = append(args, kind)
	}
	tag, err := tx.Exec(ctx, q, args...)
	return tag.RowsAffected(), err
}

func keyMap(ctx context.Context, tx db.Tx, t table, kind string) (map[string]string, error) {
	q := `SELECT source_key, ` + t.id + `::text FROM ` + t.name
	var args []any
	if t.kind {
		q += ` WHERE kind = $1`
		args = append(args, kind)
	}
	rows, err := tx.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, id string
		if err := rows.Scan(&k, &id); err != nil {
			return nil, err
		}
		out[k] = id
	}
	return out, rows.Err()
}

// UpsertPosition / UpsertLevel store one lookup row.
func UpsertPosition(ctx context.Context, tx db.Tx, r api.LookupRecord, now time.Time) (string, Change, error) {
	return upsert(ctx, tx, positionTable, "", r.Source, lookupVals(r), now)
}

// UpsertLevel stores one level.
func UpsertLevel(ctx context.Context, tx db.Tx, r api.LookupRecord, now time.Time) (string, Change, error) {
	return upsert(ctx, tx, levelTable, "", r.Source, lookupVals(r), now)
}

func lookupVals(r api.LookupRecord) []any {
	return []any{r.Slug, r.Label, r.Description, r.SortOrder, r.IsActive, r.IsSystem, r.CreatedRaw, r.UpdatedRaw}
}

// Ref is a resolved reference ready to store.
type Ref struct {
	ID    *string
	Key   *string
	State string
}

// Resolve looks key up in m: NOT_RECORDED when key is nil, MISSING when
// absent, RESOLVED otherwise. Nothing is invented for a missing target.
func Resolve(key *string, m map[string]string) Ref {
	if key == nil {
		return Ref{State: domain.RefNotRecorded}
	}
	if id, ok := m[*key]; ok {
		return Ref{ID: &id, Key: key, State: domain.RefResolved}
	}
	return Ref{Key: key, State: domain.RefMissing}
}

// UpsertOrgUnit stores one unit with its resolved parent.
func UpsertOrgUnit(ctx context.Context, tx db.Tx, r api.OrgUnitRecord, parent Ref, now time.Time) (string, Change, error) {
	state := parent.State
	if r.Kind == api.Department {
		state = "NOT_APPLICABLE"
	}
	return upsert(ctx, tx, orgUnitTable, string(r.Kind), r.Source,
		[]any{parent.ID, parent.Key, state, r.Slug, r.Label, r.Description, r.SortOrder, r.IsActive, r.CreatedRaw, r.UpdatedRaw}, now)
}

// UpsertPerson stores the Person half of a profile.
func UpsertPerson(ctx context.Context, tx db.Tx, r api.ProfileRecord, now time.Time) (string, Change, error) {
	return upsert(ctx, tx, personTable, "", r.Source, []any{r.DisplayName}, now)
}

// EmploymentRefs are the resolved references of one employment.
type EmploymentRefs struct {
	UserID                                  *string
	UserLink                                string
	Position, Level, Department, Unit, Team Ref
}

// UpsertEmployment stores the Employment half of a profile.
func UpsertEmployment(ctx context.Context, tx db.Tx, r api.ProfileRecord, personID string, refs EmploymentRefs, now time.Time) (string, Change, error) {
	hire, hireOK := domain.LegacyDate(r.HireDateRaw)
	term, termOK := domain.LegacyDate(r.TerminationDateRaw)
	var hireP, termP any
	if hireOK {
		hireP = hire
	}
	if termOK {
		termP = term
	}
	return upsert(ctx, tx, employmentTable, "", r.Source, []any{
		personID, refs.UserID, r.UserKey, refs.UserLink, r.EmployeeCode, string(r.Status), r.StatusRaw, r.TypeRaw,
		hireP, r.HireDateRaw, termP, r.TerminationDateRaw, r.IsActive,
		refs.Position.ID, refs.Position.Key, refs.Position.State, refs.Level.ID, refs.Level.Key, refs.Level.State,
		refs.Department.ID, refs.Department.Key, refs.Department.State, refs.Unit.ID, refs.Unit.Key, refs.Unit.State,
		refs.Team.ID, refs.Team.Key, refs.Team.State, r.CreatedRaw, r.UpdatedRaw,
	}, now)
}

// UpsertAssignment stores one reporting assignment.
func UpsertAssignment(ctx context.Context, tx db.Tx, r api.AssignmentRecord, child, parent Ref, now time.Time) (string, Change, error) {
	childState := child.State
	if childState == domain.RefNotRecorded {
		childState = domain.RefMissing
	}
	return upsert(ctx, tx, assignmentTable, "", r.Source, []any{
		child.ID, r.ChildKey, childState, parent.ID, parent.Key, parent.State, string(r.Kind), r.KindRaw,
		r.EffectiveFromRaw, r.EffectiveToRaw, r.IsCurrent, r.CurrentRaw, r.OriginRaw, r.RecordedByKey, r.CreatedRaw, r.UpdatedRaw,
	}, now)
}

// UpsertDefault stores one position role default (evidence, never a grant).
func UpsertDefault(ctx context.Context, tx db.Tx, r api.PositionRoleDefaultRecord, pos, lvl Ref, now time.Time) (string, Change, error) {
	mapping := "UNMAPPED"
	var role *string
	if r.RoleKey != nil {
		s := string(*r.RoleKey)
		role, mapping = &s, "MAPPED"
	}
	return upsert(ctx, tx, defaultTable, "", r.Source, []any{pos.ID, pos.Key, role, r.LegacyRole, mapping, lvl.ID, lvl.Key, r.IsActive}, now)
}

// InsertHistory appends one history row. An existing key is never rewritten:
// identical content is "unchanged"; different content is reported as
// HISTORY_CHANGED by the caller and the stored evidence is kept.
func InsertHistory(ctx context.Context, tx db.Tx, r api.HistoryRecord, child, oldP, newP Ref, now time.Time) (string, Change, bool, error) {
	if err := writable(tx); err != nil {
		return "", "", false, err
	}
	id, err := ids.NewUUID()
	if err != nil {
		return "", "", false, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO organization.org_assignment_history (history_id, source_system, source_spec, source_key,
		source_batch_id, source_row_sha256, imported_at, assignment_source_key, child_source_key, child_employment_id,
		old_parent_source_key, old_parent_employment_id, new_parent_source_key, new_parent_employment_id,
		actor_user_source_key, action_raw, recorded_raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (source_system, source_key) DO NOTHING`,
		id, r.Source.System, r.Source.Spec, r.Source.Key, r.Source.BatchID, r.Source.RowSHA256, now,
		r.AssignmentKey, r.ChildKey, child.ID, r.OldParentKey, oldP.ID, r.NewParentKey, newP.ID, r.ActorUserKey, r.ActionRaw, r.RecordedRaw)
	if err != nil {
		return "", "", false, err
	}
	if tag.RowsAffected() == 1 {
		return id, Inserted, false, nil
	}
	var same bool
	if err := tx.QueryRow(ctx, `SELECT history_id::text, source_row_sha256 = $3 FROM organization.org_assignment_history
		WHERE source_system = $1 AND source_key = $2`, r.Source.System, r.Source.Key, r.Source.RowSHA256).Scan(&id, &same); err != nil {
		return "", "", false, err
	}
	return id, Unchanged, !same, nil
}

// Key maps (legacy key → native id) after upserts.
func PositionKeys(ctx context.Context, tx db.Tx) (map[string]string, error) {
	return keyMap(ctx, tx, positionTable, "")
}

// LevelKeys maps level keys.
func LevelKeys(ctx context.Context, tx db.Tx) (map[string]string, error) {
	return keyMap(ctx, tx, levelTable, "")
}

// OrgUnitKeys maps unit keys of one kind.
func OrgUnitKeys(ctx context.Context, tx db.Tx, kind api.OrgUnitKind) (map[string]string, error) {
	return keyMap(ctx, tx, orgUnitTable, string(kind))
}

// EmploymentKeys maps profile keys to employment ids.
func EmploymentKeys(ctx context.Context, tx db.Tx) (map[string]string, error) {
	return keyMap(ctx, tx, employmentTable, "")
}

// MarkAbsent flags rows of an entity missing from a COMPLETE batch.
func MarkAbsent(ctx context.Context, tx db.Tx, e api.Entity, system string, present []string, now time.Time) (int64, error) {
	switch e {
	case api.EntityPositions:
		return markAbsent(ctx, tx, positionTable, "", system, present, now)
	case api.EntityLevels:
		return markAbsent(ctx, tx, levelTable, "", system, present, now)
	case api.EntityDepartments:
		return markAbsent(ctx, tx, orgUnitTable, string(api.Department), system, present, now)
	case api.EntityUnits:
		return markAbsent(ctx, tx, orgUnitTable, string(api.Unit), system, present, now)
	case api.EntityTeams:
		return markAbsent(ctx, tx, orgUnitTable, string(api.Team), system, present, now)
	case api.EntityProfiles:
		n1, err := markAbsent(ctx, tx, personTable, "", system, present, now)
		if err != nil {
			return 0, err
		}
		n2, err := markAbsent(ctx, tx, employmentTable, "", system, present, now)
		return n1 + n2, err
	case api.EntityAssignments:
		return markAbsent(ctx, tx, assignmentTable, "", system, present, now)
	case api.EntityDefaults:
		return markAbsent(ctx, tx, defaultTable, "", system, present, now)
	}
	return 0, nil // history is append-only evidence: never marked
}

// AnalysisInput loads the stored model (all presence states) for analysis.
func AnalysisInput(ctx context.Context, tx db.Tx) (domain.Input, error) {
	var in domain.Input
	rows, err := tx.Query(ctx, `SELECT org_unit_id::text, kind, source_key, parent_org_unit_id::text, parent_source_key, parent_state
		FROM organization.org_unit WHERE presence = 'PRESENT'`)
	if err != nil {
		return in, err
	}
	for rows.Next() {
		var u domain.UnitNode
		if err := rows.Scan(&u.ID, &u.Kind, &u.Key, &u.ParentID, &u.ParentKey, &u.ParentState); err != nil {
			rows.Close()
			return in, err
		}
		in.Units = append(in.Units, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}
	rows, err = tx.Query(ctx, `SELECT employment_id::text, source_key, user_link_state, user_source_key,
		position_id::text, position_source_key, position_ref_state, level_id::text, level_source_key, level_ref_state,
		department_id::text, department_source_key, department_ref_state, unit_id::text, unit_source_key, unit_ref_state,
		team_id::text, team_source_key, team_ref_state FROM organization.employment WHERE presence = 'PRESENT'`)
	if err != nil {
		return in, err
	}
	for rows.Next() {
		var e domain.EmploymentNode
		if err := rows.Scan(&e.ID, &e.Key, &e.UserLink, &e.UserKey,
			&e.Position.ID, &e.Position.Key, &e.Position.State, &e.Level.ID, &e.Level.Key, &e.Level.State,
			&e.Department.ID, &e.Department.Key, &e.Department.State, &e.Unit.ID, &e.Unit.Key, &e.Unit.State,
			&e.Team.ID, &e.Team.Key, &e.Team.State); err != nil {
			rows.Close()
			return in, err
		}
		in.Employments = append(in.Employments, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}
	rows, err = tx.Query(ctx, `SELECT assignment_id::text, source_key, child_employment_id::text, child_source_key,
		parent_employment_id::text, parent_source_key, parent_ref_state, relationship_kind, effective_from_raw,
		effective_to_raw, is_current, current_raw FROM organization.org_assignment WHERE presence = 'PRESENT'`)
	if err != nil {
		return in, err
	}
	for rows.Next() {
		var a domain.AssignmentNode
		if err := rows.Scan(&a.ID, &a.Key, &a.ChildID, &a.ChildKey, &a.ParentID, &a.ParentKey, &a.ParentState, &a.Kind,
			&a.EffectiveFromRaw, &a.EffectiveToRaw, &a.IsCurrent, &a.CurrentRaw); err != nil {
			rows.Close()
			return in, err
		}
		in.Assignments = append(in.Assignments, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return in, err
	}
	rows, err = tx.Query(ctx, `SELECT history_id::text, source_key, child_source_key, old_parent_source_key,
		new_parent_source_key, recorded_raw FROM organization.org_assignment_history`)
	if err != nil {
		return in, err
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.HistoryNode
		if err := rows.Scan(&h.ID, &h.Key, &h.ChildKey, &h.OldParentKey, &h.NewParentKey, &h.RecordedRaw); err != nil {
			return in, err
		}
		in.History = append(in.History, h)
	}
	return in, rows.Err()
}

// ReplaceFindings replaces the derived findings with those of runID.
func ReplaceFindings(ctx context.Context, tx db.Tx, runID string, fs []domain.Finding) error {
	if err := writable(tx); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM organization.hierarchy_finding`); err != nil {
		return err
	}
	for _, f := range fs {
		id, err := ids.NewUUID()
		if err != nil {
			return err
		}
		detail := f.Detail
		if len(detail) > 300 {
			detail = detail[:300]
		}
		if _, err := tx.Exec(ctx, `INSERT INTO organization.hierarchy_finding (finding_id, run_id, kind, severity,
			subject_type, subject_id, subject_source_key, related_id, related_source_key, detail)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			id, runID, f.Kind, string(f.Severity), f.SubjectType, f.SubjectID, f.SubjectKey, f.RelatedID, f.RelatedKey, detail); err != nil {
			return err
		}
	}
	return nil
}

// PriorRun reports whether batchID was imported before.
func PriorRun(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM organization.import_run WHERE batch_id = $1`, batchID).Scan(&n)
	return n > 0, err
}

// InsertRun records one import run.
func InsertRun(ctx context.Context, tx db.Tx, runID, batchID, snapshotID, outcome string, replay bool, counts map[string]int64, now time.Time) error {
	if err := writable(tx); err != nil {
		return err
	}
	b, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO organization.import_run (run_id, batch_id, snapshot_id, outcome, replay, counts, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, runID, batchID, snapshotID, outcome, replay, b, now)
	return err
}
