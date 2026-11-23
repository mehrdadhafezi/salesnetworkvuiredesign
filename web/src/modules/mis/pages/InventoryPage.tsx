import '../mis.css'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { useApi, type CaseSummary, type WorkflowType } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock } from '../../../shared/ui/primitives'
import { workflowLabel } from '../../sales/labels'
import { Pager, useCursors } from '../../sales/pager'
import { useCustodyFor } from '../components'
import { blockedSales } from '../labels'
import { CaseTable, PAGE } from '../workspace/lists'
import { MISSectionNav } from '../workspace/SectionNav'

/** Server filters of GET /api/v1/cases (the only ones the inventory offers). */
const FILTERS = ['workflow_type', 'state_mapping', 'customer_link_state', 'presence', 'source_key'] as const
type FilterKey = typeof FILTERS[number]

const KEY = /^[1-9][0-9]{0,19}$/
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/

/**
 * RM-P05 case inventory: GET /api/v1/cases with every server filter it
 * documents (no client-side filtering, no phone or name search), keyset
 * pages, and the custody evidence of the visible page read with the
 * case_id filter of GET /api/v1/custody/assignments (only when the server
 * hint allows custody.custody.read). Unknown or conflicting data is shown
 * as such; nothing is resolved here.
 */
export function InventoryPage() {
  const api = useApi()
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const [jumpError, setJumpError] = useState('')
  const query = Object.fromEntries(FILTERS.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : []))) as Partial<Record<FilterKey, string>>
  const badKey = query.source_key !== undefined && !KEY.test(query.source_key)
  const list = useRead(JSON.stringify({ query, cursor: c.cursor, badKey }), async () => badKey
    ? { status: 'ok' as const, data: { items: [] as CaseSummary[], page: { has_more: false } }, outcome: 'SUCCESS' as const, allowed: undefined }
    : settle(await api.GET('/api/v1/cases', {
      params: { query: compact({
        workflow_type: query.workflow_type as WorkflowType | undefined,
        state_mapping: query.state_mapping as 'KNOWN' | 'UNKNOWN' | undefined,
        customer_link_state: query.customer_link_state as 'RESOLVED' | 'RELATED' | 'UNKNOWN' | 'CONFLICT' | 'NOT_RECOVERABLE' | undefined,
        presence: query.presence as 'PRESENT' | 'ABSENT_IN_LATEST' | undefined,
        source_key: query.source_key, cursor: c.cursor, limit: PAGE }) },
    }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  const ids = list.status === 'ok' ? list.data.items.map(x => x.case_id) : undefined
  const custodyAllowed = list.status === 'ok' && actionHint(list.allowed, 'custody.custody.read').allowed
  const custody = useCustodyFor(ids, custodyAllowed)

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const next = new URLSearchParams()
    for (const k of FILTERS) {
      const v = form.get(k)
      if (typeof v === 'string' && v.trim() !== '') next.set(k, v.trim())
    }
    c.reset()
    setParams(next)
  }

  function jump(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const v = String(new FormData(e.currentTarget).get('case_id') ?? '').trim().toLowerCase()
    if (!UUID.test(v)) {
      setJumpError('شناسه پرونده باید یک UUID کامل باشد (جستجوی جزئی، تلفن یا نام پشتیبانی نمی‌شود).')
      return
    }
    setJumpError('')
    void navigate(`/sales/cases/${v}`)
  }

  const filtered = Object.keys(query).length > 0
  return (
    <>
      <PageHeader title="موجودی پرونده‌ها" subtitle="همه پرونده‌های واردشده با منبع، حضور، وضعیت و شاهد نگهداری؛ فقط فیلترهای سمت سرور." />
      <MISSectionNav />
      <div className="mis-two">
        <Card title="فیلتر و جستجو">
          <form className="mis-filters" onSubmit={apply} aria-label="فیلتر موجودی پرونده‌ها">
            <label className="lbl">کلید منبع (دقیق)
              <input className="input mis-ltr" name="source_key" inputMode="numeric" defaultValue={query.source_key ?? ''} placeholder="مثلاً 501"
                aria-describedby="mis-key-hint" />
            </label>
            <label className="lbl">نوع منبع
              <select className="input" name="workflow_type" defaultValue={query.workflow_type ?? ''}>
                <option value="">همه</option>
                {(Object.keys(workflowLabel) as WorkflowType[]).map(w => <option key={w} value={w}>{workflowLabel[w]}</option>)}
              </select>
            </label>
            <label className="lbl">حضور در آخرین واردسازی
              <select className="input" name="presence" defaultValue={query.presence ?? ''}>
                <option value="">همه</option>
                <option value="PRESENT">حاضر</option>
                <option value="ABSENT_IN_LATEST">غایب در آخرین واردسازی</option>
              </select>
            </label>
            <label className="lbl">نگاشت وضعیت
              <select className="input" name="state_mapping" defaultValue={query.state_mapping ?? ''}>
                <option value="">همه</option>
                <option value="KNOWN">شناخته‌شده</option>
                <option value="UNKNOWN">نگاشت‌نشده (خام حفظ شد)</option>
              </select>
            </label>
            <label className="lbl">پیوند مشتری
              <select className="input" name="customer_link_state" defaultValue={query.customer_link_state ?? ''}>
                <option value="">همه</option>
                <option value="RESOLVED">ثبت‌شده با شاهد</option>
                <option value="RELATED">مرتبط</option>
                <option value="UNKNOWN">نامعلوم</option>
                <option value="CONFLICT">تعارض</option>
                <option value="NOT_RECOVERABLE">قابل بازیابی نیست</option>
              </select>
            </label>
            <div className="mis-actions">
              <Button type="submit" variant="primary">اعمال فیلتر</Button>
              {filtered ? <Link to="/mis/cases" onClick={() => c.reset()}>پاک کردن فیلترها</Link> : null}
            </div>
          </form>
          <p id="mis-key-hint" className="muted mis-hint">کلید منبع فقط مرجع ردیف منبع است (نه هویت)؛ تطبیق دقیق، بدون جستجوی جزئی، تلفن یا نام.</p>
        </Card>
        <Card title="رفتن به پرونده با شناسه">
          <form className="mis-filters" onSubmit={jump} aria-label="رفتن به پرونده">
            <label className="lbl">شناسه پرونده (UUID)
              <input className="input mis-ltr" name="case_id" autoComplete="off" spellCheck={false} aria-invalid={jumpError ? true : undefined}
                aria-describedby={jumpError ? 'mis-jump-error' : undefined} />
            </label>
            <div className="mis-actions"><Button type="submit">باز کردن نمای ۳۶۰ پرونده</Button></div>
          </form>
          {jumpError ? <p id="mis-jump-error" className="mis-error" role="alert">{jumpError}</p> : null}
        </Card>
      </div>
      <Card title="فهرست پرونده‌ها">
        {badKey ? <StateBlock kind="error" title="کلید منبع نامعتبر است" detail="کلید منبع یک عدد صحیح مثبت تا ۲۰ رقم است؛ درخواستی ارسال نشد." /> : null}
        <ReadStates state={list} what="پرونده‌ها" blockedDetail={blockedSales} />
        {list.status === 'ok' && !badKey && (list.data.items.length === 0
          ? <StateBlock kind="empty" title={filtered ? 'نتیجه‌ای برای این فیلتر نیست' : 'پرونده‌ای وارد نشده است'}
              {...(filtered ? { detail: 'نبود نتیجه به معنی خالی بودن کل منبع نیست.' } : {})} />
          : <CaseTable items={list.data.items} custody={custody} />)}
        {list.status === 'ok' && !badKey && <Pager page={list.data.page} c={c} />}
        {list.status === 'ok' && <p className="muted mis-hint">{`صفحه‌بندی کلیدی سمت سرور (${PAGE} ردیف در هر صفحه)؛ جمع کل این فیلتر در قرارداد فهرست وجود ندارد و حدس زده نمی‌شود.`}</p>}
      </Card>
    </>
  )
}
