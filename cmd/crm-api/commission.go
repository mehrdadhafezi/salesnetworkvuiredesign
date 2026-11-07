package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission"
	commissionhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/http"
	commissionquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/commission/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Slice 13 commission controlled SHADOW read surface (optional; needs the
// commission login and the flag store). GET only: the API never calculates,
// recalculates, imports, approves, posts, credits, settles or reverses a
// commission; the shadow run is the offline crm-commission-shadow job.
const (
	EnvCommissionReadURL = "CRM_COMMISSION_READ_DATABASE_URL"
	// CommissionFlag is the Slice 13 rollback switch (default off).
	CommissionFlag = "commission.read_shadow"
)

// ErrCommissionConfig reports a partial Slice 13 configuration.
var ErrCommissionConfig = errors.New("crm-api: " + EnvCommissionReadURL + " needs " + EnvFeatureFlagsURL)

// NewCommissionAPI composes the Slice 13 surface over the commission runner
// and the flag store.
func NewCommissionAPI(commissionRunner, flags *db.Runner, devAuth bool) (apiserver.CommissionAPI, error) {
	if commissionRunner.Module() != commission.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: commission runners bound to the wrong modules")
	}
	subjects := subjects(devAuth)
	guard, err := authorization.NewGuard(flagFunc(flags, CommissionFlag), subjects)
	if err != nil {
		return nil, err
	}
	q, err := commissionquery.New(commissionRunner)
	if err != nil {
		return nil, err
	}
	return commissionhttp.New(guard, q), nil
}

// openCommission opens the Slice 13 pools when configured.
func openCommission(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.CommissionAPI, func(), error) {
	cu, ffu := getenv(EnvCommissionReadURL), getenv(EnvFeatureFlagsURL)
	if cu == "" {
		return nil, func() {}, nil
	}
	if ffu == "" {
		return nil, nil, ErrCommissionConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{commission.Module: cu, owner.FeatureFlags: ffu} {
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
	api, err := NewCommissionAPI(runners[commission.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}
