package main

// RM-E04B fixture: production code wiring the test-only decision seam.
import "example.test/mod/internal/authorization"

func main() {
	_, _ = authorization.NewGuardWithDecider(nil, nil, nil)
}
