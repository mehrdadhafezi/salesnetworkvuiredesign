package main

import (
	"context"
	"errors"
	"sort"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/authorization"
	identityapi "github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/identity/api"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/db"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/featureflags"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/rollout"
)

// capabilityFlags maps every registered capability to the feature flag of
// the guard that enforces it (RM-ACCESS-01). It is the availability input of
// the effective-access view, never a source of authority: it adds no
// capability to any role, creates no grant and changes no flag default.
//
// Each value is the exact flag the composing New…API function passes to the
// guard whose Check(ctx, capability) authorises that capability's
// operations. A capability missing here is unavailable (fail closed), and
// TestCapabilityFlagsCoverRegistry fails on any drift.
var capabilityFlags = map[identityapi.Capability]string{
	// Slice 4 HR read (NewReadAPI).
	identityapi.CapUserRead:       HRReadFlag,
	identityapi.CapPersonRead:     HRReadFlag,
	identityapi.CapEmploymentRead: HRReadFlag,
	identityapi.CapPositionRead:   HRReadFlag,
	identityapi.CapUnitRead:       HRReadFlag,
	identityapi.CapAssignmentRead: HRReadFlag,
	// Slice 5 customers (NewCustomersAPI).
	identityapi.CapCustomerRead:       CustomersFlag,
	identityapi.CapMergeCaseRead:      CustomersFlag,
	identityapi.CapIdentityReportRead: CustomersFlag,
	// Slice 6 sales (NewSalesAPI; cases.case.read is also the read guard of NewCaseWriteAPI).
	identityapi.CapCaseRead:        SalesFlag,
	identityapi.CapCustodyRead:     SalesFlag,
	identityapi.CapSalesReportRead: SalesFlag,
	// Slice 7 billing + payments read (NewBillingAPI).
	identityapi.CapInvoiceRead:        BillingFlag,
	identityapi.CapPaymentAttemptRead: BillingFlag,
	identityapi.CapBillingReportRead:  BillingFlag,
	// Slice 8 reporting (NewReportingAPI).
	identityapi.CapReportingMetricsRead: ReportingFlag,
	identityapi.CapReportingMISRead:     ReportingFlag,
	identityapi.CapInvoiceReviewRead:    ReportingFlag,
	// Slice 9 finance shadow (NewFinanceAPI).
	identityapi.CapFinanceShadowRead:         FinanceFlag,
	identityapi.CapFinanceLedgerRead:         FinanceFlag,
	identityapi.CapFinanceReconciliationRead: FinanceFlag,
	// Slice 10 Biawin / fulfillment read (NewBiawinAPI).
	identityapi.CapBiawinSubscriptionRead: BiawinFlag,
	identityapi.CapFulfillmentCaseRead:    BiawinFlag,
	identityapi.CapBiawinParityRead:       BiawinFlag,
	// Slice 11 controlled non-financial writes (NewCaseWriteAPI, NewHRRequestAPI, NewWritebackAPI).
	identityapi.CapContactOutcomeRecord: rollout.WriteFlag(ContactOutcomeFamily),
	identityapi.CapHRRequestCreate:      rollout.WriteFlag(HRRequestFamily),
	identityapi.CapHRRequestReview:      rollout.WriteFlag(HRRequestFamily),
	identityapi.CapHRRequestRead:        rollout.WriteFlag(HRRequestFamily),
	identityapi.CapWritebackRead:        WritebackReadFlag,
	// Slice 12 manual payments (NewPaymentsAPI: write guard and finance review guard).
	identityapi.CapPaymentUploadStage:    rollout.WriteFlag(PaymentsFamily),
	identityapi.CapPaymentBatchSubmit:    rollout.WriteFlag(PaymentsFamily),
	identityapi.CapPaymentSubmissionRead: rollout.WriteFlag(PaymentsFamily),
	identityapi.CapPaymentEvidenceRead:   rollout.WriteFlag(PaymentsFamily),
	identityapi.CapFinanceReviewRead:     FinanceReviewFlag,
	// Slice 13 commission shadow (NewCommissionAPI).
	identityapi.CapCommissionShadowRead: CommissionFlag,
	identityapi.CapCommissionPolicyRead: CommissionFlag,
	identityapi.CapCommissionOwnRead:    CommissionFlag,
	// Slice 14 operations (NewOperationsAPI: read, write and reconcile guards).
	identityapi.CapWorkRead:               OperationsReadFlag,
	identityapi.CapActivationRead:         OperationsReadFlag,
	identityapi.CapProviderOperationRead:  OperationsReadFlag,
	identityapi.CapWorkAssign:             rollout.WriteFlag(WorkFamily),
	identityapi.CapWorkExecute:            rollout.WriteFlag(WorkFamily),
	identityapi.CapProviderOperationRecon: ProviderReconcileFlag,
}

// errNoFlagStore reports an availability source without a flag store.
var errNoFlagStore = errors.New("crm-api: capability availability needs the flag store")

// capabilityAvailability reads every guard flag of capabilityFlags in one
// read-only transaction. An unreadable flag fails the whole evaluation
// (the handler returns an error, never a partial "allowed" view); an absent
// flag is off (featureflags default).
func capabilityAvailability(flags *db.Runner) (authorization.Availability, error) {
	if flags == nil {
		return nil, errNoFlagStore
	}
	keys := map[string]bool{}
	for _, k := range capabilityFlags {
		keys[k] = true
	}
	ordered := make([]string, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	return func(ctx context.Context) (map[identityapi.Capability]bool, error) {
		on := make(map[string]bool, len(ordered))
		err := flags.InTx(ctx, db.TxOptions{ReadOnly: true}, func(ctx context.Context, tx db.Tx) error {
			for _, k := range ordered {
				v, err := featureflags.Enabled(ctx, tx, k)
				if err != nil {
					return err
				}
				on[k] = v
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		out := make(map[identityapi.Capability]bool, len(capabilityFlags))
		for c, k := range capabilityFlags {
			out[c] = on[k]
		}
		return out, nil
	}, nil
}
