import '../hr.css'
import { Link } from 'react-router'
import { useApi, type Lookup } from '../../../shared/api'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadStates } from '../components/States'
import { text } from '../labels'
import { settle, useRead } from '../load'
import { HRSectionNav } from '../workspace/SectionNav'

/**
 * Positions and levels (organisation lookups), read only. A position is an
 * HR fact: it is not a role, not a capability and not a credential, and no
 * access is derived from it on this page. No create, rename, deactivate or
 * role-mapping control exists (D-34 open).
 */
export function PositionsPage() {
  const api = useApi()
  const positions = useRead('lookups:position', async () => settle(await api.GET('/api/v1/organization/positions', { params: { query: { limit: 200 } } }),
    e => e.items && e.page ? { items: e.items, more: e.page.has_more } : undefined))
  const levels = useRead('lookups:level', async () => settle(await api.GET('/api/v1/organization/levels', { params: { query: { limit: 200 } } }),
    e => e.items && e.page ? { items: e.items, more: e.page.has_more } : undefined))
  return (
    <>
      <HRSectionNav />
      <PageHeader title="سمت‌ها و سطوح" subtitle="سمت سازمانی ≠ نقش ≠ مجوز ≠ اعتبارنامه. این صفحه هیچ دسترسی‌ای نمی‌سازد یا تغییر نمی‌دهد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <div className="hr-sections hr-two">
        <Card title="سمت‌ها">
          <ReadStates state={positions} what="سمت‌ها" />
          {positions.status === 'ok' && <LookupTable kind="position" items={positions.data.items} more={positions.data.more} />}
        </Card>
        <Card title="سطوح">
          <ReadStates state={levels} what="سطوح" />
          {levels.status === 'ok' && <LookupTable kind="level" items={levels.data.items} more={levels.data.more} />}
        </Card>
      </div>
      <Card title="سمت و دسترسی">
        <p className="muted">نگاشت سمت به نقش و مجوز در سامانه قدیمی وجود دارد، اما در این سامانه فقط شواهد است و تصمیم آن (D-34) باز است.
          برای دیدن دسترسی مؤثر خودتان به <Link to="/hr/access">دسترسی من</Link> بروید.</p>
      </Card>
    </>
  )
}

function LookupTable({ kind, items, more }: { kind: 'position' | 'level'; items: Lookup[]; more: boolean }) {
  if (items.length === 0) return <StateBlock kind="empty" title={kind === 'position' ? 'سمتی وارد نشده است' : 'سطحی وارد نشده است'} />
  const filter = kind === 'position' ? 'position_id' : 'level_id'
  return (
    <>
      {more && <StateBlock kind="unknown" title="فهرست کامل دریافت نشد" detail="فقط ۲۰۰ مورد نخست نمایش داده شده است (PARTIAL)." />}
      <div className="hr-table-wrap">
        <table className="hr-table hr-stack">
          <caption className="sr">{kind === 'position' ? 'سمت‌ها' : 'سطوح'}</caption>
          <thead><tr><th scope="col">عنوان</th><th scope="col">شناسه متنی</th><th scope="col">وضعیت</th><th scope="col">منبع</th><th scope="col">نیروها</th></tr></thead>
          <tbody>
            {items.map(l => (
              <tr key={l.id}>
                <td data-label="عنوان"><b>{text(l.label)}</b>{l.description && <div className="muted">{l.description}</div>}</td>
                <td data-label="شناسه متنی" className="hr-ltr num">{text(l.slug)}</td>
                <td data-label="وضعیت">
                  {l.is_active === null ? <StatusPill tone="amber" icon="question" label="نامعلوم" />
                    : l.is_active ? <StatusPill tone="green" icon="check" label="فعال" /> : <StatusPill tone="slate" icon="ban" label="غیرفعال" />}
                  {l.is_system && <> <StatusPill tone="blue" icon="lock" label="سیستمی" /></>}
                </td>
                <td data-label="منبع" className="num">{`${l.provenance.source_spec} #${l.provenance.source_key}`}</td>
                <td data-label="نیروها"><Link to={`/hr?${filter}=${l.id}`}>مشاهده نیروها</Link></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}
