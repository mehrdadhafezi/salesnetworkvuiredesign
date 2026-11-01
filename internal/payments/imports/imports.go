// Package imports is the payments module's offline shadow import command.
// Invoice references are resolved FIRST through the published billing read
// API (its own billing-owned read-only transaction); the payments batch is
// then written in one payments-owned transaction. Billing and payments are
// never written in one transaction, and payments never writes billing.
package imports

import (
	"context"
	"errors"
	"sort"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Command is the payments import command.
type Command struct {
	runner  *db.Runner
	billing billingapi.Reader
	clock   timeutil.Clock
}

// New returns the command over a payments-owned runner and the published
// billing reader used to resolve invoice references.
func New(runner *db.Runner, billing billingapi.Reader, clock timeutil.Clock) *Command {
	return &Command{runner: runner, billing: billing, clock: clock}
}

// resolve maps every referenced invoice row to its billing invoice id.
func (c *Command) resolve(ctx context.Context, in api.PaymentsImport) (map[api.RowRef]string, error) {
	bySpec := map[string]map[string]bool{}
	add := func(r api.RowRef) {
		if bySpec[r.Spec] == nil {
			bySpec[r.Spec] = map[string]bool{}
		}
		bySpec[r.Spec][r.Key] = true
	}
	for _, a := range in.Attempts {
		add(a.Invoice)
	}
	for _, r := range in.Receipts {
		add(r.Invoice)
	}
	out := map[api.RowRef]string{}
	specs := make([]string, 0, len(bySpec))
	for sp := range bySpec {
		specs = append(specs, sp)
	}
	sort.Strings(specs)
	for _, sp := range specs {
		keys := make([]string, 0, len(bySpec[sp]))
		for k := range bySpec[sp] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		got, err := c.billing.ResolveInvoices(ctx, in.SourceSystem, sp, keys)
		if err != nil {
			return nil, err
		}
		for k, v := range got {
			out[api.RowRef{Spec: sp, Key: k}] = v.InvoiceID
		}
	}
	return out, nil
}

// ImportPayments applies one batch.
func (c *Command) ImportPayments(ctx context.Context, in api.PaymentsImport) (api.ImportReport, error) {
	rep := api.ImportReport{Counts: map[string]int64{}}
	if err := in.Validate(); err != nil {
		rep.Outcome = result.Failed
		return rep, err
	}
	invoices, err := c.resolve(ctx, in)
	if err != nil {
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
		partial := !in.AttemptsComplete || !in.ReceiptSourcesComplete
		count := func(prefix string, ch store.Change) {
			rep.Counts[prefix+"_"+string(ch)]++
			if ch != store.Unchanged {
				changed++
			}
		}
		link := func(r api.RowRef) *string {
			if id, ok := invoices[r]; ok {
				return &id
			}
			return nil
		}
		present := map[string][]string{}
		for _, a := range in.Attempts {
			inv := link(a.Invoice)
			if inv == nil {
				rep.Counts["attempts_invoice_unresolved"]++
				partial = true
			}
			_, ch, regression, err := store.UpsertAttempt(ctx, tx, inv, a, now)
			if err != nil {
				return err
			}
			count("attempts", ch)
			if regression {
				rep.Counts["attempts_state_regression"]++
			}
			if a.StateTarget == nil {
				rep.Counts["attempts_state_unmapped"]++
			}
			present[a.Source.Spec] = append(present[a.Source.Spec], a.Source.Key)
		}
		seen := []string{}
		for _, r := range in.Receipts {
			inv := link(r.Invoice)
			if inv == nil {
				rep.Counts["receipts_invoice_unresolved"]++
				partial = true
			}
			id, ch, err := store.UpsertReceipt(ctx, tx, inv, r, now)
			if err != nil {
				return err
			}
			count("receipts", ch)
			seen = append(seen, id)
		}
		if in.AttemptsComplete {
			// A COMPLETE attempt source with zero rows still marks prior rows
			// of every known spec absent.
			known, err := store.AttemptSpecs(ctx, tx, in.SourceSystem)
			if err != nil {
				return err
			}
			for _, sp := range known {
				n, err := store.MarkAbsentAttempts(ctx, tx, in.SourceSystem, sp, present[sp], now)
				if err != nil {
					return err
				}
				rep.Counts["attempts_marked_absent"] += n
				changed += n
			}
		}
		if in.ReceiptSourcesComplete {
			n, err := store.MarkAbsentReceipts(ctx, tx, in.SourceSystem, seen, now)
			if err != nil {
				return err
			}
			rep.Counts["receipts_marked_absent"] += n
			changed += n
		}
		rep.Counts["attempts_source_complete"] = boolInt(in.AttemptsComplete)
		rep.Counts["receipt_sources_complete"] = boolInt(in.ReceiptSourcesComplete)
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

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
