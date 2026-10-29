package refs

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Resolution is the identity resolution state of a CaseRef.
type Resolution uint8

const (
	resolutionInvalid Resolution = iota
	// ResolutionResolved: aliases are proven and an existing canonical id is supplied.
	ResolutionResolved
	// ResolutionRelated: a proven relationship or grouping without an existing canonical id
	// (or explicitly related-but-distinct).
	ResolutionRelated
	// ResolutionUnknown: evidence is insufficient.
	ResolutionUnknown
	// ResolutionConflict: evidence is contradictory; dependent operations fail closed.
	ResolutionConflict
)

var resolutionNames = map[Resolution]string{
	ResolutionResolved: "RESOLVED",
	ResolutionRelated:  "RELATED",
	ResolutionUnknown:  "UNKNOWN",
	ResolutionConflict: "CONFLICT",
}

// Resolutions returns every valid resolution in a stable order.
func Resolutions() []Resolution {
	return []Resolution{ResolutionResolved, ResolutionRelated, ResolutionUnknown, ResolutionConflict}
}

// Valid reports whether r is a declared state.
func (r Resolution) Valid() bool {
	_, ok := resolutionNames[r]
	return ok
}

// String returns the stable name, or "INVALID".
func (r Resolution) String() string {
	if name, ok := resolutionNames[r]; ok {
		return name
	}
	return "INVALID"
}

// ParseResolution parses an exact state name.
func ParseResolution(s string) (Resolution, error) {
	for r, name := range resolutionNames {
		if name == s {
			return r, nil
		}
	}
	return resolutionInvalid, fmt.Errorf("%w: %q", ErrInvalidResolution, s)
}

// MarshalText implements encoding.TextMarshaler.
func (r Resolution) MarshalText() ([]byte, error) {
	if !r.Valid() {
		return nil, ErrInvalidResolution
	}
	return []byte(r.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (r *Resolution) UnmarshalText(b []byte) error {
	parsed, err := ParseResolution(string(b))
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

// CanonicalID is an existing, opaque canonical identifier. This package never
// generates one; ParseCanonicalID only accepts a value that already exists.
type CanonicalID struct{ value string }

const maxCanonicalIDLen = 128

// ParseCanonicalID accepts an existing canonical id: 1..128 visible ASCII
// characters without spaces, and never phone-shaped.
func ParseCanonicalID(s string) (CanonicalID, error) {
	if s == "" || len(s) > maxCanonicalIDLen {
		return CanonicalID{}, fmt.Errorf("%w: length", ErrInvalidCanonicalID)
	}
	for i := 0; i < len(s); i++ {
		if s[i] <= ' ' || s[i] > '~' {
			return CanonicalID{}, fmt.Errorf("%w: invalid character", ErrInvalidCanonicalID)
		}
	}
	if looksLikePhone(s) {
		return CanonicalID{}, ErrPhoneShapedID
	}
	return CanonicalID{value: s}, nil
}

// String returns the opaque value.
func (c CanonicalID) String() string { return c.value }

// IsZero reports whether c holds no value.
func (c CanonicalID) IsZero() bool { return c.value == "" }

// MarshalText implements encoding.TextMarshaler.
func (c CanonicalID) MarshalText() ([]byte, error) {
	if c.IsZero() {
		return nil, ErrInvalidCanonicalID
	}
	return []byte(c.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *CanonicalID) UnmarshalText(b []byte) error {
	parsed, err := ParseCanonicalID(string(b))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// CaseRef groups the native refs of one logical case and states how far the
// grouping is resolved. RESOLVED requires an existing CanonicalID; a proven
// grouping without one is RELATED, never RESOLVED.
type CaseRef struct {
	Natives     []NativeRef
	Sources     []SourceRef
	CanonicalID *CanonicalID
	Resolution  Resolution
}

// Validate checks the CaseRef invariants.
func (c CaseRef) Validate() error {
	if len(c.Natives) == 0 {
		return ErrEmptyNatives
	}
	seen := make(map[string]bool, len(c.Natives))
	for _, n := range c.Natives {
		if err := n.Validate(); err != nil {
			return err
		}
		if seen[n.Key()] {
			return fmt.Errorf("%w: %s", ErrDuplicateNative, n.Key())
		}
		seen[n.Key()] = true
	}
	for _, s := range c.Sources {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	if !c.Resolution.Valid() {
		return ErrInvalidResolution
	}
	if c.CanonicalID != nil && c.CanonicalID.IsZero() {
		return ErrInvalidCanonicalID
	}
	if c.Resolution == ResolutionResolved && c.CanonicalID == nil {
		return ErrResolvedWithoutCanonical
	}
	return nil
}

// NewCaseRef validates and returns a CaseRef with natives in deterministic order.
func NewCaseRef(natives []NativeRef, sources []SourceRef, canonical *CanonicalID, resolution Resolution) (CaseRef, error) {
	c := CaseRef{
		Natives:     append([]NativeRef(nil), natives...),
		Sources:     append([]SourceRef(nil), sources...),
		CanonicalID: canonical,
		Resolution:  resolution,
	}
	if err := c.Validate(); err != nil {
		return CaseRef{}, err
	}
	sort.Slice(c.Natives, func(i, j int) bool { return c.Natives[i].Key() < c.Natives[j].Key() })
	return c, nil
}

type wireCase struct {
	Natives     []NativeRef  `json:"natives"`
	Sources     []SourceRef  `json:"sources,omitempty"`
	CanonicalID *CanonicalID `json:"canonical_id"`
	Resolution  Resolution   `json:"resolution"`
}

// MarshalJSON renders a validated CaseRef. An absent canonical id is null.
func (c CaseRef) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(wireCase(c))
}

// UnmarshalJSON parses and validates a CaseRef.
func (c *CaseRef) UnmarshalJSON(b []byte) error {
	var w wireCase
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	out := CaseRef(w)
	if err := out.Validate(); err != nil {
		return err
	}
	*c = out
	return nil
}
