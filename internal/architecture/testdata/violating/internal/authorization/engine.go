package authorization

// Violation: the evaluator reads organization internals, not its api.
import _ "example.test/mod/internal/organization/store"
