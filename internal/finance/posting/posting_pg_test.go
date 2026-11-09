package posting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// pgStore is a TEST-ONLY store on a SYNTHETIC schema of a disposable
// database. Production has no posting store (no posting authority).
type pgStore struct {
	pool      *pgxpool.Pool
	crashOnce bool
	mu        sync.Mutex
}

func (p *pgStore) Lookup(key string) (Recorded, bool, error) {
	var hash string
	var raw []byte
	err := p.pool.QueryRow(context.Background(), `SELECT request_hash, intent FROM syn_ledger.posting WHERE business_key = $1`, key).Scan(&hash, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recorded{}, false, nil
	}
	if err != nil {
		return Recorded{}, false, err
	}
	var i Intent
	if err := json.Unmarshal(raw, &i); err != nil {
		return Recorded{}, false, err
	}
	return Recorded{Intent: i, Hash: hash}, true, nil
}

func (p *pgStore) Reversed(key string) (bool, error) {
	var n int
	err := p.pool.QueryRow(context.Background(), `SELECT count(*) FROM syn_ledger.posting WHERE reverses = $1`, key).Scan(&n)
	return n > 0, err
}

func (p *pgStore) Append(r Recorded) (bool, error) {
	raw, err := json.Marshal(r.Intent)
	if err != nil {
		return false, err
	}
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `INSERT INTO syn_ledger.posting (business_key, request_hash, unit, reverses, intent) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`, r.Intent.BusinessKey, r.Hash, r.Intent.Unit.String(), r.Intent.Reverses, raw)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil // a concurrent writer recorded the key: no second effect
	}
	for _, l := range r.Intent.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO syn_ledger.line (business_key, account, side, amount, unit) VALUES ($1, $2, $3, $4, $5)`,
			r.Intent.BusinessKey, l.Account, l.Side, l.Amount, r.Intent.Unit.String()); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, ErrOutcomeUnknown
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.crashOnce {
		p.crashOnce = false
		return true, ErrOutcomeUnknown // committed, acknowledgement lost
	}
	return true, nil
}

// R4: on PostgreSQL, concurrent duplicate posting requests produce exactly
// one economic effect; a lost acknowledgement is reconciled, not retried
// blindly; a reversal is a separate row and the original is never updated.
func TestPostingContractOnPostgres(t *testing.T) {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(pgtest.CreateDatabase(t))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 32
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `CREATE SCHEMA syn_ledger;
		CREATE TABLE syn_ledger.posting (business_key text PRIMARY KEY, request_hash text NOT NULL, unit text NOT NULL CHECK (unit IN ('IRR', 'IRT')),
		  reverses text NULL UNIQUE REFERENCES syn_ledger.posting, intent jsonb NOT NULL);
		CREATE TABLE syn_ledger.line (business_key text NOT NULL REFERENCES syn_ledger.posting, account text NOT NULL,
		  side text NOT NULL CHECK (side IN ('DEBIT', 'CREDIT')), amount bigint NOT NULL CHECK (amount > 0), unit text NOT NULL);
		CREATE FUNCTION syn_ledger.append_only() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'ledger is append-only'; END $$;
		CREATE TRIGGER posting_append_only BEFORE UPDATE OR DELETE ON syn_ledger.posting FOR EACH ROW EXECUTE FUNCTION syn_ledger.append_only();
		CREATE TRIGGER line_append_only BEFORE UPDATE OR DELETE ON syn_ledger.line FOR EACH ROW EXECUTE FUNCTION syn_ledger.append_only();`); err != nil {
		t.Fatal(err)
	}
	st := &pgStore{pool: pool}
	a := intent("finance|invoice_issued|inv-pg|receivable|IRR", 1_000)
	var mu sync.Mutex
	got := map[string]int{}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			o := Apply(st, a)
			mu.Lock()
			got[fmt.Sprintf("%s/replay=%v", o.Status, o.Replay)]++
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()
	var rows, lines int
	var net int64
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM syn_ledger.posting), (SELECT count(*) FROM syn_ledger.line),
		(SELECT coalesce(sum(CASE side WHEN 'DEBIT' THEN amount ELSE -amount END), 0) FROM syn_ledger.line
		 WHERE account = 'customer_receivable' AND unit = 'IRR')`).Scan(&rows, &lines, &net); err != nil {
		t.Fatal(err)
	}
	if got["SUCCESS/replay=false"] != 1 || got["SUCCESS/replay=true"] != 19 || rows != 1 || lines != 2 || net != 1_000 {
		t.Fatalf("concurrent duplicates: %v rows %d lines %d net %d", got, rows, lines, net)
	}
	// Key reuse with a different intent.
	if o := Apply(st, intent("finance|invoice_issued|inv-pg|receivable|IRR", 2_000)); o.Status != result.Conflict {
		t.Fatalf("key reuse: %+v", o)
	}
	// Lost acknowledgement: UNKNOWN, then reconciled by key (no 2nd row).
	st.crashOnce = true
	b := intent("finance|payment_accepted|att-pg|cash|IRR", 400)
	if o := Apply(st, b); o.Status != result.Unknown {
		t.Fatalf("lost ack: %+v", o)
	}
	if o := Apply(st, b); o.Status != result.Success || !o.Replay {
		t.Fatalf("reconcile after unknown: %+v", o)
	}
	// Reversal: a separate compensating row; at most once; the original
	// row cannot be updated or deleted.
	rev, _ := Reversal(a, "finance|invoice_issued_reversal|inv-pg|receivable|IRR", t0.Add(3*time.Hour), t0.Add(3*time.Hour))
	if o := Apply(st, rev); o.Status != result.Success {
		t.Fatalf("reversal: %+v", o)
	}
	rev.BusinessKey = "finance|invoice_issued_reversal_again|inv-pg|receivable|IRR"
	if o := Apply(st, rev); o.Status != result.Conflict {
		t.Fatalf("double reversal: %+v", o)
	}
	for _, sql := range []string{`UPDATE syn_ledger.posting SET unit = 'IRT'`, `DELETE FROM syn_ledger.line`} {
		if _, err := pool.Exec(ctx, sql); err == nil {
			t.Fatalf("%s must be refused (append-only)", sql)
		}
	}
	if err := pool.QueryRow(ctx, `SELECT coalesce(sum(CASE side WHEN 'DEBIT' THEN amount ELSE -amount END), 0) FROM syn_ledger.line
		WHERE account = 'customer_receivable' AND unit = 'IRR'`).Scan(&net); err != nil || net != 400 { // a reversed (0) + b (400)
		t.Fatalf("net after reversal %d %v", net, err)
	}
}
