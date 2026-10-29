package audit_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/refs"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/audit"
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

var at, _ = timeutil.ParseRFC3339("2026-10-29T10:00:00Z")

func sample(t *testing.T) audit.Event {
	t.Helper()
	id, err := audit.NewEventID()
	if err != nil {
		t.Fatal(err)
	}
	reason, corr := "synthetic reason", "corr-1"
	actor := refs.NativeRef{Domain: refs.DomainEvent, ID: "11"}
	return audit.Event{
		ID: id, Actor: &actor, Subject: refs.NativeRef{Domain: refs.DomainEvent, ID: "12"},
		Action: "platform.synthetic_checked", Reason: &reason, CorrelationID: &corr, OccurredAt: at,
		Metadata: map[string]string{"source": "test", "count": "1"},
	}
}

func TestEventValidation(t *testing.T) {
	if err := sample(t).Validate(); err != nil {
		t.Fatal(err)
	}
	empty := ""
	big := map[string]string{}
	for i := 0; i < audit.MaxMetadataEntries+1; i++ {
		big["k"+strings.Repeat("a", i%10)+string(rune('a'+i%26))+string(rune('a'+i/26))] = "v"
	}
	bad := []func(*audit.Event){
		func(e *audit.Event) { e.ID = "x" },
		func(e *audit.Event) { e.Actor = &refs.NativeRef{Domain: "ACTOR", ID: "1"} },
		func(e *audit.Event) { e.Subject = refs.NativeRef{} },
		func(e *audit.Event) { e.Subject = refs.NativeRef{Domain: refs.DomainEvent, ID: "09123456789"} },
		func(e *audit.Event) { e.Action = "nodot" },
		func(e *audit.Event) { e.Reason = &empty },
		func(e *audit.Event) { e.CorrelationID = &empty },
		func(e *audit.Event) { e.OccurredAt = timeutil.Instant{} },
		func(e *audit.Event) { e.Metadata = big },
		func(e *audit.Event) { e.Metadata = map[string]string{"Bad Key": "v"} },
		func(e *audit.Event) { e.Metadata = map[string]string{"user_password": "x"} },
		func(e *audit.Event) { e.Metadata = map[string]string{"access_token": "x"} },
		func(e *audit.Event) { e.Metadata = map[string]string{"note": strings.Repeat("v", 257)} },
	}
	for i, m := range bad {
		e := sample(t)
		m(&e)
		if err := e.Validate(); !errors.Is(err, audit.ErrInvalidEvent) {
			t.Fatalf("mutation %d accepted: %v", i, err)
		}
	}
}

func TestAppendAndOwnership(t *testing.T) {
	env := pgtest.New(t)
	auditR := env.Runner(t, owner.Audit)
	e := sample(t)
	if err := auditR.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return audit.Append(c, tx, e) }); err != nil {
		t.Fatal(err)
	}
	minimal := sample(t)
	minimal.Actor, minimal.Reason, minimal.CorrelationID, minimal.Metadata = nil, nil, nil, nil
	if err := auditR.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return audit.Append(c, tx, minimal) }); err != nil {
		t.Fatal(err)
	}
	var n int
	var action, meta string
	err := auditR.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		if err := tx.QueryRow(c, "SELECT count(*) FROM audit.audit_event").Scan(&n); err != nil {
			return err
		}
		return tx.QueryRow(c, "SELECT action, metadata::text FROM audit.audit_event WHERE event_id = $1", e.ID).Scan(&action, &meta)
	})
	if err != nil || n != 2 || action != e.Action || !strings.Contains(meta, `"source": "test"`) {
		t.Fatalf("stored n=%d action=%q meta=%s err=%v", n, action, meta, err)
	}
	// Only the audit owner may append; other modules emit outbox events instead.
	if err := env.Runner(t, pgtest.Alpha).InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return audit.Append(c, tx, sample(t))
	}); !errors.Is(err, audit.ErrWrongOwner) {
		t.Fatalf("non-audit append err = %v", err)
	}
	if err := auditR.InTx(ctx(t), db.TxOptions{ReadOnly: true}, func(c context.Context, tx db.Tx) error {
		return audit.Append(c, tx, sample(t))
	}); !errors.Is(err, audit.ErrReadOnly) {
		t.Fatalf("read-only append err = %v", err)
	}
	if err := auditR.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
		return audit.Append(c, tx, e)
	}); err == nil {
		t.Fatal("duplicate audit event id must be rejected")
	}
}

func code(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestAuditIsAppendOnly(t *testing.T) {
	env := pgtest.New(t)
	auditR := env.Runner(t, owner.Audit)
	e := sample(t)
	if err := auditR.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error { return audit.Append(c, tx, e) }); err != nil {
		t.Fatal(err)
	}
	// The owner role has no UPDATE/DELETE/TRUNCATE grant.
	for _, stmt := range []string{
		"UPDATE audit.audit_event SET action = 'platform.tampered'",
		"DELETE FROM audit.audit_event",
		"TRUNCATE audit.audit_event",
	} {
		err := auditR.InTx(ctx(t), db.TxOptions{}, func(c context.Context, tx db.Tx) error {
			_, err := tx.Exec(c, stmt)
			return err
		})
		if code(err) != "42501" {
			t.Fatalf("%q as owner role: err = %v, want permission denied", stmt, err)
		}
	}
	// Even the table owner / superuser is stopped by the trigger.
	conn, err := pgx.Connect(ctx(t), env.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx(t))
	for _, stmt := range []string{
		"UPDATE audit.audit_event SET action = 'platform.tampered'",
		"DELETE FROM audit.audit_event",
		"TRUNCATE audit.audit_event",
	} {
		_, err := conn.Exec(ctx(t), stmt)
		if err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Fatalf("%q as superuser: err = %v, want append-only rejection", stmt, err)
		}
	}
	var n int
	if err := conn.QueryRow(ctx(t), "SELECT count(*) FROM audit.audit_event WHERE action = $1", e.Action).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audit row changed: n=%d err=%v", n, err)
	}
}
