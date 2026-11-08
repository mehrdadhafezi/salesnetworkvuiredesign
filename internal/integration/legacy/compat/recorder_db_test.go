package compat

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	integrationmigrations "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/migrations"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// R2 review P2 against the real recorder and transaction runner:
// a write blocked by the database is bounded by the record deadline, and a
// write after the request was cancelled is persisted.
func TestDBRecorderBoundedAndDetached(t *testing.T) {
	env := pgtest.New(t)
	ctx := context.Background()
	if _, err := migrations.Up(ctx, env.DSN, integrationmigrations.Source()); err != nil {
		t.Fatal(err)
	}
	dbrec, err := NewDBRecorder(env.Runner(t, integration.Module))
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.Connect(ctx, env.DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	rows := func() int {
		var n int
		if err := admin.QueryRow(ctx, `SELECT count(*) FROM integration.writeback_exchange`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// 1. The evidence table is locked (a stalled store): the call returns
	// within the record deadline as UNKNOWN although legacy acknowledged.
	lock, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(ctx, `LOCK TABLE integration.writeback_exchange IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	c, _ := setupWith(t, applied200, dbrec, 300*time.Millisecond)
	start := time.Now()
	res, err := c.ApplyLeadOutcome(ctx, lead())
	elapsed := time.Since(start)
	if !isUnknown(err) || res.Outcome == result.Success {
		t.Fatalf("blocked evidence store: %+v %v", res, err)
	}
	if elapsed > 3*time.Second {
		t.Fatalf("the evidence write must be bounded by its deadline: %s", elapsed)
	}
	noLeak(t, err)
	if err := lock.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if n := rows(); n != 0 {
		t.Fatalf("a timed-out write must not have committed: %d", n)
	}

	// 2. The request is cancelled mid-flight: the evidence still commits.
	release := make(chan struct{})
	c2, _ := setupWith(t, func(req request) (int, string, bool) { <-release; return applied200(req) }, dbrec, 2*time.Second)
	defer close(release)
	cctx, cancel := context.WithCancel(ctx)
	time.AfterFunc(50*time.Millisecond, cancel)
	if _, err := c2.ApplyLeadOutcome(cctx, lead()); !isUnknown(err) {
		t.Fatalf("cancelled request: %v", err)
	}
	var verdict string
	if err := admin.QueryRow(ctx, `SELECT verdict FROM integration.writeback_exchange`).Scan(&verdict); err != nil || verdict != string(VerdictTransport) {
		t.Fatalf("evidence after cancellation: %q %v", verdict, err)
	}

	// 3. Store available again: a proven acknowledgement is recorded.
	c3, _ := setupWith(t, applied200, dbrec, 2*time.Second)
	if res, err := c3.ApplyLeadOutcome(ctx, lead()); err != nil || res.Outcome != result.Success || rows() != 2 {
		t.Fatalf("recovered store: %+v %v rows=%d", res, err, rows())
	}
}
