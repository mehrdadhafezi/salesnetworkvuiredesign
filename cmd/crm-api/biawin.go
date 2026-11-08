package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin"
	biawinhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/http"
	biawinquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/biawin/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment"
	fulfillmenthttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/http"
	fulfillmentquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/fulfillment/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Slice 10 Biawin / fulfillment READ shadow surface (optional; needs both
// module logins and the flag store). GET only: the API never imports,
// activates, suspends, terminates, calls a provider, ships, assigns or
// completes anything; the import is the offline crm-biawin-shadow job.
const (
	EnvBiawinReadURL      = "CRM_BIAWIN_READ_DATABASE_URL"
	EnvFulfillmentReadURL = "CRM_FULFILLMENT_READ_DATABASE_URL"
	// BiawinFlag is the Slice 10 rollback switch (default off).
	BiawinFlag = "biawin_fulfillment.read_shadow"
)

// ErrBiawinConfig reports a partial Slice 10 configuration.
var ErrBiawinConfig = errors.New("crm-api: " + EnvBiawinReadURL + " and " + EnvFulfillmentReadURL + " need each other and " + EnvFeatureFlagsURL)

// biawinSurface composes the two module handler sets behind one flag.
type (
	biawinHandlers      = biawinhttp.Handlers
	fulfillmentHandlers = fulfillmenthttp.Handlers
	biawinSurface       struct {
		*biawinHandlers
		*fulfillmentHandlers
	}
)

// NewBiawinAPI composes the Slice 10 surface over the biawin and
// fulfillment runners and the flag store.
func NewBiawinAPI(biawinRunner, fulfillmentRunner, flags *db.Runner, devAuth bool) (apiserver.BiawinAPI, error) {
	if biawinRunner.Module() != biawin.Module || fulfillmentRunner.Module() != fulfillment.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: biawin runners bound to the wrong modules")
	}
	subjects := subjects(devAuth)
	guard, err := authorization.NewGuard(flagFunc(flags, BiawinFlag), subjects)
	if err != nil {
		return nil, err
	}
	bq, err := biawinquery.New(biawinRunner, clock.System{})
	if err != nil {
		return nil, err
	}
	fq, err := fulfillmentquery.New(fulfillmentRunner, clock.System{})
	if err != nil {
		return nil, err
	}
	return biawinSurface{biawinhttp.New(guard, bq), fulfillmenthttp.New(guard, fq)}, nil
}

// openBiawin opens the Slice 10 pools when configured.
func openBiawin(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.BiawinAPI, func(), error) {
	bu, fu, ffu := getenv(EnvBiawinReadURL), getenv(EnvFulfillmentReadURL), getenv(EnvFeatureFlagsURL)
	if bu == "" && fu == "" {
		return nil, func() {}, nil
	}
	if bu == "" || fu == "" || ffu == "" {
		return nil, nil, ErrBiawinConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{biawin.Module: bu, fulfillment.Module: fu, owner.FeatureFlags: ffu} {
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
	api, err := NewBiawinAPI(runners[biawin.Module], runners[fulfillment.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}
