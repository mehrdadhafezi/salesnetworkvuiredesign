package main

import (
	"context"
	"errors"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	authzhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization/http"
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
