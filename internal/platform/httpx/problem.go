package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// ProblemContentType is the RFC 9457 media type.
const ProblemContentType = "application/problem+json"

// problemTypeBase prefixes problem type URIs (relative references).
const problemTypeBase = "/problems/"

// Problem kinds. Each maps to one HTTP status, a stable type URI and a
// foundation-v1 reason code; there is no second error taxonomy.
type Problem struct {
	Status int
	Slug   string // type URI suffix, e.g. "not-found"
	Title  string
	Reason reasoncodes.Code
	Detail string // safe, generic text only; never internals
	Errors []oapi.FieldError
}

// Standard problems used by the HTTP foundation.
var (
	ProblemNotFound = Problem{Status: http.StatusNotFound, Slug: "not-found", Title: "Resource not found",
		Reason: reasoncodes.ContractInvalid, Detail: "No operation of the API contract matches this path."}
	ProblemMethodNotAllowed = Problem{Status: http.StatusMethodNotAllowed, Slug: "method-not-allowed", Title: "Method not allowed",
		Reason: reasoncodes.ContractInvalid, Detail: "The API contract does not define this method for this path."}
	ProblemBadRequest = Problem{Status: http.StatusBadRequest, Slug: "bad-request", Title: "Request does not match the contract",
		Reason: reasoncodes.ContractInvalid}
	ProblemBodyTooLarge = Problem{Status: http.StatusRequestEntityTooLarge, Slug: "body-too-large", Title: "Request body too large",
		Reason: reasoncodes.ContractInvalid}
	// ProblemInternal: the server failed after the request was accepted.
	// Effects of the request cannot be proven absent, so the reason is
	// OUTCOME_UNKNOWN (reconcile before any retry), never a guessed failure.
	ProblemInternal = Problem{Status: http.StatusInternalServerError, Slug: "internal", Title: "Internal server error",
		Reason: reasoncodes.OutcomeUnknown, Detail: "The request could not be completed; its outcome is unknown."}
	// ProblemNoRecord: a contract path whose identified record does not exist.
	ProblemNoRecord = Problem{Status: http.StatusNotFound, Slug: "not-found", Title: "Resource not found",
		Reason: reasoncodes.ContractInvalid, Detail: "No record with this id."}
)

// BodyCtx renders the problem inside a strict handler, where only the
// request context is available (no instance path).
func (p Problem) BodyCtx(ctx context.Context) oapi.Problem {
	body := oapi.Problem{Type: problemTypeBase + p.Slug, Title: p.Title, Status: p.Status,
		ReasonCode: oapi.ReasonCode(p.Reason), CorrelationId: CorrelationID(ctx)}
	if p.Detail != "" {
		d := p.Detail
		body.Detail = &d
	}
	return body
}

// Body renders the problem for a request.
func (p Problem) Body(r *http.Request) oapi.Problem {
	body := oapi.Problem{
		Type:          problemTypeBase + p.Slug,
		Title:         p.Title,
		Status:        p.Status,
		ReasonCode:    oapi.ReasonCode(p.Reason),
		CorrelationId: CorrelationID(r.Context()),
	}
	if p.Detail != "" {
		d := p.Detail
		body.Detail = &d
	}
	if r.URL != nil && r.URL.Path != "" {
		instance := r.URL.Path
		body.Instance = &instance
	}
	if len(p.Errors) > 0 {
		errs := p.Errors
		if len(errs) > 100 {
			errs = errs[:100]
		}
		body.Errors = &errs
	}
	return body
}

// WriteProblem writes p as application/problem+json.
func WriteProblem(w http.ResponseWriter, r *http.Request, p Problem) {
	b, err := json.Marshal(p.Body(r))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", ProblemContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_, _ = w.Write(b)
}

// Recover turns a panic into a generic 500 problem. The panic value and stack
// are logged server-side only; nothing about them reaches the response.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					if v == http.ErrAbortHandler {
						panic(v)
					}
					logger.Error("panic serving request", "correlation_id", CorrelationID(r.Context()), "path", r.URL.Path, "panic", v)
					WriteProblem(w, r, ProblemInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// LimitBody caps request bodies at maxBytes; reading beyond fails.
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				WriteProblem(w, r, ProblemBodyTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
