// Package domain holds biawin's pure rules for Slice 10 (READ / SHADOW).
package domain

import (
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
)

// NormalizationVersion is the shadow normalization version. Changing what
// is normalized requires a new version; runs of different versions coexist.
const NormalizationVersion = 1

// Parity row results.
const (
	Matched        = "MATCHED"
	MissingTarget  = "MISSING_TARGET"
	ExtraTarget    = "EXTRA_TARGET"
	Unknown        = "UNKNOWN"
	Conflict       = "CONFLICT"
	NotRecoverable = "NOT_RECOVERABLE"
	InactiveBoth   = "INACTIVE_BOTH"
)

// Parity outcomes.
const (
	OutcomeMatch      = "MATCH"
	OutcomeMismatch   = "MISMATCH"
	OutcomeUnknown    = "UNKNOWN"
	OutcomeIncomplete = "INCOMPLETE"
)

// ParityRow is one compared source key.
type ParityRow struct {
	SourceKey   string
	LegacyClass *string
	TargetState *string
	Result      string
}

// Parity is the active subscription parity of one legacy evidence against
// one target run. It is never a single count: every bucket is explicit.
type Parity struct {
	LegacyCount, TargetCount                                       int64
	Matched, MissingTarget, ExtraTarget, Unknown, Conflict, NotRec int64
	Outcome                                                        string
	Reasons                                                        []reasoncodes.Code
	Rows                                                           []ParityRow
}

// Compare compares the legacy active classification (by stable source
// key) with the target run's mapped states. ACTIVE is only ever the
// evidenced legacy raw status on one side and the evidence-backed mapped
// state on the other — never payment, invoice or item presence.
//
//	legacy ACTIVE  & target ACTIVE            → MATCHED
//	legacy ACTIVE  & no target record         → MISSING_TARGET
//	no legacy record & target ACTIVE          → EXTRA_TARGET (unsupported by source)
//	either side UNKNOWN / EMPTY               → UNKNOWN (never ACTIVE)
//	legacy NOT_RECOVERABLE                    → NOT_RECOVERABLE
//	ACTIVE on one side, a known other state   → CONFLICT
//	not active on either side                 → INACTIVE_BOTH (not counted)
func Compare(legacy map[string]string, target map[string]string, evidenceStatus string) Parity {
	keys := map[string]bool{}
	for k := range legacy {
		keys[k] = true
	}
	for k := range target {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	var p Parity
	for _, k := range sorted {
		l, inL := legacy[k]
		t, inT := target[k]
		row := ParityRow{SourceKey: k}
		if inL {
			v := l
			row.LegacyClass = &v
		}
		if inT {
			v := t
			row.TargetState = &v
		}
		if inL && l == api.LegacyActive {
			p.LegacyCount++
		}
		if inT && t == api.StateActive {
			p.TargetCount++
		}
		lActive, tActive := inL && l == api.LegacyActive, inT && t == api.StateActive
		lUnknown := inL && l == api.LegacyUnknown
		tUnknown := inT && (t == api.StateUnknown || t == api.StateEmpty)
		switch {
		case inL && l == api.LegacyNotRecoverable:
			row.Result = NotRecoverable
			p.NotRec++
		case lActive && tActive:
			row.Result = Matched
			p.Matched++
		case lUnknown || tUnknown:
			row.Result = Unknown
			p.Unknown++
		case lActive && !inT:
			row.Result = MissingTarget
			p.MissingTarget++
		case tActive && !inL:
			row.Result = ExtraTarget
			p.ExtraTarget++
		case lActive || tActive:
			row.Result = Conflict
			p.Conflict++
		default:
			row.Result = InactiveBoth
		}
		p.Rows = append(p.Rows, row)
	}
	switch {
	case evidenceStatus != api.SourceAvailable:
		p.Outcome = OutcomeIncomplete
		p.Reasons = append(p.Reasons, reasoncodes.CoverageIncomplete)
	case p.MissingTarget > 0 || p.ExtraTarget > 0 || p.Conflict > 0:
		p.Outcome = OutcomeMismatch
		p.Reasons = append(p.Reasons, reasoncodes.LinkageConflict)
	case p.Unknown > 0 || p.NotRec > 0:
		p.Outcome = OutcomeUnknown
		p.Reasons = append(p.Reasons, reasoncodes.OutcomeUnknown)
	default:
		p.Outcome = OutcomeMatch
	}
	if p.Unknown > 0 && p.Outcome != OutcomeUnknown {
		p.Reasons = append(p.Reasons, reasoncodes.OutcomeUnknown)
	}
	p.Reasons, _ = reasoncodes.Normalize(p.Reasons)
	if p.Reasons == nil {
		p.Reasons = []reasoncodes.Code{}
	}
	return p
}
