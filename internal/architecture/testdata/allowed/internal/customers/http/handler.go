package http

// Allowed: the customers http adapter uses the authorization evaluator, the
// identity capability vocabulary and its own query service.
import (
	_ "example.test/mod/internal/authorization"
	_ "example.test/mod/internal/authorization/http"
	_ "example.test/mod/internal/customers/query"
	_ "example.test/mod/internal/identity/api"
)
