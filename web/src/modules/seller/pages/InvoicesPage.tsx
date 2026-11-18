import { useMemo, useState } from 'react'
import {
  Banner, ContextHint, Drawer, DsIcon, MenuButton, Metric, MetricGroup, PageHead, Panel, QueueTabs, StatusIndicator, Toolbar,
  toFaDigits, type DsQueue, type MenuEntry,
} from '../../../shared/ui/ds'
import { FilterChip, InvoiceStatusPill, Money, Note, SearchField, StatusPill, TableFoot, Timeline, UNIT_LABEL, Who, useListState } from '../components/bits'
import { NotFoundDrawer, ScreenFrame, Withheld, useDetail, useStatePresentation } from '../components/states'
import { INVOICES } from '../fixtures/sellerFixtures'
import { DEMO_SUFFIX, INVOICE_ACTION, INVOICE_QUEUE_LABEL, INVOICE_STATUS, STAGE_STATE } from '../labels'
import { useDrawerParam } from '../state/hooks'
import { INVOICE_STATUSES, collectibleTotal, filterInvoices, invoiceAmounts, invoiceCounts, paginate, unitOf, type InvoiceAmounts, type Payable } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { StatePolicy } from '../state/uiState'
import type { Invoice, InvoiceQueue, MoneyUnit } from '../types'

/*
 * S4 — فاکتورها (prototype V.invoices + invoiceDrawer). SYNTHETIC invoices only.
 * Money is shown as four separate facts: total · financially confirmed · remaining · current payable.
 * A submitted receipt is never counted as paid; an Outcome-Unknown submission is its own state.
 * No invoice edit, resend, receipt upload, payment, approval or posting happens: buttons only toast.
 */

const PER_PAGE = 10
const PRODUCTS = Array.from(new Set(INVOICES.map(i => i.product))).map(p => ({ id: p, label: p }))

export function PayableCell({ p, unit = 'IRT' }: { p: Payable; unit?: MoneyUnit | undefined }) {
  switch (p.kind) {
    case 'amount': return <Money value={p.value} unit={unit} />
    case 'settled': return <span className="sl-settled"><DsIcon name="check" />تسویه</span>
    case 'awaiting_finance': return <span className="sl-muted-pill">در انتظار تایید مالی</span>
    case 'unknown': return <StatusIndicator state="unknown" label="نامعلوم" />
    case 'none': return <span className="muted">—</span>
    case 'withheld': return <Withheld />
  }
}

/**
 * A7: amounts that depend on stage facts. PARTIAL withholds them; an unmapped status makes them unknown.
 * Stage amounts and the invoice total are declared facts and stay visible.
 */
export function presentAmounts(a: InvoiceAmounts, policy: StatePolicy): { remaining: 'withheld' | number | null; payable: Payable } {
  if (policy.dependentAmountsWithheld) return { remaining: 'withheld', payable: { kind: 'withheld' } }
  if (policy.statusUnknown) return { remaining: null, payable: { kind: 'unknown' } }
  return { remaining: a.remaining, payable: a.payable }
}

function RemainingCell({ r, unit }: { r: 'withheld' | number | null; unit: MoneyUnit }) {
  if (r === 'withheld') return <Withheld />
  if (r === null) return <span className="muted">—</span>
  return r <= 0 ? <span className="sl-settled"><DsIcon name="check" />تسویه</span> : <Money value={r} unit={unit} />
}

function progress(inv: Invoice): number {
  const a = invoiceAmounts(inv)
  return a.total ? Math.round((a.confirmed / a.total) * 100) : 0
}

export function InvoicesPage() {
  const { toast } = useSellerPreview()
  const sp = useStatePresentation()
  const identity = !sp.policy.fieldsRestricted
  const invoices = sp.data(INVOICES)
  const [queue, setQueue] = useState<InvoiceQueue>('all')
  const [q, setQ] = useState('')
  const [product, setProduct] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [invParam, setInvParam] = useDrawerParam('inv')
  const counts = useMemo(() => invoiceCounts(invoices), [invoices])
  const list = useMemo(() => filterInvoices(invoices, queue, q, product, identity), [invoices, queue, q, product, identity])
  const pg = paginate(list, page, PER_PAGE)
  const collectible = collectibleTotal(invoices)
  const detail = useDetail(invParam, id => invoices.find(i => i.code === id))
  const open = detail.record
  const filtersActive = q !== '' || product !== null
  const clear = () => { setQ(''); setProduct(null); setPage(1) }

  const stateEl = useListState({
    count: pg.items.length, noResult: filtersActive, onClear: clear,
    emptyTitle: 'فاکتوری در این وضعیت وجود ندارد', emptyDetail: 'وضعیت دیگری را انتخاب کنید یا جستجو را تغییر دهید.',
  })

  const queues: DsQueue[] = [
    { id: 'all', label: 'همه', tone: 'neutral', count: sp.count(counts.all) },
    ...INVOICE_STATUSES.map<DsQueue>(k => ({ id: k, label: INVOICE_QUEUE_LABEL[k], tone: INVOICE_STATUS[k].tone, icon: INVOICE_STATUS[k].icon, count: sp.count(counts[k]) })),
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

  const collectibleWithheld = sp.policy.dependentAmountsWithheld || sp.policy.statusUnknown
  return (
    <ScreenFrame title="فاکتورها" head={
      <PageHead
        title="فاکتورها"
        sub="فقط فاکتورهای صادرشده توسط شما (نمونه ساختگی)"
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="در انتظار پرداخت" {...sp.kpi(counts.pre_invoice + counts.staged)} accent="violet" />
            <Metric label="در بررسی مالی" {...sp.kpi(counts.finance_review)} accent="orange" />
            <Metric label="رد شده" {...sp.kpi(counts.rejected)} accent="red" tone="bad" />
            {collectibleWithheld || collectible.byUnit.length === 0
              ? <Metric label="مانده قابل وصول" accent="teal" state={collectibleWithheld ? 'unknown' : 'ok'} value={collectibleWithheld ? undefined : '۰'}
                meaning={collectibleWithheld ? (sp.policy.statusUnknown ? 'وضعیت مراحل نامعلوم؛ محاسبه نشد' : 'به داده ناقص وابسته است؛ نمایش داده نمی‌شود') : undefined} />
              : collectible.byUnit.map(u => (
                <Metric key={u.unit} label={`مانده قابل وصول (${UNIT_LABEL[u.unit]})`} value={u.value.toLocaleString('fa-IR')} unit={UNIT_LABEL[u.unit]} accent="teal"
                  meaning={collectible.hasUnknown ? 'یک ارسال نامعلوم؛ جمع نهایی نیست' : 'واحدها با هم جمع نمی‌شوند'} />
              ))}
          </MetricGroup>
        }
      />}>
      <Panel label="فهرست فاکتورها" className="sl-has-cards">
        <QueueTabs queues={queues} value={queue} onChange={id => { setQueue(id as InvoiceQueue); setPage(1); setInvParam(null) }} />
        <Toolbar>
          <div className="sl-grow-field"><SearchField id="sl-inv-q" label="جستجوی فاکتور" placeholder="کد فاکتور، نام یا موبایل مشتری" value={q} onChange={v => { setQ(v); setPage(1) }} /></div>
          <FilterChip label="محصول" value={product} options={PRODUCTS} onChange={v => { setProduct(v); setPage(1) }} />
          <span className="ds-grow" />
          <span className="muted sl-desk" role="status">{sp.countText(list.length)} فاکتور</span>
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
                    const u = unitOf(inv)
                    const v = presentAmounts(a, sp.policy)
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
                            <div className="sl-line"><Money value={a.total} unit={u} /><span>{st > 1 ? `مرحله ${toFaDigits(cur?.n ?? st)} از ${toFaDigits(st)}` : 'پرداخت یک‌جا'}</span></div>
                            <div className="sl-bar" role="img" aria-label={`${toFaDigits(pct)}٪ تایید مالی شده`}><i style={{ width: `${pct}%` }} /></div>
                          </div>
                        </td>
                        <td><RemainingCell r={v.remaining} unit={u} /></td>
                        <td><PayableCell p={v.payable} unit={u} /></td>
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
                const u = unitOf(inv)
                const v = presentAmounts(a, sp.policy)
                const act = inv.next ? INVOICE_ACTION[inv.next] : null
                return (
                  <article key={inv.code} className="sl-lcard" aria-label={`فاکتور ${inv.code}`}>
                    <div className="sl-top"><Who name={inv.customer} sub={<>فاکتور <span className="mono">{inv.code}</span></>} /><InvoiceStatusPill inv={inv} /></div>
                    <div className="sl-meta"><span>کل <Money value={a.total} unit={u} /></span><span>مانده <RemainingCell r={v.remaining} unit={u} /></span><span>قابل پرداخت <PayableCell p={v.payable} unit={u} /></span></div>
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
      <NotFoundDrawer open={detail.notFound} kicker="فاکتور" onClose={() => setInvParam(null)} />
    </ScreenFrame>
  )
}

export function InvoiceDrawer({ inv, onClose }: { inv: Invoice | null; onClose: () => void }) {
  const { toast, ui } = useSellerPreview()
  const a = inv ? invoiceAmounts(inv) : null
  const u = inv ? unitOf(inv) : 'IRT'
  const v = a ? presentAmounts(a, ui.policy) : null
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
      {inv && a && v ? (
        <>
          <div className="sl-dr-head">
            <div className="sl-dr-title"><InvoiceStatusPill inv={inv} /></div>
            <div className="sl-dr-meta"><Who name={inv.customer} phone={inv.phone} copy={false} /><span>{inv.product}</span></div>
            <div className="sl-dr-quick">
              <button type="button" className="btn" onClick={() => toast(`نمایشی: لینکی برای ${inv.code} وجود ندارد و چیزی کپی نشد.`)}><DsIcon name="copy" />کپی لینک پرداخت</button>
            </div>
          </div>
          <section className="ds-sec sl-sec-primary" aria-label="خلاصه مبالغ">
            <div className="sl-sum-grid">
              <div className="sl-sum"><span>مبلغ کل</span><b><Money value={a.total} unit={u} /></b></div>
              <div className="sl-sum"><span>پرداخت تایید‌شده مالی</span><b className="sl-green">{ui.policy.dependentAmountsWithheld ? <Withheld /> : ui.policy.statusUnknown ? <Money value={null} /> : <Money value={a.confirmed} unit={u} />}</b></div>
              <div className="sl-sum"><span>مانده</span><b><RemainingCell r={v.remaining} unit={u} /></b></div>
              <div className="sl-sum"><span>قابل پرداخت فعلی</span><b><PayableCell p={v.payable} unit={u} /></b></div>
            </div>
            {a.submitted > 0 ? <Note tone="warn" icon="clock">مبلغ <Money value={a.submitted} unit={u} /> فیش ارسال‌شده است و تا تایید مالی، پرداخت‌شده یا کسرشده از مانده حساب نمی‌شود.</Note> : null}
            {a.unknown > 0 ? (
              <Banner kind="unknown">نتیجه ارسال فیش به مبلغ <Money value={a.unknown} unit={u} /> نامعلوم است؛ ممکن است ثبت شده باشد. پیش از هر ارسال مجدد باید تطبیق شود.</Banner>
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
                      {s.state === 'unknown' ? <StatusIndicator state="unknown" label={p.label} /> : <StatusPill tone={p.tone} icon={p.icon} label={p.label} />}
                      {s.note ? <span className="muted sl-meta-text">{s.note}</span> : null}
                    </div></div>
                    <Money value={s.amount} unit={u} />
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
