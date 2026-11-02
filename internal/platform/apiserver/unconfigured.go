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
		oapi.FinanceShadowRead, oapi.FinanceLedgerRead, oapi.FinanceReconciliationRead} {
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
