// Package reasoncodes implements the frozen foundation-v1 taxonomy (E0 §9C).
//
// It has two kinds of values:
//  1. CRM reason codes (Phase-0 spec §6): stable diagnostic codes.
//  2. LegacyValue: a raw legacy status preserved verbatim, classified against
//     the namespace-qualified known-legacy registry (registry.go). Unknown raw
//     values classify as UNKNOWN with the raw bytes preserved; they are never
//     coerced to a default.
package reasoncodes

import (
	"errors"
	"fmt"
	"sort"
)

// Version is the taxonomy version.
const Version = "foundation-v1"

// ErrUnknownCode reports a string that is not a foundation-v1 CRM reason code.
var ErrUnknownCode = errors.New("reasoncodes: unknown reason code")

// Code is a CRM reason code.
type Code string

// foundation-v1 CRM reason codes (Phase-0 spec §6).
const (
	DependencyFinancialActive     Code = "DEPENDENCY_FINANCIAL_ACTIVE"
	DependencyProtectedDownstream Code = "DEPENDENCY_PROTECTED_DOWNSTREAM"
	DependencyUnknown             Code = "DEPENDENCY_UNKNOWN"
	LinkageConflict               Code = "LINKAGE_CONFLICT"
	StaleState                    Code = "STALE_STATE"
	RowScopeDenied                Code = "ROW_SCOPE_DENIED"
	ActionDenied                  Code = "ACTION_DENIED"
	FieldDenied                   Code = "FIELD_DENIED"
	RecipientIneligible           Code = "RECIPIENT_INELIGIBLE"
	OutcomeUnknown                Code = "OUTCOME_UNKNOWN"
	SourceUnavailable             Code = "SOURCE_UNAVAILABLE"
	CoverageIncomplete            Code = "COVERAGE_INCOMPLETE"
	PolicyUnresolved              Code = "POLICY_UNRESOLVED"
	FeatureDisabled               Code = "FEATURE_DISABLED"
	ContractInvalid               Code = "CONTRACT_INVALID"
)

var allCodes = []Code{
	DependencyFinancialActive, DependencyProtectedDownstream, DependencyUnknown, LinkageConflict,
	StaleState, RowScopeDenied, ActionDenied, FieldDenied, RecipientIneligible, OutcomeUnknown,
	SourceUnavailable, CoverageIncomplete, PolicyUnresolved, FeatureDisabled, ContractInvalid,
}

var knownCodes = func() map[Code]bool {
	m := make(map[Code]bool, len(allCodes))
	for _, c := range allCodes {
		m[c] = true
	}
	return m
}()

// Codes returns every foundation-v1 CRM reason code in declaration order.
func Codes() []Code { return append([]Code(nil), allCodes...) }

// Valid reports whether c is a foundation-v1 CRM reason code.
func (c Code) Valid() bool { return knownCodes[c] }

// ParseCode parses an exact code string.
func ParseCode(s string) (Code, error) {
	c := Code(s)
	if !c.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownCode, s)
	}
	return c, nil
}

// MarshalText implements encoding.TextMarshaler.
func (c Code) MarshalText() ([]byte, error) {
	if !c.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCode, string(c))
	}
	return []byte(c), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *Code) UnmarshalText(b []byte) error {
	parsed, err := ParseCode(string(b))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// Normalize validates, deduplicates and sorts codes deterministically.
// An empty input returns an empty, non-nil slice.
func Normalize(codes []Code) ([]Code, error) {
	seen := make(map[Code]bool, len(codes))
	out := make([]Code, 0, len(codes))
	for _, c := range codes {
		if !c.Valid() {
			return nil, fmt.Errorf("%w: %q", ErrUnknownCode, string(c))
		}
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}
