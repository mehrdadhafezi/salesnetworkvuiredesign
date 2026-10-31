package apiserver

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

const someID = "0b9c2f5e-1111-4111-8111-111111111111"

func concrete(path string) string {
	return strings.NewReplacer("{user_id}", someID, "{person_id}", someID).Replace(path)
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
		for _, p := range item.Get.Parameters {
			if p.Value.In == "query" {
				got = append(got, p.Value.Name)
			}
		}
		want := append([]string(nil), allowed...)
		sort.Strings(got)
		sort.Strings(want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%s: contract %v, allowlist %v", path, got, want)
		}
	}
	if len(ContractQuery) != 10 {
		t.Fatalf("allowlist covers %d paths", len(ContractQuery))
	}
}

// Without a configured read surface every Slice 4 read is BLOCKED /
// FEATURE_DISABLED (never an empty success) and every mutation is refused.
func TestSlice4UnconfiguredAndReadOnly(t *testing.T) {
	h := handler(t, authstub.Development, true)
	for path := range ContractQuery {
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
			if env.Data == nil || env.Data.HRReadEnabled || len(env.Data.Decisions) != 6 {
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
			rec := do(h, m, concrete(path), nil)
			if rec.Code != 405 || rec.Header().Get("Allow") != "GET" || rec.Header().Get("Content-Type") != httpx.ProblemContentType {
				t.Fatalf("%s %s = %d allow=%q", m, path, rec.Code, rec.Header().Get("Allow"))
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
