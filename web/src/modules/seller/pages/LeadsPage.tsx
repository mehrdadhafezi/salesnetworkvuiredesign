import { useCallback, useEffect, useMemo, useReducer, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import {
  Banner, ContextHint, Drawer, DsIcon, Metric, MetricGroup, PageHead, Panel, QueueTabs, Toolbar, toFaDigits, type DsQueue,
} from '../../../shared/ui/ds'
import {
  FactChip, FilterChip, InvoiceStatusPill, LeadStatusPill, Money, NextStep, Note, SearchField, TableFoot, Tag, Timeline, Who,
  nextChipClass, useCustomerName, useListState,
} from '../components/bits'
import { NotFoundDrawer, Restricted, ScreenFrame, useDetail, useStatePresentation } from '../components/states'
import { OutcomeGrid } from '../components/overlays'
import { INVOICES, LEADS } from '../fixtures/sellerFixtures'
import { CLOSED_LEAD_STATUSES, DEMO_SUFFIX, LEAD_OUTCOMES, LEAD_QUEUE_ORDER, LEAD_STATUS } from '../labels'
import { useDrawerParam, usePageKeys } from '../state/hooks'
import {
  DEFAULT_LEAD_FILTER, filterLeads, followUpRate, invoiceAmounts, leadCounts, paginate, unitOf, validateOutcome,
  type DateRange, type FieldErrors, type LeadFilter, type OutcomeFields,
} from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import { DRAFT_PRODUCTS, PAY_TYPE_LABEL, SIMULATED_RESULT_NOTICE, SIMULATION_NOTICE, draftLines, draftReducer, emptyDraft, type DraftProduct, type DraftStep } from '../state/outcomeDraft'
import type { Lead, LeadOutcome, LeadQueue } from '../types'

/*
 * S1 — شماره‌های من (prototype V.leads + leadDrawer). SYNTHETIC leads only.
 * Queue / search / filters / sort / pagination / drawer / outcome selection are real local
 * interactions. RM-E04J: «ثبت» validates the local draft (existing `validateOutcome`) and opens a
 * review step; «تأیید شبیه‌سازی» shows a clearly labelled simulated result. Nothing is ever recorded:
 * no request, no storage, and `cases.contact_outcome.record` is not wired (Seller authority: D-42).
 */

const PER_PAGE = 10
const SOURCES = [{ id: 'MIS', label: 'MIS' }, { id: 'درخواستی', label: 'درخواستی' }, { id: 'ارتباط مجدد', label: 'ارتباط مجدد' }]
const PROBS = [{ id: 'ضعیف', label: 'ضعیف' }, { id: 'متوسط', label: 'متوسط' }, { id: 'بالا', label: 'بالا' }, { id: '۱۰۰٪', label: '۱۰۰٪' }, { id: '—', label: 'تعیین‌نشده' }]
const RANGES: { id: Exclude<DateRange, 'all'>; label: string }[] = [{ id: '7', label: '۷ روز اخیر' }, { id: '30', label: '۳۰ روز اخیر' }]

function isQueue(v: string | null): v is LeadQueue {
  return v !== null && (LEAD_QUEUE_ORDER as readonly string[]).includes(v)
}

export function LeadsPage() {
  const { toast, base } = useSellerPreview()
  const sp = useStatePresentation()
  const leads = sp.data(LEADS)
  const name = useCustomerName()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [filter, setFilter] = useState<LeadFilter>(() => ({ ...DEFAULT_LEAD_FILTER, queue: isQueue(params.get('queue')) ? params.get('queue') as LeadQueue : 'none' }))
  const [page, setPage] = useState(1)
  const [cursor, setCursor] = useState(0)
  const [leadParam, setLeadParam] = useDrawerParam('lead')
  const tbody = useRef<HTMLTableSectionElement>(null)
  const search = useRef<HTMLDivElement>(null)

  const counts = useMemo(() => leadCounts(leads), [leads])
  const identity = !sp.policy.fieldsRestricted
  const list = useMemo(() => filterLeads(leads, filter, identity), [leads, filter, identity])
  const pg = paginate(list, page, PER_PAGE)
  const rate = followUpRate(leads)
  const detail = useDetail(leadParam, id => leads.find(l => String(l.id) === id))
  const openLead = detail.record

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
    const src = list.some(l => l.id === openLead.id) ? list : leads
    const i = src.findIndex(l => l.id === openLead.id)
    const n = src[(i + d + src.length) % src.length]
    if (n) setLeadParam(String(n.id))
  }, [openLead, list, leads, cursor, setLeadParam])

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
    ? { id: 'all', label: 'همه', tone: 'neutral', count: sp.count(counts.all), shortcut: '0' }
    : { id: k, label: LEAD_STATUS[k].label, tone: LEAD_STATUS[k].tone, icon: LEAD_STATUS[k].icon, count: sp.count(counts[k]), shortcut: LEAD_STATUS[k].key })

  const onRowKey = (e: ReactKeyboardEvent<HTMLTableRowElement>, i: number, lead: Lead) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); focusRow(i + 1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); focusRow(i - 1) }
    else if (e.key === 'Enter') { e.preventDefault(); setLeadParam(String(lead.id)) }
  }

  const stateEl = useListState({
    count: pg.items.length, noResult: filtersActive, onClear: clearFilters,
    emptyTitle: 'در این صف موردی وجود ندارد', emptyDetail: 'شماره‌ای با این وضعیت ندارید.',
  })

  return (
    <ScreenFrame title="شماره‌های من" head={
      <PageHead
        title="شماره‌های من"
        sub={`${sp.countText(leads.length)} شماره تخصیص‌یافته (ساختگی)`}
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="بدون وضعیت" {...sp.kpi(counts.none)} accent="neutral" />
            <Metric label="تماس مجدد امروز" {...sp.kpi(leads.filter(l => l.next?.kind === 'callback' && l.next.due).length)} accent="blue" />
            <Metric label="نرخ پیگیری" value={rate === null ? undefined : toFaDigits(rate)} unit="٪" accent="teal" state={rate === null ? 'unknown' : 'ok'} />
            <Metric label="پیش‌فاکتور (نمونه)" {...sp.kpi(counts.invoiced)} accent="violet" />
          </MetricGroup>
        }
        actions={<button type="button" className="btn" onClick={() => void navigate(`${base}/request`)}><DsIcon name="plus" />درخواست شماره</button>}
      />}>
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
          <span className="muted sl-desk" role="status">{sp.countText(list.length)} نتیجه</span>
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
                            <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label={`تماس با ${name(l.name)}`} title="تماس"
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
                  <article key={l.id} className="sl-lcard" aria-label={name(l.name)}>
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
      <LeadDrawer lead={openLead} list={list} all={leads} onClose={() => setLeadParam(null)} onStep={step} />
      <NotFoundDrawer open={detail.notFound} kicker="جزئیات شماره" onClose={() => setLeadParam(null)} />
    </ScreenFrame>
  )
}


function LeadDrawer({ lead, list, all, onClose, onStep }: { lead: Lead | null; list: readonly Lead[]; all: readonly Lead[]; onClose: () => void; onStep: (d: number) => void }) {
  const name = useCustomerName()
  const src = lead && list.some(l => l.id === lead.id) ? list : all
  const idx = lead ? src.findIndex(l => l.id === lead.id) : -1
  return (
    <Drawer open={lead !== null} onClose={onClose} title={lead ? name(lead.name) : 'جزئیات'} kicker={lead ? `${toFaDigits(idx + 1)} از ${toFaDigits(src.length)}` : 'جزئیات'}
      topExtra={lead ? (
        <>
          <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label="مشتری قبلی (K)" title="قبلی (K)" onClick={() => onStep(-1)}><DsIcon name="chev" className="sl-rot180" /></button>
          <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label="مشتری بعدی (J)" title="بعدی (J)" onClick={() => onStep(1)}><DsIcon name="chev" /></button>
          <span className="sl-desk sl-micro">۱–۵ نتیجه · Enter ثبت · Esc بستن</span>
        </>
      ) : undefined}
    >
      {/* Keyed by lead: the draft (and any simulated result) belongs to one lead and never carries over. */}
      {lead ? <LeadDrawerBody key={lead.id} lead={lead} onClose={onClose} onStep={onStep} /> : null}
    </Drawer>
  )
}

function LeadDrawerBody({ lead, onClose, onStep }: { lead: Lead; onClose: () => void; onStep: (d: number) => void }) {
  const { toast, ui } = useSellerPreview()
  const restricted = ui.policy.fieldsRestricted
  const [params] = useSearchParams()
  const initial = params.get('outcome')
  const [d, dispatch] = useReducer(draftReducer, LEAD_OUTCOMES.some(o => o.id === initial) ? initial as LeadOutcome : null, emptyDraft)
  const { outcome, fields, errors, step } = d
  const [note, setNote] = useState(lead.note)
  const body = useRef<HTMLDivElement>(null)
  const stepHead = useRef<HTMLHeadingElement>(null)
  const prevStep = useRef(step)
  const attemptsNext = Math.min(3, lead.attempts + 1)
  const invoices = INVOICES.filter(i => lead.invoiceRefs?.includes(i.code))
  const s = LEAD_STATUS[lead.status]
  const chosen = LEAD_OUTCOMES.find(o => o.id === outcome)
  const lines = draftLines(d, { attemptsNext, fa: toFaDigits })

  const ctaText: Record<LeadOutcome, string> = {
    no_answer: `ثبت تماس بی‌پاسخ (${toFaDigits(attemptsNext)} از ۳)`,
    callback: `ثبت تماس مجدد — ${fields.date || '…'} ساعت ${toFaDigits(fields.hour || '…')}:${toFaDigits(fields.minute || '…')}`,
    not_purchased: 'ثبت عدم خرید',
    duplicate: 'ثبت به‌عنوان تکراری',
    success: 'صدور پیش‌فاکتور',
  }

  const pick = (o: LeadOutcome) => dispatch({ type: 'pick', outcome: o })
  const set = (k: keyof OutcomeFields) => (v: string) => dispatch({ type: 'field', key: k, value: v })
  /** Existing CTA: validates with `validateOutcome`; a valid draft opens the review step (nothing is recorded). */
  const commit = () => {
    if (!outcome || step !== 'edit') return
    const e = validateOutcome(outcome, fields)
    dispatch({ type: 'review' })
    const first = Object.keys(e)[0]
    if (first) body.current?.querySelector<HTMLElement>(`[data-field="${first}"]`)?.focus()
  }
  const confirm = () => {
    if (step !== 'review') return
    dispatch({ type: 'confirm' })
    // No customer identity in the toast: toasts outlive the drawer.
    toast(`نتیجه «${chosen?.label ?? ''}» فقط شبیه‌سازی شد ${DEMO_SUFFIX}`, 'info')
  }

  // Focus management between steps: review / result → their heading; back to edit → the chosen outcome.
  useEffect(() => {
    const was = prevStep.current
    prevStep.current = step
    if (was === step) return
    if (step !== 'edit') { stepHead.current?.focus(); return }
    const target = body.current?.querySelector<HTMLElement>(outcome ? `[data-outcome="${outcome}"]` : '[data-outcome]')
    target?.focus()
  }, [step, outcome])

  // Drawer keys (prototype): 1–5 pick an outcome, Enter commits, J/K step — only while focus is not on a control.
  // In review / result steps the outcome cannot change and Enter does nothing: confirming needs the explicit button.
  usePageKeys(useCallback((k: string) => {
    if (/^[1-5]$/.test(k)) { if (step !== 'edit') return true; const o = LEAD_OUTCOMES[Number(k) - 1]; if (o) { pick(o.id); return true } }
    if (k === 'enter' && outcome) { if (step === 'edit') commit(); return true }
    if (k === 'j') { onStep(1); return true }
    if (k === 'k') { onStep(-1); return true }
    return false
  }, [outcome, fields, step, onStep])) // eslint-disable-line react-hooks/exhaustive-deps

  const err = (k: keyof FieldErrors) => errors[k] ? <div className="field-err" id={`sl-err-${k}`} role="alert"><DsIcon name="alert" /><span>{errors[k]}</span></div> : null
  const inv = (k: keyof FieldErrors) => errors[k] ? { 'aria-invalid': true as const, 'aria-describedby': `sl-err-${k}` } : {}

  const summary = (
    <dl className="sl-kv sl-sim-kv">
      <div><dt>پرونده (ساختگی)</dt><dd className="mono">{toFaDigits(lead.id)}</dd></div>
      <div><dt>مشتری</dt><dd>{restricted ? <Restricted what="نام مشتری" /> : lead.name}</dd></div>
      <div><dt>نتیجه انتخاب‌شده</dt><dd>{chosen ? <span className={`pill t-${chosen.tone}`}><DsIcon name={chosen.icon} />{chosen.label}</span> : '—'}</dd></div>
      <div><dt>وضعیت پرونده در CRM</dt><dd>{s.label} <span className="muted sl-meta-text">(بدون تغییر)</span></dd></div>
      {lines.map(l => <div key={l.label}><dt>{l.label}</dt><dd>{l.value}</dd></div>)}
    </dl>
  )

  return (
    <div ref={body} data-draft-step={step}>
      <div className="sl-dr-head">
        <div className="sl-dr-title"><LeadStatusPill lead={lead} /></div>
        <div className="sl-dr-meta">{restricted ? <Restricted what="شماره مشتری" /> : <span className="mono sl-dr-phone">{toFaDigits(lead.phone)}</span>}<span>{lead.city}، {lead.province}</span><Tag teal={lead.source !== 'MIS'}>{lead.source}</Tag></div>
        <div className="sl-dr-facts">
          <FactChip icon="history">آخرین فعالیت: {lead.lastActivity}</FactChip>
          {lead.next ? <FactChip icon="clock" cls={nextChipClass(lead.next)}>گام بعدی: {lead.next.text}</FactChip> : <FactChip icon="arrowL">گام بعدی: ثبت اولین نتیجه</FactChip>}
          <FactChip icon="clock">تخصیص: {lead.assigned}</FactChip>
        </div>
        <div className="sl-dr-quick">
          <button type="button" className="btn" onClick={() => toast('نمایشی: شماره‌گیر باز نمی‌شود؛ پیش‌نمایش تماسی برقرار نمی‌کند.')}><DsIcon name="phone" />تماس</button>
          {restricted ? null : <button type="button" className="btn" onClick={() => toast(`نمایشی: شماره ${toFaDigits(lead.phone)} کپی نشد.`)}><DsIcon name="copy" />کپی شماره</button>}
        </div>
      </div>

      <DraftSteps step={step} />

      {step === 'edit' ? (
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
                  <select className="input" value={d.product} onChange={e => dispatch({ type: 'product', value: e.target.value as DraftProduct })}>
                    {DRAFT_PRODUCTS.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
                  </select>
                </label>
                <div className="lbl">نوع پرداخت
                  <div className="sl-seg" role="group" aria-label="نوع پرداخت">
                    {(['full', 'staged'] as const).map(v => (
                      <button key={v} type="button" aria-pressed={d.payType === v} onClick={() => dispatch({ type: 'payType', value: v })}>{PAY_TYPE_LABEL[v]}</button>
                    ))}
                  </div>
                </div>
              </div>
              <Note tone="info" icon="send">در نسخه واقعی پس از صدور، پیش‌فاکتور مطابق تنظیمات فعلی برای مشتری پیامک می‌شود. پیش‌نمایش چیزی صادر یا ارسال نمی‌کند.</Note>
            </div>
          ) : null}
        </section>
      ) : null}

      {step === 'review' ? (
        <section className="ds-sec sl-sec-primary sl-sim" aria-labelledby="sl-review-h" data-simulation="review">
          <div className="ds-sec-h"><h3 id="sl-review-h" ref={stepHead} tabIndex={-1}>بازبینی پیش از تأیید</h3><span className="ds-sec-aside">پیش‌نویس محلی</span></div>
          <Banner kind="info">{SIMULATION_NOTICE}</Banner>
          {summary}
          <p className="muted sl-meta-text">برای تغییر، «ویرایش» را بزنید؛ انتخاب‌ها و مقادیر واردشده حفظ می‌شوند.</p>
        </section>
      ) : null}

      {step === 'simulated' ? (
        <section className="ds-sec sl-sec-primary sl-sim" aria-labelledby="sl-sim-h" data-simulation="result">
          <div className="ds-sec-h"><h3 id="sl-sim-h" ref={stepHead} tabIndex={-1}>نتیجه شبیه‌سازی‌شده</h3><span className="pill t-slate sl-sim-tag"><DsIcon name="info" />شبیه‌سازی</span></div>
          <Banner kind="info">{SIMULATED_RESULT_NOTICE}</Banner>
          {summary}
          <div className="sl-sim-event" data-simulated="true">
            <div className="sl-meta-text muted">رویداد شبیه‌سازی‌شده — جزو تاریخچه CRM نیست</div>
            <Timeline label="رویداد شبیه‌سازی‌شده (ثبت نشده)" events={[{ text: `${chosen?.label ?? ''} (شبیه‌سازی)`, meta: 'بدون زمان سرور · بدون شناسه رویداد · ثبت نشده' }]} />
          </div>
        </section>
      ) : null}

      <details className="sl-details" open>
        <summary><h3>اطلاعات مشتری</h3><span className="muted sl-meta-text">فقط‌خواندنی در پیش‌نمایش</span><DsIcon name="chev" className="sl-chev" /></summary>
        <dl className="sl-kv">
          <div><dt>نام مشتری</dt><dd>{restricted ? <Restricted what="نام مشتری" /> : lead.name}</dd></div>
          <div><dt>احتمال فروش <ContextHint text="برای اولویت‌بندی پیگیری است و وضعیت مشتری را تغییر نمی‌دهد." /></dt><dd>{lead.prob || <span className="muted">—</span>}</dd></div>
          <div><dt>استان</dt><dd>{lead.province}</dd></div>
          <div><dt>شهر</dt><dd>{lead.city}</dd></div>
        </dl>
      </details>
      <details className="sl-details" open>
        <summary><h3>یادداشت</h3><span className="muted sl-meta-text">پیش‌نویس محلی — ذخیره نمی‌شود</span><DsIcon name="chev" className="sl-chev" /></summary>
        <label className="lbl"><span className="sr">یادداشت (پیش‌نویس محلی)</span>
          <textarea className="input sl-textarea" placeholder="خلاصه مکالمه، نیاز مشتری…" value={note} onChange={e => setNote(e.target.value)} />
        </label>
      </details>
      <details className="sl-details">
        <summary><h3>تاریخچه</h3><span className="muted sl-meta-text">{toFaDigits(lead.history.length)} رویداد</span><DsIcon name="chev" className="sl-chev" /></summary>
        <Timeline events={lead.history} label="تاریخچه تعامل" />
        {ui.state === 'partial'
          ? <p className="muted sl-meta-text" data-partial-history="true">تاریخچه کامل نیست (پاسخ ناقص)؛ رویدادهای قدیمی‌تر ممکن است نمایش داده نشوند.</p>
          : null}
      </details>
      <details className="sl-details" open={invoices.length > 0}>
        <summary><h3>فاکتورهای این پرونده</h3><span className="muted sl-meta-text">{toFaDigits(invoices.length)}</span><DsIcon name="chev" className="sl-chev" /></summary>
        {invoices.length
          ? <div className="sl-mini-list">{invoices.map(i => <div key={i.code} className="sl-mini"><div className="sl-grow"><span className="mono">{i.code}</span><div className="muted sl-meta-text">{i.product}</div></div><Money value={invoiceAmounts(i).total} unit={unitOf(i)} /><InvoiceStatusPill inv={i} /></div>)}</div>
          : <p className="muted">هنوز فاکتوری برای این پرونده صادر نشده است.</p>}
      </details>

      <div className="sl-dr-foot">
        {step === 'edit' ? (
          <>
            <button type="button" className="btn btn-lg btn-primary" disabled={!outcome} onClick={commit}>{outcome ? ctaText[outcome] : 'یک نتیجه انتخاب کنید'}</button>
            <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
            <span className="ds-grow" />
            <span className="sl-hint">پیش‌نمایش: پیش از تأیید بازبینی می‌شود؛ هیچ نتیجه‌ای ثبت نمی‌شود</span>
          </>
        ) : null}
        {step === 'review' ? (
          <>
            <button type="button" className="btn btn-lg btn-primary" onClick={confirm}>تأیید شبیه‌سازی</button>
            <button type="button" className="btn btn-lg" onClick={() => dispatch({ type: 'edit' })}>ویرایش</button>
            <button type="button" className="btn btn-lg btn-ghost" onClick={() => dispatch({ type: 'discard' })}>دورانداختن پیش‌نویس</button>
            <span className="ds-grow" />
            <span className="sl-hint">تأیید فقط شبیه‌سازی است؛ چیزی به CRM ارسال نمی‌شود</span>
          </>
        ) : null}
        {step === 'simulated' ? (
          <>
            {/* Holds the confirm button's place, disabled: a double click on «تأیید» cannot land on «شروع دوباره». */}
            <button type="button" className="btn btn-lg btn-primary" disabled>شبیه‌سازی انجام شد</button>
            <button type="button" className="btn btn-lg" onClick={() => dispatch({ type: 'discard' })}>شروع دوباره</button>
            <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
            <span className="ds-grow" />
            <span className="sl-hint">شبیه‌سازی — چیزی ثبت نشد</span>
          </>
        ) : null}
      </div>
    </div>
  )
}

const STEP_LABEL: Record<DraftStep, string> = { edit: 'انتخاب و تکمیل', review: 'بازبینی', simulated: 'نتیجه شبیه‌سازی' }

/** Step indicator (presentation only): where the seller is in the local preview flow. */
function DraftSteps({ step }: { step: DraftStep }) {
  const order: DraftStep[] = ['edit', 'review', 'simulated']
  const at = order.indexOf(step)
  return (
    <ol className="sl-steps" aria-label="مراحل پیش‌نمایش ثبت نتیجه">
      {order.map((k, i) => (
        <li key={k} className={i < at ? 'sl-done' : i === at ? 'sl-now' : undefined} aria-current={i === at ? 'step' : undefined}>
          <span className="sl-step-n" aria-hidden="true">{toFaDigits(i + 1)}</span>{STEP_LABEL[k]}
        </li>
      ))}
    </ol>
  )
}
