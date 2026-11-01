// Fixture: the allowed Slice 9 shapes. Never compiled.
package shadow

import (
	_ "example.test/mod/internal/billing/api"
	_ "example.test/mod/internal/payments/api"
)

type Entry struct{ BusinessKey, Basis string }

var e = Entry{BusinessKey: "payments/attempt/x/cash_receipt/IRT/CASH_SHADOW/v1", Basis: "CASH_SHADOW"}

const write = "INSERT INTO finance.ledger_entry (run_id, entry_id) VALUES ($1, $2)"
