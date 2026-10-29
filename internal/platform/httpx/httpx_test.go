package httpx

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/ids"
)

func echoCorrelation() http.Handler {
	return Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, CorrelationID(r.Context()))
	}))
}

func TestCorrelationGeneratedWhenAbsent(t *testing.T) {
	rec := httptest.NewRecorder()
	echoCorrelation().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	id := rec.Header().Get(CorrelationHeader)
	if err := ids.ValidateUUID(id); err != nil {
		t.Fatalf("generated id %q: %v", id, err)
	}
	if rec.Body.String() != id {
		t.Fatalf("context id %q != header %q", rec.Body.String(), id)
	}
}

func TestCorrelationValidPreservedInvalidReplaced(t *testing.T) {
	valid := []string{"client-req-0001", "a1b2c3d4", "ABC.def_ghi:01-x", strings.Repeat("a", 128)}
	for _, v := range valid {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(CorrelationHeader, v)
		rec := httptest.NewRecorder()
		echoCorrelation().ServeHTTP(rec, req)
		if rec.Header().Get(CorrelationHeader) != v || rec.Body.String() != v {
			t.Fatalf("valid %q not preserved: %q", v, rec.Header().Get(CorrelationHeader))
		}
	}
	invalid := []string{"short", "has space here", "-leading-dash", strings.Repeat("a", 129), "ümlaut-12345", "<script>x</script>", "a/b/c/d/e/f"}
	for _, v := range invalid {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(CorrelationHeader, v)
		rec := httptest.NewRecorder()
		echoCorrelation().ServeHTTP(rec, req)
		got := rec.Header().Get(CorrelationHeader)
		if got == v || ids.ValidateUUID(got) != nil {
			t.Fatalf("invalid %q must be replaced by a generated id, got %q", v, got)
		}
	}
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) oapi.Problem {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != ProblemContentType {
		t.Fatalf("content type %q", ct)
	}
	var p oapi.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if !p.ReasonCode.Valid() || !reasoncodes.Code(p.ReasonCode).Valid() {
		t.Fatalf("reason %q is not foundation-v1", p.ReasonCode)
	}
	if p.Status != rec.Code || p.Type == "" || p.Title == "" {
		t.Fatalf("problem = %+v (http %d)", p, rec.Code)
	}
	return p
}

func TestProblemShape(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(WithCorrelationID(t.Context(), "corr-test-01"))
	p := ProblemBadRequest
	p.Errors = make([]oapi.FieldError, 150)
	for i := range p.Errors {
		p.Errors[i] = oapi.FieldError{Path: "/amount", ReasonCode: oapi.ReasonCode(reasoncodes.ContractInvalid)}
	}
	rec := httptest.NewRecorder()
	WriteProblem(rec, req, p)
	body := decodeProblem(t, rec)
	if rec.Code != http.StatusBadRequest || body.Type != "/problems/bad-request" || body.CorrelationId != "corr-test-01" ||
		body.Instance == nil || *body.Instance != "/x" || body.Errors == nil || len(*body.Errors) != 100 {
		t.Fatalf("problem = %+v", body)
	}
	if (*body.Errors)[0].Path != "/amount" || (*body.Errors)[0].ReasonCode != "CONTRACT_INVALID" {
		t.Fatalf("field error = %+v", (*body.Errors)[0])
	}
	for _, std := range []Problem{ProblemNotFound, ProblemMethodNotAllowed, ProblemBadRequest, ProblemBodyTooLarge, ProblemInternal} {
		if !std.Reason.Valid() || std.Status < 400 || std.Slug == "" {
			t.Fatalf("standard problem %+v", std)
		}
	}
}

func TestRecoverHidesPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Correlation(Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret internal detail")
	})))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "goroutine") {
		t.Fatalf("panic leaked: %d %s", rec.Code, rec.Body.String())
	}
	if p := decodeProblem(t, rec); p.ReasonCode != "OUTCOME_UNKNOWN" || p.CorrelationId == "" {
		t.Fatalf("panic problem = %+v", p)
	}
}

func TestLimitBody(t *testing.T) {
	h := LimitBody(8)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789")))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("declared oversize body = %d", rec.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789"))
	req.ContentLength = -1 // unknown length: the reader limit applies
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("streamed oversize body = %d", rec.Code)
	}
}

func TestOutcomeHTTPMapping(t *testing.T) {
	want := map[result.Status]int{
		result.Success: 200, result.Partial: 200, result.Failed: 200, result.Unknown: 200,
		result.Blocked: 200, result.Conflict: 409, result.Unauthorized: 403,
	}
	seen := map[oapi.Outcome]bool{}
	for _, s := range result.Statuses() {
		code, err := HTTPStatus(s)
		if err != nil || code != want[s] {
			t.Fatalf("HTTPStatus(%s) = %d, %v", s, code, err)
		}
		o, err := Outcome(s)
		if err != nil || string(o) != s.String() || seen[o] {
			t.Fatalf("Outcome(%s) = %q, %v (must be distinct and identical)", s, o, err)
		}
		seen[o] = true
	}
	if len(seen) != 7 {
		t.Fatalf("outcomes = %d", len(seen))
	}
	for _, bad := range []result.Status{0, 8} {
		if _, err := HTTPStatus(bad); err == nil {
			t.Fatalf("HTTPStatus(%d) accepted", bad)
		}
		if _, err := Outcome(bad); err == nil {
			t.Fatalf("Outcome(%d) accepted", bad)
		}
	}
	// UNKNOWN and PARTIAL are never collapsed into FAILED or SUCCESS.
	if o, _ := Outcome(result.Unknown); o != oapi.Outcome("UNKNOWN") {
		t.Fatal("UNKNOWN collapsed")
	}
	if o, _ := Outcome(result.Partial); o != oapi.Outcome("PARTIAL") {
		t.Fatal("PARTIAL collapsed")
	}
}
