package http

// Violates module-boundary (custody) and read-path-import.
import (
	_ "example.test/mod/internal/cases/imports"
	_ "example.test/mod/internal/custody/query"
)
