import { useCallback, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router'
import {
  ContextHint, Drawer, DsIcon, Metric, MetricGroup, PageHead, Panel, QueueTabs, StatusIndicator, TonePill, Toolbar, toFaDigits, type DsQueue,
} from '../../../shared/ui/ds'
import { FactChip, Money, Note, OwnershipFacts, SearchField, TableFoot, Timeline, Who, useCustomerName, useListState } from '../components/bits'
import { NotFoundDrawer, Restricted, ScreenFrame, useDetail, useStatePresentation } from '../components/states'
import { OutcomeGrid } from '../components/overlays'
import { CONVERSIONS, PREVIEW_USER } from '../fixtures/sellerFixtures'
import { CASE_OUTCOMES, DEMO_SUFFIX, type CaseOutcome } from '../labels'
import { useDrawerParam, usePageKeys } from '../state/hooks'
import { filterConversions, paginate, validateOutcome, type FieldErrors, type OutcomeFields } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { Conversion, ConversionQueue } from '../types'

/*
 * S2 — آماده‌های تبدیل (prototype V.conversions + caseDrawer). SYNTHETIC cases only.
 * Attribution context uses the fixed SN-207 ownership vocabulary; the credit owner is shown as
 * UNKNOWN because no approved rule exists in the preview. No conversion, invoice issuance,
 * reassignment or attribution change happens.
 */

const PER_PAGE = 10

export function ConversionsPage() {
  const sp = useStatePresentation()
  const conversions = sp.data(CONVERSIONS)
  const [queue, setQueue] = useState<ConversionQueue>('all')
  const [q, setQ] = useState('')
  const [page, setPage] = useState(1)
  const [caseParam, setCaseParam] = useDrawerParam('case')
  const list = useMemo(() => filterConversions(conversions, queue, q, !sp.policy.fieldsRestricted), [conversions, queue, q, sp.policy.fieldsRestricted])
  const pg = paginate(list, page, PER_PAGE)
  const detail = useDetail(caseParam, id => conversions.find(c => String(c.id) === id))
  const open = detail.record
  const n = (s: ConversionQueue) => conversions.filter(c => s === 'all' || c.stage === s).length
  const remainingKnown = conversions.filter(c => c.remaining !== null)
  const stateEl = useListState({
    count: pg.items.length, noResult: q !== '', onClear: () => setQ(''),
    emptyTitle: 'پرونده‌ای برای پیگیری وجود ندارد', emptyDetail: 'پرونده‌هایی که «تبدیل توسط خودم» را برایشان انتخاب کنید اینجا می‌آیند.',
  })
  const queues: DsQueue[] = [
    { id: 'all', label: 'همه', tone: 'neutral', count: sp.count(n('all')) },
    { id: 'follow', label: 'نیازمند پیگیری', tone: 'amber', icon: 'clock', count: sp.count(n('follow')) },
    { id: 'link', label: 'لینک ارسال شد', tone: 'teal', icon: 'send', count: sp.count(n('link')) },
    { id: 'done', label: 'تکمیل‌شده', tone: 'green', icon: 'checkCircle', count: sp.count(n('done')) },
  ]

  return (
    <ScreenFrame title="آماده‌های تبدیل" contract="no_list_source" head={
      <PageHead
        title="آماده‌های تبدیل"
        sub="پرونده‌هایی که هنگام صدور فاکتور «تبدیل توسط خودم» را انتخاب کرده‌اید (نمونه ساختگی)"
        kpis={
          <MetricGroup label="شاخص‌های این صفحه">
            <Metric label="کل پرونده" {...sp.kpi(conversions.length)} accent="neutral" />
            <Metric label="نیازمند پیگیری" {...sp.kpi(n('follow'))} accent="amber" tone="warn" />
            <Metric label="تماس سررسیده" accent="red" state="undefined" meaning="در نمونه تعریف نشده" />
            <Metric label="دارای مانده" {...sp.kpi(remainingKnown.filter(c => (c.remaining ?? 0) > 0).length)} accent="teal"
              meaning={remainingKnown.length < conversions.length ? `${toFaDigits(conversions.length - remainingKnown.length)} پرونده با مانده نامعلوم` : undefined} />
          </MetricGroup>
        }
      />}>
      <Panel label="فهرست پرونده‌های تبدیل">
        <QueueTabs queues={queues} value={queue} onChange={id => { setQueue(id as ConversionQueue); setPage(1); setCaseParam(null) }} />
        <Toolbar>
          <div className="sl-grow-field"><SearchField id="sl-conv-q" label="جستجوی پرونده" placeholder="نام، موبایل، فاکتور یا شماره پرونده" value={q} onChange={v => { setQ(v); setPage(1) }} /></div>
          <span className="ds-grow" />
          <span className="muted sl-desk" role="status">{sp.countText(list.length)} پرونده</span>
        </Toolbar>
        {stateEl ?? (
          <>
            <div className="sl-tbl-wrap">
              <table className="sl-tbl sl-stackable" aria-label="فهرست پرونده‌های تبدیل">
                <thead><tr><th scope="col">پرونده</th><th scope="col">مشتری</th><th scope="col">گزینه</th><th scope="col">مانده</th><th scope="col">مرحله</th><th scope="col">نتیجه تماس</th><th scope="col" className="sl-col-actions"><span className="sr">اقدام</span></th></tr></thead>
                <tbody>
                  {pg.items.map(c => (
                    <tr key={c.id} data-row={c.id} tabIndex={0} className={open?.id === c.id ? 'sl-sel' : undefined} aria-selected={open?.id === c.id}
                      onClick={() => setCaseParam(String(c.id))} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); setCaseParam(String(c.id)) } }}>
                      <td data-l="پرونده"><span className="mono sl-code">#{toFaDigits(c.id)}</span><div className="muted sl-micro">فاکتور <span className="mono">{c.invoice}</span></div></td>
                      <td data-l="مشتری"><Who name={c.customer} phone={c.phone} /></td>
                      <td data-l="گزینه">{c.option}</td>
                      <td data-l="مانده">{c.remaining === null ? <StatusIndicator state="unknown" /> : c.remaining ? <Money value={c.remaining} /> : <span className="muted">—</span>}</td>
                      <td data-l="مرحله" className="muted">{c.stageLabel}</td>
                      <td data-l="نتیجه تماس"><TonePill tone={c.resultTone} icon={c.resultIcon} label={c.result} /></td>
                      <td className="sl-col-actions"><div className="sl-row-actions"><button type="button" className="btn btn-soft btn-sm" onClick={e => { e.stopPropagation(); setCaseParam(String(c.id)) }}>ثبت نتیجه تماس</button></div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <TableFoot page={pg} perPage={PER_PAGE} onPage={setPage} />
          </>
        )}
      </Panel>
      <CaseDrawer c={open} onClose={() => setCaseParam(null)} />
      <NotFoundDrawer open={detail.notFound} kicker="پرونده تبدیل" onClose={() => setCaseParam(null)} />
    </ScreenFrame>
  )
}

function CaseDrawer({ c, onClose }: { c: Conversion | null; onClose: () => void }) {
  const name = useCustomerName()
  return (
    <Drawer open={c !== null} onClose={onClose} title={c ? name(c.customer) : 'پرونده تبدیل'} kicker={c ? `پرونده تبدیل #${toFaDigits(c.id)}` : 'پرونده تبدیل'}>
      {c ? <CaseBody key={c.id} c={c} onClose={onClose} /> : null}
    </Drawer>
  )
}

const FIELDS: OutcomeFields = { date: '', hour: '10', minute: '00', reason: '', amount: '' }

function CaseBody({ c, onClose }: { c: Conversion; onClose: () => void }) {
  const { toast, ui } = useSellerPreview()
  const [params] = useSearchParams()
  const init = params.get('outcome')
  const [outcome, setOutcome] = useState<CaseOutcome | null>(CASE_OUTCOMES.some(o => o.id === init) ? init as CaseOutcome : null)
  const [fields, setFields] = useState(FIELDS)
  const [errors, setErrors] = useState<FieldErrors>({})
  const ref = useRef<HTMLDivElement>(null)
  const set = (k: keyof OutcomeFields) => (v: string) => { setFields(f => ({ ...f, [k]: v })); setErrors(e => ({ ...e, [k]: undefined })) }
  const commit = () => {
    if (!outcome) return
    const e = validateOutcome(outcome, fields)
    setErrors(e)
    const first = Object.keys(e)[0]
    if (first) { ref.current?.querySelector<HTMLElement>(`[data-field="${first}"]`)?.focus(); return }
    toast(`«${CASE_OUTCOMES.find(o => o.id === outcome)?.label ?? ''}» برای پرونده #${toFaDigits(c.id)} ${DEMO_SUFFIX}`)
  }
  usePageKeys(useCallback((k: string) => {
    if (/^[1-4]$/.test(k)) { const o = CASE_OUTCOMES[Number(k) - 1]; if (o) { setOutcome(o.id); setErrors({}); return true } }
    return false
  }, []))
  const err = (k: keyof FieldErrors) => errors[k] ? <div className="field-err" id={`sl-cerr-${k}`}><DsIcon name="alert" /><span>{errors[k]}</span></div> : null
  const inv = (k: keyof FieldErrors) => errors[k] ? { 'aria-invalid': true as const, 'aria-describedby': `sl-cerr-${k}` } : {}

  return (
    <div ref={ref}>
      <div className="sl-dr-head">
        <div className="sl-dr-title"><TonePill tone={c.resultTone} icon={c.resultIcon} label={c.result} /></div>
        <div className="sl-dr-meta">{ui.policy.fieldsRestricted ? <Restricted what="شماره مشتری" /> : <span className="mono sl-dr-phone">{toFaDigits(c.phone)}</span>}<span>فاکتور <span className="mono">{c.invoice}</span></span><span>{c.option}</span></div>
        <div className="sl-dr-facts"><FactChip icon="history">آخرین فعالیت: {c.lastActivity}</FactChip><FactChip icon="layers">مرحله: {c.stageLabel}</FactChip></div>
      </div>
      <section className="ds-sec sl-sec-primary" aria-labelledby="sl-case-h">
        <div className="sl-sum-grid sl-sum-3">
          <div className="sl-sum"><span>مبلغ گزینه</span><b><Money value={c.remaining} /></b></div>
          <div className="sl-sum"><span>پرداخت تایید‌شده</span><b><Money value={null} /></b></div>
          <div className="sl-sum"><span>مانده</span><b><Money value={c.remaining} /></b></div>
        </div>
        <div className="ds-sec-h sl-mt"><h3 id="sl-case-h">نتیجه تماس</h3></div>
        <OutcomeGrid options={CASE_OUTCOMES} value={outcome} onChange={o => { setOutcome(o); setErrors({}) }} label="نتیجه تماس" cols={3} />
        {outcome === 'customer_declined' ? (
          <div className="sl-cond sl-cond-box"><label className="lbl">علت انصراف <span className="sl-req" aria-hidden="true">*</span><span className="sr">(الزامی)</span>
            <textarea className="input sl-textarea" data-field="reason" placeholder="علت دقیق انصراف مشتری" value={fields.reason} onChange={e => set('reason')(e.target.value)} {...inv('reason')} />{err('reason')}</label></div>
        ) : null}
        {outcome === 'follow_up' ? (
          <div className="sl-cond sl-cond-box"><div className="sl-row2">
            <label className="lbl">تاریخ شمسی <span className="sl-req" aria-hidden="true">*</span><span className="sr">(الزامی)</span>
              <input className="input" data-field="date" placeholder="۱۴۰۵/۰۷/۱۰" value={fields.date} onChange={e => set('date')(e.target.value)} {...inv('date')} />{err('date')}</label>
            <div className="lbl" role="group" aria-label="ساعت تهران (الزامی)">ساعت تهران <span className="sl-req" aria-hidden="true">*</span>
              <span className="sl-time"><input className="input" data-field="hour" aria-label="ساعت" value={fields.hour} onChange={e => set('hour')(e.target.value)} {...inv('hour')} /> : <input className="input" data-field="minute" aria-label="دقیقه" value={fields.minute} onChange={e => set('minute')(e.target.value)} {...inv('minute')} /></span>{err('hour')}{err('minute')}</div>
          </div></div>
        ) : null}
        {outcome === 'payment_link' ? (
          <div className="sl-cond sl-cond-box"><div className="sl-row2">
            <label className="lbl">محصول نهایی<select className="input" defaultValue="o"><option value="o">{c.option}</option></select></label>
            <label className="lbl">مبلغ این مرحله<input className="input" placeholder="مثلاً ۲۰٬۰۰۰٬۰۰۰" value={fields.amount} onChange={e => set('amount')(e.target.value)} /></label>
          </div><Note tone="info" icon="send">در نسخه واقعی لینک پس از تأیید شما ساخته و برای مشتری پیامک می‌شود. پیش‌نمایش لینکی نمی‌سازد.</Note></div>
        ) : null}
      </section>
      <details className="sl-details" open>
        <summary><h3>زمینه انتساب</h3><ContextHint text="واژه‌های ثابت مالکیت (SN-207 §6). هیچ‌کدام در این پیش‌نمایش تغییر نمی‌کند." /><DsIcon name="chev" className="sl-chev" /></summary>
        <OwnershipFacts rows={[
          { term: 'مسئول فعلی', value: `شما (${PREVIEW_USER.name})` },
          { term: 'اقدام بعدی با', value: `شما (${PREVIEW_USER.name})` },
          { term: 'عامل رویداد صدور', value: `${PREVIEW_USER.name} · صادرکننده فاکتور ${c.invoice}` },
          { term: 'مالک اعتبار', value: <StatusIndicator state="unknown" detail="طبق قوانین مالی؛ در پیش‌نمایش تعیین نمی‌شود" /> },
        ]} />
      </details>
      <details className="sl-details">
        <summary><h3>یادداشت و تاریخچه</h3><DsIcon name="chev" className="sl-chev" /></summary>
        <Timeline events={c.history} label="تاریخچه پرونده" />
      </details>
      <div className="sl-dr-foot">
        <button type="button" className="btn btn-lg btn-primary" disabled={!outcome} onClick={commit}>{outcome === 'payment_link' ? 'ثبت محصول و ادامه به ارسال لینک' : outcome ? 'ذخیره نتیجه تماس' : 'یک نتیجه انتخاب کنید'}</button>
        <button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>
      </div>
    </div>
  )
}
