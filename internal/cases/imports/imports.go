// Package imports is the cases module's offline shadow import command. It
// writes the cases schema only, in one cases-owned transaction, and is never
// reachable from a read path (architecture lint: read-path-import).
//
// One staged source row is one Case; a changed row refreshes the same Case
// (mutable projection) and never forks it. State history rows and
// attribution snapshots are appended once and never rewritten: a changed
// history row is reported as a CONFLICT, a changed attribution produces a
// new snapshot. Absence is marked only for COMPLETE specs; nothing is
// deleted. Aliases record explicit native links and are never RESOLVED
// (OPD-01: continuation versus related-but-distinct is open).
package imports

import (
	"context"
	"errors"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Command is the cases import command.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

var _ api.Importer = (*Command)(nil)

// New returns the command over a cases-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) *Command {
	return &Command{runner: runner, clock: clock}
}

// OPD01Basis is appended to every alias basis: the link is explicit, its
// meaning (continuation or related distinct work) is not decided.
const OPD01Basis = "; continuation vs related-distinct unresolved (OPD-01)"

// ImportCases applies one batch.
func (c *Command) ImportCases(ctx context.Context, in api.CaseImport) (api.ImportReport, error) {
	rep := api.ImportReport{Counts: map[string]int64{}}
	if err := in.Validate(); err != nil {
		rep.Outcome = result.Failed
		return rep, err
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

		// 1. Cases (one per source row).
		present := map[string][]string{}
		for _, cr := range in.Cases {
			_, ch, err := store.UpsertCase(ctx, tx, cr, now)
			if err != nil {
				return err
			}
			rep.Counts["cases_"+string(ch)]++
			if ch != store.Unchanged {
				changed++
			}
			present[cr.Source.Spec] = append(present[cr.Source.Spec], cr.Source.Key)
			if cr.StateMapping == api.StateUnknown {
				rep.Counts["cases_state_unknown"]++
			}
		}
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
			n, err := store.MarkAbsentCases(ctx, tx, in.SourceSystem, sp, present[sp], now)
			if err != nil {
				return err
			}
			bump("cases_marked_absent", n)
		}

		resolve := func(r api.RowRef) (string, bool, error) {
			return store.CaseOf(ctx, tx, in.SourceSystem, r.Spec, r.Key)
		}

		// 2. Append-only state history.
		for _, h := range in.History {
			caseID, ok, err := resolve(h.Case)
			if err != nil {
				return err
			}
			if !ok {
				rep.Counts["history_case_unresolved"]++
				partial = true
				continue
			}
			ins, conf, err := store.AppendHistory(ctx, tx, caseID, h, now)
			if err != nil {
				return err
			}
			switch {
			case ins:
				bump("history_appended", 1)
			case conf:
				rep.Counts["history_changed_conflict"]++
				conflict = true
			}
		}

		// 3. Explicit native links (never merges).
		seenAliases := map[string][]string{}
		for _, a := range in.Aliases {
			from, ok, err := resolve(a.From)
			if err != nil {
				return err
			}
			if !ok {
				rep.Counts["alias_case_unresolved"]++
				partial = true
				continue
			}
			ai := store.AliasInput{CaseID: from, RelatedSpec: a.Related.Spec, RelatedKey: a.Related.Key, Relation: string(a.Relation)}
			if rel, ok, err := resolve(a.Related); err != nil {
				return err
			} else if ok && rel != from {
				ai.RelatedCaseID, ai.Resolution, ai.Basis = &rel, string(api.Related), clip(a.Basis+OPD01Basis)
			} else {
				ai.Resolution, ai.Basis = string(api.Unknown), clip(a.Basis+"; linked source row is not staged")
			}
			id, ch, err := store.UpsertAlias(ctx, tx, ai, in.BatchID, now)
			if err != nil {
				return err
			}
			if ch {
				bump("aliases_changed", 1)
			}
			rep.Counts["aliases_"+ai.Resolution]++
			seenAliases[a.From.Spec] = append(seenAliases[a.From.Spec], id)
		}
		for _, sp := range specs {
			if in.Complete[sp] {
				n, err := store.MarkAbsentAliases(ctx, tx, in.SourceSystem, sp, seenAliases[sp])
				if err != nil {
					return err
				}
				bump("aliases_marked_absent", n)
			}
		}

		// 4. Immutable attribution snapshots.
		for _, a := range in.Attributions {
			caseID, ok, err := resolve(a.Case)
			if err != nil {
				return err
			}
			if !ok {
				rep.Counts["attribution_case_unresolved"]++
				partial = true
				continue
			}
			si := store.SnapshotInput{CaseID: caseID, Facet: string(a.Facet), Resolution: string(a.Resolution), PositionRaw: a.PositionRaw,
				Basis: a.Basis, SourceTimeRaw: a.SourceTimeRaw}
			if a.Subject != nil {
				k := a.Subject.LegacyKey
				si.SubjectKey, si.SubjectUserID = &k, a.Subject.UserID
			}
			if a.Evidence != nil {
				sp, k := a.Evidence.Spec, a.Evidence.Key
				si.EvidenceSpec, si.EvidenceKey = &sp, &k
			}
			ins, err := store.RecordSnapshot(ctx, tx, si, in.BatchID, now)
			if err != nil {
				return err
			}
			if ins {
				bump("attribution_snapshots_recorded", 1)
			}
			rep.Counts["attribution_"+string(a.Facet)+"_"+string(a.Resolution)]++
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

func clip(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}
