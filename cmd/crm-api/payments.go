package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing"
	billingquery "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/billing/query"
	financehttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/finance/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments"
	paymentscommand "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/command"
	paymentshttp "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/payments/http"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/apiserver"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/clock"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/objectstore"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/owner"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// Slice 12 controlled financial write surface (optional). It needs the
// payments owner login, a billing read login (published PaymentTarget read),
// an absolute evidence storage root and the flag store. The API commits
// only payments' own aggregates, history and outbox; it never calls a
// gateway or a legacy connector and never wires the synthetic gateway.
// Every family is OFF by default; OFF answers BLOCKED.
const (
	EnvPaymentsWriteURL   = "CRM_PAYMENTS_WRITE_DATABASE_URL"
	EnvPaymentsBillingURL = "CRM_PAYMENTS_BILLING_READ_DATABASE_URL"
	EnvPaymentsObjectRoot = "CRM_PAYMENTS_OBJECT_ROOT"

	// PaymentsFamily is the manual payment submission family
	// (payments.manual_write / payments.manual_writeback).
	PaymentsFamily = "payments.manual"
	// FinanceReviewFlag gates the read-only Finance review queue.
	FinanceReviewFlag = "finance.review_read"
)

// ErrPaymentsConfig reports a partial Slice 12 configuration.
var ErrPaymentsConfig = errors.New("crm-api: the Slice 12 payments surface needs " + EnvPaymentsWriteURL + ", " + EnvPaymentsBillingURL + ", " +
	EnvPaymentsObjectRoot + " and " + EnvFeatureFlagsURL)

// paymentsSurface composes the payments write handlers and the Finance
// review read (composition root: neither module imports the other's http).
type paymentsSurface struct {
	*paymentshttp.WriteHandlers
	*financehttp.ReviewHandlers
}

var _ apiserver.PaymentsAPI = paymentsSurface{}

// NewPaymentsAPI composes the Slice 12 surface.
func NewPaymentsAPI(paymentsRunner, billingRunner, flags *db.Runner, root string, devAuth bool) (apiserver.PaymentsAPI, error) {
	if paymentsRunner.Module() != payments.Module || billingRunner.Module() != billing.Module || flags.Module() != owner.FeatureFlags {
		return nil, errors.New("crm-api: payments runners bound to the wrong modules")
	}
	store, err := objectstore.NewFS(root)
	if err != nil {
		return nil, err
	}
	guard, err := authorization.NewGuard(flagFunc(flags, rollout.WriteFlag(PaymentsFamily)), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	review, err := authorization.NewGuard(flagFunc(flags, FinanceReviewFlag), subjects(devAuth))
	if err != nil {
		return nil, err
	}
	mode, err := rollout.FromFlags(PaymentsFamily, flagReader(flags))
	if err != nil {
		return nil, err
	}
	svc, err := paymentscommand.New(paymentsRunner, store, billingquery.New(billingRunner), clock.System{})
	if err != nil {
		return nil, err
	}
	return paymentsSurface{WriteHandlers: paymentshttp.NewWrite(guard, mode, svc), ReviewHandlers: financehttp.NewReview(review, svc)}, nil
}

// openPayments opens the configured Slice 12 pools (nil surface when unset).
func openPayments(ctx context.Context, getenv func(string) string, devAuth bool) (apiserver.PaymentsAPI, func(), error) {
	pw, br, root, fu := getenv(EnvPaymentsWriteURL), getenv(EnvPaymentsBillingURL), getenv(EnvPaymentsObjectRoot), getenv(EnvFeatureFlagsURL)
	if pw == "" && br == "" && root == "" {
		return nil, func() {}, nil
	}
	if pw == "" || br == "" || root == "" || fu == "" {
		return nil, nil, ErrPaymentsConfig
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
	fail := func(err error) (apiserver.PaymentsAPI, func(), error) {
		closeAll()
		return nil, nil, err
	}
	flags, err := open(owner.FeatureFlags, fu)
	if err != nil {
		return fail(err)
	}
	pr, err := open(payments.Module, pw)
	if err != nil {
		return fail(err)
	}
	brn, err := open(billing.Module, br)
	if err != nil {
		return fail(err)
	}
	api, err := NewPaymentsAPI(pr, brn, flags, root, devAuth)
	if err != nil {
		return fail(err)
	}
	return api, closeAll, nil
}
