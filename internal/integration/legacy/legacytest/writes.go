package legacytest

import (
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
)

// WritesScenario is the SYNTHETIC Slice 11 organisation and lead set: a
// conflict-free chain with an out-of-scope branch. Legacy user ids:
//
//	1 HR lead (hr, no parent)
//	2 sales manager (no parent)
//	3 senior supervisor → 2
//	4 supervisor A      → 3
//	5 supervisor B      → 3
//	6 seller X          → 4      holds lead 101
//	7 seller Y          → 4      holds lead 102
//	8 supervisor C      → 2      (outside 3's scope)
//	9 seller Z          → 8
//	10 seller W         (employment status not mapped)
//
// Leads 101 (seller 6), 102 (seller 7) and 103 (no holder). No phone, no
// names beyond synthetic labels.
func WritesScenario() []snapshot.EntityRows {
	users := []Rec{}
	for i := 1; i <= 10; i++ {
		users = append(users, Rec{"ID": fmt.Sprint(i), "user_login": fmt.Sprintf("dev-w-user-%03d", i),
			"user_email": fmt.Sprintf("dev.w.%03d@example.invalid", i), "display_name": fmt.Sprintf("Synthetic Writer %03d", i),
			"user_registered": "2025-01-01 00:00:00"})
	}
	positions := []Rec{lookup("1", "hr", "Synthetic HR"), lookup("2", "sales_manager", "Synthetic Sales Manager"),
		lookup("3", "seller", "Synthetic Seller"), lookup("4", "supervisor", "Synthetic Supervisor"),
		lookup("5", "senior_supervisor", "Synthetic Senior Supervisor")}
	profiles := []Rec{
		profile("201", "1", "Synthetic HR Lead", "1", "", "active", "sn_hr", "", "", ""),
		profile("202", "2", "Synthetic Manager", "2", "", "active", "sn_sales_manager", "", "", ""),
		profile("203", "3", "Synthetic Senior", "5", "", "active", "sn_senior_supervisor", "", "", ""),
		profile("204", "4", "Synthetic Supervisor A", "4", "", "active", "sn_supervisor", "", "", ""),
		profile("205", "5", "Synthetic Supervisor B", "4", "", "active", "sn_supervisor", "", "", ""),
		profile("206", "6", "Synthetic Seller X", "3", "", "active", "sn_seller", "", "", ""),
		profile("207", "7", "Synthetic Seller Y", "3", "", "active", "sn_seller", "", "", ""),
		profile("208", "8", "Synthetic Supervisor C", "4", "", "active", "sn_supervisor", "", "", ""),
		profile("209", "9", "Synthetic Seller Z", "3", "", "active", "sn_seller", "", "", ""),
		profile("210", "10", "Synthetic Seller W", "3", "", "synthetic_unmapped_status", "sn_seller", "", "", ""),
	}
	assignments := []Rec{
		assignment("301", "203", "202", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("302", "204", "203", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("303", "205", "203", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("304", "206", "204", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("305", "207", "204", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("306", "208", "202", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("307", "209", "208", "reports_to", "2026-01-01 00:00:00", "", "1"),
		assignment("308", "210", "204", "reports_to", "2026-01-01 00:00:00", "", "1"),
	}
	return []snapshot.EntityRows{
		rows("legacy.wp_user.v1", users), rows("legacy.wp_usermeta.v1", nil),
		rows("legacy.hr_position.v1", positions), rows("legacy.hr_level.v1", nil),
		rows("legacy.hr_profile.v1", profiles), rows("legacy.hr_position_role_mapping.v1", nil),
		rows("legacy.hr_department.v1", nil), rows("legacy.hr_unit.v1", nil), rows("legacy.hr_team.v1", nil),
		rows("legacy.hr_assignment.v1", assignments), rows("legacy.hr_assignment_history.v1", nil),
		rows("legacy.lead.v1", []Rec{Lead("101", "assigned", "6", "4"), Lead("102", "assigned", "7", "4"), Lead("103", "supervisor_pool", "", "4")}),
		rows("legacy.lead_status_history.v1", nil),
		rows("legacy.distribution_item.v1", nil), rows("legacy.distribution_item_log.v1", nil),
	}
}
