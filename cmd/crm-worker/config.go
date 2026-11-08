package main

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Environment variables. Every value is configuration, never a secret,
// except the DSNs, which are never logged or printed.
const (
	EnvEnvironment       = "CRM_WORKER_ENV"
	EnvJobs              = "CRM_WORKER_JOBS"
	EnvAddr              = "CRM_WORKER_ADDR"
	EnvInterval          = "CRM_WORKER_INTERVAL"
	EnvTimeout           = "CRM_WORKER_TIMEOUT"
	EnvMaxBackoff        = "CRM_WORKER_MAX_BACKOFF"
	EnvBatch             = "CRM_WORKER_BATCH"
	EnvMaxParallel       = "CRM_WORKER_MAX_PARALLEL"
	EnvGrace             = "CRM_WORKER_GRACE"
	EnvEvidenceRoot      = "CRM_WORKER_EVIDENCE_ROOT"
	EnvUploadExpiryMode  = "CRM_WORKER_UPLOAD_EXPIRY_MODE"
	EnvWritebackAdapter  = "CRM_WORKER_WRITEBACK_ADAPTER"
	EnvProviderAdapter   = "CRM_WORKER_PROVIDER_ADAPTER"
	EnvFeatureFlagsURL   = "CRM_WORKER_FEATUREFLAGS_DATABASE_URL"
	envModuleURLTemplate = "CRM_WORKER_%s_DATABASE_URL"
)

// Environment is the deliberately declared deployment environment. It is
// never inferred (not from a hostname, not from "localhost"); the database
// must attest the same environment (see attest).
type Environment string

// Environments.
const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// Job names.
const (
	JobFulfillmentIntake       = "fulfillment_intake"
	JobWritebackIntakeCases    = "writeback_intake_cases"
	JobWritebackIntakeOrg      = "writeback_intake_organization"
	JobWritebackIntakePayments = "writeback_intake_payments"
	JobWritebackDeliver        = "writeback_deliver"
	JobWritebackBacklog        = "writeback_backlog"
	JobReceiptFinalise         = "receipt_finalise"
	JobUploadExpiry            = "upload_expiry"
	JobProviderReconcile       = "provider_reconcile"
	JobProviderBacklog         = "provider_backlog"
)

// catalog lists every job and the owner modules it needs a dedicated login
// for (least privilege: one login per owner, never a shared superuser).
var catalog = map[string][]owner.Module{
	JobFulfillmentIntake:       {"biawin", "fulfillment", "catalog", "integration"},
	JobWritebackIntakeCases:    {"cases", "integration"},
	JobWritebackIntakeOrg:      {"organization", "integration"},
	JobWritebackIntakePayments: {"payments", "integration"},
	JobWritebackDeliver:        {"integration"},
	JobWritebackBacklog:        {"integration"},
	JobReceiptFinalise:         {"payments"},
	JobUploadExpiry:            {"payments"},
	JobProviderReconcile:       {"integration"},
	JobProviderBacklog:         {"integration"},
}

// Adapter / mode values.
const (
	AdapterDisabled = "disabled"
	AdapterCompat   = "compat"
	ExpiryReport    = "report"
	ExpiryPurge     = "purge"
)

// Config is the validated worker configuration.
type Config struct {
	Environment      Environment
	Jobs             []string
	Addr             string
	Interval         time.Duration
	Timeout          time.Duration
	MaxBackoff       time.Duration
	Batch            int
	MaxParallel      int
	Grace            time.Duration
	EvidenceRoot     string
	UploadExpiryMode string
	WritebackAdapter string
	ProviderAdapter  string
	FlagsURL         string
	ModuleURLs       map[owner.Module]string
}

// ErrConfig reports an invalid or unsafe configuration: startup fails.
var ErrConfig = errors.New("crm-worker: invalid configuration")

func moduleEnv(m owner.Module) string {
	return fmt.Sprintf(envModuleURLTemplate, strings.ToUpper(string(m)))
}

func duration(get func(string) string, key string, def, lo, hi time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(get(key))
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < lo || d > hi {
		return 0, fmt.Errorf("%w: %s must be a duration in [%s, %s]", ErrConfig, key, lo, hi)
	}
	return d, nil
}

func integer(get func(string) string, key string, def, lo, hi int) (int, error) {
	v := strings.TrimSpace(get(key))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < lo || n > hi {
		return 0, fmt.Errorf("%w: %s must be an integer in [%d, %d]", ErrConfig, key, lo, hi)
	}
	return n, nil
}

// ConfigFromEnv parses and validates the configuration. Defaults are the
// proposed engineering defaults (not approved SLA values). Every external
// adapter defaults to disabled.
func ConfigFromEnv(get func(string) string) (Config, error) {
	var c Config
	switch Environment(strings.TrimSpace(get(EnvEnvironment))) {
	case Development:
		c.Environment = Development
	case Staging:
		c.Environment = Staging
	case Production:
		c.Environment = Production
	default:
		return c, fmt.Errorf("%w: %s must be set explicitly to development, staging or production (never inferred)", ErrConfig, EnvEnvironment)
	}
	seen := map[string]bool{}
	for _, j := range strings.Split(get(EnvJobs), ",") {
		j = strings.TrimSpace(j)
		if j == "" {
			continue
		}
		if _, ok := catalog[j]; !ok {
			return c, fmt.Errorf("%w: unknown job %q", ErrConfig, j)
		}
		if !seen[j] {
			seen[j] = true
			c.Jobs = append(c.Jobs, j)
		}
	}
	if len(c.Jobs) == 0 {
		return c, fmt.Errorf("%w: %s lists no job (every job is off unless named)", ErrConfig, EnvJobs)
	}
	// A blocked external job is always accompanied by its backlog job, so a
	// stopped delivery is visible.
	if seen[JobWritebackDeliver] && !seen[JobWritebackBacklog] {
		c.Jobs = append(c.Jobs, JobWritebackBacklog)
	}
	if seen[JobProviderReconcile] && !seen[JobProviderBacklog] {
		c.Jobs = append(c.Jobs, JobProviderBacklog)
	}
	sort.Strings(c.Jobs)
	c.Addr = strings.TrimSpace(get(EnvAddr))
	if c.Addr == "" {
		c.Addr = "127.0.0.1:9091"
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return c, fmt.Errorf("%w: %s", ErrConfig, EnvAddr)
	}
	var err error
	if c.Interval, err = duration(get, EnvInterval, 30*time.Second, time.Second, time.Hour); err != nil {
		return c, err
	}
	if c.Timeout, err = duration(get, EnvTimeout, 2*time.Minute, time.Second, 30*time.Minute); err != nil {
		return c, err
	}
	if c.MaxBackoff, err = duration(get, EnvMaxBackoff, 10*time.Minute, c.Interval, 24*time.Hour); err != nil {
		return c, err
	}
	if c.Grace, err = duration(get, EnvGrace, 30*time.Second, 0, 10*time.Minute); err != nil {
		return c, err
	}
	if c.Batch, err = integer(get, EnvBatch, 50, 1, 200); err != nil {
		return c, err
	}
	if c.MaxParallel, err = integer(get, EnvMaxParallel, 4, 1, 16); err != nil {
		return c, err
	}
	c.UploadExpiryMode = strings.TrimSpace(get(EnvUploadExpiryMode))
	if c.UploadExpiryMode == "" {
		c.UploadExpiryMode = ExpiryReport
	}
	switch c.UploadExpiryMode {
	case ExpiryReport:
	case ExpiryPurge:
		// Irreversible deletion needs the D-44 retention decision. Only a
		// disposable development database may rehearse it.
		if c.Environment != Development {
			return c, fmt.Errorf("%w: %s=purge is BLOCKED by D-44 (retention not approved) outside development", ErrConfig, EnvUploadExpiryMode)
		}
	default:
		return c, fmt.Errorf("%w: %s must be report or purge", ErrConfig, EnvUploadExpiryMode)
	}
	c.WritebackAdapter = strings.TrimSpace(get(EnvWritebackAdapter))
	if c.WritebackAdapter == "" {
		c.WritebackAdapter = AdapterDisabled
	}
	if c.WritebackAdapter != AdapterDisabled && c.WritebackAdapter != AdapterCompat {
		return c, fmt.Errorf("%w: %s must be disabled or compat", ErrConfig, EnvWritebackAdapter)
	}
	c.ProviderAdapter = strings.TrimSpace(get(EnvProviderAdapter))
	if c.ProviderAdapter == "" {
		c.ProviderAdapter = AdapterDisabled
	}
	if c.ProviderAdapter != AdapterDisabled {
		// No verified provider adapter exists (Biawin protocol not evidenced).
		return c, fmt.Errorf("%w: %s supports only disabled (no verified provider adapter exists)", ErrConfig, EnvProviderAdapter)
	}
	c.FlagsURL = strings.TrimSpace(get(EnvFeatureFlagsURL))
	if c.FlagsURL == "" {
		return c, fmt.Errorf("%w: %s is required (environment attestation and job switches)", ErrConfig, EnvFeatureFlagsURL)
	}
	c.ModuleURLs = map[owner.Module]string{}
	for _, j := range c.Jobs {
		for _, m := range catalog[j] {
			u := strings.TrimSpace(get(moduleEnv(m)))
			if u == "" {
				return c, fmt.Errorf("%w: job %s needs %s (a login dedicated to the %s owner role)", ErrConfig, j, moduleEnv(m), m)
			}
			c.ModuleURLs[m] = u
		}
	}
	for m, u := range c.ModuleURLs {
		if u == c.FlagsURL {
			return c, fmt.Errorf("%w: %s reuses the feature-flag login; every owner needs its own login", ErrConfig, moduleEnv(m))
		}
	}
	if (seen[JobReceiptFinalise] || seen[JobUploadExpiry] && c.UploadExpiryMode == ExpiryPurge) && strings.TrimSpace(get(EnvEvidenceRoot)) == "" {
		return c, fmt.Errorf("%w: %s is required for receipt finalisation / purge", ErrConfig, EnvEvidenceRoot)
	}
	c.EvidenceRoot = strings.TrimSpace(get(EnvEvidenceRoot))
	return c, nil
}

// Plan is the redacted, printable view of a configuration (no DSN).
type Plan struct {
	Environment      string            `json:"environment"`
	Jobs             []string          `json:"jobs"`
	Addr             string            `json:"health_addr"`
	Interval         string            `json:"interval"`
	Timeout          string            `json:"timeout"`
	MaxBackoff       string            `json:"max_backoff"`
	Batch            int               `json:"batch"`
	MaxParallel      int               `json:"max_parallel"`
	UploadExpiryMode string            `json:"upload_expiry_mode"`
	WritebackAdapter string            `json:"writeback_adapter"`
	ProviderAdapter  string            `json:"provider_adapter"`
	Gates            map[string]string `json:"gates"`
	Logins           []string          `json:"owner_logins"`
}

// Plan returns the redacted view; gates are filled by the composition.
func (c Config) Plan() Plan {
	p := Plan{Environment: string(c.Environment), Jobs: c.Jobs, Addr: c.Addr, Interval: c.Interval.String(), Timeout: c.Timeout.String(),
		MaxBackoff: c.MaxBackoff.String(), Batch: c.Batch, MaxParallel: c.MaxParallel, UploadExpiryMode: c.UploadExpiryMode,
		WritebackAdapter: c.WritebackAdapter, ProviderAdapter: c.ProviderAdapter, Gates: map[string]string{}}
	for m := range c.ModuleURLs {
		p.Logins = append(p.Logins, string(m))
	}
	sort.Strings(p.Logins)
	return p
}
