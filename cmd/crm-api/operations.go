package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/activation"
	biawinhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/catalog/registry"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	fulfillmenthttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/work"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops"
	provideropshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/providerops/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Slice 14 controlled fulfillment / Biawin operations surface (optional;
// all four owner logins plus the flag store, or none). Every flag is OFF by
// default and every capability is granted to no role. The API never calls a
// provider (no port is reachable from crm-api), never activates a
// production subscription (D-35), never charges a wallet (D-38) and never
// writes back to legacy; its writes are fulfillment work-case commands on
// target-native SHADOW cases and human reconciliation of provider
// operations.
const (
	EnvOpsFulfillmentURL = "CRM_OPS_FULFILLMENT_DATABASE_URL"
	EnvOpsBiawinURL      = "CRM_OPS_BIAWIN_DATABASE_URL"
	EnvOpsCatalogURL     = "CRM_OPS_CATALOG_DATABASE_URL"
	EnvOpsIntegrationURL = "CRM_OPS_INTEGRATION_DATABASE_URL"

	// OperationsReadFlag gates every Slice 14 read.
	OperationsReadFlag = "fulfillment.work_read"
	// WorkFamily is the work-case command family (flags fulfillment.work_write
	// and fulfillment.work_writeback; write-back is never implemented).
	WorkFamily = "fulfillment.work"
	// ProviderReconcileFlag gates human reconciliation of UNKNOWN operations.
	ProviderReconcileFlag = "integration.provider_reconcile"
)

// ErrOperationsConfig reports a partial Slice 14 configuration.
var ErrOperationsConfig = errors.New("crm-api: " + EnvOpsFulfillmentURL + ", " + EnvOpsBiawinURL + ", " + EnvOpsCatalogURL + " and " +
	EnvOpsIntegrationURL + " need each other and " + EnvFeatureFlagsURL)

type (
	workHandlers       = fulfillmenthttp.WorkHandlers
	activationHandlers = biawinhttp.ActivationHandlers
	providerHandlers   = provideropshttp.Handlers
	operationsSurface  struct {
		*workHandlers
		*activationHandlers
		*providerHandlers
	}
)

// NewOperationsAPI composes the Slice 14 surface.
func NewOperationsAPI(fulfillmentRunner, biawinRunner, catalogRunner, integrationRunner, flags *db.Runner, devAuth bool) (apiserver.OperationsAPI, error) {
	if fulfillmentRunner.Module() != fulfillment.Module || biawinRunner.Module() != biawin.Module || catalogRunner.Module() != catalog.Module ||
		integrationRunner.Module() != integration.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: operations runners bound to the wrong modules")
	}
	read, err := authorization.NewGuard(flagFunc(flags, OperationsReadFlag), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	write, err := authorization.NewGuard(flagFunc(flags, rollout.WriteFlag(WorkFamily)), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	reconcile, err := authorization.NewGuard(flagFunc(flags, ProviderReconcileFlag), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	mode, err := rollout.FromFlags(WorkFamily, flagReader(flags))
	if err != nil {
		return nil, err
	}
	cat, err := registry.New(catalogRunner, clock.System{})
	if err != nil {
		return nil, err
	}
	ops, err := providerops.New(integrationRunner, clock.System{})
	if err != nil {
		return nil, err
	}
	// providerops publishes the confirmed-operation read the work port needs
	// (composition root: fulfillment never imports integration).
	svc, err := work.New(fulfillmentRunner, cat, ops, clock.System{})
	if err != nil {
		return nil, err
	}
	act, err := activation.New(biawinRunner, cat, clock.System{})
	if err != nil {
		return nil, err
	}
	return operationsSurface{fulfillmenthttp.NewWork(read, write, mode, svc, cat), biawinhttp.NewActivation(read, act),
		provideropshttp.New(read, reconcile, ops)}, nil
}

// openOperations opens the Slice 14 pools when configured.
func openOperations(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.OperationsAPI, func(), error) {
	urls := map[owner.Module]string{fulfillment.Module: getenv(EnvOpsFulfillmentURL), biawin.Module: getenv(EnvOpsBiawinURL),
		catalog.Module: getenv(EnvOpsCatalogURL), integration.Module: getenv(EnvOpsIntegrationURL)}
	set := 0
	for _, u := range urls {
		if u != "" {
			set++
		}
	}
	if set == 0 {
		return nil, func() {}, nil
	}
	ffu := getenv(EnvFeatureFlagsURL)
	if set != len(urls) || ffu == "" {
		return nil, nil, ErrOperationsConfig
	}
	urls[owner.FeatureFlags] = ffu
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range urls {
		cfg := db.DefaultConfig(u)
		cfg.ApplicationName = "crm-api"
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("crm-api: %s database: %w", m, err)
		}
		pools = append(pools, pool)
		if runners[m], err = pool.Runner(m); err != nil {
			closeAll()
			return nil, nil, err
		}
	}
	api, err := NewOperationsAPI(runners[fulfillment.Module], runners[biawin.Module], runners[catalog.Module], runners[integration.Module],
		runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}
