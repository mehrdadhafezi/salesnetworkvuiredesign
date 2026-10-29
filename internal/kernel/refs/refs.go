// Package refs implements the frozen Slice 0 reference contracts (E0 §9B):
// NativeRef, SourceRef, CaseRef and LegacyRef.
//
// IDs are positive decimal strings, domain-qualified; equal IDs in different
// domains are unequal. A phone number, name or token is never a valid ID.
// A CanonicalID is never minted here: it can only be parsed from an existing
// value supplied by a caller. Nil, UNKNOWN, NOT_RECOVERABLE and NOT_APPLICABLE
// are distinct; zero is never a sentinel.
package refs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var (
	// ErrInvalidDomain reports an undeclared or malformed domain.
	ErrInvalidDomain = errors.New("refs: invalid domain")
	// ErrInvalidID reports an empty, zero, negative, non-decimal or overlong ID.
	ErrInvalidID = errors.New("refs: invalid id")
	// ErrPhoneShapedID reports an ID that has the shape of a phone number.
	ErrPhoneShapedID = errors.New("refs: phone-shaped value is never an id")
	// ErrInvalidProvenance reports a missing or unsupported provenance state.
	ErrInvalidProvenance = errors.New("refs: invalid provenance")
	// ErrInvalidResolution reports a missing or unsupported resolution state.
	ErrInvalidResolution = errors.New("refs: invalid resolution")
	// ErrResolvedWithoutCanonical reports RESOLVED without an existing canonical id.
	ErrResolvedWithoutCanonical = errors.New("refs: RESOLVED requires an existing canonical id")
	// ErrEmptyNatives reports a CaseRef without native refs.
	ErrEmptyNatives = errors.New("refs: CaseRef requires at least one native ref")
	// ErrDuplicateNative reports a native ref listed twice in one CaseRef.
	ErrDuplicateNative = errors.New("refs: duplicate native ref")
	// ErrInvalidCanonicalID reports a malformed canonical id.
	ErrInvalidCanonicalID = errors.New("refs: invalid canonical id")
	// ErrInvalidLegacySystem reports an unsupported legacy system.
	ErrInvalidLegacySystem = errors.New("refs: invalid legacy system")
	// ErrInvalidLegacyEntity reports a malformed legacy entity name.
	ErrInvalidLegacyEntity = errors.New("refs: invalid legacy entity")
)

// Domain qualifies a native identifier. Only declared domains are valid.
type Domain string

// Declared domains (Phase-0 spec §7). Extending this set is a kernel change.
const (
	DomainLegacyLead     Domain = "LEGACY_LEAD"
	DomainV4Distribution Domain = "V4_DISTRIBUTION"
	DomainMISRow         Domain = "MIS_ROW"
	DomainMISPool        Domain = "MIS_POOL"
	DomainDotCase        Domain = "DOT_CASE"
	DomainInvoice        Domain = "INVOICE"
	DomainPaymentStage   Domain = "PAYMENT_STAGE"
	DomainPayment        Domain = "PAYMENT"
	DomainLedgerTx       Domain = "LEDGER_TX"
	DomainEvent          Domain = "EVENT"
)

var declaredDomains = map[Domain]bool{
	DomainLegacyLead: true, DomainV4Distribution: true, DomainMISRow: true, DomainMISPool: true,
	DomainDotCase: true, DomainInvoice: true, DomainPaymentStage: true, DomainPayment: true,
	DomainLedgerTx: true, DomainEvent: true,
}

// Domains returns the declared domains in a stable order.
func Domains() []Domain {
	return []Domain{
		DomainLegacyLead, DomainV4Distribution, DomainMISRow, DomainMISPool, DomainDotCase,
		DomainInvoice, DomainPaymentStage, DomainPayment, DomainLedgerTx, DomainEvent,
	}
}

// Valid reports whether d is declared.
func (d Domain) Valid() bool { return declaredDomains[d] }

// maxIDDigits bounds IDs to the uint64 range.
const maxIDDigits = 20

// ValidateID checks that id is a positive decimal integer that fits uint64
// and does not have the shape of a phone number.
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: empty", ErrInvalidID)
	}
	if len(id) > maxIDDigits {
		return fmt.Errorf("%w: too long", ErrInvalidID)
	}
	if looksLikePhone(id) {
		return ErrPhoneShapedID
	}
	for i := 0; i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return fmt.Errorf("%w: %q is not decimal", ErrInvalidID, id)
		}
	}
	if id[0] == '0' {
		return fmt.Errorf("%w: zero or leading zero", ErrInvalidID)
	}
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return fmt.Errorf("%w: out of range", ErrInvalidID)
	}
	return nil
}

// looksLikePhone reports whether s has the shape of an Iranian mobile number
// in any common notation: 09xxxxxxxxx, 9xxxxxxxxx, 989xxxxxxxxx,
// +989xxxxxxxxx or 00989xxxxxxxxx.
//
// Consequence: a genuine 10-digit native ID starting with 9 is rejected.
// That is deliberate; phone-shaped values must never become identifiers.
func looksLikePhone(s string) bool {
	digits := s
	if len(digits) > 0 && digits[0] == '+' {
		digits = digits[1:]
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
	}
	switch {
	case len(digits) == 11 && digits[:2] == "09":
		return true
	case len(digits) == 10 && digits[0] == '9':
		return true
	case len(digits) == 12 && digits[:3] == "989":
		return true
	case len(digits) == 14 && digits[:5] == "00989":
		return true
	}
	return false
}

// NativeRef is a domain-qualified native identifier.
type NativeRef struct {
	Domain Domain
	ID     string
}

// NewNativeRef validates and returns a NativeRef.
func NewNativeRef(domain Domain, id string) (NativeRef, error) {
	r := NativeRef{Domain: domain, ID: id}
	if err := r.Validate(); err != nil {
		return NativeRef{}, err
	}
	return r, nil
}

// Validate checks the domain and the ID.
func (r NativeRef) Validate() error {
	if !r.Domain.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidDomain, string(r.Domain))
	}
	return ValidateID(r.ID)
}

// Equal reports domain and ID equality. Equal IDs in different domains are unequal.
func (r NativeRef) Equal(o NativeRef) bool { return r.Domain == o.Domain && r.ID == o.ID }

// Key returns the deterministic "DOMAIN:ID" key used for sorting and tests.
// It is not a storage or identity encoding commitment.
func (r NativeRef) Key() string { return string(r.Domain) + ":" + r.ID }

type wireNative struct {
	Domain string `json:"domain"`
	ID     string `json:"id"`
}

// MarshalJSON renders {"domain":..,"id":".."} after validation.
func (r NativeRef) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(wireNative{Domain: string(r.Domain), ID: r.ID})
}

// UnmarshalJSON parses and validates a NativeRef.
func (r *NativeRef) UnmarshalJSON(b []byte) error {
	var w wireNative
	if err := json.Unmarshal(b, &w); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidID, err)
	}
	parsed, err := NewNativeRef(Domain(w.Domain), w.ID)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
