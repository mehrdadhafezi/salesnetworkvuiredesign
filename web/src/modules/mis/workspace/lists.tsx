import { Link } from 'react-router'
import { useApi, type CaseSummary, type CustodyAssignment } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { StateBlock } from '../../../shared/ui/primitives'
import { formatFaNumber } from '../../../shared/ui/ds'
import { ParityPill, ResolutionPill } from '../../sales/components'
import { text, workflowLabel } from '../../sales/labels'
import { Pager, useCursors } from '../../sales/pager'
import { CaseLinks, CustodyCell, Holder, PresencePill, case360, caseCustody, useCustodyFor, type CustodyFor } from '../components'
import { blockedSales, derivationLabel } from '../labels'

export const PAGE = 25

type Query = Record<string, string>

/**
 * One page of GET /api/v1/custody/assignments with server filters only.
 * Keys are the contract's query parameter names; the SPA never filters rows.
 */
export function CustodyList({ query, empty }: { query: Query; empty: string }) {
  const api = useApi()
  const c = useCursors()
  const list = useRead(JSON.stringify({ query, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/custody/assignments', {
    // The query is built from an allowlisted form / exception definition; the server rejects anything else (400).
    params: { query: { ...(query as Record<string, never>), ...(c.cursor ? { cursor: c.cursor } : {}), limit: PAGE } },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  return (
    <>
      <ReadStates state={list} what="نگهداری" blockedDetail={blockedSales} />
      {list.status === 'ok' && (list.data.items.length === 0
        ? <StateBlock kind="empty" title={empty} detail="نبود نتیجه برای این فیلتر به معنی صفر بودن کل داده نیست." />
        : <CustodyTable items={list.data.items} />)}
      {list.status === 'ok' && <Pager page={list.data.page} c={c} />}
    </>
  )
}

export function CustodyTable({ items }: { items: CustodyAssignment[] }) {
  return (
    <div className="mis-table-wrap">
      <table className="mis-table mis-stack">
        <caption className="sr">پروجکشن‌های نگهداری</caption>
        <thead>
          <tr><th scope="col">پرونده</th><th scope="col">تطابق نگهدار</th><th scope="col">نگهدار استنتاجی (رویدادها)</th>
            <th scope="col">نگهدار ثبت‌شده در منبع</th><th scope="col">حضور</th><th scope="col">رویداد · یافته</th></tr>
        </thead>
        <tbody>
          {items.map(a => (
            <tr key={a.case_id}>
              <td data-label="پرونده">
                <span className="mis-links">
                  <Link to={caseCustody(a.case_id)}>شواهد نگهداری</Link>
                  <Link to={case360(a.case_id)} className="mis-sub-link">نمای ۳۶۰</Link>
                </span>
                <span className="mis-id mis-ltr" title={a.case_id}>{a.case_id.slice(0, 8)}</span>
              </td>
              <td data-label="تطابق نگهدار"><ParityPill parity={a.parity} /></td>
              <td data-label="نگهدار استنتاجی"><Holder user={a.derived_holder} none="—" />
                <span className="muted mis-small">{derivationLabel[a.derivation] ?? a.derivation}</span></td>
              <td data-label="نگهدار منبع"><Holder user={a.legacy_holder} />
                <span className="muted mis-small mis-ltr">{a.legacy_source_spec}</span></td>
              <td data-label="حضور"><PresencePill presence={a.presence} /></td>
              <td data-label="رویداد · یافته">{`${formatFaNumber(a.event_count)} رویداد · ${formatFaNumber(a.finding_count)} یافته`}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/** One page of GET /api/v1/cases with server filters only, plus the page's custody evidence. */
export function CaseList({ query, empty }: { query: Query; empty: string }) {
  const api = useApi()
  const c = useCursors()
  const list = useRead(JSON.stringify({ query, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/cases', {
    params: { query: { ...(query as Record<string, never>), ...(c.cursor ? { cursor: c.cursor } : {}), limit: PAGE } },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  const ids = list.status === 'ok' ? list.data.items.map(x => x.case_id) : undefined
  const custody = useCustodyFor(ids, list.status === 'ok' && actionHint(list.allowed, 'custody.custody.read').allowed)
  return (
    <>
      <ReadStates state={list} what="پرونده‌ها" blockedDetail={blockedSales} />
      {list.status === 'ok' && (list.data.items.length === 0
        ? <StateBlock kind="empty" title={empty} detail="نبود نتیجه برای این فیلتر به معنی صفر بودن کل داده نیست." />
        : <CaseTable items={list.data.items} custody={custody} />)}
      {list.status === 'ok' && <Pager page={list.data.page} c={c} />}
    </>
  )
}

export function CaseTable({ items, custody }: { items: CaseSummary[]; custody: CustodyFor }) {
  return (
    <div className="mis-table-wrap">
      <table className="mis-table mis-stack">
        <caption className="sr">پرونده‌ها</caption>
        <thead>
          <tr><th scope="col">پرونده (منبع)</th><th scope="col">نوع منبع</th><th scope="col">حضور</th><th scope="col">وضعیت (خام)</th>
            <th scope="col">پیوند مشتری</th><th scope="col">نگهداری</th><th scope="col">شواهد</th></tr>
        </thead>
        <tbody>
          {items.map(x => (
            <tr key={x.case_id}>
              <td data-label="پرونده (منبع)"><CaseLinks id={x.case_id} label={`${x.source.source_spec} #${x.source.source_key}`} /></td>
              <td data-label="نوع منبع">{workflowLabel[x.workflow_type]}</td>
              <td data-label="حضور"><PresencePill presence={x.presence} /></td>
              <td data-label="وضعیت (خام)"><span className="mis-ltr">{text(x.state_raw)}</span>
                {x.state_mapping === 'UNKNOWN' ? <span className="muted"> (نگاشت‌نشده)</span> : null}</td>
              <td data-label="پیوند مشتری"><ResolutionPill resolution={x.customer_link_state} /></td>
              <td data-label="نگهداری"><CustodyCell custody={custody} caseId={x.case_id} /></td>
              <td data-label="شواهد"><Link to={caseCustody(x.case_id)}>نگهداری و انتساب</Link></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
