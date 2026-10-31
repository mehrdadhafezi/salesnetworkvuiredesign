package http

// Violates read-path-import and module-boundary (identity beyond api).
import (
	_ "example.test/mod/internal/customers/imports"
	_ "example.test/mod/internal/identity/query"
)
