// Package command holds the cases module's Slice 11 controlled write:
// RecordContactOutcome (the seller's contact outcome on a legacy lead it
// holds). Every check that decides the write runs inside the owner
// transaction, after the case row is locked (apply-time recheck): rollout
// mode, idempotency, case presence and workflow, the current holder (row
// scope, read through the custody published API), the expected version and
// the value rules. A refused command changes nothing and stores nothing, so
// the same key may be retried once the refusal is resolved.
//
// The command writes only the cases schema: the case version, the current
// contact outcome, an append-only history event, an append-only command
// record, the idempotency record and the cases outbox event, all in one
// transaction. It never changes custody, attribution facets, invoices,
// payments, finance, customer identity or subscriptions, and it never calls
// a connector: integration delivers the outbox event.
package command

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/idempotency"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Scope is the idempotency scope of the command.
const Scope = "cases.contact_outcome"

// CommandType names the command in the command record.
const CommandType = "cases.record_contact_outcome"

// Decision ids named by a refusal.
const (
	DecisionD29   = "D-29"
	DecisionD31   = "D-31"
	DecisionADR01 = "ADR-01"
	// PathUnproven marks a subject whose legacy write path was not proven.
	PathUnproven = "UNPROVEN_LEGACY_PATH"
)

var (
	// ErrInvalid reports a malformed command (400).
	ErrInvalid = errors.New("cases/command: invalid command")
	// ErrNotFound reports an unknown case (404).
	ErrNotFound = errors.New("cases/command: case not found")
)

func invalid(f string) error { return fmt.Errorf("%w: %s", ErrInvalid, f) }

// Holder is the current-holder evidence of a case (custody published read).
type Holder struct {
	// Known is false when custody cannot name exactly one current holder.
	Known  bool
	UserID string
}

// Holders reads the current holder of a case. It is satisfied by an adapter
// over the custody published read API in the composition root.
type Holders interface {
	Holder(ctx context.Context, caseID string) (Holder, error)
}

// Actor is who issues the command.
type Actor struct {
	Principal string
	UserID    string
}

// Command is one contact outcome request.
type Command struct {
	CaseID          string
	Outcome         string
	Reason          *string
	CallbackAt      *time.Time
	ExpectedVersion int64
	IdempotencyKey  string
	CorrelationID   string
	Actor           Actor
}

// Result is the command outcome. It is also what an idempotent replay
// returns.
type Result struct {
	Status         result.Status      `json:"-"`
	StatusName     string             `json:"status"`
	Reasons        []reasoncodes.Code `json:"reasons"`
	BlockedBy      []string           `json:"blocked_by,omitempty"`
	CommandID      string             `json:"command_id,omitempty"`
	Mode           rollout.Mode       `json:"rollout_mode"`
	Applied        bool               `json:"applied"`
	Replayed       bool               `json:"-"`
	CaseVersion    int64              `json:"case_version,omitempty"`
	CurrentVersion int64              `json:"-"`
	WritebackState string             `json:"writeback_state"`
}

// Writeback states the command can report at commit time. The final
// legacy outcome is integration's and is read from its published API.
const (
	WritebackPending      = "LEGACY_WRITEBACK_PENDING"
	WritebackNotRequested = "NOT_REQUESTED"
)

func refuse(status result.Status, mode rollout.Mode, code reasoncodes.Code, blockedBy ...string) Result {
	return Result{Status: status, StatusName: status.String(), Reasons: []reasoncodes.Code{code}, BlockedBy: blockedBy, Mode: mode,
		WritebackState: WritebackNotRequested}
}

// Service runs the command.
type Service struct {
	runner  *db.Runner
	holders Holders
	clock   timeutil.Clock
}

// New returns the service over a cases-owned runner.
func New(runner *db.Runner, holders Holders, clock timeutil.Clock) (*Service, error) {
	if runner == nil || runner.Module() != cases.Module || holders == nil || clock == nil {
		return nil, errors.New("cases/command: runner must be owned by cases, holders and clock are required")
	}
	return &Service{runner: runner, holders: holders, clock: clock}, nil
}

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
)

// validate checks the request shape (400) and the value-level policy
// (BLOCKED for values that need an unresolved decision).
func validate(c Command) (*Result, error) {
	if !uuidPattern.MatchString(c.CaseID) {
		return nil, invalid("case_id")
	}
	if !keyPattern.MatchString(c.IdempotencyKey) {
		return nil, invalid("idempotency_key")
	}
	if c.CorrelationID == "" || len(c.CorrelationID) > 128 {
		return nil, invalid("correlation_id")
	}
	if c.ExpectedVersion < 1 {
		return nil, invalid("expected_version")
	}
	if c.Reason != nil && (len(*c.Reason) == 0 || len(*c.Reason) > 1000) {
		return nil, invalid("reason")
	}
	switch c.Outcome {
	case api.OutcomeDuplicate:
		r := refuse(result.Blocked, "", reasoncodes.PolicyUnresolved, DecisionD29, DecisionD31)
		return &r, nil
	case api.OutcomePreInvoice:
		r := refuse(result.Blocked, "", reasoncodes.PolicyUnresolved, DecisionADR01)
		return &r, nil
	case api.OutcomeNoAnswer, api.OutcomeCallback, api.OutcomeNotPurchased:
	default:
		return nil, invalid("outcome")
	}
	if (c.Outcome == api.OutcomeNotPurchased) != (c.Reason != nil) {
		return nil, invalid("reason (required exactly for NOT_PURCHASED)")
	}
	if (c.Outcome == api.OutcomeCallback) != (c.CallbackAt != nil) {
		return nil, invalid("callback_at (required exactly for CALLBACK)")
	}
	return nil, nil
}

// canonical is the deterministic request encoding hashed for idempotency.
// The actor is part of it: another actor reusing a key is a CONFLICT, never
// a replay of someone else's result.
func canonical(c Command) []byte {
	var cb *string
	if c.CallbackAt != nil {
		s := c.CallbackAt.UTC().Format(time.RFC3339)
		cb = &s
	}
	b, _ := json.Marshal(struct {
		CaseID   string  `json:"case_id"`
		Outcome  string  `json:"outcome"`
		Reason   *string `json:"reason"`
		Callback *string `json:"callback_at"`
		Expected int64   `json:"expected_version"`
		Actor    string  `json:"actor_user_id"`
	}{c.CaseID, c.Outcome, c.Reason, cb, c.ExpectedVersion, c.Actor.UserID})
	return b
}

// errRefused carries a refusal out of the transaction (rolled back).
type errRefused struct{ r Result }

func (e errRefused) Error() string { return "refused: " + e.r.StatusName }

// RecordContactOutcome runs the command in mode.
func (s *Service) RecordContactOutcome(ctx context.Context, mode rollout.Mode, c Command) (Result, error) {
	if pre, err := validate(c); err != nil || pre != nil {
		if pre != nil {
			pre.Mode = mode
			return *pre, nil
		}
		return Result{}, err
	}
	if mode == rollout.Disabled || !mode.Valid() {
		return refuse(result.Blocked, rollout.Disabled, reasoncodes.FeatureDisabled), nil
	}
	if c.Actor.UserID == "" || c.Actor.Principal == "" {
		return refuse(result.Unauthorized, mode, reasoncodes.RowScopeDenied), nil
	}
	sum := sha256.Sum256(canonical(c))
	req := idempotency.Request{Scope: Scope, Key: c.IdempotencyKey, Hash: idempotency.Hash(sum)}
	commandID := ids.Stable(Scope, c.IdempotencyKey)
	var out Result
	err := s.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := s.clock.Now()
		acq, err := idempotency.Acquire(ctx, tx, req, now)
		if err != nil {
			return err
		}
		switch acq.State {
		case idempotency.Replay:
			if err := json.Unmarshal(acq.Result, &out); err != nil {
				return err
			}
			out.Replayed = true
			out.Status, _ = result.ParseStatus(out.StatusName)
			return nil
		case idempotency.Conflict:
			return errRefused{refuse(result.Conflict, mode, reasoncodes.ContractInvalid)}
		case idempotency.InProgress:
			return errRefused{refuse(result.Unknown, mode, reasoncodes.OutcomeUnknown)}
		}
		var (
			version                     int64
			workflow, presence          string
			system, spec, key, srcState string
		)
		err = tx.QueryRow(ctx, `SELECT c.version, c.workflow_type, c.presence, s.source_system, s.source_spec, s.source_key, s.presence
			FROM cases.work_case c JOIN cases.case_source s USING (case_id) WHERE c.case_id = $1 FOR UPDATE OF c`, c.CaseID).
			Scan(&version, &workflow, &presence, &system, &spec, &key, &srcState)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if presence != "PRESENT" || srcState != "PRESENT" {
			return errRefused{refuse(result.Blocked, mode, reasoncodes.StaleState)}
		}
		if workflow != string(api.LegacyLead) {
			return errRefused{refuse(result.Blocked, mode, reasoncodes.PolicyUnresolved, PathUnproven)}
		}
		// Row scope: only the current holder of the lead (legacy owner-only
		// rule). Rank never substitutes; there is no administrator override.
		h, err := s.holders.Holder(ctx, c.CaseID)
		if err != nil {
			return err
		}
		if !h.Known {
			return errRefused{refuse(result.Blocked, mode, reasoncodes.DependencyUnknown)}
		}
		if h.UserID != c.Actor.UserID {
			return errRefused{refuse(result.Unauthorized, mode, reasoncodes.RowScopeDenied)}
		}
		if version != c.ExpectedVersion {
			r := refuse(result.Conflict, mode, reasoncodes.StaleState)
			r.CurrentVersion = version
			return errRefused{r}
		}
		var previous *string
		if err := tx.QueryRow(ctx, `SELECT outcome FROM cases.contact_outcome_current WHERE case_id = $1`, c.CaseID).Scan(&previous); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		at := now.Time().UTC()
		intent, _ := json.Marshal(map[string]any{"outcome": c.Outcome, "reason": c.Reason, "callback_at": c.CallbackAt, "previous_outcome": previous})
		var after *int64
		if mode.Applies() {
			tag, err := tx.Exec(ctx, `UPDATE cases.work_case SET version = version + 1 WHERE case_id = $1 AND version = $2`, c.CaseID, version)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return errRefused{refuse(result.Conflict, mode, reasoncodes.StaleState)}
			}
			v := version + 1
			after = &v
			if _, err := tx.Exec(ctx, `INSERT INTO cases.contact_outcome_current (case_id, outcome, reason, callback_at, last_command_id, recorded_at)
				VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (case_id) DO UPDATE SET outcome = EXCLUDED.outcome, reason = EXCLUDED.reason,
				callback_at = EXCLUDED.callback_at, last_command_id = EXCLUDED.last_command_id, recorded_at = EXCLUDED.recorded_at`,
				c.CaseID, c.Outcome, c.Reason, c.CallbackAt, commandID, at); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO cases.contact_outcome_event (event_id, case_id, command_id, outcome, reason, callback_at,
				previous_outcome, before_version, after_version, actor_principal, actor_user_id, correlation_id, occurred_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
				ids.Stable("cases.contact_outcome_event", commandID), c.CaseID, commandID, c.Outcome, c.Reason, c.CallbackAt, previous,
				version, v, c.Actor.Principal, c.Actor.UserID, c.CorrelationID, at); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO cases.command_record (command_id, command_type, case_id, idempotency_key, request_sha256,
			rollout_mode, applied, intent, before_version, after_version, actor_principal, actor_user_id, correlation_id, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			commandID, CommandType, c.CaseID, c.IdempotencyKey, sum[:], string(mode), mode.Applies(), intent, version, after,
			c.Actor.Principal, c.Actor.UserID, c.CorrelationID, at); err != nil {
			return err
		}
		payload, err := json.Marshal(api.ContactOutcomeRecorded{CommandID: commandID, IdempotencyKey: c.IdempotencyKey, CorrelationID: c.CorrelationID,
			CaseID: c.CaseID, SourceSystem: system, SourceSpec: spec, SourceKey: key, Outcome: c.Outcome, Reason: c.Reason, CallbackAt: c.CallbackAt,
			PreviousOutcome: previous, ActorUserID: c.Actor.UserID, RolloutMode: string(mode), Applied: mode.Applies(), BeforeVersion: version,
			AfterVersion: after, OccurredAt: at})
		if err != nil {
			return err
		}
		corr := c.CorrelationID
		if err := outbox.Append(ctx, tx, outbox.Event{ID: ids.Stable("cases.outbox", commandID), Type: api.EventContactOutcomeRecorded, Version: 1,
			Payload: payload, OccurredAt: now, CorrelationID: &corr}, now); err != nil {
			return err
		}
		out = Result{Status: result.Success, StatusName: result.Success.String(), Reasons: []reasoncodes.Code{}, CommandID: commandID, Mode: mode,
			Applied: mode.Applies(), CaseVersion: version, WritebackState: WritebackNotRequested}
		if after != nil {
			out.CaseVersion, out.WritebackState = *after, WritebackPending
		}
		stored, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return idempotency.Complete(ctx, tx, req, stored, now)
	})
	var refused errRefused
	if errors.As(err, &refused) {
		return refused.r, nil
	}
	if err != nil {
		return Result{}, err
	}
	return out, nil
}
