import { Link } from 'react-router'
import { useApi, type CaseSummary, type CustodyAssignment, type UserRef } from '../../shared/api'
import { settle, useRead, type ReadState } from '../../shared/read/load'
import { StatusPill } from '../../shared/ui/primitives'
import { formatFaNumber } from '../../shared/ui/ds'
import { ParityPill } from '../sales/components'
import { presenceLabel } from './labels'

/** Case 360° (existing page) and its activity timeline section on the same page. */
export const case360 = (id: string) => `/sales/cases/${id}`
export const caseTimeline = (id: string) => `/sales/cases/${id}#activity`
export const caseCustody = (id: string) => `/mis/custody/${id}`

/** A legacy user reference: never a name, never a phone. An unresolved target user stays visible as such. */
export function Holder({ user, none = 'ثبت نشده' }: { user: UserRef | undefined; none?: string }) {
  if (!user) return <span className="muted">{none}</span>
  return (
    <span className="mis-holder">
      <span className="mis-ltr">{`کاربر قدیمی #${user.legacy_key}`}</span>
      {user.user_id ? null : <StatusPill tone="amber" icon="question" label="هویت حل‌نشده" />}
    </span>
  )
}

export function PresencePill({ presence }: { presence: string }) {
  return presence === 'PRESENT'
    ? <StatusPill tone="green" icon="check" label="حاضر" />
    : <StatusPill tone="amber" icon="alert" label={presence === 'ABSENT_IN_LATEST' ? 'غایب در آخرین واردسازی' : presence} />
}

export function presenceText(p: string): string {
  return presenceLabel[p] ?? p
}

/** A count read from a named aggregate, or the honest reason it is not available. */
export function Count({ value, unavailable }: { value: number | null | undefined; unavailable?: string }) {
  if (value === null || value === undefined) return <span className="mis-count mis-count-na" title={unavailable}>—<small> {unavailable ? 'در دسترس نیست' : 'نامعلوم'}</small></span>
  return <span className="mis-count num">{formatFaNumber(value)}</span>
}

export function CaseLinks({ id, label }: { id: string; label: string }) {
  return (
    <span className="mis-links">
      <Link to={case360(id)} className="mis-ltr">{label}</Link>
      <Link to={caseTimeline(id)} className="mis-sub-link">خط زمانی</Link>
    </span>
  )
}

/** Custody evidence for one page of Cases, read with the RM-P05 case_id filter. Not requested without the server hint. */
export type CustodyFor = ReadState<Map<string, CustodyAssignment>> | { status: 'not-allowed' }

export function useCustodyFor(caseIds: string[] | undefined, allowed: boolean): CustodyFor {
  const api = useApi()
  const key = caseIds && allowed ? caseIds.join(',') : ''
  const st = useRead(`custody-for:${key}`, async (): Promise<ReadState<Map<string, CustodyAssignment>>> => {
    if (!caseIds || !allowed || caseIds.length === 0) return { status: 'ok', data: new Map(), outcome: 'SUCCESS', allowed: undefined }
    return settle(await api.GET('/api/v1/custody/assignments', { params: { query: { case_id: caseIds, limit: Math.min(200, caseIds.length) } } }),
      e => e.items ? new Map(e.items.map(a => [a.case_id, a])) : undefined)
  })
  if (caseIds && !allowed) return { status: 'not-allowed' }
  return st
}

/** One inventory row's custody cell: the stored projection or the honest reason none is shown. */
export function CustodyCell({ custody, caseId }: { custody: CustodyFor; caseId: string }) {
  switch (custody.status) {
    case 'not-allowed': return <span className="muted">بدون دسترسی نگهداری</span>
    case 'loading': return <span className="muted">…</span>
    case 'denied': return <span className="muted">بدون دسترسی نگهداری</span>
    case 'blocked': return <span className="muted">غیرفعال</span>
    case 'failed': return <StatusPill tone="slate" icon="question" label="نامعلوم (خطای خواندن)" />
    case 'ok': {
      const a = custody.data.get(caseId)
      if (!a) return <span className="muted">بدون شاهد نگهداری</span>
      return (
        <span className="mis-custody-cell">
          <ParityPill parity={a.parity} />
          <span>نگهدار استنتاجی: <Holder user={a.derived_holder} none="—" /></span>
          {a.finding_count > 0 ? <span className="muted">{`${formatFaNumber(a.finding_count)} یافته`}</span> : null}
        </span>
      )
    }
  }
}

/**
 * RM-P06: the Case source reference of one page of custody rows, read from
 * the cases module with the exact case_id filter of GET /api/v1/cases (one
 * request per visible page, never one per row). Not requested at all unless
 * the server hint allows cases.case.read; a denied read keeps no case data.
 */
export type CasesFor = ReadState<Map<string, CaseSummary>> | { status: 'not-allowed' }

export function useCasesFor(caseIds: string[] | undefined, allowed: boolean): CasesFor {
  const api = useApi()
  const ids = caseIds ? [...new Set(caseIds)] : undefined
  const key = ids && allowed ? ids.join(',') : ''
  const st = useRead(`cases-for:${key}`, async (): Promise<ReadState<Map<string, CaseSummary>>> => {
    if (!ids || !allowed || ids.length === 0) return { status: 'ok', data: new Map(), outcome: 'SUCCESS', allowed: undefined }
    return settle(await api.GET('/api/v1/cases', { params: { query: { case_id: ids, limit: Math.min(200, ids.length) } } }),
      e => e.items ? new Map(e.items.map(c => [c.case_id, c])) : undefined)
  })
  if (caseIds && !allowed) return { status: 'not-allowed' }
  return st
}

/** Links of one Case: Case 360°, its custody evidence and the existing activity timeline. */
export function CaseEvidenceLinks({ id }: { id: string }) {
  return (
    <span className="mis-links">
      <Link to={case360(id)}>نمای ۳۶۰</Link>
      <Link to={caseCustody(id)} className="mis-sub-link">شواهد نگهداری</Link>
      <Link to={caseTimeline(id)} className="mis-sub-link">خط زمانی</Link>
    </span>
  )
}

/**
 * The Case source reference beside a custody row, or the honest reason it
 * is not shown. Only data the cases read returned is rendered: a denied,
 * disabled or failed read shows no source system, spec, key or batch.
 */
export function CaseSourceCell({ cases, caseId }: { cases: CasesFor; caseId: string }) {
  switch (cases.status) {
    case 'not-allowed': return <span className="muted mis-small">منبع پرونده: بدون دسترسی خواندن پرونده</span>
    case 'loading': return <span className="muted mis-small" aria-busy="true">منبع پرونده: در حال خواندن…</span>
    case 'denied': return <span className="muted mis-small">منبع پرونده: بدون دسترسی خواندن پرونده</span>
    case 'blocked': return <span className="muted mis-small">منبع پرونده: غیرفعال</span>
    case 'failed': return <StatusPill tone="slate" icon="question" label="منبع پرونده نامعلوم (خطای خواندن)" />
    case 'ok': {
      const c = cases.data.get(caseId)
      if (!c) return <StatusPill tone="slate" icon="question" label="منبع پرونده نامعلوم (پرونده در ماژول پرونده یافت نشد)" />
      return (
        <span className="mis-source" data-testid="case-source">
          <span className="mis-ltr mis-source-ref">{`${c.source.source_system} · ${c.source.source_spec} · #${c.source.source_key}`}</span>
          <SourcePresence presence={c.source.presence} />
        </span>
      )
    }
  }
}

/** Presence of the Case's own source row; ABSENT_IN_LATEST is never "available". */
export function SourcePresence({ presence }: { presence: string }) {
  return presence === 'PRESENT'
    ? <StatusPill tone="green" icon="check" label="منبع: حاضر" />
    : <StatusPill tone="amber" icon="alert" label={presence === 'ABSENT_IN_LATEST' ? 'منبع: غایب در آخرین واردسازی' : `منبع: ${presence}`} />
}
