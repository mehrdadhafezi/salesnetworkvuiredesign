import type { HRRequest } from '../../../shared/api'
import type { Tone } from '../../../shared/ui/primitives'
import { hrStatusLabel, hrTypeLabel } from '../../writes/labels'

/** Labels come from the Slice 11 contract labels; unknown values are shown raw, never relabelled. */
export const requestStatusLabel = (s: string) => hrStatusLabel[s] ?? s
export const requestTypeLabel = (t: string) => hrTypeLabel[t] ?? t

export const hrStatusTone = (s: HRRequest['status']): Tone => ({
  pending_review: 'amber', pending_hr: 'blue', approved: 'green', rejected: 'slate', cancelled: 'slate', failed: 'red',
} as const satisfies Record<HRRequest['status'], Tone>)[s]

export const hrStatusIcon = (s: HRRequest['status']) => ({
  pending_review: 'inbox', pending_hr: 'inbox', approved: 'check', rejected: 'ban', cancelled: 'ban', failed: 'x',
} as const satisfies Record<HRRequest['status'], string>)[s]

/** Server-side status queues (one status per request; the server never widens the caller's scope). */
export const REQUEST_QUEUES = [
  { id: 'all', label: 'همه' },
  { id: 'pending_review', label: 'در انتظار تایید سلسله‌مراتبی' },
  { id: 'pending_hr', label: 'در انتظار منابع انسانی' },
  { id: 'rejected', label: 'رد شده' },
  { id: 'approved', label: 'تایید و اعمال شده' },
  { id: 'failed', label: 'خطا در اعمال' },
  { id: 'cancelled', label: 'لغو شده' },
] as const

export type RequestQueue = (typeof REQUEST_QUEUES)[number]['id']

export const isQueue = (v: string | null): v is RequestQueue => REQUEST_QUEUES.some(q => q.id === v)

/**
 * Product lifecycle (HR-PRODUCT-SPEC §11) mapped onto the preserved status ids.
 * "Apply" is an operation, not a status: in this system it is never performed
 * (OPD-05, SD-04, HR apply IMPLEMENTATION BLOCKED), so it is shown as blocked
 * unless the server itself reports approved / failed.
 */
export type StepState = 'done' | 'current' | 'blocked' | 'not_reached' | 'stopped'

export function lifecycle(status: HRRequest['status']): { key: string; label: string; state: StepState }[] {
  const closedEarly = status === 'rejected' || status === 'cancelled'
  return [
    { key: 'requested', label: 'ثبت درخواست', state: 'done' },
    { key: 'review', label: 'تایید سلسله‌مراتبی', state: status === 'pending_review' ? 'current' : closedEarly ? 'stopped' : 'done' },
    { key: 'hr', label: 'بررسی منابع انسانی', state: status === 'pending_hr' ? 'current' : status === 'pending_review' || closedEarly ? 'not_reached' : 'done' },
    { key: 'apply', label: 'اعمال نهایی', state: status === 'approved' ? 'done' : status === 'failed' ? 'stopped' : closedEarly ? 'not_reached' : 'blocked' },
  ]
}

export const stepLabel: Record<StepState, string> = {
  done: 'انجام شده', current: 'مرحله جاری', blocked: 'مسدود در این سامانه', not_reached: 'نرسیده', stopped: 'متوقف شد',
}
