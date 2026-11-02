// Fixture: commission never imports or writes finance. Never compiled.
package commission

import _ "example.test/mod/internal/finance/api"

const post = "INSERT INTO finance.ledger_entry (run_id) VALUES ($1)"
