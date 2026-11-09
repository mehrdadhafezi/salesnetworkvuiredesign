package shadow_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/issuance"
	casesapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/claimgate"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/release"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/money"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/pgtest"
)

// R4 SYNTHETIC RACE HARNESS of the ADR-01 CANDIDATE reservation protocol
// (Architecture V1 §6; Gate-0 INV-009; Cross-Role §8 / F01).
//
// It runs on a disposable PostgreSQL database with SYNTHETIC schemas that
// stand in for three modules (syn_claim = cases, syn_invoice = billing,
// syn_hold = custody), one transaction per module and no cross-module
// transaction. Each step asks the real R4 evaluators (claimgate, issuance,
// release) and proceeds only when `PreconditionsMet` (i.e. only the open
// decisions block it) — a SIMULATION of what the candidate protocol would
// do if it were accepted.
//
// Passing these tests is evidence for the decision, NOT approval of ADR-01
// or F01, and does not activate anything: no target module table, route,
// flag or capability is touched.

const r4Schema = `
CREATE SCHEMA syn_claim; CREATE SCHEMA syn_invoice; CREATE SCHEMA syn_hold;
CREATE TABLE syn_claim.work_case (case_id text PRIMARY KEY, version bigint NOT NULL, original_seller text NOT NULL,
  credit_owner text NOT NULL, return_state text NULL CHECK (return_state IN ('IN_PROGRESS')), customer text NOT NULL, lineage text NOT NULL);
CREATE TABLE syn_claim.link (case_id text NOT NULL REFERENCES syn_claim.work_case, source text NOT NULL, invoice_ref text NOT NULL,
  resolution text NOT NULL, class text NOT NULL);
CREATE TABLE syn_claim.claim (claim_id text PRIMARY KEY, case_id text NOT NULL REFERENCES syn_claim.work_case, seller text NOT NULL,
  state text NOT NULL CHECK (state IN ('ACTIVE', 'CONFIRMED', 'RECONCILIATION_REQUIRED')), case_version bigint NOT NULL,
  expires_at timestamptz NOT NULL, invoice_id text NULL);
-- At most one unresolved claim per case (two sellers can never both hold one).
CREATE UNIQUE INDEX claim_one_open ON syn_claim.claim (case_id) WHERE state IN ('ACTIVE', 'RECONCILIATION_REQUIRED');
CREATE TABLE syn_claim.history (seq bigserial PRIMARY KEY, case_id text NOT NULL, event text NOT NULL, actor text NOT NULL,
  detail text NOT NULL DEFAULT '');
CREATE FUNCTION syn_claim.append_only() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'history is append-only'; END $$;
CREATE TRIGGER history_append_only BEFORE UPDATE OR DELETE ON syn_claim.history FOR EACH ROW EXECUTE FUNCTION syn_claim.append_only();
CREATE TABLE syn_claim.idem (key text PRIMARY KEY, request_hash text NOT NULL, claim_id text NULL);
CREATE TABLE syn_claim.inbox (message_id text PRIMARY KEY);
CREATE TABLE syn_invoice.invoice (invoice_id text PRIMARY KEY, claim_id text NOT NULL UNIQUE, case_id text NOT NULL, seller text NOT NULL,
  original_seller text NOT NULL, amount bigint NOT NULL CHECK (amount > 0), unit text NOT NULL CHECK (unit IN ('IRR', 'IRT')));
CREATE TABLE syn_invoice.idem (key text PRIMARY KEY, request_hash text NOT NULL, invoice_id text NULL);
CREATE TABLE syn_invoice.outbox (event_id text PRIMARY KEY, claim_id text NOT NULL, invoice_id text NOT NULL, dispatched boolean NOT NULL DEFAULT false);
CREATE TABLE syn_hold.assignment (case_id text PRIMARY KEY, holder text NULL);
CREATE TABLE syn_hold.event (seq bigserial PRIMARY KEY, case_id text NOT NULL, from_holder text NULL, to_holder text NULL, actor text NOT NULL);
`

type r4h struct {
	t    *testing.T
	pool *pgxpool.Pool
	mu   sync.Mutex
	seq  int
	// afterEval, when set, runs after a gate evaluation and before any
	// write: it forces the worst-case interleaving (every concurrent
	// transaction has evaluated before any of them writes), so the
	// database constraints, not timing luck, must prevent the race.
	afterEval func()
}

func (h *r4h) pause() {
	if h.afterEval != nil {
		h.afterEval()
	}
}

func newR4(t *testing.T) *r4h {
	t.Helper()
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
	if _, err := pool.Exec(ctx, r4Schema); err != nil {
		t.Fatal(err)
	}
	return &r4h{t: t, pool: pool}
}

func (h *r4h) id(prefix string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	return fmt.Sprintf("%s-%04d", prefix, h.seq)
}

// seedCase creates a synthetic case held by holder.
func (h *r4h) seedCase(holder string, links ...casesapi.DependencyLink) string {
	h.t.Helper()
	c := h.id("case")
	ctx := context.Background()
	if _, err := h.pool.Exec(ctx, `INSERT INTO syn_claim.work_case VALUES ($1, 1, 'seller-orig', 'seller-orig', NULL, 'RESOLVED', 'KNOWN')`, c); err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.pool.Exec(ctx, `INSERT INTO syn_hold.assignment VALUES ($1, $2)`, c, holder); err != nil {
		h.t.Fatal(err)
	}
	for _, l := range links {
		if _, err := h.pool.Exec(ctx, `INSERT INTO syn_claim.link VALUES ($1, $2, $3, $4, $5)`, c, string(l.Source), l.InvoiceRef,
			string(l.Resolution), string(l.Class)); err != nil {
			h.t.Fatal(err)
		}
	}
	return c
}

func (h *r4h) exec(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.pool.Exec(context.Background(), sql, args...); err != nil {
		h.t.Fatal(err)
	}
}

func (h *r4h) count(sql string, args ...any) int {
	h.t.Helper()
	var n int
	if err := h.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		h.t.Fatal(err)
	}
	return n
}

// caseFacts reads the fresh facts of a case inside the cases transaction
// that holds the case row lock: version, return state, open claim, and the
// union of links (synthetic link rows plus invoices of confirmed claims).
type caseFacts struct {
	version   int64
	returning bool
	open      *claimgate.ActiveClaim
	confirmed bool
	deps      casesapi.DependencyVerdict
	customer  casesapi.IdentityResolution
	lineage   casesapi.Lineage
	holder    *string
}

func lockCase(ctx context.Context, tx pgx.Tx, caseID string) (caseFacts, error) {
	var f caseFacts
	var ret *string
	var cust, lin string
	if err := tx.QueryRow(ctx, `SELECT version, return_state, customer, lineage FROM syn_claim.work_case WHERE case_id = $1 FOR UPDATE`, caseID).
		Scan(&f.version, &ret, &cust, &lin); err != nil {
		return f, err
	}
	f.returning, f.customer, f.lineage = ret != nil, casesapi.IdentityResolution(cust), casesapi.Lineage(lin)
	var c claimgate.ActiveClaim
	err := tx.QueryRow(ctx, `SELECT claim_id, seller FROM syn_claim.claim WHERE case_id = $1 AND state IN ('ACTIVE', 'RECONCILIATION_REQUIRED')`, caseID).
		Scan(&c.ClaimID, &c.SellerUserID)
	switch {
	case err == nil:
		f.open = &c
	case !errors.Is(err, pgx.ErrNoRows):
		return f, err
	}
	var links []casesapi.DependencyLink
	rows, err := tx.Query(ctx, `SELECT source, invoice_ref, resolution, class FROM syn_claim.link WHERE case_id = $1`, caseID)
	if err != nil {
		return f, err
	}
	for rows.Next() {
		var l casesapi.DependencyLink
		var src, res, class string
		if err := rows.Scan(&src, &l.InvoiceRef, &res, &class); err != nil {
			rows.Close()
			return f, err
		}
		l.Source, l.Resolution, l.Class = casesapi.LinkSource(src), casesapi.LinkResolution(res), casesapi.InvoiceClass(class)
		links = append(links, l)
	}
	rows.Close()
	// A confirmed claim's invoice is an unpaid (committed) invoice link.
	var inv *string
	if err := tx.QueryRow(ctx, `SELECT max(invoice_id) FROM syn_claim.claim WHERE case_id = $1 AND state = 'CONFIRMED'`, caseID).Scan(&inv); err != nil {
		return f, err
	}
	if inv != nil {
		f.confirmed = true
		links = append(links, casesapi.DependencyLink{Source: casesapi.LinkLegacyInvoiceLead, InvoiceRef: *inv, Resolution: casesapi.LinkResolved,
			Class: casesapi.ClassCommitted})
	}
	f.deps = casesapi.ClassifyDependencies(links, true)
	// Custody published read (holder): a read, never a write.
	if err := tx.QueryRow(ctx, `SELECT holder FROM syn_hold.assignment WHERE case_id = $1`, caseID).Scan(&f.holder); err != nil {
		return f, err
	}
	return f, nil
}

func pgStatus(err error) result.Status {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && (pe.Code == "23505" || pe.Code == "40001" || pe.Code == "40P01") {
		return result.Conflict
	}
	return result.Failed
}

// claim: the cases transaction of ClaimCaseForInvoice (simulated).
func (h *r4h) claim(caseID, actor string, verified bool, expected int64, key string) (result.Status, string) {
	ctx := context.Background()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return result.Failed, ""
	}
	defer tx.Rollback(ctx)
	f, err := lockCase(ctx, tx, caseID)
	if err != nil {
		return result.Failed, ""
	}
	g := claimgate.Evaluate(claimgate.Input{CaseID: caseID, CasePresent: true, ActorUserID: actor, ActorVerified: verified, HolderUserID: f.holder,
		ExpectedVersion: expected, CurrentVersion: f.version, ActiveClaim: f.open, ReturnInProgress: f.returning, Dependencies: f.deps,
		Customer: f.customer, Lineage: f.lineage})
	if !g.PreconditionsMet {
		return g.Status, ""
	}
	h.pause()
	id := h.id("claim")
	if _, err := tx.Exec(ctx, `INSERT INTO syn_claim.claim VALUES ($1, $2, $3, 'ACTIVE', $4, now() + interval '10 minutes', NULL)`, id, caseID, actor,
		f.version+1); err != nil {
		return pgStatus(err), ""
	}
	if _, err := tx.Exec(ctx, `UPDATE syn_claim.work_case SET version = version + 1 WHERE case_id = $1`, caseID); err != nil {
		return pgStatus(err), ""
	}
	if _, err := tx.Exec(ctx, `INSERT INTO syn_claim.history (case_id, event, actor, detail) VALUES ($1, 'CLAIMED', $2, $3)`, caseID, actor, id); err != nil {
		return pgStatus(err), ""
	}
	if err := tx.Commit(ctx); err != nil {
		return pgStatus(err), ""
	}
	return result.Success, id
}

// issue: the billing transaction of IssueInvoice (simulated). fail injects
// a failure after the invoice row is written and before commit.
func (h *r4h) issue(caseID, claimID, actor, key, hash string, amount int64, unit money.Unit, fail bool) (result.Status, string) {
	ctx := context.Background()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return result.Failed, ""
	}
	defer tx.Rollback(ctx)
	idem := issuance.Idempotency{Key: key, RequestHash: hash}
	tag, err := tx.Exec(ctx, `INSERT INTO syn_invoice.idem VALUES ($1, $2, NULL) ON CONFLICT (key) DO NOTHING`, key, hash)
	if err != nil {
		return pgStatus(err), ""
	}
	if tag.RowsAffected() == 0 {
		var prior string
		var inv *string
		if err := tx.QueryRow(ctx, `SELECT request_hash, invoice_id FROM syn_invoice.idem WHERE key = $1`, key).Scan(&prior, &inv); err != nil {
			return result.Failed, ""
		}
		if prior == hash && inv != nil {
			return result.Success, *inv // replay: the original outcome, no new effect
		}
		idem.PriorRequestHash = &prior
	}
	// Cases published read of the claim (synchronous validation; no lock).
	var c issuance.Claim
	var expires time.Time
	var ver int64
	if err := tx.QueryRow(ctx, `SELECT claim_id, case_id, seller, state, case_version, expires_at FROM syn_claim.claim WHERE claim_id = $1`, claimID).
		Scan(&c.ClaimID, &c.CaseID, &c.SellerUserID, &c.State, &c.CaseVersion, &expires); err != nil {
		return result.Failed, ""
	}
	c.Expired = !expires.After(time.Now())
	if err := tx.QueryRow(ctx, `SELECT version FROM syn_claim.work_case WHERE case_id = $1`, caseID).Scan(&ver); err != nil {
		return result.Failed, ""
	}
	var issued *string
	_ = tx.QueryRow(ctx, `SELECT invoice_id FROM syn_invoice.invoice WHERE claim_id = $1`, claimID).Scan(&issued)
	orig := "seller-orig"
	g := issuance.Evaluate(issuance.Input{CaseID: caseID, ActorUserID: actor, ActorVerified: true, Claim: &c, ExpectedCaseVersion: ver,
		IssuedForClaim: issued, Idempotency: idem, Attribution: issuance.Attribution{SnapshotOriginalSeller: &orig, ProposedOriginalSeller: &orig},
		Total: money.Money{Amount: amount, Unit: unit}, Dependencies: casesapi.ClassifyDependencies(nil, true),
		Customer: casesapi.IdentityResolved, Lineage: casesapi.LineageKnown})
	if !g.PreconditionsMet {
		return g.Status, ""
	}
	h.pause()
	inv := h.id("inv")
	if _, err := tx.Exec(ctx, `INSERT INTO syn_invoice.invoice VALUES ($1, $2, $3, $4, $5, $6, $7)`, inv, claimID, caseID, actor, orig, amount,
		unit.String()); err != nil {
		return pgStatus(err), ""
	}
	if fail {
		return result.Failed, "" // interrupted before commit: rolled back
	}
	if _, err := tx.Exec(ctx, `UPDATE syn_invoice.idem SET invoice_id = $2 WHERE key = $1`, key, inv); err != nil {
		return pgStatus(err), ""
	}
	if _, err := tx.Exec(ctx, `INSERT INTO syn_invoice.outbox VALUES ($1, $2, $3, false)`, "evt-"+inv, claimID, inv); err != nil {
		return pgStatus(err), ""
	}
	if err := tx.Commit(ctx); err != nil {
		return pgStatus(err), ""
	}
	return result.Success, inv
}

// relay delivers InvoiceIssued to cases (inbox-deduplicated confirmation).
// crash stops after the cases commit, before the outbox is marked
// dispatched, so the next relay run redelivers.
func (h *r4h) relay(crash bool) {
	h.t.Helper()
	ctx := context.Background()
	rows, err := h.pool.Query(ctx, `SELECT event_id, claim_id, invoice_id FROM syn_invoice.outbox WHERE NOT dispatched ORDER BY event_id`)
	if err != nil {
		h.t.Fatal(err)
	}
	type ev struct{ id, claim, inv string }
	var evs []ev
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.id, &e.claim, &e.inv); err != nil {
			h.t.Fatal(err)
		}
		evs = append(evs, e)
	}
	rows.Close()
	for _, e := range evs {
		tx, err := h.pool.Begin(ctx)
		if err != nil {
			h.t.Fatal(err)
		}
		tag, err := tx.Exec(ctx, `INSERT INTO syn_claim.inbox VALUES ($1) ON CONFLICT DO NOTHING`, e.id)
		if err != nil {
			h.t.Fatal(err)
		}
		if tag.RowsAffected() == 1 {
			var caseID, state string
			if err := tx.QueryRow(ctx, `SELECT case_id, state FROM syn_claim.claim WHERE claim_id = $1 FOR UPDATE`, e.claim).Scan(&caseID, &state); err != nil {
				h.t.Fatal(err)
			}
			event, sql := "CONFIRMED", `UPDATE syn_claim.claim SET state = 'CONFIRMED', invoice_id = $2 WHERE claim_id = $1`
			if state != issuance.ClaimActive {
				// A late confirmation never releases or silently resolves:
				// the claim stays in reconciliation with the invoice noted.
				event, sql = "LATE_CONFIRMATION", `UPDATE syn_claim.claim SET invoice_id = $2 WHERE claim_id = $1`
			}
			if _, err := tx.Exec(ctx, sql, e.claim, e.inv); err != nil {
				h.t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO syn_claim.history (case_id, event, actor, detail) VALUES ($1, $2, 'relay', $3)`, caseID, event,
				e.inv); err != nil {
				h.t.Fatal(err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			h.t.Fatal(err)
		}
		if !crash {
			h.exec(`UPDATE syn_invoice.outbox SET dispatched = true WHERE event_id = $1`, e.id)
		}
	}
}

// expire moves overdue ACTIVE claims to reconciliation (never a release).
func (h *r4h) expire() {
	h.t.Helper()
	h.exec(`WITH x AS (UPDATE syn_claim.claim SET state = 'RECONCILIATION_REQUIRED' WHERE state = 'ACTIVE' AND expires_at <= now()
		RETURNING case_id, claim_id) INSERT INTO syn_claim.history (case_id, event, actor, detail) SELECT case_id, 'CLAIM_EXPIRED_TO_RECONCILIATION',
		'expiry', claim_id FROM x`)
}

// beginReturn: the cases transaction that reserves a return / reset /
// release (simulated), then the custody move in custody's own transaction.
func (h *r4h) beginReturn(caseID, actor string, kind release.Kind, expected int64) result.Status {
	ctx := context.Background()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return result.Failed
	}
	defer tx.Rollback(ctx)
	f, err := lockCase(ctx, tx, caseID)
	if err != nil {
		return result.Failed
	}
	if f.returning {
		return result.Conflict
	}
	g := release.Evaluate(release.Input{Kind: kind, ActorUserID: actor, ActorVerified: true, HolderUserID: f.holder, ExpectedVersion: expected,
		CurrentVersion: f.version, ActiveClaim: f.open != nil, Dependencies: f.deps, Lineage: f.lineage})
	if !g.PreconditionsMet {
		return g.Status
	}
	h.pause()
	if _, err := tx.Exec(ctx, `UPDATE syn_claim.work_case SET return_state = 'IN_PROGRESS', version = version + 1 WHERE case_id = $1`, caseID); err != nil {
		return pgStatus(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO syn_claim.history (case_id, event, actor) VALUES ($1, 'RETURN_RESERVED', $2)`, caseID, actor); err != nil {
		return pgStatus(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return pgStatus(err)
	}
	return result.Success
}

func (h *r4h) version(caseID string) int64 {
	return int64(h.count(`SELECT version FROM syn_claim.work_case WHERE case_id = $1`, caseID))
}

// invariants hold after every scenario: no orphan invoice, at most one
// invoice per claim, never an open claim and a return at the same time,
// attribution untouched.
func (h *r4h) invariants() {
	h.t.Helper()
	if n := h.count(`SELECT count(*) FROM syn_invoice.invoice i LEFT JOIN syn_claim.claim c USING (claim_id) WHERE c.claim_id IS NULL`); n != 0 {
		h.t.Fatalf("%d orphan invoices", n)
	}
	if n := h.count(`SELECT count(*) FROM syn_claim.work_case w WHERE w.return_state IS NOT NULL AND EXISTS (SELECT 1 FROM syn_claim.claim c
		WHERE c.case_id = w.case_id AND c.state IN ('ACTIVE', 'RECONCILIATION_REQUIRED', 'CONFIRMED'))`); n != 0 {
		h.t.Fatalf("%d cases with both a return and an invoice claim", n)
	}
	if n := h.count(`SELECT count(*) FROM syn_claim.work_case WHERE original_seller <> 'seller-orig' OR credit_owner <> 'seller-orig'`); n != 0 {
		h.t.Fatalf("%d cases lost their original attribution", n)
	}
}

// race runs fns concurrently from a common start.
func race(fns ...func()) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, fn := range fns {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; fn() }()
	}
	close(start)
	wg.Wait()
}

func TestR4SalesRaceHarness(t *testing.T) {
	h := newR4(t)
	const A, B = "seller-a", "seller-b"
	irr := money.IRR

	t.Run("01 simultaneous invoice issuance", func(t *testing.T) {
		c := h.seedCase(A)
		st, k := h.claim(c, A, true, 1, "c01")
		if st != result.Success {
			t.Fatalf("claim: %s", st)
		}
		h.afterEval = func() { time.Sleep(50 * time.Millisecond) }
		defer func() { h.afterEval = nil }()
		var mu sync.Mutex
		got := map[result.Status]int{}
		var fns []func()
		for i := range 10 {
			fns = append(fns, func() {
				s, _ := h.issue(c, k, A, fmt.Sprintf("i01-%d", i), fmt.Sprintf("h%d", i), 900, irr, false)
				mu.Lock()
				got[s]++
				mu.Unlock()
			})
		}
		race(fns...)
		if got[result.Success] != 1 || got[result.Conflict] != 9 || h.count(`SELECT count(*) FROM syn_invoice.invoice WHERE claim_id = $1`, k) != 1 {
			t.Fatalf("simultaneous issuance: %v", got)
		}
	})

	t.Run("02 issuance racing case return", func(t *testing.T) {
		h.afterEval = func() { time.Sleep(10 * time.Millisecond) }
		defer func() { h.afterEval = nil }()
		for range 25 {
			c := h.seedCase(A)
			var cs, rs result.Status
			race(func() { cs, _ = h.claim(c, A, true, 1, h.id("k")) }, func() { rs = h.beginReturn(c, A, release.KindReturn, 1) })
			if (cs == result.Success) == (rs == result.Success) {
				t.Fatalf("exactly one of claim (%s) and return (%s) must win", cs, rs)
			}
		}
		// Claim first: the return then conflicts with the in-flight issuance.
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c02")
		var is, rs result.Status
		race(func() { is, _ = h.issue(c, k, A, "i02", "h", 500, irr, false) }, func() { rs = h.beginReturn(c, A, release.KindReturn, 2) })
		if is != result.Success || rs != result.Conflict {
			t.Fatalf("issue %s / return %s", is, rs)
		}
	})

	t.Run("03 two sellers claiming the same case", func(t *testing.T) {
		h.afterEval = func() { time.Sleep(10 * time.Millisecond) }
		defer func() { h.afterEval = nil }()
		c := h.seedCase(A)
		var sa, sb result.Status
		race(func() { sa, _ = h.claim(c, A, true, 1, "c03a") }, func() { sb, _ = h.claim(c, B, true, 1, "c03b") })
		if sa != result.Success || sb != result.Unauthorized {
			t.Fatalf("holder %s / other seller %s", sa, sb)
		}
		// Same holder, two concurrent claims: the case row lock serialises
		// them and the loser sees the open claim / new version.
		c2 := h.seedCase(A)
		var x, y result.Status
		race(func() { x, _ = h.claim(c2, A, true, 1, "c03c") }, func() { y, _ = h.claim(c2, A, true, 1, "c03d") })
		if !(x == result.Success && y == result.Conflict || y == result.Success && x == result.Conflict) {
			t.Fatalf("double claim: %s / %s", x, y)
		}
		if h.count(`SELECT count(*) FROM syn_claim.claim WHERE case_id = $1`, c2) != 1 {
			t.Fatal("two claims recorded")
		}
	})

	t.Run("04 duplicate command replay", func(t *testing.T) {
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c04")
		s1, inv1 := h.issue(c, k, A, "i04", "same", 700, irr, false)
		s2, inv2 := h.issue(c, k, A, "i04", "same", 700, irr, false)
		if s1 != result.Success || s2 != result.Success || inv1 != inv2 || h.count(`SELECT count(*) FROM syn_invoice.invoice WHERE claim_id = $1`, k) != 1 {
			t.Fatalf("replay: %s %s %s %s", s1, inv1, s2, inv2)
		}
	})

	t.Run("05 reused idempotency key with a different payload", func(t *testing.T) {
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c05")
		h.issue(c, k, A, "i05", "payload-1", 700, irr, false)
		if s, _ := h.issue(c, k, A, "i05", "payload-2", 800, irr, false); s != result.Conflict {
			t.Fatalf("key reuse: %s", s)
		}
		if h.count(`SELECT count(*) FROM syn_invoice.invoice WHERE claim_id = $1 AND amount = 700`, k) != 1 {
			t.Fatal("the original invoice must be the only one")
		}
	})

	t.Run("06 stale aggregate version", func(t *testing.T) {
		c := h.seedCase(A)
		if s, _ := h.claim(c, A, true, 0, "c06"); s != result.Conflict {
			t.Fatalf("stale claim: %s", s)
		}
		if s := h.beginReturn(c, A, release.KindReturn, 7); s != result.Conflict {
			t.Fatalf("stale return: %s", s)
		}
	})

	t.Run("07 ambiguous case / source links", func(t *testing.T) {
		amb := h.seedCase(A, casesapi.DependencyLink{Source: casesapi.LinkLegacyInvoiceLead, InvoiceRef: "x", Resolution: casesapi.LinkUnresolved,
			Class: casesapi.ClassUnknown})
		con := h.seedCase(A, casesapi.DependencyLink{Source: casesapi.LinkFlowEventRef, InvoiceRef: "y", Resolution: casesapi.LinkConflicting,
			Class: casesapi.ClassCommitted})
		if s, _ := h.claim(amb, A, true, 1, "c07a"); s != result.Unknown {
			t.Fatalf("ambiguous: %s", s)
		}
		if s, _ := h.claim(con, A, true, 1, "c07b"); s != result.Conflict {
			t.Fatalf("conflicting: %s", s)
		}
		if h.count(`SELECT count(*) FROM syn_claim.claim WHERE case_id IN ($1, $2)`, amb, con) != 0 {
			t.Fatal("nothing may be written for ambiguous lineage")
		}
	})

	t.Run("08 missing invoice dependency", func(t *testing.T) {
		c := h.seedCase(A, casesapi.DependencyLink{Source: casesapi.LinkFollowInvoice, InvoiceRef: "gone", Resolution: casesapi.LinkMissingTarget,
			Class: casesapi.ClassUnknown})
		if s := h.beginReturn(c, A, release.KindReturn, 1); s != result.Unknown {
			t.Fatalf("missing target: %s", s)
		}
	})

	t.Run("09 paid invoice return attempt", func(t *testing.T) {
		c := h.seedCase(A, casesapi.DependencyLink{Source: casesapi.LinkLegacyInvoiceLead, InvoiceRef: "paid", Resolution: casesapi.LinkResolved,
			Class: casesapi.ClassPaidOrProtected})
		if s := h.beginReturn(c, A, release.KindReturn, 1); s != result.Blocked {
			t.Fatalf("paid return: %s", s)
		}
		if h.count(`SELECT count(*) FROM syn_hold.assignment WHERE case_id = $1 AND holder = $2`, c, A) != 1 {
			t.Fatal("holder must be unchanged")
		}
	})

	t.Run("10 cancelled invoice release attempt", func(t *testing.T) {
		c := h.seedCase(A, casesapi.DependencyLink{Source: casesapi.LinkLegacyInvoiceLead, InvoiceRef: "cxl", Resolution: casesapi.LinkResolved,
			Class: casesapi.ClassCancelled})
		for _, k := range []release.Kind{release.KindRelease, release.KindReset, release.KindReturn} {
			if s := h.beginReturn(c, A, k, 1); s != result.Blocked {
				t.Fatalf("cancelled %s: %s", k, s)
			}
		}
		if h.version(c) != 1 {
			t.Fatal("nothing may change")
		}
	})

	t.Run("11 interrupted transaction", func(t *testing.T) {
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c11")
		if s, _ := h.issue(c, k, A, "i11", "h", 600, irr, true); s != result.Failed {
			t.Fatalf("interrupted: %s", s)
		}
		if h.count(`SELECT count(*) FROM syn_invoice.invoice WHERE claim_id = $1`, k) != 0 ||
			h.count(`SELECT count(*) FROM syn_invoice.idem WHERE key = 'i11'`) != 0 ||
			h.count(`SELECT count(*) FROM syn_claim.claim WHERE claim_id = $1 AND state = 'ACTIVE'`, k) != 1 {
			t.Fatal("an interrupted issuance must leave no trace and the claim active")
		}
		if s, _ := h.issue(c, k, A, "i11", "h", 600, irr, false); s != result.Success {
			t.Fatalf("retry after interruption: %s", s)
		}
	})

	t.Run("12 restart and replay", func(t *testing.T) {
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c12")
		h.issue(c, k, A, "i12", "h", 600, irr, false)
		h.relay(true) // crash after the cases commit
		h.relay(false)
		h.relay(false)
		if h.count(`SELECT count(*) FROM syn_claim.history WHERE case_id = $1 AND event = 'CONFIRMED'`, c) != 1 ||
			h.count(`SELECT count(*) FROM syn_claim.claim WHERE claim_id = $1 AND state = 'CONFIRMED'`, k) != 1 {
			t.Fatal("replay must confirm exactly once")
		}
		// The confirmed (unpaid) invoice now protects the case.
		if s := h.beginReturn(c, A, release.KindReturn, h.version(c)); s != result.Blocked {
			t.Fatalf("return after issuance: %s", s)
		}
	})

	t.Run("13 historic attribution preservation", func(t *testing.T) {
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c13")
		h.issue(c, k, A, "i13", "h", 600, irr, false)
		h.relay(false)
		if h.count(`SELECT count(*) FROM syn_invoice.invoice WHERE claim_id = $1 AND original_seller = 'seller-orig'`, k) != 1 {
			t.Fatal("the invoice must carry the original seller")
		}
		for _, sql := range []string{`UPDATE syn_claim.history SET actor = 'x' WHERE case_id = $1`, `DELETE FROM syn_claim.history WHERE case_id = $1`} {
			if _, err := h.pool.Exec(context.Background(), sql, c); err == nil || !strings.Contains(err.Error(), "append-only") {
				t.Fatalf("history rewrite must be refused: %v", err)
			}
		}
	})

	t.Run("14 incorrect actor scope", func(t *testing.T) {
		c := h.seedCase(A)
		if s, _ := h.claim(c, A, false, 1, "c14a"); s != result.Unauthorized {
			t.Fatalf("unverified actor: %s", s)
		}
		_, k := h.claim(c, A, true, 1, "c14b")
		if s, _ := h.issue(c, k, B, "i14", "h", 600, irr, false); s != result.Unauthorized {
			t.Fatalf("other seller issuing: %s", s)
		}
		if s := h.beginReturn(c, B, release.KindReturn, h.version(c)); s != result.Unauthorized {
			t.Fatalf("other seller returning: %s", s)
		}
	})

	t.Run("15 cross-module consistency under failure", func(t *testing.T) {
		// Billing never commits: the claim expires into reconciliation, is
		// never released, and still protects the case.
		c := h.seedCase(A)
		_, k := h.claim(c, A, true, 1, "c15a")
		h.exec(`UPDATE syn_claim.claim SET expires_at = now() - interval '1 second' WHERE claim_id = $1`, k)
		h.expire()
		if h.count(`SELECT count(*) FROM syn_claim.claim WHERE claim_id = $1 AND state = 'RECONCILIATION_REQUIRED'`, k) != 1 {
			t.Fatal("expiry must go to reconciliation")
		}
		if s := h.beginReturn(c, A, release.KindReturn, h.version(c)); s != result.Conflict {
			t.Fatalf("return during reconciliation: %s", s)
		}
		if s, _ := h.issue(c, k, A, "i15a", "h", 600, irr, false); s != result.Conflict {
			t.Fatalf("issuance on an expired claim: %s", s)
		}
		// Billing committed, then the claim expired before delivery: the
		// late confirmation records the invoice and keeps reconciliation.
		c2 := h.seedCase(A)
		_, k2 := h.claim(c2, A, true, 1, "c15b")
		if s, _ := h.issue(c2, k2, A, "i15b", "h", 600, irr, false); s != result.Success {
			t.Fatalf("issue: %s", s)
		}
		h.exec(`UPDATE syn_claim.claim SET expires_at = now() - interval '1 second' WHERE claim_id = $1`, k2)
		h.expire()
		h.relay(false)
		if h.count(`SELECT count(*) FROM syn_claim.claim WHERE claim_id = $1 AND state = 'RECONCILIATION_REQUIRED' AND invoice_id IS NOT NULL`, k2) != 1 ||
			h.count(`SELECT count(*) FROM syn_claim.history WHERE case_id = $1 AND event = 'LATE_CONFIRMATION'`, c2) != 1 {
			t.Fatal("a late confirmation must stay in reconciliation with the invoice recorded")
		}
	})

	h.invariants()
}
