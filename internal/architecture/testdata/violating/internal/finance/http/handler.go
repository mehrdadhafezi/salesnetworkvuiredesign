// Fixture: violates the Slice 9 read-path rules. Never compiled.
package http

// read-path-import: a GET never derives the shadow ledger.
import _ "example.test/mod/internal/finance/shadow"

// finance-read-write: no write SQL on the read path.
const write = "INSERT INTO finance.shadow_run (run_id) VALUES ($1)"

// finance-write-route: no posting / approval route.
const route = "/api/v1/finance/ledger/{entry_id}/reverse"
