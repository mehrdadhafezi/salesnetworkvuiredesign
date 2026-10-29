// Package outbox is the per-module transactional outbox (Architecture V1 §6
// rules 2–4, §8.8).
//
// Outbox rows live in the writing module's own schema (<module>.outbox_event,
// created by migrations.RenderModuleInfra). Append writes into the schema of
// the transaction's owner, so a domain change and its event commit or roll
// back together and no module can append to another module's outbox. There is
// no shared, global outbox table.
//
// Delivery mechanics (Claim / MarkDispatched / MarkFailed) are primitives for
// a later dispatcher; dispatch is at-least-once, consumers deduplicate with
// the inbox package.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrInvalidEvent reports an event that breaks the outbox contract.
	ErrInvalidEvent = errors.New("outbox: invalid event")
	// ErrReadOnly reports a write attempted in a read-only transaction.
	ErrReadOnly = errors.New("outbox: read-only transaction")
	// ErrNotFound reports a dispatch update for an unknown event.
	ErrNotFound = errors.New("outbox: event not found")
)

// MaxPayloadBytes bounds an event payload.
const MaxPayloadBytes = 64 << 10

const maxErrorLen = 1000

var eventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// Event is a domain event recorded by its owning module.
type Event struct {
	ID            string // UUID; the deterministic dispatch identity
	Type          string // "<module>.<name>", lowercase dotted
	Version       int
	Subject       *refs.NativeRef
	Payload       json.RawMessage // a JSON object
	OccurredAt    timeutil.Instant
	CorrelationID *string
}

// NewEventID returns a fresh event identity.
func NewEventID() (string, error) { return ids.NewUUID() }

// Validate checks the event fields.
func (e Event) Validate() error {
	if err := ids.ValidateUUID(e.ID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	if !eventTypePattern.MatchString(e.Type) || len(e.Type) > 128 {
		return fmt.Errorf("%w: type %q", ErrInvalidEvent, e.Type)
	}
	if e.Version < 1 {
		return fmt.Errorf("%w: version must be >= 1", ErrInvalidEvent)
	}
	if e.Subject != nil {
		if err := e.Subject.Validate(); err != nil {
			return fmt.Errorf("%w: subject: %v", ErrInvalidEvent, err)
		}
	}
	if len(e.Payload) == 0 || len(e.Payload) > MaxPayloadBytes {
		return fmt.Errorf("%w: payload size", ErrInvalidEvent)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &obj); err != nil || obj == nil {
		return fmt.Errorf("%w: payload must be a JSON object", ErrInvalidEvent)
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at required", ErrInvalidEvent)
	}
	if e.CorrelationID != nil && (len(*e.CorrelationID) == 0 || len(*e.CorrelationID) > 128) {
		return fmt.Errorf("%w: correlation id length", ErrInvalidEvent)
	}
	return nil
}

func table(tx db.Tx) string {
	return pgx.Identifier{tx.Module().Schema(), "outbox_event"}.Sanitize()
}

// Append records e in the outbox of tx's owning module, inside tx.
func Append(ctx context.Context, tx db.Tx, e Event, recordedAt timeutil.Instant) error {
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	if err := e.Validate(); err != nil {
		return err
	}
	if recordedAt.IsZero() {
		return fmt.Errorf("%w: recorded_at required", ErrInvalidEvent)
	}
	var subjectDomain, subjectID *string
	if e.Subject != nil {
		d, id := string(e.Subject.Domain), e.Subject.ID
		subjectDomain, subjectID = &d, &id
	}
	_, err := tx.Exec(ctx, `INSERT INTO `+table(tx)+`
		(event_id, event_type, event_version, subject_domain, subject_id, payload, occurred_at, recorded_at, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		e.ID, e.Type, e.Version, subjectDomain, subjectID, []byte(e.Payload), e.OccurredAt.Time(), recordedAt.Time(), e.CorrelationID)
	if err != nil {
		return fmt.Errorf("outbox: append: %w", err)
	}
	return nil
}

// Record is a stored outbox row as seen by a dispatcher.
type Record struct {
	Event
	Attempts int
}

// Claim locks up to limit undispatched events of tx's module, oldest first,
// skipping rows locked by another dispatcher. Events that already reached
// maxAttempts are not claimed.
func Claim(ctx context.Context, tx db.Tx, limit, maxAttempts int) ([]Record, error) {
	if tx.ReadOnly() {
		return nil, ErrReadOnly
	}
	if limit < 1 || limit > 1000 || maxAttempts < 1 {
		return nil, fmt.Errorf("%w: claim bounds", ErrInvalidEvent)
	}
	rows, err := tx.Query(ctx, `SELECT event_id::text, event_type, event_version, subject_domain, subject_id, payload,
			occurred_at, correlation_id, attempts
		FROM `+table(tx)+`
		WHERE dispatch_state <> 'dispatched' AND attempts < $1
		ORDER BY recorded_at, event_id
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, maxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("outbox: claim: %w", err)
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var (
			r                     Record
			subjectDomain, subjID *string
			payload               []byte
			occurred              time.Time
		)
		if err := rows.Scan(&r.ID, &r.Type, &r.Version, &subjectDomain, &subjID, &payload, &occurred, &r.CorrelationID, &r.Attempts); err != nil {
			return nil, fmt.Errorf("outbox: scan: %w", err)
		}
		if subjectDomain != nil && subjID != nil {
			ref := refs.NativeRef{Domain: refs.Domain(*subjectDomain), ID: *subjID}
			r.Subject = &ref
		}
		r.Payload = payload
		if r.OccurredAt, err = timeutil.NewInstant(occurred); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkDispatched records successful delivery of event id.
func MarkDispatched(ctx context.Context, tx db.Tx, id string, at timeutil.Instant) error {
	return update(ctx, tx, `UPDATE `+table(tx)+`
		SET dispatch_state = 'dispatched', dispatched_at = $2, last_attempt_at = $2, attempts = attempts + 1, last_error = NULL
		WHERE event_id = $1 AND dispatch_state <> 'dispatched'`, id, at.Time())
}

// MarkFailed records a failed delivery attempt; the event stays claimable
// until it reaches the dispatcher's maxAttempts.
func MarkFailed(ctx context.Context, tx db.Tx, id string, at timeutil.Instant, reason string) error {
	if len(reason) > maxErrorLen {
		reason = reason[:maxErrorLen]
	}
	return update(ctx, tx, `UPDATE `+table(tx)+`
		SET dispatch_state = 'failed', last_attempt_at = $2, attempts = attempts + 1, last_error = $3
		WHERE event_id = $1 AND dispatch_state <> 'dispatched'`, id, at.Time(), reason)
}

func update(ctx context.Context, tx db.Tx, sql, id string, args ...any) error {
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	if err := ids.ValidateUUID(id); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	tag, err := tx.Exec(ctx, sql, append([]any{id}, args...)...)
	if err != nil {
		return fmt.Errorf("outbox: update: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
