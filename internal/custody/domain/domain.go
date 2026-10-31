// Package domain is the custody module's pure, deterministic logic: deriving
// the current holder of a work item from its ordered append-only events and
// classifying parity against the legacy current-holder evidence.
//
// Nothing here synthesises an event, picks one of two contradictory holders
// or repairs history. Gaps and contradictions become findings; ambiguity
// becomes UNKNOWN / MULTIPLE; missing ownership is never the current owner.
package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

// Event kinds (assigned by the integration's explicit, evidence-backed
// mapping of each legacy action; anything else is UNKNOWN).
const (
	HolderAssigned = "HOLDER_ASSIGNED"
	HolderReleased = "HOLDER_RELEASED"
	NoHolderChange = "NO_HOLDER_CHANGE"
	KindUnknown    = "UNKNOWN"
)

// Derivation states.
const (
	Derived  = "DERIVED"  // holder established by events
	Released = "RELEASED" // released to the source (no holder)
	Unknown  = "UNKNOWN"  // events exist but the holder cannot be established
	Multiple = "MULTIPLE" // contradictory, equally-ordered holders
	None     = "NONE"     // no custody event at all
)

// Parity classes (legacy current-holder evidence vs derived holder).
const (
	Match           = "MATCH"
	TargetUnknown   = "TARGET_UNKNOWN"
	LegacyUnknown   = "LEGACY_UNKNOWN"
	Conflict        = "CONFLICT"
	MultipleCurrent = "MULTIPLE_CURRENT"
	HistoryGap      = "HISTORY_GAP"
	NotRecoverable  = "NOT_RECOVERABLE"
)

// Finding kinds.
const (
	FindMissingPriorOwner  = "MISSING_PRIOR_OWNER"
	FindMissingTargetOwner = "MISSING_TARGET_OWNER"
	FindHistoryGap         = "HISTORY_GAP"
	FindMultipleCurrent    = "MULTIPLE_CURRENT"
	FindOrdering           = "ORDERING_CONTRADICTION"
	FindMissingHistory     = "MISSING_EVENT_HISTORY"
	FindSourceMismatch     = "SOURCE_CURRENT_MISMATCH"
	FindUnknownAction      = "UNKNOWN_ACTION"
)

// Event is one custody event as derivation sees it.
type Event struct {
	ID          string
	Ordinal     string // legacy insertion ordinal (unsigned decimal)
	Kind        string
	Action      string
	From, To    *string // legacy holder keys
	ToPosition  *string
	OccurredRaw *string
}

// Finding is one detected problem (never auto-repaired).
type Finding struct {
	Kind    string
	Detail  string
	EventID string
}

// Result is the derivation of one work item.
type Result struct {
	State    string
	Holder   *string
	Position *string
	Findings []Finding
	LastID   string
}

var timePattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$`)

// readable returns a comparable legacy DATETIME or "" (NULL, zero date or
// unparseable values never order anything).
func readable(raw *string) string {
	if raw == nil || !timePattern.MatchString(*raw) || (*raw)[:4] == "0000" {
		return ""
	}
	return *raw
}

func ord(o string) uint64 {
	n, err := strconv.ParseUint(o, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func eq(a, b *string) bool { return a != nil && b != nil && *a == *b }

// Derive replays events in legacy insertion order (the authoritative order
// of an append-only log). It reports ordering contradictions between that
// order and the recorded times, and refuses to choose between holders
// assigned at the same recorded time by the final holder-defining events.
func Derive(events []Event) Result {
	ev := append([]Event(nil), events...)
	sort.SliceStable(ev, func(i, j int) bool { return ord(ev[i].Ordinal) < ord(ev[j].Ordinal) })
	r := Result{State: None}
	if len(ev) == 0 {
		r.Findings = append(r.Findings, Finding{Kind: FindMissingHistory, Detail: "no custody event recorded for this work item"})
		return r
	}
	add := func(kind, detail, id string) {
		r.Findings = append(r.Findings, Finding{Kind: kind, Detail: detail, EventID: id})
	}
	lastTime := ""
	var defining []int // indexes of holder-defining events
	for i, e := range ev {
		r.LastID = e.ID
		if t := readable(e.OccurredRaw); t != "" {
			if lastTime != "" && t < lastTime {
				add(FindOrdering, fmt.Sprintf("event %s recorded at %s precedes an earlier-inserted event at %s", e.Ordinal, t, lastTime), e.ID)
			}
			if t > lastTime {
				lastTime = t
			}
		}
		switch e.Kind {
		case HolderAssigned:
			switch {
			case r.State == Derived && e.From == nil:
				add(FindMissingPriorOwner, "assignment does not name the prior holder", e.ID)
			case r.State == Derived && !eq(e.From, r.Holder):
				add(FindHistoryGap, "assignment's prior holder differs from the derived holder", e.ID)
			case (r.State == None || r.State == Unknown) && e.From != nil:
				add(FindHistoryGap, "assignment from a holder never established by an earlier event", e.ID)
			}
			if e.To == nil {
				add(FindMissingTargetOwner, "assignment does not name the new holder", e.ID)
				r.State, r.Holder, r.Position = Unknown, nil, nil
			} else {
				r.State, r.Holder, r.Position = Derived, e.To, e.ToPosition
			}
			defining = append(defining, i)
		case HolderReleased:
			if r.State == Derived && e.From != nil && !eq(e.From, r.Holder) {
				add(FindHistoryGap, "release names a holder that differs from the derived holder", e.ID)
			}
			r.State, r.Holder, r.Position = Released, nil, nil
			defining = append(defining, i)
		case NoHolderChange:
			if r.State == Derived && e.From != nil && !eq(e.From, r.Holder) {
				add(FindHistoryGap, "event records a holder that differs from the derived holder", e.ID)
			}
			if (r.State == None || r.State == Unknown) && e.From != nil {
				add(FindHistoryGap, "event records a holder never established by an earlier event", e.ID)
			}
		default:
			add(FindUnknownAction, "unmapped legacy action "+e.Action+"; holder cannot be derived past it", e.ID)
			r.State, r.Holder, r.Position = Unknown, nil, nil
		}
	}
	// MULTIPLE: the final holder-defining events share one recorded time
	// but name different outcomes; insertion order alone does not prove
	// which one is current (concurrent writes).
	if n := len(defining); n >= 2 && r.State != Unknown {
		a, b := ev[defining[n-2]], ev[defining[n-1]]
		ta, tb := readable(a.OccurredRaw), readable(b.OccurredRaw)
		if ta != "" && ta == tb && !(a.Kind == b.Kind && (a.Kind == HolderReleased || eq(a.To, b.To))) {
			add(FindMultipleCurrent, "the last two holder-defining events share the recorded time "+tb+" with different outcomes", b.ID)
			r.State, r.Holder, r.Position = Multiple, nil, nil
		}
	}
	return r
}

// Classify compares the derivation with the legacy current-holder evidence
// (nil = no legacy holder). A holder that matches but whose history has
// gaps is HISTORY_GAP, not MATCH: parity is never overstated.
func Classify(r Result, legacy *string) string {
	gaps := false
	for _, f := range r.Findings {
		switch f.Kind {
		case FindHistoryGap, FindMissingPriorOwner, FindOrdering, FindUnknownAction, FindMissingTargetOwner:
			gaps = true
		}
	}
	switch r.State {
	case Multiple:
		return MultipleCurrent
	case None, Unknown:
		if legacy == nil {
			return NotRecoverable
		}
		return TargetUnknown
	case Released:
		if legacy != nil {
			return Conflict
		}
	case Derived:
		if legacy == nil {
			return LegacyUnknown
		}
		if !eq(legacy, r.Holder) {
			return Conflict
		}
	}
	if gaps {
		return HistoryGap
	}
	return Match
}
