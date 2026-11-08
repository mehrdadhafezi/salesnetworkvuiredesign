package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	billingapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/registry"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/legacy/compat"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	paymentscommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/objectstore"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/worker"
)

// Write-back delivery bounds (proposed engineering defaults).
const (
	writebackObserveInterval   = 5 * time.Minute
	writebackObservationBudget = 12
)

// writebackDeliverTypes are the command types the worker may deliver. The
// Slice 12 manual payment write-back is excluded: no financial write-back
// runs without a separately approved contract; those commands stay PENDING
// and visible.
var writebackDeliverTypes = []string{writeback.TypeContactOutcome, writeback.TypeHRCreate, writeback.TypeHRDecide}

// Gate reasons (operator-facing codes).
const (
	ReasonSwitchOff         = "JOB_SWITCH_OFF"
	ReasonFlagUnreadable    = "FLAG_UNREADABLE"
	ReasonWritebackDisabled = "WRITEBACK_ADAPTER_DISABLED"
	ReasonProviderDisabled  = "PROVIDER_ADAPTER_NOT_VERIFIED"
)

// ErrAttestation reports a database that does not attest the declared
// environment: the worker refuses to start.
var ErrAttestation = errors.New("crm-worker: environment attestation failed")

// JobFlag is the runtime switch of a job (default absent = off = paused).
func JobFlag(job string) string { return "worker.job." + job }

// EnvironmentFlag is the attestation flag of an environment. Exactly the
// declared environment's flag must be enabled in the database the worker
// connects to; it is set by the operator who provisions that database.
func EnvironmentFlag(e Environment) string { return "worker.environment." + string(e) }

// overrides are test-only adapters. They are honoured only for a
// development environment that the database attests; the binary never sets
// them (no synthetic adapter is linked into it).
type overrides struct {
	connector writeback.Connector
	port      providerops.Port
	store     objectstore.Store
}

// composition is the built worker.
type composition struct {
	specs  []worker.Spec
	checks []worker.Check
	plan   Plan
}

func flagOn(ctx context.Context, flags *db.Runner, key string) (bool, error) {
	var on bool
	err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
		var err error
		on, err = featureflags.Enabled(ctx, tx, key)
		return err
	})
	return on, err
}

// bind verifies that every owner connection reaches the same database as
// the feature-flag connection that carries the attestation. Without it, a
// module DSN pointing at another database (e.g. a flag database attesting
// development and a payments DSN on staging) would bypass every
// environment gate.
func bind(ctx context.Context, flags *db.Runner, runners map[owner.Module]*db.Runner) error {
	want, err := flags.DatabaseIdentity(ctx)
	if err != nil {
		return fmt.Errorf("%w: database identity unreadable", ErrAttestation)
	}
	for m, r := range runners {
		got, err := r.DatabaseIdentity(ctx)
		if err != nil {
			return fmt.Errorf("%w: %s database identity unreadable", ErrAttestation, m)
		}
		if got != want {
			return fmt.Errorf("%w: the %s login reaches a different database than the attested one", ErrAttestation, m)
		}
	}
	return nil
}

// attest verifies that the database attests exactly the declared
// environment.
func attest(ctx context.Context, flags *db.Runner, env Environment) error {
	for _, e := range []Environment{Development, Staging, Production} {
		on, err := flagOn(ctx, flags, EnvironmentFlag(e))
		if err != nil {
			return fmt.Errorf("%w: attestation unreadable", ErrAttestation)
		}
		if on != (e == env) {
			return fmt.Errorf("%w: declared %s but the database attests %s=%t", ErrAttestation, env, EnvironmentFlag(e), on)
		}
	}
	return nil
}

func switchGate(flags *db.Runner, job string) worker.Gate {
	key := JobFlag(job)
	return func(ctx context.Context) (worker.GateState, string) {
		on, err := flagOn(ctx, flags, key)
		switch {
		case err != nil:
			return worker.Blocked, ReasonFlagUnreadable
		case !on:
			return worker.Paused, ReasonSwitchOff
		}
		return worker.Enabled, ""
	}
}

// refusing is the connector of an intake-only write-back worker: intake
// never calls legacy, and if anything tried, nothing would be sent.
type refusing struct{}

func (refusing) Apply(context.Context, writeback.Intent) (writeback.Applied, error) {
	return writeback.Applied{Outcome: result.Blocked, Reasons: []string{ReasonWritebackDisabled}, Detail: ReasonWritebackDisabled}, nil
}

func (refusing) Observe(context.Context, writeback.Intent) (writeback.Observed, error) {
	return writeback.Observed{}, errors.New(ReasonWritebackDisabled)
}

// noTargets is the billing target reader of the evidence jobs: finalisation
// and expiry never read payment targets, and a call would fail closed.
type noTargets struct{}

var errNoTargets = errors.New("crm-worker: payment targets are not available to evidence jobs")

func (noTargets) PaymentTarget(context.Context, string) (billingapi.PaymentTarget, bool, error) {
	return billingapi.PaymentTarget{}, false, errNoTargets
}

// compose builds the job specs from cfg over opened owner runners.
func compose(ctx context.Context, cfg Config, runners map[owner.Module]*db.Runner, flags *db.Runner, clock timeutil.Clock, log *slog.Logger,
	ov overrides) (composition, error) {
	var c composition
	c.plan = cfg.Plan()
	if err := bind(ctx, flags, runners); err != nil {
		return c, err
	}
	if err := attest(ctx, flags, cfg.Environment); err != nil {
		return c, err
	}
	if (ov.connector != nil || ov.port != nil || ov.store != nil) && cfg.Environment != Development {
		return c, fmt.Errorf("%w: synthetic adapters are allowed only in an attested development environment", ErrConfig)
	}
	spec := func(job worker.Job, gate worker.Gate, reason string) {
		c.specs = append(c.specs, worker.Spec{Job: job, Gate: gate, Interval: cfg.Interval, Timeout: cfg.Timeout, MaxBackoff: cfg.MaxBackoff})
		c.plan.Gates[job.Name()] = reason
	}
	store := func() (objectstore.Store, error) {
		if ov.store != nil {
			return ov.store, nil
		}
		return objectstore.NewFS(cfg.EvidenceRoot)
	}
	var payments *paymentscommand.Service
	paymentsSvc := func() (*paymentscommand.Service, error) {
		if payments != nil {
			return payments, nil
		}
		var st objectstore.Store = refusingStore{}
		if cfg.EvidenceRoot != "" || ov.store != nil {
			var err error
			if st, err = store(); err != nil {
				return nil, err
			}
		}
		var err error
		payments, err = paymentscommand.New(runners["payments"], st, noTargets{}, clock)
		return payments, err
	}
	intakeWorker := func() (*writeback.Worker, error) { return writeback.New(runners["integration"], clock, refusing{}) }
	ops := func() (*providerops.Service, error) { return providerops.New(runners["integration"], clock) }

	for _, name := range cfg.Jobs {
		gate := switchGate(flags, name)
		switch name {
		case JobFulfillmentIntake:
			cat, err := registry.New(runners["catalog"], clock)
			if err != nil {
				return c, err
			}
			p, err := ops()
			if err != nil {
				return c, err
			}
			svc, err := work.New(runners["fulfillment"], cat, p, clock)
			if err != nil {
				return c, err
			}
			job, err := worker.RelayJob(name, runners["biawin"], clock, cfg.Batch, svc.IntakeRoute())
			if err != nil {
				return c, err
			}
			spec(job, gate, ReasonSwitchOff+" until enabled")
		case JobWritebackIntakeCases, JobWritebackIntakeOrg, JobWritebackIntakePayments:
			src := map[string]owner.Module{JobWritebackIntakeCases: "cases", JobWritebackIntakeOrg: "organization", JobWritebackIntakePayments: "payments"}[name]
			w, err := intakeWorker()
			if err != nil {
				return c, err
			}
			route, err := w.IntakeRoute(src)
			if err != nil {
				return c, err
			}
			job, err := worker.RelayJob(name, runners[src], clock, cfg.Batch, route)
			if err != nil {
				return c, err
			}
			spec(job, gate, ReasonSwitchOff+" until enabled")
		case JobWritebackDeliver:
			conn := ov.connector
			if conn == nil && cfg.WritebackAdapter == AdapterCompat {
				// The compatibility client exists, but no contract is
				// approved: construction fails and so does startup.
				if _, err := compat.New(compat.Config{}, compat.Approval{}); err != nil {
					return c, fmt.Errorf("%w: %s=compat: %v", ErrConfig, EnvWritebackAdapter, err)
				}
			}
			if conn == nil {
				w, err := intakeWorker()
				if err != nil {
					return c, err
				}
				spec(w.DeliverJob(name, cfg.Batch), worker.BlockedBy(ReasonWritebackDisabled), ReasonWritebackDisabled)
				continue
			}
			w, err := writeback.New(runners["integration"], clock, conn)
			if err != nil {
				return c, err
			}
			if w, err = w.WithOptions(writeback.Options{Types: writebackDeliverTypes, ObserveInterval: writebackObserveInterval,
				ObservationBudget: writebackObservationBudget}); err != nil {
				return c, err
			}
			spec(w.DeliverJob(name, cfg.Batch), gate, "synthetic adapter (development only)")
		case JobWritebackBacklog:
			w, err := intakeWorker()
			if err != nil {
				return c, err
			}
			spec(w.BacklogJob(name), gate, ReasonSwitchOff+" until enabled")
		case JobReceiptFinalise:
			svc, err := paymentsSvc()
			if err != nil {
				return c, err
			}
			spec(svc.FinaliseJob(name, cfg.Batch), gate, ReasonSwitchOff+" until enabled")
		case JobUploadExpiry:
			svc, err := paymentsSvc()
			if err != nil {
				return c, err
			}
			if cfg.UploadExpiryMode == ExpiryPurge {
				spec(svc.ExpiryPurgeJob(name, cfg.Batch), gate, "purge (development only; D-44 open)")
			} else {
				spec(svc.ExpiryReportJob(name), gate, "report only (purge BLOCKED by D-44)")
			}
		case JobProviderReconcile:
			p, err := ops()
			if err != nil {
				return c, err
			}
			if ov.port == nil {
				spec(p.BacklogJob(name, cfg.Batch), worker.BlockedBy(ReasonProviderDisabled), ReasonProviderDisabled)
				continue
			}
			job, err := p.ReconcileJob(name, ov.port, cfg.Batch)
			if err != nil {
				return c, err
			}
			spec(job, gate, "synthetic adapter (development only)")
		case JobProviderBacklog:
			p, err := ops()
			if err != nil {
				return c, err
			}
			spec(p.BacklogJob(name, cfg.Batch), gate, ReasonSwitchOff+" until enabled")
		default:
			return c, fmt.Errorf("%w: job %s has no composition", ErrConfig, name)
		}
	}
	return c, nil
}

// refusingStore is the object store of a payments service that only
// reports (expiry report mode without an evidence root): every storage call
// fails, nothing is read or deleted.
type refusingStore struct{}

var _ objectstore.Store = refusingStore{}

var errNoStore = errors.New("crm-worker: no evidence store configured")

func (refusingStore) Put(context.Context, string, io.Reader, [32]byte) error       { return errNoStore }
func (refusingStore) CopyIfAbsent(context.Context, string, string, [32]byte) error { return errNoStore }
func (refusingStore) Delete(context.Context, string) error                         { return errNoStore }
func (refusingStore) Stat(context.Context, string) (objectstore.Info, error) {
	return objectstore.Info{}, errNoStore
}
