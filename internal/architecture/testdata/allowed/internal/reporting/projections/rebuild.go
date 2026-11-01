// Fixture: allowed. The offline rebuild reads owners through their
// published api and writes only reporting. Never compiled.
package projections

import (
	_ "example.test/mod/internal/billing/api"
	_ "example.test/mod/internal/cases/api"
)

const insert = "INSERT INTO reporting.case_fact (run_id, case_id) VALUES ($1, $2)"
