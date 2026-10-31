// Package imports implements the organization shadow import command
// (Architecture V1 §24.4). Account keys are resolved first through the
// published identity API in identity's own read-only transaction; the
// command then writes only organization tables in one organization-owned
// transaction. No transaction ever writes two modules.
//
// Each legacy profile row is SPLIT into a Person and an Employment (same
// provenance). Departments, units and teams MERGE into one typed OrgUnit
// tree; a parent that does not exist stays MISSING (never invented). The
// refresh strategy matches identity's: one native id per legacy key,
// replay writes nothing, changed content refreshes in place, rows missing
// from a COMPLETE entity are marked ABSENT_IN_LATEST; assignment history is
// append-only and a changed history row is reported, never rewritten.
// Hierarchy findings are recomputed from the stored model after the writes.
package imports

import (
	"context"
	"errors"
	"sort"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/domain"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/store"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// MaxIssues bounds listed issues (counts stay complete).
const MaxIssues = 200

// Command is the organization import command.
type Command struct {
	runner *db.Runner
	users  identityapi.UserResolver
	clock  timeutil.Clock
}

var _ api.Importer = (*Command)(nil)

// New returns the command over an organization-owned runner and the
// published identity resolver.
func New(runner *db.Runner, users identityapi.UserResolver, clock timeutil.Clock) *Command {
	return &Command{runner: runner, users: users, clock: clock}
}

// ImportOrganization applies one batch.
func (c *Command) ImportOrganization(ctx context.Context, in api.OrganizationImport) (api.ImportReport, error) {
	rep := api.ImportReport{Counts: map[string]int64{}, Findings: map[string]int64{}, Issues: []identityapi.Issue{}}
	if err := in.Validate(); err != nil {
		rep.Outcome = result.Failed
		return rep, err
	}
	// 1. Identity references, read through the published API (identity tx).
	var userKeys []string
	seen := map[string]bool{}
	for _, p := range in.Profiles {
		if p.UserKey != nil && !seen[*p.UserKey] {
			seen[*p.UserKey] = true
			userKeys = append(userKeys, *p.UserKey)
		}
	}
	sort.Strings(userKeys)
	users, err := c.users.ResolveUsers(ctx, in.UserSystem, userKeys)
	if err != nil {
		rep.Outcome = result.Failed
		return rep, err
	}
	profilesPerUser := map[string]int{}
	for _, p := range in.Profiles {
		if p.UserKey != nil {
			profilesPerUser[*p.UserKey]++
		}
	}

	now := c.clock.Now().Time()
	runID, err := ids.NewUUID()
	if err != nil {
		return rep, err
	}
	// 2. Organization writes (organization tx only).
	err = c.runner.InTx(ctx, db.TxOptions{Isolation: db.ReadCommitted}, func(ctx context.Context, tx db.Tx) error {
		rep = api.ImportReport{RunID: runID, Counts: map[string]int64{}, Findings: map[string]int64{}, Issues: []identityapi.Issue{}}
		if err := store.LockImport(ctx, tx); err != nil {
			return err
		}
		prior, err := store.PriorRun(ctx, tx, in.BatchID)
		if err != nil {
			return err
		}
		changed := int64(0)
		count := func(prefix string, ch store.Change) {
			rep.Counts[prefix+"_"+string(ch)]++
			if ch != store.Unchanged {
				changed++
			}
		}
		present := map[api.Entity][]string{}
		system := map[api.Entity]string{}
		track := func(e api.Entity, s api.SourceRef) {
			present[e] = append(present[e], s.Key)
			system[e] = s.System
		}

		for _, r := range in.Positions {
			_, ch, err := store.UpsertPosition(ctx, tx, r, now)
			if err != nil {
				return err
			}
			count("positions", ch)
			track(api.EntityPositions, r.Source)
		}
		for _, r := range in.Levels {
			_, ch, err := store.UpsertLevel(ctx, tx, r, now)
			if err != nil {
				return err
			}
			count("levels", ch)
			track(api.EntityLevels, r.Source)
		}
		// Org units, parents first: departments, then units, then teams.
		for _, kind := range []api.OrgUnitKind{api.Department, api.Unit, api.Team} {
			var parents map[string]string
			if pk, ok := kind.ParentKind(); ok {
				if parents, err = store.OrgUnitKeys(ctx, tx, pk); err != nil {
					return err
				}
			}
			ent := map[api.OrgUnitKind]api.Entity{api.Department: api.EntityDepartments, api.Unit: api.EntityUnits, api.Team: api.EntityTeams}[kind]
			for _, r := range in.OrgUnits {
				if r.Kind != kind {
					continue
				}
				_, ch, err := store.UpsertOrgUnit(ctx, tx, r, store.Resolve(r.ParentKey, parents), now)
				if err != nil {
					return err
				}
				count("org_units_"+string(ent), ch)
				track(ent, r.Source)
			}
		}

		positions, err := store.PositionKeys(ctx, tx)
		if err != nil {
			return err
		}
		levels, err := store.LevelKeys(ctx, tx)
		if err != nil {
			return err
		}
		depts, err := store.OrgUnitKeys(ctx, tx, api.Department)
		if err != nil {
			return err
		}
		unitKeys, err := store.OrgUnitKeys(ctx, tx, api.Unit)
		if err != nil {
			return err
		}
		teams, err := store.OrgUnitKeys(ctx, tx, api.Team)
		if err != nil {
			return err
		}
		for _, p := range in.Profiles {
			personID, ch, err := store.UpsertPerson(ctx, tx, p, now)
			if err != nil {
				return err
			}
			count("persons", ch)
			refs := store.EmploymentRefs{
				Position: store.Resolve(p.PositionKey, positions), Level: store.Resolve(p.LevelKey, levels),
				Department: store.Resolve(p.DepartmentKey, depts), Unit: store.Resolve(p.UnitKey, unitKeys), Team: store.Resolve(p.TeamKey, teams),
			}
			switch {
			case p.UserKey == nil:
				refs.UserLink = "USER_NOT_RECORDED"
			case users[*p.UserKey] == "":
				refs.UserLink = "USER_NOT_IMPORTED"
				rep.Counts["employments_user_not_imported"]++
				addIssue(&rep, identityapi.Issue{Kind: "PROFILE_USER_NOT_IMPORTED", SourceKey: p.Source.Key, Detail: "profile references an account that is not imported (account key " + *p.UserKey + ")"})
			default:
				uid := users[*p.UserKey]
				refs.UserID = &uid
				refs.UserLink = "LINKED"
				if profilesPerUser[*p.UserKey] > 1 {
					refs.UserLink = "SHARED_USER"
					rep.Counts["employments_shared_user"]++
				}
			}
			_, ch, err = store.UpsertEmployment(ctx, tx, p, personID, refs, now)
			if err != nil {
				return err
			}
			count("employments", ch)
			track(api.EntityProfiles, p.Source)
		}

		employments, err := store.EmploymentKeys(ctx, tx)
		if err != nil {
			return err
		}
		for _, a := range in.Assignments {
			child := store.Resolve(&a.ChildKey, employments)
			_, ch, err := store.UpsertAssignment(ctx, tx, a, child, store.Resolve(a.ParentKey, employments), now)
			if err != nil {
				return err
			}
			count("assignments", ch)
			track(api.EntityAssignments, a.Source)
		}
		var changedHistory []domain.Finding
		for _, h := range in.History {
			id, ch, differs, err := store.InsertHistory(ctx, tx, h, store.Resolve(h.ChildKey, employments),
				store.Resolve(h.OldParentKey, employments), store.Resolve(h.NewParentKey, employments), now)
			if err != nil {
				return err
			}
			count("history", ch)
			if differs {
				rep.Counts["history_changed_rejected"]++
				key := h.Source.Key
				changedHistory = append(changedHistory, domain.Finding{Kind: domain.KindHistoryChanged, Severity: domain.SevConflict,
					SubjectType: domain.SubjectHistory, SubjectID: &id, SubjectKey: &key,
					Detail: "history row " + key + " changed in the source; the stored history is append-only and was kept"})
			}
		}
		for _, d := range in.Defaults {
			_, ch, err := store.UpsertDefault(ctx, tx, d, store.Resolve(d.PositionKey, positions), store.Resolve(d.DefaultLevelKey, levels), now)
			if err != nil {
				return err
			}
			count("position_role_defaults", ch)
			if d.RoleKey == nil {
				rep.Counts["position_role_defaults_unmapped_role"]++
			}
			track(api.EntityDefaults, d.Source)
		}

		complete := true
		for _, e := range api.Entities() {
			if !in.Complete[e] {
				complete = false
				continue
			}
			if e == api.EntityHistory || system[e] == "" {
				continue
			}
			n, err := store.MarkAbsent(ctx, tx, e, system[e], present[e], now)
			if err != nil {
				return err
			}
			rep.Counts[string(e)+"_marked_absent"] = n
			changed += n
		}

		// Derived hierarchy findings over the stored (present) model.
		input, err := store.AnalysisInput(ctx, tx)
		if err != nil {
			return err
		}
		findings := append(domain.Analyze(input), changedHistory...)
		if err := store.ReplaceFindings(ctx, tx, runID, findings); err != nil {
			return err
		}
		for _, f := range findings {
			rep.Findings[f.Kind]++
		}

		rep.Replay = prior && changed == 0
		rep.Outcome = result.Success
		switch {
		case rep.Counts["history_changed_rejected"] > 0:
			rep.Outcome = result.Conflict
		case !complete || rep.Counts["employments_user_not_imported"] > 0:
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

func addIssue(rep *api.ImportReport, is identityapi.Issue) {
	if len(rep.Issues) < MaxIssues {
		rep.Issues = append(rep.Issues, is)
	}
}
