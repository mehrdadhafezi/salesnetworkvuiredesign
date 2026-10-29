package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return c
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestConfigValidation(t *testing.T) {
	ok := db.DefaultConfig("postgres://u@localhost/db")
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	mutate := []func(*db.Config){
		func(c *db.Config) { c.URL = " " },
		func(c *db.Config) { c.MaxConns = 0 },
		func(c *db.Config) { c.MaxConns = 101 },
		func(c *db.Config) { c.MinConns = -1 },
		func(c *db.Config) { c.MinConns = c.MaxConns + 1 },
		func(c *db.Config) { c.MaxConnLifetime = 0 },
		func(c *db.Config) { c.MaxConnIdleTime = -time.Second },
		func(c *db.Config) { c.HealthCheckPeriod = 0 },
		func(c *db.Config) { c.ConnectTimeout = 0 },
		func(c *db.Config) { c.ApplicationName = "" },
	}
	for i, m := range mutate {
		c := ok
		m(&c)
		if err := c.Validate(); !errors.Is(err, db.ErrInvalidConfig) {
			t.Fatalf("mutation %d: err = %v", i, err)
		}
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := func(m map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
	}
	if _, err := db.ConfigFromEnv(env(nil)); !errors.Is(err, db.ErrInvalidConfig) {
		t.Fatalf("missing env err = %v", err)
	}
	if _, err := db.ConfigFromEnv(env(map[string]string{db.EnvDatabaseURL: "  "})); !errors.Is(err, db.ErrInvalidConfig) {
		t.Fatalf("blank env err = %v", err)
	}
	cfg, err := db.ConfigFromEnv(env(map[string]string{db.EnvDatabaseURL: "postgres://u@h/d"}))
	if err != nil || cfg.URL != "postgres://u@h/d" || cfg.MaxConns != 10 {
		t.Fatalf("cfg = %+v, %v", cfg, err)
	}
}

func TestOpenRejectsBadConfig(t *testing.T) {
	if _, err := db.Open(context.Background(), db.Config{}); !errors.Is(err, db.ErrInvalidConfig) {
		t.Fatalf("empty config err = %v", err)
	}
	if _, err := db.Open(context.Background(), db.DefaultConfig("::not a url::")); !errors.Is(err, db.ErrInvalidConfig) {
		t.Fatalf("bad url err = %v", err)
	}
	cfg := db.DefaultConfig("postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	cfg.ConnectTimeout = time.Second
	if _, err := db.Open(context.Background(), cfg); err == nil {
		t.Fatal("unreachable server must fail Open")
	}
}

func TestClosedPool(t *testing.T) {
	var p *db.Pool
	if err := p.Ping(context.Background()); !errors.Is(err, db.ErrClosed) {
		t.Fatalf("nil pool ping err = %v", err)
	}
	p.Close() // nil-safe
	if _, err := p.Runner(pgtest.Alpha); !errors.Is(err, db.ErrClosed) {
		t.Fatalf("nil pool runner err = %v", err)
	}
}

func TestPoolLifecycle(t *testing.T) {
	env := pgtest.New(t)
	pool := env.Pool(t, pgtest.Alpha)
	if err := pool.Ping(ctx(t)); err != nil {
		t.Fatal(err)
	}
	r := env.Runner(t, pgtest.Alpha)
	pool.Close()
	pool.Close() // idempotent
	if err := pool.Ping(ctx(t)); !errors.Is(err, db.ErrClosed) {
		t.Fatalf("ping after close err = %v", err)
	}
	if err := r.InTx(ctx(t), db.TxOptions{}, func(context.Context, db.Tx) error { return nil }); !errors.Is(err, db.ErrClosed) {
		t.Fatalf("InTx after close err = %v", err)
	}
	if _, err := pool.Runner(pgtest.Alpha); !errors.Is(err, db.ErrClosed) {
		t.Fatalf("Runner after close err = %v", err)
	}
}

func widgets(t *testing.T, env *pgtest.Env) int {
	t.Helper()
	var n int
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return tx.QueryRow(c, "SELECT count(*) FROM syn_alpha.widget").Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func insertWidget(c context.Context, tx db.Tx) error {
	_, err := tx.Exec(c, "INSERT INTO widget (label) VALUES ('synthetic')") // unqualified: search_path is the owner schema
	return err
}

func TestCommitRollbackAndPanic(t *testing.T) {
	env := pgtest.New(t)
	r := env.Runner(t, pgtest.Alpha)
	if r.Module() != pgtest.Alpha {
		t.Fatal("runner module")
	}
	if err := r.InTx(ctx(t), db.TxOptions{}, insertWidget); err != nil {
		t.Fatal(err)
	}
	if widgets(t, env) != 1 {
		t.Fatal("commit not visible")
	}

	boom := errors.New("handler failed")
	err := r.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := insertWidget(c, tx); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("handler error not returned: %v", err)
	}
	if widgets(t, env) != 1 {
		t.Fatal("handler error must roll back")
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("panic must propagate")
			}
		}()
		_ = r.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
			if err := insertWidget(c, tx); err != nil {
				return err
			}
			panic("synthetic panic")
		})
	}()
	if widgets(t, env) != 1 {
		t.Fatal("panic must roll back")
	}
	// The pool is still healthy after a panic.
	if err := r.InTx(ctx(t), db.TxOptions{Isolation: db.Serializable}, insertWidget); err != nil {
		t.Fatal(err)
	}
	if err := r.InTx(ctx(t), db.TxOptions{Isolation: db.RepeatableRead}, insertWidget); err != nil {
		t.Fatal(err)
	}
	if widgets(t, env) != 3 {
		t.Fatal("isolation-level transactions not committed")
	}
	if err := r.InTx(ctx(t), db.TxOptions{Isolation: 9}, insertWidget); !errors.Is(err, db.ErrInvalidTxOptions) {
		t.Fatalf("invalid isolation err = %v", err)
	}
}

func TestTransactionRunsAsOwnerRole(t *testing.T) {
	env := pgtest.New(t)
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		var user, path string
		if err := tx.QueryRow(c, "SELECT current_user, current_setting('search_path')").Scan(&user, &path); err != nil {
			return err
		}
		if user != pgtest.Alpha.Role() || path != `syn_alpha` || tx.Module() != pgtest.Alpha || tx.ReadOnly() {
			t.Errorf("tx runs as %q search_path %q", user, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCrossSchemaWritesAreRejected(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)
	targets := map[string]string{
		"other module table":  "INSERT INTO syn_beta.widget_seen (event_id, label) VALUES ('00000000-0000-4000-8000-000000000001', 'x')",
		"other module outbox": "INSERT INTO syn_beta.outbox_event (event_id, event_type, event_version, payload, occurred_at, recorded_at) VALUES ('00000000-0000-4000-8000-000000000002', 'syn_beta.x', 1, '{}', now(), now())",
		"audit":               "INSERT INTO audit.audit_event (event_id, occurred_at, subject_domain, subject_id, action) VALUES ('00000000-0000-4000-8000-000000000003', now(), 'EVENT', '1', 'syn.x')",
		"feature flags":       "INSERT INTO platform_featureflags.feature_flag (flag_key, enabled, updated_at) VALUES ('syn.flag', true, now())",
		"migration history":   "DELETE FROM public.crm_schema_migrations",
	}
	for name, stmt := range targets {
		err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
			_, err := tx.Exec(c, stmt)
			return err
		})
		if pgCode(err) != "42501" { // insufficient_privilege
			t.Fatalf("%s: err = %v (code %q), want permission denied", name, err, pgCode(err))
		}
	}
	// DDL is not available to an owner role at runtime either.
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		_, err := tx.Exec(c, "CREATE TABLE syn_alpha.runtime_ddl (id int)")
		return err
	})
	if pgCode(err) != "42501" {
		t.Fatalf("runtime DDL err = %v, want permission denied", err)
	}
}

func TestReadOnlyTransactionCannotWrite(t *testing.T) {
	env := pgtest.New(t)
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		if !tx.ReadOnly() {
			t.Error("ReadOnly() must report true")
		}
		return insertWidget(c, tx)
	})
	if pgCode(err) != "25006" { // read_only_sql_transaction
		t.Fatalf("write in read-only tx err = %v", err)
	}
}

func TestRoleEscapeAbortsCommit(t *testing.T) {
	env := pgtest.New(t)
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := insertWidget(c, tx); err != nil {
			return err
		}
		_, err := tx.Exec(c, "RESET ROLE")
		return err
	})
	if !errors.Is(err, db.ErrRoleEscaped) {
		t.Fatalf("role escape err = %v", err)
	}
	if widgets(t, env) != 0 {
		t.Fatal("a transaction whose role escaped must not commit")
	}
}

func TestUnknownOwnerRoleFails(t *testing.T) {
	env := pgtest.New(t)
	r, err := env.Pool(t, pgtest.Alpha).Runner(owner.Module("syn_missing"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.InTx(ctx(t), db.TxOptions{}, func(context.Context, db.Tx) error { return nil }); err == nil {
		t.Fatal("a module without a migrated owner role must not get a transaction")
	}
	if _, err := env.Pool(t, pgtest.Alpha).Runner(owner.Module("Bad")); !errors.Is(err, owner.ErrInvalidModule) {
		t.Fatalf("invalid module err = %v", err)
	}
}

func TestFailedStatementRollsBack(t *testing.T) {
	env := pgtest.New(t)
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := insertWidget(c, tx); err != nil {
			return err
		}
		_, _ = tx.Exec(c, "INSERT INTO widget (label) VALUES ('')") // violates CHECK; error deliberately ignored
		return nil
	})
	if err == nil {
		t.Fatal("an aborted transaction must not report success")
	}
	if widgets(t, env) != 0 {
		t.Fatal("aborted transaction must not commit")
	}
}
