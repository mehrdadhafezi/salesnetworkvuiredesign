import { StatusPill } from '../../../shared/ui/primitives'
import { OPERATIONS } from '../operations'

/**
 * Inventory operations with the grounded reason each is unavailable. A
 * status list, never a control: no button, link or disabled placeholder is
 * rendered for an operation that has no contract.
 */
export function OperationsStatus({ absent }: { absent?: boolean }) {
  return (
    <div className="mis-ops">
      <p className="muted mis-hint">
        هیچ عملیات موجودی (تخصیص، انتقال، برگشت، آزادسازی، بازتخصیص، استفاده مجدد یا نهایی‌سازی) در قرارداد API وجود ندارد یا مجاز نیست؛ بنابراین هیچ کنترلی نمایش داده نمی‌شود.
        {absent ? ' این پرونده در آخرین واردسازی کامل غایب است؛ غیبت هرگز پرونده را قابل استفاده مجدد یا آزاد نمی‌کند.' : ''}
      </p>
      <ul className="mis-ops-list" aria-label="وضعیت عملیات موجودی">
        {OPERATIONS.map(o => (
          <li key={o.id}>
            <span className="mis-ops-head"><b>{o.label}</b> <StatusPill tone="slate" icon="ban" label="در دسترس نیست" /></span>
            <span className="muted">{o.reason}</span>
            <span className="mis-ops-by muted">{`وابسته به: ${o.blockedBy.join('، ')}`}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
