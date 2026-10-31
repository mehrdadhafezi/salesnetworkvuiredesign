import '../hr.css'
import { useApi, type OrgUnit } from '../../../shared/api'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadStates } from '../components/States'
import { findingLabel, kindLabel, text } from '../labels'
import { compact, settle, useRead, type ReadState } from '../load'

const MAX_PAGES = 20

/**
 * Organisation tree from the target OrgUnit read model (one typed tree:
 * department → unit → team). Units whose parent is missing or not recorded
 * are listed separately and labelled; they are never attached to a root.
 */
export function StructurePage() {
  const api = useApi()
  const state = useRead('org-units', async (): Promise<ReadState<{ units: OrgUnit[]; truncated: boolean }>> => {
    const units: OrgUnit[] = []
    let cursor: string | undefined
    for (let i = 0; i < MAX_PAGES; i++) {
      const page = settle(await api.GET('/api/v1/organization/org-units', { params: { query: compact({ limit: 200, cursor }) } }),
        e => e.items && e.page ? { items: e.items, page: e.page } : undefined)
      if (page.status !== 'ok') return page
      units.push(...page.data.items)
      if (!page.data.page.has_more || !page.data.page.next_cursor) return { ...page, data: { units, truncated: false } }
      cursor = page.data.page.next_cursor
    }
    return { status: 'ok', data: { units, truncated: true }, outcome: 'PARTIAL', allowed: undefined }
  })
  return (
    <>
      <PageHeader title="ساختار سازمانی" subtitle="دپارتمان ← واحد ← تیم؛ ساختار واحد با خط گزارش‌دهی یکی نیست." />
      <ReadStates state={state} what="ساختار" />
      {state.status === 'ok' && <Tree units={state.data.units} truncated={state.data.truncated} />}
    </>
  )
}

function Tree({ units, truncated }: { units: OrgUnit[]; truncated: boolean }) {
  if (units.length === 0) return <Card><StateBlock kind="empty" title="واحدی وارد نشده است" /></Card>
  const children = new Map<string, OrgUnit[]>()
  for (const u of units) {
    if (u.parent_state === 'RESOLVED' && u.parent_id) children.set(u.parent_id, [...(children.get(u.parent_id) ?? []), u])
  }
  const order = (a: OrgUnit, b: OrgUnit) => (a.sort_order ?? 0) - (b.sort_order ?? 0) || text(a.label).localeCompare(text(b.label), 'fa')
  const roots = units.filter(u => u.kind === 'DEPARTMENT').sort(order)
  const orphans = units.filter(u => u.parent_state === 'MISSING' || u.parent_state === 'NOT_RECORDED').sort(order)
  const render = (u: OrgUnit) => (
    <li key={u.org_unit_id}>
      <Node u={u} />
      {(children.get(u.org_unit_id) ?? []).length > 0 && <ul>{(children.get(u.org_unit_id) ?? []).sort(order).map(render)}</ul>}
    </li>
  )
  return (
    <>
      {truncated && <StateBlock kind="unknown" title="ساختار کامل دریافت نشد" detail="بخشی از واحدها نمایش داده نشده است (PARTIAL)." />}
      <Card title="درخت سازمانی">
        <ul className="hr-tree" aria-label="درخت سازمانی">{roots.map(render)}</ul>
      </Card>
      <Card title="واحدهای بدون والد معتبر">
        <p className="muted">این واحدها به ریشه‌ای متصل نشده‌اند: والد ثبت‌شده وجود ندارد یا ثبت نشده است. هیچ والدی ساخته یا حدس زده نمی‌شود.</p>
        {orphans.length === 0 ? <p className="muted">ندارد</p> : <ul className="hr-tree" aria-label="واحدهای بدون والد">{orphans.map(render)}</ul>}
      </Card>
    </>
  )
}

function Node({ u }: { u: OrgUnit }) {
  return (
    <span className="hr-node">
      <span className="muted">{kindLabel[u.kind]}</span> <b>{text(u.label)}</b>
      <span className="muted num"> · {u.member_count} نیرو</span>
      {u.is_active === false && <StatusPill tone="slate" icon="ban" label="غیرفعال" />}
      {u.parent_state === 'MISSING' && <StatusPill tone="orange" icon="alert" label="تعارض: والد ناموجود" />}
      {u.parent_state === 'NOT_RECORDED' && <StatusPill tone="amber" icon="question" label="والد ثبت نشده" />}
      {u.finding_kinds.filter(k => k !== 'ORG_UNIT_PARENT_MISSING' && k !== 'ORG_UNIT_PARENT_NOT_RECORDED')
        .map(k => <StatusPill key={k} tone="orange" icon="alert" label={findingLabel[k]} />)}
    </span>
  )
}
