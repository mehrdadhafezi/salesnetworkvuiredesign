// Fixture: the finance read adapter. Never compiled.
package http

import (
	_ "example.test/mod/internal/finance/query"
	_ "example.test/mod/internal/identity/api"
)

const route = "/api/v1/finance/ledger/{entry_id}"
