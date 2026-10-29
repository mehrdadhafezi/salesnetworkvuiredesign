// Fixture: named and blank imports are allowed; only dot imports are rejected. Never compiled.
package sales

import (
	_ "embed"

	kr "example.test/mod/internal/kernel/refs"
)

var _ = kr.Domains
