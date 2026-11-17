import { useMemo, useState } from 'react'
import {
  Banner, ContextHint, Drawer, DsIcon, MenuButton, Metric, MetricGroup, PageHead, Panel, QueueTabs, StatusIndicator, TonePill, Toolbar,
  toFaDigits, type DsQueue, type MenuEntry,
} from '../../../shared/ui/ds'
import { FilterChip, InvoiceStatusPill, Money, Note, SearchField, TableFoot, Timeline, Who, useListState } from '../components/bits'
import { INVOICES } from '../fixtures/sellerFixtures'
import { DEMO_SUFFIX, INVOICE_ACTION, INVOICE_QUEUE_LABEL, INVOICE_STATUS, STAGE_STATE } from '../labels'
import { useDrawerParam } from '../state/hooks'
import { INVOICE_STATUSES, collectibleTotal, filterInvoices, invoiceAmounts, invoiceCounts, paginate, type Payable } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { Invoice, InvoiceQueue } from '../types'

/*
 * S4 — فاکتورها (prototype V.invoices + invoiceDrawer). SYNTHETIC invoices only.
 * Money is shown as four separate facts: total · financially confirmed · remaining · current payable.
 * A submitted receipt is never counted as paid; an Outcome-Unknown submission is its own state.
 * No invoice edit, resend, receipt upload, payment, approval or posting happens: buttons only toast.
 */

const PER_PAGE = 10
const PRODUCTS = Array.from(new Set(INVOICES.map(i => i.product))).map(p => ({ id: p, label: p }))

export function PayableCell({ p }: { p: Payable }) {
  switch (p.kind) {
    case 'amount': return <Money value={p.value} />
    case 'settled': return <span className="sl-settled"><DsIcon name="check" />تسویه</span>
    case 'awaiting_finance': return <span className="sl-muted-pill">در انتظار تایید مالی</span>
    case 'unknown': return <StatusIndicator state="unknown" label="نامعلوم" />
    case 'none': return <span className="muted">—</span>
  }
}

function progress(inv: Invoice): number {
  const a = invoiceAmounts(inv)
  return a.total ? Math.round((a.confirmed / a.total) * 100) : 0
}

export function InvoicesPage() {
  const { toast, setSim } = useSellerPreview()
  const [queue, setQueue] = useState<InvoiceQueue>('all')
  const [q, setQ] = useState('')
  const [product, setProduct] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [invParam, setInvParam] = useDrawerParam('inv')
  const counts = useMemo(() => invoiceCounts(INVOICES), [])
  const list = useMemo(() => filterInvoices(INVOICES, queue, q, product), [queue, q, product])
  const pg = paginate(list, page, PER_PAGE)
  const collectible = collectibleTotal(INVOICES)
  const open = invParam ? INVOICES.find(i => i.code === invParam) ?? null : null
  const filtersActive = q !== '' || product !== null
  const clear = () => { setQ(''); setProduct(null); setPage(1) }

  const stateEl = useListState({
    count: pg.items.length, noResult: filtersActive, onClear: clear, onRetry: () => setSim('none'),
    emptyTitle: 'فاکتوری در این وضعیت وجود ندارد', emptyDetail: 'وضعیت دیگری را انتخاب کنید یا جستجو را تغییر دهید.',
  })

  const queues: DsQueue[] = [
    { id: 'all', label: 'همه', tone: 'neutral', count: counts.all },
    ...INVOICE_STATUSES.map<DsQueue>(k => ({ id: k, label: INVOICE_QUEUE_LABEL[k], tone: INVOICE_STATUS[k].tone, icon: INVOICE_STATUS[k].icon, count: counts[k] })),
  ]

  const moreMenu = (inv: Invoice): MenuEntry[] => {
    const a = invoiceAmounts(inv)
    const items: MenuEntry[] = [{ kind: 'item', id: 'detail', label: 'جزئیات و سوابق مالی', icon: 'history', onSelect: () => setInvParam(inv.code) }]
    if (a.confirmed === 0 && inv.status === 'pre_invoice') items.push({ kind: 'item', id: 'edit', label: 'ویرایش پیش از پرداخت', icon: 'file', disabled: true, disabledReason: 'در پیش‌نمایش ویرایش فاکتور فعال نیست.' })
    if (inv.status !== 'approved' && inv.status !== 'cancelled') {
      items.push({ kind: 'item', id: 'resend', label: 'ارسال مجدد لینک پرداخت', icon: 'send', disabled: true, disabledReason: 'پیش‌نمایش پیامکی ارسال نمی‌کند.' })
      items.push({ kind: 'item', id: 'copy', label: 'کپی لینک پرداخت', icon: 'copy', onSelect: () => toast(`نمایشی: لینکی برای ${inv.code} وجود ندارد و چیزی کپی نشد.`) })
    }
    return items
  }

  const primary = (inv: Invoice) => {
    const a = inv.next ? INVOICE_ACTION[inv.next] : null
    return a
      ? <button type="button" className="btn btn-soft btn-sm" onClick={e => { e.stopPropagation(); setInvParam(inv.code) }}><DsIcon name={a.icon} />{a.label}</button>
      : <button type="button" className="btn btn-sm" onClick={e => { e.stopPropagation(); setInvParam(inv.code) }}>جزئیات</button>
  }

  return (
    <>
      <PageHead
        title="فاکتورها"
        sub="فقط فاکتورهای صادرشده توسط شما (نمونه ساختگی)"
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="در انتظار پرداخت" value={toFaDigits(counts.pre_invoice + counts.staged)} accent="violet" />
            <Metric label="در بررسی مالی" value={toFaDigits(counts.finance_review)} accent="orange" />
            <Metric label="رد شده" value={toFaDigits(counts.rejected)} accent="red" tone="bad" />
            <Metric label="مانده قابل وصول" value={collectible.value.toLocaleString('fa-IR')} unit="تومان" accent="teal"
              meaning={collectible.hasUnknown ? 'یک ارسال نامعلوم؛ جمع نهایی نیست' : undefined} />
          </MetricGroup>
        }
      />
      <Panel label="فهرست فاکتورها" className="sl-has-cards">
        <QueueTabs queues={queues} value={queue} onChange={id => { setQueue(id as InvoiceQueue); setPage(1); setInvParam(null) }} />
        <Toolbar>
          <div className="sl-grow-field"><SearchField id="sl-inv-q" label="جستجوی فاکتور" placeholder="کد فاکتور، نام یا موبایل مشتری" value={q} onChange={v => { setQ(v); setPage(1) }} /></div>
          <FilterChip label="محصول" value={product} options={PRODUCTS} onChange={v => { setProduct(v); setPage(1) }} />
          <span className="ds-grow" />
          <span className="muted sl-desk" role="status">{toFaDigits(list.length)} فاکتور</span>
        </Toolbar>
        {stateEl ?? (
          <>
            <div className="sl-tbl-wrap">
              <table className="sl-tbl" aria-label="فهرست فاکتورها">
                <thead>
                  <tr>
                    <th scope="col">کد</th><th scope="col">مشتری</th><th scope="col" className="sl-col-opt">محصول</th>
                    <th scope="col">مبلغ و پیشرفت پرداخت <ContextHint text="پیشرفت فقط مبالغ تایید‌شده مالی را می‌شمارد؛ فیش ارسال‌شده تا تایید مالی پرداخت‌شده نیست." /></th>
                    <th scope="col">مانده</th><th scope="col">قابل پرداخت فعلی</th>
                    <th scope="col">وضعیت مالی <ContextHint text="وضعیت بررسی یا پرداخت فاکتور." /></th>
                    <th scope="col" className="sl-col-actions">اقدام بعدی</th>
                  </tr>
                </thead>
                <tbody>
                  {pg.items.map(inv => {
                    const a = invoiceAmounts(inv)
                    const pct = progress(inv)
                    const st = inv.stages.length
                    const cur = inv.stages.find(s => s.state === 'due' || s.state === 'rejected' || s.state === 'submitted' || s.state === 'unknown')
                    return (
                      <tr key={inv.code} data-row={inv.code} tabIndex={0} className={open?.code === inv.code ? 'sl-sel' : undefined} aria-selected={open?.code === inv.code}
                        onClick={() => setInvParam(inv.code)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); setInvParam(inv.code) } }}>
                        <td><span className="mono sl-code">{inv.code}</span></td>
                        <td><Who name={inv.customer} phone={inv.phone} /></td>
                        <td className="sl-col-opt">{inv.product}</td>
                        <td>
                          <div className="sl-prog">
                            <div className="sl-line"><Money value={a.total} /><span>{st > 1 ? `مرحله ${toFaDigits(cur?.n ?? st)} از ${toFaDigits(st)}` : 'پرداخت یک‌جا'}</span></div>
                            <div className="sl-bar" role="img" aria-label={`${toFaDigits(pct)}٪ تایید مالی شده`}><i style={{ width: `${pct}%` }} /></div>
                          </div>
                        </td>
                        <td>{a.remaining === null ? <span className="muted">—</span> : a.remaining <= 0 ? <span className="sl-settled"><DsIcon name="check" />تسویه</span> : <Money value={a.remaining} />}</td>
                        <td><PayableCell p={a.payable} /></td>
                        <td>
                          <InvoiceStatusPill inv={inv} />
                          {inv.rejectReason ? <div className="sl-reject">{inv.rejectReason}</div> : null}
                          {a.unknown > 0 ? <div className="sl-sub-pill"><StatusIndicator state="unknown" label="ارسال نامعلوم" /></div> : null}
                        </td>
                        <td className="sl-col-actions">
                          <div className="sl-row-actions">
                            {primary(inv)}
                            <MenuButton label={`اقدام‌های بیشتر ${inv.code}`} className="btn btn-ghost btn-sm ds-btn-icon" entries={moreMenu(inv)}><DsIcon name="more" /></MenuButton>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <div className="sl-card-list" aria-label="فهرست فاکتورها (کارت)">
              {pg.items.map(inv => {
                const a = invoiceAmounts(inv)
                const act = inv.next ? INVOICE_ACTION[inv.next] : null
                return (
                  <article key={inv.code} className="sl-lcard" aria-label={`فاکتور ${inv.code}`}>
                    <div className="sl-top"><Who name={inv.customer} sub={<>فاکتور <span className="mono">{inv.code}</span></>} /><InvoiceStatusPill inv={inv} /></div>
                    <div className="sl-meta"><span>کل <Money value={a.total} /></span><span>مانده {a.remaining === null ? '—' : <Money value={a.remaining} />}</span><span>قابل پرداخت <PayableCell p={a.payable} /></span></div>
                    <div className="sl-acts">
                      <MenuButton label={`اقدام‌های بیشتر ${inv.code}`} className="btn" entries={moreMenu(inv)}><DsIcon name="more" />بیشتر</MenuButton>
                      <button type="button" className={`btn${act ? ' btn-soft' : ''}`} onClick={() => setInvParam(inv.code)}>{act ? act.label : 'جزئیات'}</button>
                    </div>
                  </article>
                )
              })}
            </div>
            <TableFoot page={pg} perPage={PER_PAGE} onPage={setPage} />
          </>
        )}
      </Panel>
      <InvoiceDrawer inv={open} onClose={() => setInvParam(null)} />
    </>
  )
}

export function InvoiceDrawer({ inv, onClose }: { inv: Invoice | null; onClose: () => void }) {
  const { toast } = useSellerPreview()
  const a = inv ? invoiceAmounts(inv) : null
  const act = inv?.next ? INVOICE_ACTION[inv.next] : null
  return (
    <Drawer open={inv !== null} onClose={onClose} title={inv ? `فاکتور ${inv.code}` : 'فاکتور'} kicker="فاکتور" wide
      footerHint="پیش‌نمایش: هیچ اقدام مالی انجام نمی‌شود"
      footer={inv ? (
        <>
          <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
          {act ? <button type="button" className="btn btn-lg btn-primary" onClick={() => toast(`«${act.label}» برای ${inv.code} ${DEMO_SUFFIX}`)}><DsIcon name={act.icon} />{act.label}</button> : null}
        </>
      ) : undefined}
    >
      {inv && a ? (
        <>
          <div className="sl-dr-head">
            <div className="sl-dr-title"><InvoiceStatusPill inv={inv} /></div>
            <div className="sl-dr-meta"><span className="sl-strong-text">{inv.customer}</span><span className="mono">{toFaDigits(inv.phone)}</span><span>{inv.product}</span></div>
            <div className="sl-dr-quick">
              <button type="button" className="btn" onClick={() => toast(`نمایشی: لینکی برای ${inv.code} وجود ندارد و چیزی کپی نشد.`)}><DsIcon name="copy" />کپی لینک پرداخت</button>
            </div>
          </div>
          <section className="ds-sec sl-sec-primary" aria-label="خلاصه مبالغ">
            <div className="sl-sum-grid">
              <div className="sl-sum"><span>مبلغ کل</span><b><Money value={a.total} /></b></div>
              <div className="sl-sum"><span>پرداخت تایید‌شده مالی</span><b className="sl-green"><Money value={a.confirmed} /></b></div>
              <div className="sl-sum"><span>مانده</span><b>{a.remaining === null ? <span className="muted">—</span> : <Money value={a.remaining} />}</b></div>
              <div className="sl-sum"><span>قابل پرداخت فعلی</span><b><PayableCell p={a.payable} /></b></div>
            </div>
            {a.submitted > 0 ? <Note tone="warn" icon="clock">مبلغ <Money value={a.submitted} /> فیش ارسال‌شده است و تا تایید مالی، پرداخت‌شده یا کسرشده از مانده حساب نمی‌شود.</Note> : null}
            {a.unknown > 0 ? (
              <Banner kind="unknown">نتیجه ارسال فیش به مبلغ <Money value={a.unknown} /> نامعلوم است؛ ممکن است ثبت شده باشد. پیش از هر ارسال مجدد باید تطبیق شود.</Banner>
            ) : null}
            {inv.rejectReason ? <Note tone="danger" icon="alert">دلیل رد مالی: {inv.rejectReason}</Note> : null}
          </section>
          <details className="sl-details" open>
            <summary><h3>مراحل پرداخت</h3><span className="muted sl-meta-text">{toFaDigits(inv.stages.length)} مرحله</span><DsIcon name="chev" className="sl-chev" /></summary>
            <ol className="sl-stages" aria-label="مراحل پرداخت">
              {inv.stages.map(s => {
                const p = STAGE_STATE[s.state]
                return (
                  <li key={s.n} className={`sl-stage sl-stage-${s.state}`}>
                    <span className="sl-n" aria-hidden="true">{toFaDigits(s.n)}</span>
                    <div className="sl-grow">مرحله {toFaDigits(s.n)}<div className="sl-stage-state">
                      {s.state === 'unknown' ? <StatusIndicator state="unknown" label={p.label} /> : <TonePill tone={p.tone} icon={p.icon} label={p.label} />}
                      {s.note ? <span className="muted sl-meta-text">{s.note}</span> : null}
                    </div></div>
                    <Money value={s.amount} />
                  </li>
                )
              })}
            </ol>
          </details>
          <details className="sl-details" open>
            <summary><h3>سوابق مالی و شواهد</h3><DsIcon name="chev" className="sl-chev" /></summary>
            <Timeline events={inv.history} label="سوابق مالی" />
            <p className="muted sl-meta-text">زمان‌ها «زمان رویداد» نمونه‌اند؛ زمان ثبت مالی جداگانه است و در این پیش‌نمایش نمایش داده نمی‌شود.</p>
          </details>
        </>
      ) : null}
    </Drawer>
  )
}
