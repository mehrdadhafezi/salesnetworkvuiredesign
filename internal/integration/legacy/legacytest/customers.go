package legacytest

import "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"

// IssuedTo builds one SYNTHETIC legacy invoice "issued-to" row. Phones use
// reserved synthetic numbers (09000000xxx); names are labelled synthetic.
func IssuedTo(id, wpID, name, phone, phone2, created string) Rec {
	r := Rec{"id": id, "invoice_code": "SYN-INV-" + id, "customer_wp_id": wpID, "customer_name": name,
		"customer_phone": phone, "customer_phone_secondary": phone2, "created_at": created}
	for k, v := range r {
		if v == "" && k != "customer_name" && k != "customer_phone" {
			r[k] = Null
		}
	}
	return r
}

// CustomerScenario is a SYNTHETIC issued-to source:
//   - 1001 and 1002 share one phone and one phone-derived account (legacy
//     behaviour: the account is found by phone) → RELATED, never RESOLVED;
//   - 1003 shares only the phone, in another notation, with 1001 → a
//     phone-only candidate (UNKNOWN), never merged;
//   - 1004 has a distinct phone and no account;
//   - 1005 has neither name nor phone → NOT_RECOVERABLE (kept, not dropped).
func CustomerScenario(c snapshot.Completeness) []snapshot.EntityRows {
	e := rows("legacy.invoice_issued_to.v1", []Rec{
		IssuedTo("1001", "71", "Synthetic Customer A", "09000000101", "", "2025-02-01 10:00:00"),
		IssuedTo("1002", "71", "Synthetic Customer A2", "09000000101", "", "2025-03-01 10:00:00"),
		IssuedTo("1003", "", "Synthetic Customer B", "+98 900 000 0101", "", "2025-04-01 10:00:00"),
		IssuedTo("1004", "", "Synthetic Customer C", "09000000404", "09000000405", "2025-05-01 10:00:00"),
		IssuedTo("1005", "", "", "", "", "2025-06-01 10:00:00"),
	})
	e.Completeness = c
	return []snapshot.EntityRows{e}
}
