// Package domain holds the customers module's pure rules: contact
// normalization for comparison, the fixed evidence catalogue, candidate
// grouping and the deterministic resolver. Nothing here merges, picks a
// winner or rewrites a customer: it reports.
package domain

import (
	"crypto/sha256"
	"sort"
	"strings"
	"time"
)

// --- contact normalization (comparison / discovery only) ----------------------

// Normalization states.
const (
	NormValid   = "VALID"
	NormInvalid = "INVALID"
	NormEmpty   = "EMPTY"
)

// NormalizePhone returns a comparison form of a raw phone value: digits
// only, Iranian mobile variants (+98 / 0098 / 98 / leading 0 or none)
// folded to the 11-digit "09…" form. It exists ONLY for search, comparison
// and candidate discovery (Architecture V1 §10): equal results never mean
// one owner, never create a customer, a merge or a RESOLVED state, and the
// result is never a key. Anything else is INVALID (raw kept as evidence).
func NormalizePhone(raw string) (string, string) {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r >= '۰' && r <= '۹' {
			b.WriteRune('0' + (r - '۰'))
		}
	}
	d := b.String()
	if strings.TrimSpace(raw) == "" {
		return "", NormEmpty
	}
	switch {
	case strings.HasPrefix(d, "0098") && len(d) == 14:
		d = "0" + d[4:]
	case strings.HasPrefix(d, "98") && len(d) == 12:
		d = "0" + d[2:]
	case strings.HasPrefix(d, "9") && len(d) == 10:
		d = "0" + d
	}
	if len(d) == 11 && strings.HasPrefix(d, "09") {
		return d, NormValid
	}
	return "", NormInvalid
}

// NormalizeEmail lower-cases and trims an e-mail for comparison only.
func NormalizeEmail(raw string) (string, string) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "", NormEmpty
	}
	at := strings.IndexByte(s, '@')
	if at < 1 || at == len(s)-1 || strings.ContainsAny(s, " \t") {
		return "", NormInvalid
	}
	return s, NormValid
}

// Mask hides a contact value for display: first two and last two
// characters stay, the rest becomes "*". Short values are fully masked.
func Mask(raw string) string {
	r := []rune(raw)
	if len(r) <= 4 {
		return strings.Repeat("*", len(r))
	}
	return string(r[:2]) + strings.Repeat("*", len(r)-4) + string(r[len(r)-2:])
}

// --- evidence catalogue ----------------------------------------------------------

// Strength, polarity and resolution states.
const (
	Strong = "STRONG"
	Medium = "MEDIUM"
	Weak   = "WEAK"

	Equality      = "EQUALITY"
	Relation      = "RELATION"
	Discovery     = "DISCOVERY"
	Contradiction = "CONTRADICTION"
	Missing       = "MISSING"

	Resolved       = "RESOLVED"
	Related        = "RELATED"
	Unknown        = "UNKNOWN"
	Conflict       = "CONFLICT"
	NotRecoverable = "NOT_RECOVERABLE"
)

// Evidence kinds.
const (
	PhoneMatch                = "PHONE_MATCH"
	AccountRefMatch           = "ACCOUNT_REF_MATCH"
	SequentialPhoneReuse      = "SEQUENTIAL_PHONE_REUSE"
	OverlappingPhoneOwnership = "OVERLAPPING_PHONE_OWNERSHIP"
	ExclusiveIdentifierShared = "EXCLUSIVE_IDENTIFIER_SHARED"
	OTPContactControl         = "OTP_CONTACT_CONTROL"
	VerifiedEquality          = "VERIFIED_EQUALITY"
	VerifiedDistinct          = "VERIFIED_DISTINCT"
	SourceNotRecoverable      = "SOURCE_NOT_RECOVERABLE"
)

// Kind is the fixed strength and polarity of an evidence kind. Strength is
// never chosen by the caller: it is a property of the kind.
type Kind struct {
	Strength string
	Polarity string
}

// Catalogue is the complete, code-defined evidence catalogue.
var Catalogue = map[string]Kind{
	// Equal normalized phone on different subjects: discovery only (§10).
	PhoneMatch: {Weak, Discovery},
	// Same legacy account id: the legacy account is itself found by phone
	// (2.0.160), so this proves a recorded link, never equality.
	AccountRefMatch: {Weak, Relation},
	// Same phone with explicit, non-overlapping ownership windows: reuse.
	SequentialPhoneReuse: {Weak, Discovery},
	// Same phone with explicit, overlapping ownership windows: unexplained.
	OverlappingPhoneOwnership: {Weak, Contradiction},
	// One exclusive identifier claimed by different subjects.
	ExclusiveIdentifierShared: {Medium, Contradiction},
	// Control of a contact at a time; never customer equality.
	OTPContactControl: {Medium, Relation},
	// Explicit strong evidence (synthetic only in Slice 5).
	VerifiedEquality: {Strong, Equality},
	VerifiedDistinct: {Strong, Contradiction},
	// The subject's source identity is permanently missing.
	SourceNotRecoverable: {Weak, Missing},
}

// Evidence is one evidence item as the resolver sees it.
type Evidence struct {
	Kind     string
	Strength string
	Polarity string
}

// Resolve returns the state of one evidence set (one merge case) and why.
// Deterministic rules, in order:
//  1. any CONTRADICTION                    → CONFLICT
//  2. any MISSING                          → NOT_RECOVERABLE
//  3. any EQUALITY of STRONG strength      → RESOLVED
//  4. any RELATION                         → RELATED
//  5. otherwise (discovery only, or none)  → UNKNOWN
//
// Consequences (tested): WEAK-only and phone-only sets can never resolve;
// OTP (MEDIUM, RELATION) never resolves; a STRONG equality is overruled by
// any contradiction, including a STRONG distinctness.
func Resolve(ev []Evidence) (string, string) {
	has := func(pol string, strength string) bool {
		for _, e := range ev {
			if e.Polarity == pol && (strength == "" || e.Strength == strength) {
				return true
			}
		}
		return false
	}
	switch {
	case has(Contradiction, ""):
		return Conflict, "contradictory evidence: " + kinds(ev, Contradiction)
	case has(Missing, ""):
		return NotRecoverable, "source identity permanently missing"
	case has(Equality, Strong):
		return Resolved, "strong equality evidence: " + kinds(ev, Equality)
	case has(Relation, ""):
		return Related, "relationship evidence without equality: " + kinds(ev, Relation)
	case len(ev) == 0:
		return Unknown, "no evidence"
	}
	return Unknown, "discovery evidence only (" + kinds(ev, Discovery) + "); insufficient for any relation"
}

func kinds(ev []Evidence, pol string) string {
	set := map[string]bool{}
	for _, e := range ev {
		if e.Polarity == pol {
			set[e.Kind] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// CustomerState projects one customer's state from its own recoverability
// and the states of the cases it belongs to. Priority:
// NOT_RECOVERABLE > CONFLICT > RESOLVED > RELATED > UNKNOWN. A customer in
// no case is UNKNOWN: one issued-to record proves nothing beyond itself.
func CustomerState(recoverable bool, caseStates []string) (string, string) {
	if !recoverable {
		return NotRecoverable, "source identity of the issued-to record is permanently missing"
	}
	rank := map[string]int{Conflict: 4, Resolved: 3, Related: 2, Unknown: 1}
	best := ""
	for _, s := range caseStates {
		if rank[s] > rank[best] {
			best = s
		}
	}
	switch best {
	case Conflict:
		return Conflict, "member of a conflicting candidate case"
	case Resolved:
		return Resolved, "strong equality evidence in a candidate case (no merge executed)"
	case Related:
		return Related, "related to another subject without equality"
	case Unknown:
		return Unknown, "shares discovery evidence only (e.g. a phone); insufficient evidence"
	}
	return Unknown, "single source subject; no evidence beyond itself"
}

// --- candidate grouping ------------------------------------------------------------

// Window is a contact ownership window parsed from raw legacy evidence. A
// contact without an explicit window has Known=false (point observation).
type Window struct {
	Known     bool
	From, To  time.Time
	OpenEnded bool
}

// ParseWindow reads explicit raw "YYYY-MM-DD HH:MM:SS" bounds. Missing or
// unreadable bounds make the window unknown (never guessed).
func ParseWindow(fromRaw, toRaw *string) Window {
	from, ok := parse(fromRaw)
	if !ok {
		return Window{}
	}
	if toRaw == nil {
		return Window{Known: true, From: from, OpenEnded: true}
	}
	to, ok := parse(toRaw)
	if !ok || to.Before(from) {
		return Window{}
	}
	return Window{Known: true, From: from, To: to}
}

func parse(raw *string) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02 15:04:05", *raw)
	if err != nil || t.Year() < 1 {
		return time.Time{}, false
	}
	return t, true
}

// Overlap reports whether two known windows overlap (inclusive).
func Overlap(a, b Window) bool {
	aEnd, bEnd := a.To, b.To
	if a.OpenEnded {
		aEnd = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if b.OpenEnded {
		bEnd = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return !a.From.After(bEnd) && !b.From.After(aEnd)
}

// Key returns a deterministic 32-byte key over parts (case and evidence keys).
func Key(parts ...string) []byte {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}
