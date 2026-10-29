// Package audit is the append-only audit foundation (Architecture V1 §7,
// §8.7; E0 §6 #9). Rows are INSERT-only: the owner role has only SELECT and
// INSERT, and a trigger rejects UPDATE, DELETE and TRUNCATE even for the
// table owner. Business modules do not write audit rows directly; in later
// slices audit consumes their outbox events in its own transaction.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

var (
	// ErrInvalidEvent reports an event that breaks the audit contract.
	ErrInvalidEvent = errors.New("audit: invalid event")
	// ErrWrongOwner reports an append from a transaction not owned by audit.
	ErrWrongOwner = errors.New("audit: transaction is not owned by the audit module")
	// ErrReadOnly reports an append inside a read-only transaction.
	ErrReadOnly = errors.New("audit: read-only transaction")
)

// Bounds keep metadata small and structured; it is never an object dump.
const (
	MaxMetadataEntries = 32
	maxMetadataValue   = 256
	maxReasonLen       = 1000
)

var (
	actionPattern      = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	metadataKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	// secretKeys are refused as metadata keys so credentials never land in audit.
	secretKeys = []string{"password", "passwd", "secret", "token", "credential", "api_key", "apikey", "otp", "pin", "cvv", "card_number"}
)

// Event is one audit record.
type Event struct {
	ID            string // UUID
	Actor         *refs.NativeRef
	Subject       refs.NativeRef
	Action        string // dotted lowercase, e.g. "platform.flag_changed"
	Reason        *string
	CorrelationID *string
	OccurredAt    timeutil.Instant
	Metadata      map[string]string
}

// NewEventID returns a fresh audit event identity.
func NewEventID() (string, error) { return ids.NewUUID() }

// Validate checks the event and its bounded metadata.
func (e Event) Validate() error {
	if err := ids.ValidateUUID(e.ID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	if e.Actor != nil {
		if err := e.Actor.Validate(); err != nil {
			return fmt.Errorf("%w: actor: %v", ErrInvalidEvent, err)
		}
	}
	if err := e.Subject.Validate(); err != nil {
		return fmt.Errorf("%w: subject: %v", ErrInvalidEvent, err)
	}
	if !actionPattern.MatchString(e.Action) || len(e.Action) > 128 {
		return fmt.Errorf("%w: action %q", ErrInvalidEvent, e.Action)
	}
	if e.Reason != nil && (len(*e.Reason) == 0 || len(*e.Reason) > maxReasonLen) {
		return fmt.Errorf("%w: reason length", ErrInvalidEvent)
	}
	if e.CorrelationID != nil && (len(*e.CorrelationID) == 0 || len(*e.CorrelationID) > 128) {
		return fmt.Errorf("%w: correlation id length", ErrInvalidEvent)
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at required", ErrInvalidEvent)
	}
	if len(e.Metadata) > MaxMetadataEntries {
		return fmt.Errorf("%w: too many metadata entries", ErrInvalidEvent)
	}
	for k, v := range e.Metadata {
		if !metadataKeyPattern.MatchString(k) {
			return fmt.Errorf("%w: metadata key %q", ErrInvalidEvent, k)
		}
		for _, s := range secretKeys {
			if strings.Contains(k, s) {
				return fmt.Errorf("%w: metadata key %q looks like a secret", ErrInvalidEvent, k)
			}
		}
		if len(v) > maxMetadataValue {
			return fmt.Errorf("%w: metadata value for %q too long", ErrInvalidEvent, k)
		}
	}
	return nil
}

// Append inserts e. tx must be owned by the audit module.
func Append(ctx context.Context, tx db.Tx, e Event) error {
	if tx.Module() != owner.Audit {
		return ErrWrongOwner
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	if err := e.Validate(); err != nil {
		return err
	}
	meta := e.Metadata
	if meta == nil {
		meta = map[string]string{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("%w: metadata: %v", ErrInvalidEvent, err)
	}
	var actorDomain, actorID *string
	if e.Actor != nil {
		d, id := string(e.Actor.Domain), e.Actor.ID
		actorDomain, actorID = &d, &id
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit.audit_event
		(event_id, occurred_at, actor_domain, actor_id, subject_domain, subject_id, action, reason, correlation_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.ID, e.OccurredAt.Time(), actorDomain, actorID, string(e.Subject.Domain), e.Subject.ID, e.Action, e.Reason, e.CorrelationID, metaJSON)
	if err != nil {
		return fmt.Errorf("audit: append: %w", err)
	}
	return nil
}
