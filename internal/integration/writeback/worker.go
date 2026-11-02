package writeback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	orgapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/inbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/outbox"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Consumer is the inbox consumer name of the write-back.
const Consumer = "integration.writeback"

// ServiceActor is the technical actor recorded on every attempt. A
// write-back is never presented as the human target actor's own action.
const ServiceActor = "service:integration.writeback"

// States of a write-back command.
const (
	StatePending        = "PENDING"
	StateWaiting        = "WAITING_DEPENDENCY"
	StateApplied        = "APPLIED"
	StateUnknown        = "UNKNOWN"
	StateRejected       = "REJECTED"
	StateReconciliation = "RECONCILIATION_REQUIRED"
	StateCompared       = "COMPARED"
)

// Rollout modes as recorded by the source modules.
const (
	modeShadow     = "SHADOW_COMPARE"
	modeControlled = "CONTROLLED_WRITEBACK"
)

// ErrUnknownEvent reports an outbox event this consumer does not deliver.
var ErrUnknownEvent = errors.New("writeback: unknown event type")

// Worker consumes source outboxes and delivers write-back commands.
type Worker struct {
	runner    *db.Runner
	clock     timeutil.Clock
	connector Connector
}

// New returns a worker over an integration-owned runner.
func New(runner *db.Runner, clock timeutil.Clock, connector Connector) (*Worker, error) {
	if runner == nil || runner.Module() != integration.Module || clock == nil || connector == nil {
		return nil, errors.New("writeback: runner must be owned by integration; clock and connector required")
	}
	return &Worker{runner: runner, clock: clock, connector: connector}, nil
}

type received struct {
	commandID, key, corr, mode, subjectKind, subjectID, eventType string
	fields                                                        map[string]string
	dependsOnSubject                                              string
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func put(m map[string]string, k string, p *string) {
	if p != nil {
		m[k] = *p
	}
}

// decode maps a published source event to a neutral write-back command.
func decode(source owner.Module, typ string, payload []byte) (received, error) {
	switch {
	case source == "cases" && typ == casesapi.EventContactOutcomeRecorded:
		var p casesapi.ContactOutcomeRecorded
		if err := json.Unmarshal(payload, &p); err != nil {
			return received{}, err
		}
		f := map[string]string{"outcome": p.Outcome, "case_id": p.CaseID, "source_system": p.SourceSystem, "source_spec": p.SourceSpec,
			"source_key": p.SourceKey, "actor_user_id": p.ActorUserID, "before_version": strconv.FormatInt(p.BeforeVersion, 10)}
		put(f, "reason", p.Reason)
		if p.CallbackAt != nil {
			f["callback_at"] = p.CallbackAt.UTC().Format(time.RFC3339)
		}
		return received{commandID: p.CommandID, key: p.IdempotencyKey, corr: p.CorrelationID, mode: p.RolloutMode, subjectKind: "case",
			subjectID: p.CaseID, eventType: TypeContactOutcome, fields: f}, nil
	case source == "organization" && typ == orgapi.EventHRRequestCreated:
		var p orgapi.HRRequestCreated
		if err := json.Unmarshal(payload, &p); err != nil {
			return received{}, err
		}
		f := map[string]string{"request_id": p.RequestID, "request_type": p.RequestType, "subject_user_id": p.SubjectUserID,
			"requested_by_user_id": p.RequestedByUserID, "status": p.Status, "reason": p.Reason}
		put(f, "subject_parent_user_id", p.SubjectParentUserID)
		put(f, "target_parent_user_id", p.TargetParentUserID)
		put(f, "current_reviewer_user_id", p.CurrentReviewerUser)
		return received{commandID: p.CommandID, key: p.IdempotencyKey, corr: p.CorrelationID, mode: p.RolloutMode, subjectKind: "hr_request",
			subjectID: p.RequestID, eventType: TypeHRCreate, fields: f}, nil
	case source == "organization" && typ == orgapi.EventHRRequestDecided:
		var p orgapi.HRRequestDecided
		if err := json.Unmarshal(payload, &p); err != nil {
			return received{}, err
		}
		f := map[string]string{"request_id": p.RequestID, "decision": p.Decision, "from_status": p.FromStatus, "to_status": p.ToStatus,
			"actor_user_id": p.ActorUserID, "before_version": strconv.FormatInt(p.BeforeVersion, 10)}
		put(f, "next_reviewer_user_id", p.NextReviewerUserID)
		put(f, "notes", p.Notes)
		return received{commandID: p.CommandID, key: p.IdempotencyKey, corr: p.CorrelationID, mode: p.RolloutMode, subjectKind: "hr_request",
			subjectID: p.RequestID, eventType: TypeHRDecide, fields: f, dependsOnSubject: p.RequestID}, nil
	}
	return received{}, fmt.Errorf("%w: %s from %s", ErrUnknownEvent, typ, source)
}

// Receive records one source outbox event as a write-back command inside an
// integration transaction. The inbox makes a redelivery a no-op.
func (w *Worker) Receive(ctx context.Context, source owner.Module, rec outbox.Record) (inbox.Outcome, error) {
	r, err := decode(source, rec.Type, rec.Payload)
	if err != nil {
		return 0, err
	}
	if r.mode != modeShadow && r.mode != modeControlled {
		return 0, fmt.Errorf("writeback: event %s has rollout mode %q", rec.ID, r.mode)
	}
	var out inbox.Outcome
	err = w.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := w.clock.Now()
		var err error
		out, err = inbox.Consume(ctx, tx, Consumer, rec.ID, now, func(ctx context.Context) error {
			intent, _ := json.Marshal(r.fields)
			var depends *string
			if r.dependsOnSubject != "" && r.mode == modeControlled {
				var dep string
				err := tx.QueryRow(ctx, `SELECT command_id::text FROM integration.writeback_command WHERE subject_kind = 'hr_request'
					AND subject_id = $1 AND event_type = $2 AND rollout_mode = $3`, r.dependsOnSubject, TypeHRCreate, modeControlled).Scan(&dep)
				if err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
				if err == nil {
					depends = &dep
				}
			}
			state := StatePending
			if depends != nil || (r.dependsOnSubject != "" && r.mode == modeControlled) {
				state = StateWaiting
			}
			_, err := tx.Exec(ctx, `INSERT INTO integration.writeback_command (command_id, source_module, event_id, event_type, subject_kind,
				subject_id, idempotency_key, correlation_id, rollout_mode, intent, depends_on_command_id, state, attempts, version, received_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 0, 1, $13, $13)`,
				r.commandID, string(source), rec.ID, r.eventType, r.subjectKind, r.subjectID, r.key, r.corr, r.mode, intent, depends, state, now.Time())
			return err
		})
		return err
	})
	return out, err
}

// Drain claims up to limit undispatched events from a source module's own
// outbox (a source-owned transaction), records each as a write-back command
// (integration-owned transaction, inbox-deduplicated) and marks it
// dispatched in the source outbox. No transaction spans two schemas.
func (w *Worker) Drain(ctx context.Context, source *db.Runner, limit int) (int, error) {
	var recs []outbox.Record
	if err := source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		var err error
		recs, err = outbox.Claim(ctx, tx, limit, 10)
		return err
	}); err != nil {
		return 0, err
	}
	n := 0
	for _, rec := range recs {
		if _, err := w.Receive(ctx, source.Module(), rec); err != nil {
			reason := err.Error()
			_ = source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
				return outbox.MarkFailed(ctx, tx, rec.ID, w.clock.Now(), reason)
			})
			continue
		}
		if err := source.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
			return outbox.MarkDispatched(ctx, tx, rec.ID, w.clock.Now())
		}); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// command is a stored write-back command.
type command struct {
	ID, Module, EventType, SubjectKind, SubjectID, Key, Corr, Mode, State string
	Depends                                                               *string
	Attempts                                                              int
	Version                                                               int64
	Fields                                                                map[string]string
}

func (c command) intent(dependsRef string) Intent {
	return Intent{CommandID: c.ID, IdempotencyKey: c.Key, CorrelationID: c.Corr, CommandType: c.EventType, SubjectKind: c.SubjectKind,
		SubjectID: c.SubjectID, Fields: c.Fields, DependsOnLegacyRef: dependsRef}
}

const commandCols = `command_id::text, source_module, event_type, subject_kind, subject_id::text, idempotency_key, correlation_id, rollout_mode,
	state, depends_on_command_id::text, attempts, version, intent`

func scanCommand(r pgx.Row) (command, error) {
	var c command
	var intent []byte
	if err := r.Scan(&c.ID, &c.Module, &c.EventType, &c.SubjectKind, &c.SubjectID, &c.Key, &c.Corr, &c.Mode, &c.State, &c.Depends, &c.Attempts,
		&c.Version, &intent); err != nil {
		return c, err
	}
	return c, json.Unmarshal(intent, &c.Fields)
}

// Report counts what one Step did.
type Report struct {
	Applied, Unknown, Rejected, Reconciliation, Compared, Waiting, Retried int
}

// Step processes up to limit actionable commands (oldest first), one at a
// time, each with its own claim. It is safe to run concurrently.
func (w *Worker) Step(ctx context.Context, limit int) (Report, error) {
	var rep Report
	var pending []string
	if err := w.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		rows, err := tx.Query(ctx, `SELECT command_id::text FROM integration.writeback_command
			WHERE state IN ('PENDING', 'WAITING_DEPENDENCY', 'UNKNOWN') ORDER BY received_at, command_id LIMIT $1`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			pending = append(pending, id)
		}
		return rows.Err()
	}); err != nil {
		return rep, err
	}
	for _, id := range pending {
		if err := w.process(ctx, id, &rep); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

func (w *Worker) load(ctx context.Context, tx db.Tx, id string, lock bool) (command, bool, error) {
	q := `SELECT ` + commandCols + ` FROM integration.writeback_command WHERE command_id = $1`
	if lock {
		q += ` FOR UPDATE SKIP LOCKED`
	}
	c, err := scanCommand(tx.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, false, nil
	}
	return c, err == nil, err
}

func (w *Worker) process(ctx context.Context, id string, rep *Report) error {
	var c command
	var ok bool
	if err := w.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var err error
		c, ok, err = w.load(ctx, tx, id, false)
		return err
	}); err != nil || !ok {
		return err
	}
	switch {
	case c.Mode == modeShadow && c.State == StatePending:
		return w.compare(ctx, c, rep)
	case c.State == StateWaiting:
		return w.resolveDependency(ctx, c, rep)
	case c.State == StateUnknown:
		return w.reconcile(ctx, c, rep)
	case c.State == StatePending:
		return w.apply(ctx, c, rep)
	}
	return nil
}

// dependencyRef returns the legacy ref of the command c depends on.
func (w *Worker) dependencyRef(ctx context.Context, c command) (ref, state string, err error) {
	if c.Depends == nil {
		return "", "", nil
	}
	err = w.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var r *string
		if err := tx.QueryRow(ctx, `SELECT state, legacy_ref FROM integration.writeback_command WHERE command_id = $1`, *c.Depends).Scan(&state, &r); err != nil {
			return err
		}
		ref = str(r)
		return nil
	})
	return ref, state, err
}

func (w *Worker) resolveDependency(ctx context.Context, c command, rep *Report) error {
	if c.Depends == nil {
		// The request's create may arrive after its decision (same-instant
		// events). Look it up again; a create recorded only in SHADOW_COMPARE
		// means the request does not exist in legacy.
		var dep, mode string
		err := w.runner.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			return tx.QueryRow(ctx, `SELECT command_id::text, rollout_mode FROM integration.writeback_command WHERE subject_kind = 'hr_request'
				AND subject_id = $1 AND event_type = $2 ORDER BY received_at LIMIT 1`, c.SubjectID, TypeHRCreate).Scan(&dep, &mode)
		})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			rep.Waiting++
			return nil
		case err != nil:
			return err
		case mode != modeControlled:
			rep.Reconciliation++
			return w.transition(ctx, c, StateReconciliation, nil, &recon{reason: "WRITEBACK_REJECTED", action: "DOMAIN_OWNER_REVIEW",
				evidence: map[string]any{"detail": "the request was created in SHADOW_COMPARE and does not exist in legacy"}})
		}
		if err := w.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE integration.writeback_command SET depends_on_command_id = $3, version = version + 1, updated_at = $4
				WHERE command_id = $1 AND version = $2`, c.ID, c.Version, dep, w.clock.Now().Time())
			return err
		}); err != nil {
			return err
		}
		c.Depends = &dep
		c.Version++
	}
	ref, state, err := w.dependencyRef(ctx, c)
	if err != nil {
		return err
	}
	switch state {
	case StateApplied:
		if ref == "" {
			rep.Reconciliation++
			return w.transition(ctx, c, StateReconciliation, nil, &recon{reason: "LEGACY_AMBIGUOUS", action: "DOMAIN_OWNER_REVIEW",
				evidence: map[string]any{"detail": "request create applied without a legacy reference"}})
		}
		rep.Retried++
		return w.transition(ctx, c, StatePending, nil, nil)
	case StateRejected, StateReconciliation:
		rep.Reconciliation++
		return w.transition(ctx, c, StateReconciliation, nil, &recon{reason: "WRITEBACK_REJECTED", action: "DOMAIN_OWNER_REVIEW",
			evidence: map[string]any{"detail": "request create was not applied in legacy", "dependency_state": state}})
	}
	rep.Waiting++
	return nil
}

// apply delivers a PENDING command: write-ahead claim (→ UNKNOWN), legacy
// call outside any transaction, then the recorded outcome.
func (w *Worker) apply(ctx context.Context, c command, rep *Report) error {
	ref, depState, err := w.dependencyRef(ctx, c)
	if err != nil {
		return err
	}
	if c.Depends != nil && (depState != StateApplied || ref == "") {
		return w.transition(ctx, c, StateWaiting, nil, nil)
	}
	claimed, attemptNo, err := w.claim(ctx, c)
	if err != nil || !claimed {
		return err
	}
	c.Version++
	c.State = StateUnknown
	started := w.clock.Now().Time()
	res, callErr := w.connector.Apply(ctx, c.intent(ref))
	finished := w.clock.Now().Time()
	if callErr != nil {
		res = Applied{Outcome: result.Unknown, Reasons: []string{"OUTCOME_UNKNOWN"}, Detail: truncate("transport: " + callErr.Error())}
	}
	att := attempt{no: attemptNo, kind: "APPLY", outcome: res.Outcome, reasons: res.Reasons, legacyRef: res.LegacyRef, detail: res.Detail,
		started: started, finished: finished}
	switch res.Outcome {
	case result.Success:
		rep.Applied++
		if err := w.transition(ctx, c, StateApplied, &att, nil, withRef(res.LegacyRef)); err != nil {
			return err
		}
		c.Version++
		c.State = StateApplied
		return w.parity(ctx, c, ref, rep)
	case result.Unknown, result.Partial:
		rep.Unknown++
		return w.transition(ctx, c, StateUnknown, &att, &recon{reason: "WRITEBACK_UNKNOWN", action: "OBSERVE_LEGACY_BEFORE_RETRY",
			attempt: attemptNo, evidence: map[string]any{"outcome": res.Outcome.String(), "detail": res.Detail}})
	default: // FAILED, CONFLICT, BLOCKED, UNAUTHORIZED: legacy refused, nothing committed.
		rep.Rejected++
		return w.transition(ctx, c, StateRejected, &att, &recon{reason: "WRITEBACK_REJECTED", action: "DOMAIN_OWNER_REVIEW",
			attempt: attemptNo, evidence: map[string]any{"outcome": res.Outcome.String(), "reasons": res.Reasons, "detail": res.Detail}})
	}
}

// reconcile observes legacy for an UNKNOWN command before anything else.
func (w *Worker) reconcile(ctx context.Context, c command, rep *Report) error {
	ref, _, err := w.dependencyRef(ctx, c)
	if err != nil {
		return err
	}
	started := w.clock.Now().Time()
	obs, callErr := w.connector.Observe(ctx, c.intent(ref))
	finished := w.clock.Now().Time()
	if callErr != nil {
		obs = Observed{Observation: Unseen, Detail: truncate("transport: " + callErr.Error())}
	}
	att := attempt{no: c.Attempts + 1, kind: "OBSERVE", outcome: observeOutcome(obs.Observation), observation: obs.Observation,
		reasons: []string{}, legacyRef: obs.LegacyRef, detail: obs.Detail, started: started, finished: finished}
	switch obs.Observation {
	case Committed:
		rep.Applied++
		if err := w.transition(ctx, c, StateApplied, &att, nil, withRef(obs.LegacyRef), countAttempt()); err != nil {
			return err
		}
		c.Version++
		c.State = StateApplied
		return w.recordParity(ctx, c, obs)
	case Absent:
		// Proven not applied: a retry of the same intent is safe now.
		rep.Retried++
		return w.transition(ctx, c, StatePending, &att, nil, countAttempt())
	case Ambiguous:
		rep.Reconciliation++
		return w.transition(ctx, c, StateReconciliation, &att, &recon{reason: "LEGACY_AMBIGUOUS", action: "DOMAIN_OWNER_REVIEW",
			attempt: c.Attempts + 1, evidence: map[string]any{"detail": obs.Detail}}, countAttempt())
	}
	rep.Unknown++
	return w.transition(ctx, c, StateUnknown, &att, nil, countAttempt())
}

func observeOutcome(o Observation) result.Status {
	switch o {
	case Committed, Absent:
		return result.Success
	case Ambiguous:
		return result.Conflict
	}
	return result.Unknown
}

// compare observes legacy for a SHADOW_COMPARE command (no legacy write).
func (w *Worker) compare(ctx context.Context, c command, rep *Report) error {
	started := w.clock.Now().Time()
	obs, callErr := w.connector.Observe(ctx, c.intent(""))
	finished := w.clock.Now().Time()
	if callErr != nil {
		obs = Observed{Observation: Unseen, Detail: truncate("transport: " + callErr.Error())}
	}
	att := attempt{no: c.Attempts + 1, kind: "OBSERVE", outcome: observeOutcome(obs.Observation), observation: obs.Observation,
		reasons: []string{}, legacyRef: obs.LegacyRef, detail: obs.Detail, started: started, finished: finished}
	if err := w.transition(ctx, c, StateCompared, &att, nil, countAttempt()); err != nil {
		return err
	}
	rep.Compared++
	c.Version++
	c.State = StateCompared
	return w.recordParity(ctx, c, obs)
}

func (w *Worker) parity(ctx context.Context, c command, ref string, rep *Report) error {
	obs, err := w.connector.Observe(ctx, c.intent(ref))
	if err != nil {
		obs = Observed{Observation: Unseen, Detail: truncate("transport: " + err.Error())}
	}
	return w.recordParity(ctx, c, obs)
}

// ParityOf classifies an observation against the expected effect.
func ParityOf(expected map[string]string, obs Observed) string {
	switch obs.Observation {
	case Unseen:
		return "UNKNOWN"
	case Ambiguous:
		return "CONFLICT"
	case Absent:
		return "INCOMPLETE"
	}
	for k, v := range expected {
		if got, ok := obs.State[k]; !ok || got != v {
			return "MISMATCH"
		}
	}
	return "MATCH"
}

func (w *Worker) recordParity(ctx context.Context, c command, obs Observed) error {
	expected := Expected(c.intent(""))
	res := ParityOf(expected, obs)
	observed := obs.State
	if observed == nil {
		observed = map[string]string{}
	}
	return w.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := w.clock.Now().Time()
		ej, _ := json.Marshal(expected)
		oj, _ := json.Marshal(map[string]any{"observation": obs.Observation, "state": observed})
		tag, err := tx.Exec(ctx, `INSERT INTO integration.writeback_parity (parity_id, command_id, rollout_mode, result, expected, observed,
			correlation_id, observed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (command_id) DO NOTHING`,
			ids.Stable("integration.writeback_parity", c.ID), c.ID, c.Mode, res, ej, oj, c.Corr, now)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE integration.writeback_command SET parity = $2, version = version + 1, updated_at = $3 WHERE command_id = $1`,
			c.ID, res, now); err != nil {
			return err
		}
		if res == "MISMATCH" || res == "CONFLICT" {
			// Parity never corrects anything; it opens a diagnostic record.
			ev, _ := json.Marshal(map[string]any{"expected": expected, "observed": observed})
			_, err = tx.Exec(ctx, `INSERT INTO integration.writeback_reconciliation (reconciliation_id, command_id, domain_owner, subject_kind,
				subject_id, command_type, reason, attempt_no, evidence, next_safe_action, correlation_id, opened_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'PARITY_MISMATCH', $7, $8, 'NO_AUTOMATIC_ACTION', $9, $10) ON CONFLICT DO NOTHING`,
				ids.Stable("integration.writeback_reconciliation", c.ID, "PARITY_MISMATCH"), c.ID, c.Module, c.SubjectKind, c.SubjectID,
				c.EventType, max(c.Attempts, 1), ev, c.Corr, now)
		}
		return err
	})
}

// claim durably moves a PENDING command to UNKNOWN before the legacy call.
func (w *Worker) claim(ctx context.Context, c command) (bool, int, error) {
	var ok bool
	no := c.Attempts + 1
	err := w.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		cur, found, err := w.load(ctx, tx, c.ID, true)
		if err != nil || !found || cur.State != StatePending || cur.Version != c.Version {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE integration.writeback_command SET state = 'UNKNOWN', attempts = attempts + 1, version = version + 1,
			updated_at = $3 WHERE command_id = $1 AND version = $2 AND state = 'PENDING'`, c.ID, c.Version, w.clock.Now().Time())
		ok = err == nil && tag.RowsAffected() == 1
		return err
	})
	return ok, no, err
}

type attempt struct {
	no                int
	kind              string
	outcome           result.Status
	observation       Observation
	reasons           []string
	legacyRef, detail string
	started, finished time.Time
}

type recon struct {
	reason, action string
	attempt        int
	evidence       map[string]any
}

type txOpt func(*txOpts)

type txOpts struct {
	ref        *string
	addAttempt bool
}

func withRef(r string) txOpt {
	return func(o *txOpts) {
		if r != "" {
			o.ref = &r
		}
	}
}

func countAttempt() txOpt { return func(o *txOpts) { o.addAttempt = true } }

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

// transition records an attempt (append-only), the new state (versioned)
// and an optional reconciliation record in one integration transaction.
func (w *Worker) transition(ctx context.Context, c command, state string, a *attempt, r *recon, opts ...txOpt) error {
	var o txOpts
	for _, f := range opts {
		f(&o)
	}
	return w.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		now := w.clock.Now().Time()
		inc := 0
		if o.addAttempt {
			inc = 1
		}
		tag, err := tx.Exec(ctx, `UPDATE integration.writeback_command SET state = $3, attempts = attempts + $4,
			legacy_ref = COALESCE($5, legacy_ref), version = version + 1, updated_at = $6 WHERE command_id = $1 AND version = $2`,
			c.ID, c.Version, state, inc, o.ref, now)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return nil // another worker moved it; its own attempt record stands
		}
		if a != nil {
			reasons := a.reasons
			if reasons == nil {
				reasons = []string{}
			}
			sort.Strings(reasons)
			rj, _ := json.Marshal(reasons)
			var obs *string
			if a.kind == "OBSERVE" {
				s := string(a.observation)
				obs = &s
			}
			var ref, detail *string
			if a.legacyRef != "" {
				ref = &a.legacyRef
			}
			if a.detail != "" {
				d := truncate(a.detail)
				detail = &d
			}
			if _, err := tx.Exec(ctx, `INSERT INTO integration.writeback_attempt (attempt_id, command_id, attempt_no, kind, outcome, observation,
				reasons, legacy_ref, detail, service_actor, correlation_id, started_at, finished_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
				ids.Stable("integration.writeback_attempt", c.ID, strconv.Itoa(a.no)), c.ID, a.no, a.kind, a.outcome.String(), obs, rj, ref,
				detail, ServiceActor, c.Corr, a.started, a.finished); err != nil {
				return err
			}
		}
		if r != nil {
			ev := r.evidence
			if ev == nil {
				ev = map[string]any{}
			}
			ej, _ := json.Marshal(ev)
			no := r.attempt
			if no < 1 {
				no = max(c.Attempts, 1)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO integration.writeback_reconciliation (reconciliation_id, command_id, domain_owner, subject_kind,
				subject_id, command_type, reason, attempt_no, evidence, next_safe_action, correlation_id, opened_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) ON CONFLICT DO NOTHING`,
				ids.Stable("integration.writeback_reconciliation", c.ID, r.reason, strconv.Itoa(no)), c.ID, c.Module, c.SubjectKind, c.SubjectID,
				c.EventType, r.reason, no, ej, r.action, c.Corr, now); err != nil {
				return err
			}
		}
		return nil
	})
}
