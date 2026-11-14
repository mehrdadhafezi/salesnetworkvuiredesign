package shadow_test

// CI-PERF-01 isolation proof for pgtest.NewWith (template databases): a clone
// is identical to a directly migrated database, clones never see each
// other's rows, the published template accepts no session, and concurrent
// clones are independent. These run with every module the shadow suites use.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	biawinmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/migrations"
	billingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/migrations"
	casesmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/migrations"
	commissionmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/migrations"
	custodymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/migrations"
	customersmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/migrations"
	financemigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/migrations"
	fulfillmentmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/migrations"
	identitymigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/migrations"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	organizationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/migrations"
	paymentsmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
	reportingmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/reporting/migrations"
)

func allShadowSources() []migrations.Source {
	return []migrations.Source{integrationmigrations.Source(), identitymigrations.Source(), organizationmigrations.Source(),
		casesmigrations.Source(), custodymigrations.Source(), customersmigrations.Source(), billingmigrations.Source(),
		paymentsmigrations.Source(), biawinmigrations.Source(), fulfillmentmigrations.Source(), financemigrations.Source(),
		commissionmigrations.Source(), reportingmigrations.Source()}
}

// schemaCatalog returns a sorted, OID-free description of everything a migration
// can create: schemas, relations (kind, owner, ACL, RLS), columns, constraints,
// indexes, functions, triggers, policies, extensions, default privileges, and
// the content of every non-empty table (migration bookkeeping and seeded
// reference rows) with goose's applied-at timestamp excluded.
func schemaCatalog(t *testing.T, dsn string) []string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	const user = `NOT IN ('pg_catalog', 'information_schema') AND n.nspname NOT LIKE 'pg_toast%' AND n.nspname NOT LIKE 'pg_temp%'`
	q := `
SELECT 'nsp|' || n.nspname || '|' || pg_get_userbyid(n.nspowner) || '|' || coalesce(n.nspacl::text, '') FROM pg_namespace n WHERE n.nspname ` + user + `
UNION ALL
SELECT 'rel|' || n.nspname || '.' || c.relname || '|' || c.relkind::text || '|' || pg_get_userbyid(c.relowner) || '|' || coalesce(c.relacl::text, '') || '|' || c.relrowsecurity || c.relforcerowsecurity
  FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname ` + user + `
UNION ALL
SELECT 'col|' || table_schema || '.' || table_name || '|' || ordinal_position || '|' || column_name || '|' || data_type || '|' || is_nullable || '|' || coalesce(column_default, '')
  FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
UNION ALL
SELECT 'con|' || n.nspname || '|' || coalesce(c.conrelid::regclass::text, '') || '|' || c.conname || '|' || pg_get_constraintdef(c.oid)
  FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace WHERE n.nspname ` + user + `
UNION ALL
SELECT 'idx|' || schemaname || '|' || indexname || '|' || indexdef FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
UNION ALL
SELECT 'fn|' || p.oid::regprocedure::text || '|' || pg_get_userbyid(p.proowner) || '|' || coalesce(p.proacl::text, '') || '|' || md5(pg_get_functiondef(p.oid))
  FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace WHERE n.nspname ` + user + ` AND p.prokind IN ('f', 'p')
UNION ALL
SELECT 'trg|' || tgrelid::regclass::text || '|' || tgname || '|' || pg_get_triggerdef(oid) FROM pg_trigger WHERE NOT tgisinternal
UNION ALL
SELECT 'pol|' || schemaname || '.' || tablename || '|' || policyname || '|' || array_to_string(roles, ',') || '|' || cmd || '|' || coalesce(qual, '') || '|' || coalesce(with_check, '') FROM pg_policies
UNION ALL
SELECT 'ext|' || extname || '|' || extversion FROM pg_extension
UNION ALL
SELECT 'dacl|' || pg_get_userbyid(defaclrole) || '|' || coalesce(defaclnamespace::regnamespace::text, '') || '|' || defaclobjtype::text || '|' || defaclacl::text FROM pg_default_acl`
	rows, err := conn.Query(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	rows.Close()
	// Row content of every table (version tables and seeded reference data).
	trows, err := conn.Query(ctx, `SELECT quote_ident(table_schema) || '.' || quote_ident(table_name) FROM information_schema.tables
		WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for trows.Next() {
		var n string
		if err := trows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	trows.Close()
	for _, tbl := range tables {
		var sum *string
		if err := conn.QueryRow(ctx, `SELECT md5(string_agg(r, '|' ORDER BY r)) FROM (SELECT (to_jsonb(x) - 'tstamp')::text r FROM `+tbl+` x) s`).
			Scan(&sum); err != nil {
			t.Fatal(err)
		}
		if sum != nil {
			out = append(out, "data|"+tbl+"|"+*sum)
		}
	}
	sort.Strings(out)
	return out
}

func TestPgtestTemplateCloneEqualsDirectMigration(t *testing.T) {
	direct := pgtest.New(t)
	for _, s := range allShadowSources() {
		if _, err := migrations.Up(context.Background(), direct.DSN, s); err != nil {
			t.Fatal(err)
		}
	}
	clone := pgtest.NewWith(t, allShadowSources()...)
	a, b := schemaCatalog(t, direct.DSN), schemaCatalog(t, clone.DSN)
	if len(a) < 1000 {
		t.Fatalf("catalog fingerprint too small (%d lines): the comparison would prove nothing", len(a))
	}
	// Sensitivity: a clone without the last module must NOT compare equal.
	all := allShadowSources()
	if fewer := schemaCatalog(t, pgtest.NewWith(t, all[:len(all)-1]...).DSN); strings.Join(fewer, "\n") == strings.Join(a, "\n") {
		t.Fatal("the catalog comparison cannot tell a missing module apart")
	}
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		da, db := map[string]bool{}, map[string]bool{}
		for _, l := range a {
			da[l] = true
		}
		for _, l := range b {
			db[l] = true
		}
		n := 0
		for _, l := range a {
			if !db[l] && n < 10 {
				t.Errorf("only in direct: %s", l)
				n++
			}
		}
		for _, l := range b {
			if !da[l] && n < 20 {
				t.Errorf("only in clone: %s", l)
				n++
			}
		}
		t.Fatal("a template clone differs from a directly migrated database")
	}
	// Re-running every migration on the clone is a no-op.
	for _, s := range allShadowSources() {
		if applied, err := migrations.Up(context.Background(), clone.DSN, s); err != nil || len(applied) != 0 {
			t.Fatalf("migrations on a clone: %d applied, %v", len(applied), err)
		}
	}
	t.Logf("%d catalog / data lines identical", len(a))
}

func TestPgtestTemplateClonesAreIsolated(t *testing.T) {
	ctx := context.Background()
	src := allShadowSources()[:3]
	one, two := pgtest.NewWith(t, src...), pgtest.NewWith(t, src...)
	if one.DSN == two.DSN {
		t.Fatal("two clones share a database")
	}
	c1, err := pgx.Connect(ctx, one.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close(ctx)
	if _, err := c1.Exec(ctx, `INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, updated_at) VALUES ('pgtest.isolation_probe', true, now())`); err != nil {
		t.Fatal(err)
	}
	for _, e := range []*pgtest.Env{two, pgtest.NewWith(t, src...)} { // a sibling, and a clone made after the write
		c, err := pgx.Connect(ctx, e.DSN)
		if err != nil {
			t.Fatal(err)
		}
		var n int
		if err := c.QueryRow(ctx, `SELECT count(*) FROM platform_featureflags.feature_flag WHERE flag_key = 'pgtest.isolation_probe'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		c.Close(ctx)
		if n != 0 {
			t.Fatal("a write in one clone is visible in another clone or in the template")
		}
	}
	// The published template is frozen: a template, and no session may connect.
	admin, err := pgx.Connect(ctx, pgtest.AdminURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	var templates, writable int
	if err := admin.QueryRow(ctx, `SELECT count(*) FILTER (WHERE datistemplate), count(*) FILTER (WHERE datallowconn)
		FROM pg_database WHERE datname LIKE 'crm_tpl\_%' AND datname NOT LIKE 'crm_tplbuild\_%'`).Scan(&templates, &writable); err != nil {
		t.Fatal(err)
	}
	if templates == 0 || writable != 0 {
		t.Fatalf("templates %d, accepting connections %d", templates, writable)
	}
}

func TestPgtestTemplateConcurrentClones(t *testing.T) {
	src := allShadowSources()
	dsns := make([]string, 8)
	t.Run("clones", func(t *testing.T) {
		for i := range dsns {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				t.Parallel()
				e := pgtest.NewWith(t, src...)
				var n int
				c, err := pgx.Connect(context.Background(), e.DSN)
				if err != nil {
					t.Fatal(err)
				}
				defer c.Close(context.Background())
				if err := c.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'reporting'`).Scan(&n); err != nil || n == 0 {
					t.Fatalf("clone %d incomplete: %d %v", i, n, err)
				}
				dsns[i] = e.DSN
			})
		}
	})
	seen := map[string]bool{}
	for _, d := range dsns {
		if seen[d] {
			t.Fatal("two concurrent clones share a database")
		}
		seen[d] = true
	}
}
