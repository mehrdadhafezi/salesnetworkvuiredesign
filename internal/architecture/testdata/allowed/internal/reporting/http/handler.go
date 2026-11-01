// Fixture: allowed. The reporting http adapter uses the evaluator, the
// capability vocabulary and its own query service. Never compiled.
package http

import (
	_ "example.test/mod/internal/authorization"
	_ "example.test/mod/internal/identity/api"
	_ "example.test/mod/internal/reporting/query"
)
