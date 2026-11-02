package http

import (
	"context"
	"strconv"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// IfMatchVersion parses an If-Match aggregate version: a positive decimal,
// optionally quoted. Weak or list forms are not versions.
func IfMatchVersion(v string) (int64, bool) {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	n, err := strconv.ParseInt(v, 10, 64)
	return n, err == nil && n > 0 && !strings.ContainsAny(v, "+-")
}

// CommandEnvelope builds the result envelope of one command.
func CommandEnvelope(ctx context.Context, status result.Status, codes []reasoncodes.Code, cr *oapi.CommandResult) oapi.CommandEnvelope {
	return oapi.CommandEnvelope{Outcome: Outcome(status), Reasons: Reasons(codes...), CorrelationId: httpx.CorrelationID(ctx), Command: cr}
}
