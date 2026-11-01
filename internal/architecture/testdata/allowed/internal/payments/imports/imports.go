// Fixture: allowed. Payments resolves invoices through the published billing
// API and writes only its own schema. Never compiled.
package imports

import _ "example.test/mod/internal/billing/api"

const own = "INSERT INTO payments.payment_attempt (attempt_id) VALUES ($1)"
const history = "INSERT INTO payments.attempt_state_history (history_id) VALUES ($1)"
