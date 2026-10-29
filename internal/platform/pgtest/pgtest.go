// Package pgtest is TEST-ONLY PostgreSQL infrastructure. Architecture lint
// allows importing it from _test.go files only.
//
// New creates a disposable database on the server named by
// CRM_TEST_DATABASE_URL, applies the platform migrations through the real
// migration runner, then applies the synthetic test modules syn_alpha and
// syn_beta (rendered with migrations.RenderModuleInfra) from a separate
// version table. Transactions under test never run as the superuser: each
// Runner uses its own pool that logs in as a freshly provisioned restricted
// role (LoginDSN), member of a single owner role — the intended application
// connection model. The database and roles are dropped when the test ends. Without
// CRM_TEST_DATABASE_URL the test is skipped, unless
// CRM_TEST_REQUIRE_POSTGRES=1 (CI), which makes it fail instead.
package pgtest

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"net/url"
	"os"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Environment variables.
const (
	EnvAdminURL = "CRM_TEST_DATABASE_URL"
	EnvRequire  = "CRM_TEST_REQUIRE_POSTGRES"
)

// Synthetic test-only modules. They exist only in disposable test databases.
const (
	Alpha owner.Module = "syn_alpha"
	Beta  owner.Module = "syn_beta"
)

// SyntheticTable is the version table of the synthetic test migrations.
const SyntheticTable = "pgtest_synthetic_migrations"

//go:embed synthetic/*.sql
var syntheticFS embed.FS

// Env is one disposable, fully migrated test database.
type Env struct {
	DSN string // superuser provisioning/migrator DSN of the disposable database

	mu    sync.Mutex
	pools map[owner.Module]*db.Pool
}

// AdminURL returns the configured server URL, skipping or failing the test
// when it is absent.
func AdminURL(t testing.TB) string {
	t.Helper()
	u := os.Getenv(EnvAdminURL)
	if u == "" {
		if os.Getenv(EnvRequire) == "1" {
			t.Fatalf("%s is required (%s=1)", EnvAdminURL, EnvRequire)
		}
		t.Skipf("%s not set; skipping PostgreSQL integration test", EnvAdminURL)
	}
	return u
}

// SyntheticSource returns the rendered synthetic module migrations.
func SyntheticSource(t testing.TB) migrations.Source {
	t.Helper()
	files := fstest.MapFS{}
	for i, m := range []owner.Module{Alpha, Beta} {
		body, err := syntheticFS.ReadFile("synthetic/" + string(m) + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		sql, err := migrations.RenderModuleInfra(m, string(body))
		if err != nil {
			t.Fatal(err)
		}
		files[[]string{"00001_", "00002_"}[i]+string(m)+".sql"] = &fstest.MapFile{Data: []byte(sql)}
	}
	return migrations.Source{FS: files, Table: SyntheticTable}
}

// CreateDatabase creates an empty disposable database and returns its DSN.
// It is dropped at test cleanup.
func CreateDatabase(t testing.TB) string {
	t.Helper()
	admin := AdminURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	name := "crm_test_" + hex.EncodeToString(b[:])
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatalf("pgtest: connect admin: %v", err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("pgtest: create database: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c, err := pgx.Connect(ctx, admin)
		if err != nil {
			t.Logf("pgtest: cleanup connect: %v", err)
			return
		}
		defer c.Close(ctx)
		if _, err := c.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Logf("pgtest: drop database %s: %v", name, err)
		}
	})
	u, err := url.Parse(admin)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("pgtest: %s must be a postgres:// URL", EnvAdminURL)
	}
	u.Path = "/" + name
	return u.String()
}

// New returns a migrated disposable database. The provisioning connection
// (DSN) is a superuser; transactions under test never use it, they run
// through per-owner pools that log in as restricted roles (see Pool).
func New(t testing.TB) *Env {
	t.Helper()
	dsn := CreateDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, err := migrations.Up(ctx, dsn, migrations.Platform()); err != nil {
		t.Fatalf("pgtest: platform migrations: %v", err)
	}
	if _, err := migrations.Up(ctx, dsn, SyntheticSource(t)); err != nil {
		t.Fatalf("pgtest: synthetic migrations: %v", err)
	}
	return &Env{DSN: dsn, pools: map[owner.Module]*db.Pool{}}
}

// LoginDSN provisions a fresh LOGIN role that represents the intended
// application connection model: NOINHERIT, not superuser, no CREATEROLE /
// BYPASSRLS, and a member of exactly the given owner roles. It returns a DSN
// that logs in as that role. The role is dropped at cleanup. Passing more
// than one module builds a deliberately mis-provisioned role for negative tests.
func (e *Env) LoginDSN(t testing.TB, members ...owner.Module) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var nb, pb [8]byte
	if _, err := rand.Read(nb[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(pb[:]); err != nil {
		t.Fatal(err)
	}
	name, password := "crmt_login_"+hex.EncodeToString(nb[:]), hex.EncodeToString(pb[:])
	conn, err := pgx.Connect(ctx, e.DSN)
	if err != nil {
		t.Fatalf("pgtest: connect provisioning: %v", err)
	}
	defer conn.Close(ctx)
	role := pgx.Identifier{name}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE ROLE "+role+" LOGIN NOINHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOBYPASSRLS PASSWORD '"+password+"'"); err != nil {
		t.Fatalf("pgtest: create login role: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c, err := pgx.Connect(ctx, e.DSN)
		if err != nil {
			t.Logf("pgtest: cleanup connect: %v", err)
			return
		}
		defer c.Close(ctx)
		if _, err := c.Exec(ctx, "DROP ROLE IF EXISTS "+role); err != nil {
			t.Logf("pgtest: drop role %s: %v", name, err)
		}
	})
	for _, m := range members {
		if err := m.Validate(); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(ctx, "GRANT "+pgx.Identifier{m.Role()}.Sanitize()+" TO "+role); err != nil {
			t.Fatalf("pgtest: grant %s: %v", m.Role(), err)
		}
	}
	u, err := url.Parse(e.DSN)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(name, password)
	return u.String()
}

// OpenPool opens a pool on dsn, closed at cleanup.
func OpenPool(t testing.TB, dsn string) *db.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg := db.DefaultConfig(dsn)
	cfg.ApplicationName = "crm-pgtest"
	pool, err := db.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("pgtest: open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Pool returns the dedicated pool for module m: it logs in as a restricted
// role that can assume only m's owner role. Pools are cached per Env.
func (e *Env) Pool(t testing.TB, m owner.Module) *db.Pool {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if p, ok := e.pools[m]; ok {
		return p
	}
	p := OpenPool(t, e.LoginDSN(t, m))
	e.pools[m] = p
	return p
}

// Runner returns a runner for module m over its dedicated pool.
func (e *Env) Runner(t testing.TB, m owner.Module) *db.Runner {
	t.Helper()
	r, err := e.Pool(t, m).Runner(m)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
