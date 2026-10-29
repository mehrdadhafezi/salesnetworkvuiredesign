// Package featureflags is the boolean feature-flag foundation for strangler
// rollout (E0 §6 #17). Flags are default-off: an absent flag is disabled.
// There is no targeting, percentage rollout or segmentation.
package featureflags

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

var (
	// ErrInvalidFlag reports a malformed flag.
	ErrInvalidFlag = errors.New("featureflags: invalid flag")
	// ErrWrongOwner reports use from a transaction not owned by the flag store.
	ErrWrongOwner = errors.New("featureflags: transaction is not owned by platform_featureflags")
	// ErrReadOnly reports a write inside a read-only transaction.
	ErrReadOnly = errors.New("featureflags: read-only transaction")
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)

const maxDescriptionLen = 500

// Flag is one boolean flag.
type Flag struct {
	Key         string
	Enabled     bool
	Description *string
	UpdatedAt   timeutil.Instant
}

// ValidateKey checks the stable flag key format.
func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("%w: key %q", ErrInvalidFlag, key)
	}
	return nil
}

func checkOwner(tx db.Tx) error {
	if tx.Module() != owner.FeatureFlags {
		return ErrWrongOwner
	}
	return nil
}

// Set creates or updates a flag.
func Set(ctx context.Context, tx db.Tx, f Flag) error {
	if err := checkOwner(tx); err != nil {
		return err
	}
	if tx.ReadOnly() {
		return ErrReadOnly
	}
	if err := ValidateKey(f.Key); err != nil {
		return err
	}
	if f.Description != nil && (len(*f.Description) == 0 || len(*f.Description) > maxDescriptionLen) {
		return fmt.Errorf("%w: description length", ErrInvalidFlag)
	}
	if f.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: updated_at required", ErrInvalidFlag)
	}
	_, err := tx.Exec(ctx, `INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, description, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (flag_key) DO UPDATE
			SET enabled = EXCLUDED.enabled, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at`,
		f.Key, f.Enabled, f.Description, f.UpdatedAt.Time())
	if err != nil {
		return fmt.Errorf("featureflags: set: %w", err)
	}
	return nil
}

// Get returns the stored flag, if present.
func Get(ctx context.Context, tx db.Tx, key string) (Flag, bool, error) {
	if err := checkOwner(tx); err != nil {
		return Flag{}, false, err
	}
	if err := ValidateKey(key); err != nil {
		return Flag{}, false, err
	}
	f := Flag{Key: key}
	var updated time.Time
	err := tx.QueryRow(ctx, `SELECT enabled, description, updated_at FROM platform_featureflags.feature_flag WHERE flag_key = $1`, key).
		Scan(&f.Enabled, &f.Description, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return Flag{}, false, nil
	}
	if err != nil {
		return Flag{}, false, fmt.Errorf("featureflags: get: %w", err)
	}
	if f.UpdatedAt, err = timeutil.NewInstant(updated); err != nil {
		return Flag{}, false, err
	}
	return f, true, nil
}

// Enabled reports whether key is enabled. An absent flag is disabled.
func Enabled(ctx context.Context, tx db.Tx, key string) (bool, error) {
	f, ok, err := Get(ctx, tx, key)
	if err != nil {
		return false, err
	}
	return ok && f.Enabled, nil
}
