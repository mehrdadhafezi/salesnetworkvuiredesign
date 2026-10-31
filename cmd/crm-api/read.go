package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customershttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/http"
	customersquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	organizationquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
)

// Slice 4 read-surface configuration. All three DSNs or none: without them
// the HR read surface answers BLOCKED / FEATURE_DISABLED.
const (
	EnvIdentityReadURL     = "CRM_IDENTITY_READ_DATABASE_URL"
	EnvOrganizationReadURL = "CRM_ORGANIZATION_READ_DATABASE_URL"
	EnvFeatureFlagsURL     = "CRM_FEATUREFLAGS_DATABASE_URL"
)

// HRReadFlag is the Slice 4 rollback switch (platform feature flags,
// default off). Off: every HR read answers BLOCKED / FEATURE_DISABLED and
// the SPA hides the HR navigation.
const HRReadFlag = "organization.hr_read_shadow"

// ErrReadConfig reports a partial Slice 4 configuration.
var ErrReadConfig = errors.New("crm-api: Slice 4 read surface needs all of " + EnvIdentityReadURL + ", " + EnvOrganizationReadURL + ", " + EnvFeatureFlagsURL)

// Aliases name the embedded module handlers (all are called Handlers).
type (
	authorizationHandlers = authzhttp.Handlers
	identityHandlers      = identityhttp.Handlers
	organizationHandlers  = organizationhttp.Handlers
)

// readAPI composes the module read handlers into apiserver.ReadAPI.
type readAPI struct {
	*authorizationHandlers
	*identityHandlers
	*organizationHandlers
	guard *authorization.Guard
}

func (r readAPI) HRReadEnabled(ctx context.Context) bool {
	on, err := r.guard.Enabled(ctx)
	return err == nil && on
}

// NewReadAPI composes the Slice 4 read surface over three module runners.
// devAuth selects the synthetic DEV/TEST subjects; otherwise every request
// is anonymous (there is no production authentication in Slice 4).
func NewReadAPI(identityRunner, organizationRunner, flags *db.Runner, devAuth bool) (apiserver.ReadAPI, error) {
	if identityRunner.Module() != identity.Module || organizationRunner.Module() != organization.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: read runners bound to the wrong modules")
	}
	enabled := func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, HRReadFlag)
			return err
		})
		return on, err
	}
	var subjects authorization.SubjectResolver = authorization.Anonymous{}
	if devAuth {
		subjects = authorization.DevSubjects{}
	}
	guard, err := authorization.NewGuard(enabled, subjects)
	if err != nil {
		return nil, err
	}
	return readAPI{
		authorizationHandlers: authzhttp.New(guard),
		identityHandlers:      identityhttp.New(guard, identityquery.New(identityRunner)),
		organizationHandlers:  organizationhttp.New(guard, organizationquery.New(organizationRunner)),
		guard:                 guard,
	}, nil
}

// Slice 5 customer identity read surface (optional; needs the flag store).
const (
	EnvCustomersReadURL = "CRM_CUSTOMERS_READ_DATABASE_URL"
	// CustomersFlag is the Slice 5 rollback switch (default off).
	CustomersFlag = "customers.identity_shadow_read"
)

// ErrCustomersConfig reports a customers DSN without the flag store DSN.
var ErrCustomersConfig = errors.New("crm-api: " + EnvCustomersReadURL + " needs " + EnvFeatureFlagsURL)

func flagFunc(flags *db.Runner, key string) authorization.FlagFunc {
	return func(ctx context.Context) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, key)
			return err
		})
		return on, err
	}
}

// NewCustomersAPI composes the Slice 5 surface over a customers runner and
// the flag store. devAuth selects synthetic DEV/TEST subjects.
func NewCustomersAPI(customersRunner, flags *db.Runner, devAuth bool) (apiserver.CustomersAPI, error) {
	if customersRunner.Module() != customers.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: customers runners bound to the wrong modules")
	}
	var subjects authorization.SubjectResolver = authorization.Anonymous{}
	if devAuth {
		subjects = authorization.DevSubjects{}
	}
	guard, err := authorization.NewGuard(flagFunc(flags, CustomersFlag), subjects)
	if err != nil {
		return nil, err
	}
	return customershttp.New(guard, customersquery.New(customersRunner)), nil
}

// openCustomers opens the Slice 5 pools when configured.
func openCustomers(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.CustomersAPI, func(), error) {
	cu, fu := getenv(EnvCustomersReadURL), getenv(EnvFeatureFlagsURL)
	if cu == "" {
		return nil, func() {}, nil
	}
	if fu == "" {
		return nil, nil, ErrCustomersConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{customers.Module: cu, owner.FeatureFlags: fu} {
		cfg := db.DefaultConfig(u)
		cfg.ApplicationName = "crm-api"
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("crm-api: %s read database: %w", m, err)
		}
		pools = append(pools, pool)
		if runners[m], err = pool.Runner(m); err != nil {
			closeAll()
			return nil, nil, err
		}
	}
	api, err := NewCustomersAPI(runners[customers.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}
