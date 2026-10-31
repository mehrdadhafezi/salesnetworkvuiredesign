package shadowok

// Allowed: the connector calls published module apis and the evaluator, and
// writes only its own schema.
import (
	_ "example.test/mod/internal/authorization"
	_ "example.test/mod/internal/identity/api"
	_ "example.test/mod/internal/organization/api"
)

const own = "UPDATE integration.import_batch SET state = $2 WHERE batch_id = $1"
