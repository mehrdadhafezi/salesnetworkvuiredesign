package refs

import (
	"encoding/json"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
)

// Provenance qualifies how much of a source's origin is known.
// The zero value is invalid; every state is distinct and none collapses into another.
type Provenance uint8

const (
	provenanceInvalid Provenance = iota
	// ProvenanceVerified: origin is backed by evidence.
	ProvenanceVerified
	// ProvenanceIncomplete: some origin facts are present, others missing.
	ProvenanceIncomplete
	// ProvenanceUnknown: origin is not known yet and may still be reconciled.
	ProvenanceUnknown
	// ProvenanceNotRecoverable: origin is known to be permanently unavailable.
	ProvenanceNotRecoverable
	// ProvenanceNotApplicable: the origin field does not apply to this source.
	ProvenanceNotApplicable
)

var provenanceNames = map[Provenance]string{
	ProvenanceVerified:       "VERIFIED",
	ProvenanceIncomplete:     "INCOMPLETE",
	ProvenanceUnknown:        "UNKNOWN",
	ProvenanceNotRecoverable: "NOT_RECOVERABLE",
	ProvenanceNotApplicable:  "NOT_APPLICABLE",
}

// Provenances returns every valid provenance state in a stable order.
func Provenances() []Provenance {
	return []Provenance{ProvenanceVerified, ProvenanceIncomplete, ProvenanceUnknown, ProvenanceNotRecoverable, ProvenanceNotApplicable}
}

// Valid reports whether p is a declared state.
func (p Provenance) Valid() bool { _, ok := provenanceNames[p]; return ok }

// String returns the stable name, or "INVALID".
func (p Provenance) String() string {
	if name, ok := provenanceNames[p]; ok {
		return name
	}
	return "INVALID"
}

// ParseProvenance parses an exact state name.
func ParseProvenance(s string) (Provenance, error) {
	for p, name := range provenanceNames {
		if name == s {
			return p, nil
		}
	}
	return provenanceInvalid, fmt.Errorf("%w: %q", ErrInvalidProvenance, s)
}

// MarshalText implements encoding.TextMarshaler.
func (p Provenance) MarshalText() ([]byte, error) {
	if !p.Valid() {
		return nil, ErrInvalidProvenance
	}
	return []byte(p.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (p *Provenance) UnmarshalText(b []byte) error {
	parsed, err := ParseProvenance(string(b))
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// SourceRef records where a native object came from. Origin fields hold only
// references that actually exist; missing history is expressed by Provenance,
// never by today's owner or time.
type SourceRef struct {
	Native      NativeRef
	OriginBatch *NativeRef
	OriginActor *NativeRef
	OriginAt    *timeutil.SourceTime
	Provenance  Provenance
}

// Validate checks the native ref, optional origin refs and provenance.
func (s SourceRef) Validate() error {
	if err := s.Native.Validate(); err != nil {
		return err
	}
	for _, r := range []*NativeRef{s.OriginBatch, s.OriginActor} {
		if r != nil {
			if err := r.Validate(); err != nil {
				return err
			}
		}
	}
	if s.OriginAt != nil {
		if err := s.OriginAt.Validate(); err != nil {
			return err
		}
	}
	if !s.Provenance.Valid() {
		return ErrInvalidProvenance
	}
	return nil
}

type wireSourceTime struct {
	UTC        timeutil.Instant `json:"utc"`
	SourceZone string           `json:"source_zone"`
	RawText    *string          `json:"raw_text,omitempty"`
}

type wireSource struct {
	Native      NativeRef       `json:"native"`
	OriginBatch *NativeRef      `json:"origin_batch,omitempty"`
	OriginActor *NativeRef      `json:"origin_actor,omitempty"`
	OriginAt    *wireSourceTime `json:"origin_at,omitempty"`
	Provenance  Provenance      `json:"provenance"`
}

// MarshalJSON renders a validated SourceRef.
func (s SourceRef) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	w := wireSource{Native: s.Native, OriginBatch: s.OriginBatch, OriginActor: s.OriginActor, Provenance: s.Provenance}
	if s.OriginAt != nil {
		w.OriginAt = &wireSourceTime{UTC: s.OriginAt.UTC, SourceZone: s.OriginAt.SourceZone, RawText: s.OriginAt.RawText}
	}
	return json.Marshal(w)
}

// UnmarshalJSON parses and validates a SourceRef.
func (s *SourceRef) UnmarshalJSON(b []byte) error {
	var w wireSource
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	out := SourceRef{Native: w.Native, OriginBatch: w.OriginBatch, OriginActor: w.OriginActor, Provenance: w.Provenance}
	if w.OriginAt != nil {
		out.OriginAt = &timeutil.SourceTime{UTC: w.OriginAt.UTC, SourceZone: w.OriginAt.SourceZone, RawText: w.OriginAt.RawText}
	}
	if err := out.Validate(); err != nil {
		return err
	}
	*s = out
	return nil
}
