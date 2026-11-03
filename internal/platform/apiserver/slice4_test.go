package apiserver

import (
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

const someID = "0b9c2f5e-1111-4111-8111-111111111111"

func concrete(path string) string {
	return strings.NewReplacer("{user_id}", someID, "{person_id}", someID, "{customer_id}", someID, "{merge_case_id}", someID, "{case_id}", someID,
		"{invoice_id}", someID, "{attempt_id}", someID, "{receipt_id}", someID, "{metric_id}", "cases.staged_total", "{run_id}", someID, "{account_id}", someID,
		"{entry_id}", someID, "{subscription_id}", someID, "{request_id}", someID, "{command_id}", someID, "{batch_id}", someID).Replace(path)
}

// The query allowlist is exactly the contract's query parameters.
func TestQueryAllowlistMatchesContract(t *testing.T) {
	doc := loadContract(t)
	for path, item := range doc.Paths.Map() {
		allowed, guarded := ContractQuery[path]
		if !strings.HasPrefix(path, "/api/v1/") || path == "/api/v1/session" {
			if guarded {
				t.Fatalf("%s is not a Slice 4 path", path)
			}
			continue
		}
		if !guarded {
			t.Fatalf("Slice 4 path %s has no query allowlist", path)
		}
		var got []string
		for _, op := range item.Operations() {
			for _, p := range op.Parameters {
				if p.Value.In == "query" && !slices.Contains(got, p.Value.Name) {
					got = append(got, p.Value.Name)
				}
			}
		}
		want := append([]string(nil), allowed...)
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%s: contract %v, allowlist %v", path, got, want)
		}
	}
	if len(ContractQuery) != 81 {
		t.Fatalf("allowlist covers %d paths", len(ContractQuery))
	}
}

// Without a configured read surface every Slice 4 read is BLOCKED /
// FEATURE_DISABLED (never an empty success) and every mutation is refused.
func TestSlice4UnconfiguredAndReadOnly(t *testing.T) {
	h := handler(t, authstub.Development, true)
	for path := range ContractQuery {
		methods := ContractPaths[path]
		if !slices.Contains(methods, http.MethodGet) {
			if rec := do(h, http.MethodGet, concrete(path), nil); rec.Code != 405 {
				t.Fatalf("GET %s = %d", path, rec.Code)
			}
			continue
		}
		rec := do(h, http.MethodGet, concrete(path), map[string]string{authstub.Header: "dev-user-001"})
		if rec.Code != 200 {
			t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body.String())
		}
		var env struct {
			Outcome string   `json:"outcome"`
			Reasons []string `json:"reasons"`
			Items   any      `json:"items"`
			Data    *struct {
				HRReadEnabled bool `json:"hr_read_enabled"`
				Decisions     []struct {
					Allowed bool `json:"allowed"`
				} `json:"decisions"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if path == "/api/v1/authorization/effective-access" {
			if env.Data == nil || env.Data.HRReadEnabled || len(env.Data.Decisions) != 29 {
				t.Fatalf("effective access = %s", rec.Body.String())
			}
			for _, d := range env.Data.Decisions {
				if d.Allowed {
					t.Fatalf("unconfigured surface allowed something: %s", rec.Body.String())
				}
			}
			continue
		}
		if env.Outcome != "BLOCKED" || len(env.Reasons) != 1 || env.Reasons[0] != "FEATURE_DISABLED" || env.Items != nil {
			t.Fatalf("GET %s = %s", path, rec.Body.String())
		}
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			if slices.Contains(methods, m) {
				continue // a Slice 11 command: covered by TestSlice11UnconfiguredCommandsBlocked
			}
			rec := do(h, m, concrete(path), nil)
			if rec.Code != 405 || strings.Join(rec.Header().Values("Allow"), ",") != strings.Join(methods, ",") ||
				rec.Header().Get("Content-Type") != httpx.ProblemContentType {
				t.Fatalf("%s %s = %d allow=%q", m, path, rec.Code, rec.Header().Values("Allow"))
			}
		}
	}
}

func TestSlice4RejectsUnknownQueryAndInvalidIDs(t *testing.T) {
	h := handler(t, authstub.Production, false)
	for _, target := range []string{"/api/v1/organization/persons?salary=1", "/api/v1/identity/users?login=x",
		"/api/v1/organization/org-units?order=label", "/api/v1/identity/users/" + someID + "?expand=credentials"} {
		rec := do(h, http.MethodGet, target, nil)
		if rec.Code != 400 || rec.Header().Get("Content-Type") != httpx.ProblemContentType || !strings.Contains(rec.Body.String(), "CONTRACT_INVALID") {
			t.Fatalf("%s = %d %s", target, rec.Code, rec.Body.String())
		}
	}
	if rec := do(h, http.MethodGet, "/api/v1/organization/persons/not-a-uuid", nil); rec.Code != 400 {
		t.Fatalf("invalid id = %d", rec.Code)
	}
	// Slice 2 paths keep their behaviour (no allowlist there).
	if rec := do(h, http.MethodGet, "/health/live?x=1", nil); rec.Code != 200 {
		t.Fatalf("health with query = %d", rec.Code)
	}
}

func TestSessionAdvertisesHRReadFeature(t *testing.T) {
	env := session(t, handler(t, authstub.Production, false), nil)
	if env.Data.Features == nil || env.Data.Features.HrRead {
		t.Fatalf("features = %+v", env.Data.Features)
	}
	var _ oapi.SessionContext = env.Data
}

// Without their modules, the Slice 11 commands answer BLOCKED /
// FEATURE_DISABLED with TARGET_WRITE_DISABLED: nothing is accepted.
func TestSlice11UnconfiguredCommandsBlocked(t *testing.T) {
	h := handler(t, authstub.Development, true)
	for path, body := range map[string]string{
		"/api/v1/cases/{case_id}/contact-outcomes":   `{"outcome":"NO_ANSWER"}`,
		"/api/v1/hr/requests":                        `{"request_type":"terminate_seller","subject_user_id":"` + someID + `","reason":"x"}`,
		"/api/v1/hr/requests/{request_id}/decisions": `{"decision":"reject"}`,
	} {
		rec := doBody(h, http.MethodPost, concrete(path), body, map[string]string{authstub.Header: "dev-user-001",
			"Idempotency-Key": "synthetic-key-0000000001", "If-Match": "1", "Content-Type": "application/json"})
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"outcome":"BLOCKED"`) || !strings.Contains(rec.Body.String(), "TARGET_WRITE_DISABLED") {
			t.Fatalf("POST %s = %d %s", path, rec.Code, rec.Body.String())
		}
		// A field outside the command's field scope is refused before any handler.
		rec = doBody(h, http.MethodPost, concrete(path), `{"seller_id":"1"}`, map[string]string{"Idempotency-Key": "synthetic-key-0000000002",
			"If-Match": "1", "Content-Type": "application/json"})
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), "field scope") {
			t.Fatalf("POST %s with an out-of-scope field = %d %s", path, rec.Code, rec.Body.String())
		}
	}
}

// Slice 12: without a configured payments surface the upload and the batch
// are BLOCKED with nothing applied; the batch's nested field scope is
// enforced before any handler; only the upload path has a raised body limit.
func TestSlice12UnconfiguredPaymentsBlocked(t *testing.T) {
	h := handler(t, authstub.Development, true)
	dev := map[string]string{authstub.Header: "dev-user-001", "Content-Type": "application/octet-stream"}
	rec := doBody(h, http.MethodPost, "/api/v1/payments/uploads", "%PDF-1.4 synthetic", dev)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"outcome":"BLOCKED"`) || !strings.Contains(rec.Body.String(), "TARGET_WRITE_DISABLED") {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body.String())
	}
	// 2 MiB is above the 1 MiB server default but within the 5 MiB receipt limit.
	if rec := doBody(h, http.MethodPost, "/api/v1/payments/uploads", strings.Repeat("x", 2<<20), dev); rec.Code != 200 {
		t.Fatalf("2 MiB upload = %d", rec.Code)
	}
	if rec := doBody(h, http.MethodPost, "/api/v1/payments/uploads", strings.Repeat("x", 6<<20), dev); rec.Code != 413 {
		t.Fatalf("6 MiB upload = %d", rec.Code)
	}
	// The raised limit applies to the upload path only.
	if rec := doBody(h, http.MethodPost, concrete("/api/v1/hr/requests"), strings.Repeat("x", 2<<20), map[string]string{
		"Content-Type": "application/json", "Idempotency-Key": "synthetic-key-0000000003"}); rec.Code != 413 {
		t.Fatalf("2 MiB JSON command = %d", rec.Code)
	}
	path := concrete("/api/v1/payments/invoices/{invoice_id}/batches")
	headers := map[string]string{authstub.Header: "dev-user-001", "Idempotency-Key": "synthetic-key-0000000004", "If-Match": "0",
		"Content-Type": "application/json"}
	good := `{"stage_no":1,"entry_mode":"RECEIPT","items":[{"item_key":"a","amount":100,"unit":"IRT","receipts":[{"upload_id":"` + someID +
		`","content_sha256":"` + strings.Repeat("a", 64) + `"}]}]}`
	if rec := doBody(h, http.MethodPost, path, good, headers); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"outcome":"BLOCKED"`) ||
		!strings.Contains(rec.Body.String(), `"applied":false`) {
		t.Fatalf("batch = %d %s", rec.Code, rec.Body.String())
	}
	for _, bad := range []string{
		`{"stage_no":1,"entry_mode":"RECEIPT","items":[{"item_key":"a","amount":100,"unit":"IRT","receipts":[],"seller_id":"9"}]}`,
		`{"stage_no":1,"entry_mode":"RECEIPT","items":[{"item_key":"a","amount":100,"unit":"IRT","receipts":[{"upload_id":"` + someID +
			`","content_sha256":"` + strings.Repeat("a", 64) + `","path":"../../etc/passwd"}]}]}`,
		`{"stage_no":1,"entry_mode":"RECEIPT","items":{"item_key":"a"}}`,
		`{"stage_no":1,"entry_mode":"RECEIPT","items":[],"approve":true}`,
	} {
		if rec := doBody(h, http.MethodPost, path, bad, headers); rec.Code != 400 || !strings.Contains(rec.Body.String(), "field scope") {
			t.Fatalf("out-of-scope batch body %s = %d %s", bad, rec.Code, rec.Body.String())
		}
	}
	// No review decision, posting, refund, next-stage, invoice-issue or
	// gateway route exists.
	for _, p := range []string{"/api/v1/finance/review/attempts/" + someID + "/decisions", "/api/v1/finance/review/attempts/" + someID + "/approve",
		"/api/v1/finance/ledger/postings", "/api/v1/payments/refunds", "/api/v1/billing/invoices/" + someID + "/next-stage",
		"/api/v1/billing/invoices", "/api/v1/payments/gateway/start", "/api/v1/payments/gateway/callback", "/api/v1/cases/" + someID + "/return"} {
		if rec := doBody(h, http.MethodPost, p, `{}`, headers); rec.Code < 400 {
			t.Fatalf("POST %s = %d", p, rec.Code)
		}
	}
}
