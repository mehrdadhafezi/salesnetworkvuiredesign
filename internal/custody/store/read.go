package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Assignment is the read view of one derived projection.
type Assignment struct {
	CaseID, Derivation                    string
	DerivedKey, DerivedUserID, DerivedPos *string
	LegacyKey, LegacyUserID, LegacyPos    *string
	LegacySpec, Parity                    string
	EventCount                            int
	LastEventID                           *string
	Presence                              string
	RefreshedAt                           time.Time
	FindingCount                          int64
}

const assignmentSelect = `SELECT a.case_id::text, a.derivation, a.derived_legacy_key, a.derived_user_id::text, a.derived_position_raw,
	a.legacy_holder_key, a.legacy_holder_user_id::text, a.legacy_position_raw, a.legacy_source_spec, a.parity, a.event_count,
	a.last_event_id::text, a.presence, a.refreshed_at,
	(SELECT count(*) FROM custody.finding f WHERE f.case_id = a.case_id AND f.presence = 'PRESENT')
	FROM custody.assignment a`

func scanAssignment(r pgx.Row) (Assignment, error) {
	var a Assignment
	err := r.Scan(&a.CaseID, &a.Derivation, &a.DerivedKey, &a.DerivedUserID, &a.DerivedPos, &a.LegacyKey, &a.LegacyUserID, &a.LegacyPos,
		&a.LegacySpec, &a.Parity, &a.EventCount, &a.LastEventID, &a.Presence, &a.RefreshedAt, &a.FindingCount)
	return a, err
}

// AssignmentFilter is the allowlisted filter.
type AssignmentFilter struct{ Parity, Derivation, Presence *string }

// ListAssignments lists projections by case id (keyset).
func ListAssignments(ctx context.Context, tx db.Tx, f AssignmentFilter, after *string, limit int) ([]Assignment, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, assignmentSelect+`
		WHERE ($1::text IS NULL OR a.parity = $1) AND ($2::text IS NULL OR a.derivation = $2) AND ($3::text IS NULL OR a.presence = $3)
		  AND ($4::uuid IS NULL OR a.case_id > $4)
		ORDER BY a.case_id LIMIT $5`, f.Parity, f.Derivation, f.Presence, after, limit+1)
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

// AssignmentOf returns the projection of one case.
func AssignmentOf(ctx context.Context, tx db.Tx, caseID string) (Assignment, error) {
	if err := ReadTx(tx); err != nil {
		return Assignment{}, err
	}
	a, err := scanAssignment(tx.QueryRow(ctx, assignmentSelect+` WHERE a.case_id = $1`, caseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	return a, err
}

// Event is the read view of one custody event.
type Event struct {
	EventID, SourceSpec, SourceKey, Ordinal string
	Action, Kind, Context                   string
	ActorKey, ActorUserID                   *string
	FromKey, FromUserID, ToKey, ToUserID    *string
	FromPos, ToPos, StepOrder, OccurredRaw  *string
	RecordedAt                              time.Time
}

// EventView lists a case's events in legacy insertion order.
func EventView(ctx context.Context, tx db.Tx, caseID string) ([]Event, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT event_id::text, source_spec, source_key, source_ordinal::text, legacy_action, kind, context,
		actor_legacy_key, actor_user_id::text, from_legacy_key, from_user_id::text, to_legacy_key, to_user_id::text,
		from_position_raw, to_position_raw, step_order_raw, occurred_raw, recorded_at
		FROM custody.custody_event WHERE case_id = $1 ORDER BY source_ordinal`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.EventID, &e.SourceSpec, &e.SourceKey, &e.Ordinal, &e.Action, &e.Kind, &e.Context, &e.ActorKey, &e.ActorUserID,
			&e.FromKey, &e.FromUserID, &e.ToKey, &e.ToUserID, &e.FromPos, &e.ToPos, &e.StepOrder, &e.OccurredRaw, &e.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Finding is the read view of one finding.
type Finding struct {
	FindingID, Kind, Detail string
	EventID                 *string
	Presence                string
	RecordedAt              time.Time
}

// FindingsOf lists a case's findings.
func FindingsOf(ctx context.Context, tx db.Tx, caseID string) ([]Finding, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT finding_id::text, kind, detail, event_id::text, presence, recorded_at FROM custody.finding
		WHERE case_id = $1 ORDER BY recorded_at, finding_id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Finding
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.FindingID, &f.Kind, &f.Detail, &f.EventID, &f.Presence, &f.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ParityReport computes the custody parity report.
func ParityReport(ctx context.Context, tx db.Tx) (api.Parity, error) {
	if err := ReadTx(tx); err != nil {
		return api.Parity{}, err
	}
	r := api.Parity{ByParity: map[string]int64{}, ByDerivation: map[string]int64{}, FindingsByKind: map[string]int64{},
		EventsByKind: map[string]int64{}, UnknownActions: map[string]int64{}}
	for q, dst := range map[string]*int64{
		`SELECT count(*) FROM custody.assignment WHERE presence = 'PRESENT'`: &r.Assignments,
		`SELECT count(*) FROM custody.custody_event`:                         &r.Events,
		`SELECT count(DISTINCT case_id) FROM custody.custody_event`:          &r.CasesWithEvents,
	} {
		if err := tx.QueryRow(ctx, q).Scan(dst); err != nil {
			return r, err
		}
	}
	for q, into := range map[string]map[string]int64{
		`SELECT parity, count(*) FROM custody.assignment WHERE presence = 'PRESENT' GROUP BY 1`:       r.ByParity,
		`SELECT derivation, count(*) FROM custody.assignment WHERE presence = 'PRESENT' GROUP BY 1`:   r.ByDerivation,
		`SELECT kind, count(*) FROM custody.finding WHERE presence = 'PRESENT' GROUP BY 1`:            r.FindingsByKind,
		`SELECT kind, count(*) FROM custody.custody_event GROUP BY 1`:                                 r.EventsByKind,
		`SELECT legacy_action, count(*) FROM custody.custody_event WHERE kind = 'UNKNOWN' GROUP BY 1`: r.UnknownActions,
	} {
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return r, err
		}
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				rows.Close()
				return r, err
			}
			into[k] = n
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return r, err
		}
	}
	// Full parity only when every present projection MATCHes; never claimed
	// on an empty or partial basis.
	r.FullParity = r.Assignments > 0 && r.ByParity["MATCH"] == r.Assignments
	return r, nil
}

// CaseCoverage maps case ids to the parity of their PRESENT projection.
func CaseCoverage(ctx context.Context, tx db.Tx, caseIDs []string) (map[string]string, error) {
	if err := ReadTx(tx); err != nil {
		return nil, err
	}
	if caseIDs == nil {
		caseIDs = []string{}
	}
	rows, err := tx.Query(ctx, `SELECT case_id::text, parity FROM custody.assignment WHERE presence = 'PRESENT' AND case_id = ANY($1::uuid[])`, caseIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, p string
		if err := rows.Scan(&id, &p); err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, rows.Err()
}

// BatchImported reports whether a custody import run exists for a batch.
func BatchImported(ctx context.Context, tx db.Tx, batchID string) (bool, error) {
	if err := ReadTx(tx); err != nil {
		return false, err
	}
	var n int64
	err := tx.QueryRow(ctx, `SELECT count(*) FROM custody.import_run WHERE batch_id = $1`, batchID).Scan(&n)
	return n > 0, err
}
