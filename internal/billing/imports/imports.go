// Package imports is the billing module's offline shadow import command. It
// writes the billing schema only, in one billing-owned transaction, and is
// never reachable from a read path (architecture lint: read-path-import).
// It never writes payments state and never derives an invoice state from
// stage or payment evidence.
package imports

import (
	"context"
	"errors"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Command is the billing import command.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

// New returns the command over a billing-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) *Command {
	return &Command{runner: runner, clock: clock}
}

// ImportBilling applies one batch.
func (c *Command) ImportBilling(ctx context.Context, in api.BillingImport) (api.ImportReport, error) {
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
		partial := false
		count := func(prefix string, ch store.Change) {
			rep.Counts[prefix+"_"+string(ch)]++
			if ch != store.Unchanged {
				changed++
			}
		}
		present := map[string]map[string][]string{"invoice": {}, "item": {}, "stage": {}}
		for _, v := range in.Invoices {
			_, ch, err := store.UpsertInvoice(ctx, tx, v, now)
			if err != nil {
				return err
			}
			count("invoices", ch)
			present["invoice"][v.Source.Spec] = append(present["invoice"][v.Source.Spec], v.Source.Key)
			for _, s := range v.States {
				rep.Counts["state_"+string(s.Field)+"_"+string(s.Mapping)]++
			}
		}
		resolve := func(r api.RowRef) (string, bool, error) { return store.InvoiceOf(ctx, tx, in.SourceSystem, r) }
		for _, it := range in.Items {
			id, ok, err := resolve(it.Invoice)
			if err != nil {
				return err
			}
			if !ok {
				rep.Counts["items_invoice_unresolved"]++
				partial = true
				continue
			}
			ch, err := store.UpsertItem(ctx, tx, id, it, now)
			if err != nil {
				return err
			}
			count("items", ch)
			present["item"][it.Source.Spec] = append(present["item"][it.Source.Spec], it.Source.Key)
		}
		for _, st := range in.Stages {
			id, ok, err := resolve(st.Invoice)
			if err != nil {
				return err
			}
			if !ok {
				rep.Counts["stages_invoice_unresolved"]++
				partial = true
				continue
			}
			ch, err := store.UpsertStage(ctx, tx, id, st, now)
			if err != nil {
				return err
			}
			count("stages", ch)
			if st.StateTarget == nil {
				rep.Counts["stages_state_unmapped"]++
			}
			present["stage"][st.Source.Spec] = append(present["stage"][st.Source.Spec], st.Source.Key)
		}
		// Absence: only a COMPLETE spec may mark its rows absent; the spec's
		// kind is known from the rows of this command or from a prior
		// table, so an empty COMPLETE spec is checked in every table.
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
			for _, which := range []string{"invoice", "item", "stage"} {
				n, err := store.MarkAbsent(ctx, tx, which, in.SourceSystem, sp, present[which][sp], now)
				if err != nil {
					return err
				}
				if n > 0 {
					rep.Counts[which+"s_marked_absent"] += n
					changed += n
				}
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE billing.invoice SET items_source_complete = $2, stages_source_complete = $3
			WHERE last_batch_id = $1 AND (items_source_complete, stages_source_complete) IS DISTINCT FROM ($2, $3)`,
			in.BatchID, in.ItemsComplete, in.StagesComplete); err != nil {
			return err
		}
		rep.Replay = prior && changed == 0
		rep.Outcome = result.Success
		if partial {
			rep.Outcome = result.Partial
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
