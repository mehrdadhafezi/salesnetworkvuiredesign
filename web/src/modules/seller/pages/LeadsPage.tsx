import { useCallback, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import {
  ContextHint, Drawer, DsIcon, Metric, MetricGroup, PageHead, Panel, QueueTabs, Toolbar, toFaDigits, type DsQueue,
} from '../../../shared/ui/ds'
import {
  FactChip, FilterChip, InvoiceStatusPill, LeadStatusPill, Money, NextStep, Note, SearchField, TableFoot, Tag, Timeline, Who,
  nextChipClass, useListState,
} from '../components/bits'
import { OutcomeGrid } from '../components/overlays'
import { INVOICES, LEADS } from '../fixtures/sellerFixtures'
import { CLOSED_LEAD_STATUSES, DEMO_SUFFIX, LEAD_OUTCOMES, LEAD_QUEUE_ORDER, LEAD_STATUS } from '../labels'
import { useDrawerParam, usePageKeys } from '../state/hooks'
import {
  DEFAULT_LEAD_FILTER, filterLeads, followUpRate, invoiceAmounts, leadCounts, paginate, validateOutcome,
  type DateRange, type FieldErrors, type LeadFilter, type OutcomeFields,
} from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { Lead, LeadOutcome, LeadQueue } from '../types'

/*
 * S1 — شماره‌های من (prototype V.leads + leadDrawer). SYNTHETIC leads only.
 * Queue / search / filters / sort / pagination / drawer / outcome selection are real local
 * interactions. «ثبت» never records anything: it validates and shows a demo toast.
 */

const PER_PAGE = 10
const SOURCES = [{ id: 'MIS', label: 'MIS' }, { id: 'درخواستی', label: 'درخواستی' }, { id: 'ارتباط مجدد', label: 'ارتباط مجدد' }]
const PROBS = [{ id: 'ضعیف', label: 'ضعیف' }, { id: 'متوسط', label: 'متوسط' }, { id: 'بالا', label: 'بالا' }, { id: '۱۰۰٪', label: '۱۰۰٪' }, { id: '—', label: 'تعیین‌نشده' }]
const RANGES: { id: Exclude<DateRange, 'all'>; label: string }[] = [{ id: '7', label: '۷ روز اخیر' }, { id: '30', label: '۳۰ روز اخیر' }]

function isQueue(v: string | null): v is LeadQueue {
  return v !== null && (LEAD_QUEUE_ORDER as readonly string[]).includes(v)
}

export function LeadsPage() {
  const { toast, setSim, base } = useSellerPreview()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [filter, setFilter] = useState<LeadFilter>(() => ({ ...DEFAULT_LEAD_FILTER, queue: isQueue(params.get('queue')) ? params.get('queue') as LeadQueue : 'none' }))
  const [page, setPage] = useState(1)
  const [cursor, setCursor] = useState(0)
  const [leadParam, setLeadParam] = useDrawerParam('lead')
  const tbody = useRef<HTMLTableSectionElement>(null)
  const search = useRef<HTMLDivElement>(null)

  const counts = useMemo(() => leadCounts(LEADS), [])
  const list = useMemo(() => filterLeads(LEADS, filter), [filter])
  const pg = paginate(list, page, PER_PAGE)
  const rate = followUpRate(LEADS)
  const openLead = leadParam ? LEADS.find(l => String(l.id) === leadParam) ?? null : null

  const update = (patch: Partial<LeadFilter>) => { setFilter(f => ({ ...f, ...patch })); setPage(1); setCursor(0) }
  const switchQueue = useCallback((q: LeadQueue) => { setFilter(f => ({ ...f, queue: q })); setPage(1); setCursor(0); setLeadParam(null) }, [setLeadParam])
  const filtersActive = filter.q !== '' || filter.source !== null || filter.prob !== null || filter.range !== 'all'
  const clearFilters = () => update({ q: '', source: null, prob: null, range: 'all' })

  const focusRow = (i: number) => {
    const rows = tbody.current?.querySelectorAll<HTMLTableRowElement>('tr[data-row]')
    if (!rows?.length) return
    const n = Math.max(0, Math.min(rows.length - 1, i))
    setCursor(n)
    rows[n]?.focus({ preventScroll: false })
  }

  /** J/K: next / previous lead — in the drawer it steps through the current list (prototype step()). */
  const step = useCallback((d: number) => {
    if (!openLead) { focusRow(cursor + d); return }
    const src = list.some(l => l.id === openLead.id) ? list : LEADS
    const i = src.findIndex(l => l.id === openLead.id)
    const n = src[(i + d + src.length) % src.length]
    if (n) setLeadParam(String(n.id))
  }, [openLead, list, cursor, setLeadParam])

  usePageKeys(useCallback((k: string) => {
    if (openLead) return false // the drawer owns 1–5 / Enter / J / K while open (handled inside it)
    if (k === 'f') { search.current?.querySelector('input')?.focus(); return true }
    if (k === 'j') { step(1); return true }
    if (k === 'k') { step(-1); return true }
    if (/^[0-6]$/.test(k)) {
      const q = k === '0' ? 'all' : LEAD_QUEUE_ORDER[Number(k) - 1]
      if (q) { switchQueue(q); return true }
    }
    return false
  }, [openLead, step, switchQueue]))

  const queues: DsQueue[] = LEAD_QUEUE_ORDER.map(k => k === 'all'
    ? { id: 'all', label: 'همه', tone: 'neutral', count: counts.all, shortcut: '0' }
    : { id: k, label: LEAD_STATUS[k].label, tone: LEAD_STATUS[k].tone, icon: LEAD_STATUS[k].icon, count: counts[k], shortcut: LEAD_STATUS[k].key })

  const onRowKey = (e: ReactKeyboardEvent<HTMLTableRowElement>, i: number, lead: Lead) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); focusRow(i + 1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); focusRow(i - 1) }
    else if (e.key === 'Enter') { e.preventDefault(); setLeadParam(String(lead.id)) }
  }

  const stateEl = useListState({
    count: pg.items.length, noResult: filtersActive, onClear: clearFilters, onRetry: () => setSim('none'),
    emptyTitle: 'در این صف موردی وجود ندارد', emptyDetail: 'شماره‌ای با این وضعیت ندارید.',
  })

  return (
    <>
      <PageHead
        title="شماره‌های من"
        sub={`${toFaDigits(LEADS.length)} شماره تخصیص‌یافته (ساختگی)`}
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="بدون وضعیت" value={toFaDigits(counts.none)} accent="neutral" />
            <Metric label="تماس مجدد امروز" value={toFaDigits(LEADS.filter(l => l.next?.kind === 'callback' && l.next.due).length)} accent="blue" />
            <Metric label="نرخ پیگیری" value={rate === null ? undefined : toFaDigits(rate)} unit="٪" accent="teal" state={rate === null ? 'unknown' : 'ok'} />
            <Metric label="پیش‌فاکتور (نمونه)" value={toFaDigits(counts.invoiced)} accent="violet" />
          </MetricGroup>
        }
        actions={<button type="button" className="btn" onClick={() => void navigate(`${base}/request`)}><DsIcon name="plus" />درخواست شماره</button>}
      />
      <Panel label="فهرست شماره‌ها" className="sl-has-cards">
        <QueueTabs queues={queues} value={filter.queue} onChange={id => switchQueue(id as LeadQueue)} />
        <Toolbar>
          <div ref={search} className="sl-grow-field">
            <SearchField id="sl-lead-q" label="جستجو در این صف" placeholder="جستجو در این صف: نام، شماره یا شهر" value={filter.q} onChange={q => update({ q })} />
          </div>
          <FilterChip label="منبع" value={filter.source} options={SOURCES} onChange={source => update({ source })} />
          <FilterChip label="احتمال فروش" value={filter.prob} options={PROBS} onChange={prob => update({ prob })} />
          <FilterChip label="تاریخ تخصیص" icon="clock" value={filter.range === 'all' ? null : filter.range} options={RANGES} onChange={r => update({ range: r ?? 'all' })} />
          <span className="ds-grow" />
          <span className="muted sl-desk" role="status">{toFaDigits(list.length)} نتیجه</span>
          <button type="button" className="btn btn-ghost btn-sm sl-desk" aria-label={`مرتب‌سازی: ${filter.sort === 'newest' ? 'جدیدترین' : 'قدیمی‌ترین'}`}
            onClick={() => update({ sort: filter.sort === 'newest' ? 'oldest' : 'newest' })}>
            <DsIcon name="swap" />{filter.sort === 'newest' ? 'جدیدترین' : 'قدیمی‌ترین'}
          </button>
        </Toolbar>
        {stateEl ?? (
          <>
            <div className="sl-tbl-wrap">
              <table className="sl-tbl" aria-label="فهرست شماره‌ها">
                <thead>
                  <tr>
                    <th scope="col">وضعیت</th><th scope="col">مشتری</th><th scope="col">شهر</th>
                    <th scope="col" className="sl-col-opt">منبع</th><th scope="col" className="sl-col-opt">تخصیص</th>
                    <th scope="col">گام بعدی <ContextHint text="اقدام پیشنهادی بعدی بر اساس وضعیت فعلی پرونده." /></th>
                    <th scope="col" className="sl-col-opt">احتمال <ContextHint text="برای اولویت‌بندی پیگیری است و وضعیت مشتری را تغییر نمی‌دهد." /></th>
                    <th scope="col" className="sl-col-actions"><span className="sr">اقدام</span></th>
                  </tr>
                </thead>
                <tbody ref={tbody}>
                  {pg.items.map((l, i) => {
                    const sel = openLead?.id === l.id
                    const closed = CLOSED_LEAD_STATUSES.includes(l.status)
                    return (
                      <tr key={l.id} data-row={l.id} tabIndex={i === cursor ? 0 : -1} className={sel ? 'sl-sel' : undefined} aria-selected={sel}
                        onClick={() => setLeadParam(String(l.id))} onKeyDown={e => onRowKey(e, i, l)} onFocus={() => setCursor(i)}>
                        <td><LeadStatusPill lead={l} /></td>
                        <td><Who name={l.name} phone={l.phone} /></td>
                        <td>{l.city}</td>
                        <td className="sl-col-opt"><Tag teal={l.source !== 'MIS'}>{l.source}</Tag></td>
                        <td className="sl-col-opt muted">{l.assignedAgo}</td>
                        <td><NextStep next={l.next} />{l.next?.kind === 'attempts' ? <ContextHint text={`این مشتری ${toFaDigits(l.attempts)} بار پاسخ نداده است؛ سقف این مرحله ۳ تلاش است.`} /> : null}</td>
                        <td className="sl-col-opt">{l.prob || <span className="muted">—</span>}</td>
                        <td className="sl-col-actions">
                          <div className="sl-row-actions">
                            <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label={`تماس با ${l.name}`} title="تماس"
                              onClick={e => { e.stopPropagation(); toast('نمایشی: روی موبایل شماره‌گیر باز می‌شود؛ پیش‌نمایش تماسی برقرار نمی‌کند.') }}>
                              <DsIcon name="phone" />
                            </button>
                            <button type="button" className={`btn btn-sm${closed ? '' : ' btn-soft'}`}
                              onClick={e => { e.stopPropagation(); setLeadParam(String(l.id)) }}>
                              {closed ? (l.status === 'invoiced' ? 'مشاهده فاکتور' : 'مشاهده') : 'ثبت نتیجه'}
                            </button>
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            <div className="sl-card-list" aria-label="فهرست شماره‌ها (کارت)">
              {pg.items.map(l => {
                const closed = CLOSED_LEAD_STATUSES.includes(l.status)
                return (
                  <article key={l.id} className="sl-lcard" aria-label={l.name}>
                    <div className="sl-top"><Who name={l.name} phone={l.phone} copy={false} /><LeadStatusPill lead={l} /></div>
                    <div className="sl-meta"><span>{l.city}</span><span>{l.source}</span><span>{l.assignedAgo}</span>{l.next ? <NextStep next={l.next} /> : null}</div>
                    <div className="sl-acts">
                      <button type="button" className="btn sl-call" onClick={() => toast('نمایشی: شماره‌گیر باز نمی‌شود؛ پیش‌نمایش تماسی برقرار نمی‌کند.')}><DsIcon name="phone" />تماس</button>
                      <button type="button" className={`btn${closed ? '' : ' btn-soft'}`} onClick={() => setLeadParam(String(l.id))}>{closed ? 'مشاهده' : 'ثبت نتیجه'}</button>
                    </div>
                  </article>
                )
              })}
            </div>
            <TableFoot page={pg} perPage={PER_PAGE} onPage={p => { setPage(p); setCursor(0) }} />
          </>
        )}
      </Panel>
      <LeadDrawer lead={openLead} list={list} onClose={() => setLeadParam(null)} onStep={step} />
    </>
  )
}

const EMPTY_FIELDS: OutcomeFields = { date: '۱۴۰۵/۰۷/۱۰', hour: '10', minute: '30', reason: '', amount: '' }

function LeadDrawer({ lead, list, onClose, onStep }: { lead: Lead | null; list: Lead[]; onClose: () => void; onStep: (d: number) => void }) {
  const src = lead && list.some(l => l.id === lead.id) ? list : LEADS
  const idx = lead ? src.findIndex(l => l.id === lead.id) : -1
  return (
    <Drawer open={lead !== null} onClose={onClose} title={lead?.name ?? 'جزئیات'} kicker={lead ? `${toFaDigits(idx + 1)} از ${toFaDigits(src.length)}` : 'جزئیات'}
      topExtra={lead ? (
        <>
          <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label="مشتری قبلی (K)" title="قبلی (K)" onClick={() => onStep(-1)}><DsIcon name="chev" className="sl-rot180" /></button>
          <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label="مشتری بعدی (J)" title="بعدی (J)" onClick={() => onStep(1)}><DsIcon name="chev" /></button>
          <span className="sl-desk sl-micro">۱–۵ نتیجه · Enter ثبت · Esc بستن</span>
        </>
      ) : undefined}
    >
      {lead ? <LeadDrawerBody key={lead.id} lead={lead} onClose={onClose} onStep={onStep} /> : null}
    </Drawer>
  )
}

function LeadDrawerBody({ lead, onClose, onStep }: { lead: Lead; onClose: () => void; onStep: (d: number) => void }) {
  const { toast } = useSellerPreview()
  const [params] = useSearchParams()
  const initial = params.get('outcome')
  const [outcome, setOutcome] = useState<LeadOutcome | null>(LEAD_OUTCOMES.some(o => o.id === initial) ? initial as LeadOutcome : null)
  const [fields, setFields] = useState<OutcomeFields>(EMPTY_FIELDS)
  const [errors, setErrors] = useState<FieldErrors>({})
  const [draft, setDraft] = useState(lead.note)
  const body = useRef<HTMLDivElement>(null)
  const attemptsNext = Math.min(3, lead.attempts + 1)
  const invoices = INVOICES.filter(i => lead.invoiceRefs?.includes(i.code))
  const s = LEAD_STATUS[lead.status]

  const ctaText: Record<LeadOutcome, string> = {
    no_answer: `ثبت تماس بی‌پاسخ (${toFaDigits(attemptsNext)} از ۳)`,
    callback: `ثبت تماس مجدد — ${fields.date || '…'} ساعت ${toFaDigits(fields.hour || '…')}:${toFaDigits(fields.minute || '…')}`,
    not_purchased: 'ثبت عدم خرید',
    duplicate: 'ثبت به‌عنوان تکراری',
    success: 'صدور پیش‌فاکتور',
  }

  const pick = (o: LeadOutcome) => { setOutcome(o); setErrors({}) }
  const set = (k: keyof OutcomeFields) => (v: string) => { setFields(f => ({ ...f, [k]: v })); setErrors(e => ({ ...e, [k]: undefined })) }
  const commit = () => {
    if (!outcome) return
    const e = validateOutcome(outcome, fields)
    setErrors(e)
    const first = Object.keys(e)[0]
    if (first) { body.current?.querySelector<HTMLElement>(`[data-field="${first}"]`)?.focus(); return }
    toast(`«${ctaText[outcome]}» برای ${lead.name} ${DEMO_SUFFIX}`, 'info')
  }

  // Drawer keys (prototype): 1–5 pick an outcome, Enter commits, J/K step — only while focus is not on a control.
  usePageKeys(useCallback((k: string) => {
    if (/^[1-5]$/.test(k)) { const o = LEAD_OUTCOMES[Number(k) - 1]; if (o) { pick(o.id); return true } }
    if (k === 'enter' && outcome) { commit(); return true }
    if (k === 'j') { onStep(1); return true }
    if (k === 'k') { onStep(-1); return true }
    return false
  }, [outcome, fields, onStep])) // eslint-disable-line react-hooks/exhaustive-deps

  const err = (k: keyof FieldErrors) => errors[k] ? <div className="field-err" id={`sl-err-${k}`}><DsIcon name="alert" /><span>{errors[k]}</span></div> : null
  const inv = (k: keyof FieldErrors) => errors[k] ? { 'aria-invalid': true as const, 'aria-describedby': `sl-err-${k}` } : {}

  return (
    <div ref={body}>
      <div className="sl-dr-head">
        <div className="sl-dr-title"><LeadStatusPill lead={lead} /></div>
        <div className="sl-dr-meta"><span className="mono sl-dr-phone">{toFaDigits(lead.phone)}</span><span>{lead.city}، {lead.province}</span><Tag teal={lead.source !== 'MIS'}>{lead.source}</Tag></div>
        <div className="sl-dr-facts">
          <FactChip icon="history">آخرین فعالیت: {lead.lastActivity}</FactChip>
          {lead.next ? <FactChip icon="clock" cls={nextChipClass(lead.next)}>گام بعدی: {lead.next.text}</FactChip> : <FactChip icon="arrowL">گام بعدی: ثبت اولین نتیجه</FactChip>}
          <FactChip icon="clock">تخصیص: {lead.assigned}</FactChip>
        </div>
        <div className="sl-dr-quick">
          <button type="button" className="btn" onClick={() => toast('نمایشی: شماره‌گیر باز نمی‌شود؛ پیش‌نمایش تماسی برقرار نمی‌کند.')}><DsIcon name="phone" />تماس</button>
          <button type="button" className="btn" onClick={() => toast(`نمایشی: شماره ${toFaDigits(lead.phone)} کپی نشد.`)}><DsIcon name="copy" />کپی شماره</button>
        </div>
      </div>

      <section className="ds-sec sl-sec-primary" aria-labelledby="sl-outcome-h">
        <div className="ds-sec-h"><h3 id="sl-outcome-h">ثبت نتیجه تماس</h3><span className="ds-sec-aside">وضعیت فعلی: {s.label}</span></div>
        {lead.attempts >= 2 && outcome !== 'no_answer'
          ? <Note tone="warn" icon="alert">این مشتری {toFaDigits(lead.attempts)} بار پاسخ نداده است. ثبت بی‌پاسخ بعدی می‌تواند باعث انتقال پرونده به بایگانی مدیر فروش شود.</Note>
          : null}
        <OutcomeGrid options={LEAD_OUTCOMES} value={outcome} onChange={pick} label="نتیجه تماس"
          hintFor={id => (id === 'no_answer' ? `تلاش ${toFaDigits(attemptsNext)} از ۳` : null)} />
        {outcome === 'no_answer' ? (
          <div className="sl-cond">{attemptsNext >= 3
            ? <Note tone="warn" icon="alert">با ثبت این تلاش سقف ۳ تماس پر می‌شود. اگر تا ۳ روز فعالیت دیگری ثبت نشود، شماره به بایگانی مدیر فروش منتقل می‌شود.</Note>
            : <Note tone="info" icon="phoneOff">هر تماس واقعیِ بی‌پاسخ جدا ثبت می‌شود. این تلاش {toFaDigits(attemptsNext)} از ۳ است.</Note>}</div>
        ) : null}
        {outcome === 'callback' ? (
          <div className="sl-cond sl-cond-box">
            <div className="sl-row2">
              <label className="lbl">تاریخ شمسی <span className="sl-req" aria-hidden="true">*</span><span className="sr">(الزامی)</span>
                <input className="input" data-field="date" value={fields.date} onChange={e => set('date')(e.target.value)} {...inv('date')} />{err('date')}
              </label>
              <div className="lbl" role="group" aria-label="ساعت تهران (الزامی)">ساعت تهران <span className="sl-req" aria-hidden="true">*</span>
                <span className="sl-time">
                  <input className="input" data-field="hour" aria-label="ساعت" inputMode="numeric" value={fields.hour} onChange={e => set('hour')(e.target.value)} {...inv('hour')} />
                  :
                  <input className="input" data-field="minute" aria-label="دقیقه" inputMode="numeric" value={fields.minute} onChange={e => set('minute')(e.target.value)} {...inv('minute')} />
                </span>{err('hour')}{err('minute')}
              </div>
            </div>
            <p className="muted sl-meta-text">یادآور در نسخه واقعی برای همین زمان تنظیم می‌شود؛ پیش‌نمایش یادآوری نمی‌سازد.</p>
          </div>
        ) : null}
        {outcome === 'not_purchased' ? (
          <div className="sl-cond sl-cond-box">
            <label className="lbl">دلیل عدم خرید <span className="sl-req" aria-hidden="true">*</span><span className="sr">(الزامی)</span>
              <textarea className="input sl-textarea" data-field="reason" placeholder="دلیل اعلام‌شده یا نتیجه نهایی مکالمه" value={fields.reason} onChange={e => set('reason')(e.target.value)} {...inv('reason')} />{err('reason')}
            </label>
          </div>
        ) : null}
        {outcome === 'duplicate' ? <div className="sl-cond"><Note tone="info" icon="copy">شماره به‌عنوان تکراری علامت می‌خورد و نام شما به‌عنوان ثبت‌کننده ذخیره می‌شود (در نسخه واقعی).</Note></div> : null}
        {outcome === 'success' ? (
          <div className="sl-cond sl-cond-box">
            <div className="sl-row2">
              <label className="lbl">محصول <span className="sl-req" aria-hidden="true">*</span>
                <select className="input" defaultValue="p1"><option value="p1">اشتراک نمونه ۱۰ ستاره — ۱۰٬۰۰۰٬۰۰۰ تومان</option><option value="p2">اشتراک نمونه نقره‌ای</option></select>
              </label>
              <PayTypeSeg />
            </div>
            <Note tone="info" icon="send">در نسخه واقعی پس از صدور، پیش‌فاکتور مطابق تنظیمات فعلی برای مشتری پیامک می‌شود. پیش‌نمایش چیزی صادر یا ارسال نمی‌کند.</Note>
          </div>
        ) : null}
      </section>

      <details className="sl-details" open>
        <summary><h3>اطلاعات مشتری</h3><span className="muted sl-meta-text">فقط‌خواندنی در پیش‌نمایش</span><DsIcon name="chev" className="sl-chev" /></summary>
        <dl className="sl-kv">
          <div><dt>نام مشتری</dt><dd>{lead.name}</dd></div>
          <div><dt>احتمال فروش <ContextHint text="برای اولویت‌بندی پیگیری است و وضعیت مشتری را تغییر نمی‌دهد." /></dt><dd>{lead.prob || <span className="muted">—</span>}</dd></div>
          <div><dt>استان</dt><dd>{lead.province}</dd></div>
          <div><dt>شهر</dt><dd>{lead.city}</dd></div>
        </dl>
      </details>
      <details className="sl-details" open>
        <summary><h3>یادداشت</h3><span className="muted sl-meta-text">پیش‌نویس محلی — ذخیره نمی‌شود</span><DsIcon name="chev" className="sl-chev" /></summary>
        <label className="lbl"><span className="sr">یادداشت (پیش‌نویس محلی)</span>
          <textarea className="input sl-textarea" placeholder="خلاصه مکالمه، نیاز مشتری…" value={draft} onChange={e => setDraft(e.target.value)} />
        </label>
      </details>
      <details className="sl-details">
        <summary><h3>تاریخچه</h3><span className="muted sl-meta-text">{toFaDigits(lead.history.length)} رویداد</span><DsIcon name="chev" className="sl-chev" /></summary>
        <Timeline events={lead.history} label="تاریخچه تعامل" />
      </details>
      <details className="sl-details" open={invoices.length > 0}>
        <summary><h3>فاکتورهای این پرونده</h3><span className="muted sl-meta-text">{toFaDigits(invoices.length)}</span><DsIcon name="chev" className="sl-chev" /></summary>
        {invoices.length
          ? <div className="sl-mini-list">{invoices.map(i => <div key={i.code} className="sl-mini"><div className="sl-grow"><span className="mono">{i.code}</span><div className="muted sl-meta-text">{i.product}</div></div><Money value={invoiceAmounts(i).total} /><InvoiceStatusPill inv={i} /></div>)}</div>
          : <p className="muted">هنوز فاکتوری برای این پرونده صادر نشده است.</p>}
      </details>

      <div className="sl-dr-foot">
        <button type="button" className="btn btn-lg btn-primary" disabled={!outcome} onClick={commit}>{outcome ? ctaText[outcome] : 'یک نتیجه انتخاب کنید'}</button>
        <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
        <span className="ds-grow" />
        <span className="sl-hint">پیش‌نمایش: هیچ نتیجه‌ای ثبت نمی‌شود</span>
      </div>
    </div>
  )
}

function PayTypeSeg() {
  const [v, setV] = useState<'full' | 'staged'>('full')
  return (
    <div className="lbl">نوع پرداخت
      <div className="sl-seg" role="group" aria-label="نوع پرداخت">
        <button type="button" aria-pressed={v === 'full'} onClick={() => setV('full')}>پرداخت کامل</button>
        <button type="button" aria-pressed={v === 'staged'} onClick={() => setV('staged')}>مرحله‌ای</button>
      </div>
    </div>
  )
}
