// Package rollout names the per-family write rollout mode of the controlled
// strangler (Architecture V1 §31 Slice 11). Legacy stays the master: a mode
// never claims the target is master. A mode is derived from two bounded
// feature flags per command family, both OFF by default:
//
//	<family>_write      OFF → TARGET_WRITE_DISABLED (legacy only)
//	<family>_write      ON, <family>_writeback OFF → SHADOW_COMPARE
//	<family>_write      ON, <family>_writeback ON  → CONTROLLED_WRITEBACK
//
// SHADOW_COMPARE evaluates a command (authority, scope, invariants, version)
// and records the evaluated intent without changing the aggregate or writing
// to legacy; CONTROLLED_WRITEBACK commits the aggregate and writes back to
// legacy through the integration outbox path. There is no automatic cutover.
package rollout

import (
	"context"
	"errors"
	"regexp"
)

// Mode is a command family's rollout mode.
type Mode string

// Modes.
const (
	Disabled   Mode = "TARGET_WRITE_DISABLED"
	Shadow     Mode = "SHADOW_COMPARE"
	Controlled Mode = "CONTROLLED_WRITEBACK"
)

// Valid reports a known mode.
func (m Mode) Valid() bool { return m == Disabled || m == Shadow || m == Controlled }

// Applies reports whether the mode commits the target aggregate.
func (m Mode) Applies() bool { return m == Controlled }

// Flag reads one boolean flag.
type Flag func(ctx context.Context, key string) (bool, error)

// Source resolves a family's mode.
type Source func(ctx context.Context) (Mode, error)

var familyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)

// ErrFamily reports a malformed family name.
var ErrFamily = errors.New("rollout: invalid family")

// WriteFlag and WritebackFlag are the two flag keys of a family.
func WriteFlag(family string) string     { return family + "_write" }
func WritebackFlag(family string) string { return family + "_writeback" }

// FromFlags returns the mode source of family. An absent or unreadable flag
// never enables anything: a read error is returned, not defaulted.
func FromFlags(family string, flag Flag) (Source, error) {
	if !familyPattern.MatchString(family) || flag == nil {
		return nil, ErrFamily
	}
	return func(ctx context.Context) (Mode, error) {
		on, err := flag(ctx, WriteFlag(family))
		if err != nil {
			return Disabled, err
		}
		if !on {
			return Disabled, nil
		}
		wb, err := flag(ctx, WritebackFlag(family))
		if err != nil {
			return Disabled, err
		}
		if wb {
			return Controlled, nil
		}
		return Shadow, nil
	}, nil
}
