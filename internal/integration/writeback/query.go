package writeback

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Read errors.
var (
	ErrInvalid  = errors.New("writeback: invalid query")
	ErrNotFound = errors.New("writeback: not found")
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var states = map[string]bool{StatePending: true, StateWaiting: true, StateApplied: true, StateUnknown: true, StateRejected: true,
	StateReconciliation: true, StateCompared: true}

// Query is the read-only diagnostics service. It never writes, delivers,
// reconciles or retries anything.
type Query struct{ runner *db.Runner }

// NewQuery returns the read service over an integration-owned runner.
func NewQuery(runner *db.Runner) (*Query, error) {
	if runner == nil || runner.Module() != integration.Module {
		return nil, errors.New("writeback: query runner must be owned by integration")
	}
	return &Query{runner: runner}, nil
}

func (q *Query) read(ctx context.Context, fn func(ctx context.Context, tx db.Tx) error) error {
	return q.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, fn)
}

// Count is one grouped count.
type Count struct {
	Key string
	N   int64
}

// Summary is the bounded diagnostics summary.
type Summary struct{ States, Parity, Reconciliations []Count }

func counts(ctx context.Context, tx db.Tx, sql string) ([]Count, error) {
	rows, err := tx.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Count{}
	for rows.Next() {
		var c Count
		if err := rows.Scan(&c.Key, &c.N); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Summary counts states, parity results and reconciliation reasons.
func (q *Query) Summary(ctx context.Context) (Summary, error) {
	var s Summary
	err := q.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		if s.States, err = counts(ctx, tx, `SELECT state, count(*) FROM integration.writeback_command GROUP BY 1 ORDER BY 1`); err != nil {
			return err
		}
		if s.Parity, err = counts(ctx, tx, `SELECT parity, count(*) FROM integration.writeback_command WHERE parity IS NOT NULL GROUP BY 1 ORDER BY 1`); err != nil {
			return err
		}
		s.Reconciliations, err = counts(ctx, tx, `SELECT reason, count(*) FROM integration.writeback_reconciliation GROUP BY 1 ORDER BY 1`)
		return err
	})
	return s, err
}

// Attempt is one recorded attempt.
type Attempt struct {
	No                          int
	Kind, Outcome, ServiceActor string
	Observation, LegacyRef      *string
	Reasons                     []string
	Started, Finished           time.Time
}

// Reconciliation is one reconciliation record.
type Reconciliation struct {
	Reason, NextSafeAction string
	AttemptNo              int
	OpenedAt               time.Time
}

// Command is the read view of one write-back command (no intent payload).
type Command struct {
	CommandID, SourceModule, CommandType, SubjectKind, SubjectID, Mode, State string
	Parity, LegacyRef                                                         *string
	Attempts                                                                  int
	Received, Updated                                                         time.Time
	Log                                                                       []Attempt
	Reconciliations                                                           []Reconciliation
}

const readCols = `command_id::text, source_module, event_type, subject_kind, subject_id::text, rollout_mode, state, parity, legacy_ref,
	attempts, received_at, updated_at`

func scanRead(r pgx.Row) (Command, error) {
	var c Command
	err := r.Scan(&c.CommandID, &c.SourceModule, &c.CommandType, &c.SubjectKind, &c.SubjectID, &c.Mode, &c.State, &c.Parity, &c.LegacyRef,
		&c.Attempts, &c.Received, &c.Updated)
	return c, err
}

// List returns commands (optionally of one state), cursor paged.
func (q *Query) List(ctx context.Context, state *string, cursor *string, limit int) ([]Command, *string, error) {
	if state != nil && !states[*state] {
		return nil, nil, fmt.Errorf("%w: state", ErrInvalid)
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	var after *string
	if cursor != nil {
		b, err := base64.RawURLEncoding.DecodeString(*cursor)
		if err != nil || !uuidPattern.MatchString(string(b)) {
			return nil, nil, fmt.Errorf("%w: cursor", ErrInvalid)
		}
		v := string(b)
		after = &v
	}
	var out []Command
	err := q.read(ctx, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+readCols+` FROM integration.writeback_command WHERE ($1::text IS NULL OR state = $1)
			AND ($2::uuid IS NULL OR command_id > $2) ORDER BY command_id LIMIT $3`, state, after, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanRead(rows)
			if err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, nil, err
	}
	var next *string
	if len(out) > limit {
		out = out[:limit]
		c := base64.RawURLEncoding.EncodeToString([]byte(out[limit-1].CommandID))
		next = &c
	}
	return out, next, nil
}

// Get returns one command with its attempts and reconciliation records.
func (q *Query) Get(ctx context.Context, commandID string) (Command, error) {
	if !uuidPattern.MatchString(commandID) {
		return Command{}, fmt.Errorf("%w: command_id", ErrInvalid)
	}
	var c Command
	err := q.read(ctx, func(ctx context.Context, tx db.Tx) error {
		var err error
		c, err = scanRead(tx.QueryRow(ctx, `SELECT `+readCols+` FROM integration.writeback_command WHERE command_id = $1`, commandID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT attempt_no, kind, outcome, service_actor, observation, legacy_ref, reasons, started_at, finished_at
			FROM integration.writeback_attempt WHERE command_id = $1 ORDER BY attempt_no LIMIT 50`, commandID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var a Attempt
			var reasons []byte
			if err := rows.Scan(&a.No, &a.Kind, &a.Outcome, &a.ServiceActor, &a.Observation, &a.LegacyRef, &reasons, &a.Started, &a.Finished); err != nil {
				rows.Close()
				return err
			}
			_ = json.Unmarshal(reasons, &a.Reasons)
			c.Log = append(c.Log, a)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		rr, err := tx.Query(ctx, `SELECT reason, next_safe_action, attempt_no, opened_at FROM integration.writeback_reconciliation
			WHERE command_id = $1 ORDER BY opened_at, attempt_no LIMIT 50`, commandID)
		if err != nil {
			return err
		}
		defer rr.Close()
		for rr.Next() {
			var r Reconciliation
			if err := rr.Scan(&r.Reason, &r.NextSafeAction, &r.AttemptNo, &r.OpenedAt); err != nil {
				return err
			}
			c.Reconciliations = append(c.Reconciliations, r)
		}
		return rr.Err()
	})
	return c, err
}
