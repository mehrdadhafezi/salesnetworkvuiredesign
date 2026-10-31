package legacytest

import (
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
)

// Rec is one synthetic row by column name; absent columns are NULL.
type Rec map[string]string

// nullValue marks an explicit NULL inside a Rec.
const nullValue = "\x00NULL"

// Null is an explicit NULL for a Rec value.
var Null = nullValue

func rows(specID string, recs []Rec) snapshot.EntityRows {
	s, err := spec.Default().Lookup(specID)
	if err != nil {
		panic(err)
	}
	out := snapshot.EntityRows{SpecID: specID, Completeness: snapshot.Complete}
	for _, r := range recs {
		cells := make([]snapshot.Cell, len(s.Columns))
		for i, c := range s.Columns {
			v, ok := r[c.Name]
			if !ok || v == nullValue {
				cells[i] = N()
				continue
			}
			cells[i] = T(v)
		}
		out.Rows = append(out.Rows, cells)
	}
	return out
}

func profile(id, user, name, pos, lvl, status, role, dept, unit, team string) Rec {
	r := Rec{"id": id, "user_id": user, "hr_display_name": name, "employee_code": "SYN-E-" + id,
		"employment_status": status, "employment_type": "synthetic_full_time", "contract_status": "synthetic_contract",
		"training_status": "synthetic_training", "hire_date": "2025-01-15", "base_salary": "999000001",
		"salary_currency": "SYN", "national_id": "SYNTHETIC-NID-" + id, "work_phone": "TEST-PHONE-" + id,
		"emergency_phone": "TEST-PHONE-E" + id, "notes": "synthetic sensitive note " + id, "legacy_role": role,
		"legacy_source": "synthetic", "is_active": "1", "created_at": "2025-01-15 08:00:00", "updated_at": "2026-01-01 08:00:00"}
	for k, v := range map[string]string{"position_id": pos, "level_id": lvl, "department_id": dept, "unit_id": unit, "team_id": team} {
		if v != "" {
			r[k] = v
		}
	}
	return r
}

func assignment(id, child, parent, rel, from, to, current string) Rec {
	r := Rec{"id": id, "child_profile_id": child, "relationship_type": rel, "is_current": current, "source": "synthetic",
		"created_by": "1", "created_at": "2025-01-01 00:00:00", "updated_at": "2025-01-01 00:00:00"}
	for k, v := range map[string]string{"parent_profile_id": parent, "effective_from": from, "effective_to": to} {
		if v != "" {
			r[k] = v
		}
	}
	return r
}

func lookup(id, slug, label string) Rec {
	return Rec{"id": id, "slug": slug, "label": label, "description": "synthetic " + slug, "sort_order": id,
		"is_active": "1", "is_system": "0", "created_at": "2025-01-01 00:00:00", "updated_at": "2025-01-01 00:00:00"}
}

func unit(id, parentCol, parent, slug string) Rec {
	r := Rec{"id": id, "slug": slug, "label": "Synthetic " + slug, "is_active": "1", "sort_order": id,
		"created_at": "2025-01-01 00:00:00", "updated_at": "2025-01-01 00:00:00"}
	if parentCol != "" && parent != "" {
		r[parentCol] = parent
	}
	return r
}

// HRScenario returns a SYNTHETIC organisation that exercises every Slice 4
// path: an HR user with the HR position and role label, a manager with
// reports, a cycle, a self-parent, multiple current parents, an interval
// overlap and a history-explained gap, missing parents/children, a shared
// account, unmapped roles/statuses/relationships, an account without a
// profile and a profile without an account. Sensitive columns carry
// obviously synthetic markers so tests can prove they are never imported.
func HRScenario() []snapshot.EntityRows {
	users := []Rec{}
	for i := 1; i <= 13; i++ {
		users = append(users, Rec{"ID": fmt.Sprint(i), "user_login": fmt.Sprintf("dev-hr-user-%03d", i),
			"user_email": fmt.Sprintf("dev.hr.%03d@example.invalid", i), "display_name": fmt.Sprintf("Synthetic Account %03d", i),
			"user_registered": "2025-01-01 00:00:00"})
	}
	usermeta := []Rec{
		{"umeta_id": "501", "user_id": "1", "meta_key": "synth_capabilities", "meta_value": `a:1:{s:5:"sn_hr";b:1;}`},
		{"umeta_id": "502", "user_id": "1", "meta_key": "synth_user_level", "meta_value": "0"},
		{"umeta_id": "503", "user_id": "2", "meta_key": "synth_capabilities", "meta_value": `a:1:{s:16:"sn_sales_manager";b:1;}`},
	}
	positions := []Rec{lookup("1", "hr", "Synthetic HR"), lookup("2", "sales_manager", "Synthetic Sales Manager"),
		lookup("3", "seller", "Synthetic Seller"), lookup("4", "supervisor", "Synthetic Supervisor"), lookup("5", "converter", "Synthetic Converter")}
	positions[0]["panel_key"] = "hr"
	levels := []Rec{lookup("1", "junior", "Synthetic Junior"), lookup("2", "senior", "Synthetic Senior")}
	depts := []Rec{unit("1", "", "", "sales"), unit("2", "", "", "people")}
	units := []Rec{unit("1", "department_id", "1", "sales-east"), unit("2", "department_id", "99", "orphan-unit"), unit("3", "department_id", "", "unparented-unit")}
	teams := []Rec{unit("1", "unit_id", "1", "east-team"), unit("2", "unit_id", "77", "orphan-team")}
	profiles := []Rec{
		profile("101", "1", "Synthetic HR Lead", "1", "2", "active", "sn_hr", "2", "", ""),
		profile("102", "2", "Synthetic Manager", "2", "2", "active", "sn_sales_manager", "1", "1", ""),
		profile("103", "3", "Synthetic Seller A", "3", "1", "active", "sn_seller", "1", "1", "1"),
		profile("104", "4", "Synthetic Seller B", "3", "1", "synthetic_on_leave_x", "sn_seller", "2", "1", ""),
		profile("105", "5", "Synthetic Supervisor", "4", "1", "probation", "sn_supervisor", "1", "1", ""),
		profile("106", "7", "Synthetic Shared A", "3", "1", "active", "sn_seller", "", "", ""),
		profile("107", "7", "Synthetic Shared B", "1", "1", "active", "sn_hr", "", "", ""),
		profile("108", "8", "Synthetic Cycle A", "3", "1", "active", "sn_seller", "", "", ""),
		profile("109", "9", "Synthetic Cycle B", "3", "1", "active", "sn_seller", "", "", ""),
		profile("110", "10", "Synthetic Self Parent", "3", "1", "inactive", "seller", "", "", ""),
		profile("111", "11", "Synthetic Converter", "5", "", "active", "sn_converter", "", "", ""),
		profile("112", "99", "Synthetic No Account", "3", "1", "terminated", "sn_seller", "", "", ""),
		profile("113", "12", "Synthetic HR Position Seller Role", "1", "1", "active", "sn_seller", "", "", ""),
		profile("114", "13", "Synthetic HR Role Seller Position", "3", "1", "active", "sn_hr", "", "", ""),
	}
	profiles[11]["termination_date"] = "2026-02-01"
	profiles[2]["team_id"] = "1"
	assignments := []Rec{
		assignment("201", "103", "102", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("202", "104", "102", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("203", "104", "105", "reports_to", "2026-02-01 00:00:00", "", "1"),
		assignment("204", "105", "102", "reports_to", "2025-01-01 00:00:00", "2025-06-01 00:00:00", "0"),
		assignment("205", "105", "101", "reports_to", "2025-07-01 00:00:00", "", "1"),
		assignment("206", "108", "109", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("207", "109", "108", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("208", "110", "110", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("209", "111", "999", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("210", "888", "102", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("211", "102", "101", "synthetic_mentor_of", "2026-01-01 00:00:00", "", "1"),
		assignment("212", "103", "102", "reports_to", "2026-01-02 00:00:00", "", "1"),
		assignment("213", "114", "102", "reports_to", "2026-01-01 00:00:00", "2026-03-01 00:00:00", "1"),
		assignment("214", "112", "102", "reports_to", "0000-00-00 00:00:00", "", "1"),
	}
	history := []Rec{
		{"id": "301", "assignment_id": "205", "child_profile_id": "105", "old_parent_profile_id": "102", "new_parent_profile_id": "101",
			"actor_user_id": "1", "action": "synthetic_reassign", "reason": "synthetic reason text", "context": `{"synthetic":true}`, "created_at": "2025-07-01 00:00:00"},
		{"id": "302", "assignment_id": "201", "child_profile_id": "103", "new_parent_profile_id": "102",
			"actor_user_id": "1", "action": "synthetic_assign", "reason": "synthetic reason text", "context": `{"synthetic":true}`, "created_at": "2026-01-01 00:00:00"},
		{"id": "303", "child_profile_id": "102", "new_parent_profile_id": "105", "actor_user_id": "1", "action": "synthetic_assign",
			"created_at": "2026-04-01 00:00:00"},
	}
	mappings := []Rec{
		{"id": "401", "legacy_role": "sn_hr", "position_id": "1", "default_level_id": "2", "is_active": "1", "created_at": "2025-01-01 00:00:00", "updated_at": "2025-01-01 00:00:00"},
		{"id": "402", "legacy_role": "sn_seller", "position_id": "3", "default_level_id": "1", "is_active": "1", "created_at": "2025-01-01 00:00:00", "updated_at": "2025-01-01 00:00:00"},
		{"id": "403", "legacy_role": "sn_converter", "position_id": "5", "is_active": "1", "created_at": "2025-01-01 00:00:00", "updated_at": "2025-01-01 00:00:00"},
	}
	return []snapshot.EntityRows{
		rows("legacy.wp_user.v1", users), rows("legacy.wp_usermeta.v1", usermeta),
		rows("legacy.hr_position.v1", positions), rows("legacy.hr_level.v1", levels),
		rows("legacy.hr_profile.v1", profiles), rows("legacy.hr_position_role_mapping.v1", mappings),
		rows("legacy.hr_department.v1", depts), rows("legacy.hr_unit.v1", units), rows("legacy.hr_team.v1", teams),
		rows("legacy.hr_assignment.v1", assignments), rows("legacy.hr_assignment_history.v1", history),
	}
}
