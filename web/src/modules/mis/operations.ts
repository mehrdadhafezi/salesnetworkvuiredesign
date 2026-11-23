/**
 * RM-P05 inventory operations and why none is offered here.
 *
 * Inspected before writing this file (main fa8f0e0):
 * - The API router (`internal/platform/apiserver/server.go`, ContractMethods)
 *   has no assignment, transfer, return, reset, release, reallocation,
 *   recycle/reuse or finalization route for Cases or custody: custody and
 *   cases expose GET only; the only Case write is the seller's own contact
 *   outcome (`POST /api/v1/cases/{case_id}/contact-outcomes`, feature
 *   `cases.contact_outcome_write`, holder-only row scope), which is not an
 *   inventory operation.
 * - `internal/custody/release` evaluates return / reset / release as a pure
 *   gate with no route; it never answers SUCCESS while ADR-01 and F01 (and,
 *   for reset / release, OPD-03) are open.
 * - `internal/cases/api/dependency.go` requires the union of every financial
 *   link before any inventory writer; unknown links fail closed.
 * - The MIS role carries no capability (`internal/identity/api/api.go`).
 *
 * Every entry below is therefore UNAVAILABLE with its grounded reason. No
 * control, button or disabled placeholder is rendered for any of them.
 */

export interface OperationStatus {
  id: string
  label: string
  /** Why it is not offered (contract / decision / invariant). */
  reason: string
  /** Decision ids or invariant references that must change first. */
  blockedBy: string[]
}

export const NO_ROUTE = 'هیچ مسیر یا قرارداد API برای این عملیات وجود ندارد.'

export const OPERATIONS: OperationStatus[] = [
  { id: 'assign', label: 'تخصیص پرونده', reason: `${NO_ROUTE} تخصیص منبع در سامانه قدیمی می‌ماند و در مدل سایه شبیه‌سازی نمی‌شود.`, blockedBy: ['MIS-PRODUCT-SPEC §8', 'ADR-01'] },
  { id: 'transfer', label: 'انتقال نگهداری', reason: `${NO_ROUTE} نگهداری فقط از رویدادهای منبع استنتاج می‌شود.`, blockedBy: ['ADR-01'] },
  { id: 'return', label: 'برگشت به منبع', reason: `${NO_ROUTE} ارزیاب برگشت بدون مسیر است و تا تصویب F01 هرگز SUCCESS نمی‌دهد؛ فقط نگهدار فعلی و پس از بررسی همه پیوندهای مالی.`, blockedBy: ['F01', 'ADR-01', 'MIS-G01'] },
  { id: 'reset', label: 'بازنشانی', reason: `${NO_ROUTE} علاوه بر F01 به سیاست بازنشانی (OPD-03) نیاز دارد.`, blockedBy: ['F01', 'ADR-01', 'OPD-03'] },
  { id: 'release', label: 'آزادسازی', reason: `${NO_ROUTE} لغو یا ابطال فاکتور خودکار چیزی را آزاد نمی‌کند (OPD-03 / OPD-04).`, blockedBy: ['F01', 'ADR-01', 'OPD-03', 'OPD-04'] },
  { id: 'reallocate', label: 'بازتخصیص', reason: `${NO_ROUTE} مقام MIS یا دید گسترده داده اجازه بازپس‌گیری نمی‌دهد.`, blockedBy: ['ADR-01', 'MIS-G07'] },
  { id: 'reuse', label: 'استفاده مجدد / بازیافت', reason: `${NO_ROUTE} غیبت در آخرین واردسازی (ABSENT_IN_LATEST) هرگز پرونده را قابل استفاده مجدد نمی‌کند.`, blockedBy: ['GATE-0 INV-006…010', 'OPD-03'] },
  { id: 'finalize', label: 'نهایی‌سازی', reason: `${NO_ROUTE} وضعیت نهایی از منبع خوانده می‌شود و در اینجا تعیین نمی‌شود.`, blockedBy: ['STATUS-TRANSITIONS'] },
]
