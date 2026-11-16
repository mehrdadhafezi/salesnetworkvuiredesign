import type { components } from '../../shared/api'
import type { DsIconName } from '../../shared/ui/ds'

/**
 * RM-E02 destination registry: every navigable CRM destination that EXISTS in
 * `routes.tsx` today, nothing else. It is presentation metadata only — it never
 * grants anything. A destination is listed in the navigation only when BOTH
 *   1. the server session reports its feature switch on (`features[feature] === true`), and
 *   2. the server's effective-access evaluation (`GET /api/v1/authorization/effective-access`)
 *      returns an ALLOWED decision for its capability (`allowed: true`, outcome SUCCESS).
 * A feature switch alone never shows a destination. A capability the OpenAPI
 * contract does not enumerate is UNKNOWN here and fails closed (the route still
 * works by direct link; the server authorises every request).
 */

/** Capability values the OpenAPI `Capability` enum documents (the only ones the SPA may read). */
export type ContractCapability = components['schemas']['Capability']
/** Server feature switches (navigation hint only, never authority). */
export type FeatureKey = keyof NonNullable<components['schemas']['SessionContext']['features']>

/**
 * How the destination's server capability is known:
 * - CONTRACT: the list endpoint's `Requires …` capability is in the OpenAPI enum.
 * - UNKNOWN: the server checks a capability that the OpenAPI `Capability` enum does not
 *   list (contract drift). Hidden from navigation (fail closed). Since RM-ACCESS-01 the
 *   enum equals the backend registry (a Go contract test fails on drift), so no
 *   destination is UNKNOWN today; the variant stays as the fail-closed rule for drift.
 *
 * Membership in the enum never shows a link by itself: the server's decision for that
 * capability must be ALLOWED (which, since RM-ACCESS-01, already includes the capability's
 * own feature flag) AND the session must report the destination's feature on.
 */
export type CapabilityBasis =
  | { kind: 'CONTRACT'; capability: ContractCapability }
  | { kind: 'UNKNOWN'; serverCapability: string; reason: 'NOT_IN_OPENAPI_ENUM' }

/** What the existing page does (from the slice that shipped it). */
export type DestinationKind = 'READ' | 'CONTROLLED_WRITE' | 'DIAGNOSTIC'

export type ModuleKey = 'hr' | 'customers' | 'sales' | 'billing' | 'reporting' | 'finance' | 'biawin' | 'commission' | 'operations' | 'integration' | 'payments'

export interface Destination {
  id: string
  /** Existing route (unchanged URL). */
  to: string
  /** Existing page label (the page's own h1); never a spec label for a screen that does not exist. */
  label: string
  /** Short label for the mobile bottom bar. */
  short: string
  icon: DsIconName
  module: ModuleKey
  kind: DestinationKind
  feature: FeatureKey
  capability: CapabilityBasis
}

const c = (capability: ContractCapability): CapabilityBasis => ({ kind: 'CONTRACT', capability })

export const MODULE_LABEL: Record<ModuleKey, string> = {
  hr: 'منابع انسانی',
  customers: 'مشتریان',
  sales: 'فروش',
  billing: 'صورت‌حساب و پرداخت',
  reporting: 'گزارش و MIS',
  finance: 'مالی (سایه)',
  biawin: 'بیاوین و اجرای خدمت',
  commission: 'پورسانت (سایه)',
  operations: 'عملیات اجرای خدمت',
  integration: 'یکپارچه‌سازی',
  payments: 'پرداخت‌ها',
}

/** Order = module order of the previous flat navigation (routes.tsx before RM-E02). */
export const DESTINATIONS: readonly Destination[] = [
  { id: 'hr.workforce', to: '/hr', label: 'نیروها', short: 'نیروها', icon: 'users', module: 'hr', kind: 'READ', feature: 'hr_read', capability: c('org.person.read') },
  { id: 'hr.structure', to: '/hr/structure', label: 'ساختار سازمانی', short: 'ساختار', icon: 'layers', module: 'hr', kind: 'READ', feature: 'hr_read', capability: c('org.unit.read') },
  { id: 'hr.exceptions', to: '/hr/exceptions', label: 'استثناهای سلسله‌مراتب', short: 'استثناها', icon: 'alert', module: 'hr', kind: 'READ', feature: 'hr_read', capability: c('org.assignment.read') },
  { id: 'hr.requests', to: '/hr/requests', label: 'درخواست‌های منابع انسانی', short: 'درخواست‌ها', icon: 'send', module: 'hr', kind: 'CONTROLLED_WRITE', feature: 'hr_request_write', capability: c('org.hr_request.read') },
  { id: 'customers.list', to: '/customers', label: 'مشتریان', short: 'مشتریان', icon: 'user', module: 'customers', kind: 'READ', feature: 'customers_identity', capability: c('customers.customer.read') },
  { id: 'customers.merge', to: '/customers/merge-cases', label: 'پرونده‌های تطبیق', short: 'تطبیق', icon: 'swap', module: 'customers', kind: 'READ', feature: 'customers_identity', capability: c('customers.merge_case.read') },
  { id: 'customers.report', to: '/customers/identity-report', label: 'گزارش تطبیق هویت', short: 'گزارش هویت', icon: 'file', module: 'customers', kind: 'READ', feature: 'customers_identity', capability: c('customers.identity_report.read') },
  { id: 'sales.cases', to: '/sales/cases', label: 'پرونده‌های فروش', short: 'پرونده‌ها', icon: 'inbox', module: 'sales', kind: 'READ', feature: 'sales_read', capability: c('cases.case.read') },
  { id: 'sales.custody', to: '/sales/custody', label: 'نگهداری پرونده‌ها', short: 'نگهداری', icon: 'target', module: 'sales', kind: 'READ', feature: 'sales_read', capability: c('custody.custody.read') },
  { id: 'sales.report', to: '/sales/report', label: 'گزارش پوشش فروش', short: 'پوشش فروش', icon: 'file', module: 'sales', kind: 'READ', feature: 'sales_read', capability: c('sales.report.read') },
  { id: 'billing.invoices', to: '/billing/invoices', label: 'فاکتورها', short: 'فاکتورها', icon: 'receipt', module: 'billing', kind: 'READ', feature: 'billing_read', capability: c('billing.invoice.read') },
  { id: 'billing.attempts', to: '/billing/attempts', label: 'تلاش‌های پرداخت', short: 'پرداخت‌ها', icon: 'repeat', module: 'billing', kind: 'READ', feature: 'billing_read', capability: c('payments.attempt.read') },
  { id: 'billing.report', to: '/billing/report', label: 'گزارش تطبیق صورت‌حساب', short: 'تطبیق', icon: 'file', module: 'billing', kind: 'READ', feature: 'billing_read', capability: c('billing.reconciliation.read') },
  { id: 'mis.today', to: '/mis', label: 'امروز MIS', short: 'امروز', icon: 'grid', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.mis.read') },
  { id: 'mis.quality', to: '/mis/source-quality', label: 'کیفیت منبع', short: 'کیفیت', icon: 'check', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.mis.read') },
  { id: 'mis.reconciliation', to: '/mis/reconciliation', label: 'تطبیق و پوشش', short: 'تطبیق', icon: 'swap', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.mis.read') },
  { id: 'reports.cases', to: '/reports/cases', label: 'گزارش پرونده‌ها', short: 'پرونده‌ها', icon: 'file', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.metrics.read') },
  { id: 'reports.custody', to: '/reports/custody', label: 'گزارش نگهداری', short: 'نگهداری', icon: 'file', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.metrics.read') },
  { id: 'reports.invoices', to: '/reports/invoices', label: 'گزارش فاکتورها', short: 'فاکتورها', icon: 'file', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.metrics.read') },
  { id: 'reports.payments', to: '/reports/payments', label: 'گزارش پرداخت‌ها', short: 'پرداخت‌ها', icon: 'file', module: 'reporting', kind: 'READ', feature: 'reporting_read', capability: c('reporting.metrics.read') },
  { id: 'finance.dashboard', to: '/finance', label: 'دفتر سایه مالی', short: 'دفتر سایه', icon: 'wallet', module: 'finance', kind: 'READ', feature: 'finance_shadow_read', capability: c('finance.shadow.read') },
  { id: 'finance.ledger', to: '/finance/ledger', label: 'دفتر روزنامه سایه', short: 'روزنامه', icon: 'file', module: 'finance', kind: 'READ', feature: 'finance_shadow_read', capability: c('finance.ledger.read') },
  { id: 'finance.balances', to: '/finance/balances', label: 'مانده‌های سایه', short: 'مانده‌ها', icon: 'layers', module: 'finance', kind: 'READ', feature: 'finance_shadow_read', capability: c('finance.shadow.read') },
  { id: 'finance.reconciliation', to: '/finance/reconciliation', label: 'تطبیق سایه', short: 'تطبیق', icon: 'swap', module: 'finance', kind: 'READ', feature: 'finance_shadow_read', capability: c('finance.reconciliation.read') },
  { id: 'biawin.subscriptions', to: '/biawin/subscriptions', label: 'اشتراک‌های بیاوین', short: 'اشتراک‌ها', icon: 'key', module: 'biawin', kind: 'READ', feature: 'biawin_fulfillment_read', capability: c('biawin.subscription.read') },
  { id: 'biawin.parity', to: '/biawin/parity', label: 'تطبیق اشتراک‌های فعال', short: 'تطبیق', icon: 'swap', module: 'biawin', kind: 'READ', feature: 'biawin_fulfillment_read', capability: c('biawin.parity.read') },
  { id: 'fulfillment.cases', to: '/fulfillment/cases', label: 'پرونده‌های اجرای خدمت', short: 'اجرای خدمت', icon: 'inbox', module: 'biawin', kind: 'READ', feature: 'biawin_fulfillment_read', capability: c('fulfillment.case.read') },
  { id: 'commission.overview', to: '/commission', label: 'پورسانت · نمای سایه', short: 'پورسانت', icon: 'wallet', module: 'commission', kind: 'READ', feature: 'commission_read', capability: c('commission.shadow.read') },
  { id: 'commission.calculations', to: '/commission/calculations', label: 'محاسبات سایه پورسانت', short: 'محاسبات', icon: 'file', module: 'commission', kind: 'READ', feature: 'commission_read', capability: c('commission.shadow.read') },
  { id: 'commission.policies', to: '/commission/policies', label: 'نسخه‌های سیاست پورسانت', short: 'سیاست‌ها', icon: 'layers', module: 'commission', kind: 'READ', feature: 'commission_read', capability: c('commission.policy.read') },
  { id: 'commission.me', to: '/commission/me', label: 'پورسانت‌های من', short: 'پورسانت من', icon: 'wallet', module: 'commission', kind: 'READ', feature: 'commission_read', capability: c('commission.own.read') },
  { id: 'operations.work', to: '/operations/work', label: 'صف کار اجرای خدمت', short: 'صف کار', icon: 'inbox', module: 'operations', kind: 'CONTROLLED_WRITE', feature: 'fulfillment_work', capability: c('fulfillment.work.read') },
  { id: 'operations.activations', to: '/operations/activations', label: 'تصمیم‌های فعال‌سازی اشتراک', short: 'فعال‌سازی', icon: 'key', module: 'operations', kind: 'READ', feature: 'fulfillment_work', capability: c('biawin.activation.read') },
  { id: 'operations.providers', to: '/operations/provider-operations', label: 'عملیات ارائه‌دهنده خارجی', short: 'ارائه‌دهنده', icon: 'activity', module: 'operations', kind: 'CONTROLLED_WRITE', feature: 'fulfillment_work', capability: c('integration.provider_operation.read') },
  { id: 'integration.writeback', to: '/writeback', label: 'بازنویسی به سامانه قدیمی', short: 'بازنویسی', icon: 'history', module: 'integration', kind: 'DIAGNOSTIC', feature: 'writeback_read', capability: c('integration.writeback.read') },
  { id: 'payments.evidence', to: '/payments/evidence', label: 'شواهد پرداخت', short: 'شواهد', icon: 'receipt', module: 'payments', kind: 'DIAGNOSTIC', feature: 'payment_submission', capability: c('payments.evidence.read') },
  { id: 'finance.review', to: '/finance/review', label: 'صف بررسی مالی (فقط خواندنی)', short: 'بررسی مالی', icon: 'checkCircle', module: 'finance', kind: 'READ', feature: 'finance_review_read', capability: c('finance.review.read') },
]

/**
 * Safe destinations: shown to every visitor whatever the evaluation says. They
 * carry no business data (home is the role-entry page; system status is
 * technical health only).
 */
export const SAFE_DESTINATIONS = {
  home: { id: 'home', to: '/', label: 'خانه', short: 'خانه', icon: 'grid' },
  system: { id: 'system', to: '/system', label: 'وضعیت سامانه', short: 'سامانه', icon: 'activity' },
} as const satisfies Record<string, { id: string; to: string; label: string; short: string; icon: DsIconName }>

export function destinationById(id: string): Destination | undefined {
  return DESTINATIONS.find(d => d.id === id)
}
