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
		"{entry_id}", someID, "{subscription_id}", someID, "{request_id}", someID, "{command_id}", someID).Replace(path)
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
	if len(ContractQuery) != 74 {
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
