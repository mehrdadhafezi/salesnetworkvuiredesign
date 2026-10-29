package migrations_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestPlatformSourceIsValid(t *testing.T) {
	names, err := migrations.Validate(migrations.Platform())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00001_platform_audit.sql", "00002_platform_featureflags.sql"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("platform migrations = %v, want %v", names, want)
	}
}

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

func TestValidateRejects(t *testing.T) {
	up := "-- +goose Up\nSELECT 1;\n"
	cases := map[string]migrations.Source{
		"nil fs":         {Table: "t_versions"},
		"bad table":      {FS: fstest.MapFS{"00001_a.sql": file(up)}, Table: "Bad-Table"},
		"empty":          {FS: fstest.MapFS{}, Table: "t_versions"},
		"bad name":       {FS: fstest.MapFS{"1_a.sql": file(up)}, Table: "t_versions"},
		"uppercase name": {FS: fstest.MapFS{"00001_A.sql": file(up)}, Table: "t_versions"},
		"dup version":    {FS: fstest.MapFS{"00001_a.sql": file(up), "00001_b.sql": file(up)}, Table: "t_versions"},
		"no up":          {FS: fstest.MapFS{"00001_a.sql": file("SELECT 1;")}, Table: "t_versions"},
		"no tx":          {FS: fstest.MapFS{"00001_a.sql": file("-- +goose NO TRANSACTION\n" + up)}, Table: "t_versions"},
		"directory":      {FS: fstest.MapFS{"00001_a.sql": file(up), "sub/00002_b.sql": file(up)}, Table: "t_versions"},
	}
	for name, src := range cases {
		if _, err := migrations.Validate(src); !errors.Is(err, migrations.ErrInvalidMigration) && !errors.Is(err, migrations.ErrInvalidSource) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
	if _, err := migrations.Up(context.Background(), "", migrations.Platform()); !errors.Is(err, migrations.ErrInvalidSource) {
		t.Fatalf("empty DSN err = %v", err)
	}
	if _, err := migrations.Status(context.Background(), " ", migrations.Platform()); !errors.Is(err, migrations.ErrInvalidSource) {
		t.Fatalf("blank DSN err = %v", err)
	}
}

func TestRenderModuleInfra(t *testing.T) {
	sql, err := migrations.RenderModuleInfra("syn_gamma", "CREATE TABLE syn_gamma.thing (id int);")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"-- +goose Up", "CREATE ROLE crm_owner_syn_gamma NOLOGIN", "CREATE SCHEMA syn_gamma;",
		"syn_gamma.outbox_event", "syn_gamma.inbox_message", "syn_gamma.idempotency_record",
		"PRIMARY KEY (scope, idem_key)", "TO crm_owner_syn_gamma;", "CREATE TABLE syn_gamma.thing",
		"-- +goose Down", "DROP SCHEMA syn_gamma CASCADE;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("rendered migration lacks %q", want)
		}
	}
	// Infrastructure tables are per-module: nothing targets another schema.
	if strings.Contains(sql, "public.") || strings.Contains(sql, "audit.") {
		t.Fatal("module infrastructure must stay inside the module schema")
	}
	if strings.Contains(sql, "{{") || strings.Contains(sql, "<no value>") {
		t.Fatal("template not fully rendered")
	}
	if _, err := migrations.RenderModuleInfra("Bad", ""); !errors.Is(err, owner.ErrInvalidModule) {
		t.Fatalf("invalid module err = %v", err)
	}
	if _, err := migrations.RenderModuleInfra("syn_gamma", "-- +goose Down\nDROP SCHEMA x;"); !errors.Is(err, migrations.ErrInvalidMigration) {
		t.Fatalf("goose directive in module SQL err = %v", err)
	}
}

func objects(t *testing.T, dsn string) (schemas, tables []string) {
	t.Helper()
	c, err := pgx.Connect(ctx(t), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx(t))
	rows, err := c.Query(ctx(t), `SELECT nspname FROM pg_namespace
		WHERE nspname NOT LIKE 'pg\_%' AND nspname NOT IN ('information_schema', 'public') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	schemas, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	rows, err = c.Query(ctx(t), `SELECT table_schema || '.' || table_name FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	tables, err = pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return schemas, tables
}

func TestUpOnEmptyDatabaseIsRepeatable(t *testing.T) {
	dsn := pgtest.CreateDatabase(t)
	applied, err := migrations.Up(ctx(t), dsn, migrations.Platform())
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 2 || applied[0].Version != 1 || applied[1].Version != 2 {
		t.Fatalf("first Up applied %+v", applied)
	}
	again, err := migrations.Up(ctx(t), dsn, migrations.Platform())
	if err != nil || len(again) != 0 {
		t.Fatalf("second Up = %+v, %v; want no-op", again, err)
	}
	states, err := migrations.Status(ctx(t), dsn, migrations.Platform())
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range states {
		if !s.Applied {
			t.Fatalf("%05d %s not applied", s.Version, s.Name)
		}
	}
	// Only platform infrastructure exists: no business-domain tables.
	schemas, tables := objects(t, dsn)
	if strings.Join(schemas, ",") != "audit,platform_featureflags" {
		t.Fatalf("schemas = %v", schemas)
	}
	want := []string{"audit.audit_event", "platform_featureflags.feature_flag", "public." + migrations.PlatformTable}
	sort.Strings(want)
	if strings.Join(tables, ",") != strings.Join(want, ",") {
		t.Fatalf("tables = %v, want %v", tables, want)
	}
}

func TestFailedMigrationIsNotRecorded(t *testing.T) {
	dsn := pgtest.CreateDatabase(t)
	src := migrations.Source{Table: "fixture_migrations", FS: fstest.MapFS{
		"00001_ok.sql":   file("-- +goose Up\nCREATE TABLE public.fixture_ok (id int);\n-- +goose Down\nDROP TABLE public.fixture_ok;\n"),
		"00002_bad.sql":  file("-- +goose Up\nCREATE TABLE public.fixture_partial (id int);\nSELECT 1/0;\n"),
		"00003_next.sql": file("-- +goose Up\nCREATE TABLE public.fixture_next (id int);\n"),
	}}
	applied, err := migrations.Up(ctx(t), dsn, src)
	if err == nil {
		t.Fatal("failing migration must return an error")
	}
	if len(applied) != 1 || applied[0].Version != 1 {
		t.Fatalf("applied before failure = %+v, want only 00001", applied)
	}
	states, err := migrations.Status(ctx(t), dsn, src)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]bool{}
	for _, s := range states {
		got[s.Version] = s.Applied
	}
	if !got[1] || got[2] || got[3] {
		t.Fatalf("status after failure = %v; 00002 must not count as applied and 00003 must not run", got)
	}
	_, tables := objects(t, dsn)
	for _, tbl := range tables {
		if tbl == "public.fixture_partial" || tbl == "public.fixture_next" {
			t.Fatalf("partial migration left %s behind", tbl)
		}
	}
	if _, err := migrations.Up(ctx(t), dsn, src); err == nil {
		t.Fatal("re-running a failing migration must fail again, never silently succeed")
	}
}

func TestConcurrentUpIsSerialized(t *testing.T) {
	dsn := pgtest.CreateDatabase(t)
	var wg sync.WaitGroup
	results := make([][]migrations.Applied, 4)
	errs := make([]error, 4)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = migrations.Up(context.Background(), dsn, migrations.Platform())
		}(i)
	}
	wg.Wait()
	total := 0
	for i := range results {
		if errs[i] != nil {
			t.Fatalf("runner %d: %v", i, errs[i])
		}
		total += len(results[i])
	}
	if total != 2 {
		t.Fatalf("migrations applied %d times across concurrent runners, want exactly 2", total)
	}
}
