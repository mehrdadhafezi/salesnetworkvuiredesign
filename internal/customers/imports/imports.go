// Package imports implements the customers shadow import command
// (Architecture V1 §10, §24.4). The integration connector reads legacy
// staging and calls this command through customers/api; customers writes
// only its own tables, in one customers-owned transaction.
//
// Contract (Slice 5, D-29/D-30/D-31 open):
//   - one customer per source subject (one legacy issued-to record), minted
//     once; a changed source row refreshes the same customer, never forks;
//   - customers are never merged, collapsed, deleted or redirected; there
//     is no winner and no canonical choice — candidate cases only;
//   - contacts and identifiers are evidence; equal normalized phones never
//     make one customer; shared identifiers are reported (CONFLICT for
//     exclusive schemes);
//   - evidence content is append-only (deduplicated by a deterministic key);
//     changed facts append new evidence and the projection is recomputed;
//   - absence is a marker (ABSENT_IN_LATEST) set only for COMPLETE sources.
package imports

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// Command is the customers import command.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

var _ api.Importer = (*Command)(nil)

// New returns the command over a customers-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) *Command {
	return &Command{runner: runner, clock: clock}
}

type supplied struct {
	rec      api.EvidenceRecord
	subjects []string // customer ids
	contact  *string  // normalized contact for OTP
}

// ImportCustomers applies one batch.
func (c *Command) ImportCustomers(ctx context.Context, in api.CustomerImport) (api.ImportReport, error) {
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

		// 1. Subjects → customers (+ their contacts and identifiers).
		byKey := map[string]string{} // kind/source key → customer id
		presentByKind := map[api.SubjectKind][]string{}
		var touched, contactIDs, identifierIDs []string
		for _, s := range in.Subjects {
			id, ch, err := store.UpsertCustomer(ctx, tx, s, now)
			if err != nil {
				return err
			}
			rep.Counts["customers_"+string(ch)]++
			if ch != store.Unchanged {
				changed++
			}
			byKey[string(s.Kind)+"/"+s.Source.Key] = id
			presentByKind[s.Kind] = append(presentByKind[s.Kind], s.Source.Key)
			touched = append(touched, id)
			for _, ct := range s.Contacts {
				norm, state := domain.NormalizePhone(ct.Raw)
				if ct.Kind == api.Email {
					norm, state = domain.NormalizeEmail(ct.Raw)
				}
				var np *string
				if state == domain.NormValid {
					np = &norm
				}
				rep.Counts["contacts_"+strings.ToLower(state)]++
				cid, ch, err := store.UpsertContact(ctx, tx, store.ContactInput{CustomerID: id, Contact: ct, Normalized: np,
					Normalization: state, ObservedRaw: s.ObservedRaw, Source: s.Source}, now)
				if err != nil {
					return err
				}
				contactIDs = append(contactIDs, cid)
				if ch {
					bump("contacts_changed", 1)
				}
			}
			for _, idr := range s.Identifiers {
				iid, ch, err := store.UpsertIdentifier(ctx, tx, id, idr, s.Source, now)
				if err != nil {
					return err
				}
				identifierIDs = append(identifierIDs, iid)
				if ch {
					bump("identifiers_changed", 1)
				}
			}
		}
		rep.Counts["subjects_staged"] = int64(len(in.Subjects))
		n, err := store.MarkAbsentChildren(ctx, tx, touched, contactIDs, identifierIDs, now)
		if err != nil {
			return err
		}
		bump("contacts_identifiers_marked_absent", n)
		if in.Complete {
			for _, kind := range []api.SubjectKind{api.IssuedTo, api.SyntheticSubject} {
				n, err := store.MarkAbsentCustomers(ctx, tx, in.SourceSystem, kind, presentByKind[kind], now)
				if err != nil {
					return err
				}
				bump("customers_marked_absent", n)
			}
		}

		// 2. Supplied evidence (never from legacy in Slice 5).
		var sup []supplied
		var seenSupplied []string
		for _, e := range in.Evidence {
			var subj []string
			missing := false
			for _, k := range e.SubjectKeys {
				id := ""
				for _, kind := range []api.SubjectKind{api.IssuedTo, api.SyntheticSubject} {
					if v, ok := byKey[string(kind)+"/"+k]; ok {
						id = v
					}
				}
				if id == "" {
					missing = true
					break
				}
				subj = append(subj, id)
			}
			if missing {
				rep.Counts["supplied_evidence_unresolved_subject"]++
				continue
			}
			sort.Strings(subj)
			kind := domain.Catalogue[string(e.Kind)]
			var contact *string
			ref := string(e.Kind)
			if e.ContactRaw != nil {
				norm, state := domain.NormalizePhone(*e.ContactRaw)
				if state == domain.NormValid {
					contact = &norm
				}
				ref = string(e.Kind) + " on a contact"
			}
			key := e.Source.Key
			eid, isNew, err := store.RecordEvidence(ctx, tx, store.EvidenceInput{
				Key: domain.Key("SUPPLIED", string(e.Kind), e.Source.Spec, e.Source.Key, strings.Join(subj, ",")), Kind: string(e.Kind),
				Strength: kind.Strength, Polarity: kind.Polarity, Origin: "SUPPLIED", Reference: ref, ObservedRaw: e.ObservedRaw,
				System: e.Source.System, Spec: e.Source.Spec, SourceKey: &key, Subjects: subj}, in.BatchID, now)
			if err != nil {
				return err
			}
			if isNew {
				bump("evidence_supplied_new", 1)
			}
			seenSupplied = append(seenSupplied, eid)
			sup = append(sup, supplied{rec: e, subjects: subj, contact: contact})
		}
		if in.Complete {
			n, err := store.MarkAbsentEvidence(ctx, tx, "SUPPLIED", seenSupplied)
			if err != nil {
				return err
			}
			bump("evidence_supplied_marked_absent", n)
		}

		// 3. Derived evidence and candidate cases over the present model.
		snap, err := store.LoadSnapshot(ctx, tx)
		if err != nil {
			return err
		}
		cases, conflictingIDs, err := derive(ctx, tx, snap, sup, in, now, bump, func(k string, n int64) { rep.Counts[k] += n })
		if err != nil {
			return err
		}
		if n, err := store.SetIdentifierStatus(ctx, tx, conflictingIDs); err != nil {
			return err
		} else if n > 0 {
			bump("identifier_status_changed", n)
		}

		// 4. Case and customer projections.
		var seenCases []string
		statesByCustomer := map[string][]string{}
		for _, cs := range cases {
			id, ch, err := store.UpsertCase(ctx, tx, cs, runID, in.BatchID, now)
			if err != nil {
				return err
			}
			if ch {
				bump("cases_changed", 1)
			}
			seenCases = append(seenCases, id)
			rep.Counts["cases_"+strings.ToLower(cs.State)]++
			for _, m := range cs.Members {
				statesByCustomer[m] = append(statesByCustomer[m], cs.State)
			}
		}
		n, err = store.MarkAbsentCases(ctx, tx, seenCases, now)
		if err != nil {
			return err
		}
		bump("cases_marked_absent", n)
		for id, rec := range snap.Customers {
			state, reason := domain.CustomerState(rec, statesByCustomer[id])
			ch, err := store.SetCustomerState(ctx, tx, id, state, reason, runID, now)
			if err != nil {
				return err
			}
			if ch {
				bump("customer_states_changed", 1)
			}
			rep.Counts["customers_state_"+strings.ToLower(state)]++
		}

		rep.Replay = prior && changed == 0
		rep.Outcome = result.Success
		if !in.Complete || rep.Counts["supplied_evidence_unresolved_subject"] > 0 {
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
