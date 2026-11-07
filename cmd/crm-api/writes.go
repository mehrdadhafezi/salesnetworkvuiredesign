package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases"
	casescommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/command"
	caseshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/cases/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody"
	custodyquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/custody/query"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback"
	writebackhttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/integration/writeback/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/hrrequest"
	orghttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/organization/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Slice 11 controlled non-financial write surfaces (optional; each needs its
// owner login(s) and the flag store). The API commits only the owning
// module's aggregate, history and outbox; it never calls a legacy
// connector (integration delivers the outbox offline). Every family is
// behind its own flags, OFF by default; OFF answers BLOCKED and legacy
// remains the only writer.
const (
	EnvCasesWriteURL        = "CRM_CASES_WRITE_DATABASE_URL"
	EnvCustodyHolderURL     = "CRM_CUSTODY_HOLDER_DATABASE_URL"
	EnvOrganizationWriteURL = "CRM_ORGANIZATION_WRITE_DATABASE_URL"
	EnvWritebackReadURL     = "CRM_WRITEBACK_READ_DATABASE_URL"

	// Families (rollout.FromFlags adds _write / _writeback).
	ContactOutcomeFamily = "cases.nonfinancial"
	HRRequestFamily      = "organization.hr_request"
	// WritebackReadFlag gates the diagnostics read.
	WritebackReadFlag = "integration.writeback_read"
)

// ErrWritesConfig reports a partial Slice 11 configuration.
var ErrWritesConfig = errors.New("crm-api: Slice 11 surfaces need their owner logins and " + EnvFeatureFlagsURL +
	" (" + EnvCasesWriteURL + " needs " + EnvCustodyHolderURL + ")")

// holderAdapter maps the custody published current-holder read to the cases
// command port (composition root: neither module imports the other).
type holderAdapter struct{ q *custodyquery.Service }

func (h holderAdapter) Holder(ctx context.Context, caseID string) (casescommand.Holder, error) {
	c, err := h.q.CurrentHolder(ctx, caseID)
	return casescommand.Holder{Known: c.Known, UserID: c.UserID}, err
}

func flagReader(flags *db.Runner) rollout.Flag {
	return func(ctx context.Context, key string) (bool, error) {
		var on bool
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			var err error
			on, err = featureflags.Enabled(ctx, tx, key)
			return err
		})
		return on, err
	}
}

// sessionSubjects is the production subject resolver. Without R1 sessions
// configured no request ever carries a session principal, so it resolves
// every request to the anonymous subject (as Anonymous did); with sessions,
// openAuth sets its grant loader to identity's ACTIVE grants.
var sessionSubjects = authorization.SessionSubjects{}

// subjects returns the request subject resolver. The DEV/TEST stub and
// R1 sessions are mutually exclusive (apiserver refuses both): dev auth
// never falls back to, or coexists with, a production principal.
func subjects(devAuth bool) authorization.SubjectResolver {
	if devAuth {
		return authorization.DevSubjects{}
	}
	return sessionSubjects
}

// Writes are the composed Slice 11 surfaces (nil when not configured).
type Writes struct {
	CaseWrite apiserver.CasesWriteAPI
	HRRequest apiserver.HRRequestAPI
	Writeback apiserver.WritebackAPI
}

// NewCaseWriteAPI composes the contact outcome surface.
func NewCaseWriteAPI(casesRunner, custodyRunner, flags *db.Runner, devAuth bool) (apiserver.CasesWriteAPI, error) {
	if casesRunner.Module() != cases.Module || custodyRunner.Module() != custody.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: contact outcome runners bound to the wrong modules")
	}
	read, err := authorization.NewGuard(flagFunc(flags, SalesFlag), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	write, err := authorization.NewGuard(flagFunc(flags, rollout.WriteFlag(ContactOutcomeFamily)), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	mode, err := rollout.FromFlags(ContactOutcomeFamily, flagReader(flags))
	if err != nil {
		return nil, err
	}
	svc, err := casescommand.New(casesRunner, holderAdapter{custodyquery.New(custodyRunner)}, clock.System{})
	if err != nil {
		return nil, err
	}
	return caseshttp.NewWrite(read, write, mode, svc), nil
}

// NewHRRequestAPI composes the HR change request surface.
func NewHRRequestAPI(orgRunner, flags *db.Runner, devAuth bool) (apiserver.HRRequestAPI, error) {
	if orgRunner.Module() != organization.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: HR request runners bound to the wrong modules")
	}
	guard, err := authorization.NewGuard(flagFunc(flags, rollout.WriteFlag(HRRequestFamily)), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	mode, err := rollout.FromFlags(HRRequestFamily, flagReader(flags))
	if err != nil {
		return nil, err
	}
	svc, err := hrrequest.New(orgRunner, clock.System{})
	if err != nil {
		return nil, err
	}
	return orghttp.NewHRRequests(guard, mode, svc), nil
}

// NewWritebackAPI composes the write-back diagnostics surface.
func NewWritebackAPI(integrationRunner, flags *db.Runner, devAuth bool) (apiserver.WritebackAPI, error) {
	if integrationRunner.Module() != integration.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: write-back runners bound to the wrong modules")
	}
	guard, err := authorization.NewGuard(flagFunc(flags, WritebackReadFlag), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	q, err := writeback.NewQuery(integrationRunner)
	if err != nil {
		return nil, err
	}
	return writebackhttp.New(guard, q), nil
}

// openWrites opens the configured Slice 11 pools.
func openWrites(ctx context.Context, getenv func(string) string, devAuth bool) (Writes, func(), error) {
	cw, ch, ow, wr, fu := getenv(EnvCasesWriteURL), getenv(EnvCustodyHolderURL), getenv(EnvOrganizationWriteURL), getenv(EnvWritebackReadURL), getenv(EnvFeatureFlagsURL)
	var out Writes
	if cw == "" && ch == "" && ow == "" && wr == "" {
		return out, func() {}, nil
	}
	if fu == "" || (cw == "") != (ch == "") {
		return out, nil, ErrWritesConfig
	}
	var pools []*db.Pool
	closeAll := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	open := func(m owner.Module, url string) (*db.Runner, error) {
		cfg := db.DefaultConfig(url)
		cfg.ApplicationName = "crm-api"
		pool, err := db.Open(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("crm-api: %s database: %w", m, err)
		}
		pools = append(pools, pool)
		return pool.Runner(m)
	}
	fail := func(err error) (Writes, func(), error) {
		closeAll()
		return Writes{}, nil, err
	}
	flags, err := open(owner.FeatureFlags, fu)
	if err != nil {
		return fail(err)
	}
	if cw != "" {
		cr, err := open(cases.Module, cw)
		if err != nil {
			return fail(err)
		}
		sr, err := open(custody.Module, ch)
		if err != nil {
			return fail(err)
		}
		if out.CaseWrite, err = NewCaseWriteAPI(cr, sr, flags, devAuth); err != nil {
			return fail(err)
		}
	}
	if ow != "" {
		or, err := open(organization.Module, ow)
		if err != nil {
			return fail(err)
		}
		if out.HRRequest, err = NewHRRequestAPI(or, flags, devAuth); err != nil {
			return fail(err)
		}
	}
	if wr != "" {
		ir, err := open(integration.Module, wr)
		if err != nil {
			return fail(err)
		}
		if out.Writeback, err = NewWritebackAPI(ir, flags, devAuth); err != nil {
			return fail(err)
		}
	}
	return out, closeAll, nil
}
