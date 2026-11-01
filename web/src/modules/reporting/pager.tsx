import { Button } from '../../shared/ui/primitives'
import type { PageInfo } from '../../shared/api'

/** Server cursor controls; the cursor lives in the URL with the query context. */
export function Pager({ page, depth, onBack, onNext, label = 'صفحه‌بندی' }:
  { page: PageInfo; depth: number; onBack: () => void; onNext: (cursor: string) => void; label?: string }) {
  return (
    <nav className="rp-pager" aria-label={label}>
      <Button disabled={depth === 0} onClick={onBack}>صفحه قبل</Button>
      <Button disabled={!page.has_more || !page.next_cursor} onClick={() => { if (page.next_cursor) onNext(page.next_cursor) }}>صفحه بعد</Button>
    </nav>
  )
}
