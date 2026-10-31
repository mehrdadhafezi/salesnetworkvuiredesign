import '../sales.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type Parity } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ParityPill, ReadOnlyNote, UserRefText } from '../components'
import { blockedDetail, parityLabel } from '../labels'
import { Pager, useCursors } from '../pager'

/**
 * Custody projections (read only): the holder derived from append-only
 * events next to the legacy holder, with their parity. Conflicts stay
 * visible; nothing here assigns, transfers, returns or releases.
 */
export function CustodyPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const parity = params.get('parity') ?? undefined
  const list = useRead(JSON.stringify({ parity, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/custody/assignments', {
    params: { query: compact({ parity: parity as Parity | undefined, cursor: c.cursor, limit: 25 }) },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const v = new FormData(e.currentTarget).get('parity')
    c.reset()
    setParams(typeof v === 'string' && v !== '' ? { parity: v } : {})
  }

  return (
    <>
      <PageHeader title="نگهداری پرونده‌ها" subtitle="نگهدار استنتاجی از رویدادها در کنار نگهدار ثبت‌شده در منبع">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <Card title="فیلتر">
        <form className="sa-filters" onSubmit={apply} aria-label="فیلتر نگهداری">
          <label className="lbl">تطابق نگهدار
            <select className="input" name="parity" defaultValue={parity ?? ''}>
              <option value="">همه</option>
              {(Object.keys(parityLabel) as Parity[]).map(p => <option key={p} value={p}>{parityLabel[p]}</option>)}
            </select>
          </label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="فهرست نگهداری">
        <ReadStates state={list} what="نگهداری" blockedDetail={blockedDetail} />
        {list.status === 'ok' && (list.data.items.length === 0 ? <StateBlock kind="empty" title="موردی نیست" /> : (
          <div className="sa-table-wrap">
            <table className="sa-table">
              <caption className="sr">نگهداری</caption>
              <thead><tr><th scope="col">پرونده</th><th scope="col">تطابق</th><th scope="col">استنتاج از رویدادها</th><th scope="col">ثبت‌شده در منبع</th><th scope="col">رویداد · یافته</th></tr></thead>
              <tbody>
                {list.data.items.map(a => (
                  <tr key={a.case_id}>
                    <td><Link to={`/sales/cases/${a.case_id}`}>مشاهده پرونده</Link></td>
                    <td><ParityPill parity={a.parity} /></td>
                    <td><UserRefText user={a.derived_holder} /></td>
                    <td><UserRefText user={a.legacy_holder} /></td>
                    <td className="num">{a.event_count} · {a.finding_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
        {list.status === 'ok' && <Pager page={list.data.page} c={c} />}
      </Card>
    </>
  )
}
