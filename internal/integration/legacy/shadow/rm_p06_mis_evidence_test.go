package shadow_test

// RM-P06 MIS case-to-custody evidence continuity: the exact case lookup.
//
//   - GET /api/v1/cases?case_id=…(repeat, 1..200) returns exactly the Cases
//     whose own native case_id is listed. It is the batch read behind the
//     custody and exception pages (one request per visible page, never one
//     per row) and is never an identity join: phone, names, legacy keys and
//     customer ids are not accepted in its place.
//
// Every expectation is checked against the stored rows of a disposable
// PostgreSQL loaded with synthetic fixtures, and every read is proven to
// write nothing (table fingerprints before and after).

import (
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
)

func caseIDQuery(ids ...string) url.Values {
	q := url.Values{}
	for _, id := range ids {
		q.Add("case_id", id)
	}
	return q
}

func TestRMP06ExactCaseLookup(t *testing.T) {
	e := setupSales(t)
	b := e.stage(t, "syn-mis-p06-001", withSales(legacytest.HRScenario(), snapshot.Complete))
	e.importAll(t, b)
	e.importSales(t, b)
	h := e.salesServerWith(t, p05Subjects())

	all := e.ids(t, `SELECT case_id::text FROM cases.work_case`)
	if len(all) < 6 {
		t.Fatalf("fixture: %d cases", len(all))
	}
	// Flag off: BLOCKED, never an empty page dressed as "no case".
	if r := p05Get(t, h, "/api/v1/cases?case_id="+all[0], "mis"); r.code != 200 || !strings.Contains(r.body, `"FEATURE_DISABLED"`) || strings.Contains(r.body, `"items"`) {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.setSalesFlag(t, true)
	fp := e.md5(t, salesTables...)

	// One id: exactly that Case with its own source row (provenance as stored).
	one := p05Get(t, h, "/api/v1/cases?case_id="+all[1], "mis")
	if one.code != 200 || !slices.Equal(caseIDsOf(one), []string{all[1]}) {
		t.Fatalf("exact: %d %s", one.code, one.body)
	}
	src := items(one)[0]["source"].(map[string]any)
	var sys, spec, key, first, last string
	if err := e.admin.QueryRow(t.Context(), `SELECT source_system, source_spec, source_key, first_batch_id::text, last_batch_id::text
		FROM cases.case_source WHERE case_id = $1`, all[1]).Scan(&sys, &spec, &key, &first, &last); err != nil {
		t.Fatal(err)
	}
	if src["source_system"] != sys || src["source_spec"] != spec || src["source_key"] != key || src["first_batch_id"] != first || src["last_batch_id"] != last {
		t.Fatalf("source provenance differs from the stored row: %v", src)
	}

	// Many ids in ONE request (the page-level batch read; no N+1).
	pick := []string{all[0], all[2], all[4], all[5]}
	if r := p05Get(t, h, "/api/v1/cases?"+caseIDQuery(pick...).Encode(), "mis"); r.code != 200 || !slices.Equal(caseIDsOf(r), pick) {
		t.Fatalf("batch: %v", caseIDsOf(r))
	}
	// Duplicates do not duplicate rows.
	if r := p05Get(t, h, "/api/v1/cases?"+caseIDQuery(all[3], all[3]).Encode(), "mis"); !slices.Equal(caseIDsOf(r), []string{all[3]}) {
		t.Fatalf("duplicate ids: %v", caseIDsOf(r))
	}

	// Keyset pagination stays inside the requested set.
	q := caseIDQuery(pick...)
	q.Set("limit", "3")
	p1 := p05Get(t, h, "/api/v1/cases?"+q.Encode(), "mis")
	page := p1.json["page"].(map[string]any)
	next, _ := page["next_cursor"].(string)
	if len(items(p1)) != 3 || page["has_more"] != true || next == "" {
		t.Fatalf("page 1: %s", p1.body)
	}
	q.Set("cursor", next)
	p2 := p05Get(t, h, "/api/v1/cases?"+q.Encode(), "mis")
	if got := append(caseIDsOf(p1), caseIDsOf(p2)...); !slices.Equal(got, pick) || p2.json["page"].(map[string]any)["has_more"] != false {
		t.Fatalf("keyset: %v", got)
	}

	// Conjunctive with every other filter (expected sets come from the DB).
	for _, c := range []struct{ extra, where string }{
		{"workflow_type=LEGACY_LEAD", "workflow_type = 'LEGACY_LEAD'"},
		{"workflow_type=DISTRIBUTION_ITEM", "workflow_type = 'DISTRIBUTION_ITEM'"},
		{"presence=PRESENT", "presence = 'PRESENT'"},
		{"presence=ABSENT_IN_LATEST", "presence = 'ABSENT_IN_LATEST'"},
		{"state_mapping=UNKNOWN", "state_mapping = 'UNKNOWN'"},
		{"customer_link_state=UNKNOWN", "customer_link_state = 'UNKNOWN'"},
	} {
		want := e.ids(t, `SELECT case_id::text FROM cases.work_case WHERE case_id = ANY($1::uuid[]) AND `+c.where, pick)
		r := p05Get(t, h, "/api/v1/cases?"+caseIDQuery(pick...).Encode()+"&"+c.extra, "mis")
		if r.code != 200 || !slices.Equal(caseIDsOf(r), want) {
			t.Fatalf("case_id AND %s: got %v want %v", c.extra, caseIDsOf(r), want)
		}
	}
	// case_id AND another Case's source_key: empty. The filter never widens
	// to "the case with that key" (no cross-identity inference).
	var otherKey string
	if err := e.admin.QueryRow(t.Context(), `SELECT source_key FROM cases.case_source WHERE case_id = $1`, all[0]).Scan(&otherKey); err != nil {
		t.Fatal(err)
	}
	shared := e.ids(t, `SELECT case_id::text FROM cases.case_source WHERE source_key = $1 AND case_id = $2::uuid`, otherKey, all[1])
	if r := p05Get(t, h, "/api/v1/cases?case_id="+all[1]+"&source_key="+otherKey, "mis"); r.code != 200 || !slices.Equal(caseIDsOf(r), shared) {
		t.Fatalf("case_id AND foreign source_key must not widen: %v", caseIDsOf(r))
	}

	// Unknown ids are absent (empty page), never a fabricated Case. A custody
	// projection's own id is the Case id; an unrelated uuid matches nothing.
	if r := p05Get(t, h, "/api/v1/cases?case_id=00000000-0000-4000-8000-000000000000", "mis"); r.code != 200 || len(items(r)) != 0 {
		t.Fatalf("unknown id: %s", r.body)
	}
	tooMany := url.Values{}
	for i := 0; i < 201; i++ {
		tooMany.Add("case_id", all[i%len(all)])
	}
	// Not identifiers: phone, name, legacy key, customer id spelled as a
	// key, partial uuids, upper-case variants of the contract form.
	for _, bad := range []string{"case_id=not-a-uuid", "case_id=09121234567", "case_id=" + url.QueryEscape("علی"), "case_id=" + key,
		"case_id=" + all[0][:8], "case_id=", tooMany.Encode(), "case_ids=" + all[0], "case=" + all[0], "phone=09121234567"} {
		if r := p05Get(t, h, "/api/v1/cases?"+bad, "mis"); r.code != 400 || strings.Contains(r.body, `"items"`) {
			t.Fatalf("%s must be 400, got %d %s", bad[:min(len(bad), 40)], r.code, r.body)
		}
	}

	// Permission boundaries unchanged: case_id needs cases.case.read only.
	for _, c := range []struct {
		actor string
		code  int
	}{{"mis", 200}, {"cases-only", 200}, {"custody", 403}, {"unrelated", 403}, {"", 403}} {
		r := p05Get(t, h, "/api/v1/cases?case_id="+all[0], c.actor)
		if r.code != c.code {
			t.Fatalf("%q = %d, want %d", c.actor, r.code, c.code)
		}
		// The source key is a short digit string; the random server correlation id is removed before the
		// substring check so a coincidental digit run in that UUID is never mistaken for leaked case data.
		body := r.body
		if cid, _ := r.json["correlation_id"].(string); cid != "" {
			body = strings.ReplaceAll(body, cid, "")
		}
		if c.code == 403 && (strings.Contains(body, `"items"`) || strings.Contains(body, `"source_key"`) || strings.Contains(body, key)) {
			t.Fatalf("denied body leaks case data: %s", r.body)
		}
	}
	// Custody stays independently readable when cases are denied.
	if r := p05Get(t, h, "/api/v1/custody/assignments?case_id="+all[0], "custody"); r.code != 200 {
		t.Fatalf("custody independent of cases: %d", r.code)
	}

	if e.md5(t, salesTables...) != fp {
		t.Fatal("an RM-P06 read changed cases/custody state")
	}
}

// Cases absent from the latest COMPLETE import stay resolvable by exact
// id, labelled ABSENT_IN_LATEST with the batch ids they were last seen in.
// Absence is never turned into "no case", eligibility or a different Case.
func TestRMP06AbsentCaseStaysExactlyResolvable(t *testing.T) {
	e := setupSales(t)
	b := e.stage(t, "syn-mis-p06-a", withSales(legacytest.HRScenario(), snapshot.Complete))
	e.importAll(t, b)
	e.importSales(t, b)
	before := e.ids(t, `SELECT case_id::text || ':' || first_batch_id::text || ':' || last_batch_id::text FROM cases.case_source`)

	empty := legacytest.SalesScenario(snapshot.Complete)
	for i := range empty {
		empty[i].Rows = nil
	}
	e.importSales(t, e.stage(t, "syn-mis-p06-b", empty))
	e.setSalesFlag(t, true)
	h := e.salesServerWith(t, p05Subjects())
	fp := e.md5(t, salesTables...)

	if after := e.ids(t, `SELECT case_id::text || ':' || first_batch_id::text || ':' || last_batch_id::text FROM cases.case_source`); !slices.Equal(after, before) {
		t.Fatalf("absence rewrote batch provenance: %v -> %v", before, after)
	}
	all := e.ids(t, `SELECT case_id::text FROM cases.work_case`)
	r := p05Get(t, h, "/api/v1/cases?"+caseIDQuery(all...).Encode(), "mis")
	if r.code != 200 || !slices.Equal(caseIDsOf(r), all) {
		t.Fatalf("absent cases by id: %v", caseIDsOf(r))
	}
	for _, it := range items(r) {
		src := it["source"].(map[string]any)
		if it["presence"] != "ABSENT_IN_LATEST" || src["presence"] != "ABSENT_IN_LATEST" {
			t.Fatalf("absent case presented as present: %v", it)
		}
	}
	if strings.Contains(r.body, "ELIGIBLE") || strings.Contains(r.body, "REUSABLE") {
		t.Fatalf("absence surfaced as eligibility: %s", r.body)
	}
	if p := p05Get(t, h, "/api/v1/cases?"+caseIDQuery(all...).Encode()+"&presence=PRESENT", "mis"); len(items(p)) != 0 {
		t.Fatalf("absent case returned as PRESENT: %s", p.body)
	}
	if e.md5(t, salesTables...) != fp {
		t.Fatal("an absent-case read changed state")
	}
}
