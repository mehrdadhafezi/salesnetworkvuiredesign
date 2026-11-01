// Fixture: a reporting read handler importing the rebuild (read-path-import)
// and an owner api (module-boundary: only the rebuild reads owners). Never compiled.
package http

import (
	_ "example.test/mod/internal/payments/api"
	_ "example.test/mod/internal/reporting/projections"
)
