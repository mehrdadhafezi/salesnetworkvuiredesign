package shadow_test

// Financial no-mutation guard shared by the R7-2 browser suite
// (TestBrowserPaymentsFullStack, build tag e2e) and its mutation regression
// below (no build tag: it runs with every PostgreSQL-backed `go test`).
//
// A snapshot records, per base table of the billing, payments and finance
// schemas, the row count and a SHA-256 fingerprint of the full content of
// every row. Rows are fingerprinted in bytewise order of their text form
// (COLLATE "C"), so physical row order, scan order and the database
// collation never affect the result; an INSERT, DELETE or UPDATE does.
// Differences are reported by table name and row count only: no row
// content, amount or fingerprint is ever printed.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// financialSchemas are the schemas whose tables the guard snapshots.
var financialSchemas = []string{"billing", "payments", "finance"}

// tableState is one table's row count and the fingerprint of its content.
type tableState struct {
	rows int64
	sum  string
}

// financialSnapshot snapshots every base table of the financial schemas.
func financialSnapshot(t *testing.T, admin *pgx.Conn) map[string]tableState {
	t.Helper()
	return schemaSnapshot(t, admin, financialSchemas)
}

// schemaSnapshot snapshots every base table of schemas the same way (R7-3
// adds the fulfillment, biawin, catalog and integration schemas).
func schemaSnapshot(t *testing.T, admin *pgx.Conn, schemas []string) map[string]tableState {
	t.Helper()
	ctx := context.Background()
	rows, err := admin.Query(ctx, `SELECT quote_ident(table_schema) || '.' || quote_ident(table_name) FROM information_schema.tables
		WHERE table_type = 'BASE TABLE' AND table_schema = ANY($1) ORDER BY 1`, schemas)
	if err != nil {
		t.Fatal("list guarded tables failed") // the driver error is never printed
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal("list financial tables failed") // the driver error is never printed
		}
		tables = append(tables, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal("list financial tables failed") // the driver error is never printed
	}
	out := make(map[string]tableState, len(tables))
	for _, n := range tables {
		// n is a quoted identifier from information_schema of the disposable database.
		var st tableState
		if err := admin.QueryRow(ctx, `SELECT count(*),
			coalesce(encode(sha256(convert_to(string_agg(r::text, E'\n' ORDER BY r::text COLLATE "C"), 'UTF8')), 'hex'), '')
			FROM `+n+` r`).Scan(&st.rows, &st.sum); err != nil {
			t.Fatalf("snapshot %s failed", n) // the driver error may quote row data; never print it
		}
		out[n] = st
	}
	return out
}

// missingSeed lists each required table that is absent from the snapshot or
// holds no row. Messages carry table names and row counts only: never the
// snapshot itself, which holds the fingerprints.
func missingSeed(snap map[string]tableState, required ...string) []string {
	var out []string
	for _, table := range required {
		st, ok := snap[table]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s: table missing", table))
		case st.rows == 0:
			out = append(out, fmt.Sprintf("%s: 0 rows", table))
		}
	}
	return out
}

// diffFinancial lists every difference between two snapshots: a changed row
// count, changed row content with an unchanged count, or a table that
// appeared or disappeared. Messages carry table names and counts only.
func diffFinancial(before, after map[string]tableState) []string {
	var out []string
	for table, b := range before {
		a, ok := after[table]
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s disappeared", table))
		case a.rows != b.rows:
			out = append(out, fmt.Sprintf("%s: %d rows before, %d after", table, b.rows, a.rows))
		case a.sum != b.sum:
			out = append(out, fmt.Sprintf("%s: row content changed (row count %d unchanged)", table, b.rows))
		}
	}
	for table := range after {
		if _, ok := before[table]; !ok {
			out = append(out, fmt.Sprintf("%s appeared", table))
		}
	}
	sort.Strings(out)
	return out
}

// TestFinancialSnapshotDetectsUpdate is the mutation regression of the
// guard: on the synthetic Slice 12 payments scenario, an UPDATE that changes
// one financial value without changing any row count must be reported, and
// physical reordering of identical content must not be.
func TestFinancialSnapshotDetectsUpdate(t *testing.T) {
	e := setupPayments(t) // disposable DB + SYNTHETIC payments scenario (dropped on cleanup)
	ctx := context.Background()
	before := financialSnapshot(t, e.admin)
	if m := missingSeed(before, `billing.invoice`, `payments.payment_attempt`); len(m) != 0 {
		t.Fatalf("the synthetic scenario is incomplete: %v", m)
	}
	// The seed report names tables and counts only, never a fingerprint.
	seed := map[string]tableState{`billing.invoice`: before[`billing.invoice`], `payments.payment_attempt`: {rows: 0, sum: before[`payments.payment_attempt`].sum}}
	m := missingSeed(seed, `billing.invoice`, `payments.payment_attempt`, `finance.absent_table`)
	if want := []string{"payments.payment_attempt: 0 rows", "finance.absent_table: table missing"}; len(m) != 2 || m[0] != want[0] || m[1] != want[1] {
		t.Fatalf("missingSeed = %v, want %v", m, want)
	}
	if containsAny(m, before[`billing.invoice`].sum, before[`payments.payment_attempt`].sum) {
		t.Fatal("the seed report must not carry a fingerprint")
	}
	if d := diffFinancial(before, financialSnapshot(t, e.admin)); len(d) != 0 {
		t.Fatalf("an unchanged database must give an identical snapshot: %v", d)
	}

	// Order independence: rewrite the first row of a multi-row table to the
	// same content. PostgreSQL writes a new tuple version, which moves the row
	// to the end of the heap scan; the content is unchanged, so the snapshot
	// must be unchanged. The scan order is checked to have really changed, so
	// this step can never pass vacuously.
	if before[`billing.invoice`].rows < 2 {
		t.Fatal("order independence needs a financial table with at least two rows")
	}
	scanOrder := func() string {
		var s string
		if err := e.admin.QueryRow(ctx, `SELECT string_agg(source_key, ',') FROM (SELECT source_key FROM billing.invoice) q`).Scan(&s); err != nil {
			t.Fatalf("read scan order failed")
		}
		return s
	}
	order := scanOrder()
	if _, err := e.admin.Exec(ctx, `UPDATE billing.invoice SET source_key = source_key
		WHERE ctid = (SELECT ctid FROM billing.invoice ORDER BY ctid LIMIT 1)`); err != nil {
		t.Fatalf("rewrite in place failed")
	}
	if scanOrder() == order {
		t.Fatal("the rewrite did not change the physical scan order; the order-independence check would be vacuous")
	}
	if d := diffFinancial(before, financialSnapshot(t, e.admin)); len(d) != 0 {
		t.Fatalf("physical reordering of identical content must not be reported: %v", d)
	}

	// The mutation: one financial value changes, no row is added or removed.
	tag, err := e.admin.Exec(ctx, `UPDATE payments.payment_attempt SET amount = amount + 1
		WHERE ctid = (SELECT ctid FROM payments.payment_attempt WHERE amount IS NOT NULL ORDER BY ctid LIMIT 1)`)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("planted UPDATE did not apply to exactly one row")
	}
	after := financialSnapshot(t, e.admin)
	if after[`payments.payment_attempt`].rows != before[`payments.payment_attempt`].rows {
		t.Fatal("the planted UPDATE must not change the row count")
	}
	d := diffFinancial(before, after)
	want := []string{fmt.Sprintf("payments.payment_attempt: row content changed (row count %d unchanged)", before[`payments.payment_attempt`].rows)}
	if len(d) != 1 || d[0] != want[0] {
		t.Fatalf("diff = %v, want %v", d, want)
	}
	// The report names the table and the count only: never a fingerprint.
	if s := after[`payments.payment_attempt`].sum; s == "" || containsAny(d, s, before[`payments.payment_attempt`].sum) {
		t.Fatal("the difference report must not carry a fingerprint")
	}
}

func containsAny(msgs []string, subs ...string) bool {
	for _, m := range msgs {
		for _, sub := range subs {
			if sub != "" && strings.Contains(m, sub) {
				return true
			}
		}
	}
	return false
}
