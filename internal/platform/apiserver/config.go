// Package apiserver assembles the HTTP API: the generated strict server
// (internal/platform/httpx/oapi), the httpx foundation middleware and the
// optional DEV/TEST auth stub. Slice 2 serves only technical health and the
// session context; it connects no business module and no database.
package apiserver

import (
	"errors"
	"fmt"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
)

// Environment variables.
const (
	EnvAddr    = "CRM_API_ADDR"
	EnvEnv     = "CRM_ENV"
	EnvDevAuth = "CRM_DEV_AUTH"
)

// ErrInvalidConfig reports an unusable configuration.
var ErrInvalidConfig = errors.New("apiserver: invalid config")

// Config is the server configuration. Timeouts and limits are explicit.
type Config struct {
	Addr              string
	Environment       authstub.Environment
	DevAuth           bool
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
	MaxBodyBytes      int64
}

// DefaultConfig returns safe defaults: loopback, production, no dev auth.
func DefaultConfig() Config {
	return Config{
		Addr:              "127.0.0.1:8080",
		Environment:       authstub.Production,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		MaxHeaderBytes:    64 << 10,
		MaxBodyBytes:      1 << 20,
	}
}

// ConfigFromEnv reads CRM_API_ADDR, CRM_ENV and CRM_DEV_AUTH. CRM_ENV
// defaults to production. CRM_DEV_AUTH must be exactly "1" to request the
// stub, and the request is refused in production.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := DefaultConfig()
	if a := getenv(EnvAddr); a != "" {
		cfg.Addr = a
	}
	env, err := authstub.ParseEnvironment(getenv(EnvEnv))
	if err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	cfg.Environment = env
	switch getenv(EnvDevAuth) {
	case "":
	case "1":
		cfg.DevAuth = true
	default:
		return Config{}, fmt.Errorf("%w: %s must be empty or 1", ErrInvalidConfig, EnvDevAuth)
	}
	return cfg, cfg.Validate()
}

// Validate checks the configuration; dev auth in production is a hard error.
func (c Config) Validate() error {
	if c.DevAuth && c.Environment != authstub.Development && c.Environment != authstub.Test {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, authstub.ErrProductionLockout)
	}
	switch {
	case c.Addr == "":
		return fmt.Errorf("%w: address required", ErrInvalidConfig)
	case c.ReadHeaderTimeout <= 0, c.ReadTimeout <= 0, c.WriteTimeout <= 0, c.IdleTimeout <= 0, c.ShutdownTimeout <= 0:
		return fmt.Errorf("%w: timeouts must be positive", ErrInvalidConfig)
	case c.MaxHeaderBytes <= 0 || c.MaxBodyBytes <= 0:
		return fmt.Errorf("%w: limits must be positive", ErrInvalidConfig)
	}
	return nil
}
