import type { RouteObject } from 'react-router'
import type { AppConfig } from './config'
import { Shell, type DevNavItem } from './Shell'

/**
 * Every existing route (URLs unchanged since RM-E02; bookmarks and deep links
 * keep working) under the role-oriented Shell. Navigation links are NOT listed
 * here: the Shell builds them from navigation/destinations.ts, showing a
 * destination only when the server reports its feature on AND its capability
 * ALLOWED (effective-access). Routes are never guarded client-side: every page
 * renders the server's own BLOCKED / UNAUTHORIZED answer, so hiding a link is
 * presentation, not security. DEV-only routes are never mounted in production.
 */
export function buildRoutes(config: AppConfig): RouteObject[] {
  const devNav: DevNavItem[] = []
  const children: RouteObject[] = [
    { index: true, lazy: async () => ({ Component: (await import('../routes/HomePage')).HomePage }) },
    { path: 'system', lazy: async () => ({ Component: (await import('../routes/SystemPage')).SystemPage }) },
    { path: 'hr', lazy: async () => ({ Component: (await import('../modules/hr/pages/WorkforcePage')).WorkforcePage }) },
    { path: 'hr/people/:personId', lazy: async () => ({ Component: (await import('../modules/hr/pages/PersonPage')).PersonPage }) },
    { path: 'hr/structure', lazy: async () => ({ Component: (await import('../modules/hr/pages/StructurePage')).StructurePage }) },
    // RM-P04 HR operational workspace (read only; static paths precede none of the :params above).
    { path: 'hr/assignments', lazy: async () => ({ Component: (await import('../modules/hr/pages/AssignmentsPage')).AssignmentsPage }) },
    { path: 'hr/positions', lazy: async () => ({ Component: (await import('../modules/hr/pages/PositionsPage')).PositionsPage }) },
    { path: 'hr/access', lazy: async () => ({ Component: (await import('../modules/hr/pages/AccessPage')).AccessPage }) },
    { path: 'hr/exceptions', lazy: async () => ({ Component: (await import('../modules/hr/pages/ExceptionsPage')).ExceptionsPage }) },
    // Slice 5 customer identity (read-only). Static paths precede :customerId.
    { path: 'customers', lazy: async () => ({ Component: (await import('../modules/customers/pages/CustomersPage')).CustomersPage }) },
    { path: 'customers/merge-cases', lazy: async () => ({ Component: (await import('../modules/customers/pages/MergeCasesPage')).MergeCasesPage }) },
    { path: 'customers/identity-report', lazy: async () => ({ Component: (await import('../modules/customers/pages/IdentityReportPage')).IdentityReportPage }) },
    { path: 'customers/:customerId', lazy: async () => ({ Component: (await import('../modules/customers/pages/CustomerPage')).CustomerPage }) },
    // Slice 6 sales read (read-only; no action route).
    { path: 'sales/cases', lazy: async () => ({ Component: (await import('../modules/sales/pages/CasesPage')).CasesPage }) },
    { path: 'sales/cases/:caseId', lazy: async () => ({ Component: (await import('../modules/sales/pages/CasePage')).CasePage }) },
    { path: 'sales/custody', lazy: async () => ({ Component: (await import('../modules/sales/pages/CustodyPage')).CustodyPage }) },
    { path: 'sales/report', lazy: async () => ({ Component: (await import('../modules/sales/pages/SalesReportPage')).SalesReportPage }) },
    // Slice 7 billing + payments read (read-only; no action route).
    { path: 'billing/invoices', lazy: async () => ({ Component: (await import('../modules/billing/pages/InvoicesPage')).InvoicesPage }) },
    { path: 'billing/invoices/:invoiceId', lazy: async () => ({ Component: (await import('../modules/billing/pages/InvoicePage')).InvoicePage }) },
    { path: 'billing/attempts', lazy: async () => ({ Component: (await import('../modules/billing/pages/AttemptsPage')).AttemptsPage }) },
    { path: 'billing/report', lazy: async () => ({ Component: (await import('../modules/billing/pages/BillingReportPage')).BillingReportPage }) },
    // Slice 8 MIS / reporting (read-only). No Invoice Review page (D-24): its query surface is API only.
    // RM-P05 MIS operational workspace (read only): landing, case inventory, custody evidence, exceptions.
    { path: 'mis', lazy: async () => ({ Component: (await import('../modules/mis/pages/HomePage')).HomePage }) },
    { path: 'mis/cases', lazy: async () => ({ Component: (await import('../modules/mis/pages/InventoryPage')).InventoryPage }) },
    { path: 'mis/custody', lazy: async () => ({ Component: (await import('../modules/mis/pages/CustodyPage')).CustodyPage }) },
    { path: 'mis/custody/:caseId', lazy: async () => ({ Component: (await import('../modules/mis/pages/CaseCustodyPage')).CaseCustodyPage }) },
    { path: 'mis/exceptions', lazy: async () => ({ Component: (await import('../modules/mis/pages/ExceptionsPage')).ExceptionsPage }) },
    { path: 'mis/source-quality', lazy: async () => ({ Component: (await import('../modules/reporting/pages/SourceQualityPage')).SourceQualityPage }) },
    { path: 'mis/reconciliation', lazy: async () => ({ Component: (await import('../modules/reporting/pages/ReconciliationPage')).ReconciliationPage }) },
    { path: 'reports/:report', lazy: async () => ({ Component: (await import('../modules/reporting/pages/ReportPage')).ReportPage }) },
    // Slice 9 finance SHADOW ledger (read only; no mutation route or control).
    { path: 'finance', lazy: async () => ({ Component: (await import('../modules/finance/pages/FinanceDashboardPage')).FinanceDashboardPage }) },
    { path: 'finance/ledger', lazy: async () => ({ Component: (await import('../modules/finance/pages/LedgerPage')).LedgerPage }) },
    { path: 'finance/balances', lazy: async () => ({ Component: (await import('../modules/finance/pages/BalancesPage')).BalancesPage }) },
    { path: 'finance/reconciliation', lazy: async () => ({ Component: (await import('../modules/finance/pages/FinanceReconciliationPage')).FinanceReconciliationPage }) },
    // Slice 11 controlled writes (only evidenced commands; server-decided actions).
    { path: 'hr/requests', lazy: async () => ({ Component: (await import('../modules/writes/pages/HRRequestsPage')).HRRequestsPage }) },
    { path: 'hr/requests/:requestId', lazy: async () => ({ Component: (await import('../modules/writes/pages/HRRequestPage')).HRRequestPage }) },
    { path: 'writeback', lazy: async () => ({ Component: (await import('../modules/writes/pages/WritebackPage')).WritebackPage }) },
    // Slice 12 controlled financial writes (payments) and the read-only Finance review queue.
    { path: 'payments/batches/:batchId', lazy: async () => ({ Component: (await import('../modules/payments/pages/PaymentBatchPage')).PaymentBatchPage }) },
    { path: 'payments/evidence', lazy: async () => ({ Component: (await import('../modules/payments/pages/PaymentEvidencePage')).PaymentEvidencePage }) },
    { path: 'finance/review', lazy: async () => ({ Component: (await import('../modules/payments/pages/FinanceReviewPage')).FinanceReviewPage }) },
    { path: 'finance/review/:attemptId', lazy: async () => ({ Component: (await import('../modules/payments/pages/FinanceReviewAttemptPage')).FinanceReviewAttemptPage }) },
    // Slice 10 Biawin / fulfillment READ shadow (no activation, provider, portal or workflow control).
    { path: 'biawin/subscriptions', lazy: async () => ({ Component: (await import('../modules/biawin/pages/SubscriptionsPage')).SubscriptionsPage }) },
    { path: 'biawin/subscriptions/:subscriptionId', lazy: async () => ({ Component: (await import('../modules/biawin/pages/SubscriptionPage')).SubscriptionPage }) },
    { path: 'biawin/parity', lazy: async () => ({ Component: (await import('../modules/biawin/pages/ParityPage')).ParityPage }) },
    { path: 'fulfillment/cases', lazy: async () => ({ Component: (await import('../modules/biawin/pages/CasesPage')).CasesPage }) },
    { path: 'fulfillment/cases/:caseId', lazy: async () => ({ Component: (await import('../modules/biawin/pages/CasePage')).CasePage }) },
    // Slice 13 commission controlled SHADOW read (no calculate, approve, post, credit or settle control).
    { path: 'commission', lazy: async () => ({ Component: (await import('../modules/commission/pages/OverviewPage')).OverviewPage }) },
    { path: 'commission/calculations', lazy: async () => ({ Component: (await import('../modules/commission/pages/CalculationsPage')).CalculationsPage }) },
    { path: 'commission/policies', lazy: async () => ({ Component: (await import('../modules/commission/pages/PoliciesPage')).PoliciesPage }) },
    { path: 'commission/me', lazy: async () => ({ Component: (await import('../modules/commission/pages/EarningsPage')).EarningsPage }) },
    // Slice 14 controlled operations (SHADOW work cases; no activation, wallet charge or provider dispatch control).
    { path: 'operations/work', lazy: async () => ({ Component: (await import('../modules/operations/pages/WorkQueuePage')).WorkQueuePage }) },
    { path: 'operations/work/:caseId', lazy: async () => ({ Component: (await import('../modules/operations/pages/WorkCasePage')).WorkCasePage }) },
    { path: 'operations/activations', lazy: async () => ({ Component: (await import('../modules/operations/pages/ActivationDecisionsPage')).ActivationDecisionsPage }) },
    { path: 'operations/provider-operations', lazy: async () => ({ Component: (await import('../modules/operations/pages/ProviderOperationsPage')).ProviderOperationsPage }) },
  ]
  // import.meta.env.DEV is a build-time constant: production bundles drop this branch entirely.
  if (import.meta.env.DEV && config.devTools) {
    devNav.push({ to: '/dev/primitives', label: 'نمونه اجزا' })
    devNav.push({ to: '/dev/design-system', label: 'سامانه طراحی' })
    children.push({ path: 'dev/primitives', lazy: async () => ({ Component: (await import('../routes/dev/PrimitivesPage')).PrimitivesPage }) })
  }
  children.push({ path: '*', lazy: async () => ({ Component: (await import('../routes/NotFoundPage')).NotFoundPage }) })
  const routes: RouteObject[] = [{ path: '/', element: <Shell devNav={devNav} />, children }]
  // RM-E01 design-system showcase: DEV-only, synthetic, rendered outside the module Shell so it can
  // show the full shared shell (its own header, nav, main, drawer, bottom nav). Same build-time guard.
  if (import.meta.env.DEV && config.devTools) {
    routes.push({ path: '/dev/design-system', lazy: async () => ({ Component: (await import('../routes/dev/DesignSystemPage')).DesignSystemPage }) })
    // RM-E03 Seller V1 preview: DEV/TEST-only, synthetic fixtures, own preview shell. Development-preview
    // URLs, NOT approved business URLs; never added to the role navigation or the production bundle.
    const seller = async () => ({ Component: (await import('../modules/seller/SellerPreviewPage')).SellerPreviewPage })
    routes.push({ path: '/dev/seller', lazy: seller })
    routes.push({ path: '/dev/seller/:section', lazy: seller })
  }
  return routes
}
