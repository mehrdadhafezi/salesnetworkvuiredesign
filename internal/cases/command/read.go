package command

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

// Current is a case's current contact outcome.
type Current struct {
	Outcome       string
	Reason        *string
	CallbackAt    *time.Time
	LastCommandID string
	RecordedAt    time.Time
}

// Event is one contact outcome history event.
type Event struct {
	EventID, CommandID, Outcome string
	Reason, PreviousOutcome     *string
	CallbackAt                  *time.Time
	BeforeVersion, AfterVersion int64
	ActorUserID, CorrelationID  string
	OccurredAt                  time.Time
}

// View is the contact outcome read of one case.
type View struct {
	CaseID, Workflow string
	Version          int64
	Current          *Current
	History          []Event
}

// ContactOutcomes reads a case's contact outcome facts in a READ ONLY
// transaction. It never writes, enqueues or reconciles anything.
func (s *Service) ContactOutcomes(ctx context.Context, caseID string) (View, error) {
	if !uuidPattern.MatchString(caseID) {
		return View{}, invalid("case_id")
	}
	v := View{CaseID: caseID}
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true, Isolation: db.RepeatableRead}, func(ctx context.Context, tx db.Tx) error {
		err := tx.QueryRow(ctx, `SELECT workflow_type, version FROM cases.work_case WHERE case_id = $1`, caseID).Scan(&v.Workflow, &v.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var c Current
		err = tx.QueryRow(ctx, `SELECT outcome, reason, callback_at, last_command_id::text, recorded_at FROM cases.contact_outcome_current
			WHERE case_id = $1`, caseID).Scan(&c.Outcome, &c.Reason, &c.CallbackAt, &c.LastCommandID, &c.RecordedAt)
		switch {
		case err == nil:
			v.Current = &c
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		rows, err := tx.Query(ctx, `SELECT event_id::text, command_id::text, outcome, reason, callback_at, previous_outcome, before_version,
			after_version, actor_user_id::text, correlation_id, occurred_at FROM cases.contact_outcome_event WHERE case_id = $1
			ORDER BY occurred_at DESC, after_version DESC LIMIT 200`, caseID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e Event
			if err := rows.Scan(&e.EventID, &e.CommandID, &e.Outcome, &e.Reason, &e.CallbackAt, &e.PreviousOutcome, &e.BeforeVersion,
				&e.AfterVersion, &e.ActorUserID, &e.CorrelationID, &e.OccurredAt); err != nil {
				return err
			}
			v.History = append(v.History, e)
		}
		return rows.Err()
	})
	return v, err
}

// Eligible evaluates the command's row scope for actor on caseID without
// writing anything: the same checks the command re-runs at commit. It is
// the source of allowed_actions (capability and mode are checked by the
// caller with the same evaluator the command endpoint uses).
func (s *Service) Eligible(ctx context.Context, caseID string, actor Actor) (bool, reasoncodes.Code, error) {
	if !uuidPattern.MatchString(caseID) {
		return false, reasoncodes.ContractInvalid, nil
	}
	if actor.UserID == "" {
		return false, reasoncodes.RowScopeDenied, nil
	}
	var workflow, presence string
	err := s.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		return tx.QueryRow(ctx, `SELECT workflow_type, presence FROM cases.work_case WHERE case_id = $1`, caseID).Scan(&workflow, &presence)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, reasoncodes.ContractInvalid, ErrNotFound
	}
	if err != nil {
		return false, "", err
	}
	if presence != "PRESENT" {
		return false, reasoncodes.StaleState, nil
	}
	if workflow != string(api.LegacyLead) {
		return false, reasoncodes.PolicyUnresolved, nil
	}
	h, err := s.holders.Holder(ctx, caseID)
	if err != nil {
		return false, "", err
	}
	if !h.Known {
		return false, reasoncodes.DependencyUnknown, nil
	}
	if h.UserID != actor.UserID {
		return false, reasoncodes.RowScopeDenied, nil
	}
	return true, "", nil
}
