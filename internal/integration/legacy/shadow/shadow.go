package shadow

import (
	"context"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	organizationapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/api"
)

func src(b *Batch, specID string, r Row) identityapi.SourceRef {
	return identityapi.SourceRef{System: SourceSystem, Spec: specID, Key: r.Key, BatchID: b.BatchID, RowSHA256: r.SHA}
}

// IdentityCommand builds the identity import command from staged rows.
func IdentityCommand(b *Batch) (identityapi.IdentityImport, map[string]int64) {
	counts := map[string]int64{}
	users := b.Entity(SpecUser)
	profiles := b.Entity(SpecProfile)
	in := identityapi.IdentityImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID,
		UsersComplete: users.Complete(), EvidenceComplete: profiles.Complete()}
	for _, r := range users.Rows {
		// user_email and every credential column are not imported.
		in.Users = append(in.Users, identityapi.UserRecord{Source: src(b, SpecUser, r),
			Login: text(r.Get("user_login")), DisplayName: text(r.Get("display_name")), RegisteredRaw: text(r.Get("user_registered"))})
	}
	for _, r := range profiles.Rows {
		role := r.Get("legacy_role")
		if role == nil || *role == "" {
			counts["profiles_without_role_evidence"]++
			continue
		}
		userKey, bad := key(r.Get("user_id"))
		if userKey == nil {
			if bad {
				counts["role_evidence_unreadable_user_reference"]++
			} else {
				counts["role_evidence_without_user"]++
			}
			continue
		}
		value := *role
		if len(value) > 120 {
			counts["role_evidence_too_long"]++
			continue
		}
		in.RoleEvidence = append(in.RoleEvidence, identityapi.RoleEvidence{Evidence: src(b, SpecProfile, r), UserKey: *userKey,
			LegacyValue: value, RoleKey: MapLegacyRole(value)})
	}
	return in, counts
}

// OrganizationCommand builds the organization import command.
func OrganizationCommand(b *Batch) (organizationapi.OrganizationImport, map[string]int64) {
	counts := map[string]int64{}
	ref := func(raw *string, what string) *string {
		k, bad := key(raw)
		if bad {
			counts["unreadable_reference_"+what]++
		}
		return k
	}
	in := organizationapi.OrganizationImport{BatchID: b.BatchID, SnapshotID: b.SnapshotID, UserSystem: SourceSystem,
		Complete: map[organizationapi.Entity]bool{}}
	lookup := func(specID string) []organizationapi.LookupRecord {
		var out []organizationapi.LookupRecord
		for _, r := range b.Entity(specID).Rows {
			out = append(out, organizationapi.LookupRecord{Source: src(b, specID, r), Slug: text(r.Get("slug")), Label: text(r.Get("label")),
				Description: text(r.Get("description")), SortOrder: integer(r.Get("sort_order")), IsActive: flag(r.Get("is_active")),
				IsSystem: flag(r.Get("is_system")), CreatedRaw: text(r.Get("created_at")), UpdatedRaw: text(r.Get("updated_at"))})
		}
		return out
	}
	in.Positions, in.Levels = lookup(SpecPosition), lookup(SpecLevel)
	units := []struct {
		spec, parentCol string
		kind            organizationapi.OrgUnitKind
	}{{SpecDepartment, "", organizationapi.Department}, {SpecUnit, "department_id", organizationapi.Unit}, {SpecTeam, "unit_id", organizationapi.Team}}
	for _, u := range units {
		for _, r := range b.Entity(u.spec).Rows {
			rec := organizationapi.OrgUnitRecord{Kind: u.kind, Source: src(b, u.spec, r), Slug: text(r.Get("slug")), Label: text(r.Get("label")),
				Description: text(r.Get("description")), SortOrder: integer(r.Get("sort_order")), IsActive: flag(r.Get("is_active")),
				CreatedRaw: text(r.Get("created_at")), UpdatedRaw: text(r.Get("updated_at"))}
			if u.parentCol != "" {
				rec.ParentKey = ref(r.Get(u.parentCol), "org_unit_parent")
			}
			in.OrgUnits = append(in.OrgUnits, rec)
		}
	}
	// Profile → Person + Employment. Not imported: national_id, base_salary,
	// salary_currency, work_phone, emergency_phone, notes, contract_status,
	// training_status, legacy_source (raw evidence stays in staging).
	for _, r := range b.Entity(SpecProfile).Rows {
		in.Profiles = append(in.Profiles, organizationapi.ProfileRecord{Source: src(b, SpecProfile, r),
			UserKey: ref(r.Get("user_id"), "user"), DisplayName: text(r.Get("hr_display_name")), EmployeeCode: text(r.Get("employee_code")),
			Status: MapEmploymentStatus(r.Get("employment_status")), StatusRaw: text(r.Get("employment_status")),
			TypeRaw: text(r.Get("employment_type")), HireDateRaw: text(r.Get("hire_date")), TerminationDateRaw: text(r.Get("termination_date")),
			IsActive: flag(r.Get("is_active")), PositionKey: ref(r.Get("position_id"), "position"), LevelKey: ref(r.Get("level_id"), "level"),
			DepartmentKey: ref(r.Get("department_id"), "department"), UnitKey: ref(r.Get("unit_id"), "unit"), TeamKey: ref(r.Get("team_id"), "team"),
			CreatedRaw: text(r.Get("created_at")), UpdatedRaw: text(r.Get("updated_at"))})
	}
	for _, r := range b.Entity(SpecAssignment).Rows {
		child := ref(r.Get("child_profile_id"), "assignment_child")
		if child == nil {
			counts["assignments_without_child"]++
			continue
		}
		in.Assignments = append(in.Assignments, organizationapi.AssignmentRecord{Source: src(b, SpecAssignment, r), ChildKey: *child,
			ParentKey: ref(r.Get("parent_profile_id"), "assignment_parent"), Kind: MapRelationship(r.Get("relationship_type")),
			KindRaw: text(r.Get("relationship_type")), EffectiveFromRaw: text(r.Get("effective_from")), EffectiveToRaw: text(r.Get("effective_to")),
			IsCurrent: flag(r.Get("is_current")), CurrentRaw: text(r.Get("is_current")), OriginRaw: text(r.Get("source")),
			RecordedByKey: ref(r.Get("created_by"), "assignment_recorded_by"), CreatedRaw: text(r.Get("created_at")), UpdatedRaw: text(r.Get("updated_at"))})
	}
	// History: reason and context (free text) are not imported.
	for _, r := range b.Entity(SpecAssignmentHist).Rows {
		in.History = append(in.History, organizationapi.HistoryRecord{Source: src(b, SpecAssignmentHist, r),
			AssignmentKey: ref(r.Get("assignment_id"), "history_assignment"), ChildKey: ref(r.Get("child_profile_id"), "history_child"),
			OldParentKey: ref(r.Get("old_parent_profile_id"), "history_old_parent"), NewParentKey: ref(r.Get("new_parent_profile_id"), "history_new_parent"),
			ActorUserKey: ref(r.Get("actor_user_id"), "history_actor"), ActionRaw: text(r.Get("action")), RecordedRaw: text(r.Get("created_at"))})
	}
	for _, r := range b.Entity(SpecRoleMapping).Rows {
		var role *identityapi.Role
		if v := r.Get("legacy_role"); v != nil {
			role = MapLegacyRole(*v)
		}
		in.Defaults = append(in.Defaults, organizationapi.PositionRoleDefaultRecord{Source: src(b, SpecRoleMapping, r),
			PositionKey: ref(r.Get("position_id"), "default_position"), LegacyRole: text(r.Get("legacy_role")), RoleKey: role,
			DefaultLevelKey: ref(r.Get("default_level_id"), "default_level"), IsActive: flag(r.Get("is_active"))})
	}
	for e, specID := range map[organizationapi.Entity]string{
		organizationapi.EntityPositions: SpecPosition, organizationapi.EntityLevels: SpecLevel,
		organizationapi.EntityDepartments: SpecDepartment, organizationapi.EntityUnits: SpecUnit, organizationapi.EntityTeams: SpecTeam,
		organizationapi.EntityProfiles: SpecProfile, organizationapi.EntityAssignments: SpecAssignment,
		organizationapi.EntityHistory: SpecAssignmentHist, organizationapi.EntityDefaults: SpecRoleMapping,
	} {
		in.Complete[e] = b.Entity(specID).Complete() && (e != organizationapi.EntityAssignments || counts["assignments_without_child"] == 0)
	}
	return in, counts
}

// --- reconciliation report ---------------------------------------------------

// Section is one module section of the reconciliation report.
type Section struct {
	Outcome result.Status      `json:"outcome"`
	Reasons []reasoncodes.Code `json:"reasons"`
	Staged  map[string]int64   `json:"staged"`
	Target  map[string]int64   `json:"target"`
	Notes   []string           `json:"notes"`
}

// DiffRow is one non-MATCH capability comparison (keys only, no PII).
type DiffRow struct {
	UserKey           string                      `json:"legacy_user_key"`
	UserID            *string                     `json:"user_id"`
	Capability        identityapi.Capability      `json:"capability"`
	Legacy            authorization.LegacyVerdict `json:"legacy"`
	LegacyBasis       string                      `json:"legacy_basis"`
	Target            authorization.TargetVerdict `json:"target"`
	Class             authorization.DiffClass     `json:"class"`
	SecuritySensitive bool                        `json:"security_sensitive"`
}

// Authorization is the authorization section with the capability diff.
type Authorization struct {
	Section
	Coverage             string                            `json:"coverage"`
	LegacyGate           string                            `json:"legacy_gate"`
	Comparison           map[authorization.DiffClass]int64 `json:"comparison"`
	TargetMorePermissive int64                             `json:"target_more_permissive"`
	SecurityRows         []DiffRow                         `json:"security_sensitive_rows"`
	Rows                 []DiffRow                         `json:"rows"`
}

// Report is the machine-readable Slice 4 reconciliation report.
type Report struct {
	Command       string        `json:"command"`
	SnapshotID    string        `json:"snapshot_id"`
	BatchID       string        `json:"batch_id"`
	Outcome       result.Status `json:"outcome"`
	ConsumedSpecs []string      `json:"consumed_specs"`
	Identity      Section       `json:"identity"`
	Organization  Section       `json:"organization"`
	Authorization Authorization `json:"authorization"`
}

// MaxRows bounds listed diff rows (counts stay complete).
const MaxRows = 200

func nonNil(m map[string]int64) map[string]int64 {
	if m == nil {
		return map[string]int64{}
	}
	return m
}

// Reconcile builds the report from staging and the published read APIs. It
// reads only; it never writes any module.
func Reconcile(ctx context.Context, b *Batch, ident identityapi.Reader, org organizationapi.Reader) (Report, error) {
	rep := Report{Command: "reconcile-identity-org", SnapshotID: b.SnapshotID, BatchID: b.BatchID, ConsumedSpecs: ConsumedSpecs()}

	// Identity.
	users := b.Entity(SpecUser)
	userKeys := make([]string, 0, len(users.Rows))
	for _, r := range users.Rows {
		userKeys = append(userKeys, r.Key)
	}
	resolved, err := ident.ResolveUsers(ctx, SourceSystem, userKeys)
	if err != nil {
		return rep, err
	}
	icounts, err := ident.Counts(ctx)
	if err != nil {
		return rep, err
	}
	profiles := b.Entity(SpecProfile)
	profilesByUser := map[string][]Row{}
	orphanProfiles := int64(0)
	stagedUsers := map[string]bool{}
	for _, k := range userKeys {
		stagedUsers[k] = true
	}
	for _, r := range profiles.Rows {
		if k, _ := key(r.Get("user_id")); k != nil {
			profilesByUser[*k] = append(profilesByUser[*k], r)
			if !stagedUsers[*k] {
				orphanProfiles++
			}
		} else {
			orphanProfiles++
		}
	}
	usersWithoutProfile, usersMultiProfile := int64(0), int64(0)
	for _, k := range userKeys {
		switch n := len(profilesByUser[k]); {
		case n == 0:
			usersWithoutProfile++
		case n > 1:
			usersMultiProfile++
		}
	}
	unresolved := int64(len(userKeys) - len(resolved))
	rep.Identity = Section{Reasons: []reasoncodes.Code{}, Notes: []string{},
		Staged: map[string]int64{"wp_user_rows": int64(len(users.Rows)), "users_without_hr_profile": usersWithoutProfile,
			"users_with_multiple_hr_profiles": usersMultiProfile, "hr_profiles_referencing_missing_user": orphanProfiles},
		Target: nonNil(icounts)}
	rep.Identity.Target["staged_users_imported"] = int64(len(resolved))
	rep.Identity.Target["staged_users_unresolved"] = unresolved
	rep.Identity.Notes = append(rep.Identity.Notes, "no credential, session or login is materialised; login_enabled is false for every account",
		"user_email and credential columns are not imported; phone, e-mail and login are never keys")
	switch {
	case !users.Usable():
		rep.Identity.Outcome = result.Unknown
		rep.Identity.Reasons = append(rep.Identity.Reasons, reasoncodes.SourceUnavailable)
	case unresolved > 0 || !users.Complete():
		rep.Identity.Outcome = result.Partial
		rep.Identity.Reasons = append(rep.Identity.Reasons, reasoncodes.CoverageIncomplete)
	case usersMultiProfile > 0:
		rep.Identity.Outcome = result.Conflict
		rep.Identity.Reasons = append(rep.Identity.Reasons, reasoncodes.LinkageConflict)
	default:
		rep.Identity.Outcome = result.Success
	}

	// Organization.
	ocounts, err := org.Counts(ctx)
	if err != nil {
		return rep, err
	}
	staged := map[string]int64{}
	for name, specID := range map[string]string{"hr_profile_rows": SpecProfile, "hr_position_rows": SpecPosition, "hr_level_rows": SpecLevel,
		"hr_department_rows": SpecDepartment, "hr_unit_rows": SpecUnit, "hr_team_rows": SpecTeam, "hr_assignment_rows": SpecAssignment,
		"hr_assignment_history_rows": SpecAssignmentHist, "hr_position_role_mapping_rows": SpecRoleMapping} {
		staged[name] = int64(len(b.Entity(specID).Rows))
	}
	rep.Organization = Section{Reasons: []reasoncodes.Code{}, Staged: staged, Target: nonNil(ocounts),
		Notes: []string{"each hr profile row is split into one Person and one Employment",
			"not imported: national_id, base_salary, salary_currency, work_phone, emergency_phone, notes, contract_status, training_status, history reason/context",
			"hierarchy findings are reported, never auto-fixed"}}
	conflicts, unknowns := int64(0), int64(0)
	for k, n := range ocounts {
		if len(k) > 8 && k[:8] == "finding." {
			switch k[8:] {
			case "ORG_UNIT_PARENT_NOT_RECORDED", "UNREADABLE_DATE", "UNMAPPED_RELATIONSHIP", "USER_NOT_IMPORTED", "HISTORY_GAP", "UNEXPLAINED_GAP":
				unknowns += n
			default:
				conflicts += n
			}
		}
	}
	countMismatch := staged["hr_profile_rows"] != ocounts["persons"] || staged["hr_profile_rows"] != ocounts["employments"] ||
		staged["hr_position_rows"] != ocounts["positions"] || staged["hr_level_rows"] != ocounts["levels"] ||
		staged["hr_department_rows"] != ocounts["departments"] || staged["hr_unit_rows"] != ocounts["units"] ||
		staged["hr_team_rows"] != ocounts["teams"]
	switch {
	case !profiles.Usable():
		rep.Organization.Outcome = result.Unknown
		rep.Organization.Reasons = append(rep.Organization.Reasons, reasoncodes.SourceUnavailable)
	case conflicts > 0:
		rep.Organization.Outcome = result.Conflict
		rep.Organization.Reasons = append(rep.Organization.Reasons, reasoncodes.LinkageConflict)
	case countMismatch || unknowns > 0:
		rep.Organization.Outcome = result.Partial
		rep.Organization.Reasons = append(rep.Organization.Reasons, reasoncodes.CoverageIncomplete)
	default:
		rep.Organization.Outcome = result.Success
	}

	// Authorization: capability diff.
	grants, err := ident.Grants(ctx)
	if err != nil {
		return rep, err
	}
	contexts, err := org.SubjectContexts(ctx)
	if err != nil {
		return rep, err
	}
	grantsByUser := map[string][]identityapi.Grant{}
	for _, g := range grants {
		grantsByUser[g.UserID] = append(grantsByUser[g.UserID], g)
	}
	positionSlug := map[string]*string{}
	for _, r := range b.Entity(SpecPosition).Rows {
		positionSlug[r.Key] = r.Get("slug")
	}
	roleEvidence := int64(0)
	for _, r := range profiles.Rows {
		if v := r.Get("legacy_role"); v != nil && *v != "" {
			roleEvidence++
		}
	}
	az := Authorization{Coverage: "INCOMPLETE", LegacyGate: LegacyHRGate,
		Comparison: map[authorization.DiffClass]int64{}, SecurityRows: []DiffRow{}, Rows: []DiffRow{}}
	for _, c := range authorization.DiffClasses() {
		az.Comparison[c] = 0
	}
	sort.Strings(userKeys)
	for _, uk := range userKeys {
		ps := profilesByUser[uk]
		legacy, basis := authorization.LegacyUnknown, "WordPress role assignments not staged"
		switch {
		case len(ps) > 1:
			legacy, basis = authorization.LegacyConflict, "account linked to several hr profiles; legacy reads one with LIMIT 1 and no order"
		case len(ps) == 1:
			if pk, _ := key(ps[0].Get("position_id")); pk != nil {
				if slug := positionSlug[*pk]; slug != nil && *slug == hrPositionSlug {
					legacy, basis = authorization.LegacyAllowed, "hr profile position slug grants the legacy HR panel"
				}
			}
		}
		uid, imported := resolved[uk]
		subject := authorization.Subject{PrincipalID: uid, Grants: grantsByUser[uid], EvidenceConflict: len(ps) > 1}
		if sc, ok := contexts[uid]; ok {
			subject.Org = &authorization.OrgContext{PositionSlug: sc.PositionSlug, LevelSlug: sc.LevelSlug, IsManager: sc.Reports > 0, SubordinateCount: sc.Reports}
		}
		unmapped := false
		for _, g := range subject.Grants {
			if g.RoleKey == nil {
				unmapped = true
			}
		}
		for _, cd := range identityapi.Capabilities() {
			target := authorization.TargetUnknown
			if imported {
				target = authorization.TargetVerdictOf(authorization.Evaluate(authorization.Shadow, subject, authorization.Request{Capability: cd.Key}))
			}
			class, sensitive := authorization.Classify(legacy, target, unmapped)
			az.Comparison[class]++
			if class == authorization.DiffMatch && !sensitive {
				continue
			}
			row := DiffRow{UserKey: uk, Capability: cd.Key, Legacy: legacy, LegacyBasis: basis, Target: target, Class: class, SecuritySensitive: sensitive}
			if imported {
				id := uid
				row.UserID = &id
			}
			if sensitive {
				az.TargetMorePermissive++
				if len(az.SecurityRows) < MaxRows {
					az.SecurityRows = append(az.SecurityRows, row)
				}
			}
			if len(az.Rows) < MaxRows {
				az.Rows = append(az.Rows, row)
			}
		}
	}
	az.Section = Section{Reasons: []reasoncodes.Code{reasoncodes.CoverageIncomplete},
		Staged: map[string]int64{"wp_usermeta_rows": b.UsermetaRows, "wp_usermeta_capability_rows": b.UsermetaCapabilityRows,
			"hr_profile_role_evidence_rows": roleEvidence, "hr_position_role_mapping_rows": staged["hr_position_role_mapping_rows"]},
		Target: map[string]int64{"roles": icounts["roles"], "capabilities": icounts["capabilities"], "role_grants": icounts["role_grants"],
			"role_grants_mapped": icounts["role_grants_mapped"], "role_grants_unmapped": icounts["role_grants_unmapped"],
			"role_grants_active": icounts["role_grants_active"], "position_role_defaults": ocounts["position_role_defaults"],
			"position_role_defaults_mapped": ocounts["position_role_defaults_mapped"]},
		Notes: []string{"WordPress role assignments live in wp_usermeta meta_value, which is never parsed (row scope is an open Owner decision): legacy role/manage_options branches are UNKNOWN",
			"role grants are SHADOW grants from the hr profile legacy role label; they never authorise a runtime request",
			"position-role mappings are imported as PositionRoleDefault evidence only, never as grants",
			"rows where the target allows and legacy does not prove allowed are security-sensitive; nothing is broadened to match legacy"}}
	az.Outcome = result.Partial
	if az.Comparison[authorization.DiffConflict] > 0 {
		az.Outcome = result.Conflict
		az.Reasons = append(az.Reasons, reasoncodes.LinkageConflict)
	}
	rep.Authorization = az

	agg, err := result.Aggregate([]result.Status{rep.Identity.Outcome, rep.Organization.Outcome, rep.Authorization.Outcome})
	if err != nil {
		return rep, err
	}
	rep.Outcome = agg
	return rep, nil
}
