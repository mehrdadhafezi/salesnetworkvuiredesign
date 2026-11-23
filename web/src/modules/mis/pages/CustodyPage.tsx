import '../mis.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import type { Parity } from '../../../shared/api'
import { Button, Card, PageHeader } from '../../../shared/ui/primitives'
import { findingLabel, parityLabel } from '../../sales/labels'
import { derivationLabel } from '../labels'
import { CustodyList } from '../workspace/lists'
import { MISSectionNav } from '../workspace/SectionNav'

const FILTERS = ['parity', 'derivation', 'presence', 'finding_kind', 'holder_link'] as const

/**
 * RM-P05 custody and assignments: the derived custody projection of every
 * Case next to the legacy holder of record, with every server filter of
 * GET /api/v1/custody/assignments. Current custody is shown apart from
 * historical attribution (opened per case); nothing assigns or transfers.
 */
export function CustodyPage() {
  const [params, setParams] = useSearchParams()
  const query = Object.fromEntries(FILTERS.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const next = new URLSearchParams()
    for (const k of FILTERS) {
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') next.set(k, v)
    }
    setParams(next)
  }

  return (
    <>
      <PageHeader title="نگهداری و انتساب" subtitle="نگهدار فعلی استنتاج‌شده از رویدادها در کنار نگهدار ثبت‌شده در منبع؛ انتساب تاریخی جدا و تغییرناپذیر است." />
      <MISSectionNav />
      <div className="mis-concepts" aria-label="مفاهیم جدا">
        <section className="mis-concept"><h2>نگهدار فعلی</h2><p className="muted">از رویدادهای افزایشی منبع استنتاج می‌شود و با نگهدار ثبت‌شده در منبع مقایسه می‌شود (تطابق).</p></section>
        <section className="mis-concept"><h2>انتساب تاریخی</h2><p className="muted">مالک و فروشنده اولیه و صاحب اعتبار، هنگام واردسازی منجمد شده‌اند و با چارت سازمانی امروز بازنویسی نمی‌شوند.</p></section>
        <section className="mis-concept"><h2>اقدام‌کننده بعدی</h2><p className="muted">در این برش مدل نشده است؛ نگهدار فعلی به معنی اقدام‌کننده بعدی نیست.</p></section>
      </div>
      <Card title="فیلتر">
        <form className="mis-filters" onSubmit={apply} aria-label="فیلتر نگهداری">
          <label className="lbl">تطابق نگهدار
            <select className="input" name="parity" defaultValue={query.parity ?? ''}>
              <option value="">همه</option>
              {(Object.keys(parityLabel) as Parity[]).map(p => <option key={p} value={p}>{parityLabel[p]}</option>)}
            </select>
          </label>
          <label className="lbl">وضعیت استنتاج
            <select className="input" name="derivation" defaultValue={query.derivation ?? ''}>
              <option value="">همه</option>
              {Object.entries(derivationLabel).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
            </select>
          </label>
          <label className="lbl">یافته نگهداری
            <select className="input" name="finding_kind" defaultValue={query.finding_kind ?? ''}>
              <option value="">همه</option>
              {Object.entries(findingLabel).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
            </select>
          </label>
          <label className="lbl">هویت نگهدار
            <select className="input" name="holder_link" defaultValue={query.holder_link ?? ''}>
              <option value="">همه</option>
              <option value="UNRESOLVED">حل‌نشده (کلید قدیمی بدون حساب هدف)</option>
            </select>
          </label>
          <label className="lbl">حضور
            <select className="input" name="presence" defaultValue={query.presence ?? ''}>
              <option value="">همه</option>
              <option value="PRESENT">حاضر</option>
              <option value="ABSENT_IN_LATEST">غایب در آخرین واردسازی</option>
            </select>
          </label>
          <div className="mis-actions">
            <Button type="submit" variant="primary">اعمال فیلتر</Button>
            {Object.keys(query).length ? <Link to="/mis/custody">پاک کردن فیلترها</Link> : null}
          </div>
        </form>
      </Card>
      <Card title="پروجکشن‌های نگهداری">
        <CustodyList key={JSON.stringify(query)} query={query} empty={Object.keys(query).length ? 'نتیجه‌ای برای این فیلتر نیست' : 'شاهد نگهداری‌ای وارد نشده است'} />
      </Card>
    </>
  )
}
