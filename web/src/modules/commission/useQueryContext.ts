import { useSearchParams } from 'react-router'

/**
 * The query context lives in the URL: allowlisted filters, the pinned
 * run_id and the cursor stack. Paging and detail always carry the run that
 * answered the first page (no cross-run drill-down); changing a filter
 * re-evaluates at the latest run.
 */
export function useQueryContext(filterKeys: readonly string[]) {
  const [params, setParams] = useSearchParams()
  const filters: Record<string, string> = {}
  for (const k of filterKeys) {
    const v = params.get(k)
    if (v) filters[k] = v
  }
  const runId = params.get('run_id') ?? undefined
  const cursors = params.getAll('cursor')
  const cursor = cursors[cursors.length - 1]
  const update = (fn: (p: URLSearchParams) => void) => {
    const next = new URLSearchParams(params)
    fn(next)
    setParams(next)
  }
  /** Pins the resolved run (if the URL has none) before fn. */
  const pinned = (resolved: string | undefined, fn: (p: URLSearchParams) => void) => update(p => {
    if (!p.get('run_id') && resolved) p.set('run_id', resolved)
    fn(p)
  })
  const setFilters = (values: Record<string, string>) => update(p => {
    for (const k of filterKeys) {
      const v = values[k]
      if (v) p.set(k, v)
      else p.delete(k)
    }
    p.delete('cursor')
    p.delete('run_id') // a changed query is evaluated at the latest run
  })
  const back = () => update(p => { const all = p.getAll('cursor'); p.delete('cursor'); all.slice(0, -1).forEach(c => p.append('cursor', c)) })
  return { params, filters, runId, cursors, cursor, update, pinned, setFilters, back }
}
