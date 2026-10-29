// Package inbox is the per-module idempotent consumption record
// (Architecture V1 §6 rule 4, §8.8). It is technical delivery state only,
// never business outcome state.
//
// Consume runs the handler and records the message as processed in the same
// transaction as the handler's own writes, in the consuming module's schema.
// A concurrent duplicate blocks on the row and then sees it processed, so a
// handler never runs twice for one (consumer, message) pair.
package inbox

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

var (
	// ErrInvalidMessage reports a malformed consumer or message id.
	ErrInvalidMessage = errors.New("inbox: invalid message")
	// ErrReadOnly reports use inside a read-only transaction.
	ErrReadOnly = errors.New("inbox: read-only transaction")
)

const maxErrorLen = 1000

var consumerPattern = regexp.MustCompile(`^[a-z][a-z0-9_.]{1,127}$`)

// Outcome is the delivery decision for one message.
type Outcome uint8

const (
	outcomeInvalid Outcome = iota
	// Processed: the handler ran and committed with this transaction.
	Processed
	// Duplicate: the message was already processed; the handler did not run.
	Duplicate
)

func (o Outcome) String() string {
	switch o {
	case Processed:
		return "processed"
	case Duplicate:
		return "duplicate"
	default:
		return "invalid"
	}
}

func table(tx db.Tx) string {
	return pgx.Identifier{tx.Module().Schema(), "inbox_message"}.Sanitize()
}

func validate(consumer, messageID string) error {
	if !consumerPattern.MatchString(consumer) {
		return fmt.Errorf("%w: consumer %q", ErrInvalidMessage, consumer)
	}
	if err := ids.ValidateUUID(messageID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	return nil
}

// Consume claims (consumer, messageID) in tx's module inbox and, unless the
// message was already processed, runs handle and marks it processed. If
// handle fails, the caller's transaction must roll back; the claim then
// disappears with it and the message stays deliverable.
func Consume(ctx context.Context, tx db.Tx, consumer, messageID string, at timeutil.Instant, handle func(ctx context.Context) error) (Outcome, error) {
	if tx.ReadOnly() {
		return outcomeInvalid, ErrReadOnly
	}
	if err := validate(consumer, messageID); err != nil {
		return outcomeInvalid, err
	}
	if at.IsZero() {
		return outcomeInvalid, fmt.Errorf("%w: timestamp required", ErrInvalidMessage)
	}
	var state string
	err := tx.QueryRow(ctx, `INSERT INTO `+table(tx)+` AS m
			(consumer, message_id, state, first_seen_at, last_attempt_at, attempts)
		VALUES ($1, $2, 'processing', $3, $3, 1)
		ON CONFLICT (consumer, message_id) DO UPDATE
			SET state = 'processing', last_attempt_at = EXCLUDED.last_attempt_at, attempts = m.attempts + 1
			WHERE m.state <> 'processed'
		RETURNING m.state`, consumer, messageID, at.Time()).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return Duplicate, nil
	}
	if err != nil {
		return outcomeInvalid, fmt.Errorf("inbox: claim: %w", err)
	}
	if err := handle(ctx); err != nil {
		return outcomeInvalid, err
	}
	if _, err := tx.Exec(ctx, `UPDATE `+table(tx)+`
		SET state = 'processed', processed_at = $3, last_error = NULL
		WHERE consumer = $1 AND message_id = $2`, consumer, messageID, at.Time()); err != nil {
		return outcomeInvalid, fmt.Errorf("inbox: mark processed: %w", err)
	}
	return Processed, nil
}

// RecordFailure records a failed delivery attempt in its own transaction,
// after the consuming transaction rolled back. first_seen_at is preserved; an
// already processed message is never downgraded.
func RecordFailure(ctx context.Context, tx db.Tx, consumer, messageID string, at timeutil.Instant, reason string) error {
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	if err := validate(consumer, messageID); err != nil {
		return err
	}
	if len(reason) > maxErrorLen {
		reason = reason[:maxErrorLen]
	}
	_, err := tx.Exec(ctx, `INSERT INTO `+table(tx)+` AS m
			(consumer, message_id, state, first_seen_at, last_attempt_at, attempts, last_error)
		VALUES ($1, $2, 'failed', $3, $3, 1, $4)
		ON CONFLICT (consumer, message_id) DO UPDATE
			SET state = 'failed', last_attempt_at = EXCLUDED.last_attempt_at, attempts = m.attempts + 1, last_error = EXCLUDED.last_error
			WHERE m.state <> 'processed'`, consumer, messageID, at.Time(), reason)
	if err != nil {
		return fmt.Errorf("inbox: record failure: %w", err)
	}
	return nil
}

// Status is the stored delivery state of one message.
type Status struct {
	State    string
	Attempts int
}

// Lookup returns the stored state of (consumer, messageID), if any.
func Lookup(ctx context.Context, tx db.Tx, consumer, messageID string) (Status, bool, error) {
	if err := validate(consumer, messageID); err != nil {
		return Status{}, false, err
	}
	var s Status
	err := tx.QueryRow(ctx, `SELECT state, attempts FROM `+table(tx)+` WHERE consumer = $1 AND message_id = $2`,
		consumer, messageID).Scan(&s.State, &s.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return Status{}, false, nil
	}
	if err != nil {
		return Status{}, false, fmt.Errorf("inbox: lookup: %w", err)
	}
	return s, true, nil
}
