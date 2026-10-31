import { useState } from 'react'
import type { PageInfo } from '../../shared/api'
import { Button } from '../../shared/ui/primitives'

/** Server cursor stack: the SPA never computes pages itself. */
export function useCursors() {
  const [cursors, setCursors] = useState<string[]>([])
  return { cursor: cursors[cursors.length - 1], depth: cursors.length, reset: () => setCursors([]),
    back: () => setCursors(c => c.slice(0, -1)), next: (n: string) => setCursors(c => [...c, n]) }
}

export function Pager({ page, c }: { page: PageInfo; c: ReturnType<typeof useCursors> }) {
  return (
    <nav className="cu-pager" aria-label="صفحه‌بندی">
      <Button disabled={c.depth === 0} onClick={c.back}>صفحه قبل</Button>
      <Button disabled={!page.has_more || !page.next_cursor} onClick={() => { if (page.next_cursor) c.next(page.next_cursor) }}>صفحه بعد</Button>
    </nav>
  )
}
