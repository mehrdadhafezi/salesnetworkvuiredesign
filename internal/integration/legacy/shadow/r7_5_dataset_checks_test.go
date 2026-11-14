package shadow_test

// R7-5 dataset checks: determinism, referential integrity, repeat execution,
// test-database safety, refusals and the opt-in package export. See
// r7_5_integrated_dataset_test.go for the seed pipeline.
//
// BACKEND-ONLY COVERAGE: reads use the existing development auth stub
// (authorization.DevSubjects) exactly as the module suites do; nothing here
// is evidence of production authorization.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/legacytest"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/shadow"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/snapshot"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/spec"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
)

// ---- source level (no database) ----------------------------------------------

// The composed source and the written package are byte-identical across runs.
func TestR75IntegratedPackageDeterministic(t *testing.T) {
	if !reflect.DeepEqual(legacytest.IntegratedScenario(), legacytest.IntegratedScenario()) {
		t.Fatal("IntegratedScenario is not deterministic")
	}
	a := legacytest.Write(t, legacytest.IntegratedSnapshotID, legacytest.IntegratedExtractedAt, legacytest.IntegratedScenario())
	b := legacytest.Write(t, legacytest.IntegratedSnapshotID, legacytest.IntegratedExtractedAt, legacytest.IntegratedScenario())
	fa, fb := packageFiles(t, a), packageFiles(t, b)
	if len(fa) == 0 || !reflect.DeepEqual(fa, fb) {
		t.Fatalf("package files differ between two writes (%d vs %d files)", len(fa), len(fb))
	}
	p, err := snapshot.Open(a, spec.Default())
	if err != nil {
		t.Fatal(err)
	}
	if p.Manifest.DataOrigin != snapshot.OriginSynthetic || p.Manifest.SnapshotID != legacytest.IntegratedSnapshotID {
		t.Fatalf("manifest: origin %q id %q", p.Manifest.DataOrigin, p.Manifest.SnapshotID)
	}
	t.Logf("dataset %s: %d package files, content identity %s", legacytest.IntegratedVersion, len(fa), p.Manifest.ContentIdentity())
}

// packageFiles maps every file of a package directory to its SHA-256.
func packageFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		sum := sha256.Sum256(b)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

// column returns every non-NULL value of spec.column in the composed source.
func column(ents []snapshot.EntityRows, specID, col string) []string {
	s, err := spec.Default().Lookup(specID)
	if err != nil {
		return nil
	}
	idx := -1
	for i, c := range s.Columns {
		if c.Name == col {
			idx = i
		}
	}
	var out []string
	for _, e := range ents {
		if e.SpecID != specID || idx < 0 {
			continue
		}
		for _, r := range e.Rows {
			if !r[idx].Null && r[idx].Text != "" {
				out = append(out, r[idx].Text)
			}
		}
	}
	return out
}

func set(vals ...[]string) map[string]bool {
	m := map[string]bool{}
	for _, vs := range vals {
		for _, v := range vs {
			m[v] = true
		}
	}
	return m
}

var (
	reservedPhone = regexp.MustCompile(`^(09000000\d{3}|\+98 900 000 0\d{3}|TEST-PHONE-[A-Z]?\d{3,4})$`)
	syntheticMail = regexp.MustCompile(`@example\.invalid$`)
)

// Every cross-entity reference in the composed source resolves, except the
// references the fixtures leave unresolved on purpose; contact data is
// unmistakably synthetic.
func TestR75IntegratedSourceIntegrity(t *testing.T) {
	ents := legacytest.IntegratedScenario()
	invoices := set(column(ents, "legacy.invoice.v2", "id"), legacytest.IntegratedKnownUnresolved["invoice"])
	wallets := set(column(ents, "legacy.wallet.v1", "id"))
	refs := []struct{ spec, col string }{
		{"legacy.invoice_item.v1", "invoice_id"}, {"legacy.invoice_payment_stage.v1", "invoice_id"}, {"legacy.payment.v1", "invoice_id"},
		{"legacy.wallet_transaction.v1", "invoice_id"}, {"legacy.project_membership.v1", "invoice_id"},
		{"legacy.commission_dry_run_item.v1", "invoice_id"}, {"legacy.commission_posting_log.v1", "invoice_id"},
	}
	checked := 0
	for _, r := range refs {
		vals := column(ents, r.spec, r.col)
		checked += len(vals)
		for _, v := range vals {
			if !invoices[v] {
				t.Errorf("%s.%s = %s: invoice not in the dataset and not a documented unresolved reference", r.spec, r.col, v)
			}
		}
	}
	for _, v := range column(ents, "legacy.wallet_transaction.v1", "wallet_id") {
		checked++
		if !wallets[v] {
			t.Errorf("wallet transaction names wallet %s, which is not in the dataset", v)
		}
	}
	if checked == 0 {
		t.Fatal("no reference was checked (column names drifted?)")
	}
	reg := spec.Default()
	for _, e := range ents {
		s, _ := reg.Lookup(e.SpecID)
		for i, c := range s.Columns {
			lower := strings.ToLower(c.Name)
			for _, r := range e.Rows {
				v := r[i]
				if v.Null || v.Text == "" {
					continue
				}
				switch {
				case strings.Contains(lower, "phone") && !reservedPhone.MatchString(v.Text):
					t.Errorf("%s.%s: %q is not a reserved synthetic phone", e.SpecID, c.Name, v.Text)
				case strings.Contains(lower, "email") && !syntheticMail.MatchString(v.Text):
					t.Errorf("%s.%s: %q is not an example.invalid address", e.SpecID, c.Name, v.Text)
				case strings.Contains(lower, "pass") || strings.Contains(lower, "secret") || strings.Contains(lower, "token"):
					t.Errorf("%s.%s: credential-like column carries a value", e.SpecID, c.Name)
				}
			}
		}
	}
}

// ---- seeded database -------------------------------------------------------

// r75RuntimeColumns are derived at runtime (hashes over generated UUIDs and
// ordinals numbered in UUID order); they are excluded from the logical
// fingerprint and documented in docs/remediation/r7-5-synthetic-dataset.md.
var r75RuntimeColumns = []string{"input_sha256", "facts_sha256", "snapshot_key", "finding_key", "evidence_key", "finding_no", "recon_no", "entry_seq"}

var uuidRe = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// r75Anchors are tables whose generated UUID primary key belongs to a legacy
// source key; the fingerprint replaces those UUIDs with "table:spec:key", so
// references are compared by meaning, not by generated value.
var r75Anchors = [][2]string{
	{"identity.user_account", "user_id"}, {"organization.person", "person_id"}, {"organization.employment", "employment_id"},
	{"organization.position", "position_id"}, {"organization.level", "level_id"}, {"organization.org_unit", "org_unit_id"},
	{"organization.org_assignment", "assignment_id"}, {"organization.org_assignment_history", "history_id"},
	{"organization.position_role_default", "default_id"}, {"customers.customer", "customer_id"}, {"customers.contact_point", "contact_point_id"},
	{"customers.customer_identifier", "identifier_id"}, {"customers.identity_evidence", "evidence_id"},
	{"cases.case_source", "case_source_id"}, {"cases.case_state_history", "history_id"}, {"custody.custody_event", "event_id"},
	{"billing.invoice", "invoice_id"}, {"billing.invoice_item", "item_id"}, {"billing.payment_stage", "stage_id"},
	{"payments.payment_attempt", "attempt_id"}, {"payments.receipt", "receipt_id"},
}

func (e *integratedEnv) labels(t *testing.T) map[string]string {
	t.Helper()
	ctx := context.Background()
	out := map[string]string{}
	add := func(q string) {
		rows, err := e.admin.Query(ctx, q)
		if err != nil {
			t.Fatalf("labels: %s: %v", q, err)
		}
		defer rows.Close()
		for rows.Next() {
			var id, label string
			if err := rows.Scan(&id, &label); err != nil {
				t.Fatal(err)
			}
			if prev, ok := out[id]; ok && prev != label {
				t.Fatalf("uuid %s labelled twice (%s, %s)", id, prev, label)
			}
			out[id] = label
		}
	}
	for _, a := range r75Anchors {
		specExpr := "''"
		var has bool
		if err := e.admin.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema || '.' || table_name = $1
			AND column_name = 'source_spec')`, a[0]).Scan(&has); err != nil {
			t.Fatal(err)
		}
		if has {
			specExpr = "source_spec"
		}
		add(`SELECT ` + a[1] + `::text, '` + a[0] + `:' || ` + specExpr + ` || ':' || source_key FROM ` + a[0] + ` WHERE source_key IS NOT NULL`)
	}
	// A native case is named by its (first) legacy source.
	add(`SELECT case_id::text, 'cases.work_case:' || min(source_spec || ':' || source_key) FROM cases.case_source GROUP BY case_id`)
	return out
}

// tables lists every non-empty base table of the dataset schemas.
func (e *integratedEnv) tables(t *testing.T) []string {
	t.Helper()
	ctx := context.Background()
	rows, err := e.admin.Query(ctx, `SELECT table_schema || '.' || table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE'
		AND table_schema = ANY($1) ORDER BY 1`, r75Schemas)
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		all = append(all, n)
	}
	rows.Close()
	var out []string
	for _, tbl := range all {
		if e.count(t, `SELECT count(*) FROM `+tbl) > 0 {
			out = append(out, tbl)
		}
	}
	return out
}

// logical fingerprints every non-empty table: rows as JSON without runtime
// columns, anchored UUIDs replaced by their source label, other UUIDs by "U",
// sorted and hashed. Equal fingerprints = the same logical records and
// references.
func (e *integratedEnv) logical(t *testing.T) map[string]string {
	t.Helper()
	ctx := context.Background()
	labels := e.labels(t)
	out := map[string]string{}
	for _, tbl := range e.tables(t) {
		rows, err := e.admin.Query(ctx, `SELECT (to_jsonb(x) - $1::text[])::text FROM `+tbl+` x`, r75RuntimeColumns)
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, uuidRe.ReplaceAllStringFunc(v, func(u string) string {
				if l, ok := labels[u]; ok {
					return l
				}
				return "U"
			}))
		}
		rows.Close()
		sort.Strings(lines)
		sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
		out[tbl] = hex.EncodeToString(sum[:8])
	}
	return out
}

// r75Steps is the pinned outcome of every import step on a fresh database.
// PARTIAL is the fixtures' own evidence (payments: an attempt on unstaged
// invoice 2999; commission: unverified routes), never a seed failure.
var r75Steps = []seedStep{{"billing", result.Success, false}, {"payments", result.Partial, false}, {"biawin", result.Success, false},
	{"fulfillment", result.Success, false}, {"finance", result.Success, false}, {"commission", result.Partial, false},
	{"reporting", result.Success, false}}

// r75Written is the exact set of tables the seed may populate. Any other
// non-empty table (a credential, session, HR request, payment submission,
// work case, provider operation, outbox or write-back row, an enabled flag)
// fails the seed.
var r75Written = []string{
	"billing.amount_evidence", "billing.import_run", "billing.invoice", "billing.invoice_item", "billing.invoice_state_evidence",
	"billing.payment_stage",
	"biawin.import_run", "biawin.item_invoice_link", "biawin.parity_check", "biawin.parity_row", "biawin.run_source", "biawin.subscription",
	"biawin.subscription_event", "biawin.subscription_item",
	"cases.attribution_snapshot", "cases.case_alias", "cases.case_source", "cases.case_state_history", "cases.import_run", "cases.lead",
	"cases.work_case",
	"commission.calculation", "commission.entitlement_identity", "commission.historical_entitlement", "commission.historical_posting_log",
	"commission.historical_run", "commission.historical_run_item", "commission.parity_summary", "commission.policy_version",
	"commission.reconciliation_finding", "commission.run_source", "commission.shadow_run",
	"custody.assignment", "custody.custody_event", "custody.finding", "custody.import_run",
	"customers.contact_point", "customers.customer", "customers.customer_identifier", "customers.customer_resolution_history",
	"customers.evidence_subject", "customers.identity_evidence", "customers.import_run", "customers.merge_case", "customers.merge_case_evidence",
	"customers.merge_case_member", "customers.merge_case_state_history",
	"finance.finding", "finance.ledger_account", "finance.ledger_entry", "finance.ledger_line", "finance.legacy_balance", "finance.reconciliation",
	"finance.run_source", "finance.shadow_run",
	"fulfillment.assignment", "fulfillment.case_event", "fulfillment.fulfillment_case", "fulfillment.import_run", "fulfillment.run_source",
	"fulfillment.service_evidence",
	"identity.capability", "identity.import_run",
	"integration.import_batch", "integration.import_entity", "identity.role", "identity.role_capability", "identity.role_grant", "identity.user_account",
	"organization.employment", "organization.hierarchy_finding", "organization.import_run", "organization.level", "organization.org_assignment",
	"organization.org_assignment_history", "organization.org_unit", "organization.person", "organization.position",
	"organization.position_role_default",
	"payments.gateway_attempt", "payments.import_run", "payments.payment_attempt", "payments.receipt",
	"reporting.case_fact", "reporting.custody_fact", "reporting.invoice_fact", "reporting.invoice_unit_fact", "reporting.metric_contract",
	"reporting.payment_fact", "reporting.projection_run", "reporting.projection_source", "reporting.source_aggregate",
}

// r75Orphans are cross-module references by generated id; each must resolve.
var r75Orphans = []struct{ child, col, parent, pk string }{
	{"organization.employment", "user_id", "identity.user_account", "user_id"},
	{"identity.role_grant", "user_id", "identity.user_account", "user_id"},
	{"custody.assignment", "case_id", "cases.work_case", "case_id"},
	{"custody.assignment", "derived_user_id", "identity.user_account", "user_id"},
	{"custody.custody_event", "to_user_id", "identity.user_account", "user_id"},
	{"cases.attribution_snapshot", "subject_user_id", "identity.user_account", "user_id"},
	{"payments.payment_attempt", "invoice_id", "billing.invoice", "invoice_id"},
	{"payments.receipt", "invoice_id", "billing.invoice", "invoice_id"},
	{"fulfillment.fulfillment_case", "invoice_id", "billing.invoice", "invoice_id"},
	{"fulfillment.assignment", "user_id", "identity.user_account", "user_id"},
	{"finance.ledger_account", "holder_user_id", "identity.user_account", "user_id"},
	{"finance.ledger_account", "customer_id", "customers.customer", "customer_id"},
	{"commission.calculation", "recipient_user_id", "identity.user_account", "user_id"},
	{"commission.historical_entitlement", "invoice_id", "billing.invoice", "invoice_id"},
	{"reporting.case_fact", "case_id", "cases.work_case", "case_id"},
	{"reporting.invoice_fact", "invoice_id", "billing.invoice", "invoice_id"},
	{"reporting.payment_fact", "attempt_id", "payments.payment_attempt", "attempt_id"},
}

func TestR75IntegratedSeed(t *testing.T) {
	e := setupIntegrated(t)
	_, steps := e.seedIntegrated(t)
	if !reflect.DeepEqual(steps, r75Steps) {
		t.Fatalf("step outcomes:\n got %+v\nwant %+v", steps, r75Steps)
	}
	got := e.tables(t)
	var owned, staged []string
	for _, tbl := range got {
		if strings.HasPrefix(tbl, "integration.stg_") {
			staged = append(staged, tbl)
		} else {
			owned = append(owned, tbl)
		}
	}
	pinned := append([]string(nil), r75Written...)
	sort.Strings(pinned)
	if !reflect.DeepEqual(owned, pinned) {
		t.Fatalf("populated tables differ from the pinned set:\n extra %v\n missing %v", minus(owned, r75Written), minus(r75Written, owned))
	}
	// Staging holds exactly one raw table per non-empty package entity.
	entities := 0
	for _, ent := range legacytest.IntegratedScenario() {
		if len(ent.Rows) > 0 {
			entities++
		}
	}
	if len(staged) != entities {
		t.Fatalf("%d staging tables populated, want %d (one per non-empty entity)", len(staged), entities)
	}
	resolved := 0
	for _, o := range r75Orphans {
		n := e.count(t, `SELECT count(*) FROM `+o.child+` c WHERE c.`+o.col+` IS NOT NULL AND NOT EXISTS (SELECT 1 FROM `+o.parent+` p WHERE p.`+o.pk+` = c.`+o.col+`)`)
		if n != 0 {
			t.Errorf("%s.%s: %d references do not resolve to %s", o.child, o.col, n, o.parent)
		}
		resolved += int(e.count(t, `SELECT count(*) FROM `+o.child+` WHERE `+o.col+` IS NOT NULL`))
	}
	if resolved == 0 {
		t.Fatal("no cross-module reference was checked")
	}
	// Safety: imported grants are SHADOW evidence only (also a schema CHECK),
	// never ACTIVE; no credential or session exists; no flag is enabled.
	if n := e.count(t, `SELECT count(*) FROM identity.role_grant WHERE status <> 'SHADOW'`); n != 0 {
		t.Fatalf("%d non-SHADOW grants", n)
	}
	if n := e.count(t, `SELECT (SELECT count(*) FROM identity.credential) + (SELECT count(*) FROM identity.session)`); n != 0 {
		t.Fatalf("%d credentials / sessions", n)
	}
	if n := e.count(t, `SELECT count(*) FROM platform_featureflags.feature_flag WHERE enabled`); n != 0 {
		t.Fatalf("%d enabled feature flags", n)
	}
	t.Logf("seeded %d tables; %d cross-module references resolved", len(got), resolved)
}

// Two fresh disposable databases receive the same logical records and
// references (runtime UUIDs and UUID-derived hashes excluded).
func TestR75IntegratedDeterministicAcrossDatabases(t *testing.T) {
	a, b := setupIntegrated(t), setupIntegrated(t)
	a.seedIntegrated(t)
	b.seedIntegrated(t)
	fa, fb := a.logical(t), b.logical(t)
	if len(fa) < len(r75Written) || len(fa) != len(fb) {
		t.Fatalf("fingerprinted %d and %d tables (pinned owner tables: %d)", len(fa), len(fb), len(r75Written))
	}
	for tbl, h := range fa {
		if fb[tbl] != h {
			t.Errorf("%s: logical content differs between two seeds", tbl)
		}
	}
	t.Logf("%d tables logically identical across two fresh databases", len(fa))
}

// r75ReportingRunTables hold one row set per projection run: a rebuild is a
// new projection VERSION by design (reads use the latest run), not a
// duplicate. A repeat seed adds one run whose content equals the first.
var r75ReportingRunTables = []string{"reporting.projection_source", "reporting.source_aggregate", "reporting.case_fact", "reporting.custody_fact",
	"reporting.invoice_fact", "reporting.invoice_unit_fact", "reporting.payment_fact"}

// perRun fingerprints a run-scoped table per run (run id excluded).
func (e *integratedEnv) perRun(t *testing.T, tbl string, labels map[string]string) []string {
	t.Helper()
	rows, err := e.admin.Query(context.Background(), `SELECT run_id::text, (to_jsonb(x) - 'run_id' - $1::text[])::text FROM `+tbl+` x`, r75RuntimeColumns)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	byRun := map[string][]string{}
	for rows.Next() {
		var run, v string
		if err := rows.Scan(&run, &v); err != nil {
			t.Fatal(err)
		}
		byRun[run] = append(byRun[run], uuidRe.ReplaceAllStringFunc(v, func(u string) string {
			if l, ok := labels[u]; ok {
				return l
			}
			return "U"
		}))
	}
	var out []string
	for _, lines := range byRun {
		sort.Strings(lines)
		sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
		out = append(out, hex.EncodeToString(sum[:8]))
	}
	return out
}

// Repeat execution on the same database: re-staging the same package is
// idempotent, every import replays or re-records the same evidence, and no
// authoritative record is duplicated. Only run metadata grows: one more
// import_run row per module and one more reporting projection run with
// identical content.
func TestR75IntegratedRepeatRun(t *testing.T) {
	e := setupIntegrated(t)
	b, _ := e.seedIntegrated(t)
	before := e.logical(t)
	counts := map[string]int64{}
	for tbl := range before {
		counts[tbl] = e.count(t, `SELECT count(*) FROM `+tbl)
	}
	// Re-stage the same package: the synthetic loader must not stage it twice.
	p, err := snapshot.Open(legacytest.Write(t, legacytest.IntegratedSnapshotID, t0, legacytest.IntegratedScenario()), spec.Default())
	if err != nil {
		t.Fatal(err)
	}
	batches := e.count(t, `SELECT count(*) FROM integration.import_batch`)
	if rep, err := e.loader.Load(context.Background(), p); err != nil || rep.Outcome != result.Success {
		t.Fatalf("re-stage: %s %v", rep.Outcome, err)
	}
	if n := e.count(t, `SELECT count(*) FROM integration.import_batch`); n != batches {
		t.Fatalf("re-staging created a second batch (%d → %d)", batches, n)
	}
	e.importAll(t, b)
	e.importSales(t, b)
	in, _ := shadow.CustomersCommand(b)
	if _, err := e.custIn.ImportCustomers(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	steps := e.importModules(t, b)
	want := []seedStep{{"billing", result.Success, false}, {"payments", result.Partial, true}, {"biawin", result.Success, true},
		{"fulfillment", result.Success, true}, {"finance", result.Success, true}, {"commission", result.Partial, true},
		{"reporting", result.Success, false}}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("repeat steps:\n got %+v\nwant %+v", steps, want)
	}
	runTables := set(r75ReportingRunTables, []string{"reporting.projection_run"})
	after := e.logical(t)
	if len(after) != len(before) {
		t.Fatalf("repeat changed the populated table set (%d → %d)", len(before), len(after))
	}
	labels := e.labels(t)
	grown := 0
	for tbl, h := range before {
		n := e.count(t, `SELECT count(*) FROM `+tbl)
		switch {
		case strings.HasSuffix(tbl, ".import_run"):
			if n != counts[tbl] && n != counts[tbl]+1 { // a replay returns the stored run (biawin, fulfillment) or records one more run
				t.Errorf("%s: %d → %d rows, want at most one more run record", tbl, counts[tbl], n)
			}
			grown++
		case runTables[tbl]:
			if n != 2*counts[tbl] {
				t.Errorf("%s: %d → %d rows, want one more projection run of the same size", tbl, counts[tbl], n)
			}
			if tbl != "reporting.projection_run" {
				if runs := e.perRun(t, tbl, labels); len(runs) != 2 || runs[0] != runs[1] {
					t.Errorf("%s: the repeat projection run differs from the first", tbl)
				}
			}
			grown++
		case after[tbl] != h:
			t.Errorf("%s: authoritative content changed on repeat (%d → %d rows)", tbl, counts[tbl], n)
		}
	}
	t.Logf("repeat: %d run-metadata tables grew as designed; %d tables unchanged", grown, len(before)-grown)
}

// Reads over the seeded data keep their gates: feature disabled by default,
// capability required, and reading changes no financial or source table.
func TestR75IntegratedReadRefusals(t *testing.T) {
	e := setupIntegrated(t)
	e.seedIntegrated(t)
	guarded := schemaSnapshot(t, e.admin, []string{"billing", "payments", "finance", "commission", "customers", "cases", "custody", "identity"})
	h := e.rep.server(t, e.rep.auditor(t))
	if r := get(t, h, "/api/v1/reporting/metrics", "dev-user-001"); !strings.Contains(r.body, `"FEATURE_DISABLED"`) {
		t.Fatalf("flag off: %d %s", r.code, r.body)
	}
	e.rep.setFlag(t, true)
	if r := get(t, h, "/api/v1/reports/invoices", "dev-user-002"); r.code != 403 {
		t.Fatalf("subject without the capability: %d", r.code)
	}
	if r := get(t, h, "/api/v1/reports/invoices", ""); r.code != 403 { // the dev stub refuses an anonymous subject
		t.Fatalf("anonymous: %d", r.code)
	}
	if r := get(t, h, "/api/v1/reports/invoices", "dev-user-001"); r.code != 200 || r.json["outcome"] == nil {
		t.Fatalf("permitted read: %d %s", r.code, r.body)
	}
	for _, d := range diffFinancial(guarded, schemaSnapshot(t, e.admin, []string{"billing", "payments", "finance", "commission", "customers",
		"cases", "custody", "identity"})) {
		t.Errorf("a read changed data: %s", d)
	}
}

// The synthetic loader refuses the same rows under a non-synthetic origin.
func TestR75IntegratedNonSyntheticIsBlocked(t *testing.T) {
	e := setupIntegrated(t)
	h := legacytest.Header("syn-r7-5-unclassified", t0)
	h.DataOrigin = snapshot.OriginUnclassified
	dir := filepath.Join(t.TempDir(), "snapshot")
	if _, err := snapshot.Write(dir, spec.Default(), h, legacytest.IntegratedScenario()); err != nil {
		t.Fatal(err)
	}
	p, err := snapshot.Open(dir, spec.Default())
	if err != nil {
		t.Fatal(err)
	}
	rep, err := e.loader.Load(context.Background(), p)
	if err != nil || rep.Outcome != result.Blocked {
		t.Fatalf("non-synthetic integrated package: %s %v", rep.Outcome, err)
	}
	if n := e.count(t, `SELECT count(*) FROM integration.import_batch`); n != 0 {
		t.Fatalf("a blocked package staged %d batches", n)
	}
}

// Opt-in export of the package for local use with the existing
// crm-legacy-import tool (its loader stages SYNTHETIC packages only and
// needs dedicated owner DSNs). Never overwrites: the target must not exist.
//
//	CRM_R75_EXPORT_DIR=/tmp/r7-5 go test -run TestR75ExportIntegratedPackage ./internal/integration/legacy/shadow/
func TestR75ExportIntegratedPackage(t *testing.T) {
	dir := os.Getenv("CRM_R75_EXPORT_DIR")
	if dir == "" {
		t.Skip("CRM_R75_EXPORT_DIR not set")
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("refusing to write into an existing path %s", dir)
	}
	if _, err := snapshot.Write(dir, spec.Default(), legacytest.Header(legacytest.IntegratedSnapshotID, legacytest.IntegratedExtractedAt),
		legacytest.IntegratedScenario()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(must(os.ReadFile(filepath.Join(dir, "manifest.json"))), []byte(`"synthetic"`)) {
		t.Fatal("exported manifest is not synthetic")
	}
	t.Logf("wrote %s (%s)", dir, legacytest.IntegratedVersion)
}

func minus(a, b []string) []string {
	m := set(b)
	var out []string
	for _, v := range a {
		if !m[v] {
			out = append(out, v)
		}
	}
	return out
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
