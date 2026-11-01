import { Link } from 'react-router'
import type { MetricValue, ReportMeta, ReportSummary } from '../../shared/api'
import { StatusPill } from '../../shared/ui/primitives'
import {
  coverageLabel, coverageTone, freshnessLabel, freshnessTone, reportLabel, sourceLabel, timeBasisLabel, valueStateIcon, valueStateLabel, valueStateTone,
} from './labels'

/** Read-only statement shown on every reporting surface. */
export function ReadOnlyNote() {
  return (
    <p className="muted rp-note" role="note">
      این نما فقط خواندنی و از پروجکشن‌های گزارش است (منبع حقیقت نیست). هیچ ورود، توزیع، برگشت، اصلاح منبع، حذف، عملیات مالی یا خروجی فایل از اینجا انجام نمی‌شود.
    </p>
  )
}

/** Freshness, coverage, scope, time basis and cutoff of an answer. */
export function MetaBar({ meta }: { meta: ReportMeta }) {
  return (
    <div className="rp-meta" aria-label="فراداده پاسخ">
      <StatusPill tone={freshnessTone[meta.freshness] ?? 'slate'} icon={meta.freshness === 'FRESH' ? 'check' : 'alert'} label={freshnessLabel[meta.freshness] ?? meta.freshness} />
      <StatusPill tone={coverageTone[meta.coverage] ?? 'slate'} icon={meta.coverage === 'AVAILABLE' ? 'check' : 'alert'} label={coverageLabel[meta.coverage] ?? meta.coverage} />
      <span className="muted">دامنه: {meta.query.scope} · مبنای زمانی: {timeBasisLabel[meta.query.time_basis] ?? meta.query.time_basis}</span>
      {meta.run
        ? <span className="muted rp-ltr">{`cutoff ${meta.run.source_cutoff} · run ${meta.run.run_id.slice(0, 8)} · v${meta.run.projection_version} · ${meta.run.outcome}`}</span>
        : <span className="muted">هیچ اجرای پروجکشنی پاسخ نمی‌دهد</span>}
      {Object.entries(meta.source_coverage).map(([s, c]) => (
        <span key={s} className="muted">{`${sourceLabel[s] ?? s}: ${coverageLabel[c] ?? c}`}</span>
      ))}
    </div>
  )
}

/** A summary value: only a real evaluated count is a number; zero is zero. */
export function SummaryValue({ summary, state }: { summary?: ReportSummary | undefined; state: string }) {
  if (state === 'NOT_EVALUATED' || state === 'UNKNOWN' || !summary) {
    return <StatusPill tone={valueStateTone[state] ?? 'slate'} icon={valueStateIcon[state] ?? 'question'} label={valueStateLabel[state] ?? state} />
  }
  return <span className="rp-value num">{summary.total}</span>
}

function queryString(m: MetricValue): string {
  const q = m.meta.query
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(q.filters)) p.set(k, v)
  p.set('time_basis', q.time_basis)
  if (q.run_id) p.set('run_id', q.run_id)
  if (q.from) p.set('from', q.from)
  if (q.to) p.set('to', q.to)
  if (q.as_of) p.set('as_of', q.as_of)
  return p.toString()
}

/**
 * One metric card. The definition, scope, time basis, freshness and coverage
 * are always discoverable; the drill-down carries the SAME query contract
 * (filters, scope, time basis and pinned run/cutoff) to the table.
 */
export function MetricCard({ m }: { m: MetricValue }) {
  const c = m.contract
  const s = m.summary
  return (
    <section className="rp-card" aria-label={c.name}>
      <h3>{c.name}</h3>
      <SummaryValue summary={s} state={m.value_state} />
      {m.value_state === 'INCOMPLETE' && <StatusPill tone="amber" icon="alert" label={valueStateLabel.INCOMPLETE ?? 'INCOMPLETE'} />}
      {s?.amounts && s.amounts.map(a => (
        <span key={a.unit} className="rp-ltr">{a.money ? `${a.money.amount} ${a.unit}` : `${a.unit}: سرریز (مقداری ادعا نمی‌شود)`}</span>
      ))}
      {s && c.metric_id.endsWith('by_parity') && <CountList counts={s.counts.parity} />}
      {s && c.metric_id === 'payments.by_state' && <CountList counts={s.counts.state} />}
      {s?.unit_classes && Object.entries(s.unit_classes).map(([u, cls]) => (
        <span key={u} className="rp-ltr">{`${u}: ${Object.entries(cls).map(([k, v]) => `${k} ${v}`).join(' · ')}`}</span>
      ))}
      <MetaBar meta={m.meta} />
      {m.reason && <p className="muted">{m.reason}</p>}
      <details>
        <summary>تعریف شاخص (v{c.version})</summary>
        <dl className="rp-kv">
          <dt>دانه</dt><dd>{c.grain}</dd>
          <dt>منبع</dt><dd>{c.source}</dd>
          <dt>جمعیت</dt><dd>{c.cohort || '—'}</dd>
          <dt>فرمول</dt><dd className="rp-ltr">{c.formula || '—'}</dd>
          <dt>واحد</dt><dd>{c.unit}</dd>
          <dt>دامنه</dt><dd>{c.scope}</dd>
          <dt>پوشش</dt><dd>{c.coverage || '—'}</dd>
          <dt>وضعیت قرارداد</dt><dd>{c.status}{c.status_reason ? ` — ${c.status_reason}` : ''}</dd>
        </dl>
      </details>
      {m.value_state !== 'NOT_EVALUATED' && m.meta.available && (
        <Link to={`/reports/${m.meta.query.report}?${queryString(m)}`}>{`جدول همین شاخص (${reportLabel[m.meta.query.report] ?? m.meta.query.report})`}</Link>
      )}
    </section>
  )
}

export function CountList({ counts }: { counts?: Record<string, number> | undefined }) {
  if (!counts || Object.keys(counts).length === 0) return <span className="muted">هیچ ردیفی در این عضویت نیست</span>
  return (
    <ul className="rp-list">
      {Object.entries(counts).sort().map(([k, v]) => <li key={k} className="rp-ltr">{`${k}: ${v}`}</li>)}
    </ul>
  )
}
