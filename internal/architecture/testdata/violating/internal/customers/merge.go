package customers

// Violates module-boundary (organization internals) and merge-path.
import _ "example.test/mod/internal/organization/query"

// MergeCustomers is forbidden: Slice 5 has no merge path.
func MergeCustomers() {}

func canonicalizeByPhone() {}
