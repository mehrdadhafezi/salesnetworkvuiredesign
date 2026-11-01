import type { FormEvent } from 'react'
import { Button, Card } from '../../shared/ui/primitives'

/** Allowlisted, server-validated filters only (no free query builder). */
export function Filters({ options, values, onApply, label }:
  { options: Record<string, readonly string[]>; values: Record<string, string>; onApply: (v: Record<string, string>) => void; label: string }) {
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const out: Record<string, string> = {}
    for (const k of Object.keys(options)) {
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') out[k] = v
    }
    onApply(out)
  }
  return (
    <Card title="فیلترها">
      <form className="fn-filters" onSubmit={submit} aria-label={label}>
        {Object.entries(options).map(([k, vs]) => (
          <label key={k} className="lbl">{k}
            <select className="input" name={k} defaultValue={values[k] ?? ''}>
              <option value="">همه</option>
              {vs.map(v => <option key={v} value={v}>{v}</option>)}
            </select>
          </label>
        ))}
        <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
      </form>
    </Card>
  )
}
