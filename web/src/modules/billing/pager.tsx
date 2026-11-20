import { useState } from 'react'
import type { PageInfo } from '../../shared/api'
import { Button } from '../../shared/ui/primitives'

/** Server cursor stack: the SPA never computes pages itself. */
export function useCursors() {
  const [cursors, setCursors] = useState<string[]>([])
  return { cursor: cursors[cursors.length - 1], depth: cursors.length, reset: () => setCursors([]),
    back: () => setCursors(c => c.slice(0, -1)), next: (n: string) => setCursors(c => [...c, n]) }
}

/** Renders nothing for a single page: no disabled, non-functional controls. */
export function Pager({ page, c, label = 'صفحه‌بندی' }: { page: PageInfo; c: ReturnType<typeof useCursors>; label?: string }) {
  if (c.depth === 0 && !page.has_more) return null
  return (
    <nav className="bi-pager" aria-label={label}>
      <Button disabled={c.depth === 0} onClick={c.back}>صفحه قبل</Button>
      <Button disabled={!page.has_more || !page.next_cursor} onClick={() => { if (page.next_cursor) c.next(page.next_cursor) }}>صفحه بعد</Button>
    </nav>
  )
}
