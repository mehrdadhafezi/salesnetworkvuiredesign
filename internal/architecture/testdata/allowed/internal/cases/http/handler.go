package http

// Allowed: the cases http adapter uses the evaluator, the capability
// vocabulary and its own query service.
import (
	_ "example.test/mod/internal/authorization"
	_ "example.test/mod/internal/authorization/http"
	_ "example.test/mod/internal/cases/query"
	_ "example.test/mod/internal/identity/api"
)
