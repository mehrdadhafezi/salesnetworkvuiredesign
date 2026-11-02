package legacytest

import (
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
)

// Membership builds one SYNTHETIC legacy project membership (no customer
// name or phone column is staged at all).
func Membership(id, invoice, status, dot string) Rec {
	return nullable(Rec{"id": id, "source_key": "syn:" + id, "source_type": "dot", "source_invoice_id": invoice, "dot_case_id": dot,
		"subscription_product_id": "701", "subscription_name_snapshot": "Synthetic Bundle " + id, "customer_wp_id": "", "origin_user_id": "5",
		"first_payment_invoice_id": invoice, "status": status, "activated_at": "2025-03-04 10:00:00", "created_at": "2025-03-04 10:00:00",
		"updated_at": "2025-03-05 10:00:00"})
}

// MembershipItem builds one SYNTHETIC membership item.
func MembershipItem(id, membership, product, title, status string) Rec {
	return nullable(Rec{"id": id, "membership_id": membership, "content_product_id": product, "content_name_snapshot": title, "manager_user_id": "4",
		"expert_user_id": "5", "workflow_status": status, "assigned_at": "2025-03-04 11:00:00", "status_updated_at": "2025-03-05 11:00:00",
		"created_at": "2025-03-04 10:00:00", "updated_at": "2025-03-05 11:00:00"})
}

// BiawinScenario is a SYNTHETIC Biawin / fulfillment source:
//   - 501 active (invoice 2001) with 12 items 601..612, two of them with the
//     same title (never merged);
//   - 502 suspended (invoice 2002) whose item 614 has a COMPLETED execution
//     case (it must not activate anything);
//   - 503 an unknown raw status (never ACTIVE);
//   - 504 active on invoice 2999 (not in billing: customer UNKNOWN) with a
//     raw Dot case ref;
//   - 505 active on invoice 2003 (another customer);
//   - project events incl. a chat message (MESSAGING_ONLY) and provider
//     evidence; an upgrade action; execution cases, events and a form
//     submission; shipping orders and a log.
func BiawinScenario(c snapshot.Completeness) []snapshot.EntityRows {
	items := []Rec{}
	for i := 1; i <= 12; i++ {
		title := fmt.Sprintf("Synthetic Service %02d", i)
		if i == 3 || i == 4 {
			title = "Synthetic Service A"
		}
		status := []string{"waiting", "no_answer", "follow_up", "normal", "purchased", "cancelled"}[i%6]
		if i == 12 {
			status = "synthetic_wf_state"
		}
		items = append(items, MembershipItem(fmt.Sprint(600+i), "501", fmt.Sprint(800+i), title, status))
	}
	items = append(items, MembershipItem("613", "504", "813", "Synthetic Service 13", "waiting"),
		MembershipItem("614", "502", "814", "Synthetic Service 14", "normal"))
	ents := []snapshot.EntityRows{
		rows("legacy.project_membership.v1", []Rec{
			Membership("501", "2001", "active", ""), Membership("502", "2002", "suspended", ""), Membership("503", "2004", "synthetic_paused", ""),
			Membership("504", "2999", "active", "77"), Membership("505", "2003", "active", ""),
		}),
		rows("legacy.project_membership_item.v1", items),
		rows("legacy.project_event.v1", []Rec{
			nullable(Rec{"id": "901", "membership_id": "501", "membership_item_id": "", "actor_user_id": "5", "event_key": "membership_activated", "created_at": "2025-03-04 10:00:00"}),
			nullable(Rec{"id": "902", "membership_id": "501", "membership_item_id": "601", "actor_user_id": "4", "event_key": "workflow_status_changed", "created_at": "2025-03-05 11:00:00"}),
			nullable(Rec{"id": "903", "membership_id": "501", "membership_item_id": "601", "actor_user_id": "4", "event_key": "chat_message_sent", "created_at": "2025-03-05 12:00:00"}),
			nullable(Rec{"id": "904", "membership_id": "501", "membership_item_id": "602", "actor_user_id": "", "event_key": "project_remote_wallet_charged", "created_at": "2025-03-05 13:00:00"}),
			nullable(Rec{"id": "905", "membership_id": "502", "membership_item_id": "", "actor_user_id": "", "event_key": "membership_suspended_finance", "created_at": "2025-03-06 09:00:00"}),
		}),
		rows("legacy.project_invoice_link.v1", []Rec{
			nullable(Rec{"id": "951", "membership_item_id": "601", "invoice_id": "2001", "content_product_id": "801", "status": "verified", "verified_at": "2025-03-04 12:00:00", "created_at": "2025-03-04 12:00:00"}),
		}),
		rows("legacy.project_action.v1", []Rec{
			nullable(Rec{"id": "961", "membership_item_id": "602", "actor_user_id": "5", "action_type": "upgrade", "fulfillment_mode": "remote", "target_product_id": "820",
				"status": "remote_failed", "invoice_id": "2002", "created_at": "2025-03-05 10:00:00", "updated_at": "2025-03-05 10:30:00"}),
			nullable(Rec{"id": "962", "membership_item_id": "603", "actor_user_id": "5", "action_type": "upsell", "fulfillment_mode": "manual", "target_product_id": "",
				"status": "synthetic_action_state", "invoice_id": "", "created_at": "2025-03-05 10:00:00", "updated_at": "2025-03-05 10:30:00"}),
		}),
		rows("legacy.operations_execution_case.v1", []Rec{
			execCase("1001", "601", "wallet_charge", "completed", "SYN-EXT-0001", "", "2025-03-06 10:00:00"),
			execCase("1002", "602", "form", "form_completed", "", "", ""),
			execCase("1003", "605", "activation_code", "synthetic_exec_state", "", "", ""),
			execCase("1004", "614", "wallet_charge", "completed", "SYN-EXT-0004", "", "2025-03-06 11:00:00"),
		}),
		rows("legacy.operations_execution_event.v1", []Rec{
			nullable(Rec{"id": "1101", "execution_case_id": "1001", "operation_id": "1001", "membership_item_id": "601", "actor_user_id": "4", "event_key": "case_assigned", "created_at": "2025-03-05 10:00:00"}),
			nullable(Rec{"id": "1102", "execution_case_id": "1001", "operation_id": "1001", "membership_item_id": "601", "actor_user_id": "5", "event_key": "case_completed", "created_at": "2025-03-06 10:00:00"}),
		}),
		rows("legacy.operations_form_submission.v1", []Rec{
			nullable(Rec{"id": "1201", "execution_case_id": "1002", "form_definition_id": "3", "submitted_by_user_id": "5", "submitted_at": "2025-03-06 09:00:00",
				"final_decision": "executed", "decided_by_user_id": "4", "decided_at": "2025-03-06 09:30:00"}),
		}),
		rows("legacy.shipping_order.v1", []Rec{
			shipOrder("1301", "2001", "shipped", "SYN-TRK-0001"),
			shipOrder("1302", "2004", "synthetic_ship_state", ""),
		}),
		rows("legacy.shipping_order_log.v1", []Rec{
			nullable(Rec{"id": "1401", "shipping_order_id": "1301", "invoice_id": "2001", "actor_user_id": "5", "event_key": "status_changed",
				"old_status": "preparing", "new_status": "shipped", "created_at": "2025-03-07 10:00:00"}),
		}),
	}
	for i := range ents {
		ents[i].Completeness = c
	}
	return ents
}

func execCase(id, item, typ, status, ext, tracking, completed string) Rec {
	return nullable(Rec{"id": id, "operation_id": id, "membership_item_id": item, "content_product_id": "801", "fulfillment_type": typ,
		"manager_user_id": "4", "expert_user_id": "5", "assigned_by_user_id": "4", "assigned_at": "2025-03-05 10:00:00", "status": status,
		"wallet_status": "", "wallet_external_id": ext, "shipping_status": "", "shipping_carrier": "", "shipping_tracking_code": tracking,
		"form_submission_id": "", "final_decision": "", "completed_at": completed, "completed_by_user_id": "", "cancelled_at": "",
		"created_at": "2025-03-05 10:00:00", "updated_at": "2025-03-06 10:00:00"})
}

func shipOrder(id, invoice, status, tracking string) Rec {
	return nullable(Rec{"id": id, "invoice_id": invoice, "seller_id": "5", "customer_wp_id": "", "contact_status": "contacted",
		"fulfillment_status": status, "shipping_method": "post", "carrier": "Synthetic Post", "tracking_code": tracking, "handled_by": "5",
		"shipped_at": "2025-03-07 10:00:00", "delivered_at": "", "created_at": "2025-03-06 10:00:00", "updated_at": "2025-03-07 10:00:00"})
}

// BiawinEntity builds one Slice 10 entity from rows.
func BiawinEntity(specID string, recs []Rec) snapshot.EntityRows { return rows(specID, recs) }
