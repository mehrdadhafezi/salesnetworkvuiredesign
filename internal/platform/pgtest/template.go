package pgtest

// Template databases (CI-PERF-01). Migrating every disposable database from
// scratch costs about 1.2 s per test; cloning a database that already holds
// the same migrations costs about 0.1 s.
//
// NewWith(t, sources...) returns exactly what New(t) followed by
// migrations.Up of each source, in order, would return: a NEW, empty,
// isolated database dropped at cleanup. Only how it is produced changes:
// CREATE DATABASE ... TEMPLATE a cached template built once by the real
// migration runner.
//
//   - A template is content-addressed: its name is a hash of every SQL file
//     (path and bytes) of the platform, synthetic and given sources, their
//     version tables, their order and templateFormat. Different migrations
//     never share a template, and a template is never modified after it is
//     published.
//   - Templates are built under a server-wide advisory lock, in a temporary
//     database that is renamed into place only after every migration
//     succeeded, then marked IS_TEMPLATE and ALLOW_CONNECTIONS false (no
//     session can write to it; clones are unaffected).
//   - Templates hold schema and migration bookkeeping only. Tests write to
//     their own clone; nothing a test does reaches the template or another
//     clone.
//   - Templates stay on the (disposable) test server and are reused by later
//     runs with identical migrations. They are named crm_tpl_<hash>.
//
// TestTemplateCloneEqualsDirectMigration proves a clone is catalog- and
// bookkeeping-identical to a directly migrated database, and
// TestTemplateClonesAreIsolated proves clones never see each other's rows.

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// templateFormat is part of every template key; bump it when the template
// procedure or the migration runner's behaviour changes without a SQL change.
const templateFormat = "pgtest-template-v1"

var (
	templateMu    sync.Mutex
	templateNames = map[string]bool{} // published templates seen by this process
)

// NewWith returns a disposable database that holds the platform and
// synthetic migrations followed by sources, in order. It is equivalent to
// New(t) then migrations.Up(sources[i]) for each i.
func NewWith(t testing.TB, sources ...migrations.Source) *Env {
	t.Helper()
	all := append([]migrations.Source{migrations.Platform(), SyntheticSource(t)}, sources...)
	key, err := templateKey(all)
	if err != nil {
		t.Fatalf("pgtest: template key: %v", err)
	}
	tpl := ensureTemplate(t, key, all)
	return &Env{DSN: cloneDatabase(t, tpl), pools: map[owner.Module]*db.Pool{}}
}

// templateKey hashes every file of every source, in order.
func templateKey(sources []migrations.Source) (string, error) {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n", templateFormat)
	for i, s := range sources {
		fmt.Fprintf(h, "source %d table %s\n", i, s.Table)
		var paths []string
		if err := fs.WalkDir(s.FS, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				paths = append(paths, p)
			}
			return nil
		}); err != nil {
			return "", err
		}
		sort.Strings(paths)
		for _, p := range paths {
			b, err := fs.ReadFile(s.FS, p)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(h, "file %s %d\n", p, len(b))
			h.Write(b)
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:24], nil
}

// ensureTemplate returns the name of the published template for key,
// building it (once per server, under an advisory lock) when absent.
func ensureTemplate(t testing.TB, key string, sources []migrations.Source) string {
	t.Helper()
	name := "crm_tpl_" + key
	templateMu.Lock()
	defer templateMu.Unlock()
	if templateNames[name] {
		return name
	}
	admin := AdminURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatalf("pgtest: connect admin: %v", err)
	}
	defer conn.Close(ctx)
	sum := sha256.Sum256([]byte("pgtest-template-lock:" + key))
	lock := int64(binary.BigEndian.Uint64(sum[:8]))
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lock); err != nil {
		t.Fatalf("pgtest: template lock: %v", err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lock) }()

	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1 AND datistemplate)`, name).Scan(&exists); err != nil {
		t.Fatalf("pgtest: template lookup: %v", err)
	}
	if !exists {
		build := "crm_tplbuild_" + randomHex(t, 6)
		if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{build}.Sanitize()); err != nil {
			t.Fatalf("pgtest: create template build database: %v", err)
		}
		dsn := databaseDSN(t, admin, build)
		for i, s := range sources {
			if _, err := migrations.Up(ctx, dsn, s); err != nil {
				_, _ = conn.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{build}.Sanitize()+" WITH (FORCE)")
				t.Fatalf("pgtest: template migration %d: %v", i, err)
			}
		}
		// Publish atomically: rename only a fully migrated database, then
		// freeze it (template, no connections).
		for _, stmt := range []string{
			"ALTER DATABASE " + pgx.Identifier{build}.Sanitize() + " RENAME TO " + pgx.Identifier{name}.Sanitize(),
			"ALTER DATABASE " + pgx.Identifier{name}.Sanitize() + " WITH IS_TEMPLATE true ALLOW_CONNECTIONS false",
		} {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				_, _ = conn.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{build}.Sanitize()+" WITH (FORCE)")
				t.Fatalf("pgtest: publish template: %v", err)
			}
		}
	}
	templateNames[name] = true
	return name
}

// cloneDatabase creates a disposable database from template tpl and drops
// it at cleanup (the same lifecycle as CreateDatabase).
func cloneDatabase(t testing.TB, tpl string) string {
	t.Helper()
	admin := AdminURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	name := "crm_test_" + randomHex(t, 6)
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatalf("pgtest: connect admin: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" TEMPLATE "+pgx.Identifier{tpl}.Sanitize()); err != nil {
		t.Fatalf("pgtest: clone template: %v", err)
	}
	dropAtCleanup(t, admin, name)
	return databaseDSN(t, admin, name)
}

// databaseDSN returns the admin URL pointed at database name.
func databaseDSN(t testing.TB, admin, name string) string {
	t.Helper()
	u, err := url.Parse(admin)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("pgtest: %s must be a postgres:// URL", EnvAdminURL)
	}
	u.Path = "/" + name
	return u.String()
}
