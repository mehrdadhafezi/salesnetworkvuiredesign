package organization

// Violation: organization reaches identity internals instead of identity/api.
import _ "example.test/mod/internal/identity/query"
