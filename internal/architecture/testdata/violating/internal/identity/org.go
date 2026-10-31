package identity

// Violation: identity never imports organization, not even its api.
import _ "example.test/mod/internal/organization/api"
