// Package imports is the custody module's offline shadow import command. It
// resolves Case references through the published cases API (a separate,
// read-only cases transaction), then writes the custody schema only, in one
// custody-owned transaction. It is never reachable from a read path.
//
// Events are appended once (a changed legacy row is a CONFLICT, never a
// rewrite). The Assignment projection is re-derived from ALL stored events
// of each case next to the legacy current-holder evidence; findings report
// gaps and contradictions and are never repaired. Absence is marked only for
// COMPLETE holder specs; events are never removed.
package imports

import (
	"context"
	"errors"
	"sort"

	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Command is the custody import command.
type Command struct {
	runner *db.Runner
	cases  casesapi.Reader
	clock  timeutil.Clock
}

var _ api.Importer = (*Command)(nil)

// New returns the command over a custody-owned runner and the published
// cases read API.
func New(runner *db.Runner, cases casesapi.Reader, clock timeutil.Clock) *Command {
	return &Command{runner: runner, cases: cases, clock: clock}
}

// ImportCustody applies one batch.
func (c *Command) ImportCustody(ctx context.Context, in api.CustodyImport) (api.ImportReport, error) {
	rep := api.ImportReport{Counts: map[string]int64{}}
	if err := in.Validate(); err != nil {
		rep.Outcome = result.Failed
		return rep, err
	}
	// Resolve every referenced case through the published cases API.
	bySpec := map[string][]string{}
	for _, h := range in.Holders {
		bySpec[h.Case.Spec] = append(bySpec[h.Case.Spec], h.Case.Key)
	}
	for _, e := range in.Events {
		bySpec[e.Item.Spec] = append(bySpec[e.Item.Spec], e.Item.Key)
	}
	caseOf := map[api.RowRef]string{}
	for spec, keys := range bySpec {
		m, err := c.cases.ResolveSources(ctx, in.SourceSystem, spec, keys)
		if err != nil {
			return rep, err
		}
		for k, sc := range m {
			caseOf[api.RowRef{Spec: spec, Key: k}] = sc.CaseID
		}
	}

	now := c.clock.Now().Time()
	runID, err := ids.NewUUID()
	if err != nil {
		return rep, err
	}
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		rep = api.ImportReport{RunID: runID, Counts: map[string]int64{}}
		if err := store.LockImport(ctx, tx); err != nil {
			return err
		}
		prior, err := store.PriorRun(ctx, tx, in.BatchID)
		if err != nil {
			return err
		}
		changed := int64(0)
		bump := func(name string, n int64) {
			rep.Counts[name] += n
			changed += n
		}
		conflict, partial := false, false

		// 1. Append-only events.
		for _, e := range in.Events {
			caseID, ok := caseOf[e.Item]
			if !ok {
				rep.Counts["events_case_unresolved"]++
				partial = true
				continue
			}
			ins, conf, err := store.AppendEvent(ctx, tx, caseID, e, now)
			if err != nil {
				return err
			}
			switch {
			case ins:
				bump("events_appended", 1)
			case conf:
				rep.Counts["events_changed_conflict"]++
				conflict = true
			}
			rep.Counts["events_kind_"+e.Kind]++
		}

		// 2. Derived projection per case, from ALL stored events.
		seenBySpec := map[string][]string{}
		var processed, seenFindings []string
		for _, h := range in.Holders {
			caseID, ok := caseOf[h.Case]
			if !ok {
				rep.Counts["holders_case_unresolved"]++
				partial = true
				continue
			}
			events, users, err := store.EventsOf(ctx, tx, caseID)
			if err != nil {
				return err
			}
			d := domain.Derive(events)
			var legacy *string
			a := store.AssignmentInput{CaseID: caseID, Derivation: d.State, DerivedKey: d.Holder, DerivedPosition: d.Position,
				LegacyPosition: h.PositionRaw, LegacySpec: h.Case.Spec, EventCount: len(events)}
			if h.Holder != nil {
				k := h.Holder.LegacyKey
				legacy, a.LegacyKey, a.LegacyUserID = &k, &k, h.Holder.UserID
			}
			if d.Holder != nil {
				if u, ok := users[*d.Holder]; ok {
					a.DerivedUserID = &u
				}
			}
			if d.LastID != "" {
				last := d.LastID
				a.LastEventID = &last
			}
			a.Parity = domain.Classify(d, legacy)
			findings := d.Findings
			if a.Parity == domain.Conflict {
				findings = append(findings, domain.Finding{Kind: domain.FindSourceMismatch,
					Detail: "legacy current holder differs from the holder derived from custody events"})
			}
			ch, err := store.UpsertAssignment(ctx, tx, a, in.BatchID, now)
			if err != nil {
				return err
			}
			if ch {
				bump("assignments_changed", 1)
			}
			rep.Counts["parity_"+a.Parity]++
			for _, f := range findings {
				id, ch, err := store.RecordFinding(ctx, tx, caseID, f, in.BatchID, now)
				if err != nil {
					return err
				}
				if ch {
					bump("findings_changed", 1)
				}
				seenFindings = append(seenFindings, id)
				rep.Counts["finding_"+f.Kind]++
			}
			processed = append(processed, caseID)
			seenBySpec[h.Case.Spec] = append(seenBySpec[h.Case.Spec], caseID)
		}
		n, err := store.MarkAbsentFindings(ctx, tx, processed, seenFindings)
		if err != nil {
			return err
		}
		bump("findings_marked_absent", n)
		specs := make([]string, 0, len(in.Complete))
		for sp := range in.Complete {
			specs = append(specs, sp)
		}
		sort.Strings(specs)
		for _, sp := range specs {
			if !in.Complete[sp] {
				partial = true
				continue
			}
			n, err := store.MarkAbsentAssignments(ctx, tx, sp, seenBySpec[sp], now)
			if err != nil {
				return err
			}
			bump("assignments_marked_absent", n)
		}

		rep.Replay = prior && changed == 0
		switch {
		case conflict:
			rep.Outcome = result.Conflict
		case partial:
			rep.Outcome = result.Partial
		default:
			rep.Outcome = result.Success
		}
		return store.InsertRun(ctx, tx, runID, in.BatchID, in.SnapshotID, rep.Outcome.String(), rep.Replay, rep.Counts, now)
	})
	if err != nil {
		rep.Outcome = result.Failed
		if errors.Is(err, db.ErrCommitOutcomeUnknown) {
			rep.Outcome = result.Unknown
		}
		return rep, err
	}
	return rep, nil
}
