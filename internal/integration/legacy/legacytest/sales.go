package legacytest

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"

func nullable(r Rec) Rec {
	for k, v := range r {
		if v == "" {
			r[k] = Null
		}
	}
	return r
}

// Lead builds one SYNTHETIC legacy lead row (no phone or name: not staged).
func Lead(id, status, seller, supervisor string) Rec {
	return nullable(Rec{"id": id, "marketing_form_id": "syn-form-1", "import_code": "SYN-IMPORT-" + id, "status": status,
		"lead_status": "", "destination_panel": "", "seller_id": seller, "supervisor_id": supervisor,
		"imported_at": "2025-01-01 10:00:00", "assigned_at": "2025-01-02 10:00:00", "updated_at": "2025-01-03 10:00:00"})
}

// DistItem builds one SYNTHETIC distribution item row.
func DistItem(id, owner, position, liveLead, status string) Rec {
	return nullable(Rec{"id": id, "source_type": "mis_pool", "source_id": "9" + id, "batch_id": "77", "current_owner_user_id": owner,
		"current_owner_position": position, "current_step_order": "1", "final_seller_user_id": "", "live_lead_id": liveLead,
		"conversion_status": "", "status": status, "created_at": "2025-02-01 08:00:00", "updated_at": "2025-02-05 08:00:00"})
}

// DistLog builds one SYNTHETIC distribution item log row.
func DistLog(id, item, action, from, to, toPos, at string) Rec {
	return nullable(Rec{"id": id, "item_id": item, "action": action, "actor_user_id": "3", "from_user_id": from, "to_user_id": to,
		"from_position": "", "to_position": toPos, "step_order": "1", "created_at": at})
}

// SalesScenario is a SYNTHETIC sales source (user ids match HRScenario's
// synthetic accounts 1..13 where resolvable):
//   - leads 101 (holder 5, two status history rows) and 102 (no holder);
//   - item 501: clean chain 4 → 5, legacy holder 5 → MATCH, original owner 4,
//     original seller 5;
//   - item 502: log says 6, legacy says 7 → CONFLICT;
//   - item 503: converted to lead 101 (explicit link → RELATED alias);
//   - item 504: an unverified legacy action → UNKNOWN, holder TARGET_UNKNOWN.
func SalesScenario(c snapshot.Completeness) []snapshot.EntityRows {
	ents := []snapshot.EntityRows{
		rows("legacy.lead.v1", []Rec{Lead("101", "assigned", "5", "2"), Lead("102", "synthetic_unfamiliar_status", "", "")}),
		rows("legacy.lead_status_history.v1", []Rec{
			{"id": "9001", "lead_id": "101", "user_id": "3", "old_status": Null, "new_status": "assigned", "created_at": "2025-01-02 10:00:00"},
			{"id": "9002", "lead_id": "101", "user_id": "5", "old_status": "assigned", "new_status": "callback", "created_at": "2025-01-03 10:00:00"},
		}),
		rows("legacy.distribution_item.v1", []Rec{
			DistItem("501", "5", "seller", "", "delivered_to_seller"),
			DistItem("502", "7", "seller", "", "delivered_to_seller"),
			DistItem("503", "5", "seller", "101", "live_lead_created"),
			DistItem("504", "8", "supervisor", "", "synthetic_unfamiliar_status"),
		}),
		rows("legacy.distribution_item_log.v1", []Rec{
			DistLog("1", "501", "item_created_from_mis_pool", "", "4", "supervisor", "2025-02-01 08:00:00"),
			DistLog("2", "501", "delivered_to_seller", "4", "5", "seller", "2025-02-02 08:00:00"),
			DistLog("3", "501", "seller_item_updated", "5", "5", "seller", "2025-02-03 08:00:00"),
			DistLog("4", "502", "item_created_from_mis_pool", "", "6", "seller", "2025-02-01 09:00:00"),
			DistLog("5", "503", "item_created_from_mis_pool", "", "5", "seller", "2025-02-01 10:00:00"),
			DistLog("6", "503", "live_lead_created", "5", "5", "seller", "2025-02-04 10:00:00"),
			DistLog("7", "504", "item_created_from_mis_pool", "", "4", "supervisor", "2025-02-01 11:00:00"),
			DistLog("8", "504", "synthetic_unverified_action", "4", "8", "supervisor", "2025-02-02 11:00:00"),
		}),
	}
	for i := range ents {
		ents[i].Completeness = c
	}
	return ents
}
