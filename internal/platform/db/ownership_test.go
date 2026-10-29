package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// These tests run every transaction under test through a restricted LOGIN
// role (pgtest.LoginDSN / env.Runner): NOINHERIT, not superuser, member of a
// single owner role. That is the intended application connection model; the
// superuser connection is used only to provision and to inspect results.

const (
	betaRole  = "crm_owner_syn_beta"
	alphaRole = "crm_owner_syn_alpha"
	betaWrite = "INSERT INTO syn_beta.widget_seen (event_id, label) VALUES ('00000000-0000-4000-8000-0000000000be', 'stolen')"
)

func betaRows(t *testing.T, env *pgtest.Env) int {
	t.Helper()
	var n int
	err := env.Runner(t, pgtest.Beta).InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return tx.QueryRow(c, "SELECT count(*) FROM widget_seen").Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestAlphaToBetaToAlphaCannotCommit attempts the exact escape sequence:
// open an alpha transaction, switch to beta's owner role, write beta's
// schema, switch back to alpha, commit. Savepoints let the attempt keep going
// after each refusal, so the test also proves the transaction never holds
// beta's authority at any point, not just at commit time.
func TestAlphaToBetaToAlphaCannotCommit(t *testing.T) {
	env := pgtest.New(t)
	alpha := env.Runner(t, pgtest.Alpha)

	switches := []string{
		"SET LOCAL ROLE " + betaRole,
		"SET ROLE " + betaRole,
		"SELECT set_config('role', '" + betaRole + "', true)",
		"SELECT set_config('role', '" + betaRole + "', false)",
		"SET SESSION AUTHORIZATION " + betaRole,
	}
	err := alpha.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if _, err := tx.Exec(c, "SAVEPOINT attempt"); err != nil {
			return err
		}
		// Step 2: every way of assuming beta's role is refused by PostgreSQL.
		for _, stmt := range switches {
			if _, err := tx.Exec(c, stmt); pgCode(err) != "42501" {
				t.Errorf("%q: err = %v, want permission denied (42501)", stmt, err)
			}
			if _, err := tx.Exec(c, "ROLLBACK TO SAVEPOINT attempt"); err != nil {
				return err
			}
		}
		var user string
		if err := tx.QueryRow(c, "SELECT current_user").Scan(&user); err != nil {
			return err
		}
		if user != alphaRole {
			t.Errorf("after refused switches current_user = %q, want %q", user, alphaRole)
		}
		// Step 3: beta's schema is not writable from this transaction.
		if _, err := tx.Exec(c, betaWrite); pgCode(err) != "42501" {
			t.Errorf("beta write: err = %v, want permission denied", err)
		}
		if _, err := tx.Exec(c, "ROLLBACK TO SAVEPOINT attempt"); err != nil {
			return err
		}
		// Step 4: "switching back" to alpha is a no-op that grants nothing new.
		if _, err := tx.Exec(c, "SET LOCAL ROLE "+alphaRole); err != nil {
			return err
		}
		return insertWidget(c, tx) // a legitimate alpha write in the same transaction
	})
	// Step 5: the commit carries only alpha's own write.
	if err != nil {
		t.Fatalf("alpha transaction: %v", err)
	}
	if n := betaRows(t, env); n != 0 {
		t.Fatalf("beta schema has %d rows; an alpha transaction must never write it", n)
	}
	if n := widgets(t, env); n != 1 {
		t.Fatalf("alpha widgets = %d, want the one legitimate write", n)
	}
}

// TestRefusedSwitchAbortsWithoutSavepoint: without savepoints, the refused
// switch aborts the transaction; nothing after it runs and nothing commits.
func TestRefusedSwitchAbortsWithoutSavepoint(t *testing.T) {
	env := pgtest.New(t)
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if err := insertWidget(c, tx); err != nil {
			return err
		}
		_, _ = tx.Exec(c, "SET LOCAL ROLE "+betaRole) // refused; error deliberately ignored
		_, _ = tx.Exec(c, betaWrite)
		_, _ = tx.Exec(c, "SET LOCAL ROLE "+alphaRole)
		return nil // try to commit anyway
	})
	if err == nil {
		t.Fatal("a transaction that attempted a role switch must not commit")
	}
	if betaRows(t, env) != 0 || widgets(t, env) != 0 {
		t.Fatal("nothing from the aborted transaction may persist")
	}
}

// TestResetRoleGrantsNoAuthority: RESET ROLE drops to the NOINHERIT login
// role, which can write nothing; the commit is then refused.
func TestResetRoleGrantsNoAuthority(t *testing.T) {
	env := pgtest.New(t)
	err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		if _, err := tx.Exec(c, "SAVEPOINT s"); err != nil {
			return err
		}
		if _, err := tx.Exec(c, "RESET ROLE"); err != nil {
			return err
		}
		for _, stmt := range []string{betaWrite, "INSERT INTO syn_alpha.widget (label) VALUES ('x')"} {
			if _, err := tx.Exec(c, stmt); pgCode(err) != "42501" {
				t.Errorf("%q after RESET ROLE: err = %v, want permission denied", stmt, err)
			}
			if _, err := tx.Exec(c, "ROLLBACK TO SAVEPOINT s"); err != nil {
				return err
			}
			if _, err := tx.Exec(c, "RESET ROLE"); err != nil {
				return err
			}
		}
		return nil
	})
	if !errors.Is(err, db.ErrRoleEscaped) {
		t.Fatalf("commit after RESET ROLE: err = %v, want ErrRoleEscaped", err)
	}
	if betaRows(t, env) != 0 {
		t.Fatal("beta schema was written")
	}
}

// TestSharedRoleWouldAllowLateralSwitch documents the root cause the Runner
// guards against: outside the Runner, a session whose login role is a member
// of two owner roles CAN do alpha → beta → alpha in one transaction. The
// transaction is rolled back; nothing persists.
func TestSharedRoleWouldAllowLateralSwitch(t *testing.T) {
	env := pgtest.New(t)
	conn, err := pgx.Connect(ctx(t), env.LoginDSN(t, pgtest.Alpha, pgtest.Beta))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx(t))
	tx, err := conn.Begin(ctx(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx(t)) }()
	for _, stmt := range []string{"SET LOCAL ROLE " + alphaRole, "SET LOCAL ROLE " + betaRole, betaWrite, "SET LOCAL ROLE " + alphaRole} {
		if _, err := tx.Exec(ctx(t), stmt); err != nil {
			t.Fatalf("precondition: %q failed (%v); the shared-role exploit should be possible without the Runner", stmt, err)
		}
	}
}

// TestNonIsolatedSessionRolesAreRefused: a pool whose login role could reach
// several owner roles, or a superuser pool, never gets a transaction, so the
// alpha→beta→alpha sequence cannot even start.
func TestNonIsolatedSessionRolesAreRefused(t *testing.T) {
	env := pgtest.New(t)
	pools := map[string]*db.Pool{
		"member of alpha and beta": pgtest.OpenPool(t, env.LoginDSN(t, pgtest.Alpha, pgtest.Beta)),
		"superuser":                pgtest.OpenPool(t, env.DSN),
		"member of nothing":        pgtest.OpenPool(t, env.LoginDSN(t)),
	}
	for name, pool := range pools {
		r, err := pool.Runner(pgtest.Alpha)
		if err != nil {
			t.Fatal(err)
		}
		ran := false
		err = r.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
			ran = true
			_, _ = tx.Exec(c, "SET LOCAL ROLE "+betaRole)
			_, _ = tx.Exec(c, betaWrite)
			_, _ = tx.Exec(c, "SET LOCAL ROLE "+alphaRole)
			return nil
		})
		if !errors.Is(err, db.ErrSessionRoleNotIsolated) {
			t.Fatalf("%s: err = %v, want ErrSessionRoleNotIsolated", name, err)
		}
		if ran {
			t.Fatalf("%s: caller SQL ran in a non-isolated session", name)
		}
	}
	if betaRows(t, env) != 0 {
		t.Fatal("beta schema was written")
	}
}

// TestDedicatedRolesStillWork: the restricted model keeps normal writes and
// reads working for each owner.
func TestDedicatedRolesStillWork(t *testing.T) {
	env := pgtest.New(t)
	if err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, insertWidget); err != nil {
		t.Fatal(err)
	}
	err := env.Runner(t, pgtest.Beta).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		_, err := tx.Exec(c, "INSERT INTO widget_seen (event_id, label) VALUES ('00000000-0000-4000-8000-00000000000b', 'own')")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if widgets(t, env) != 1 || betaRows(t, env) != 1 {
		t.Fatal("owners must still write their own schemas")
	}
}
