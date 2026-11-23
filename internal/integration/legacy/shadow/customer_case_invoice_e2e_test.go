package shadow_test

// RM-P01 — Customer → Case → Invoice read navigation.
//
// Real disposable PostgreSQL, synthetic fixtures only. The fixtures cover
// every link state (RESOLVED, RELATED, UNKNOWN, CONFLICT, NOT_RECOVERABLE)
// and prove that the customer_id / case_id list filters follow only the
// authoritative RESOLVED link of the row itself: never a candidate id on an
// ambiguous link, never a cross-module inference (invoice → case →
// customer), never a phone or a coincident legacy source key.

import (
	"context"
	"crypto/sha256"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	billinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/http"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	custodyhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Synthetic native ids (lowercase canonical UUIDs).
const (
	p01CustA = "a0000000-0000-4000-8000-00000000000a"
	p01CustB = "b0000000-0000-4000-8000-00000000000b"
	p01CustZ = "f0000000-0000-4000-8000-0000000000ff" // never referenced

	p01CaseA1 = "c0000000-0000-4000-8000-0000000000a1"
	p01CaseA2 = "c0000000-0000-4000-8000-0000000000a2"
	p01CaseA3 = "c0000000-0000-4000-8000-0000000000a3"
	p01CaseA4 = "c0000000-0000-4000-8000-0000000000a4"
	p01CaseA5 = "c0000000-0000-4000-8000-0000000000a5"
	p01CaseAx = "c0000000-0000-4000-8000-0000000000a6" // RESOLVED to A, ABSENT_IN_LATEST
	p01CaseB1 = "c0000000-0000-4000-8000-0000000000b1"
	p01CaseRe = "c0000000-0000-4000-8000-0000000000e1" // RELATED, candidate A
	p01CaseCo = "c0000000-0000-4000-8000-0000000000e2" // CONFLICT, candidate A
	p01CaseUn = "c0000000-0000-4000-8000-0000000000e3" // UNKNOWN, coincident source key
	p01CaseNR = "c0000000-0000-4000-8000-0000000000e4" // NOT_RECOVERABLE

	p01Inv1 = "10000000-0000-4000-8000-000000000001" // cust A RESOLVED, case A1 RESOLVED
	p01Inv2 = "10000000-0000-4000-8000-000000000002" // cust A RESOLVED, case A2 RESOLVED
	p01Inv3 = "10000000-0000-4000-8000-000000000003" // cust A RESOLVED, case UNKNOWN
	p01Inv4 = "10000000-0000-4000-8000-000000000004" // cust B RESOLVED, case B1 RESOLVED
	p01Inv5 = "10000000-0000-4000-8000-000000000005" // cust RELATED(A), case RELATED(A1)
	p01Inv6 = "10000000-0000-4000-8000-000000000006" // cust CONFLICT(A), case CONFLICT(A1)
	p01Inv7 = "10000000-0000-4000-8000-000000000007" // cust UNKNOWN, case A1 RESOLVED (no inference to A)
	p01Inv8 = "10000000-0000-4000-8000-000000000008" // cust A RESOLVED, case B1 RESOLVED
	p01Inv9 = "10000000-0000-4000-8000-000000000009" // cust NOT_RECOVERABLE, case NOT_RECOVERABLE

	p01Batch = "99999999-0000-4000-8000-000000000001"
)

type p01Case struct {
	id, spec, key, link string
	customer            *string
	presence            string
}

type p01Invoice struct {
	id, key                string
	customer, case_        *string
	customerLink, caseLink string
}

func sp(s string) *string { return &s }

func (e *billingEnv) seedP01(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	sha := sha256.Sum256([]byte("rm-p01"))
	casesRows := []p01Case{
		{p01CaseA1, "SYN.lead", "7001", "RESOLVED", sp(p01CustA), "PRESENT"},
		{p01CaseA2, "SYN.lead", "7002", "RESOLVED", sp(p01CustA), "PRESENT"},
		{p01CaseA3, "SYN.lead", "7003", "RESOLVED", sp(p01CustA), "PRESENT"},
		{p01CaseA4, "SYN.lead", "7004", "RESOLVED", sp(p01CustA), "PRESENT"},
		{p01CaseA5, "SYN.lead", "7005", "RESOLVED", sp(p01CustA), "PRESENT"},
		{p01CaseAx, "SYN.lead", "7006", "RESOLVED", sp(p01CustA), "ABSENT_IN_LATEST"},
		{p01CaseB1, "SYN.lead", "7101", "RESOLVED", sp(p01CustB), "PRESENT"},
		{p01CaseRe, "SYN.lead", "7201", "RELATED", sp(p01CustA), "PRESENT"},
		{p01CaseCo, "SYN.lead", "7202", "CONFLICT", sp(p01CustA), "PRESENT"},
		// Same legacy source key as A1 under another spec: a coincidence,
		// never a relationship.
		{p01CaseUn, "SYN.other", "7001", "UNKNOWN", nil, "PRESENT"},
		{p01CaseNR, "SYN.lead", "7204", "NOT_RECOVERABLE", nil, "PRESENT"},
	}
	for _, c := range casesRows {
		if _, err := e.admin.Exec(ctx, `INSERT INTO cases.work_case (case_id, workflow_type, state_namespace, state_raw, state_mapping,
			customer_id, customer_link_state, customer_link_reason, presence, imported_at, refreshed_at, version)
			VALUES ($1, 'LEGACY_LEAD', 'SYN', 'new', 'KNOWN', $2, $3, 'synthetic RM-P01 fixture', $4, now(), now(), 1)`,
			c.id, c.customer, c.link, c.presence); err != nil {
			t.Fatalf("seed case %s: %v", c.id, err)
		}
		if _, err := e.admin.Exec(ctx, `INSERT INTO cases.case_source (case_source_id, case_id, source_system, source_spec, source_key,
			first_batch_id, last_batch_id, row_sha256, presence, first_seen_at, last_seen_at)
			VALUES (gen_random_uuid(), $1, 'SYN', $2, $3, $4, $4, $5, $6, now(), now())`,
			c.id, c.spec, c.key, p01Batch, sha[:], c.presence); err != nil {
			t.Fatalf("seed case source %s: %v", c.id, err)
		}
	}
	invoices := []p01Invoice{
		{p01Inv1, "9001", sp(p01CustA), sp(p01CaseA1), "RESOLVED", "RESOLVED"},
		{p01Inv2, "9002", sp(p01CustA), sp(p01CaseA2), "RESOLVED", "RESOLVED"},
		{p01Inv3, "9003", sp(p01CustA), nil, "RESOLVED", "UNKNOWN"},
		{p01Inv4, "9004", sp(p01CustB), sp(p01CaseB1), "RESOLVED", "RESOLVED"},
		{p01Inv5, "9005", sp(p01CustA), sp(p01CaseA1), "RELATED", "RELATED"},
		{p01Inv6, "9006", sp(p01CustA), sp(p01CaseA1), "CONFLICT", "CONFLICT"},
		{p01Inv7, "7001", nil, sp(p01CaseA1), "UNKNOWN", "RESOLVED"}, // source key coincides with case A1's
		{p01Inv8, "9008", sp(p01CustA), sp(p01CaseB1), "RESOLVED", "RESOLVED"},
		{p01Inv9, "9009", nil, nil, "NOT_RECOVERABLE", "NOT_RECOVERABLE"},
	}
	for _, v := range invoices {
		if _, err := e.admin.Exec(ctx, `INSERT INTO billing.invoice (invoice_id, source_system, source_spec, source_key, first_batch_id,
			last_batch_id, row_sha256, first_seen_at, last_seen_at, presence, lifecycle_state, customer_id, customer_link_state,
			customer_link_basis, case_id, case_link_state, case_link_basis)
			VALUES ($1, 'SYN', 'SYN.invoice', $2, $3, $3, $4, now(), now(), 'PRESENT', 'UNKNOWN', $5, $6, 'synthetic RM-P01 fixture', $7, $8,
			'synthetic RM-P01 fixture')`, v.id, v.key, p01Batch, sha[:], v.customer, v.customerLink, v.case_, v.caseLink); err != nil {
			t.Fatalf("seed invoice %s: %v", v.id, err)
		}
	}
}

func flagGuard(t *testing.T, e *billingEnv, flag string) *authorization.Guard {
	t.Helper()
	flags := e.Runner(t, owner.FeatureFlags)
	g, err := authorization.NewGuard(func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, flag)
			return err
		})
		return on, err
	}, authorization.DevSubjects{})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// p01Server composes the Sales and Billing surfaces exactly as the other
// shadow suites do (each behind its own feature-flag guard and the dev
// subjects): no grant, capability or role is added for this test.
func (e *billingEnv) p01Server(t *testing.T) http.Handler {
	t.Helper()
	sg, bg := flagGuard(t, e, salesFlag), flagGuard(t, e, billingFlag)
	cfg := apiserver.DefaultConfig()
	cfg.Environment, cfg.DevAuth = authstub.Development, true
	at, _ := timeutil.ParseRFC3339("2026-11-01T10:00:00Z")
	h, err := apiserver.NewHandlerWithSurfaces(cfg, timeutil.FixedClock{At: at}, slog.New(slog.NewTextHandler(io.Discard, nil)),
		apiserver.Surfaces{
			Sales:   salesSurface{caseshttp.New(sg, e.casesQ), custodyhttp.New(sg, e.custodyQ)},
			Billing: billingSurface{billinghttp.New(bg, e.billingQ, e.paymentsQ), paymentshttp.New(bg, e.paymentsQ)},
		})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// pageAll follows next_cursor and returns every id (field) in order,
// failing on a duplicate.
func pageAll(t *testing.T, h http.Handler, base, field string) ([]string, int) {
	t.Helper()
	var ids []string
	seen := map[string]bool{}
	next, pages := base, 0
	for next != "" {
		if pages++; pages > 50 {
			t.Fatalf("%s: runaway pagination", base)
		}
		r := get(t, h, next, "dev-user-001")
		if r.code != 200 || r.json["outcome"] != "SUCCESS" {
			t.Fatalf("GET %s = %d %s", next, r.code, r.body)
		}
		for _, it := range items(r) {
			id := it[field].(string)
			if seen[id] {
				t.Fatalf("%s: duplicate %s across pages", base, id)
			}
			seen[id] = true
			ids = append(ids, id)
		}
		next = ""
		if pg, _ := r.json["page"].(map[string]any); pg != nil && pg["has_more"] == true {
			sep := "&"
			if !strings.Contains(base, "?") {
				sep = "?"
			}
			next = base + sep + "cursor=" + url.QueryEscape(pg["next_cursor"].(string))
		}
	}
	return ids, pages
}

func sorted(ids ...string) []string {
	out := slices.Clone(ids)
	sort.Strings(out)
	return out
}

func wantIDs(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, sorted(want...)) {
		t.Fatalf("%s:\n got  %v\n want %v", label, got, sorted(want...))
	}
}

func TestCustomerCaseInvoiceNavigation(t *testing.T) {
	e := setupBilling(t)
	e.seedP01(t)
	h := e.p01Server(t)

	// Flags off → BLOCKED/FEATURE_DISABLED even with a valid filter; no rows.
	for _, p := range []string{"/api/v1/cases?customer_id=" + p01CustA, "/api/v1/invoices?customer_id=" + p01CustA + "&case_id=" + p01CaseA1} {
		r := get(t, h, p, "dev-user-001")
		if r.code != 200 || !strings.Contains(r.body, `"FEATURE_DISABLED"`) || len(items(r)) != 0 {
			t.Fatalf("flag off %s: %d %s", p, r.code, r.body)
		}
	}
	e.setSalesFlag(t, true)
	e.setFlag(t, true)
	fp := e.md5(t, append(slices.Clone(salesTables), billingModel...)...)

	all := []string{p01CaseA1, p01CaseA2, p01CaseA3, p01CaseA4, p01CaseA5, p01CaseAx, p01CaseB1, p01CaseRe, p01CaseCo, p01CaseUn, p01CaseNR}
	allInv := []string{p01Inv1, p01Inv2, p01Inv3, p01Inv4, p01Inv5, p01Inv6, p01Inv7, p01Inv8, p01Inv9}

	t.Run("cases by customer", func(t *testing.T) {
		got, _ := pageAll(t, h, "/api/v1/cases?limit=200&customer_id="+p01CustA, "case_id")
		// (1) only RESOLVED-to-A Cases; RELATED/CONFLICT candidates of A excluded.
		wantIDs(t, "cases A", got, p01CaseA1, p01CaseA2, p01CaseA3, p01CaseA4, p01CaseA5, p01CaseAx)
		// (2) B never receives A's Cases.
		got, _ = pageAll(t, h, "/api/v1/cases?customer_id="+p01CustB, "case_id")
		wantIDs(t, "cases B", got, p01CaseB1)
		// Conjunctive with existing filters.
		got, _ = pageAll(t, h, "/api/v1/cases?presence=PRESENT&customer_id="+p01CustA, "case_id")
		wantIDs(t, "cases A present", got, p01CaseA1, p01CaseA2, p01CaseA3, p01CaseA4, p01CaseA5)
		got, _ = pageAll(t, h, "/api/v1/cases?customer_link_state=RESOLVED&customer_id="+p01CustA, "case_id")
		wantIDs(t, "cases A resolved", got, p01CaseA1, p01CaseA2, p01CaseA3, p01CaseA4, p01CaseA5, p01CaseAx)
		// Unknown customer: empty page, no existence signal.
		r := get(t, h, "/api/v1/cases?customer_id="+p01CustZ, "dev-user-001")
		if r.code != 200 || r.json["outcome"] != "SUCCESS" || len(items(r)) != 0 {
			t.Fatalf("unknown customer: %d %s", r.code, r.body)
		}
		// Uppercase UUID is the same native id.
		got, _ = pageAll(t, h, "/api/v1/cases?customer_id="+strings.ToUpper(p01CustB), "case_id")
		wantIDs(t, "cases B upper", got, p01CaseB1)
		// Every returned summary carries its own link state and id.
		for _, it := range items(get(t, h, "/api/v1/cases?customer_id="+p01CustA, "dev-user-001")) {
			if it["customer_link_state"] != "RESOLVED" || it["customer_id"] != p01CustA {
				t.Fatalf("summary link: %v", it)
			}
		}
	})

	t.Run("ambiguous links stay ambiguous", func(t *testing.T) {
		// (6) UNKNOWN/CONFLICT/RELATED/NOT_RECOVERABLE remain listable by
		// state, unchanged, and are never upgraded to a confirmed link.
		for state, want := range map[string][]string{"RELATED": {p01CaseRe}, "CONFLICT": {p01CaseCo},
			"UNKNOWN": {p01CaseUn}, "NOT_RECOVERABLE": {p01CaseNR}} {
			got, _ := pageAll(t, h, "/api/v1/cases?customer_link_state="+state, "case_id")
			wantIDs(t, "cases "+state, got, want...)
			r := get(t, h, "/api/v1/cases?customer_link_state="+state+"&customer_id="+p01CustA, "dev-user-001")
			if r.code != 400 || len(items(r)) != 0 {
				t.Fatalf("customer_id with %s must be 400: %d %s", state, r.code, r.body)
			}
		}
		for state, want := range map[string][]string{"RELATED": {p01Inv5}, "CONFLICT": {p01Inv6}, "UNKNOWN": {p01Inv7},
			"NOT_RECOVERABLE": {p01Inv9}} {
			got, _ := pageAll(t, h, "/api/v1/invoices?customer_link_state="+state, "invoice_id")
			wantIDs(t, "invoices customer "+state, got, want...)
			for _, p := range []string{"customer_link_state=" + state + "&customer_id=" + p01CustA, "case_link_state=" + state + "&case_id=" + p01CaseA1} {
				if r := get(t, h, "/api/v1/invoices?"+p, "dev-user-001"); r.code != 400 || len(items(r)) != 0 {
					t.Fatalf("%s must be 400: %d %s", p, r.code, r.body)
				}
			}
		}
	})

	t.Run("invoices by customer and case", func(t *testing.T) {
		// (3) by customer: own RESOLVED customer link only.
		got, _ := pageAll(t, h, "/api/v1/invoices?customer_id="+p01CustA, "invoice_id")
		wantIDs(t, "invoices A", got, p01Inv1, p01Inv2, p01Inv3, p01Inv8)
		got, _ = pageAll(t, h, "/api/v1/invoices?customer_id="+p01CustB, "invoice_id")
		wantIDs(t, "invoices B", got, p01Inv4)
		// (4) by case: inv7 has an UNKNOWN customer but a RESOLVED case A1
		// — it is listed for A1 and is NOT inferred to belong to customer A.
		got, _ = pageAll(t, h, "/api/v1/invoices?case_id="+p01CaseA1, "invoice_id")
		wantIDs(t, "invoices A1", got, p01Inv1, p01Inv7)
		got, _ = pageAll(t, h, "/api/v1/invoices?case_id="+p01CaseB1, "invoice_id")
		wantIDs(t, "invoices B1", got, p01Inv4, p01Inv8)
		// (5) both filters: AND, never OR.
		got, _ = pageAll(t, h, "/api/v1/invoices?customer_id="+p01CustA+"&case_id="+p01CaseA1, "invoice_id")
		wantIDs(t, "invoices A∧A1", got, p01Inv1)
		got, _ = pageAll(t, h, "/api/v1/invoices?customer_id="+p01CustA+"&case_id="+p01CaseB1, "invoice_id")
		wantIDs(t, "invoices A∧B1", got, p01Inv8)
		got, _ = pageAll(t, h, "/api/v1/invoices?customer_id="+p01CustB+"&case_id="+p01CaseA1, "invoice_id")
		wantIDs(t, "invoices B∧A1", got)
		got, _ = pageAll(t, h, "/api/v1/invoices?customer_id="+p01CustA+"&case_link_state=RESOLVED&presence=PRESENT", "invoice_id")
		wantIDs(t, "invoices A∧case RESOLVED", got, p01Inv1, p01Inv2, p01Inv8)
		// A Case id that only has ambiguous invoice links yields nothing.
		got, _ = pageAll(t, h, "/api/v1/invoices?case_id="+p01CaseCo, "invoice_id")
		wantIDs(t, "invoices conflict case", got)
	})

	t.Run("no identity through phone or source keys", func(t *testing.T) {
		for _, p := range []string{
			"/api/v1/cases?phone=09120000000", "/api/v1/cases?customer_phone=09120000000",
			"/api/v1/cases?customer_id=09120000000", "/api/v1/cases?customer_id=7001", "/api/v1/cases?customer_key=7001",
			"/api/v1/invoices?phone=09120000000", "/api/v1/invoices?customer_id=09120000000",
			"/api/v1/invoices?case_id=7001", "/api/v1/invoices?source_key=7001", "/api/v1/invoices?case_source_key=7001",
		} {
			if r := get(t, h, p, "dev-user-001"); r.code != 400 || len(items(r)) != 0 {
				t.Fatalf("%s must be 400: %d %s", p, r.code, r.body)
			}
		}
		// inv7 shares legacy source key 7001 with case A1 and the UNKNOWN
		// case shares it too: neither coincidence creates a link.
		got, _ := pageAll(t, h, "/api/v1/invoices?case_id="+p01CaseUn, "invoice_id")
		wantIDs(t, "coincident key case", got)
		got, _ = pageAll(t, h, "/api/v1/cases?customer_id="+p01CustA, "case_id")
		if slices.Contains(got, p01CaseUn) {
			t.Fatal("coincident source key linked a Case to a customer")
		}
	})

	t.Run("invalid uuid and cursor", func(t *testing.T) {
		for _, p := range []string{
			"/api/v1/cases?customer_id=not-a-uuid", "/api/v1/cases?customer_id=", "/api/v1/cases?customer_id=" + p01CustA + "x",
			"/api/v1/invoices?customer_id=not-a-uuid", "/api/v1/invoices?case_id=not-a-uuid", "/api/v1/invoices?case_id=",
			"/api/v1/cases?customer_id=" + p01CustA + "&cursor=garbage", "/api/v1/invoices?case_id=" + p01CaseA1 + "&cursor=garbage",
		} {
			r := get(t, h, p, "dev-user-001")
			if r.code != 400 || len(items(r)) != 0 || !strings.Contains(r.body, `"status":400`) {
				t.Fatalf("%s must be 400 problem: %d %s", p, r.code, r.body)
			}
			for _, leak := range []string{p01CaseA1, p01Inv1, "SQLSTATE", "pgx", "billing.", "cases."} {
				if strings.Contains(r.body, leak) {
					t.Fatalf("%s leaks %q: %s", p, leak, r.body)
				}
			}
		}
	})

	t.Run("pagination across filtered pages", func(t *testing.T) {
		// (8) 6 Cases of A at limit 2 → exactly 3 pages, no duplicates or gaps.
		got, pages := pageAll(t, h, "/api/v1/cases?limit=2&customer_id="+p01CustA, "case_id")
		wantIDs(t, "paged cases A", got, p01CaseA1, p01CaseA2, p01CaseA3, p01CaseA4, p01CaseA5, p01CaseAx)
		if pages != 3 {
			t.Fatalf("cases pages = %d, want 3", pages)
		}
		got, pages = pageAll(t, h, "/api/v1/invoices?limit=1&customer_id="+p01CustA, "invoice_id")
		wantIDs(t, "paged invoices A", got, p01Inv1, p01Inv2, p01Inv3, p01Inv8)
		if pages != 4 {
			t.Fatalf("invoice pages = %d, want 4", pages)
		}
		got, _ = pageAll(t, h, "/api/v1/invoices?limit=1&case_id="+p01CaseB1+"&customer_id="+p01CustA, "invoice_id")
		wantIDs(t, "paged invoices A∧B1", got, p01Inv8)
	})

	t.Run("unfiltered regression", func(t *testing.T) {
		// (9) without the new filters nothing changes.
		got, _ := pageAll(t, h, "/api/v1/cases?limit=3", "case_id")
		wantIDs(t, "all cases", got, all...)
		got, _ = pageAll(t, h, "/api/v1/invoices?limit=4", "invoice_id")
		wantIDs(t, "all invoices", got, allInv...)
		got, _ = pageAll(t, h, "/api/v1/invoices?customer_link_state=RESOLVED", "invoice_id")
		wantIDs(t, "resolved invoices", got, p01Inv1, p01Inv2, p01Inv3, p01Inv4, p01Inv8)
	})

	t.Run("authorization unchanged", func(t *testing.T) {
		// (10) A viewer without the read capability stays denied; an
		// anonymous caller never gets rows; the filter cannot widen access.
		for _, p := range []string{"/api/v1/cases?customer_id=" + p01CustA, "/api/v1/invoices?customer_id=" + p01CustA,
			"/api/v1/invoices?case_id=" + p01CaseA1, "/api/v1/invoices?customer_id=" + p01CustA + "&case_id=" + p01CaseA1} {
			if r := get(t, h, p, "dev-user-002"); r.code != 403 || len(items(r)) != 0 {
				t.Fatalf("viewer %s: %d %s", p, r.code, r.body)
			}
			if r := get(t, h, p, ""); r.code == 200 || len(items(r)) != 0 {
				t.Fatalf("anonymous %s: %d %s", p, r.code, r.body)
			}
		}
	})

	t.Run("read-only transactions", func(t *testing.T) {
		// (11) No filtered GET above changed any cases/billing row. The
		// store-level refusal of a writable transaction is asserted in the
		// module-owned tests (cases/store, billing/store: TestListRefuses...).
		if e.md5(t, append(slices.Clone(salesTables), billingModel...)...) != fp {
			t.Fatal("a filtered GET changed cases/billing state")
		}
	})

	t.Run("contract allowlist", func(t *testing.T) {
		// (12) The server allowlist carries exactly the new parameters (the
		// full spec ⇄ allowlist equality is TestQueryAllowlistMatchesContract).
		if !slices.Contains(apiserver.ContractQuery["/api/v1/cases"], "customer_id") {
			t.Fatal("cases allowlist lacks customer_id")
		}
		for _, k := range []string{"customer_id", "case_id"} {
			if !slices.Contains(apiserver.ContractQuery["/api/v1/invoices"], k) {
				t.Fatalf("invoices allowlist lacks %s", k)
			}
		}
		// RM-P06 added the exact native case_id filter to the cases list (the
		// batch read of the MIS custody pages); its exactness, bounds and
		// non-inference are proven in rm_p06_mis_evidence_test.go. Nothing
		// identity-shaped (phone, name, legacy key) is ever accepted.
		if !slices.Contains(apiserver.ContractQuery["/api/v1/cases"], "case_id") {
			t.Fatal("cases allowlist lacks the RM-P06 exact case_id filter")
		}
		for _, k := range []string{"phone", "mobile", "name", "customer_name", "q", "search", "legacy_key"} {
			if slices.Contains(apiserver.ContractQuery["/api/v1/cases"], k) {
				t.Fatalf("cases list must not accept %s", k)
			}
		}
	})
}
