// Package imports implements the identity shadow import command
// (Architecture V1 §24.4): the integration connector reads legacy staging
// and calls this command through identity/api; identity writes only its own
// tables, in one identity-owned transaction. It is an offline migration
// path, never reachable from a read endpoint (architecture lint).
//
// Refresh strategy (documented contract):
//   - a legacy key maps to exactly one native user_id, minted once;
//   - the same staged row again (same checksum) changes nothing;
//   - the same legacy key with different content refreshes the shadow row in
//     place (version+1) — it never forks a second account;
//   - accounts missing from a COMPLETE batch are marked ABSENT_IN_LATEST,
//     never deleted;
//   - role evidence becomes SHADOW grants only; evidence for an account that
//     is not imported is reported, never invented.
package imports

import (
	"context"
	"errors"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// MaxIssues bounds the issues listed in a report (counts stay complete).
const MaxIssues = 200

// Command is the identity import command.
type Command struct {
	runner *db.Runner
	clock  timeutil.Clock
}

var _ api.Importer = (*Command)(nil)

// New returns the command over an identity-owned runner.
func New(runner *db.Runner, clock timeutil.Clock) *Command {
	return &Command{runner: runner, clock: clock}
}

// ImportIdentity applies one batch in one identity transaction.
func (c *Command) ImportIdentity(ctx context.Context, in api.IdentityImport) (api.ImportReport, error) {
	rep := api.ImportReport{Counts: map[string]int64{}, Issues: []api.Issue{}}
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
		rep = api.ImportReport{RunID: runID, Counts: map[string]int64{}, Issues: []api.Issue{}}
		if err := store.LockImport(ctx, tx); err != nil {
			return err
		}
		prior, err := store.PriorRun(ctx, tx, in.BatchID)
		if err != nil {
			return err
		}
		changed := int64(0)
		userIDs := map[string]string{}
		var keys []string
		system := ""
		for _, u := range in.Users {
			id, ch, err := store.UpsertUser(ctx, tx, u, now)
			if err != nil {
				return err
			}
			userIDs[u.Source.Key] = id
			keys = append(keys, u.Source.Key)
			system = u.Source.System
			rep.Counts["users_"+string(ch)]++
			if ch != store.Unchanged {
				changed++
			}
		}
		rep.Counts["users_staged"] = int64(len(in.Users))
		if in.UsersComplete && system != "" {
			n, err := store.MarkAbsentUsers(ctx, tx, system, keys, now)
			if err != nil {
				return err
			}
			rep.Counts["users_marked_absent"] = n
			changed += n
		}

		// Role evidence: resolve accounts (this batch, then previously imported).
		var missing []string
		for _, e := range in.RoleEvidence {
			if _, ok := userIDs[e.UserKey]; !ok {
				missing = append(missing, e.UserKey)
			}
		}
		if len(missing) > 0 && system != "" {
			found, err := store.UserIDsByKey(ctx, tx, system, missing)
			if err != nil {
				return err
			}
			for k, v := range found {
				userIDs[k] = v
			}
		}
		var evidenceKeys []string
		for _, e := range in.RoleEvidence {
			evidenceKeys = append(evidenceKeys, e.Evidence.Key)
			uid, ok := userIDs[e.UserKey]
			if !ok {
				rep.Counts["role_evidence_user_missing"]++
				addIssue(&rep, api.Issue{Kind: "ROLE_EVIDENCE_USER_MISSING", SourceKey: e.Evidence.Key, Detail: "role evidence references an account that is not imported (account key " + e.UserKey + ")"})
				continue
			}
			ch, err := store.UpsertShadowGrant(ctx, tx, e, uid, now)
			if err != nil {
				return err
			}
			rep.Counts["shadow_grants_"+string(ch)]++
			if e.RoleKey == nil {
				rep.Counts["shadow_grants_unmapped_role"]++
			}
			if ch != store.Unchanged {
				changed++
			}
		}
		rep.Counts["role_evidence_staged"] = int64(len(in.RoleEvidence))
		if in.EvidenceComplete {
			n, err := store.MarkAbsentGrants(ctx, tx, evidenceKeys, now)
			if err != nil {
				return err
			}
			rep.Counts["shadow_grants_marked_absent"] = n
			changed += n
		}

		rep.Replay = prior && changed == 0
		rep.Outcome = result.Success
		if rep.Counts["role_evidence_user_missing"] > 0 || !in.UsersComplete || !in.EvidenceComplete {
			rep.Outcome = result.Partial
		}
		return store.InsertRun(ctx, tx, runID, in.BatchID, in.SnapshotID, rep.Outcome.String(), rep.Replay, rep.Counts, now)
	})
	if err != nil {
		rep.Outcome = result.Failed
		if isUnknown(err) {
			rep.Outcome = result.Unknown
		}
		return rep, err
	}
	sort.Slice(rep.Issues, func(i, j int) bool { return rep.Issues[i].SourceKey < rep.Issues[j].SourceKey })
	return rep, nil
}

func addIssue(rep *api.ImportReport, is api.Issue) {
	if len(rep.Issues) < MaxIssues {
		rep.Issues = append(rep.Issues, is)
	}
}

func isUnknown(err error) bool {
	return err != nil && errors.Is(err, db.ErrCommitOutcomeUnknown)
}
