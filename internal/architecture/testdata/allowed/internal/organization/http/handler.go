package http

// Allowed: a module http adapter uses the authorization evaluator and its
// own query service.
import (
	_ "example.test/mod/internal/authorization"
	_ "example.test/mod/internal/authorization/http"
	_ "example.test/mod/internal/organization/query"
)
