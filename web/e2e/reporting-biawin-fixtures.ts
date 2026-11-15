import type { APIResponse, Page } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

/** Page texts (Persian UI) of the reporting / MIS and Biawin read surfaces. */
export const T = {
  denied: 'دسترسی به این بخش ندارید',
  disabled: 'این بخش هنوز فعال نشده است',
  readOnly: 'فقط خواندنی',
  logout: 'خروج',
  login: 'ورود',
  mis: 'امروز MIS',
  sourceQuality: 'کیفیت منبع',
  reconciliation: 'تطبیق و پوشش',
  subscriptions: 'اشتراک‌های بیاوین',
  subscription: 'اشتراک بیاوین',
  parity: 'تطبیق اشتراک‌های فعال',
  applyFilter: 'اعمال فیلتر',
  nextPage: 'صفحه بعد',
  prevPage: 'صفحه قبل',
  details: 'جزئیات',
}

/** Report pages: route segment → heading. */
export const REPORTS: [string, string][] = [
  ['cases', 'گزارش پرونده‌ها'], ['custody', 'گزارش نگهداری'], ['invoices', 'گزارش فاکتورها'], ['payments', 'گزارش تلاش‌های پرداخت'],
]

/** Every list page of the two surfaces: path → heading. */
export const PAGES: [string, string][] = [
  ['/mis', T.mis], ['/mis/source-quality', T.sourceQuality], ['/mis/reconciliation', T.reconciliation],
  ...REPORTS.map(([r, h]) => [`/reports/${r}`, h] as [string, string]),
  ['/biawin/subscriptions', T.subscriptions], ['/biawin/parity', T.parity],
]

/** Navigation entries the session's feature hints show or hide. */
export const NAV = [T.mis, T.sourceQuality, T.reconciliation, 'گزارش پرونده‌ها', T.subscriptions, T.parity]

/** Ids of real rows of the seeded dataset (class A: a refusal is never a 404 of an empty database). */
export interface RowIds { caseId: string; invoiceId: string; attemptId: string; subscription: string; reportingRun: string; biawinRun: string }

/** Every reporting / MIS / Biawin read of the two surfaces, including the API-only ones (metrics, Invoice Review). */
export function allReads(ids: RowIds): string[] {
  return [
    '/api/v1/mis/overview', '/api/v1/mis/source-quality', '/api/v1/mis/reconciliation',
    '/api/v1/reporting/metrics', '/api/v1/reporting/metrics/cases.staged_total',
    '/api/v1/reports/cases', `/api/v1/reports/cases/${ids.caseId}`,
    '/api/v1/reports/custody', `/api/v1/reports/custody/${ids.caseId}`,
    '/api/v1/reports/invoices', `/api/v1/reports/invoices/${ids.invoiceId}`,
    '/api/v1/reports/payments', `/api/v1/reports/payments/${ids.attemptId}`,
    '/api/v1/invoice-review', `/api/v1/invoice-review/${ids.invoiceId}`,
    '/api/v1/biawin/subscriptions', `/api/v1/biawin/subscriptions/${ids.subscription}`, `/api/v1/biawin/subscriptions/${ids.subscription}/items`,
    '/api/v1/biawin/parity',
  ]
}

/** A well-formed id that names no row. */
export const unknownId = '00000000-0000-4000-8000-0000000000e7'

/** The capability refusal (403 UNAUTHORIZED / ACTION_DENIED), with no data. */
export async function expectDenied(r: APIResponse, label: string) {
  expect(r.status(), label).toBe(403)
  const b = (await r.json()) as { outcome: string; reasons: string[]; data?: unknown; items?: unknown; summary?: unknown }
  expect(b.outcome, label).toBe('UNAUTHORIZED')
  expect(b.reasons, label).toEqual(['ACTION_DENIED'])
  expect(b.data, label).toBeUndefined()
  expect(b.items, label).toBeUndefined()
  expect(b.summary, label).toBeUndefined()
}

/** The read disabled contract: 200 BLOCKED / FEATURE_DISABLED, with no data. */
export async function expectDisabled(r: APIResponse, label: string) {
  expect(r.status(), label).toBe(200)
  const b = (await r.json()) as { outcome: string; reasons: string[]; data?: unknown; items?: unknown }
  expect(b.outcome, label).toBe('BLOCKED')
  expect(b.reasons, label).toEqual(['FEATURE_DISABLED'])
  expect(b.data, label).toBeUndefined()
  expect(b.items, label).toBeUndefined()
}

/** A page shows the given state and no data table or row link. */
export async function expectNoData(page: Page, text: string, label: string) {
  await expect(page.getByText(text).first(), label).toBeVisible()
  await expect(page.getByRole('table'), label).toHaveCount(0)
  await expect(page.locator('a[href^="/biawin/subscriptions/"]'), label).toHaveCount(0)
  await expect(page.locator('a[href^="/reports/"]').filter({ hasText: 'جدول همین شاخص' }), label).toHaveCount(0)
}

/** The API paths of the two read surfaces. */
export const surface = /^\/api\/v1\/(mis|reports|reporting|invoice-review|biawin)(\/|$)/

/**
 * Read-only guard: records every request the page sends to the two read
 * surfaces that is not a GET. The surfaces are read only, so the list must
 * stay empty (login / logout are outside these paths).
 */
export const readOnly = guarded.extend<{ writes: string[] }>({
  writes: async ({ page }, provide) => {
    const writes: string[] = []
    page.on('request', r => {
      const path = new URL(r.url()).pathname
      if (!surface.test(path) || r.method() === 'GET' || r.method() === 'HEAD') return
      writes.push(`${r.method()} ${path}`)
    })
    await provide(writes)
    expect(writes, 'the reporting / Biawin pages sent a non-read request').toEqual([])
  },
})

export { expect, required }
