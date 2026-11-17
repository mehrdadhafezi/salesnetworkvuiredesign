import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router'
import { Drawer, Metric, MetricGroup, PageHead, Panel, StatusIndicator, Toolbar, toFaDigits } from '../../../shared/ui/ds'
import { SearchField, TableFoot, Timeline, Who, useListState } from '../components/bits'
import { BEHAVIOR, INVOICES } from '../fixtures/sellerFixtures'
import { useDrawerParam } from '../state/hooks'
import { filterBehavior, paginate } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { BehaviorItem } from '../types'

/*
 * S5 — رفتار مشتریان (prototype V.behavior + behaviorDrawer). SYNTHETIC events on the seller's own
 * synthetic invoices only. Search and the «فعال» filter run in memory over the fixture; there is no
 * customer search, no organisation-wide request and no client-side filtering of real API data.
 */

const PER_PAGE = 10

export function BehaviorPage() {
  const { base, setSim } = useSellerPreview()
  const navigate = useNavigate()
  const [q, setQ] = useState('')
  const [onlyHot, setOnlyHot] = useState(false)
  const [page, setPage] = useState(1)
  const [param, setParam] = useDrawerParam('b')
  const list = useMemo(() => filterBehavior(BEHAVIOR, q, onlyHot), [q, onlyHot])
  const pg = paginate(list, page, PER_PAGE)
  const open = param ? BEHAVIOR.find(b => b.id === param) ?? null : null
  const stateEl = useListState({
    count: pg.items.length, noResult: q !== '' || onlyHot, onClear: () => { setQ(''); setOnlyHot(false) }, onRetry: () => setSim('none'),
    emptyTitle: 'رفتاری ثبت نشده است', emptyDetail: 'هنوز مشتری‌ای لینک فاکتور را باز نکرده است.',
  })
  const goInvoice = (code: string) => void navigate(`${base}/invoices?inv=${encodeURIComponent(code)}`)

  return (
    <>
      <PageHead
        title="رفتار مشتریان"
        sub="باز شدن لینک، مشاهده محصول، گردونه، کد تخفیف و پرداخت روی صفحه فاکتور (نمونه ساختگی)"
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="مشتری فعال امروز" value={toFaDigits(BEHAVIOR.filter(b => b.hot).length)} accent="teal" />
            <Metric label="لینک باز شده (۷ روز)" accent="blue" state="unavailable" meaning="پوشش رویداد در نمونه ناقص است" />
            <Metric label="پرداخت پس از بازدید" accent="green" state="undefined" meaning="فرمول در پیش‌نمایش تعریف نشده" />
          </MetricGroup>
        }
      />
      <Panel label="رفتار مشتریان">
        <Toolbar>
          <div className="sl-grow-field"><SearchField id="sl-beh-q" label="جستجوی مشتری یا فاکتور" placeholder="شماره مشتری یا کد فاکتور" value={q} onChange={v => { setQ(v); setPage(1) }} /></div>
          <div className="sl-seg" role="group" aria-label="نوع فعالیت">
            <button type="button" aria-pressed={!onlyHot} onClick={() => { setOnlyHot(false); setPage(1) }}>همه</button>
            <button type="button" aria-pressed={onlyHot} onClick={() => { setOnlyHot(true); setPage(1) }}>فقط فعال</button>
          </div>
        </Toolbar>
        {stateEl ?? (
          <>
            <div className="sl-tbl-wrap">
              <table className="sl-tbl sl-stackable" aria-label="رفتار مشتریان">
                <thead><tr><th scope="col">مشتری</th><th scope="col">فاکتور</th><th scope="col">آخرین فعالیت</th><th scope="col">زمان</th><th scope="col">تعداد</th><th scope="col" className="sl-col-actions"><span className="sr">اقدام</span></th></tr></thead>
                <tbody>
                  {pg.items.map(b => (
                    <tr key={b.id} data-row={b.id} tabIndex={0} className={open?.id === b.id ? 'sl-sel' : undefined} aria-selected={open?.id === b.id}
                      onClick={() => setParam(b.id)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); setParam(b.id) } }}>
                      <td data-l="مشتری"><Who name={b.name} phone={b.phone} /></td>
                      <td data-l="فاکتور"><span className="mono">{b.invoice}</span></td>
                      <td data-l="آخرین فعالیت">{b.last}{b.hot ? <span className="sl-hot">فعال</span> : null}</td>
                      <td data-l="زمان" className="muted">{b.when}</td>
                      <td data-l="تعداد">{b.count === null ? <StatusIndicator state="incomplete" label="نامعلوم" /> : `${toFaDigits(b.count)} رویداد`}</td>
                      <td className="sl-col-actions"><div className="sl-row-actions"><button type="button" className="btn btn-sm" onClick={e => { e.stopPropagation(); setParam(b.id) }}>مشاهده روند</button></div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <TableFoot page={pg} perPage={PER_PAGE} onPage={setPage} />
          </>
        )}
      </Panel>
      <BehaviorDrawer b={open} onClose={() => setParam(null)} onInvoice={goInvoice} />
    </>
  )
}

function BehaviorDrawer({ b, onClose, onInvoice }: { b: BehaviorItem | null; onClose: () => void; onInvoice: (code: string) => void }) {
  const hasInvoice = b ? INVOICES.some(i => i.code === b.invoice) : false
  return (
    <Drawer open={b !== null} onClose={onClose} title={b?.name ?? 'روند رفتار مشتری'} kicker="روند رفتار مشتری"
      footer={b ? (
        <>
          <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
          <button type="button" className="btn btn-lg" disabled={!hasInvoice} onClick={() => onInvoice(b.invoice)}>مشاهده فاکتور</button>
        </>
      ) : undefined}
    >
      {b ? (
        <>
          <div className="sl-dr-head">
            <div className="sl-dr-meta"><span className="mono">{toFaDigits(b.phone)}</span><span>فاکتور <span className="mono">{b.invoice}</span></span>{b.hot ? <span className="sl-hot">فعال</span> : null}</div>
          </div>
          <section className="ds-sec" aria-label="رویدادها">
            <Timeline events={b.events} label="رویدادهای مشتری" />
            {b.count === null ? <p className="muted sl-meta-text">شمار کل رویدادها در دسترس نیست؛ فهرست بالا کامل نیست.</p> : null}
          </section>
        </>
      ) : null}
    </Drawer>
  )
}
