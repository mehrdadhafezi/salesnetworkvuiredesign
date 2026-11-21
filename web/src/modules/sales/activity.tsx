import type { AttributionSnapshot, CaseCustody, ContactOutcomeEvent, ContactOutcomeView, PageInfo, StateHistoryEntry } from '../../shared/api'
import type { ReadState } from '../../shared/read/load'
import { ActivityTimeline, mapRead, TimeText, type TimelineEvent, type TimelineSource } from '../../shared/timeline/ActivityTimeline'
import { instantTime, legacyTime } from '../../shared/timeline/model'
import { StatusPill } from '../../shared/ui/primitives'
import { outcomeLabel } from '../writes/labels'
import { ResolutionPill, UserRefText } from './components'
import { blockedDetail, eventKindLabel, facetLabel, findingLabel, text } from './labels'

/**
 * RM-P03 case activity: the case's own append-only sources, each read by
 * the case page through its module endpoint and each authorized on its
 * own (case history and attribution: cases.case.read; custody: the
 * custody.custody.read hint, not requested otherwise; contact outcomes:
 * their own read). Nothing is joined or inferred here.
 */

type Cursors = { depth: number; back: () => void; next: (c: string) => void }

const imported = 'ثبت در واردسازی سایه'

export function historyEvents(items: StateHistoryEntry[]): TimelineEvent[] {
  return items.map(h => ({
    id: h.history_id, type: 'case.state_changed', label: 'تغییر وضعیت پرونده', icon: 'swap',
    time: legacyTime(h.occurred_raw),
    transition: { from: text(h.old_state_raw), to: text(h.new_state_raw) },
    actor: h.actor ? <UserRefText user={h.actor} /> : null,
    provenance: `${h.source_spec} #${h.source_key}`,
    otherTimes: [{ label: imported, time: instantTime(h.recorded_at, 'IMPORT_OBSERVED') }],
  }))
}

const holder = (u: { legacy_key: string } | undefined) => (u ? `#${u.legacy_key}` : null)

export function custodyEvents(c: CaseCustody | null): TimelineEvent[] {
  return (c?.events ?? []).map(e => ({
    id: e.event_id, type: `custody.${e.kind.toLowerCase()}`, label: eventKindLabel[e.kind] ?? e.kind, icon: 'user',
    time: legacyTime(e.occurred_raw),
    transition: { from: holder(e.from), to: holder(e.to) },
    actor: e.actor ? <UserRefText user={e.actor} /> : null,
    facts: [{ label: 'کنش خام منبع', value: <span className="sa-ltr">{e.legacy_action}</span> }],
    provenance: `${e.source_spec} #${e.source_key}`,
    otherTimes: [{ label: imported, time: instantTime(e.recorded_at, 'IMPORT_OBSERVED') }],
    details: (
      <dl className="sa-kv">
        <dt>جایگاه مبدأ → مقصد (خام)</dt><dd className="sa-ltr">{`${text(e.from_position_raw)} → ${text(e.to_position_raw)}`}</dd>
        <dt>زمینه</dt><dd className="sa-ltr">{e.context}</dd>
      </dl>
    ),
  }))
}

export function contactOutcomeEvents(items: ContactOutcomeEvent[], zone: string): TimelineEvent[] {
  return items.map(e => ({
    id: e.event_id, type: 'case.contact_outcome_recorded', label: `ثبت پیامد تماس: ${outcomeLabel[e.outcome] ?? e.outcome}`, icon: 'phone',
    time: instantTime(e.occurred_at, 'TARGET_UTC'),
    transition: { from: e.previous_outcome ?? null, to: e.outcome },
    actor: <span className="sa-ltr">{`کاربر سامانه ${e.actor_user_id}`}</span>,
    provenance: `cases.contact_outcome_event · command ${e.command_id}`,
    details: (
      <dl className="sa-kv">
        <dt>نسخه پرونده</dt><dd className="sa-ltr">{`${e.before_version} → ${e.after_version}`}</dd>
        {e.reason ? <><dt>دلیل</dt><dd>{e.reason}</dd></> : null}
        {e.callback_at ? <><dt>زمان تماس مجدد</dt><dd><TimeText t={instantTime(e.callback_at, 'TARGET_UTC')} zone={zone} inline /></dd></> : null}
        <dt>شناسه پیگیری</dt><dd className="sa-ltr">{e.correlation_id}</dd>
      </dl>
    ),
  }))
}

export function attributionEvents(items: AttributionSnapshot[]): TimelineEvent[] {
  return items.map(s => ({
    id: s.snapshot_id, type: `attribution.${s.facet.toLowerCase()}`, label: `انتساب تاریخی: ${facetLabel[s.facet]}`, icon: 'target',
    time: legacyTime(s.source_time_raw),
    facts: [
      { label: 'وضعیت تعیین', value: <ResolutionPill resolution={s.resolution} /> },
      { label: 'فرد', value: <UserRefText user={s.subject} /> },
      { label: 'مبنا', value: s.basis },
    ],
    provenance: `${s.evidence_spec ?? '—'} #${s.evidence_key ?? '—'} · batch ${s.captured_batch_id}`,
    otherTimes: [{ label: 'ثبت اسنپ‌شات', time: instantTime(s.captured_at, 'IMPORT_OBSERVED') }],
  }))
}

export function CaseActivity({ history, hc, custody, canCustody, attribution, outcomes, zone }: {
  history: ReadState<{ items: StateHistoryEntry[]; page: PageInfo }>
  hc: Cursors
  custody: ReadState<CaseCustody | null>
  canCustody: boolean
  attribution: ReadState<AttributionSnapshot[]>
  outcomes: ReadState<ContactOutcomeView>
  zone: string
}) {
  const findings = canCustody && custody.status === 'ok' ? custody.data?.findings ?? [] : []
  const sources: TimelineSource[] = [
    {
      id: 'history', label: 'سابقه وضعیت (فقط افزودنی)', icon: 'history', order: 'SOURCE_ORDINAL', blockedDetail,
      provenance: 'ماژول پرونده · cases.case_state_history (سابقه وضعیت منبع قدیمی)',
      read: mapRead(history, d => historyEvents(d.items)),
      paging: history.status === 'ok' ? { ...hc, page: history.data.page } : undefined,
    },
    {
      id: 'custody', label: 'خط زمانی نگهداری (فقط افزودنی)', icon: 'user', order: 'SOURCE_ORDINAL', blockedDetail, cap: 1000,
      provenance: 'ماژول نگهداری · custody.custody_event (رویدادهای منبع قدیمی)',
      read: canCustody ? mapRead(custody, custodyEvents) : { status: 'locked', title: 'دسترسی به نگهداری ندارید' },
      notes: findings.length > 0 ? [{ kind: 'conflict', text: `${findings.length} یافته نگهداری ثبت شده است؛ سابقه ممکن است ناقص یا متعارض باشد (خودکار اصلاح نمی‌شود).` }] : undefined,
      footer: findings.length > 0 && (
        <ul className="sa-list" aria-label="یافته‌های نگهداری">
          {findings.map(f => <li key={f.finding_id}><StatusPill tone="orange" icon="alert" label={findingLabel[f.kind] ?? f.kind} /> <span className="muted">{f.detail}</span></li>)}
        </ul>
      ),
    },
    {
      id: 'outcomes', label: 'پیامدهای تماس (سامانه جدید)', icon: 'phone', order: 'NEWEST_FIRST', cap: 200,
      blockedDetail: 'پیامدهای تماس فقط وقتی ویژگی ثبت پیامد تماس (contact_outcome_write / cases.nonfinancial_write) برای نشست فعال باشد خوانده می‌شود؛ در غیر این صورت درخواستی ارسال نمی‌شود.',
      provenance: 'ماژول پرونده · cases.contact_outcome_event (فرمان‌های ثبت‌شده در سامانه جدید)',
      read: mapRead(outcomes, d => contactOutcomeEvents(d.history, zone)),
    },
    {
      id: 'attribution', label: 'انتساب تاریخی (اسنپ‌شات)', icon: 'target', order: 'LIST_ORDER', blockedDetail,
      provenance: 'ماژول پرونده · cases.attribution_snapshot (تغییرناپذیر، از شواهد منبع)',
      read: mapRead(attribution, attributionEvents),
    },
  ]
  return (
    <ActivityTimeline label="خط زمانی فعالیت پرونده" zone={zone} sources={sources}
      coverage="سابقه وضعیت، رویدادهای نگهداری، پیامدهای تماس ثبت‌شده در سامانه جدید و اسنپ‌شات‌های انتساب همین پرونده؛ هر منبع جداگانه مجوز می‌گیرد. رویدادهای فاکتور و پرداخت در صفحه فاکتور است." />
  )
}
