import { useMemo, useState } from 'react'
import { ContextHint, Drawer, DsIcon, Metric, MetricGroup, PageHead, Panel, QueueTabs, Toolbar, toFaDigits, type DsQueue } from '../../../shared/ui/ds'
import { FactChip, Money, Note, OwnershipFacts, SearchField, TableFoot, Timeline, Who, useCustomerName, useListState } from '../components/bits'
import { NotFoundDrawer, Restricted, ScreenFrame, useDetail, useStatePresentation } from '../components/states'
import { REPEAT_ITEMS } from '../fixtures/sellerFixtures'
import { DEMO_SUFFIX } from '../labels'
import { useDrawerParam } from '../state/hooks'
import { filterRepeat, paginate } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { RepeatItem } from '../types'

/*
 * S3 — نیاز به اقدام مجدد (prototype V.repeat). The frozen prototype shows only the empty state;
 * the preview adds SYNTHETIC rows so the queue, ownership context and history can be evaluated.
 * The original seller's attribution is displayed, never rewritten; no reassignment, handoff or
 * notification happens.
 */

const PER_PAGE = 10

export function RepeatPage() {
  const sp = useStatePresentation()
  const items = sp.data(REPEAT_ITEMS)
  const [queue, setQueue] = useState<'open' | 'done'>('open')
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const [param, setParam] = useDrawerParam('item')
  const list = useMemo(() => filterRepeat(items, queue, q, !sp.policy.fieldsRestricted), [items, queue, q, sp.policy.fieldsRestricted])
  const pg = paginate(list, page, PER_PAGE)
  const detail = useDetail(param, id => items.find(r => String(r.id) === id))
  const open = detail.record
  const openCount = items.filter(r => !r.done).length
  const doneCount = items.length - openCount
  const stateEl = useListState({
    count: pg.items.length, noResult: q !== '', onClear: () => setQ(''),
    emptyTitle: 'کاری در این صف نیست',
    emptyDetail: 'وقتی مرحله بعدی یک پرداخت به شما ارجاع شود، اینجا نمایش داده می‌شود. مالک فروش و پورسانت پایه برای فروشنده صادرکننده اولیه حفظ می‌شود.',
  })
  const queues: DsQueue[] = [
    { id: 'open', label: 'باز', tone: 'amber', icon: 'clock', count: sp.count(openCount) },
    { id: 'done', label: 'انجام‌شده', tone: 'green', icon: 'checkCircle', count: sp.count(doneCount) },
  ]
  return (
    <ScreenFrame title="نیاز به اقدام مجدد" contract="no_source" head={
      <PageHead
        title="نیاز به اقدام مجدد"
        sub="پرداخت‌های مرحله‌ای که مرحله بعدشان به شما ارجاع شده است (نمونه ساختگی)"
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="باز" {...sp.kpi(openCount)} accent="neutral" />
            <Metric label="این ماه انجام‌شده" {...sp.kpi(doneCount)} accent="green" tone="good" />
          </MetricGroup>
        }
      />}>
      <Panel label="صف اقدام مجدد">
        <QueueTabs queues={queues} value={queue} onChange={id => { setQueue(id as 'open' | 'done'); setPage(1); setParam(null) }} />
        <Toolbar>
          <div className="sl-grow-field"><SearchField id="sl-rep-q" label="جستجو در صف" placeholder="نام، موبایل یا کد فاکتور" value={q} onChange={v => { setQ(v); setPage(1) }} /></div>
        </Toolbar>
        {stateEl ?? (
          <>
            <div className="sl-tbl-wrap">
              <table className="sl-tbl sl-stackable" aria-label="صف اقدام مجدد">
                <thead><tr>
                  <th scope="col">مشتری</th><th scope="col">فاکتور و مرحله</th><th scope="col">مبلغ مرحله</th>
                  <th scope="col">فروشنده اولیه <ContextHint text="صادرکننده اولیه فاکتور؛ با ارجاع مرحله بعد تغییر نمی‌کند." /></th>
                  <th scope="col">اقدام بعدی با</th><th scope="col">زمان اقدام</th>
                  <th scope="col" className="sl-col-actions"><span className="sr">اقدام</span></th>
                </tr></thead>
                <tbody>
                  {pg.items.map(r => (
                    <tr key={r.id} data-row={r.id} tabIndex={0} className={open?.id === r.id ? 'sl-sel' : undefined} aria-selected={open?.id === r.id}
                      onClick={() => setParam(String(r.id))} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); setParam(String(r.id)) } }}>
                      <td data-l="مشتری"><Who name={r.customer} phone={r.phone} /></td>
                      <td data-l="فاکتور و مرحله"><span className="mono">{r.invoice}</span><div className="muted sl-micro">{r.stageLabel}</div></td>
                      <td data-l="مبلغ مرحله"><Money value={r.amount} /></td>
                      <td data-l="فروشنده اولیه">{r.originalSeller}</td>
                      <td data-l="اقدام بعدی با">{r.nextActor}</td>
                      <td data-l="زمان اقدام"><span className={`sl-next ${r.dueTone ? `sl-${r.dueTone}` : ''}`}><DsIcon name={r.done ? 'check' : 'clock'} />{r.due}</span></td>
                      <td className="sl-col-actions"><div className="sl-row-actions"><button type="button" className={`btn btn-sm${r.done ? '' : ' btn-soft'}`} onClick={e => { e.stopPropagation(); setParam(String(r.id)) }}>{r.done ? 'مشاهده' : 'پیگیری'}</button></div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <TableFoot page={pg} perPage={PER_PAGE} onPage={setPage} />
          </>
        )}
      </Panel>
      <RepeatDrawer r={open} onClose={() => setParam(null)} />
      <NotFoundDrawer open={detail.notFound} kicker="اقدام مجدد" onClose={() => setParam(null)} />
    </ScreenFrame>
  )
}

function RepeatDrawer({ r, onClose }: { r: RepeatItem | null; onClose: () => void }) {
  const { toast, ui } = useSellerPreview()
  const name = useCustomerName()
  return (
    <Drawer open={r !== null} onClose={onClose} title={r ? name(r.customer) : 'اقدام مجدد'} kicker={r ? `فاکتور ${r.invoice} · ${r.stageLabel}` : 'اقدام مجدد'}
      footerHint="پیش‌نمایش: ارجاع یا اعلانی انجام نمی‌شود"
      footer={r && !r.done ? (
        <>
          <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
          <button type="button" className="btn btn-lg btn-primary" onClick={() => toast(`«ثبت پیگیری مرحله» برای ${r.invoice} ${DEMO_SUFFIX}`)}>ثبت پیگیری مرحله</button>
        </>
      ) : <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>}
    >
      {r ? (
        <>
          <div className="sl-dr-head">
            <div className="sl-dr-meta">{ui.policy.fieldsRestricted ? <Restricted what="شماره مشتری" /> : <span className="mono sl-dr-phone">{toFaDigits(r.phone)}</span>}<span>مبلغ مرحله <Money value={r.amount} /></span></div>
            <div className="sl-dr-facts"><FactChip icon="clock" cls={r.dueTone ? `sl-${r.dueTone}` : ''}>زمان اقدام: {r.due}</FactChip></div>
          </div>
          <section className="ds-sec sl-sec-primary" aria-label="مالکیت و مسئولیت">
            <div className="ds-sec-h"><h3>مالکیت و مسئولیت</h3></div>
            <OwnershipFacts rows={[
              { term: 'مالک اولیه (فروشنده صادرکننده)', value: r.originalSeller },
              { term: 'مسئول فعلی مرحله', value: r.currentCustody },
              { term: 'اقدام بعدی با', value: r.nextActor },
            ]} />
            <Note tone="info" icon="info">مالک فروش و پورسانت پایه برای فروشنده صادرکننده اولیه حفظ می‌شود؛ ارجاع مرحله بعد این انتساب تاریخی را بازنویسی نمی‌کند.</Note>
          </section>
          <details className="sl-details" open>
            <summary><h3>تاریخچه و زمینه</h3><DsIcon name="chev" className="sl-chev" /></summary>
            <Timeline events={r.history} label="تاریخچه مرحله" />
          </details>
        </>
      ) : null}
    </Drawer>
  )
}
