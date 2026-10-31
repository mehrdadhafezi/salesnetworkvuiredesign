package authorization

// Allowed: the evaluator reads the published identity and organization apis.
import (
	_ "example.test/mod/internal/identity/api"
	_ "example.test/mod/internal/organization/api"
)
