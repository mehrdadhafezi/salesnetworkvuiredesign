import type { Outcome } from '../../shared/api'
import { StateBlock } from '../../shared/ui/primitives'

/**
 * The server answers UNKNOWN (with no run, meta or rows) before any shadow
 * run exists: that is a distinct state, never a transport failure and never
 * an empty zero. Any other envelope without data stays a failure.
 */
export function noRun(e: { outcome: Outcome }): null | undefined {
  return e.outcome === 'UNKNOWN' ? null : undefined
}

export function NoRun() {
  return <StateBlock kind="unknown" title="هنوز هیچ اجرای سایه‌ای ثبت نشده است"
    detail="نامعلوم است، نه صفر و نه خطای ارتباط. اجرا فقط با کار آفلاین crm-finance-shadow ساخته می‌شود." />
}
