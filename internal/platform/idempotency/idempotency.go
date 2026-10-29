// Package idempotency is per-module command idempotency storage
// (Architecture V1 §8.8, §15.2.6, §23): UNIQUE (scope, key) in the owning
// module's schema.
//
//   - same key + same request hash  → Replay of the stored result
//   - same key + different hash     → Conflict
//   - first use                     → Acquired; the caller performs the
//     command and calls Complete in the SAME transaction
//
// Acquisition is concurrency-safe: a second transaction using the same key
// blocks on the unique row until the first commits or rolls back, then sees
// the committed record (Replay/Conflict) or acquires the key itself.
package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
)

var (
	// ErrInvalidRequest reports a malformed scope, key or hash.
	ErrInvalidRequest = errors.New("idempotency: invalid request")
	// ErrReadOnly reports use inside a read-only transaction.
	ErrReadOnly = errors.New("idempotency: read-only transaction")
	// ErrNotAcquired reports Complete without a matching in-progress acquisition.
	ErrNotAcquired = errors.New("idempotency: key not acquired by this request")
	// ErrResultTooLarge reports a stored result above MaxResultBytes.
	ErrResultTooLarge = errors.New("idempotency: result too large")
)

// MaxResultBytes bounds a stored technical result.
const MaxResultBytes = 64 << 10

var (
	scopePattern = regexp.MustCompile(`^[a-z][a-z0-9_.]{1,63}$`)
	keyPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
)

// Hash is the SHA-256 of a request's canonical encoding.
type Hash [32]byte

// HashRequest hashes the caller's canonical request bytes. Callers must use a
// deterministic encoding; the platform does not guess one.
func HashRequest(canonical []byte) Hash { return sha256.Sum256(canonical) }

// Request identifies one idempotent command invocation.
type Request struct {
	Scope string
	Key   string
	Hash  Hash
}

// Validate checks scope, key and hash.
func (r Request) Validate() error {
	if !scopePattern.MatchString(r.Scope) {
		return fmt.Errorf("%w: scope %q", ErrInvalidRequest, r.Scope)
	}
	if !keyPattern.MatchString(r.Key) {
		return fmt.Errorf("%w: key format", ErrInvalidRequest)
	}
	if r.Hash == (Hash{}) {
		return fmt.Errorf("%w: zero hash", ErrInvalidRequest)
	}
	return nil
}

// State is the result of Acquire.
type State uint8

const (
	stateInvalid State = iota
	// Acquired: first use; perform the command, then Complete.
	Acquired
	// Replay: same key and same request already completed; return Result.
	Replay
	// Conflict: same key used with a different request.
	Conflict
	// InProgress: a committed record was never completed. Its outcome is
	// unknown and must be reconciled; never re-run the command blindly.
	InProgress
)

func (s State) String() string {
	switch s {
	case Acquired:
		return "acquired"
	case Replay:
		return "replay"
	case Conflict:
		return "conflict"
	case InProgress:
		return "in_progress"
	default:
		return "invalid"
	}
}

// Acquisition is the outcome of Acquire.
type Acquisition struct {
	State  State
	Result json.RawMessage // set for Replay
}

func table(tx db.Tx) string {
	return pgx.Identifier{tx.Module().Schema(), "idempotency_record"}.Sanitize()
}

// Acquire claims req in tx's module, or reports how an earlier use of the key resolved.
func Acquire(ctx context.Context, tx db.Tx, req Request, at timeutil.Instant) (Acquisition, error) {
	if tx.ReadOnly() {
		return Acquisition{}, ErrReadOnly
	}
	if err := req.Validate(); err != nil {
		return Acquisition{}, err
	}
	if at.IsZero() {
		return Acquisition{}, fmt.Errorf("%w: timestamp required", ErrInvalidRequest)
	}
	tag, err := tx.Exec(ctx, `INSERT INTO `+table(tx)+` (scope, idem_key, request_hash, state, created_at)
		VALUES ($1, $2, $3, 'in_progress', $4)
		ON CONFLICT (scope, idem_key) DO NOTHING`, req.Scope, req.Key, req.Hash[:], at.Time())
	if err != nil {
		return Acquisition{}, fmt.Errorf("idempotency: acquire: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return Acquisition{State: Acquired}, nil
	}
	var (
		storedHash []byte
		state      string
		result     []byte
	)
	if err := tx.QueryRow(ctx, `SELECT request_hash, state, result FROM `+table(tx)+`
		WHERE scope = $1 AND idem_key = $2`, req.Scope, req.Key).Scan(&storedHash, &state, &result); err != nil {
		return Acquisition{}, fmt.Errorf("idempotency: read existing: %w", err)
	}
	switch {
	case !bytes.Equal(storedHash, req.Hash[:]):
		return Acquisition{State: Conflict}, nil
	case state == "completed":
		return Acquisition{State: Replay, Result: result}, nil
	default:
		return Acquisition{State: InProgress}, nil
	}
}

// Complete stores the technical result of an acquired request in the same
// transaction that performed the command.
func Complete(ctx context.Context, tx db.Tx, req Request, result json.RawMessage, at timeutil.Instant) error {
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	if err := req.Validate(); err != nil {
		return err
	}
	if len(result) > MaxResultBytes {
		return ErrResultTooLarge
	}
	if len(result) > 0 && !json.Valid(result) {
		return fmt.Errorf("%w: result is not JSON", ErrInvalidRequest)
	}
	var stored any
	if len(result) > 0 {
		stored = []byte(result)
	}
	tag, err := tx.Exec(ctx, `UPDATE `+table(tx)+`
		SET state = 'completed', result = $4, completed_at = $5
		WHERE scope = $1 AND idem_key = $2 AND request_hash = $3 AND state = 'in_progress'`,
		req.Scope, req.Key, req.Hash[:], stored, at.Time())
	if err != nil {
		return fmt.Errorf("idempotency: complete: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotAcquired
	}
	return nil
}
