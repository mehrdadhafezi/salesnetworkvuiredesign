import { useRef, useState, type FormEvent } from 'react'
import { Banner, DsIcon, PageHead, TonePill, toFaDigits } from '../../../shared/ui/ds'
import { Timeline } from '../components/bits'
import { EXTRA_REQUESTS } from '../fixtures/sellerFixtures'
import { validateExtraRequest, type FieldErrors } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { ExtraRequest } from '../types'

/*
 * S6 — درخواست اضافه کردن شماره (prototype V.request). OPD-10 is unresolved: the form validates
 * locally and then shows a demo notice. No request, assignment, notification or record is created,
 * and the recent-requests list is a static synthetic fixture (it never grows).
 */

const REQ_STATE: Record<ExtraRequest['status'], { label: string; tone: 'orange' | 'green' | 'red'; icon: 'clock' | 'checkCircle' | 'xCircle' }> = {
  pending: { label: 'در انتظار تایید مدیر', tone: 'orange', icon: 'clock' },
  approved: { label: 'تایید شد', tone: 'green', icon: 'checkCircle' },
  rejected: { label: 'رد شد', tone: 'red', icon: 'xCircle' },
}

export function RequestPage() {
  const { toast } = useSellerPreview()
  const [v, setV] = useState({ phone: '', name: '', note: '' })
  const [errors, setErrors] = useState<FieldErrors>({})
  const [openReq, setOpenReq] = useState<string | null>(null)
  const form = useRef<HTMLFormElement>(null)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const errs = validateExtraRequest(v)
    setErrors(errs)
    const first = Object.keys(errs)[0]
    if (first) { form.current?.querySelector<HTMLElement>(`[name="${first}"]`)?.focus(); return }
    toast('نمایشی: درخواست ثبت نشد — تصمیم OPD-10 هنوز باز است و پیش‌نمایش درخواستی نمی‌سازد.', 'warning')
  }
  const field = (k: 'phone' | 'name' | 'note') => ({
    name: k, value: v[k],
    onChange: (e: { target: { value: string } }) => { setV(s => ({ ...s, [k]: e.target.value })); setErrors(x => ({ ...x, [k]: undefined })) },
    ...(errors[k] ? { 'aria-invalid': true as const, 'aria-describedby': `sl-rq-${k}` } : {}),
  })
  const err = (k: 'phone' | 'note') => errors[k] ? <div className="field-err" id={`sl-rq-${k}`}><DsIcon name="alert" /><span>{errors[k]}</span></div> : null

  return (
    <>
      <PageHead title="درخواست اضافه کردن شماره" sub="برای شماره‌ای خارج از لیست تخصیص‌یافته؛ پس از تایید مدیر فروش به «شماره‌های من» اضافه می‌شود" />
      <Banner kind="info">پیش‌نمایش: گردش‌کار این درخواست (OPD-10) هنوز تصویب نشده است. فرم فقط اعتبارسنجی محلی دارد و چیزی ثبت یا ارسال نمی‌کند.</Banner>
      <div className="sl-form-card">
        <section className="ds-panel sl-pad" aria-labelledby="sl-rq-h">
          <div className="ds-sec-h"><h2 id="sl-rq-h" className="sl-h3">درخواست جدید</h2></div>
          <form ref={form} className="sl-form" noValidate onSubmit={submit}>
            <label className="lbl">شماره موبایل مشتری <span className="sl-req" aria-hidden="true">*</span><span className="sr">(الزامی)</span>
              <input className="input sl-ltr" inputMode="tel" placeholder="09xxxxxxxxx" dir="ltr" autoComplete="off" {...field('phone')} />{err('phone')}
              <span className="help">۱۱ رقم، با ۰۹ شروع می‌شود. شماره فقط برای یافتن است، نه شناسه پرونده.</span>
            </label>
            <label className="lbl">نام مشتری <span className="muted">(اختیاری)</span>
              <input className="input" autoComplete="off" {...field('name')} />
            </label>
            <label className="lbl">یادداشت <span className="muted">(اختیاری)</span>
              <textarea className="input sl-textarea" placeholder="علت درخواست یا توضیح کوتاه" {...field('note')} />{err('note')}
              <span className="help">{toFaDigits(v.note.length)} از ۳۰۰ نویسه</span>
            </label>
            <div><button type="submit" className="btn btn-primary">ثبت درخواست (نمایشی)</button></div>
          </form>
        </section>
        <section className="ds-panel sl-pad" aria-labelledby="sl-rq-recent">
          <div className="ds-sec-h"><h2 id="sl-rq-recent" className="sl-h3">درخواست‌های اخیر من</h2><span className="ds-sec-aside">{toFaDigits(EXTRA_REQUESTS.length)} مورد (ساختگی)</span></div>
          <ul className="sl-mini-list" aria-label="درخواست‌های اخیر">
            {EXTRA_REQUESTS.map(r => {
              const st = REQ_STATE[r.status]
              const expanded = openReq === r.id
              return (
                <li key={r.id} className="sl-mini sl-mini-col">
                  <div className="sl-mini-row">
                    <div className="sl-grow"><span className="mono">{toFaDigits(r.phone)}</span><div className="muted sl-meta-text">{r.name} · {r.when}</div></div>
                    <TonePill tone={st.tone} icon={st.icon} label={st.label} />
                    <button type="button" className="btn btn-ghost btn-sm" aria-expanded={expanded} aria-controls={`sl-rq-tl-${r.id}`} onClick={() => setOpenReq(expanded ? null : r.id)}>
                      روند<DsIcon name="chev" className={expanded ? 'sl-rot180' : undefined} />
                    </button>
                  </div>
                  <div id={`sl-rq-tl-${r.id}`} hidden={!expanded}><Timeline events={r.timeline} label={`روند درخواست ${toFaDigits(r.phone)}`} /></div>
                </li>
              )
            })}
          </ul>
        </section>
      </div>
    </>
  )
}
