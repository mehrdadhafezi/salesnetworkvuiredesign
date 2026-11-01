import type { Money, ShadowAmount, ShadowRun } from '../../shared/api'
import { Button, StatusPill } from '../../shared/ui/primitives'
import type { PageInfo } from '../../shared/api'
import { coverageLabel, coverageTone, freshnessLabel, reconIcon, reconLabel, reconTone, sourceLabel } from './labels'

/** The SHADOW statement shown on every finance surface. */
export function ShadowBanner() {
  return (
    <div className="fn-banner" role="note" aria-label="هشدار دفتر سایه">
      <strong>SHADOW · سایه</strong>
      <span>
        این دفتر فقط برای اعتبارسنجی معماری و تطبیق است: سامانه رسمی مالی نیست، پول تسویه‌شده، تأییدشده توسط بانک یا تأییدشده توسط مالی نیست و
        مانده رسمی یا حسابداری نیست. هیچ ثبت، تأیید، برگشت، تسویه، استرداد، بستانکار یا بدهکار کردنی از اینجا انجام نمی‌شود.
      </span>
    </div>
  )
}

/** Run, cutoff, outcome, freshness and per-source coverage of an answer. */
export function RunBar({ run }: { run: ShadowRun }) {
  return (
    <div className="fn-meta" aria-label="اجرای سایه">
      <span className="muted fn-ltr">{`run ${run.run_id.slice(0, 8)} · cutoff ${run.source_cutoff} · v${run.derivation_version}`}</span>
      <StatusPill tone={run.outcome === 'SUCCESS' ? 'green' : 'amber'} icon={run.outcome === 'SUCCESS' ? 'check' : 'alert'}
        label={run.outcome === 'SUCCESS' ? 'اجرای فنی کامل (نه تطبیق مالی)' : 'اجرای فنی ناقص'} />
      <StatusPill tone={run.freshness === 'FRESH' ? 'green' : 'orange'} icon={run.freshness === 'FRESH' ? 'check' : 'alert'}
        label={freshnessLabel[run.freshness] ?? run.freshness} />
      {run.sources.map(s => (
        <StatusPill key={s.source} tone={coverageTone[s.status] ?? 'slate'} icon={s.status === 'AVAILABLE' ? 'check' : 'alert'}
          label={`${sourceLabel[s.source] ?? s.source}: ${coverageLabel[s.status] ?? s.status}`} />
      ))}
    </div>
  )
}

/** A money value, or an explicit unknown — never a zero for unknown. */
export function MoneyCell({ m, absent = 'نامعلوم' }: { m?: Money | undefined; absent?: string }) {
  if (!m) return <span className="muted">{absent}</span>
  return <span className="fn-ltr num">{`${m.amount} ${m.unit}`}</span>
}

/** A shadow amount: OVERFLOW claims no value. */
export function AmountCell({ a }: { a: ShadowAmount }) {
  if (a.state === 'OVERFLOW' || !a.money) return <StatusPill tone="slate" icon="question" label="سرریز؛ مقداری ادعا نمی‌شود" />
  return <MoneyCell m={a.money} />
}

export function ReconPill({ state }: { state: string }) {
  return <StatusPill tone={reconTone[state] ?? 'slate'} icon={reconIcon[state] ?? 'question'} label={reconLabel[state] ?? state} />
}

/** Server cursor controls. */
export function Pager({ page, depth, onBack, onNext, label }:
  { page: PageInfo; depth: number; onBack: () => void; onNext: (cursor: string) => void; label: string }) {
  return (
    <nav className="fn-pager" aria-label={label}>
      <Button disabled={depth === 0} onClick={onBack}>صفحه قبل</Button>
      <Button disabled={!page.has_more || !page.next_cursor} onClick={() => { if (page.next_cursor) onNext(page.next_cursor) }}>صفحه بعد</Button>
    </nav>
  )
}
