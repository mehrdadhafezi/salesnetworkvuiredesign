package http

// RM-E04B fixture: a test may use the decision seam (NT-07).
import "example.test/mod/internal/authorization"

var _, _ = authorization.NewGuardWithDecider(nil, nil, nil)
