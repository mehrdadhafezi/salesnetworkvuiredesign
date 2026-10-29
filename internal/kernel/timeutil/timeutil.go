// Package timeutil implements the frozen Slice 0 time contract (E0 §9E).
//
// Canonical storage is a UTC instant. SourceTime preserves source evidence:
// the UTC instant, the IANA source zone and, optionally, the raw source text.
// Parsing local wall-clock text requires an explicit zone; the machine-local
// zone is never used. Jalali is presentation and evidence only (jalali.go).
// The clock is injected; this package never reads the system clock.
package timeutil

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidTime reports unparseable or invalid time input.
	ErrInvalidTime = errors.New("timeutil: invalid time")
	// ErrZoneRequired reports a missing, nil or machine-local zone.
	ErrZoneRequired = errors.New("timeutil: explicit zone required")
	// ErrNonexistentLocalTime reports wall-clock text that falls in a zone gap.
	ErrNonexistentLocalTime = errors.New("timeutil: local time does not exist in zone")
	// ErrAmbiguousLocalTime reports wall-clock text that maps to more than one instant.
	ErrAmbiguousLocalTime = errors.New("timeutil: local time is ambiguous in zone")
)

// Clock is the injected source of the current instant.
type Clock interface {
	Now() Instant
}

// FixedClock is a deterministic Clock that always returns the same instant.
type FixedClock struct{ At Instant }

// Now returns the fixed instant.
func (c FixedClock) Now() Instant { return c.At }

// Instant is a canonical UTC instant without a monotonic clock reading.
// The zero value is invalid.
type Instant struct{ t time.Time }

// NewInstant normalizes t to UTC. The zero time.Time is rejected.
func NewInstant(t time.Time) (Instant, error) {
	if t.IsZero() {
		return Instant{}, fmt.Errorf("%w: zero time", ErrInvalidTime)
	}
	return Instant{t: t.Round(0).UTC()}, nil
}

// Time returns the UTC time.Time.
func (i Instant) Time() time.Time { return i.t }

// IsZero reports whether i is the invalid zero Instant.
func (i Instant) IsZero() bool { return i.t.IsZero() }

// Equal reports whether two instants are the same moment.
func (i Instant) Equal(o Instant) bool { return i.t.Equal(o.t) }

// String renders RFC 3339 with nanoseconds in UTC.
func (i Instant) String() string { return i.t.Format(time.RFC3339Nano) }

// MarshalText renders RFC 3339 UTC.
func (i Instant) MarshalText() ([]byte, error) {
	if i.IsZero() {
		return nil, fmt.Errorf("%w: zero instant", ErrInvalidTime)
	}
	return []byte(i.String()), nil
}

// UnmarshalText parses RFC 3339 text that carries an explicit offset or Z.
func (i *Instant) UnmarshalText(b []byte) error {
	parsed, err := ParseRFC3339(string(b))
	if err != nil {
		return err
	}
	*i = parsed
	return nil
}

// ParseRFC3339 parses text with an explicit offset ("Z" or "±hh:mm") and
// returns the UTC instant. Text without an offset is rejected.
func ParseRFC3339(s string) (Instant, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Instant{}, fmt.Errorf("%w: %v", ErrInvalidTime, err)
	}
	return NewInstant(t)
}

// SourceTime is canonical UTC plus the source evidence it was derived from.
type SourceTime struct {
	UTC        Instant
	SourceZone string  // IANA zone name, e.g. "Asia/Tehran"; never "Local"
	RawText    *string // verbatim source text, when the source provided text
}

// Validate checks the SourceTime invariants.
func (s SourceTime) Validate() error {
	if s.UTC.IsZero() {
		return fmt.Errorf("%w: zero instant", ErrInvalidTime)
	}
	if s.SourceZone == "" || s.SourceZone == "Local" {
		return ErrZoneRequired
	}
	return nil
}

// InZone returns the instant rendered in the source zone, for presentation.
func (s SourceTime) InZone(loc *time.Location) (time.Time, error) {
	if err := checkZone(loc); err != nil {
		return time.Time{}, err
	}
	if loc.String() != s.SourceZone {
		return time.Time{}, fmt.Errorf("%w: zone %q does not match source zone %q", ErrZoneRequired, loc.String(), s.SourceZone)
	}
	return s.UTC.t.In(loc), nil
}

func checkZone(loc *time.Location) error {
	if loc == nil || loc == time.Local || loc.String() == "Local" || loc.String() == "" {
		return ErrZoneRequired
	}
	return nil
}

// ParseLocal parses wall-clock text that has no offset, interpreting it in the
// explicit zone loc. The raw text is preserved verbatim. The machine-local zone
// is rejected. Text that falls into a zone gap or overlap is rejected rather
// than silently shifted.
func ParseLocal(layout, raw string, loc *time.Location) (SourceTime, error) {
	if err := checkZone(loc); err != nil {
		return SourceTime{}, err
	}
	if layoutHasZone(layout) {
		return SourceTime{}, fmt.Errorf("%w: layout carries a zone or offset; use ParseRFC3339", ErrInvalidTime)
	}
	wall, err := time.Parse(layout, raw)
	if err != nil {
		return SourceTime{}, fmt.Errorf("%w: %v", ErrInvalidTime, err)
	}
	t := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), loc)
	if !sameWallClock(t, wall) {
		return SourceTime{}, fmt.Errorf("%w: %q in %s", ErrNonexistentLocalTime, raw, loc)
	}
	if ambiguous(t, wall, loc) {
		return SourceTime{}, fmt.Errorf("%w: %q in %s", ErrAmbiguousLocalTime, raw, loc)
	}
	instant, err := NewInstant(t)
	if err != nil {
		return SourceTime{}, err
	}
	text := raw
	return SourceTime{UTC: instant, SourceZone: loc.String(), RawText: &text}, nil
}

// zoneTokens are the reference-layout elements that carry an offset or zone.
var zoneTokens = []string{"Z07", "-07", "MST"}

func layoutHasZone(layout string) bool {
	for _, token := range zoneTokens {
		if strings.Contains(layout, token) {
			return true
		}
	}
	return false
}

func sameWallClock(t, wall time.Time) bool {
	return t.Year() == wall.Year() && t.Month() == wall.Month() && t.Day() == wall.Day() &&
		t.Hour() == wall.Hour() && t.Minute() == wall.Minute() && t.Second() == wall.Second() &&
		t.Nanosecond() == wall.Nanosecond()
}

// ambiguousShifts covers the zone offset changes that occur in practice.
var ambiguousShifts = []time.Duration{30 * time.Minute, time.Hour, 2 * time.Hour}

func ambiguous(t, wall time.Time, loc *time.Location) bool {
	for _, d := range ambiguousShifts {
		for _, candidate := range []time.Time{t.Add(d), t.Add(-d)} {
			if sameWallClock(candidate.In(loc), wall) {
				return true
			}
		}
	}
	return false
}

// FromInstant builds SourceTime evidence for an instant observed in loc
// without raw text (e.g. a native timestamp with a declared zone).
func FromInstant(i Instant, loc *time.Location) (SourceTime, error) {
	if err := checkZone(loc); err != nil {
		return SourceTime{}, err
	}
	if i.IsZero() {
		return SourceTime{}, fmt.Errorf("%w: zero instant", ErrInvalidTime)
	}
	return SourceTime{UTC: i, SourceZone: loc.String()}, nil
}
