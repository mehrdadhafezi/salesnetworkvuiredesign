// Fixture: violates the Slice 7 rules. Never compiled.
package payments

// module-boundary: billing internals.
import _ "example.test/mod/internal/billing/query"

// append-only-write.
const rewrite = "UPDATE payments.attempt_state_history SET regression = false"

// foreign-schema-write: payments never writes billing.
const writeBilling = "INSERT INTO billing.invoice (invoice_id) VALUES ($1)"

// phone-key.
const byPhone = "SELECT attempt_id FROM x WHERE customer_phone = $1"

// money-unit-conversion: a conversion-like function and arithmetic by 10.
func toRial(toman int64) int64 { return toman * 10 }
