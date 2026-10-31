package imports_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/imports"
	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

type env struct {
	*pgtest.Env
	admin *pgx.Conn
	cmd   *imports.Command
	q     *query.Service
	r     *db.Runner
}

func setup(t *testing.T) *env {
	t.Helper()
	e := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, e.DSN, customersmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	at, _ := timeutil.ParseRFC3339("2026-10-31T09:00:00Z")
	r := e.Runner(t, customers.Module)
	return &env{Env: e, admin: admin, cmd: imports.New(r, timeutil.FixedClock{At: at}), q: query.New(r), r: r}
}

func (e *env) count(t *testing.T, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := e.admin.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

const batch1 = "11111111-1111-4111-8111-111111111111"

func src(batch, key string) api.SourceRef {
	sum := sha256.Sum256([]byte(batch + "/" + key))
	return api.SourceRef{System: "wp", Spec: "synthetic.subject.v1", Key: key, BatchID: batch, RowSHA256: sum[:]}
}

func str(s string) *string { return &s }

// subject builds a SYNTHETIC subject with one synthetic phone and optional window.
func subject(batch, key, phone string, from, to *string, ids ...api.IdentifierRecord) api.SubjectRecord {
	return api.SubjectRecord{Kind: api.SyntheticSubject, Source: src(batch, key), Name: str("Synthetic Subject " + key), Recoverable: true,
		Contacts: []api.ContactRecord{{Kind: api.Phone, Role: api.Primary, Raw: phone, ValidFromRaw: from, ValidToRaw: to}}, Identifiers: ids}
}

func (e *env) run(t *testing.T, in api.CustomerImport) api.ImportReport {
	t.Helper()
	if in.BatchID == "" {
		in.BatchID = batch1
	}
	in.SnapshotID, in.SourceSystem = "synthetic-customers-01", "wp"
	rep, err := e.cmd.ImportCustomers(context.Background(), in)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return rep
}

func (e *env) report(t *testing.T) api.Resolution {
	t.Helper()
	r, err := e.q.ResolutionReport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.PhoneOnlyResolved != 0 || r.Merges != 0 || r.PhoneOnlyMerges != 0 {
		t.Fatalf("merge invariants broken: %+v", r)
	}
	return r
}

// Two distinct source subjects with the same phone stay two customers.
func TestSamePhoneNeverMerges(t *testing.T) {
	e := setup(t)
	e.run(t, api.CustomerImport{Complete: true, Subjects: []api.SubjectRecord{
		subject(batch1, "1", "0900 000 0001", nil, nil), subject(batch1, "2", "+98 900 000 0001", nil, nil)}})
	if n := e.count(t, `SELECT count(*) FROM customers.customer`); n != 2 {
		t.Fatalf("customers = %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.identity_evidence WHERE kind = 'PHONE_MATCH' AND strength = 'WEAK'`); n != 1 {
		t.Fatal("phone match evidence missing")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE reason = 'SHARED_PHONE' AND state = 'UNKNOWN'`); n != 1 {
		t.Fatal("phone-only candidate must be UNKNOWN")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE resolution_state = 'RESOLVED'`); n != 0 {
		t.Fatal("resolved by phone")
	}
	r := e.report(t)
	if r.PhoneOnlyCandidates != 1 || r.ByState["UNKNOWN"] != 2 {
		t.Fatalf("report = %+v", r)
	}
}

func TestSequentialAndOverlappingOwnership(t *testing.T) {
	e := setup(t)
	e.run(t, api.CustomerImport{Complete: true, Subjects: []api.SubjectRecord{
		subject(batch1, "1", "09000000002", str("2024-01-01 00:00:00"), str("2024-06-30 00:00:00")),
		subject(batch1, "2", "09000000002", str("2025-01-01 00:00:00"), nil),
		subject(batch1, "3", "09000000003", str("2025-01-01 00:00:00"), str("2025-12-31 00:00:00")),
		subject(batch1, "4", "09000000003", str("2025-06-01 00:00:00"), nil),
	}})
	if n := e.count(t, `SELECT count(*) FROM customers.customer`); n != 4 {
		t.Fatalf("customers = %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM customers.identity_evidence WHERE kind = 'SEQUENTIAL_PHONE_REUSE'`); n != 1 {
		t.Fatal("sequential reuse not recorded")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case k JOIN customers.merge_case_evidence l USING (merge_case_id)
		JOIN customers.identity_evidence v USING (evidence_id) WHERE v.kind = 'SEQUENTIAL_PHONE_REUSE' AND k.state = 'UNKNOWN'`); n != 1 {
		t.Fatal("sequential ownership must stay a distinct, unmerged UNKNOWN candidate")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE state = 'CONFLICT'`); n != 1 {
		t.Fatal("overlapping ownership must be CONFLICT")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE resolution_state = 'CONFLICT'`); n != 2 {
		t.Fatal("both overlapping owners must be CONFLICT")
	}
	if r := e.report(t); r.OverlappingPhoneConflict != 1 {
		t.Fatalf("report = %+v", r)
	}
}

// OTP proves contact control only: MEDIUM, RELATED, never RESOLVED.
func TestOTPIsContactControlNotEquality(t *testing.T) {
	e := setup(t)
	e.run(t, api.CustomerImport{Complete: true,
		Subjects: []api.SubjectRecord{subject(batch1, "1", "09000000004", nil, nil), subject(batch1, "2", "09000000004", nil, nil)},
		Evidence: []api.EvidenceRecord{{Kind: api.OTPContactControl, Source: src(batch1, "901"), SubjectKeys: []string{"1"}, ContactRaw: str("09000000004")}}})
	if n := e.count(t, `SELECT count(*) FROM customers.identity_evidence WHERE kind = 'OTP_CONTACT_CONTROL' AND strength = 'MEDIUM'`); n != 1 {
		t.Fatal("OTP evidence must be MEDIUM")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE state = 'RELATED'`); n != 1 {
		t.Fatal("OTP + phone must be RELATED")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer`); n != 2 || e.count(t, `SELECT count(*) FROM customers.customer WHERE resolution_state = 'RESOLVED'`) != 0 {
		t.Fatal("OTP merged or resolved customers")
	}
	e.report(t)
}

// Only explicit STRONG evidence (synthetic) resolves — and still no merge.
func TestStrongEvidenceResolvesWithoutMerging(t *testing.T) {
	e := setup(t)
	e.run(t, api.CustomerImport{Complete: true,
		Subjects: []api.SubjectRecord{subject(batch1, "1", "09000000005", nil, nil), subject(batch1, "2", "09000000006", nil, nil)},
		Evidence: []api.EvidenceRecord{{Kind: api.VerifiedEquality, Source: src(batch1, "902"), SubjectKeys: []string{"1", "2"}}}})
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE state = 'RESOLVED' AND max_strength = 'STRONG'`); n != 1 {
		t.Fatal("strong equality must resolve the candidate")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE resolution_state = 'RESOLVED'`); n != 2 {
		t.Fatal("both subjects must project RESOLVED")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer`); n != 2 {
		t.Fatal("RESOLVED executed a merge")
	}
	r := e.report(t)
	if r.StrongEvidence != 1 {
		t.Fatalf("report = %+v", r)
	}
}

// One exclusive identifier on two customers: CONFLICT, never reassigned.
func TestDuplicateExclusiveIdentifierConflicts(t *testing.T) {
	e := setup(t)
	woo := api.IdentifierRecord{Scheme: api.WooCustomer, Value: "SYN-WOO-1", Derivation: api.Direct}
	acct := api.IdentifierRecord{Scheme: api.LegacyWPAccount, Value: "77", Derivation: api.PhoneDerived}
	e.run(t, api.CustomerImport{Complete: true, Subjects: []api.SubjectRecord{
		subject(batch1, "1", "09000000007", nil, nil, woo), subject(batch1, "2", "09000000008", nil, nil, woo),
		subject(batch1, "3", "09000000009", nil, nil, acct), subject(batch1, "4", "09000000010", nil, nil, acct)}})
	if n := e.count(t, `SELECT count(*) FROM customers.customer_identifier WHERE scheme = 'WOO_CUSTOMER' AND status = 'CONFLICT'`); n != 2 {
		t.Fatal("both claims of the exclusive identifier must be CONFLICT")
	}
	if n := e.count(t, `SELECT count(DISTINCT customer_id) FROM customers.customer_identifier WHERE scheme = 'WOO_CUSTOMER'`); n != 2 {
		t.Fatal("identifier reassigned")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE reason = 'SHARED_EXCLUSIVE_IDENTIFIER' AND state = 'CONFLICT'`); n != 1 {
		t.Fatal("exclusive identifier conflict case missing")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE reason = 'SHARED_ACCOUNT_REF' AND state = 'RELATED'`); n != 1 {
		t.Fatal("shared phone-derived account must be RELATED, not equality")
	}
	if r := e.report(t); r.DuplicateExternalID != 1 {
		t.Fatalf("report = %+v", r)
	}
}

func TestReplayAbsenceAndNoDelete(t *testing.T) {
	e := setup(t)
	in := api.CustomerImport{Complete: true, Subjects: []api.SubjectRecord{
		subject(batch1, "1", "09000000011", nil, nil), subject(batch1, "2", "09000000011", nil, nil)}}
	e.run(t, in)
	fp := func() string {
		var s string
		var parts []string
		for _, tbl := range []string{"customer", "contact_point", "customer_identifier", "identity_evidence", "evidence_subject", "merge_case", "merge_case_member", "merge_case_evidence"} {
			if err := e.admin.QueryRow(context.Background(), `SELECT COALESCE(md5(string_agg(row_to_json(x)::text, '|' ORDER BY row_to_json(x)::text)), '')
				FROM customers.`+tbl+` x`).Scan(&s); err != nil {
				t.Fatal(err)
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, ",")
	}
	before := fp()
	if rep := e.run(t, in); !rep.Replay {
		t.Fatalf("replay not detected: %+v", rep.Counts)
	}
	if fp() != before {
		t.Fatal("replay changed customer data")
	}

	// An INCOMPLETE batch without subject 2 must not mark it absent.
	b2 := "22222222-2222-4222-8222-222222222222"
	e.run(t, api.CustomerImport{BatchID: b2, Complete: false, Subjects: []api.SubjectRecord{subject(b2, "1", "09000000011", nil, nil)}})
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE presence = 'ABSENT_IN_LATEST'`); n != 0 {
		t.Fatal("incomplete batch marked a customer absent")
	}
	// A COMPLETE but empty batch marks everything absent and deletes nothing.
	b3 := "33333333-3333-4333-8333-333333333333"
	e.run(t, api.CustomerImport{BatchID: b3, Complete: true})
	if n := e.count(t, `SELECT count(*) FROM customers.customer WHERE presence = 'PRESENT'`); n != 0 {
		t.Fatal("complete empty batch left customers present")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.customer`); n != 2 {
		t.Fatal("customers deleted")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.identity_evidence`); n == 0 {
		t.Fatal("evidence deleted")
	}
	if n := e.count(t, `SELECT count(*) FROM customers.merge_case WHERE presence = 'PRESENT'`); n != 0 {
		t.Fatal("stale candidate still queued")
	}

	// Even the owner cannot delete or rewrite evidence / customers.
	for _, q := range []string{`DELETE FROM customers.customer`, `DELETE FROM customers.identity_evidence`,
		`UPDATE customers.identity_evidence SET kind = 'VERIFIED_EQUALITY'`, `UPDATE customers.evidence_subject SET customer_id = customer_id`} {
		err := e.r.InTx(context.Background(), db.TxOptions{}, func(ctx context.Context, tx db.Tx) error { _, err := tx.Exec(ctx, q); return err })
		if err == nil || !strings.Contains(err.Error(), "permission denied") {
			t.Fatalf("%s: %v", q, err)
		}
	}
}

func TestSchemaHasNoPhoneKeys(t *testing.T) {
	e := setup(t)
	n := e.count(t, `SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid = i.indrelid JOIN pg_namespace s ON s.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
		WHERE s.nspname = 'customers' AND (i.indisunique OR i.indisprimary)
		  AND a.attname IN ('normalized_value', 'external_value', 'name_evidence')
		  AND NOT (c.relname IN ('contact_point', 'customer_identifier') AND array_length(i.indkey, 1) > 2)`)
	if n != 0 {
		t.Fatalf("%d unique/primary keys on contact or identifier values", n)
	}
	// The per-customer uniqueness (customer, kind, role, raw) is allowed; a global phone uniqueness is not.
	if n := e.count(t, `SELECT count(*) FROM pg_index i JOIN pg_class c ON c.oid = i.indrelid JOIN pg_namespace s ON s.oid = c.relnamespace
		WHERE s.nspname = 'customers' AND i.indisunique AND c.relname = 'contact_point'
		AND NOT EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND a.attnum = ANY(i.indkey) AND a.attname = 'customer_id')
		AND NOT EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND a.attnum = ANY(i.indkey) AND a.attname = 'contact_point_id')`); n != 0 {
		t.Fatal("global unique constraint on contact points")
	}
	_ = fmt.Sprint
}
