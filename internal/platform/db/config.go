// Package db is the PostgreSQL connection and transaction foundation.
//
// There is no global pool: callers Open a Pool explicitly, pass it where it
// is needed and Close it on shutdown. The raw pgx pool is never exported.
// Business code reaches the database only through a Runner bound to one
// write owner (owner.Module); see tx.go.
package db

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// EnvDatabaseURL is the environment variable holding the application DSN.
const EnvDatabaseURL = "CRM_DATABASE_URL"

// ErrInvalidConfig reports a missing or out-of-bounds setting.
var ErrInvalidConfig = errors.New("db: invalid config")

// Config is the minimal database configuration contract.
type Config struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
	ApplicationName   string
}

// Bounds applied to pool settings.
const (
	maxAllowedConns = 100
	defaultMaxConns = 10
)

// DefaultConfig returns bounded defaults for the given DSN.
func DefaultConfig(url string) Config {
	return Config{
		URL:               url,
		MaxConns:          defaultMaxConns,
		MinConns:          0,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   15 * time.Minute,
		HealthCheckPeriod: time.Minute,
		ConnectTimeout:    10 * time.Second,
		ApplicationName:   "crm",
	}
}

// ConfigFromEnv builds a Config from lookup (normally os.LookupEnv). Only
// the DSN comes from the environment; pool bounds use DefaultConfig.
func ConfigFromEnv(lookup func(string) (string, bool)) (Config, error) {
	url, ok := lookup(EnvDatabaseURL)
	if !ok || strings.TrimSpace(url) == "" {
		return Config{}, fmt.Errorf("%w: %s is not set", ErrInvalidConfig, EnvDatabaseURL)
	}
	cfg := DefaultConfig(url)
	return cfg, cfg.Validate()
}

// Validate checks that every setting is present and bounded.
func (c Config) Validate() error {
	switch {
	case strings.TrimSpace(c.URL) == "":
		return fmt.Errorf("%w: URL is required", ErrInvalidConfig)
	case c.MaxConns < 1 || c.MaxConns > maxAllowedConns:
		return fmt.Errorf("%w: MaxConns must be 1..%d", ErrInvalidConfig, maxAllowedConns)
	case c.MinConns < 0 || c.MinConns > c.MaxConns:
		return fmt.Errorf("%w: MinConns must be 0..MaxConns", ErrInvalidConfig)
	case c.MaxConnLifetime <= 0, c.MaxConnIdleTime <= 0, c.HealthCheckPeriod <= 0, c.ConnectTimeout <= 0:
		return fmt.Errorf("%w: durations must be positive", ErrInvalidConfig)
	case c.ApplicationName == "":
		return fmt.Errorf("%w: ApplicationName is required", ErrInvalidConfig)
	}
	return nil
}
