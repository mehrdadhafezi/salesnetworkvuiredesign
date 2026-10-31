// Package domain holds the organization module's pure rules: legacy date
// reading and hierarchy conflict detection. Nothing here repairs data. The
// analysis reports what the evidence says; missing parents stay missing,
// cycles stay cycles, and an unreadable date is UNKNOWN, never guessed.
package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// --- legacy dates -------------------------------------------------------------

var (
	dateTimeRaw = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	dateRaw     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// LegacyDateTime reads a raw legacy DATETIME as a wall-clock value whose zone
// is NOT asserted. It is comparable only with other values of the same
// source; it is never converted to an instant. Zero dates and anything that
// is not exactly "YYYY-MM-DD HH:MM:SS" are unreadable (ok=false).
func LegacyDateTime(raw *string) (time.Time, bool) {
	if raw == nil || !dateTimeRaw.MatchString(*raw) {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02 15:04:05", *raw)
	if err != nil || t.Year() < 1 {
		return time.Time{}, false
	}
	return t, true
}

// LegacyDate reads a raw legacy DATE ("YYYY-MM-DD"). Zero or invalid dates
// are unreadable.
func LegacyDate(raw *string) (time.Time, bool) {
	if raw == nil || !dateRaw.MatchString(*raw) {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", *raw)
	if err != nil || t.Year() < 1 {
		return time.Time{}, false
	}
	return t, true
}

// --- hierarchy analysis --------------------------------------------------------

// Severity of a finding.
type Severity string

const (
	SevConflict Severity = "CONFLICT"
	SevUnknown  Severity = "UNKNOWN"
	SevInfo     Severity = "INFO"
)

// Finding kinds (schema CHECK mirrors this list).
const (
	KindMissingParent           = "MISSING_PARENT"
	KindSelfParent              = "SELF_PARENT"
	KindCycle                   = "CYCLE"
	KindDuplicateCurrent        = "DUPLICATE_CURRENT"
	KindMultipleCurrentParents  = "MULTIPLE_CURRENT_PARENTS"
	KindIntervalOverlap         = "INTERVAL_OVERLAP"
	KindHistoryGap              = "HISTORY_GAP"
	KindUnexplainedGap          = "UNEXPLAINED_GAP"
	KindMissingChild            = "MISSING_CHILD"
	KindOrgUnitParentMissing    = "ORG_UNIT_PARENT_MISSING"
	KindOrgUnitParentNotRecord  = "ORG_UNIT_PARENT_NOT_RECORDED"
	KindMembershipMismatch      = "MEMBERSHIP_MISMATCH"
	KindCurrentFlagInconsistent = "CURRENT_FLAG_INCONSISTENT"
	KindHistoryCurrentDisagree  = "HISTORY_CURRENT_DISAGREE"
	KindUnreadableDate          = "UNREADABLE_DATE"
	KindUnmappedRelationship    = "UNMAPPED_RELATIONSHIP"
	KindMissingReference        = "MISSING_REFERENCE"
	KindSharedUser              = "SHARED_USER"
	KindUserNotImported         = "USER_NOT_IMPORTED"
	KindHistoryChanged          = "HISTORY_CHANGED"
)

// FindingKinds lists every kind in a stable order.
func FindingKinds() []string {
	return []string{KindMissingParent, KindSelfParent, KindCycle, KindDuplicateCurrent, KindMultipleCurrentParents,
		KindIntervalOverlap, KindHistoryGap, KindUnexplainedGap, KindMissingChild, KindOrgUnitParentMissing,
		KindOrgUnitParentNotRecord, KindMembershipMismatch, KindCurrentFlagInconsistent, KindHistoryCurrentDisagree,
		KindUnreadableDate, KindUnmappedRelationship, KindMissingReference, KindSharedUser, KindUserNotImported, KindHistoryChanged}
}

// Subject types.
const (
	SubjectAssignment = "ASSIGNMENT"
	SubjectEmployment = "EMPLOYMENT"
	SubjectOrgUnit    = "ORG_UNIT"
	SubjectHistory    = "HISTORY"
)

// Finding is one reported hierarchy problem. Detail names keys and reasons,
// never personal values.
type Finding struct {
	Kind        string
	Severity    Severity
	SubjectType string
	SubjectID   *string
	SubjectKey  *string
	RelatedID   *string
	RelatedKey  *string
	Detail      string
}

// Ref states (schema CHECKs mirror these).
const (
	RefResolved    = "RESOLVED"
	RefMissing     = "MISSING"
	RefNotRecorded = "NOT_RECORDED"
)

// UnitNode is one org unit as resolved by the import.
type UnitNode struct {
	ID          string
	Kind        string
	Key         string
	ParentID    *string
	ParentKey   *string
	ParentState string // NOT_APPLICABLE | NOT_RECORDED | RESOLVED | MISSING
}

// RefState is one resolved reference of an employment.
type RefState struct {
	ID    *string
	Key   *string
	State string
}

// EmploymentNode is one employment as resolved by the import.
type EmploymentNode struct {
	ID         string
	Key        string // legacy profile key
	UserLink   string
	UserKey    *string
	Position   RefState
	Level      RefState
	Department RefState
	Unit       RefState
	Team       RefState
}

// AssignmentNode is one reporting assignment.
type AssignmentNode struct {
	ID               string
	Key              string
	ChildID          *string
	ChildKey         string
	ParentID         *string
	ParentKey        *string
	ParentState      string
	Kind             string // REPORTS_TO | UNMAPPED
	EffectiveFromRaw *string
	EffectiveToRaw   *string
	IsCurrent        *bool
	CurrentRaw       *string
}

// HistoryNode is one assignment history row.
type HistoryNode struct {
	ID           string
	Key          string
	ChildKey     *string
	OldParentKey *string
	NewParentKey *string
	RecordedRaw  *string
}

// Input is the whole resolved organisation model of one import.
type Input struct {
	Units       []UnitNode
	Employments []EmploymentNode
	Assignments []AssignmentNode
	History     []HistoryNode
}

func ptr(s string) *string { return &s }

func strOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

// Analyze returns every finding, sorted deterministically. It never
// changes the input.
func Analyze(in Input) []Finding {
	var out []Finding
	add := func(f Finding) { out = append(out, f) }

	units := map[string]UnitNode{}
	for _, u := range in.Units {
		units[u.ID] = u
		switch u.ParentState {
		case RefMissing:
			add(Finding{Kind: KindOrgUnitParentMissing, Severity: SevConflict, SubjectType: SubjectOrgUnit, SubjectID: ptr(u.ID), SubjectKey: ptr(u.Key),
				RelatedKey: u.ParentKey, Detail: fmt.Sprintf("%s %s references parent key %s that does not exist", strings.ToLower(u.Kind), u.Key, strOr(u.ParentKey, "?"))})
		case RefNotRecorded:
			add(Finding{Kind: KindOrgUnitParentNotRecord, Severity: SevUnknown, SubjectType: SubjectOrgUnit, SubjectID: ptr(u.ID), SubjectKey: ptr(u.Key),
				Detail: fmt.Sprintf("%s %s has no parent recorded; it is not attached to any root", strings.ToLower(u.Kind), u.Key)})
		}
	}

	// Employments: references, membership consistency and account links.
	byUser := map[string][]EmploymentNode{}
	for _, e := range in.Employments {
		for _, r := range []struct {
			name string
			ref  RefState
		}{{"position", e.Position}, {"level", e.Level}, {"department", e.Department}, {"unit", e.Unit}, {"team", e.Team}} {
			if r.ref.State == RefMissing {
				add(Finding{Kind: KindMissingReference, Severity: SevConflict, SubjectType: SubjectEmployment, SubjectID: ptr(e.ID), SubjectKey: ptr(e.Key),
					RelatedKey: r.ref.Key, Detail: fmt.Sprintf("profile %s references %s key %s that does not exist", e.Key, r.name, strOr(r.ref.Key, "?"))})
			}
		}
		if e.Team.ID != nil && e.Unit.ID != nil {
			if t, ok := units[*e.Team.ID]; ok && (t.ParentID == nil || *t.ParentID != *e.Unit.ID) {
				add(Finding{Kind: KindMembershipMismatch, Severity: SevConflict, SubjectType: SubjectEmployment, SubjectID: ptr(e.ID), SubjectKey: ptr(e.Key),
					RelatedID: e.Team.ID, Detail: fmt.Sprintf("profile %s: team %s does not belong to unit %s", e.Key, strOr(e.Team.Key, "?"), strOr(e.Unit.Key, "?"))})
			}
		}
		if e.Unit.ID != nil && e.Department.ID != nil {
			if u, ok := units[*e.Unit.ID]; ok && (u.ParentID == nil || *u.ParentID != *e.Department.ID) {
				add(Finding{Kind: KindMembershipMismatch, Severity: SevConflict, SubjectType: SubjectEmployment, SubjectID: ptr(e.ID), SubjectKey: ptr(e.Key),
					RelatedID: e.Unit.ID, Detail: fmt.Sprintf("profile %s: unit %s does not belong to department %s", e.Key, strOr(e.Unit.Key, "?"), strOr(e.Department.Key, "?"))})
			}
		}
		switch e.UserLink {
		case "USER_NOT_IMPORTED":
			add(Finding{Kind: KindUserNotImported, Severity: SevUnknown, SubjectType: SubjectEmployment, SubjectID: ptr(e.ID), SubjectKey: ptr(e.Key),
				RelatedKey: e.UserKey, Detail: fmt.Sprintf("profile %s references account key %s that is not imported", e.Key, strOr(e.UserKey, "?"))})
		}
		if e.UserKey != nil && e.UserLink != "USER_NOT_IMPORTED" {
			byUser[*e.UserKey] = append(byUser[*e.UserKey], e)
		}
	}
	for userKey, es := range byUser {
		if len(es) < 2 {
			continue
		}
		for _, e := range es {
			add(Finding{Kind: KindSharedUser, Severity: SevConflict, SubjectType: SubjectEmployment, SubjectID: ptr(e.ID), SubjectKey: ptr(e.Key),
				RelatedKey: ptr(userKey), Detail: fmt.Sprintf("account key %s is linked to %d profiles; the legacy lookup picks one without an order", userKey, len(es))})
		}
	}

	// Assignments.
	type interval struct {
		a        AssignmentNode
		from, to time.Time
		hasTo    bool
	}
	perChild := map[string][]interval{}
	currentByChild := map[string][]AssignmentNode{}
	for _, a := range in.Assignments {
		subj := func(kind string, sev Severity, detail string) Finding {
			return Finding{Kind: kind, Severity: sev, SubjectType: SubjectAssignment, SubjectID: ptr(a.ID), SubjectKey: ptr(a.Key), Detail: detail}
		}
		if a.ChildID == nil {
			add(subj(KindMissingChild, SevConflict, fmt.Sprintf("assignment %s: child profile key %s does not exist", a.Key, a.ChildKey)))
		}
		if a.ParentState == RefMissing {
			f := subj(KindMissingParent, SevConflict, fmt.Sprintf("assignment %s: parent profile key %s does not exist", a.Key, strOr(a.ParentKey, "?")))
			f.RelatedKey = a.ParentKey
			add(f)
		}
		if a.Kind != "REPORTS_TO" {
			add(subj(KindUnmappedRelationship, SevUnknown, fmt.Sprintf("assignment %s: relationship type is not a known reporting line; kept raw", a.Key)))
		}
		if a.ParentKey != nil && *a.ParentKey == a.ChildKey {
			add(subj(KindSelfParent, SevConflict, fmt.Sprintf("assignment %s: profile %s reports to itself", a.Key, a.ChildKey)))
		}
		from, fromOK := LegacyDateTime(a.EffectiveFromRaw)
		to, toOK := LegacyDateTime(a.EffectiveToRaw)
		if (a.EffectiveFromRaw != nil && !fromOK) || (a.EffectiveToRaw != nil && !toOK) {
			add(subj(KindUnreadableDate, SevUnknown, fmt.Sprintf("assignment %s: effective interval is not a readable legacy datetime; interval is UNKNOWN", a.Key)))
		}
		switch {
		case a.IsCurrent == nil:
			add(subj(KindCurrentFlagInconsistent, SevUnknown, fmt.Sprintf("assignment %s: current flag is not readable", a.Key)))
		case *a.IsCurrent && a.EffectiveToRaw != nil:
			add(subj(KindCurrentFlagInconsistent, SevConflict, fmt.Sprintf("assignment %s: marked current but has an end", a.Key)))
		case !*a.IsCurrent && a.EffectiveToRaw == nil:
			add(subj(KindCurrentFlagInconsistent, SevConflict, fmt.Sprintf("assignment %s: not current but has no end", a.Key)))
		}
		if a.Kind != "REPORTS_TO" {
			continue
		}
		if a.IsCurrent != nil && *a.IsCurrent {
			currentByChild[a.ChildKey] = append(currentByChild[a.ChildKey], a)
		}
		if fromOK && (a.EffectiveToRaw == nil || toOK) {
			perChild[a.ChildKey] = append(perChild[a.ChildKey], interval{a: a, from: from, to: to, hasTo: a.EffectiveToRaw != nil})
		}
	}

	// Current parents per child.
	for child, cur := range currentByChild {
		if len(cur) < 2 {
			continue
		}
		parents := map[string]int{}
		for _, a := range cur {
			parents[strOr(a.ParentKey, "")]++
		}
		for _, a := range cur {
			kind := KindDuplicateCurrent
			detail := fmt.Sprintf("profile %s has %d current assignments to the same parent", child, len(cur))
			if len(parents) > 1 {
				kind = KindMultipleCurrentParents
				detail = fmt.Sprintf("profile %s has %d current reporting parents; legacy reads one without a defined winner", child, len(parents))
			}
			add(Finding{Kind: kind, Severity: SevConflict, SubjectType: SubjectAssignment, SubjectID: ptr(a.ID), SubjectKey: ptr(a.Key), Detail: detail})
		}
	}

	// History lookup for gap explanations and current disagreement.
	type hist struct {
		h   HistoryNode
		at  time.Time
		ok  bool
		key string
	}
	histByChild := map[string][]hist{}
	for _, h := range in.History {
		if h.ChildKey == nil {
			continue
		}
		at, ok := LegacyDateTime(h.RecordedRaw)
		histByChild[*h.ChildKey] = append(histByChild[*h.ChildKey], hist{h: h, at: at, ok: ok, key: h.Key})
	}
	linked := func(child string, oldP, newP *string) bool {
		for _, h := range histByChild[child] {
			if strOr(h.h.OldParentKey, "") == strOr(oldP, "") && strOr(h.h.NewParentKey, "") == strOr(newP, "") {
				return true
			}
		}
		return false
	}

	// Intervals: overlap and gaps.
	for child, ivs := range perChild {
		sort.Slice(ivs, func(i, j int) bool {
			if !ivs[i].from.Equal(ivs[j].from) {
				return ivs[i].from.Before(ivs[j].from)
			}
			return keyLess(ivs[i].a.Key, ivs[j].a.Key)
		})
		for i := 1; i < len(ivs); i++ {
			prev, next := ivs[i-1], ivs[i]
			if !prev.hasTo || next.from.Before(prev.to) {
				add(Finding{Kind: KindIntervalOverlap, Severity: SevConflict, SubjectType: SubjectAssignment, SubjectID: ptr(next.a.ID), SubjectKey: ptr(next.a.Key),
					RelatedID: ptr(prev.a.ID), RelatedKey: ptr(prev.a.Key), Detail: fmt.Sprintf("profile %s: assignment %s starts before assignment %s ends", child, next.a.Key, prev.a.Key)})
				continue
			}
			if next.from.After(prev.to) {
				kind, sev, why := KindUnexplainedGap, SevInfo, "no history row explains the change"
				if linked(child, prev.a.ParentKey, next.a.ParentKey) {
					kind, sev, why = KindHistoryGap, SevUnknown, "history records the change, so continuity is expected; the gap period is UNKNOWN"
				}
				add(Finding{Kind: kind, Severity: sev, SubjectType: SubjectAssignment, SubjectID: ptr(next.a.ID), SubjectKey: ptr(next.a.Key),
					RelatedID: ptr(prev.a.ID), RelatedKey: ptr(prev.a.Key), Detail: fmt.Sprintf("profile %s: gap between assignment %s and %s; %s", child, prev.a.Key, next.a.Key, why)})
			}
		}
	}

	// History versus current.
	for child, hs := range histByChild {
		var readable []hist
		for _, h := range hs {
			if h.ok {
				readable = append(readable, h)
			}
		}
		if len(readable) == 0 {
			continue
		}
		sort.Slice(readable, func(i, j int) bool {
			if !readable[i].at.Equal(readable[j].at) {
				return readable[i].at.Before(readable[j].at)
			}
			return keyLess(readable[i].key, readable[j].key)
		})
		latest := readable[len(readable)-1].h
		cur := currentByChild[child]
		if len(cur) > 1 {
			continue // already a current-parent conflict
		}
		curParent := ""
		subjectID, subjectKey := (*string)(nil), (*string)(nil)
		if len(cur) == 1 {
			curParent = strOr(cur[0].ParentKey, "")
			subjectID, subjectKey = ptr(cur[0].ID), ptr(cur[0].Key)
		}
		if strOr(latest.NewParentKey, "") != curParent {
			f := Finding{Kind: KindHistoryCurrentDisagree, Severity: SevConflict, SubjectType: SubjectHistory, SubjectID: ptr(latest.ID), SubjectKey: ptr(latest.Key),
				Detail: fmt.Sprintf("profile %s: latest history sets parent %s but the current assignment says %s", child, strOr(latest.NewParentKey, "none"), orNone(curParent))}
			if subjectID != nil {
				f.RelatedID, f.RelatedKey = subjectID, subjectKey
			}
			add(f)
		}
	}

	// Cycles over current reporting edges between resolved employments.
	edges := map[string][]AssignmentNode{}
	for _, cur := range currentByChild {
		for _, a := range cur {
			if a.ChildID != nil && a.ParentID != nil && *a.ChildID != *a.ParentID {
				edges[*a.ChildID] = append(edges[*a.ChildID], a)
			}
		}
	}
	for _, cyc := range cycles(edges) {
		for _, a := range cyc {
			add(Finding{Kind: KindCycle, Severity: SevConflict, SubjectType: SubjectAssignment, SubjectID: ptr(a.ID), SubjectKey: ptr(a.Key),
				Detail: fmt.Sprintf("assignment %s is part of a reporting cycle of %d profiles", a.Key, len(cyc))})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		ki, kj := strOr(out[i].SubjectKey, ""), strOr(out[j].SubjectKey, "")
		if ki != kj {
			return keyLess(ki, kj)
		}
		return out[i].Detail < out[j].Detail
	})
	return out
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// keyLess orders canonical decimal keys numerically.
func keyLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// cycles returns each elementary cycle (as its edges) found by a
// deterministic DFS over child→parent edges; each cycle is reported once.
func cycles(edges map[string][]AssignmentNode) [][]AssignmentNode {
	nodes := make([]string, 0, len(edges))
	for n := range edges {
		nodes = append(nodes, n)
		sort.Slice(edges[n], func(i, j int) bool { return keyLess(edges[n][i].Key, edges[n][j].Key) })
	}
	sort.Strings(nodes)
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var stack []AssignmentNode
	seen := map[string]bool{}
	var out [][]AssignmentNode
	var visit func(n string)
	visit = func(n string) {
		color[n] = grey
		for _, e := range edges[n] {
			p := *e.ParentID
			switch color[p] {
			case white:
				stack = append(stack, e)
				visit(p)
				stack = stack[:len(stack)-1]
			case grey:
				// Cycle: edges from p back around to e.
				cyc := []AssignmentNode{e}
				for i := len(stack) - 1; i >= 0; i-- {
					cyc = append(cyc, stack[i])
					if *stack[i].ChildID == p {
						break
					}
				}
				var keys []string
				for _, c := range cyc {
					keys = append(keys, c.Key)
				}
				sort.Strings(keys)
				sig := strings.Join(keys, ",")
				if !seen[sig] {
					seen[sig] = true
					sort.Slice(cyc, func(i, j int) bool { return keyLess(cyc[i].Key, cyc[j].Key) })
					out = append(out, cyc)
				}
			}
		}
		color[n] = black
	}
	for _, n := range nodes {
		if color[n] == white {
			visit(n)
		}
	}
	return out
}
