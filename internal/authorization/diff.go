package authorization

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"

// Capability diff (Slice 4 acceptance "capability diff ≈ legacy"): a
// comparison report, not a parity claim. The legacy side is computed by the
// legacy connector from staged evidence; the target side by Evaluate in
// Shadow mode. Classification never broadens target permissions.

// LegacyVerdict is what staged legacy evidence says about one capability.
type LegacyVerdict string

const (
	LegacyAllowed  LegacyVerdict = "ALLOWED"
	LegacyDenied   LegacyVerdict = "DENIED"
	LegacyUnknown  LegacyVerdict = "UNKNOWN"
	LegacyConflict LegacyVerdict = "CONFLICT"
)

// TargetVerdict is what the target evaluator says.
type TargetVerdict string

const (
	TargetAllowed  TargetVerdict = "ALLOWED"
	TargetDenied   TargetVerdict = "DENIED"
	TargetUnknown  TargetVerdict = "UNKNOWN"
	TargetConflict TargetVerdict = "CONFLICT"
)

// DiffClass is the comparison classification.
type DiffClass string

const (
	DiffMatch                    DiffClass = "MATCH"
	DiffTargetDeniesLegacyAllows DiffClass = "TARGET_DENIES_LEGACY_ALLOWED"
	DiffTargetAllowsLegacyDenies DiffClass = "TARGET_ALLOWS_LEGACY_DENIED"
	DiffUnknownLegacy            DiffClass = "UNKNOWN_LEGACY"
	DiffUnknownTarget            DiffClass = "UNKNOWN_TARGET"
	DiffUnmappedRole             DiffClass = "UNMAPPED_ROLE"
	DiffConflict                 DiffClass = "CONFLICT"
)

// DiffClasses lists every class in report order.
func DiffClasses() []DiffClass {
	return []DiffClass{DiffMatch, DiffTargetDeniesLegacyAllows, DiffTargetAllowsLegacyDenies, DiffUnknownLegacy,
		DiffUnknownTarget, DiffUnmappedRole, DiffConflict}
}

// TargetVerdictOf maps a shadow decision to a verdict.
func TargetVerdictOf(d Decision) TargetVerdict {
	switch {
	case d.Allowed:
		return TargetAllowed
	case d.Reason != nil && *d.Reason == reasoncodes.LinkageConflict:
		return TargetConflict
	case d.Reason != nil && *d.Reason == reasoncodes.PolicyUnresolved:
		return TargetUnknown
	}
	return TargetDenied
}

// Classify compares one (subject, capability) pair. hasUnmappedRole means
// the subject carries legacy role evidence that maps to no target role.
//
// SecuritySensitive is true whenever the target would allow something the
// legacy evidence does not prove allowed (legacy DENIED or UNKNOWN): such
// rows are surfaced prominently and are never "fixed" by broadening.
func Classify(legacy LegacyVerdict, target TargetVerdict, hasUnmappedRole bool) (class DiffClass, securitySensitive bool) {
	securitySensitive = target == TargetAllowed && legacy != LegacyAllowed
	switch {
	case legacy == LegacyConflict || target == TargetConflict:
		return DiffConflict, securitySensitive
	case target == TargetUnknown:
		return DiffUnknownTarget, securitySensitive
	case target == TargetDenied && hasUnmappedRole && legacy != LegacyDenied:
		return DiffUnmappedRole, securitySensitive
	case legacy == LegacyUnknown:
		return DiffUnknownLegacy, securitySensitive
	case legacy == LegacyAllowed && target == TargetAllowed, legacy == LegacyDenied && target == TargetDenied:
		return DiffMatch, securitySensitive
	case legacy == LegacyAllowed && target == TargetDenied:
		return DiffTargetDeniesLegacyAllows, securitySensitive
	case legacy == LegacyDenied && target == TargetAllowed:
		return DiffTargetAllowsLegacyDenies, securitySensitive
	}
	return DiffUnknownLegacy, securitySensitive
}
