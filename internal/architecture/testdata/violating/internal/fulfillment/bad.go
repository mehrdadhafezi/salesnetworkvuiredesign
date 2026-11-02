// Fixture: violates the Slice 10 fulfillment rules. Never compiled.
package fulfillment

// module-boundary: fulfillment never reaches biawin internals.
import _ "example.test/mod/internal/biawin/query"

// foreign-schema-write + biawin-foreign-sql: fulfillment never mutates billing.
const mutateBilling = "INSERT INTO billing.payment_stage (id) VALUES (1)"
