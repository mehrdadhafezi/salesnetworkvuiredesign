package shadow_test

import (
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"

	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
)

// RM-P02: the Finance review queue filtered to one invoice. Attempts are
// created through the real payment-submission command (synthetic legacy,
// disposable PostgreSQL). invoice_id matches the attempt's own native
// invoice reference only; the filter never widens access, never changes
// the D-43 / maker-checker gate and never allows a decision.
func TestFinanceReviewQueueInvoiceFilter(t *testing.T) {
	e := setupPayments(t)
	seller := e.actor("6")
	inv1, inv5, other := e.invoices["3001"], e.invoices["3005"], e.invoices["3002"]
	if r := e.submit(t, batch("p02-review-0001", inv1, 0, seller, item("a", 300, e.upload(t, "6", 81)), item("b", 200, e.upload(t, "6", 82)))); r.Status != result.Success {
		t.Fatalf("submit 3001: %+v", r)
	}
	if r := e.submit(t, batch("p02-review-0002", inv5, 0, seller, item("a", 100, e.upload(t, "6", 83)))); r.Status != result.Success {
		t.Fatalf("submit 3005: %+v", r)
	}
	h := e.server(t, testSubjects{users: e.users, caps: map[string][]identityapi.Capability{"1": {identityapi.CapFinanceReviewRead}}})

	// Flag off → BLOCKED even with a valid filter; no rows.
	if r := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts?invoice_id="+inv1, "1", "", nil); r.code != 200 ||
		r.json["outcome"] != "BLOCKED" || r.json["data"] != nil {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.setFlags(t, map[string]bool{"finance.review_read": true})
	fp := e.md5(t, "payments.submitted_attempt", "payments.payment_batch", "payments.receipt", "payments.receipt_state_history", "payments.upload_object")

	walk := func(base string) ([]string, int) {
		t.Helper()
		var ids []string
		next, pages := base, 0
		for next != "" {
			if pages++; pages > 20 {
				t.Fatal("runaway pagination")
			}
			r := call(t, h, http.MethodGet, next, "1", "", nil)
			if r.code != 200 || r.json["outcome"] != "SUCCESS" {
				t.Fatalf("GET %s = %d %s", next, r.code, r.body)
			}
			data, _ := r.json["data"].([]any)
			for _, x := range data {
				m := x.(map[string]any)
				if q, _ := url.Parse(base); q.Query().Get("invoice_id") != "" && m["invoice_id"] != q.Query().Get("invoice_id") {
					t.Fatalf("%s returned another invoice's attempt: %v", base, m["invoice_id"])
				}
				id := m["attempt_id"].(string)
				if slices.Contains(ids, id) {
					t.Fatalf("%s: duplicate %s across pages", base, id)
				}
				ids = append(ids, id)
				// The gate is unchanged by the filter: no decision is ever allowed.
				if !strings.Contains(r.body, `"D-41"`) {
					t.Fatalf("gate missing: %s", r.body)
				}
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

	all, _ := walk("/api/v1/finance/review/attempts")
	of1, _ := walk("/api/v1/finance/review/attempts?invoice_id=" + inv1)
	of5, _ := walk("/api/v1/finance/review/attempts?invoice_id=" + inv5)
	if len(all) != 3 || len(of1) != 2 || len(of5) != 1 {
		t.Fatalf("all=%d 3001=%d 3005=%d", len(all), len(of1), len(of5))
	}
	union := append(slices.Clone(of1), of5...)
	sort.Strings(union)
	sorted := slices.Clone(all)
	sort.Strings(sorted)
	if !slices.Equal(union, sorted) {
		t.Fatalf("filtered pages must partition the queue: %v vs %v", union, sorted)
	}
	// Pagination across filtered pages.
	paged, pages := walk("/api/v1/finance/review/attempts?limit=1&invoice_id=" + inv1)
	if pages != 2 || !slices.Equal(paged, of1) {
		t.Fatalf("paged 3001: %v in %d pages (want %v in 2)", paged, pages, of1)
	}
	// An invoice without submitted attempts, and an unknown id: empty, not an error.
	for _, id := range []string{other, "00000000-0000-4000-8000-0000000000ff"} {
		if ids, _ := walk("/api/v1/finance/review/attempts?invoice_id=" + id); len(ids) != 0 {
			t.Fatalf("invoice %s: %v", id, ids)
		}
	}
	// Malformed ids, phone or legacy keys: contract 400, never data.
	for _, p := range []string{"invoice_id=not-a-uuid", "invoice_id=3001", "invoice_id=09120000000", "invoice_key=3001", "phone=09120000000"} {
		if r := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts?"+p, "1", "", nil); r.code != 400 || r.json["data"] != nil {
			t.Fatalf("%s must be 400: %d %s", p, r.code, r.body)
		}
	}
	// No capability → 403 with the filter too; the submitter-only payment caps do not grant review.
	for _, who := range []string{"9", "6"} {
		if r := call(t, h, http.MethodGet, "/api/v1/finance/review/attempts?invoice_id="+inv1, who, "", nil); r.code != 403 || r.json["data"] != nil {
			t.Fatalf("%s without review read: %d %s", who, r.code, r.body)
		}
	}
	// Read only.
	if e.md5(t, "payments.submitted_attempt", "payments.payment_batch", "payments.receipt", "payments.receipt_state_history", "payments.upload_object") != fp {
		t.Fatal("a review read changed payments state")
	}
	if !slices.Contains(apiserver.ContractQuery["/api/v1/finance/review/attempts"], "invoice_id") {
		t.Fatal("allowlist lacks invoice_id")
	}
}
