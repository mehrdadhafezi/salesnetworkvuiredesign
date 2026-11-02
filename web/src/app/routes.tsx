import type { RouteObject } from 'react-router'
import type { AppConfig } from './config'
import { Shell, type NavItem } from './Shell'

/**
 * Shell routes (home, system status, 404), the Slice 4 HR read routes, the
 * Slice 5 customer identity read routes, and a
 * DEV-only primitives route that production builds never mount. Pages load
 * lazily. HR navigation is shown only when the server reports the HR read
 * feature enabled; the routes themselves render the server's BLOCKED /
 * UNAUTHORIZED answer, so hiding the link is a convenience, not security.
 */
export function buildRoutes(config: AppConfig): RouteObject[] {
  const nav: NavItem[] = [
    { to: '/', label: 'خانه' },
    { to: '/hr', label: 'نیروها', feature: 'hr_read' },
    { to: '/hr/structure', label: 'ساختار سازمانی', feature: 'hr_read' },
    { to: '/hr/exceptions', label: 'استثناهای سلسله‌مراتب', feature: 'hr_read' },
    { to: '/customers', label: 'مشتریان', feature: 'customers_identity' },
    { to: '/customers/merge-cases', label: 'پرونده‌های تطبیق', feature: 'customers_identity' },
    { to: '/customers/identity-report', label: 'گزارش تطبیق هویت', feature: 'customers_identity' },
    { to: '/sales/cases', label: 'پرونده‌های فروش', feature: 'sales_read' },
    { to: '/sales/custody', label: 'نگهداری پرونده‌ها', feature: 'sales_read' },
    { to: '/sales/report', label: 'گزارش پوشش فروش', feature: 'sales_read' },
    { to: '/billing/invoices', label: 'فاکتورها', feature: 'billing_read' },
    { to: '/billing/attempts', label: 'تلاش‌های پرداخت', feature: 'billing_read' },
    { to: '/billing/report', label: 'گزارش تطبیق صورت‌حساب', feature: 'billing_read' },
    { to: '/mis', label: 'امروز MIS', feature: 'reporting_read' },
    { to: '/mis/source-quality', label: 'کیفیت منبع', feature: 'reporting_read' },
    { to: '/mis/reconciliation', label: 'تطبیق و پوشش', feature: 'reporting_read' },
    { to: '/reports/cases', label: 'گزارش پرونده‌ها', feature: 'reporting_read' },
    { to: '/reports/custody', label: 'گزارش نگهداری', feature: 'reporting_read' },
    { to: '/reports/invoices', label: 'گزارش فاکتورها', feature: 'reporting_read' },
    { to: '/reports/payments', label: 'گزارش پرداخت‌ها', feature: 'reporting_read' },
    { to: '/finance', label: 'دفتر سایه مالی', feature: 'finance_shadow_read' },
    { to: '/finance/ledger', label: 'دفتر روزنامه سایه', feature: 'finance_shadow_read' },
    { to: '/finance/balances', label: 'مانده‌های سایه', feature: 'finance_shadow_read' },
    { to: '/finance/reconciliation', label: 'تطبیق سایه', feature: 'finance_shadow_read' },
    { to: '/biawin/subscriptions', label: 'اشتراک‌های بیاوین', feature: 'biawin_fulfillment_read' },
    { to: '/biawin/parity', label: 'تطبیق اشتراک‌های فعال', feature: 'biawin_fulfillment_read' },
    { to: '/fulfillment/cases', label: 'پرونده‌های اجرای خدمت', feature: 'biawin_fulfillment_read' },
    { to: '/system', label: 'وضعیت سامانه' },
  ]
  const children: RouteObject[] = [
    { index: true, lazy: async () => ({ Component: (await import('../routes/HomePage')).HomePage }) },
    { path: 'system', lazy: async () => ({ Component: (await import('../routes/SystemPage')).SystemPage }) },
    { path: 'hr', lazy: async () => ({ Component: (await import('../modules/hr/pages/WorkforcePage')).WorkforcePage }) },
    { path: 'hr/people/:personId', lazy: async () => ({ Component: (await import('../modules/hr/pages/PersonPage')).PersonPage }) },
    { path: 'hr/structure', lazy: async () => ({ Component: (await import('../modules/hr/pages/StructurePage')).StructurePage }) },
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
    { path: 'mis', lazy: async () => ({ Component: (await import('../modules/reporting/pages/MisOverviewPage')).MisOverviewPage }) },
    { path: 'mis/source-quality', lazy: async () => ({ Component: (await import('../modules/reporting/pages/SourceQualityPage')).SourceQualityPage }) },
    { path: 'mis/reconciliation', lazy: async () => ({ Component: (await import('../modules/reporting/pages/ReconciliationPage')).ReconciliationPage }) },
    { path: 'reports/:report', lazy: async () => ({ Component: (await import('../modules/reporting/pages/ReportPage')).ReportPage }) },
    // Slice 9 finance SHADOW ledger (read only; no mutation route or control).
    { path: 'finance', lazy: async () => ({ Component: (await import('../modules/finance/pages/FinanceDashboardPage')).FinanceDashboardPage }) },
    { path: 'finance/ledger', lazy: async () => ({ Component: (await import('../modules/finance/pages/LedgerPage')).LedgerPage }) },
    { path: 'finance/balances', lazy: async () => ({ Component: (await import('../modules/finance/pages/BalancesPage')).BalancesPage }) },
    { path: 'finance/reconciliation', lazy: async () => ({ Component: (await import('../modules/finance/pages/FinanceReconciliationPage')).FinanceReconciliationPage }) },
    // Slice 10 Biawin / fulfillment READ shadow (no activation, provider, portal or workflow control).
    { path: 'biawin/subscriptions', lazy: async () => ({ Component: (await import('../modules/biawin/pages/SubscriptionsPage')).SubscriptionsPage }) },
    { path: 'biawin/subscriptions/:subscriptionId', lazy: async () => ({ Component: (await import('../modules/biawin/pages/SubscriptionPage')).SubscriptionPage }) },
    { path: 'biawin/parity', lazy: async () => ({ Component: (await import('../modules/biawin/pages/ParityPage')).ParityPage }) },
    { path: 'fulfillment/cases', lazy: async () => ({ Component: (await import('../modules/biawin/pages/CasesPage')).CasesPage }) },
    { path: 'fulfillment/cases/:caseId', lazy: async () => ({ Component: (await import('../modules/biawin/pages/CasePage')).CasePage }) },
  ]
  // import.meta.env.DEV is a build-time constant: production bundles drop this branch entirely.
  if (import.meta.env.DEV && config.devTools) {
    nav.push({ to: '/dev/primitives', label: 'نمونه اجزا', dev: true })
    children.push({ path: 'dev/primitives', lazy: async () => ({ Component: (await import('../routes/dev/PrimitivesPage')).PrimitivesPage }) })
  }
  children.push({ path: '*', lazy: async () => ({ Component: (await import('../routes/NotFoundPage')).NotFoundPage }) })
  return [{ path: '/', element: <Shell nav={nav} />, children }]
}
