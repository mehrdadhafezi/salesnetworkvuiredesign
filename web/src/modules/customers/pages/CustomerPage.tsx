import '../customers.css'
import { Link, useParams } from 'react-router'
import { useApi, type ContactPoint, type CustomerDetail, type IdentityEvidence, type MergeCase } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { NoMergeNote, StatePill } from '../components'
import { blockedDetail, evidenceLabel, reasonLabel, schemeLabel, strengthLabel, text } from '../labels'

/**
 * Customer 360 (identity section only), read-only. Sections: Customer,
 * Contact points (masked, shared indicator), Identifiers, Evidence, Merge
 * cases, Provenance. No merge, edit or reassign control exists.
 */
export function CustomerPage() {
  const { customerId = '' } = useParams()
  const api = useApi()
  const detail = useRead(`c:${customerId}`, async () => settle(await api.GET('/api/v1/customers/{customer_id}', {
    params: { path: { customer_id: customerId } } }), e => e.data))
  const contacts = useRead(`cp:${customerId}`, async () => settle(await api.GET('/api/v1/customers/{customer_id}/contact-points', {
    params: { path: { customer_id: customerId } } }), e => e.items))
  const evidence = useRead(`ev:${customerId}`, async () => settle(await api.GET('/api/v1/customers/{customer_id}/evidence', {
    params: { path: { customer_id: customerId }, query: { limit: 100 } } }), e => e.items))

  const name = detail.status === 'ok' ? text(detail.data.customer.name_masked) : '…'
  return (
    <>
      <PageHeader title={`مشتری: ${name === '—' ? 'بدون نام ثبت‌شده' : name}`} subtitle="هویت مشتری در مدل سایه">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <p><Link to="/customers">بازگشت به فهرست مشتریان</Link></p>
      <NoMergeNote />
      <div className="cu-sections">
        <Card title="مشتری">
          <ReadStates state={detail} what="مشتری" blockedDetail={blockedDetail} />
          {detail.status === 'ok' && <Summary d={detail.data} />}
        </Card>
        <Card title="نقاط تماس">
          <ReadStates state={contacts} what="نقاط تماس" blockedDetail={blockedDetail} />
          {contacts.status === 'ok' && <Contacts items={contacts.data} />}
        </Card>
        <Card title="شناسه‌ها">
          {detail.status === 'ok' && (detail.data.identifiers.length === 0
            ? <StateBlock kind="empty" title="شناسه‌ای ثبت نشده است" />
            : <ul className="cu-list">{detail.data.identifiers.map(i => (
              <li key={i.identifier_id}>
                {schemeLabel[i.scheme] ?? i.scheme}: <span className="cu-ltr">{i.masked_value}</span>
                {i.derivation === 'PHONE_DERIVED' && <span className="muted"> (مشتق از تلفن؛ اثبات هویت نیست)</span>}
                {i.status === 'CONFLICT' && <> <StatusPill tone="orange" icon="alert" label="تعارض شناسه" /></>}
              </li>))}</ul>)}
        </Card>
        <Card title="شواهد">
          <ReadStates state={evidence} what="شواهد" blockedDetail={blockedDetail} />
          {evidence.status === 'ok' && <Evidence items={evidence.data} />}
        </Card>
        <Card title="پرونده‌های تطبیق">
          {detail.status === 'ok' && (actionHint(detail.allowed, 'customers.merge_case.read').allowed
            ? <Cases items={detail.data.merge_cases} />
            : <StateBlock kind="locked" title="دسترسی به پرونده‌های تطبیق ندارید" />)}
        </Card>
        <Card title="منبع">
          {detail.status === 'ok' && <Provenance d={detail.data} />}
        </Card>
      </div>
    </>
  )
}

function Summary({ d }: { d: CustomerDetail }) {
  const c = d.customer
  return (
    <dl className="cu-kv">
      <dt>وضعیت هویت</dt><dd><StatePill state={c.resolution_state} /> <span className="muted">{c.resolution_reason}</span></dd>
      <dt>زمان مشاهده (خام)</dt><dd className="cu-ltr">{text(c.observed_raw)}</dd>
      <dt>نقاط تماس · شناسه‌ها · شواهد</dt><dd className="num">{c.contact_count} · {c.identifier_count} · {c.evidence_count}</dd>
      <dt>پرونده‌های باز · تعارض</dt><dd className="num">{c.open_case_count} · {c.conflict_case_count}</dd>
    </dl>
  )
}

function Contacts({ items }: { items: ContactPoint[] }) {
  if (items.length === 0) return <StateBlock kind="empty" title="نقطه تماسی ثبت نشده است" />
  return (
    <ul className="cu-list">
      {items.map(p => (
        <li key={p.contact_point_id}>
          {p.kind === 'PHONE' ? 'تلفن' : 'ایمیل'} {p.role === 'PRIMARY' ? 'اصلی' : 'دوم'}: <span className="cu-ltr">{p.masked_value}</span>
          {p.normalization !== 'VALID' && <span className="muted"> ({p.normalization === 'EMPTY' ? 'خالی' : 'نامعتبر'})</span>}
          {p.shared_with_customers > 0 && <> <StatusPill tone="amber" icon="split" label={`مشترک با ${p.shared_with_customers} مشتری دیگر (فقط کشف)`} /></>}
        </li>
      ))}
    </ul>
  )
}

function Evidence({ items }: { items: IdentityEvidence[] }) {
  if (items.length === 0) return <StateBlock kind="empty" title="شاهدی ثبت نشده است" />
  return (
    <div className="cu-table-wrap">
      <table className="cu-table">
        <caption className="sr">شواهد</caption>
        <thead><tr><th scope="col">نوع</th><th scope="col">قوت</th><th scope="col">منشأ</th><th scope="col">مرجع</th><th scope="col">حضور</th></tr></thead>
        <tbody>
          {items.map(e => (
            <tr key={e.evidence_id}>
              <td>{evidenceLabel[e.kind]}</td>
              <td>{strengthLabel[e.strength]}</td>
              <td>{e.origin === 'SUPPLIED' ? 'ارائه‌شده' : 'استنتاجی'}</td>
              <td className="cu-ltr">{e.reference}</td>
              <td>{e.presence === 'PRESENT' ? 'حاضر' : 'در آخرین دسته نیست'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function Cases({ items }: { items: MergeCase[] }) {
  if (items.length === 0) return <StateBlock kind="empty" title="پرونده تطبیقی وجود ندارد" />
  return (
    <ul className="cu-list">
      {items.map(m => (
        <li key={m.merge_case_id}>
          <Link to={`/customers/merge-cases?case=${m.merge_case_id}`}>{reasonLabel[m.reason]}</Link> <StatePill state={m.state} />
          <span className="muted"> · {m.member_customer_ids.length} مشتری · قوی‌ترین شاهد: {strengthLabel[m.max_strength]}</span>
        </li>
      ))}
    </ul>
  )
}

function Provenance({ d }: { d: CustomerDetail }) {
  const p = d.customer.provenance
  return (
    <dl className="cu-kv">
      <dt>سامانه · مشخصه</dt><dd className="cu-ltr">{p.source_system} · {p.source_spec}</dd>
      <dt>کلید منبع</dt><dd className="cu-ltr">{p.source_key}</dd>
      <dt>دسته</dt><dd className="cu-ltr">{p.batch_id}</dd>
      <dt>حضور</dt><dd>{p.presence === 'PRESENT' ? 'در آخرین دسته کامل حاضر است' : 'در آخرین دسته کامل نیست (حذف نشده است)'}</dd>
    </dl>
  )
}
