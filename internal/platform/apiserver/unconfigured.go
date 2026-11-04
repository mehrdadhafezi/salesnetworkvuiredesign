package apiserver

import (
	"context"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/reasoncodes"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// unconfigured is the Slice 4 read surface when no module is wired (no
// read databases configured): every operation is BLOCKED with
// FEATURE_DISABLED, never an empty success, and nothing is evaluated as
// allowed.
type unconfigured struct{}

var _ ReadAPI = unconfigured{}

func blocked(ctx context.Context) (oapi.Outcome, []oapi.ReasonCode, string) {
	return oapi.Outcome(result.Blocked.String()), []oapi.ReasonCode{oapi.ReasonCode(reasoncodes.FeatureDisabled)}, httpx.CorrelationID(ctx)
}

func (unconfigured) HRReadEnabled(context.Context) bool { return false }

func (unconfigured) GetEffectiveAccess(ctx context.Context, _ oapi.GetEffectiveAccessRequestObject) (oapi.GetEffectiveAccessResponseObject, error) {
	data := oapi.EffectiveAccess{Decisions: []oapi.AccessDecision{}}
	reason := oapi.ReasonCode(reasoncodes.FeatureDisabled)
	for _, c := range []oapi.Capability{oapi.IdentityUserRead, oapi.OrgPersonRead, oapi.OrgEmploymentRead, oapi.OrgPositionRead, oapi.OrgUnitRead,
		oapi.OrgAssignmentRead, oapi.CustomersCustomerRead, oapi.CustomersMergeCaseRead, oapi.CustomersIdentityReportRead,
		oapi.CasesCaseRead, oapi.CustodyCustodyRead, oapi.SalesReportRead, oapi.BillingInvoiceRead, oapi.PaymentsAttemptRead,
		oapi.BillingReconciliationRead, oapi.ReportingMetricsRead, oapi.ReportingMisRead, oapi.ReportingInvoiceReviewRead,
		oapi.FinanceShadowRead, oapi.FinanceLedgerRead, oapi.FinanceReconciliationRead,
		oapi.BiawinSubscriptionRead, oapi.FulfillmentCaseRead, oapi.BiawinParityRead,
		oapi.CasesContactOutcomeRecord, oapi.OrgHrRequestCreate, oapi.OrgHrRequestReview, oapi.OrgHrRequestRead, oapi.IntegrationWritebackRead} {
		data.Decisions = append(data.Decisions, oapi.AccessDecision{Capability: c, Outcome: oapi.Outcome(result.Blocked.String()),
			ReasonCode: &reason, Scope: oapi.AccessDecisionScopeNONE, MatchedRoles: []string{}})
	}
	return oapi.GetEffectiveAccess200JSONResponse{Body: oapi.EffectiveAccessEnvelope{Outcome: oapi.Outcome(result.Success.String()),
		Reasons: []oapi.ReasonCode{}, CorrelationId: httpx.CorrelationID(ctx), Data: data}}, nil
}

func (unconfigured) ListIdentityUsers(ctx context.Context, _ oapi.ListIdentityUsersRequestObject) (oapi.ListIdentityUsersResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListIdentityUsers200JSONResponse{Body: oapi.UserListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) GetIdentityUser(ctx context.Context, _ oapi.GetIdentityUserRequestObject) (oapi.GetIdentityUserResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetIdentityUser200JSONResponse{Body: oapi.UserEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationPersons(ctx context.Context, _ oapi.ListOrganizationPersonsRequestObject) (oapi.ListOrganizationPersonsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationPersons200JSONResponse{Body: oapi.PersonListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) GetOrganizationPerson(ctx context.Context, _ oapi.GetOrganizationPersonRequestObject) (oapi.GetOrganizationPersonResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetOrganizationPerson200JSONResponse{Body: oapi.PersonEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationPositions(ctx context.Context, _ oapi.ListOrganizationPositionsRequestObject) (oapi.ListOrganizationPositionsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationPositions200JSONResponse{Body: oapi.LookupListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationLevels(ctx context.Context, _ oapi.ListOrganizationLevelsRequestObject) (oapi.ListOrganizationLevelsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationLevels200JSONResponse{Body: oapi.LookupListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationOrgUnits(ctx context.Context, _ oapi.ListOrganizationOrgUnitsRequestObject) (oapi.ListOrganizationOrgUnitsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationOrgUnits200JSONResponse{Body: oapi.OrgUnitListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationAssignments(ctx context.Context, _ oapi.ListOrganizationAssignmentsRequestObject) (oapi.ListOrganizationAssignmentsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationAssignments200JSONResponse{Body: oapi.AssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfigured) ListOrganizationHierarchyFindings(ctx context.Context, _ oapi.ListOrganizationHierarchyFindingsRequestObject) (oapi.ListOrganizationHierarchyFindingsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListOrganizationHierarchyFindings200JSONResponse{Body: oapi.FindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredCustomers is the Slice 5 surface without a customers module:
// every operation is BLOCKED / FEATURE_DISABLED.
type unconfiguredCustomers struct{}

var _ CustomersAPI = unconfiguredCustomers{}

func (unconfiguredCustomers) CustomersEnabled(context.Context) bool { return false }

func (unconfiguredCustomers) ListCustomers(ctx context.Context, _ oapi.ListCustomersRequestObject) (oapi.ListCustomersResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCustomers200JSONResponse{Body: oapi.CustomerListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredCustomers) GetCustomer(ctx context.Context, _ oapi.GetCustomerRequestObject) (oapi.GetCustomerResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCustomer200JSONResponse{Body: oapi.CustomerEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredCustomers) ListCustomerContactPoints(ctx context.Context, _ oapi.ListCustomerContactPointsRequestObject) (oapi.ListCustomerContactPointsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCustomerContactPoints200JSONResponse{Body: oapi.ContactPointListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredCustomers) ListCustomerEvidence(ctx context.Context, _ oapi.ListCustomerEvidenceRequestObject) (oapi.ListCustomerEvidenceResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCustomerEvidence200JSONResponse{Body: oapi.EvidenceListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredCustomers) ListMergeCases(ctx context.Context, _ oapi.ListMergeCasesRequestObject) (oapi.ListMergeCasesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListMergeCases200JSONResponse{Body: oapi.MergeCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredCustomers) GetMergeCase(ctx context.Context, _ oapi.GetMergeCaseRequestObject) (oapi.GetMergeCaseResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetMergeCase200JSONResponse{Body: oapi.MergeCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredCustomers) GetCustomerResolutionReport(ctx context.Context, _ oapi.GetCustomerResolutionReportRequestObject) (oapi.GetCustomerResolutionReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCustomerResolutionReport200JSONResponse{Body: oapi.ResolutionReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredSales is the Slice 6 surface without the cases/custody
// modules: every operation is BLOCKED / FEATURE_DISABLED.
type unconfiguredSales struct{}

var _ SalesAPI = unconfiguredSales{}

func (unconfiguredSales) SalesEnabled(context.Context) bool { return false }

func (unconfiguredSales) ListCases(ctx context.Context, _ oapi.ListCasesRequestObject) (oapi.ListCasesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCases200JSONResponse{Body: oapi.CaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) GetCase(ctx context.Context, _ oapi.GetCaseRequestObject) (oapi.GetCaseResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCase200JSONResponse{Body: oapi.CaseEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) ListCaseHistory(ctx context.Context, _ oapi.ListCaseHistoryRequestObject) (oapi.ListCaseHistoryResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCaseHistory200JSONResponse{Body: oapi.StateHistoryListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) ListCaseAttribution(ctx context.Context, _ oapi.ListCaseAttributionRequestObject) (oapi.ListCaseAttributionResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCaseAttribution200JSONResponse{Body: oapi.AttributionListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) ListCustodyAssignments(ctx context.Context, _ oapi.ListCustodyAssignmentsRequestObject) (oapi.ListCustodyAssignmentsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCustodyAssignments200JSONResponse{Body: oapi.CustodyAssignmentListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) GetCaseCustody(ctx context.Context, _ oapi.GetCaseCustodyRequestObject) (oapi.GetCaseCustodyResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCaseCustody200JSONResponse{Body: oapi.CaseCustodyEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) GetSalesCoverageReport(ctx context.Context, _ oapi.GetSalesCoverageReportRequestObject) (oapi.GetSalesCoverageReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetSalesCoverageReport200JSONResponse{Body: oapi.SalesCoverageReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredSales) GetCustodyParityReport(ctx context.Context, _ oapi.GetCustodyParityReportRequestObject) (oapi.GetCustodyParityReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCustodyParityReport200JSONResponse{Body: oapi.CustodyParityReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredBilling is the Slice 7 surface without the billing/payments
// modules: every operation is BLOCKED / FEATURE_DISABLED.
type unconfiguredBilling struct{}

var _ BillingAPI = unconfiguredBilling{}

func (unconfiguredBilling) BillingEnabled(context.Context) bool { return false }

func (unconfiguredBilling) ListInvoices(ctx context.Context, _ oapi.ListInvoicesRequestObject) (oapi.ListInvoicesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListInvoices200JSONResponse{Body: oapi.InvoiceListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) GetInvoice(ctx context.Context, _ oapi.GetInvoiceRequestObject) (oapi.GetInvoiceResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetInvoice200JSONResponse{Body: oapi.InvoiceEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) ListInvoiceItems(ctx context.Context, _ oapi.ListInvoiceItemsRequestObject) (oapi.ListInvoiceItemsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListInvoiceItems200JSONResponse{Body: oapi.InvoiceLineListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) ListInvoiceStages(ctx context.Context, _ oapi.ListInvoiceStagesRequestObject) (oapi.ListInvoiceStagesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListInvoiceStages200JSONResponse{Body: oapi.PaymentStageListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) GetInvoiceTotals(ctx context.Context, _ oapi.GetInvoiceTotalsRequestObject) (oapi.GetInvoiceTotalsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetInvoiceTotals200JSONResponse{Body: oapi.InvoiceTotalsEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) ListInvoicePaymentAttempts(ctx context.Context, _ oapi.ListInvoicePaymentAttemptsRequestObject) (oapi.ListInvoicePaymentAttemptsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListInvoicePaymentAttempts200JSONResponse{Body: oapi.PaymentAttemptListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) ListInvoiceReceipts(ctx context.Context, _ oapi.ListInvoiceReceiptsRequestObject) (oapi.ListInvoiceReceiptsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListInvoiceReceipts200JSONResponse{Body: oapi.ReceiptReferenceListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) ListPaymentAttempts(ctx context.Context, _ oapi.ListPaymentAttemptsRequestObject) (oapi.ListPaymentAttemptsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListPaymentAttempts200JSONResponse{Body: oapi.PaymentAttemptListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) GetPaymentAttempt(ctx context.Context, _ oapi.GetPaymentAttemptRequestObject) (oapi.GetPaymentAttemptResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetPaymentAttempt200JSONResponse{Body: oapi.PaymentAttemptEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) ListPaymentAttemptHistory(ctx context.Context, _ oapi.ListPaymentAttemptHistoryRequestObject) (oapi.ListPaymentAttemptHistoryResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListPaymentAttemptHistory200JSONResponse{Body: oapi.AttemptHistoryListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) GetReceiptReference(ctx context.Context, _ oapi.GetReceiptReferenceRequestObject) (oapi.GetReceiptReferenceResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetReceiptReference200JSONResponse{Body: oapi.ReceiptReferenceEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredBilling) GetBillingReconciliationReport(ctx context.Context, _ oapi.GetBillingReconciliationReportRequestObject) (oapi.GetBillingReconciliationReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetBillingReconciliationReport200JSONResponse{Body: oapi.BillingReconciliationReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredReporting is the Slice 8 surface without the reporting module:
// every operation is BLOCKED / FEATURE_DISABLED.
type unconfiguredReporting struct{}

var _ ReportingAPI = unconfiguredReporting{}

func (unconfiguredReporting) ReportingEnabled(context.Context) bool { return false }

func (unconfiguredReporting) ListReportingMetrics(ctx context.Context, _ oapi.ListReportingMetricsRequestObject) (oapi.ListReportingMetricsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListReportingMetrics200JSONResponse{Body: oapi.MetricListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetReportingMetric(ctx context.Context, _ oapi.GetReportingMetricRequestObject) (oapi.GetReportingMetricResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetReportingMetric200JSONResponse{Body: oapi.MetricEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetCaseReport(ctx context.Context, _ oapi.GetCaseReportRequestObject) (oapi.GetCaseReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCaseReport200JSONResponse{Body: oapi.CaseReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetCaseReportRow(ctx context.Context, _ oapi.GetCaseReportRowRequestObject) (oapi.GetCaseReportRowResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCaseReportRow200JSONResponse{Body: oapi.CaseDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetCustodyReport(ctx context.Context, _ oapi.GetCustodyReportRequestObject) (oapi.GetCustodyReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCustodyReport200JSONResponse{Body: oapi.CustodyReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetCustodyReportRow(ctx context.Context, _ oapi.GetCustodyReportRowRequestObject) (oapi.GetCustodyReportRowResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCustodyReportRow200JSONResponse{Body: oapi.CustodyDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetInvoiceReport(ctx context.Context, _ oapi.GetInvoiceReportRequestObject) (oapi.GetInvoiceReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetInvoiceReport200JSONResponse{Body: oapi.InvoiceReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetInvoiceReportRow(ctx context.Context, _ oapi.GetInvoiceReportRowRequestObject) (oapi.GetInvoiceReportRowResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetInvoiceReportRow200JSONResponse{Body: oapi.InvoiceDetailFactEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetPaymentReport(ctx context.Context, _ oapi.GetPaymentReportRequestObject) (oapi.GetPaymentReportResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetPaymentReport200JSONResponse{Body: oapi.PaymentReportEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetPaymentReportRow(ctx context.Context, _ oapi.GetPaymentReportRowRequestObject) (oapi.GetPaymentReportRowResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetPaymentReportRow200JSONResponse{Body: oapi.PaymentDetailEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetMisOverview(ctx context.Context, _ oapi.GetMisOverviewRequestObject) (oapi.GetMisOverviewResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetMisOverview200JSONResponse{Body: oapi.MisOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetMisSourceQuality(ctx context.Context, _ oapi.GetMisSourceQualityRequestObject) (oapi.GetMisSourceQualityResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetMisSourceQuality200JSONResponse{Body: oapi.SourceQualityEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetMisReconciliation(ctx context.Context, _ oapi.GetMisReconciliationRequestObject) (oapi.GetMisReconciliationResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetMisReconciliation200JSONResponse{Body: oapi.MisReconciliationEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) ListInvoiceReview(ctx context.Context, _ oapi.ListInvoiceReviewRequestObject) (oapi.ListInvoiceReviewResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredReporting) GetInvoiceReview(ctx context.Context, _ oapi.GetInvoiceReviewRequestObject) (oapi.GetInvoiceReviewResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetInvoiceReview200JSONResponse{Body: oapi.InvoiceReviewEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredFinance is the Slice 9 surface without the finance module:
// every operation is BLOCKED / FEATURE_DISABLED.
type unconfiguredFinance struct{}

var _ FinanceAPI = unconfiguredFinance{}

func (unconfiguredFinance) FinanceEnabled(context.Context) bool { return false }

func (unconfiguredFinance) ListFinanceShadowRuns(ctx context.Context, _ oapi.ListFinanceShadowRunsRequestObject) (oapi.ListFinanceShadowRunsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFinanceShadowRuns200JSONResponse{Body: oapi.ShadowRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: oapi.SHADOWDerivedForArchitectureValidationAndReconciliationOnlyNotTheFinancialSystemOfRecordNotSettledBankConfirmedOrFinanceApprovedMoneyNotAnOfficialOrAccountingBalance}}, nil
}

func (unconfiguredFinance) GetFinanceShadowRun(ctx context.Context, _ oapi.GetFinanceShadowRunRequestObject) (oapi.GetFinanceShadowRunResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetFinanceShadowRun200JSONResponse{Body: oapi.ShadowRunEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: oapi.SHADOWDerivedForArchitectureValidationAndReconciliationOnlyNotTheFinancialSystemOfRecordNotSettledBankConfirmedOrFinanceApprovedMoneyNotAnOfficialOrAccountingBalance}}, nil
}

func (unconfiguredFinance) ListFinanceAccounts(ctx context.Context, _ oapi.ListFinanceAccountsRequestObject) (oapi.ListFinanceAccountsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFinanceAccounts200JSONResponse{Body: oapi.AccountListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredFinance) GetFinanceAccount(ctx context.Context, _ oapi.GetFinanceAccountRequestObject) (oapi.GetFinanceAccountResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetFinanceAccount200JSONResponse{Body: oapi.AccountEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredFinance) ListFinanceLedger(ctx context.Context, _ oapi.ListFinanceLedgerRequestObject) (oapi.ListFinanceLedgerResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFinanceLedger200JSONResponse{Body: oapi.LedgerListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredFinance) GetFinanceLedgerEntry(ctx context.Context, _ oapi.GetFinanceLedgerEntryRequestObject) (oapi.GetFinanceLedgerEntryResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetFinanceLedgerEntry200JSONResponse{Body: oapi.LedgerEntryEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredFinance) ListFinanceBalances(ctx context.Context, _ oapi.ListFinanceBalancesRequestObject) (oapi.ListFinanceBalancesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFinanceBalances200JSONResponse{Body: oapi.BalanceListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredFinance) ListFinanceReconciliation(ctx context.Context, _ oapi.ListFinanceReconciliationRequestObject) (oapi.ListFinanceReconciliationResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFinanceReconciliation200JSONResponse{Body: oapi.ReconciliationListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredBiawin is the Slice 10 surface without the biawin and
// fulfillment modules: every operation is BLOCKED / FEATURE_DISABLED.
type unconfiguredBiawin struct{}

var _ BiawinAPI = unconfiguredBiawin{}

const biawinNotice = oapi.SHADOWImportedLegacySubscriptionAndFulfillmentEvidenceForArchitectureValidationAndParityOnlyNotLiveProviderStateNoActivationProviderShippingOrWorkflowAuthority

func (unconfiguredBiawin) BiawinEnabled(context.Context) bool { return false }

func (unconfiguredBiawin) ListBiawinSubscriptions(ctx context.Context, _ oapi.ListBiawinSubscriptionsRequestObject) (oapi.ListBiawinSubscriptionsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListBiawinSubscriptions200JSONResponse{Body: oapi.SubscriptionListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

func (unconfiguredBiawin) GetBiawinSubscription(ctx context.Context, _ oapi.GetBiawinSubscriptionRequestObject) (oapi.GetBiawinSubscriptionResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetBiawinSubscription200JSONResponse{Body: oapi.SubscriptionEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

func (unconfiguredBiawin) ListBiawinSubscriptionItems(ctx context.Context, _ oapi.ListBiawinSubscriptionItemsRequestObject) (oapi.ListBiawinSubscriptionItemsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListBiawinSubscriptionItems200JSONResponse{Body: oapi.SubscriptionItemListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

func (unconfiguredBiawin) GetBiawinParity(ctx context.Context, _ oapi.GetBiawinParityRequestObject) (oapi.GetBiawinParityResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetBiawinParity200JSONResponse{Body: oapi.ParityEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

func (unconfiguredBiawin) ListFulfillmentCases(ctx context.Context, _ oapi.ListFulfillmentCasesRequestObject) (oapi.ListFulfillmentCasesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFulfillmentCases200JSONResponse{Body: oapi.FulfillmentCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

func (unconfiguredBiawin) GetFulfillmentCase(ctx context.Context, _ oapi.GetFulfillmentCaseRequestObject) (oapi.GetFulfillmentCaseResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetFulfillmentCase200JSONResponse{Body: oapi.FulfillmentCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

func (unconfiguredBiawin) ListFulfillmentCaseHistory(ctx context.Context, _ oapi.ListFulfillmentCaseHistoryRequestObject) (oapi.ListFulfillmentCaseHistoryResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFulfillmentCaseHistory200JSONResponse{Body: oapi.FulfillmentCaseHistoryEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: biawinNotice}}, nil
}

// Slice 11 surfaces without their modules: every operation is BLOCKED /
// FEATURE_DISABLED (commands included), never an empty success.

type unconfiguredCaseWrite struct{}

var _ CasesWriteAPI = unconfiguredCaseWrite{}

func (unconfiguredCaseWrite) ContactOutcomeWriteEnabled(context.Context) bool { return false }

func (unconfiguredCaseWrite) GetCaseContactOutcomes(ctx context.Context, _ oapi.GetCaseContactOutcomesRequestObject) (oapi.GetCaseContactOutcomesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCaseContactOutcomes200JSONResponse{Body: oapi.ContactOutcomeEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func blockedCommand(ctx context.Context) oapi.CommandEnvelope {
	o, r, c := blocked(ctx)
	return oapi.CommandEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Command: &oapi.CommandResult{RolloutMode: oapi.RolloutModeTARGETWRITEDISABLED,
		WritebackState: oapi.CommandWritebackStateNOTREQUESTED, BlockedBy: []string{}}}
}

func (unconfiguredCaseWrite) RecordContactOutcome(ctx context.Context, _ oapi.RecordContactOutcomeRequestObject) (oapi.RecordContactOutcomeResponseObject, error) {
	return oapi.RecordContactOutcome200JSONResponse{Body: blockedCommand(ctx)}, nil
}

type unconfiguredHRRequest struct{}

var _ HRRequestAPI = unconfiguredHRRequest{}

func (unconfiguredHRRequest) HRRequestWriteEnabled(context.Context) bool { return false }

func (unconfiguredHRRequest) ListHRRequests(ctx context.Context, _ oapi.ListHRRequestsRequestObject) (oapi.ListHRRequestsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListHRRequests200JSONResponse{Body: oapi.HRRequestListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredHRRequest) GetHRRequest(ctx context.Context, _ oapi.GetHRRequestRequestObject) (oapi.GetHRRequestResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetHRRequest200JSONResponse{Body: oapi.HRRequestEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredHRRequest) CreateHRRequest(ctx context.Context, _ oapi.CreateHRRequestRequestObject) (oapi.CreateHRRequestResponseObject, error) {
	return oapi.CreateHRRequest200JSONResponse{Body: blockedCommand(ctx)}, nil
}

func (unconfiguredHRRequest) DecideHRRequest(ctx context.Context, _ oapi.DecideHRRequestRequestObject) (oapi.DecideHRRequestResponseObject, error) {
	return oapi.DecideHRRequest200JSONResponse{Body: blockedCommand(ctx)}, nil
}

type unconfiguredWriteback struct{}

var _ WritebackAPI = unconfiguredWriteback{}

func (unconfiguredWriteback) WritebackEnabled(context.Context) bool { return false }

func (unconfiguredWriteback) GetWritebackSummary(ctx context.Context, _ oapi.GetWritebackSummaryRequestObject) (oapi.GetWritebackSummaryResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetWritebackSummary200JSONResponse{Body: oapi.WritebackSummaryEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredWriteback) ListWritebackCommands(ctx context.Context, _ oapi.ListWritebackCommandsRequestObject) (oapi.ListWritebackCommandsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListWritebackCommands200JSONResponse{Body: oapi.WritebackCommandListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredWriteback) GetWritebackCommand(ctx context.Context, _ oapi.GetWritebackCommandRequestObject) (oapi.GetWritebackCommandResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetWritebackCommand200JSONResponse{Body: oapi.WritebackCommandEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredPayments answers the Slice 12 payments surface when it is not
// wired: every read is BLOCKED / FEATURE_DISABLED and every command is
// BLOCKED with nothing applied.
type unconfiguredPayments struct{}

var _ PaymentsAPI = unconfiguredPayments{}

func (unconfiguredPayments) PaymentSubmissionEnabled(context.Context) bool { return false }
func (unconfiguredPayments) FinanceReviewEnabled(context.Context) bool     { return false }

func (unconfiguredPayments) StagePaymentUpload(ctx context.Context, _ oapi.StagePaymentUploadRequestObject) (oapi.StagePaymentUploadResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.StagePaymentUpload200JSONResponse{Body: oapi.PaymentUploadEnvelope{Outcome: o, Reasons: r, CorrelationId: c,
		Data: &oapi.PaymentUploadResult{RolloutMode: oapi.RolloutModeTARGETWRITEDISABLED, BlockedBy: []string{}}}}, nil
}

func (unconfiguredPayments) GetPaymentSubmission(ctx context.Context, _ oapi.GetPaymentSubmissionRequestObject) (oapi.GetPaymentSubmissionResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetPaymentSubmission200JSONResponse{Body: oapi.PaymentSubmissionEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredPayments) SubmitPaymentBatch(ctx context.Context, req oapi.SubmitPaymentBatchRequestObject) (oapi.SubmitPaymentBatchResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.SubmitPaymentBatch200JSONResponse{Body: oapi.PaymentBatchEnvelope{Outcome: o, Reasons: r, CorrelationId: c,
		Data: &oapi.PaymentBatchResult{RolloutMode: oapi.RolloutModeTARGETWRITEDISABLED, InvoiceId: req.InvoiceId, Items: []oapi.PaymentBatchItemOutcome{},
			WritebackState: oapi.PaymentBatchResultWritebackStateNOTREQUESTED, BlockedBy: []string{}}}}, nil
}

func (unconfiguredPayments) GetPaymentBatch(ctx context.Context, _ oapi.GetPaymentBatchRequestObject) (oapi.GetPaymentBatchResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetPaymentBatch200JSONResponse{Body: oapi.PaymentBatchReadEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredPayments) GetPaymentEvidence(ctx context.Context, _ oapi.GetPaymentEvidenceRequestObject) (oapi.GetPaymentEvidenceResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetPaymentEvidence200JSONResponse{Body: oapi.EvidenceDiagnosticsEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredPayments) ListFinanceReviewQueue(ctx context.Context, _ oapi.ListFinanceReviewQueueRequestObject) (oapi.ListFinanceReviewQueueResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFinanceReviewQueue200JSONResponse{Body: oapi.ReviewQueueEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredPayments) GetFinanceReviewAttempt(ctx context.Context, _ oapi.GetFinanceReviewAttemptRequestObject) (oapi.GetFinanceReviewAttemptResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetFinanceReviewAttempt200JSONResponse{Body: oapi.ReviewAttemptEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

// unconfiguredCommission is the Slice 13 surface without the commission
// login: every operation answers BLOCKED / FEATURE_DISABLED.
type unconfiguredCommission struct{}

var _ CommissionAPI = unconfiguredCommission{}

const commissionNotice = oapi.SHADOWOnlyNoCommissionWasCalculatedApprovedPostedOrCreditedByThisReadLegacyRemainsTheSystemOfRecord

func (unconfiguredCommission) CommissionEnabled(context.Context) bool { return false }

func (unconfiguredCommission) GetCommissionOverview(ctx context.Context, _ oapi.GetCommissionOverviewRequestObject) (oapi.GetCommissionOverviewResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetCommissionOverview200JSONResponse{Body: oapi.CommissionOverviewEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

func (unconfiguredCommission) ListCommissionRuns(ctx context.Context, _ oapi.ListCommissionRunsRequestObject) (oapi.ListCommissionRunsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCommissionRuns200JSONResponse{Body: oapi.CommissionRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

func (unconfiguredCommission) ListCommissionCalculations(ctx context.Context, _ oapi.ListCommissionCalculationsRequestObject) (oapi.ListCommissionCalculationsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCommissionCalculations200JSONResponse{Body: oapi.CommissionCalculationListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

func (unconfiguredCommission) ListCommissionFindings(ctx context.Context, _ oapi.ListCommissionFindingsRequestObject) (oapi.ListCommissionFindingsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCommissionFindings200JSONResponse{Body: oapi.CommissionFindingListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

func (unconfiguredCommission) ListCommissionPolicies(ctx context.Context, _ oapi.ListCommissionPoliciesRequestObject) (oapi.ListCommissionPoliciesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCommissionPolicies200JSONResponse{Body: oapi.CommissionPolicyListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

func (unconfiguredCommission) ListCommissionHistoricalRuns(ctx context.Context, _ oapi.ListCommissionHistoricalRunsRequestObject) (oapi.ListCommissionHistoricalRunsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListCommissionHistoricalRuns200JSONResponse{Body: oapi.CommissionHistoricalRunListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

func (unconfiguredCommission) ListMyCommissionEarnings(ctx context.Context, _ oapi.ListMyCommissionEarningsRequestObject) (oapi.ListMyCommissionEarningsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListMyCommissionEarnings200JSONResponse{Body: oapi.CommissionEarningListEnvelope{Outcome: o, Reasons: r, CorrelationId: c, Notice: commissionNotice}}, nil
}

// unconfiguredOperations is the Slice 14 surface without its module logins:
// every operation answers BLOCKED / FEATURE_DISABLED and nothing runs.
type unconfiguredOperations struct{}

var _ OperationsAPI = unconfiguredOperations{}

func (unconfiguredOperations) OperationsEnabled(context.Context) bool { return false }

func (unconfiguredOperations) ListFulfillmentWorkCases(ctx context.Context, _ oapi.ListFulfillmentWorkCasesRequestObject) (oapi.ListFulfillmentWorkCasesResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListFulfillmentWorkCases200JSONResponse{Body: oapi.WorkCaseListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) GetFulfillmentWorkCase(ctx context.Context, _ oapi.GetFulfillmentWorkCaseRequestObject) (oapi.GetFulfillmentWorkCaseResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.GetFulfillmentWorkCase200JSONResponse{Body: oapi.WorkCaseEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) AssignFulfillmentWorkCase(ctx context.Context, _ oapi.AssignFulfillmentWorkCaseRequestObject) (oapi.AssignFulfillmentWorkCaseResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.AssignFulfillmentWorkCase200JSONResponse{Body: oapi.WorkCommandEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) RecordFulfillmentWorkAction(ctx context.Context, _ oapi.RecordFulfillmentWorkActionRequestObject) (oapi.RecordFulfillmentWorkActionResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.RecordFulfillmentWorkAction200JSONResponse{Body: oapi.WorkCommandEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) AttachFulfillmentWorkEvidence(ctx context.Context, _ oapi.AttachFulfillmentWorkEvidenceRequestObject) (oapi.AttachFulfillmentWorkEvidenceResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.AttachFulfillmentWorkEvidence200JSONResponse{Body: oapi.WorkCommandEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) CompleteFulfillmentWorkCase(ctx context.Context, _ oapi.CompleteFulfillmentWorkCaseRequestObject) (oapi.CompleteFulfillmentWorkCaseResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.CompleteFulfillmentWorkCase200JSONResponse{Body: oapi.WorkCommandEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) ListBiawinActivationDecisions(ctx context.Context, _ oapi.ListBiawinActivationDecisionsRequestObject) (oapi.ListBiawinActivationDecisionsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListBiawinActivationDecisions200JSONResponse{Body: oapi.ActivationListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) ListProviderOperations(ctx context.Context, _ oapi.ListProviderOperationsRequestObject) (oapi.ListProviderOperationsResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ListProviderOperations200JSONResponse{Body: oapi.ProviderOperationListEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}

func (unconfiguredOperations) ResolveProviderOperation(ctx context.Context, _ oapi.ResolveProviderOperationRequestObject) (oapi.ResolveProviderOperationResponseObject, error) {
	o, r, c := blocked(ctx)
	return oapi.ResolveProviderOperation200JSONResponse{Body: oapi.ProviderOperationEnvelope{Outcome: o, Reasons: r, CorrelationId: c}}, nil
}
