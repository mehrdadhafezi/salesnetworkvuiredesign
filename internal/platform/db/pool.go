package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrClosed reports use of a closed pool.
var ErrClosed = errors.New("db: pool is closed")

// Pool is an explicitly opened, explicitly closed connection pool.
type Pool struct {
	pool *pgxpool.Pool
}

// Open validates cfg, creates a bounded pool and verifies connectivity.
func Open(ctx context.Context, cfg Config) (*Pool, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	pc, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	pc.MaxConns = cfg.MaxConns
	pc.MinConns = cfg.MinConns
	pc.MaxConnLifetime = cfg.MaxConnLifetime
	pc.MaxConnIdleTime = cfg.MaxConnIdleTime
	pc.HealthCheckPeriod = cfg.HealthCheckPeriod
	pc.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	pc.ConnConfig.RuntimeParams["application_name"] = cfg.ApplicationName
	// All persisted instants are UTC (TAD (E0) #14); the session never
	// inherits a server or machine-local zone.
	pc.ConnConfig.RuntimeParams["timezone"] = "UTC"

	p, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("db: open pool: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return &Pool{pool: p}, nil
}

// Ping verifies that a connection can be acquired and used.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return ErrClosed
	}
	return p.pool.Ping(ctx)
}

// Close releases every connection. It is safe to call more than once.
func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
	p.pool = nil
}
