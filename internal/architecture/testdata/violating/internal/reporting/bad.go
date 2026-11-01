// Fixture: violates the Slice 8 rules. Never compiled.
package reporting

// module-boundary: an owner's internals (not its published api).
import _ "example.test/mod/internal/cases/query"

// reporting-foreign-read: reporting never reads an owner schema.
const ownerRead = "SELECT invoice_id FROM billing.invoice"

// append-only-write: reporting facts are never rewritten.
const rewrite = "UPDATE reporting.case_fact SET presence = 'X'"

// metric-formula: aggregates only in the shared query spec.
const formula = "SELECT count(*) FROM reporting.case_fact"

// export-endpoint: no export route of any kind.
const exportRoute = "/api/v1/invoice-review/export"

// phone-key: no phone in reporting SQL.
const byPhone = "SELECT case_id FROM reporting.case_fact WHERE customer_phone = $1"
