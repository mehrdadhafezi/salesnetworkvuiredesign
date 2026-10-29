package httpx

import (
	"fmt"
	"net/http"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// HTTPStatus maps a business outcome to its HTTP status (Architecture V1 §23).
// Business outcomes are carried in a ResultEnvelope; HTTP 200 is used for
// SUCCESS, PARTIAL, FAILED, UNKNOWN and BLOCKED, so a 2xx never implies
// business SUCCESS. CONFLICT → 409, UNAUTHORIZED → 403. An invalid status is
// a programming error.
func HTTPStatus(s result.Status) (int, error) {
	switch s {
	case result.Success, result.Partial, result.Failed, result.Unknown, result.Blocked:
		return http.StatusOK, nil
	case result.Conflict:
		return http.StatusConflict, nil
	case result.Unauthorized:
		return http.StatusForbidden, nil
	default:
		return 0, fmt.Errorf("httpx: invalid outcome %d", s)
	}
}

// Outcome converts a kernel status to the contract enum without collapsing
// any of the seven values.
func Outcome(s result.Status) (oapi.Outcome, error) {
	if !s.Valid() {
		return "", fmt.Errorf("httpx: invalid outcome %d", s)
	}
	o := oapi.Outcome(s.String())
	if !o.Valid() {
		return "", fmt.Errorf("httpx: outcome %s missing from contract", s)
	}
	return o, nil
}
