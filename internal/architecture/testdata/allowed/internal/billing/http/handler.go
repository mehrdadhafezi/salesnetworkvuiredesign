// Fixture: allowed. The billing http adapter reads payment sums through the
// published payments API and uses the evaluator. Never compiled.
package http

import (
	_ "example.test/mod/internal/authorization"
	_ "example.test/mod/internal/billing/query"
	_ "example.test/mod/internal/identity/api"
	_ "example.test/mod/internal/payments/api"
)

// Separate states, never a generic resolver; amounts keep their unit.
func stageState(target *string) string {
	if target == nil {
		return "UNKNOWN"
	}
	return *target
}
