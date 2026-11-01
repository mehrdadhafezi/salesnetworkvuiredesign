// Fixture: violates the Slice 9 rules. Never compiled.
package finance

// module-boundary: an owner's internals (not its published api).
import _ "example.test/mod/internal/billing/query"

// finance-foreign-read: finance never reads another schema.
const ownerRead = "SELECT amount FROM payments.payment_attempt"

// foreign-schema-write: finance never writes billing.
const billingWrite = "UPDATE billing.invoice SET lifecycle_state = 'PAID'"

// append-only-write: a ledger fact is never rewritten.
const rewrite = "DELETE FROM finance.ledger_entry WHERE run_id = $1"

// finance-cross-unit-sum: a sum must be scoped by unit.
const crossUnit = "SELECT sum(amount) FROM finance.ledger_line WHERE run_id = $1"

// phone-key: never a phone account key.
const byPhone = "customer_receivable|phone:"

// finance-float: money is never a float.
func ratioOf(a float64) float64 { return a }

// money-unit-conversion: no Toman/Rial conversion.
func toRial(v int64) int64 { return v * 10 }

// phone-key: no phone identifier.
var mobileKey string
