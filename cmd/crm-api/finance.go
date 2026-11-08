package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance"
	financehttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/http"
	financequery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Slice 9 finance SHADOW ledger read surface (optional; needs the flag
// store). GET only: the API never derives, posts, approves, settles,
// refunds, credits or debits anything; the shadow rebuild is the offline
// crm-finance-shadow job.
const (
	EnvFinanceReadURL = "CRM_FINANCE_READ_DATABASE_URL"
	// FinanceFlag is the Slice 9 rollback switch (default off).
	FinanceFlag = "finance.shadow_ledger_read"
)

// ErrFinanceConfig reports a partial Slice 9 configuration.
var ErrFinanceConfig = errors.New("crm-api: " + EnvFinanceReadURL + " needs " + EnvFeatureFlagsURL)

// NewFinanceAPI composes the Slice 9 surface over the finance runner and the
// flag store.
func NewFinanceAPI(financeRunner, flags *db.Runner, devAuth bool) (apiserver.FinanceAPI, error) {
	if financeRunner.Module() != finance.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: finance runners bound to the wrong modules")
	}
	subjects := subjects(devAuth)
	guard, err := authorization.NewGuard(flagFunc(flags, FinanceFlag), subjects)
	if err != nil {
		return nil, err
	}
	q, err := financequery.New(financeRunner, clock.System{})
	if err != nil {
		return nil, err
	}
	return financehttp.New(guard, q), nil
}

// openFinance opens the Slice 9 pools when configured.
func openFinance(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.FinanceAPI, func(), error) {
	fin, fu := getenv(EnvFinanceReadURL), getenv(EnvFeatureFlagsURL)
	if fin == "" {
		return nil, func() {}, nil
	}
	if fu == "" {
		return nil, nil, ErrFinanceConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{finance.Module: fin, owner.FeatureFlags: fu} {
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
	api, err := NewFinanceAPI(runners[finance.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}
