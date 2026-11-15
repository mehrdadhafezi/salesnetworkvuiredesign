package shadow_test

// DEF-RP-01 (found by R7-7): a report list read with an explicit run_id that
// names no projection run answered 500, because the list handlers did not map
// query.ErrNotFound and the list contracts declared no 404. The row reads
// already answered 404 for the same run. The list reads now answer 404 too.
// "No run can answer" is a different case and keeps its existing answer: 200
// with meta.available = false (UNKNOWN, never zero).

import (
	"context"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
)

var reportLists = []string{"cases", "custody", "invoices", "payments"}

const unknownRun = "00000000-0000-4000-8000-0000000000d1"

func TestReportingListUnknownRunIsNotFound(t *testing.T) {
	e := setupReporting(t)
	ctx := context.Background()
	e.setFlag(t, true)
	h := e.server(t, e.auditor(t))

	// No projection run yet: UNKNOWN (meta.available = false), never a zero, never an error.
	for _, r := range reportLists {
		res := get(t, h, "/api/v1/reports/"+r, "dev-user-001")
		meta, _ := res.json["meta"].(map[string]any)
		if res.code != 200 || meta == nil || meta["available"] != false || res.json["summary"] != nil {
			t.Fatalf("%s without a run: %d %s", r, res.code, res.body)
		}
	}

	b := e.prepare(t, "syn-rep-def-rp-01", billingSource(snapshot.Complete))
	e.importBillingOnly(t, b)
	e.importPaymentsOnly(t, b)
	run, err := e.rebuild.Rebuild(ctx, e.sources())
	if err != nil {
		t.Fatal(err)
	}
	fpReporting, fpOwners := e.md5(t, reportingTables...), e.md5(t, ownerTables...)

	for _, r := range reportLists {
		// The run that exists answers with a summary.
		ok := get(t, h, "/api/v1/reports/"+r+"?run_id="+run.RunID, "dev-user-001")
		if ok.code != 200 || ok.json["summary"] == nil {
			t.Fatalf("%s known run: %d %s", r, ok.code, ok.body)
		}
		// A run that does not exist is 404 problem+json, with no value.
		nf := get(t, h, "/api/v1/reports/"+r+"?run_id="+unknownRun, "dev-user-001")
		if nf.code != 404 || nf.json["status"] != float64(404) || nf.json["summary"] != nil || nf.json["items"] != nil {
			t.Fatalf("%s unknown run: want 404 problem, got %d %s", r, nf.code, nf.body)
		}
		if strings.Contains(nf.body, `"total"`) {
			t.Fatalf("%s unknown run carried a value: %s", r, nf.body)
		}
		// A malformed run id stays a contract error.
		if bad := get(t, h, "/api/v1/reports/"+r+"?run_id=not-a-uuid", "dev-user-001"); bad.code != 400 {
			t.Fatalf("%s malformed run: %d %s", r, bad.code, bad.body)
		}
		// Authorization still runs first: a subject without the capability gets 403, not 404.
		if den := get(t, h, "/api/v1/reports/"+r+"?run_id="+unknownRun, "dev-user-002"); den.code != 403 {
			t.Fatalf("%s unknown run, no grant: %d %s", r, den.code, den.body)
		}
	}
	// The row read answers the same unknown run the same way (unchanged).
	if nf := get(t, h, "/api/v1/reports/cases/"+unknownRun+"?run_id="+unknownRun, "dev-user-001"); nf.code != 404 {
		t.Fatalf("row read, unknown run: %d %s", nf.code, nf.body)
	}
	// With the flag off the disabled contract still wins.
	e.setFlag(t, false)
	if off := get(t, h, "/api/v1/reports/cases?run_id="+unknownRun, "dev-user-001"); off.code != 200 || !strings.Contains(off.body, `"FEATURE_DISABLED"`) {
		t.Fatalf("flag off, unknown run: %d %s", off.code, off.body)
	}
	if e.md5(t, reportingTables...) != fpReporting || e.md5(t, ownerTables...) != fpOwners {
		t.Fatal("a GET changed reporting or owner state")
	}
}
