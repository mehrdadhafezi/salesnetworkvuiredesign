// Package httpx is the HTTP foundation shared by the API server: correlation
// handling, RFC 9457 problem responses, the outcome→HTTP status mapping,
// panic recovery and request body limits (Architecture V1 §23). It contains
// no business logic and imports no business module.
package httpx

import (
	"context"
	"net/http"
	"regexp"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

// CorrelationHeader carries the request correlation id in both directions.
const CorrelationHeader = "X-Correlation-ID"

// correlationPattern is the OpenAPI CorrelationID schema: 8..128 characters,
// alphanumeric first, then [A-Za-z0-9._:-]. No spaces, no control characters.
var correlationPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

type correlationKey struct{}

// ValidCorrelationID reports whether s satisfies the contract pattern.
func ValidCorrelationID(s string) bool { return correlationPattern.MatchString(s) }

// CorrelationID returns the request's correlation id ("" outside the middleware).
func CorrelationID(ctx context.Context) string {
	id, _ := ctx.Value(correlationKey{}).(string)
	return id
}

// WithCorrelationID returns ctx carrying id (used by the middleware and tests).
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

// Correlation keeps a valid inbound X-Correlation-ID and replaces a missing
// or invalid one with a generated UUID. Invalid values are always replaced,
// never echoed. The id is put in the request context and on the response.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(CorrelationHeader)
		if !ValidCorrelationID(id) {
			generated, err := ids.NewUUID()
			if err != nil {
				http.Error(w, "correlation id unavailable", http.StatusInternalServerError)
				return
			}
			id = generated
		}
		w.Header().Set(CorrelationHeader, id)
		next.ServeHTTP(w, r.WithContext(WithCorrelationID(r.Context(), id)))
	})
}
