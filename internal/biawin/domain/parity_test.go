package domain

import (
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/api"
)

func TestActiveParity(t *testing.T) {
	// Known fixture: legacy active set == target active set → MATCH.
	p := Compare(map[string]string{"1": api.LegacyActive, "2": api.LegacyActive, "3": api.LegacyKnownOther},
		map[string]string{"1": api.StateActive, "2": api.StateActive, "3": api.StateSuspended}, api.SourceAvailable)
	if p.Outcome != OutcomeMatch || p.Matched != 2 || p.LegacyCount != 2 || p.TargetCount != 2 {
		t.Fatalf("match: %+v", p)
	}
	// Legacy active absent from target → MISSING_TARGET with its source key.
	p = Compare(map[string]string{"1": api.LegacyActive, "2": api.LegacyActive}, map[string]string{"1": api.StateActive}, api.SourceAvailable)
	if p.Outcome != OutcomeMismatch || p.MissingTarget != 1 || p.Rows[1].SourceKey != "2" || p.Rows[1].Result != MissingTarget {
		t.Fatalf("missing target: %+v", p)
	}
	// Target active unsupported by the source → EXTRA_TARGET.
	p = Compare(map[string]string{"1": api.LegacyActive}, map[string]string{"1": api.StateActive, "9": api.StateActive}, api.SourceAvailable)
	if p.Outcome != OutcomeMismatch || p.ExtraTarget != 1 || p.TargetCount != 2 {
		t.Fatalf("extra target: %+v", p)
	}
	// Unknown legacy status is never ACTIVE and never a match.
	p = Compare(map[string]string{"1": api.LegacyUnknown}, map[string]string{"1": api.StateUnknown}, api.SourceAvailable)
	if p.Outcome != OutcomeUnknown || p.Unknown != 1 || p.LegacyCount != 0 || p.TargetCount != 0 {
		t.Fatalf("unknown: %+v", p)
	}
	// Active vs a known other state → CONFLICT.
	p = Compare(map[string]string{"1": api.LegacyActive}, map[string]string{"1": api.StateSuspended}, api.SourceAvailable)
	if p.Outcome != OutcomeMismatch || p.Conflict != 1 {
		t.Fatalf("conflict: %+v", p)
	}
	// Not recoverable and incomplete evidence never claim a MATCH.
	p = Compare(map[string]string{"1": api.LegacyActive, "2": api.LegacyNotRecoverable}, map[string]string{"1": api.StateActive}, api.SourceAvailable)
	if p.Outcome != OutcomeUnknown || p.NotRec != 1 {
		t.Fatalf("not recoverable: %+v", p)
	}
	p = Compare(map[string]string{"1": api.LegacyActive}, map[string]string{"1": api.StateActive}, api.SourcePartial)
	if p.Outcome != OutcomeIncomplete {
		t.Fatalf("partial evidence: %+v", p)
	}
	// Empty sets: a real zero, still explicit.
	p = Compare(map[string]string{}, map[string]string{}, api.SourceAvailable)
	if p.Outcome != OutcomeMatch || p.LegacyCount != 0 || len(p.Rows) != 0 {
		t.Fatalf("empty: %+v", p)
	}
}
