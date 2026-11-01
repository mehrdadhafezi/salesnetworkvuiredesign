package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billinghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/http"
	billingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	casesquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	custodyhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/http"
	custodyquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers"
	customershttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/http"
	customersquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/customers/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity"
	identityhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/http"
	identityquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	organizationhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	organizationquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	paymentsquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/query"
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

// Slice 6 sales read surface (optional, all or none; needs the flag store).
const (
	EnvCasesReadURL   = "CRM_CASES_READ_DATABASE_URL"
	EnvCustodyReadURL = "CRM_CUSTODY_READ_DATABASE_URL"
	// SalesFlag is the Slice 6 rollback switch (default off).
	SalesFlag = "sales.read_shadow"
)

// ErrSalesConfig reports a partial Slice 6 configuration.
var ErrSalesConfig = errors.New("crm-api: " + EnvCasesReadURL + " and " + EnvCustodyReadURL + " are all-or-none and need " + EnvFeatureFlagsURL)

// salesAPI composes the cases and custody read handlers. Each module keeps
// its own runner; there is no cross-module transaction.
type (
	casesHandlers   = caseshttp.Handlers
	custodyHandlers = custodyhttp.Handlers
)

type salesAPI struct {
	*casesHandlers
	*custodyHandlers
}

// NewSalesAPI composes the Slice 6 surface over the cases and custody
// runners and the flag store. devAuth selects synthetic DEV/TEST subjects.
func NewSalesAPI(casesRunner, custodyRunner, flags *db.Runner, devAuth bool) (apiserver.SalesAPI, error) {
	if casesRunner.Module() != cases.Module || custodyRunner.Module() != custody.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: sales runners bound to the wrong modules")
	}
	var subjects authorization.SubjectResolver = authorization.Anonymous{}
	if devAuth {
		subjects = authorization.DevSubjects{}
	}
	guard, err := authorization.NewGuard(flagFunc(flags, SalesFlag), subjects)
	if err != nil {
		return nil, err
	}
	return salesAPI{caseshttp.New(guard, casesquery.New(casesRunner)), custodyhttp.New(guard, custodyquery.New(custodyRunner))}, nil
}

// openSales opens the Slice 6 pools when configured.
func openSales(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.SalesAPI, func(), error) {
	cu, su, fu := getenv(EnvCasesReadURL), getenv(EnvCustodyReadURL), getenv(EnvFeatureFlagsURL)
	if cu == "" && su == "" {
		return nil, func() {}, nil
	}
	if cu == "" || su == "" || fu == "" {
		return nil, nil, ErrSalesConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{cases.Module: cu, custody.Module: su, owner.FeatureFlags: fu} {
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
	api, err := NewSalesAPI(runners[cases.Module], runners[custody.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}

// Slice 7 billing + payments read surface (optional, all or none; needs the
// flag store).
const (
	EnvBillingReadURL  = "CRM_BILLING_READ_DATABASE_URL"
	EnvPaymentsReadURL = "CRM_PAYMENTS_READ_DATABASE_URL"
	// BillingFlag is the Slice 7 rollback switch (default off).
	BillingFlag = "billing_payments.read_shadow"
)

// ErrBillingConfig reports a partial Slice 7 configuration.
var ErrBillingConfig = errors.New("crm-api: " + EnvBillingReadURL + " and " + EnvPaymentsReadURL + " are all-or-none and need " + EnvFeatureFlagsURL)

// billingAPI composes the billing and payments read handlers. Each module
// keeps its own runner; there is no cross-module transaction. Billing reads
// payment sums through the published payments read API only.
type (
	billingHandlers  = billinghttp.Handlers
	paymentsHandlers = paymentshttp.Handlers
)

type billingAPI struct {
	*billingHandlers
	*paymentsHandlers
}

// NewBillingAPI composes the Slice 7 surface over the billing and payments
// runners and the flag store. devAuth selects synthetic DEV/TEST subjects.
func NewBillingAPI(billingRunner, paymentsRunner, flags *db.Runner, devAuth bool) (apiserver.BillingAPI, error) {
	if billingRunner.Module() != billing.Module || paymentsRunner.Module() != payments.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: billing runners bound to the wrong modules")
	}
	var subjects authorization.SubjectResolver = authorization.Anonymous{}
	if devAuth {
		subjects = authorization.DevSubjects{}
	}
	guard, err := authorization.NewGuard(flagFunc(flags, BillingFlag), subjects)
	if err != nil {
		return nil, err
	}
	pq := paymentsquery.New(paymentsRunner)
	return billingAPI{billinghttp.New(guard, billingquery.New(billingRunner), pq), paymentshttp.New(guard, pq)}, nil
}

// openBilling opens the Slice 7 pools when configured.
func openBilling(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.BillingAPI, func(), error) {
	bu, pu, fu := getenv(EnvBillingReadURL), getenv(EnvPaymentsReadURL), getenv(EnvFeatureFlagsURL)
	if bu == "" && pu == "" {
		return nil, func() {}, nil
	}
	if bu == "" || pu == "" || fu == "" {
		return nil, nil, ErrBillingConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	runners := map[owner.Module]*db.Runner{}
	for m, u := range map[owner.Module]string{billing.Module: bu, payments.Module: pu, owner.FeatureFlags: fu} {
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
	api, err := NewBillingAPI(runners[billing.Module], runners[payments.Module], runners[owner.FeatureFlags], devAuth)
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return api, closeAll, nil
}
