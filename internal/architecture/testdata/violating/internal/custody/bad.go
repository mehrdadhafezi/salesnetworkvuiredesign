package custody

// Violates module-boundary (cases internals, organization), append-only-write
// and phone-key.
import (
	_ "example.test/mod/internal/cases/query"
	_ "example.test/mod/internal/organization/api"
)

const rewrite = "UPDATE custody.custody_event SET to_legacy_key = $1"

const joinByPhone = "SELECT case_id FROM x WHERE customer_phone = $1"
