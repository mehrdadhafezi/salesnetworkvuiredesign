package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

var (
	// ErrCommitFailed reports a commit the server rejected: nothing was committed.
	ErrCommitFailed = errors.New("db: commit failed")
	// ErrCommitOutcomeUnknown reports a commit whose outcome cannot be
	// determined (e.g. the connection dropped). Reconcile before any retry.
	ErrCommitOutcomeUnknown = errors.New("db: commit outcome unknown")
	// ErrRoleEscaped reports a transaction whose role no longer matches its owner.
	ErrRoleEscaped = errors.New("db: transaction role changed; refusing to commit")
	// ErrInvalidTxOptions reports unsupported transaction options.
	ErrInvalidTxOptions = errors.New("db: invalid transaction options")
)

// Isolation is a transaction isolation level. The zero value is READ COMMITTED.
type Isolation uint8

const (
	ReadCommitted Isolation = iota
	RepeatableRead
	Serializable
)

// TxOptions are the only per-transaction choices. There is deliberately no
// option to add a second owner.
type TxOptions struct {
	Isolation Isolation
	// ReadOnly runs the transaction READ ONLY; query paths use it so that a
	// read can never perform a business write.
	ReadOnly bool
}

func (o TxOptions) pgx() (pgx.TxOptions, error) {
	out := pgx.TxOptions{AccessMode: pgx.ReadWrite}
	if o.ReadOnly {
		out.AccessMode = pgx.ReadOnly
	}
	switch o.Isolation {
	case ReadCommitted:
		out.IsoLevel = pgx.ReadCommitted
	case RepeatableRead:
		out.IsoLevel = pgx.RepeatableRead
	case Serializable:
		out.IsoLevel = pgx.Serializable
	default:
		return pgx.TxOptions{}, ErrInvalidTxOptions
	}
	return out, nil
}

// Tx is the handle a command or query receives. It belongs to exactly one
// write owner and cannot be committed, rolled back or nested by the callee.
type Tx interface {
	Module() owner.Module
	ReadOnly() bool
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ownedTx struct {
	tx       pgx.Tx
	module   owner.Module
	readOnly bool
}

func (t *ownedTx) Module() owner.Module { return t.module }
func (t *ownedTx) ReadOnly() bool       { return t.readOnly }
func (t *ownedTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}
func (t *ownedTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}
func (t *ownedTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

// Runner runs transactions for one write owner. A Runner never spans modules:
// every transaction it opens assumes only that module's owner role.
type Runner struct {
	pool   *Pool
	module owner.Module
}

// Runner returns a transaction runner bound to module m.
func (p *Pool) Runner(m owner.Module) (*Runner, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if p == nil || p.pool == nil {
		return nil, ErrClosed
	}
	return &Runner{pool: p, module: m}, nil
}

// Module returns the runner's write owner.
func (r *Runner) Module() owner.Module { return r.module }

// InTx runs fn inside one transaction owned by the runner's module.
//
//   - the transaction assumes the module's owner role (SET LOCAL ROLE) and
//     search_path, so PostgreSQL rejects writes to any other schema;
//   - fn returning an error, or panicking, rolls the transaction back;
//   - before commit the role is re-checked; a changed role aborts the commit;
//   - a server-rejected commit returns ErrCommitFailed; an undeterminable
//     commit returns ErrCommitOutcomeUnknown and must be reconciled.
func (r *Runner) InTx(ctx context.Context, opts TxOptions, fn func(ctx context.Context, tx Tx) error) (err error) {
	pgxOpts, err := opts.pgx()
	if err != nil {
		return err
	}
	pool := r.pool.pool
	if pool == nil {
		return ErrClosed
	}
	ptx, err := beginOwned(ctx, pool, pgxOpts, r.module)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		rbErr := ptx.Rollback(context.WithoutCancel(ctx))
		if p := recover(); p != nil {
			panic(p)
		}
		if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("db: rollback: %w", rbErr))
		}
	}()

	if err = fn(ctx, &ownedTx{tx: ptx, module: r.module, readOnly: opts.ReadOnly}); err != nil {
		return err
	}
	var current string
	if err = ptx.QueryRow(ctx, "SELECT current_user").Scan(&current); err != nil {
		return fmt.Errorf("db: verify role: %w", err)
	}
	if current != r.module.Role() {
		return fmt.Errorf("%w: %q, want %q", ErrRoleEscaped, current, r.module.Role())
	}
	if cErr := ptx.Commit(ctx); cErr != nil {
		committed = true // the transaction is finished either way; do not roll back again
		var pgErr *pgconn.PgError
		if errors.As(cErr, &pgErr) || errors.Is(cErr, pgx.ErrTxCommitRollback) {
			return fmt.Errorf("%w: %v", ErrCommitFailed, cErr)
		}
		return fmt.Errorf("%w: %v", ErrCommitOutcomeUnknown, cErr)
	}
	committed = true
	return nil
}

func beginOwned(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, m owner.Module) (pgx.Tx, error) {
	ptx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("db: begin: %w", err)
	}
	role := pgx.Identifier{m.Role()}.Sanitize()
	schema := pgx.Identifier{m.Schema()}.Sanitize()
	if _, err := ptx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
		_ = ptx.Rollback(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("db: assume owner role %s: %w", m.Role(), err)
	}
	if _, err := ptx.Exec(ctx, "SET LOCAL search_path TO "+schema); err != nil {
		_ = ptx.Rollback(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("db: set search_path: %w", err)
	}
	return ptx, nil
}
