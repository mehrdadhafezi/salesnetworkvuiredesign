package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/websession"
)

// ContractPaths are the paths of the OpenAPI contract with their methods.
// They drive the central 404/405 problem handling; TestRoutesMatchContract
// fails if they drift from the bundled contract.
var ContractPaths = map[string][]string{
	"/health/live":    {http.MethodGet},
	"/health/ready":   {http.MethodGet},
	"/api/v1/session": {http.MethodGet},
	// R1 production authentication (server-side sessions).
	"/api/v1/auth/login":                       {http.MethodPost},
	"/api/v1/auth/logout":                      {http.MethodPost},
	"/api/v1/auth/logout-all":                  {http.MethodPost},
	"/api/v1/identity/users":                   {http.MethodGet},
	"/api/v1/identity/users/{user_id}":         {http.MethodGet},
	"/api/v1/organization/persons":             {http.MethodGet},
	"/api/v1/organization/persons/{person_id}": {http.MethodGet},
	"/api/v1/organization/positions":           {http.MethodGet},
	"/api/v1/organization/levels":              {http.MethodGet},
	"/api/v1/organization/org-units":           {http.MethodGet},
	"/api/v1/organization/assignments":         {http.MethodGet},
	"/api/v1/organization/hierarchy-findings":  {http.MethodGet},
	"/api/v1/authorization/effective-access":   {http.MethodGet},
	// Slice 5 customer identity (read/shadow only).
	"/api/v1/customers":                                     {http.MethodGet},
	"/api/v1/customers/{customer_id}":                       {http.MethodGet},
	"/api/v1/customers/{customer_id}/contact-points":        {http.MethodGet},
	"/api/v1/customers/{customer_id}/evidence":              {http.MethodGet},
	"/api/v1/customer-identity/merge-cases":                 {http.MethodGet},
	"/api/v1/customer-identity/merge-cases/{merge_case_id}": {http.MethodGet},
	"/api/v1/customer-identity/resolution-report":           {http.MethodGet},
	// Slice 6 sales read shadow (read only; no action endpoint exists).
	"/api/v1/cases":                       {http.MethodGet},
	"/api/v1/cases/{case_id}":             {http.MethodGet},
	"/api/v1/cases/{case_id}/history":     {http.MethodGet},
	"/api/v1/cases/{case_id}/attribution": {http.MethodGet},
	"/api/v1/custody/assignments":         {http.MethodGet},
	"/api/v1/custody/cases/{case_id}":     {http.MethodGet},
	"/api/v1/sales/coverage-report":       {http.MethodGet},
	"/api/v1/sales/custody-parity":        {http.MethodGet},
	// Slice 7 billing + payments read shadow (read only; no action endpoint exists).
	"/api/v1/invoices":                               {http.MethodGet},
	"/api/v1/invoices/{invoice_id}":                  {http.MethodGet},
	"/api/v1/invoices/{invoice_id}/items":            {http.MethodGet},
	"/api/v1/invoices/{invoice_id}/stages":           {http.MethodGet},
	"/api/v1/invoices/{invoice_id}/totals":           {http.MethodGet},
	"/api/v1/invoices/{invoice_id}/payments":         {http.MethodGet},
	"/api/v1/invoices/{invoice_id}/receipts":         {http.MethodGet},
	"/api/v1/payments/attempts":                      {http.MethodGet},
	"/api/v1/payments/attempts/{attempt_id}":         {http.MethodGet},
	"/api/v1/payments/attempts/{attempt_id}/history": {http.MethodGet},
	"/api/v1/payments/receipts/{receipt_id}":         {http.MethodGet},
	"/api/v1/billing/reconciliation-report":          {http.MethodGet},
	// Slice 8 MIS / reporting read (read only; no export path of any kind).
	"/api/v1/reporting/metrics":             {http.MethodGet},
	"/api/v1/reporting/metrics/{metric_id}": {http.MethodGet},
	"/api/v1/reports/cases":                 {http.MethodGet},
	"/api/v1/reports/cases/{case_id}":       {http.MethodGet},
	"/api/v1/reports/custody":               {http.MethodGet},
	"/api/v1/reports/custody/{case_id}":     {http.MethodGet},
	"/api/v1/reports/invoices":              {http.MethodGet},
	"/api/v1/reports/invoices/{invoice_id}": {http.MethodGet},
	"/api/v1/reports/payments":              {http.MethodGet},
	"/api/v1/reports/payments/{attempt_id}": {http.MethodGet},
	"/api/v1/mis/overview":                  {http.MethodGet},
	"/api/v1/mis/source-quality":            {http.MethodGet},
	"/api/v1/mis/reconciliation":            {http.MethodGet},
	// Slice 9 finance SHADOW ledger read (GET only; no posting, approval,
	// reversal, settlement, refund, credit or debit path of any kind).
	"/api/v1/finance/shadow-runs":           {http.MethodGet},
	"/api/v1/finance/shadow-runs/{run_id}":  {http.MethodGet},
	"/api/v1/finance/accounts":              {http.MethodGet},
	"/api/v1/finance/accounts/{account_id}": {http.MethodGet},
	"/api/v1/finance/ledger":                {http.MethodGet},
	"/api/v1/finance/ledger/{entry_id}":     {http.MethodGet},
	"/api/v1/finance/balances":              {http.MethodGet},
	"/api/v1/finance/reconciliation":        {http.MethodGet},
	// Slice 10 Biawin / fulfillment READ shadow (GET only; no activation,
	// suspension, provider, wallet, shipping, assignment or completion path).
	"/api/v1/biawin/subscriptions":                         {http.MethodGet},
	"/api/v1/biawin/subscriptions/{subscription_id}":       {http.MethodGet},
	"/api/v1/biawin/subscriptions/{subscription_id}/items": {http.MethodGet},
	"/api/v1/biawin/parity":                                {http.MethodGet},
	// Slice 14 controlled fulfillment and Biawin writes: per-service work
	// cases and the evidenced, non-financial commands; SHADOW activation
	// decisions; the provider operation ledger with human reconciliation.
	"/api/v1/fulfillment/work-cases":                                    {http.MethodGet},
	"/api/v1/fulfillment/work-cases/{case_id}":                          {http.MethodGet},
	"/api/v1/fulfillment/work-cases/{case_id}/assignment":               {http.MethodPost},
	"/api/v1/fulfillment/work-cases/{case_id}/operational-actions":      {http.MethodPost},
	"/api/v1/fulfillment/work-cases/{case_id}/evidence-records":         {http.MethodPost},
	"/api/v1/fulfillment/work-cases/{case_id}/outcome":                  {http.MethodPost},
	"/api/v1/biawin/activation-decisions":                               {http.MethodGet},
	"/api/v1/integration/provider-operations":                           {http.MethodGet},
	"/api/v1/integration/provider-operations/{operation_id}/resolution": {http.MethodPost},
	// Slice 13 commission controlled SHADOW read (GET only; no calculate,
	// approve, post, credit, settle or reverse operation exists).
	"/api/v1/commission/overview":                 {http.MethodGet},
	"/api/v1/commission/runs":                     {http.MethodGet},
	"/api/v1/commission/calculations":             {http.MethodGet},
	"/api/v1/commission/findings":                 {http.MethodGet},
	"/api/v1/commission/policies":                 {http.MethodGet},
	"/api/v1/commission/historical-runs":          {http.MethodGet},
	"/api/v1/commission/me/earnings":              {http.MethodGet},
	"/api/v1/fulfillment/cases":                   {http.MethodGet},
	"/api/v1/fulfillment/cases/{case_id}":         {http.MethodGet},
	"/api/v1/fulfillment/cases/{case_id}/history": {http.MethodGet},
	// Slice 11 controlled non-financial writes: only the evidenced safe
	// commands accept POST; every other path stays GET only. There is no
	// invoice-issue, return, release, reset, transfer, allocation, ready, HR
	// final-apply or generic state-transition route.
	"/api/v1/cases/{case_id}/contact-outcomes":   {http.MethodGet, http.MethodPost},
	"/api/v1/hr/requests":                        {http.MethodGet, http.MethodPost},
	"/api/v1/hr/requests/{request_id}":           {http.MethodGet},
	"/api/v1/hr/requests/{request_id}/decisions": {http.MethodPost},
	"/api/v1/writeback/summary":                  {http.MethodGet},
	"/api/v1/writeback/commands":                 {http.MethodGet},
	"/api/v1/writeback/commands/{command_id}":    {http.MethodGet},
	// Slice 12 controlled financial writes (payments) and the read-only
	// Finance review queue.
	"/api/v1/payments/uploads":                          {http.MethodPost},
	"/api/v1/payments/invoices/{invoice_id}/submission": {http.MethodGet},
	"/api/v1/payments/invoices/{invoice_id}/batches":    {http.MethodPost},
	"/api/v1/payments/batches/{batch_id}":               {http.MethodGet},
	"/api/v1/payments/evidence":                         {http.MethodGet},
	"/api/v1/finance/review/attempts":                   {http.MethodGet},
	"/api/v1/finance/review/attempts/{attempt_id}":      {http.MethodGet},
	"/api/v1/invoice-review":                            {http.MethodGet},
	"/api/v1/invoice-review/{invoice_id}":               {http.MethodGet},
}

// ContractQuery is the allowlist of query parameters per Slice 4 path. A
// parameter outside it is a 400 CONTRACT_INVALID problem rather than being
// ignored (Architecture V1 §23). TestQueryAllowlistMatchesContract keeps it
// equal to the contract.
var ContractQuery = map[string][]string{
	"/api/v1/auth/login":                                    {},
	"/api/v1/auth/logout":                                   {},
	"/api/v1/auth/logout-all":                               {},
	"/api/v1/identity/users":                                {"q", "presence", "sort", "cursor", "limit"},
	"/api/v1/identity/users/{user_id}":                      {},
	"/api/v1/organization/persons":                          {"q", "employment_status", "position_id", "level_id", "org_unit_id", "is_active", "has_findings", "sort", "cursor", "limit"},
	"/api/v1/organization/persons/{person_id}":              {},
	"/api/v1/organization/positions":                        {"cursor", "limit"},
	"/api/v1/organization/levels":                           {"cursor", "limit"},
	"/api/v1/organization/org-units":                        {"kind", "cursor", "limit"},
	"/api/v1/organization/assignments":                      {"person_id", "relation", "is_current", "relationship_kind", "has_findings", "cursor", "limit"},
	"/api/v1/organization/hierarchy-findings":               {"kind", "severity", "subject_type", "person_id", "cursor", "limit"},
	"/api/v1/authorization/effective-access":                {},
	"/api/v1/customers":                                     {"state", "has_conflict", "presence", "cursor", "limit"},
	"/api/v1/customers/{customer_id}":                       {},
	"/api/v1/customers/{customer_id}/contact-points":        {},
	"/api/v1/customers/{customer_id}/evidence":              {"strength", "cursor", "limit"},
	"/api/v1/customer-identity/merge-cases":                 {"state", "evidence_strength", "reason", "customer_id", "cursor", "limit"},
	"/api/v1/customer-identity/merge-cases/{merge_case_id}": {},
	"/api/v1/customer-identity/resolution-report":           {},
	"/api/v1/cases":                                         {"workflow_type", "state_mapping", "customer_link_state", "presence", "cursor", "limit"},
	"/api/v1/cases/{case_id}":                               {},
	"/api/v1/cases/{case_id}/history":                       {"cursor", "limit"},
	"/api/v1/cases/{case_id}/attribution":                   {},
	"/api/v1/custody/assignments":                           {"parity", "derivation", "presence", "cursor", "limit"},
	"/api/v1/custody/cases/{case_id}":                       {},
	"/api/v1/sales/coverage-report":                         {},
	"/api/v1/sales/custody-parity":                          {},
	"/api/v1/invoices":                                      {"presence", "customer_link_state", "case_link_state", "has_unknown_status", "cursor", "limit"},
	"/api/v1/invoices/{invoice_id}":                         {},
	"/api/v1/invoices/{invoice_id}/items":                   {"cursor", "limit"},
	"/api/v1/invoices/{invoice_id}/stages":                  {"cursor", "limit"},
	"/api/v1/invoices/{invoice_id}/totals":                  {},
	"/api/v1/invoices/{invoice_id}/payments":                {"cursor", "limit"},
	"/api/v1/invoices/{invoice_id}/receipts":                {"cursor", "limit"},
	"/api/v1/payments/attempts":                             {"channel", "state_mapping", "state", "approval_provenance", "presence", "invoice_link_state", "cursor", "limit"},
	"/api/v1/payments/attempts/{attempt_id}":                {},
	"/api/v1/payments/attempts/{attempt_id}/history":        {"cursor", "limit"},
	"/api/v1/payments/receipts/{receipt_id}":                {},
	"/api/v1/billing/reconciliation-report":                 {},
	"/api/v1/reporting/metrics":                             {},
	"/api/v1/reporting/metrics/{metric_id}":                 {"time_basis", "from", "to", "as_of", "run_id"},
	"/api/v1/reports/cases":                                 {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "workflow_type", "state_mapping", "customer_link_state", "original_owner_resolution", "cursor", "limit"},
	"/api/v1/reports/cases/{case_id}":                       {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "workflow_type", "state_mapping", "customer_link_state", "original_owner_resolution"},
	"/api/v1/reports/custody":                               {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "parity", "derivation", "cursor", "limit"},
	"/api/v1/reports/custody/{case_id}":                     {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "parity", "derivation"},
	"/api/v1/reports/invoices":                              {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "customer_link_state", "case_link_state", "has_unknown_status", "main_mapping", "total_class", "unit", "cursor", "limit"},
	"/api/v1/reports/invoices/{invoice_id}":                 {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "customer_link_state", "case_link_state", "has_unknown_status", "main_mapping", "total_class", "unit"},
	"/api/v1/reports/payments":                              {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "channel", "state", "state_mapping", "approval_provenance", "invoice_link_state", "unit", "cursor", "limit"},
	"/api/v1/reports/payments/{attempt_id}":                 {"scope", "time_basis", "from", "to", "as_of", "run_id", "presence", "channel", "state", "state_mapping", "approval_provenance", "invoice_link_state", "unit"},
	"/api/v1/mis/overview":                                  {},
	"/api/v1/mis/source-quality":                            {},
	"/api/v1/mis/reconciliation":                            {"cursor", "limit"},
	"/api/v1/finance/shadow-runs":                           {"cursor", "limit"},
	"/api/v1/finance/shadow-runs/{run_id}":                  {},
	"/api/v1/finance/accounts":                              {"run_id", "account_category", "cursor", "limit"},
	"/api/v1/finance/accounts/{account_id}":                 {"run_id"},
	"/api/v1/finance/ledger":                                {"run_id", "basis", "unit", "state", "source_type", "account_category", "effective_from", "effective_to", "cursor", "limit"},
	"/api/v1/finance/ledger/{entry_id}":                     {"run_id"},
	"/api/v1/finance/balances":                              {"run_id", "basis", "unit", "account_category", "state", "cursor", "limit"},
	"/api/v1/finance/reconciliation":                        {"run_id", "basis", "unit", "state", "subject_type", "cursor", "limit"},
	// Slice 11: commands take no query parameters; reads are bounded.
	"/api/v1/cases/{case_id}/contact-outcomes":          {},
	"/api/v1/hr/requests":                               {"cursor", "limit"},
	"/api/v1/hr/requests/{request_id}":                  {},
	"/api/v1/hr/requests/{request_id}/decisions":        {},
	"/api/v1/writeback/summary":                         {},
	"/api/v1/writeback/commands":                        {"state", "cursor", "limit"},
	"/api/v1/writeback/commands/{command_id}":           {},
	"/api/v1/payments/uploads":                          {},
	"/api/v1/payments/invoices/{invoice_id}/submission": {},
	"/api/v1/payments/invoices/{invoice_id}/batches":    {},
	"/api/v1/payments/batches/{batch_id}":               {},
	"/api/v1/payments/evidence":                         {},
	"/api/v1/finance/review/attempts":                   {"cursor", "limit"},
	"/api/v1/finance/review/attempts/{attempt_id}":      {},
	// Slice 10: no phone (or any contact) parameter exists on any path.
	"/api/v1/biawin/subscriptions":                                      {"run_id", "state", "customer_link_state", "invoice_link_state", "activation_evidence", "product", "cursor", "limit"},
	"/api/v1/biawin/subscriptions/{subscription_id}":                    {"run_id"},
	"/api/v1/biawin/subscriptions/{subscription_id}/items":              {"run_id", "cursor", "limit"},
	"/api/v1/biawin/parity":                                             {"run_id", "check_id", "result", "cursor", "limit"},
	"/api/v1/commission/overview":                                       {"run_id"},
	"/api/v1/fulfillment/work-cases":                                    {"service_key", "state", "subscription_id", "cursor", "limit"},
	"/api/v1/fulfillment/work-cases/{case_id}":                          {},
	"/api/v1/fulfillment/work-cases/{case_id}/assignment":               {},
	"/api/v1/fulfillment/work-cases/{case_id}/operational-actions":      {},
	"/api/v1/fulfillment/work-cases/{case_id}/evidence-records":         {},
	"/api/v1/fulfillment/work-cases/{case_id}/outcome":                  {},
	"/api/v1/biawin/activation-decisions":                               {"decision", "limit"},
	"/api/v1/integration/provider-operations":                           {"state", "limit"},
	"/api/v1/integration/provider-operations/{operation_id}/resolution": {},
	"/api/v1/commission/runs":                                           {"limit"},
	"/api/v1/commission/calculations":                                   {"run_id", "parity_class", "engine", "cursor", "limit"},
	"/api/v1/commission/findings":                                       {"run_id", "kind", "cursor", "limit"},
	"/api/v1/commission/policies":                                       {"cursor", "limit"},
	"/api/v1/commission/historical-runs":                                {"run_id"},
	"/api/v1/commission/me/earnings":                                    {"run_id", "cursor", "limit"},
	"/api/v1/fulfillment/cases":                                         {"run_id", "kind", "status", "subscription_id", "item_id", "assignee", "cursor", "limit"},
	"/api/v1/fulfillment/cases/{case_id}":                               {"run_id"},
	"/api/v1/fulfillment/cases/{case_id}/history":                       {"run_id", "cursor", "limit"},
	"/api/v1/invoice-review":                                            {"presence", "customer_link_state", "case_link_state", "has_unknown_status", "total_class", "unit", "cursor", "limit"},
	"/api/v1/invoice-review/{invoice_id}":                               {},
}

// CustomersAPI is the Slice 5 customer identity read surface (customers
// module, composed by cmd/crm-api). Without one, every operation answers
// BLOCKED / FEATURE_DISABLED.
type CustomersAPI interface {
	ListCustomers(ctx context.Context, request oapi.ListCustomersRequestObject) (oapi.ListCustomersResponseObject, error)
	GetCustomer(ctx context.Context, request oapi.GetCustomerRequestObject) (oapi.GetCustomerResponseObject, error)
	ListCustomerContactPoints(ctx context.Context, request oapi.ListCustomerContactPointsRequestObject) (oapi.ListCustomerContactPointsResponseObject, error)
	ListCustomerEvidence(ctx context.Context, request oapi.ListCustomerEvidenceRequestObject) (oapi.ListCustomerEvidenceResponseObject, error)
	ListMergeCases(ctx context.Context, request oapi.ListMergeCasesRequestObject) (oapi.ListMergeCasesResponseObject, error)
	GetMergeCase(ctx context.Context, request oapi.GetMergeCaseRequestObject) (oapi.GetMergeCaseResponseObject, error)
	GetCustomerResolutionReport(ctx context.Context, request oapi.GetCustomerResolutionReportRequestObject) (oapi.GetCustomerResolutionReportResponseObject, error)
	// CustomersEnabled reports the Slice 5 flag for the session navigation hint.
	CustomersEnabled(ctx context.Context) bool
}

// SalesAPI is the Slice 6 sales read surface (cases and custody modules,
// composed by cmd/crm-api). Without one, every operation answers
// BLOCKED / FEATURE_DISABLED. It has no write operation of any kind.
type SalesAPI interface {
	ListCases(ctx context.Context, request oapi.ListCasesRequestObject) (oapi.ListCasesResponseObject, error)
	GetCase(ctx context.Context, request oapi.GetCaseRequestObject) (oapi.GetCaseResponseObject, error)
	ListCaseHistory(ctx context.Context, request oapi.ListCaseHistoryRequestObject) (oapi.ListCaseHistoryResponseObject, error)
	ListCaseAttribution(ctx context.Context, request oapi.ListCaseAttributionRequestObject) (oapi.ListCaseAttributionResponseObject, error)
	ListCustodyAssignments(ctx context.Context, request oapi.ListCustodyAssignmentsRequestObject) (oapi.ListCustodyAssignmentsResponseObject, error)
	GetCaseCustody(ctx context.Context, request oapi.GetCaseCustodyRequestObject) (oapi.GetCaseCustodyResponseObject, error)
	GetSalesCoverageReport(ctx context.Context, request oapi.GetSalesCoverageReportRequestObject) (oapi.GetSalesCoverageReportResponseObject, error)
	GetCustodyParityReport(ctx context.Context, request oapi.GetCustodyParityReportRequestObject) (oapi.GetCustodyParityReportResponseObject, error)
	// SalesEnabled reports the Slice 6 flag for the session navigation hint.
	SalesEnabled(ctx context.Context) bool
}

// BillingAPI is the Slice 7 billing + payments read surface (billing and
// payments modules, composed by cmd/crm-api). Without one, every operation
// answers BLOCKED / FEATURE_DISABLED. It has no write operation of any kind.
type BillingAPI interface {
	ListInvoices(ctx context.Context, request oapi.ListInvoicesRequestObject) (oapi.ListInvoicesResponseObject, error)
	GetInvoice(ctx context.Context, request oapi.GetInvoiceRequestObject) (oapi.GetInvoiceResponseObject, error)
	ListInvoiceItems(ctx context.Context, request oapi.ListInvoiceItemsRequestObject) (oapi.ListInvoiceItemsResponseObject, error)
	ListInvoiceStages(ctx context.Context, request oapi.ListInvoiceStagesRequestObject) (oapi.ListInvoiceStagesResponseObject, error)
	GetInvoiceTotals(ctx context.Context, request oapi.GetInvoiceTotalsRequestObject) (oapi.GetInvoiceTotalsResponseObject, error)
	ListInvoicePaymentAttempts(ctx context.Context, request oapi.ListInvoicePaymentAttemptsRequestObject) (oapi.ListInvoicePaymentAttemptsResponseObject, error)
	ListInvoiceReceipts(ctx context.Context, request oapi.ListInvoiceReceiptsRequestObject) (oapi.ListInvoiceReceiptsResponseObject, error)
	ListPaymentAttempts(ctx context.Context, request oapi.ListPaymentAttemptsRequestObject) (oapi.ListPaymentAttemptsResponseObject, error)
	GetPaymentAttempt(ctx context.Context, request oapi.GetPaymentAttemptRequestObject) (oapi.GetPaymentAttemptResponseObject, error)
	ListPaymentAttemptHistory(ctx context.Context, request oapi.ListPaymentAttemptHistoryRequestObject) (oapi.ListPaymentAttemptHistoryResponseObject, error)
	GetReceiptReference(ctx context.Context, request oapi.GetReceiptReferenceRequestObject) (oapi.GetReceiptReferenceResponseObject, error)
	GetBillingReconciliationReport(ctx context.Context, request oapi.GetBillingReconciliationReportRequestObject) (oapi.GetBillingReconciliationReportResponseObject, error)
	// BillingEnabled reports the Slice 7 flag for the session navigation hint.
	BillingEnabled(ctx context.Context) bool
}

// ReportingAPI is the Slice 8 MIS / reporting read surface (reporting
// module, composed by cmd/crm-api). Without one, every operation answers
// BLOCKED / FEATURE_DISABLED. It has no write or export operation.
type ReportingAPI interface {
	ListReportingMetrics(ctx context.Context, request oapi.ListReportingMetricsRequestObject) (oapi.ListReportingMetricsResponseObject, error)
	GetReportingMetric(ctx context.Context, request oapi.GetReportingMetricRequestObject) (oapi.GetReportingMetricResponseObject, error)
	GetCaseReport(ctx context.Context, request oapi.GetCaseReportRequestObject) (oapi.GetCaseReportResponseObject, error)
	GetCaseReportRow(ctx context.Context, request oapi.GetCaseReportRowRequestObject) (oapi.GetCaseReportRowResponseObject, error)
	GetCustodyReport(ctx context.Context, request oapi.GetCustodyReportRequestObject) (oapi.GetCustodyReportResponseObject, error)
	GetCustodyReportRow(ctx context.Context, request oapi.GetCustodyReportRowRequestObject) (oapi.GetCustodyReportRowResponseObject, error)
	GetInvoiceReport(ctx context.Context, request oapi.GetInvoiceReportRequestObject) (oapi.GetInvoiceReportResponseObject, error)
	GetInvoiceReportRow(ctx context.Context, request oapi.GetInvoiceReportRowRequestObject) (oapi.GetInvoiceReportRowResponseObject, error)
	GetPaymentReport(ctx context.Context, request oapi.GetPaymentReportRequestObject) (oapi.GetPaymentReportResponseObject, error)
	GetPaymentReportRow(ctx context.Context, request oapi.GetPaymentReportRowRequestObject) (oapi.GetPaymentReportRowResponseObject, error)
	GetMisOverview(ctx context.Context, request oapi.GetMisOverviewRequestObject) (oapi.GetMisOverviewResponseObject, error)
	GetMisSourceQuality(ctx context.Context, request oapi.GetMisSourceQualityRequestObject) (oapi.GetMisSourceQualityResponseObject, error)
	GetMisReconciliation(ctx context.Context, request oapi.GetMisReconciliationRequestObject) (oapi.GetMisReconciliationResponseObject, error)
	ListInvoiceReview(ctx context.Context, request oapi.ListInvoiceReviewRequestObject) (oapi.ListInvoiceReviewResponseObject, error)
	GetInvoiceReview(ctx context.Context, request oapi.GetInvoiceReviewRequestObject) (oapi.GetInvoiceReviewResponseObject, error)
	// ReportingEnabled reports the Slice 8 flag for the session navigation hint.
	ReportingEnabled(ctx context.Context) bool
}

// FinanceAPI is the Slice 9 finance SHADOW ledger read surface (finance
// module, composed by cmd/crm-api). Without one, every operation answers
// BLOCKED / FEATURE_DISABLED. It has no write operation of any kind.
type FinanceAPI interface {
	ListFinanceShadowRuns(ctx context.Context, request oapi.ListFinanceShadowRunsRequestObject) (oapi.ListFinanceShadowRunsResponseObject, error)
	GetFinanceShadowRun(ctx context.Context, request oapi.GetFinanceShadowRunRequestObject) (oapi.GetFinanceShadowRunResponseObject, error)
	ListFinanceAccounts(ctx context.Context, request oapi.ListFinanceAccountsRequestObject) (oapi.ListFinanceAccountsResponseObject, error)
	GetFinanceAccount(ctx context.Context, request oapi.GetFinanceAccountRequestObject) (oapi.GetFinanceAccountResponseObject, error)
	ListFinanceLedger(ctx context.Context, request oapi.ListFinanceLedgerRequestObject) (oapi.ListFinanceLedgerResponseObject, error)
	GetFinanceLedgerEntry(ctx context.Context, request oapi.GetFinanceLedgerEntryRequestObject) (oapi.GetFinanceLedgerEntryResponseObject, error)
	ListFinanceBalances(ctx context.Context, request oapi.ListFinanceBalancesRequestObject) (oapi.ListFinanceBalancesResponseObject, error)
	ListFinanceReconciliation(ctx context.Context, request oapi.ListFinanceReconciliationRequestObject) (oapi.ListFinanceReconciliationResponseObject, error)
	// FinanceEnabled reports the Slice 9 flag for the session navigation hint.
	FinanceEnabled(ctx context.Context) bool
}

// OperationsAPI is the Slice 14 controlled fulfillment / Biawin surface
// (fulfillment, biawin and integration modules, composed by cmd/crm-api).
// Without one, every operation answers BLOCKED / FEATURE_DISABLED.
type OperationsAPI interface {
	ListFulfillmentWorkCases(ctx context.Context, request oapi.ListFulfillmentWorkCasesRequestObject) (oapi.ListFulfillmentWorkCasesResponseObject, error)
	GetFulfillmentWorkCase(ctx context.Context, request oapi.GetFulfillmentWorkCaseRequestObject) (oapi.GetFulfillmentWorkCaseResponseObject, error)
	AssignFulfillmentWorkCase(ctx context.Context, request oapi.AssignFulfillmentWorkCaseRequestObject) (oapi.AssignFulfillmentWorkCaseResponseObject, error)
	RecordFulfillmentWorkAction(ctx context.Context, request oapi.RecordFulfillmentWorkActionRequestObject) (oapi.RecordFulfillmentWorkActionResponseObject, error)
	AttachFulfillmentWorkEvidence(ctx context.Context, request oapi.AttachFulfillmentWorkEvidenceRequestObject) (oapi.AttachFulfillmentWorkEvidenceResponseObject, error)
	CompleteFulfillmentWorkCase(ctx context.Context, request oapi.CompleteFulfillmentWorkCaseRequestObject) (oapi.CompleteFulfillmentWorkCaseResponseObject, error)
	ListBiawinActivationDecisions(ctx context.Context, request oapi.ListBiawinActivationDecisionsRequestObject) (oapi.ListBiawinActivationDecisionsResponseObject, error)
	ListProviderOperations(ctx context.Context, request oapi.ListProviderOperationsRequestObject) (oapi.ListProviderOperationsResponseObject, error)
	ResolveProviderOperation(ctx context.Context, request oapi.ResolveProviderOperationRequestObject) (oapi.ResolveProviderOperationResponseObject, error)
	// OperationsEnabled reports the Slice 14 read flag for the session navigation hint.
	OperationsEnabled(ctx context.Context) bool
}

// CommissionAPI is the Slice 13 commission controlled SHADOW read surface
// (commission module, composed by cmd/crm-api). GET only. Without one,
// every operation answers BLOCKED / FEATURE_DISABLED.
type CommissionAPI interface {
	GetCommissionOverview(ctx context.Context, request oapi.GetCommissionOverviewRequestObject) (oapi.GetCommissionOverviewResponseObject, error)
	ListCommissionRuns(ctx context.Context, request oapi.ListCommissionRunsRequestObject) (oapi.ListCommissionRunsResponseObject, error)
	ListCommissionCalculations(ctx context.Context, request oapi.ListCommissionCalculationsRequestObject) (oapi.ListCommissionCalculationsResponseObject, error)
	ListCommissionFindings(ctx context.Context, request oapi.ListCommissionFindingsRequestObject) (oapi.ListCommissionFindingsResponseObject, error)
	ListCommissionPolicies(ctx context.Context, request oapi.ListCommissionPoliciesRequestObject) (oapi.ListCommissionPoliciesResponseObject, error)
	ListCommissionHistoricalRuns(ctx context.Context, request oapi.ListCommissionHistoricalRunsRequestObject) (oapi.ListCommissionHistoricalRunsResponseObject, error)
	ListMyCommissionEarnings(ctx context.Context, request oapi.ListMyCommissionEarningsRequestObject) (oapi.ListMyCommissionEarningsResponseObject, error)
	// CommissionEnabled reports the Slice 13 flag for the session navigation hint.
	CommissionEnabled(ctx context.Context) bool
}

// BiawinAPI is the Slice 10 Biawin / fulfillment READ shadow surface
// (biawin and fulfillment modules, composed by cmd/crm-api). Without one,
// every operation answers BLOCKED / FEATURE_DISABLED. It has no write,
// activation, provider, shipping, assignment or completion operation.
type BiawinAPI interface {
	ListBiawinSubscriptions(ctx context.Context, request oapi.ListBiawinSubscriptionsRequestObject) (oapi.ListBiawinSubscriptionsResponseObject, error)
	GetBiawinSubscription(ctx context.Context, request oapi.GetBiawinSubscriptionRequestObject) (oapi.GetBiawinSubscriptionResponseObject, error)
	ListBiawinSubscriptionItems(ctx context.Context, request oapi.ListBiawinSubscriptionItemsRequestObject) (oapi.ListBiawinSubscriptionItemsResponseObject, error)
	GetBiawinParity(ctx context.Context, request oapi.GetBiawinParityRequestObject) (oapi.GetBiawinParityResponseObject, error)
	ListFulfillmentCases(ctx context.Context, request oapi.ListFulfillmentCasesRequestObject) (oapi.ListFulfillmentCasesResponseObject, error)
	GetFulfillmentCase(ctx context.Context, request oapi.GetFulfillmentCaseRequestObject) (oapi.GetFulfillmentCaseResponseObject, error)
	ListFulfillmentCaseHistory(ctx context.Context, request oapi.ListFulfillmentCaseHistoryRequestObject) (oapi.ListFulfillmentCaseHistoryResponseObject, error)
	// BiawinEnabled reports the Slice 10 flag for the session navigation hint.
	BiawinEnabled(ctx context.Context) bool
}

// ContractBody is the field scope of every command body: top-level fields
// outside the list are refused (400) before any handler runs, so a field
// outside the command's capability can never be smuggled in.
var ContractBody = map[string][]string{
	"/api/v1/auth/login":                                                {"login", "password"},
	"/api/v1/cases/{case_id}/contact-outcomes":                          {"outcome", "reason", "callback_at"},
	"/api/v1/hr/requests":                                               {"request_type", "subject_user_id", "target_parent_user_id", "reason"},
	"/api/v1/hr/requests/{request_id}/decisions":                        {"decision", "notes"},
	"/api/v1/payments/invoices/{invoice_id}/batches":                    {"stage_no", "entry_mode", "items"},
	"/api/v1/fulfillment/work-cases/{case_id}/assignment":               {"assignee_user_id"},
	"/api/v1/fulfillment/work-cases/{case_id}/operational-actions":      {"action", "follow_up_at", "note"},
	"/api/v1/fulfillment/work-cases/{case_id}/evidence-records":         {"evidence_type", "content_sha256", "label"},
	"/api/v1/fulfillment/work-cases/{case_id}/outcome":                  {"outcome", "reason"},
	"/api/v1/integration/provider-operations/{operation_id}/resolution": {"present", "external_ref", "evidence_note"},
}

// ContractBodyNested extends the field scope into arrays of objects: the
// key is a dotted path of array fields, the value the allowed fields of each
// element. A path present in the body must hold an array of objects.
var ContractBodyNested = map[string]map[string][]string{
	"/api/v1/payments/invoices/{invoice_id}/batches": {
		"items":          {"item_key", "amount", "unit", "receipts"},
		"items.receipts": {"upload_id", "content_sha256"},
	},
}

// ContractBodyLimit raises the body limit of a raw-upload path above the
// server default (a receipt is at most 5 MiB; Architecture V1 §15.1).
var ContractBodyLimit = map[string]int64{"/api/v1/payments/uploads": 5<<20 + 1}

// PaymentsAPI is the Slice 12 payments write surface and the read-only
// Finance review queue.
type PaymentsAPI interface {
	StagePaymentUpload(ctx context.Context, request oapi.StagePaymentUploadRequestObject) (oapi.StagePaymentUploadResponseObject, error)
	GetPaymentSubmission(ctx context.Context, request oapi.GetPaymentSubmissionRequestObject) (oapi.GetPaymentSubmissionResponseObject, error)
	SubmitPaymentBatch(ctx context.Context, request oapi.SubmitPaymentBatchRequestObject) (oapi.SubmitPaymentBatchResponseObject, error)
	GetPaymentBatch(ctx context.Context, request oapi.GetPaymentBatchRequestObject) (oapi.GetPaymentBatchResponseObject, error)
	GetPaymentEvidence(ctx context.Context, request oapi.GetPaymentEvidenceRequestObject) (oapi.GetPaymentEvidenceResponseObject, error)
	ListFinanceReviewQueue(ctx context.Context, request oapi.ListFinanceReviewQueueRequestObject) (oapi.ListFinanceReviewQueueResponseObject, error)
	GetFinanceReviewAttempt(ctx context.Context, request oapi.GetFinanceReviewAttemptRequestObject) (oapi.GetFinanceReviewAttemptResponseObject, error)
	PaymentSubmissionEnabled(ctx context.Context) bool
	FinanceReviewEnabled(ctx context.Context) bool
}

// CasesWriteAPI is the Slice 11 contact outcome surface (cases module).
type CasesWriteAPI interface {
	GetCaseContactOutcomes(ctx context.Context, request oapi.GetCaseContactOutcomesRequestObject) (oapi.GetCaseContactOutcomesResponseObject, error)
	RecordContactOutcome(ctx context.Context, request oapi.RecordContactOutcomeRequestObject) (oapi.RecordContactOutcomeResponseObject, error)
	ContactOutcomeWriteEnabled(ctx context.Context) bool
}

// HRRequestAPI is the Slice 11 HR change request surface (organization).
type HRRequestAPI interface {
	ListHRRequests(ctx context.Context, request oapi.ListHRRequestsRequestObject) (oapi.ListHRRequestsResponseObject, error)
	GetHRRequest(ctx context.Context, request oapi.GetHRRequestRequestObject) (oapi.GetHRRequestResponseObject, error)
	CreateHRRequest(ctx context.Context, request oapi.CreateHRRequestRequestObject) (oapi.CreateHRRequestResponseObject, error)
	DecideHRRequest(ctx context.Context, request oapi.DecideHRRequestRequestObject) (oapi.DecideHRRequestResponseObject, error)
	HRRequestWriteEnabled(ctx context.Context) bool
}

// WritebackAPI is the Slice 11 write-back diagnostics surface (integration).
type WritebackAPI interface {
	GetWritebackSummary(ctx context.Context, request oapi.GetWritebackSummaryRequestObject) (oapi.GetWritebackSummaryResponseObject, error)
	ListWritebackCommands(ctx context.Context, request oapi.ListWritebackCommandsRequestObject) (oapi.ListWritebackCommandsResponseObject, error)
	GetWritebackCommand(ctx context.Context, request oapi.GetWritebackCommandRequestObject) (oapi.GetWritebackCommandResponseObject, error)
	WritebackEnabled(ctx context.Context) bool
}

// Surfaces are the optional business read surfaces; nil ones answer
// BLOCKED / FEATURE_DISABLED.
type Surfaces struct {
	// Metrics, when set, observes every request (R3).
	Metrics    *Metrics
	Read       ReadAPI
	Customers  CustomersAPI
	Sales      SalesAPI
	Billing    BillingAPI
	Reporting  ReportingAPI
	Finance    FinanceAPI
	Biawin     BiawinAPI
	Commission CommissionAPI
	Operations OperationsAPI
	// Auth is the R1 session authentication surface (nil: anonymous only).
	Auth      AuthAPI
	CaseWrite CasesWriteAPI
	HRRequest HRRequestAPI
	Writeback WritebackAPI
	Payments  PaymentsAPI
}

// ReadAPI is the Slice 4 read surface (identity, organization,
// authorization). Business modules implement it and cmd/crm-api composes
// it; platform code never imports a business module. Without one, every
// Slice 4 operation answers BLOCKED / FEATURE_DISABLED.
type ReadAPI interface {
	GetEffectiveAccess(ctx context.Context, request oapi.GetEffectiveAccessRequestObject) (oapi.GetEffectiveAccessResponseObject, error)
	ListIdentityUsers(ctx context.Context, request oapi.ListIdentityUsersRequestObject) (oapi.ListIdentityUsersResponseObject, error)
	GetIdentityUser(ctx context.Context, request oapi.GetIdentityUserRequestObject) (oapi.GetIdentityUserResponseObject, error)
	ListOrganizationPersons(ctx context.Context, request oapi.ListOrganizationPersonsRequestObject) (oapi.ListOrganizationPersonsResponseObject, error)
	GetOrganizationPerson(ctx context.Context, request oapi.GetOrganizationPersonRequestObject) (oapi.GetOrganizationPersonResponseObject, error)
	ListOrganizationPositions(ctx context.Context, request oapi.ListOrganizationPositionsRequestObject) (oapi.ListOrganizationPositionsResponseObject, error)
	ListOrganizationLevels(ctx context.Context, request oapi.ListOrganizationLevelsRequestObject) (oapi.ListOrganizationLevelsResponseObject, error)
	ListOrganizationOrgUnits(ctx context.Context, request oapi.ListOrganizationOrgUnitsRequestObject) (oapi.ListOrganizationOrgUnitsResponseObject, error)
	ListOrganizationAssignments(ctx context.Context, request oapi.ListOrganizationAssignmentsRequestObject) (oapi.ListOrganizationAssignmentsResponseObject, error)
	ListOrganizationHierarchyFindings(ctx context.Context, request oapi.ListOrganizationHierarchyFindingsRequestObject) (oapi.ListOrganizationHierarchyFindingsResponseObject, error)
	// HRReadEnabled reports the Slice 4 flag for the session navigation hint.
	HRReadEnabled(ctx context.Context) bool
}

// displayZone is the presentation default zone; never storage truth.
const displayZone = oapi.SessionContextDisplayZone("Asia/Tehran")

type api struct {
	ReadAPI
	CustomersAPI
	SalesAPI
	BillingAPI
	ReportingAPI
	FinanceAPI
	BiawinAPI
	CommissionAPI
	OperationsAPI
	CasesWriteAPI
	HRRequestAPI
	WritebackAPI
	PaymentsAPI
	AuthAPI
	clock    timeutil.Clock
	devAuth  bool
	sessions bool
}

var _ oapi.StrictServerInterface = (*api)(nil)

func (a *api) GetHealthLive(ctx context.Context, _ oapi.GetHealthLiveRequestObject) (oapi.GetHealthLiveResponseObject, error) {
	return oapi.GetHealthLive200JSONResponse{Body: oapi.HealthLive{Status: "live"}}, nil
}

// GetHealthReady: Slice 2 has no dependencies, so it is ready with no checks.
// Health is infrastructure status, deliberately not a business ResultEnvelope.
func (a *api) GetHealthReady(ctx context.Context, _ oapi.GetHealthReadyRequestObject) (oapi.GetHealthReadyResponseObject, error) {
	body := oapi.HealthReady{Status: "ready"}
	body.Checks = make([]struct {
		Name   string                       `json:"name"`
		Status oapi.HealthReadyChecksStatus `json:"status"`
	}, 0)
	return oapi.GetHealthReady200JSONResponse{Body: body}, nil
}

func (a *api) GetSession(ctx context.Context, _ oapi.GetSessionRequestObject) (oapi.GetSessionResponseObject, error) {
	outcome, err := httpx.Outcome(result.Success)
	if err != nil {
		return nil, err
	}
	data := oapi.SessionContext{
		Authenticated: false,
		AuthMode:      "none",
		DisplayZone:   displayZone,
		ServerTime:    a.clock.Now().Time(),
	}
	state := oapi.SessionContextSessionState(websession.StateFrom(ctx))
	data.SessionState = &state
	if a.sessions {
		data.AuthMode = "session"
		enabled := a.AuthAPI.LoginEnabled(ctx)
		data.LoginEnabled = &enabled
		if p, ok := websession.FromContext(ctx); ok {
			uid, err := uuid.Parse(p.UserID)
			if err != nil {
				return nil, err
			}
			data.Authenticated = true
			data.Session = &oapi.AuthSession{UserId: uid, DisplayName: p.DisplayName, CsrfToken: p.CSRF,
				IdleExpiresAt: p.IdleExpiresAt.UTC(), AbsoluteExpiresAt: p.AbsoluteExpiresAt.UTC()}
		}
	}
	if a.devAuth {
		data.AuthMode = "dev_stub"
		if p, ok := authstub.FromContext(ctx); ok {
			hints := append([]string(nil), p.CapabilityHints...)
			data.Authenticated = true
			data.CapabilityHints = &hints
			data.Principal = &struct {
				DisplayName string                                `json:"display_name"`
				PrincipalId string                                `json:"principal_id"`
				Synthetic   oapi.SessionContextPrincipalSynthetic `json:"synthetic"`
			}{DisplayName: p.DisplayName, PrincipalId: p.ID, Synthetic: true}
		}
	}
	data.Features = &struct {
		BiawinFulfillmentRead bool `json:"biawin_fulfillment_read"`
		BillingRead           bool `json:"billing_read"`
		CommissionRead        bool `json:"commission_read"`
		ContactOutcomeWrite   bool `json:"contact_outcome_write"`
		CustomersIdentity     bool `json:"customers_identity"`
		FinanceReviewRead     bool `json:"finance_review_read"`
		FinanceShadowRead     bool `json:"finance_shadow_read"`
		FulfillmentWork       bool `json:"fulfillment_work"`
		HrRead                bool `json:"hr_read"`
		HrRequestWrite        bool `json:"hr_request_write"`
		PaymentSubmission     bool `json:"payment_submission"`
		ReportingRead         bool `json:"reporting_read"`
		SalesRead             bool `json:"sales_read"`
		WritebackRead         bool `json:"writeback_read"`
	}{BillingRead: a.BillingAPI.BillingEnabled(ctx), CustomersIdentity: a.CustomersAPI.CustomersEnabled(ctx), HrRead: a.ReadAPI.HRReadEnabled(ctx),
		ReportingRead: a.ReportingAPI.ReportingEnabled(ctx), SalesRead: a.SalesAPI.SalesEnabled(ctx),
		FinanceShadowRead: a.FinanceAPI.FinanceEnabled(ctx), BiawinFulfillmentRead: a.BiawinAPI.BiawinEnabled(ctx),
		ContactOutcomeWrite: a.CasesWriteAPI.ContactOutcomeWriteEnabled(ctx), HrRequestWrite: a.HRRequestAPI.HRRequestWriteEnabled(ctx),
		WritebackRead: a.WritebackAPI.WritebackEnabled(ctx), PaymentSubmission: a.PaymentsAPI.PaymentSubmissionEnabled(ctx),
		FinanceReviewRead: a.PaymentsAPI.FinanceReviewEnabled(ctx), CommissionRead: a.CommissionAPI.CommissionEnabled(ctx),
		FulfillmentWork: a.OperationsAPI.OperationsEnabled(ctx)}
	return oapi.GetSession200JSONResponse{Body: oapi.SessionEnvelope{
		Outcome:       outcome,
		Reasons:       []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx),
		Data:          data,
	}}, nil
}

// NewHandler builds the full handler chain without a Slice 4 read surface.
func NewHandler(cfg Config, clock timeutil.Clock, logger *slog.Logger) (http.Handler, error) {
	return NewHandlerWith(cfg, clock, logger, nil, nil)
}

// NewHandlerWith builds the full handler chain:
// Correlation → Recover → LimitBody → (dev auth stub) → query allowlist → routes.
// read and customers may be nil (their operations are then BLOCKED /
// FEATURE_DISABLED).
func NewHandlerWith(cfg Config, clock timeutil.Clock, logger *slog.Logger, read ReadAPI, customers CustomersAPI) (http.Handler, error) {
	return NewHandlerWithSurfaces(cfg, clock, logger, Surfaces{Read: read, Customers: customers})
}

// NewHandlerWithSurfaces builds the full handler chain over every surface.
func NewHandlerWithSurfaces(cfg Config, clock timeutil.Clock, logger *slog.Logger, sf Surfaces) (http.Handler, error) {
	read, customers, sales, billing, rep, fin, bw := sf.Read, sf.Customers, sf.Sales, sf.Billing, sf.Reporting, sf.Finance, sf.Biawin
	cw, hrq, wbk, pay, com, ops := sf.CaseWrite, sf.HRRequest, sf.Writeback, sf.Payments, sf.Commission, sf.Operations
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	auth := sf.Auth
	if auth != nil && cfg.DevAuth {
		return nil, ErrAuthConflict
	}
	if auth != nil && cfg.Environment == authstub.Production && len(cfg.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("%w: %s is required for R1 sessions in production", ErrInvalidConfig, EnvAllowedOrigins)
	}
	sessions := auth != nil
	if auth == nil {
		auth = unconfiguredAuth{}
	}
	if clock == nil || logger == nil {
		return nil, ErrInvalidConfig
	}
	if read == nil {
		read = unconfigured{}
	}
	if customers == nil {
		customers = unconfiguredCustomers{}
	}
	if sales == nil {
		sales = unconfiguredSales{}
	}
	if billing == nil {
		billing = unconfiguredBilling{}
	}
	if rep == nil {
		rep = unconfiguredReporting{}
	}
	if fin == nil {
		fin = unconfiguredFinance{}
	}
	if bw == nil {
		bw = unconfiguredBiawin{}
	}
	if cw == nil {
		cw = unconfiguredCaseWrite{}
	}
	if hrq == nil {
		hrq = unconfiguredHRRequest{}
	}
	if wbk == nil {
		wbk = unconfiguredWriteback{}
	}
	if pay == nil {
		pay = unconfiguredPayments{}
	}
	if com == nil {
		com = unconfiguredCommission{}
	}
	if ops == nil {
		ops = unconfiguredOperations{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", fallback)
	strict := oapi.NewStrictHandlerWithOptions(&api{ReadAPI: read, CustomersAPI: customers, SalesAPI: sales, BillingAPI: billing, ReportingAPI: rep, FinanceAPI: fin, BiawinAPI: bw, CommissionAPI: com, OperationsAPI: ops, AuthAPI: auth, sessions: sessions, CasesWriteAPI: cw, HRRequestAPI: hrq, WritebackAPI: wbk, PaymentsAPI: pay, clock: clock, devAuth: cfg.DevAuth}, nil, oapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteProblem(w, r, httpx.ProblemBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("response error", "correlation_id", httpx.CorrelationID(r.Context()), "error", err)
			httpx.WriteProblem(w, r, httpx.ProblemInternal)
		},
	})
	var h http.Handler = oapi.HandlerWithOptions(strict, oapi.StdHTTPServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteProblem(w, r, httpx.ProblemBadRequest)
		},
	})
	h = queryAllowlist(h)
	h = bodyAllowlist(h)
	if cfg.DevAuth {
		stub, err := authstub.New(cfg.Environment)
		if err != nil {
			return nil, err
		}
		h = stub.Middleware(func(w http.ResponseWriter, r *http.Request, err error) {
			p := httpx.ProblemBadRequest
			p.Detail = "Unknown " + authstub.Header + " synthetic identity."
			httpx.WriteProblem(w, r, p)
		})(h)
	}
	if sessions {
		h = crossSiteGuard(cfg.AllowedOrigins, auth, logger)(h)
		h = sessionAuth(auth, logger)(h)
	}
	h = limitBody(cfg.MaxBodyBytes)(h)
	h = noStore(h)
	h = httpx.Recover(logger)(h)
	h = httpx.Correlation(h)
	if sf.Metrics != nil {
		h = sf.Metrics.Middleware(h)
	}
	return h, nil
}

// template returns the contract path template matching path, if any. When
// several match, the one with the most literal segments wins (deterministic).
func template(path string) (string, bool) {
	if _, ok := ContractPaths[path]; ok {
		return path, true
	}
	segs := strings.Split(path, "/")
	best, bestLiterals := "", -1
	for t := range ContractPaths {
		ts := strings.Split(t, "/")
		if len(ts) != len(segs) {
			continue
		}
		match, literals := true, 0
		for i := 0; i < len(ts) && match; i++ {
			switch {
			case ts[i] == segs[i]:
				literals++
			case strings.HasPrefix(ts[i], "{") && segs[i] != "":
			default:
				match = false
			}
		}
		if match && (literals > bestLiterals || literals == bestLiterals && t < best) {
			best, bestLiterals = t, literals
		}
	}
	return best, best != ""
}

// queryAllowlist rejects query parameters a Slice 4 operation does not
// declare, instead of silently ignoring them.
func queryAllowlist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t, ok := template(r.URL.Path); ok {
			if allowed, guarded := ContractQuery[t]; guarded {
				// A query the standard parser would partially drop (e.g. a ";")
				// is refused, never silently reduced to a default.
				q, err := url.ParseQuery(r.URL.RawQuery)
				if err != nil {
					p := httpx.ProblemBadRequest
					p.Detail = "Malformed query string."
					httpx.WriteProblem(w, r, p)
					return
				}
				for k := range q {
					if !slices.Contains(allowed, k) {
						p := httpx.ProblemBadRequest
						p.Detail = "Unknown query parameter " + strconv.Quote(k) + "."
						httpx.WriteProblem(w, r, p)
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// bodyAllowlist refuses a command body that is not a single JSON object of
// allowlisted top-level fields (ContractBody). It reads the bounded body
// (LimitBody runs first) and restores it for the handler.
func bodyAllowlist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if t, ok := template(r.URL.Path); ok {
				if allowed, guarded := ContractBody[t]; guarded {
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						httpx.WriteProblem(w, r, httpx.ProblemBadRequest)
						return
					}
					var fields map[string]json.RawMessage
					dec := json.NewDecoder(bytes.NewReader(raw))
					if err := dec.Decode(&fields); err != nil || fields == nil || dec.More() {
						p := httpx.ProblemBadRequest
						p.Detail = "The body must be one JSON object."
						httpx.WriteProblem(w, r, p)
						return
					}
					for k := range fields {
						if !slices.Contains(allowed, k) {
							p := httpx.ProblemBadRequest
							p.Detail = "Field " + strconv.Quote(k) + " is outside this command's field scope."
							httpx.WriteProblem(w, r, p)
							return
						}
					}
					if bad := nestedViolation(fields, ContractBodyNested[t]); bad != "" {
						p := httpx.ProblemBadRequest
						p.Detail = "Field " + strconv.Quote(bad) + " is outside this command's field scope."
						httpx.WriteProblem(w, r, p)
						return
					}
					r.Body = io.NopCloser(bytes.NewReader(raw))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// nestedViolation checks the nested field scope of a body; it returns the
// first offending dotted field, or "".
func nestedViolation(fields map[string]json.RawMessage, nested map[string][]string) string {
	var walk func(obj map[string]json.RawMessage, prefix string) string
	walk = func(obj map[string]json.RawMessage, prefix string) string {
		for k, raw := range obj {
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			allowed, ok := nested[path]
			if !ok {
				continue
			}
			var elems []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &elems); err != nil {
				return path
			}
			for _, e := range elems {
				if e == nil {
					return path
				}
				for f := range e {
					if !slices.Contains(allowed, f) {
						return path + "." + f
					}
				}
				if bad := walk(e, path); bad != "" {
					return bad
				}
			}
		}
		return ""
	}
	return walk(fields, "")
}

// limitBody applies the server body limit, raised only for the raw-upload
// paths listed in ContractBodyLimit.
func limitBody(maxBytes int64) func(http.Handler) http.Handler {
	def := httpx.LimitBody(maxBytes)
	return func(next http.Handler) http.Handler {
		limited := def(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if t, ok := template(r.URL.Path); ok && r.Method == http.MethodPost {
				if n, raised := ContractBodyLimit[t]; raised && n > maxBytes {
					httpx.LimitBody(n)(next).ServeHTTP(w, r)
					return
				}
			}
			limited.ServeHTTP(w, r)
		})
	}
}

// fallback answers every request no contract operation matched.
func fallback(w http.ResponseWriter, r *http.Request) {
	if t, ok := template(r.URL.Path); ok {
		methods := ContractPaths[t]
		for _, m := range methods {
			w.Header().Add("Allow", m)
		}
		httpx.WriteProblem(w, r, httpx.ProblemMethodNotAllowed)
		return
	}
	httpx.WriteProblem(w, r, httpx.ProblemNotFound)
}

// NewServer returns an *http.Server with the configured timeouts and limits.
func NewServer(cfg Config, clock timeutil.Clock, logger *slog.Logger) (*http.Server, error) {
	return NewServerWith(cfg, clock, logger, nil, nil)
}

// NewServerWith is NewServer with the Slice 4 and Slice 5 read surfaces (either may be nil).
func NewServerWith(cfg Config, clock timeutil.Clock, logger *slog.Logger, read ReadAPI, customers CustomersAPI) (*http.Server, error) {
	return NewServerWithSurfaces(cfg, clock, logger, Surfaces{Read: read, Customers: customers})
}

// NewServerWithSurfaces builds the server over every surface.
func NewServerWithSurfaces(cfg Config, clock timeutil.Clock, logger *slog.Logger, sf Surfaces) (*http.Server, error) {
	h, err := NewHandlerWithSurfaces(cfg, clock, logger, sf)
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}, nil
}

// Serve runs srv until ctx is cancelled, then shuts down gracefully within
// cfg.ShutdownTimeout. It returns nil after a clean shutdown.
func Serve(ctx context.Context, srv *http.Server, cfg Config, logger *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	logger.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
