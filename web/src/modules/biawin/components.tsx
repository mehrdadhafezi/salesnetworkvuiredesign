import type { FormEvent } from 'react'
import type { BiawinLink, BiawinRun, Outcome, PageInfo } from '../../shared/api'
import { useSession } from '../../shared/auth/currentSession'
import { formatInstant } from '../../shared/ui/time'
import { Button, Card, StateBlock, StatusPill } from '../../shared/ui/primitives'
import { coverageLabel, coverageTone, freshnessLabel, linkLabel, linkTone } from './labels'

/** The SHADOW statement shown on every Biawin / fulfillment surface. */
export function ShadowBanner() {
  return (
    <div className="bw-banner" role="note" aria-label="هشدار نمای سایه">
      <strong>SHADOW · سایه</strong>
      <span>
        این نما فقط شواهد واردشده از نسخه قدیمی را برای اعتبارسنجی و تطبیق نشان می‌دهد: وضعیت زنده تأمین‌کننده نیست و هیچ فعال‌سازی، تعلیق،
        خاتمه، سفارش، شارژ کیف پول، ارسال، تخصیص، تکمیل یا بارگذاری شاهدی از اینجا انجام نمی‌شود.
      </span>
    </div>
  )
}

/** Displays an instant in the session's display zone (Jalali). */
export function When({ utc }: { utc: string }) {
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  return <span>{formatInstant(utc, zone)}</span>
}

/** A raw legacy time: shown verbatim, never converted (its zone is not asserted). */
export function RawTime({ v }: { v?: string | null | undefined }) {
  if (!v) return <span className="muted">ثبت نشده</span>
  return <span className="bw-ltr" title="زمان خام قدیمی؛ منطقه زمانی تأیید نشده">{v}</span>
}

/** Run, cutoff, outcome, freshness and per-source coverage of an answer. */
export function RunBar({ run }: { run: BiawinRun }) {
  return (
    <div className="bw-meta" aria-label="اجرای ورود">
      <span className="muted bw-ltr">{`run ${run.run_id.slice(0, 8)} · ${run.snapshot_id} · v${run.normalization_version}`}</span>
      <span className="muted">برش منبع: <When utc={run.source_cutoff} /></span>
      <StatusPill tone={run.outcome === 'SUCCESS' ? 'green' : 'amber'} icon={run.outcome === 'SUCCESS' ? 'check' : 'alert'}
        label={run.outcome === 'SUCCESS' ? 'ورود فنی کامل' : 'ورود فنی ناقص'} />
      <StatusPill tone={run.freshness === 'FRESH' ? 'green' : 'orange'} icon={run.freshness === 'FRESH' ? 'check' : 'alert'}
        label={freshnessLabel[run.freshness] ?? run.freshness} />
      {run.sources.filter(s => s.status !== 'AVAILABLE').map(s => (
        <StatusPill key={s.source} tone={coverageTone[s.status] ?? 'slate'} icon="alert" label={`${s.source}: ${coverageLabel[s.status] ?? s.status}`} />
      ))}
    </div>
  )
}

/** A cross-module link: its state always, the id only when RESOLVED. */
export function LinkCell({ l }: { l: BiawinLink }) {
  return (
    <span>
      <StatusPill tone={linkTone[l.state] ?? 'slate'} icon={l.state === 'RESOLVED' ? 'check' : 'question'} label={linkLabel[l.state] ?? l.state} />
      {l.legacy_key ? <span className="muted bw-ltr"> {`#${l.legacy_key}`}</span> : null}
    </span>
  )
}

/** A count: a real zero is shown as 0, and the meaning stays explicit. */
export function Count({ label, n, testId }: { label: string; n: number; testId: string }) {
  return (
    <div className="bw-count">
      <span className="muted">{label}</span>
      <strong data-testid={testId}>{n}</strong>
    </div>
  )
}

/**
 * The server answers UNKNOWN (with no run, meta or rows) before any import
 * run exists: a distinct state, never a transport failure and never zero.
 */
export function noRun(e: { outcome: Outcome }): null | undefined {
  return e.outcome === 'UNKNOWN' ? null : undefined
}

export function NoRun() {
  return <StateBlock kind="unknown" title="هنوز هیچ اجرای ورودی ثبت نشده است"
    detail="نامعلوم است، نه صفر و نه خطای ارتباط. اجرا فقط با کار آفلاین crm-biawin-shadow ساخته می‌شود." />
}

/** Allowlisted, server-validated filters only (no free text, no phone). */
export function Filters({ options, values, onApply, label }:
  { options: Record<string, readonly string[]>; values: Record<string, string>; onApply: (v: Record<string, string>) => void; label: string }) {
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const out: Record<string, string> = {}
    for (const k of Object.keys(options)) {
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') out[k] = v
    }
    onApply(out)
  }
  return (
    <Card title="فیلترها">
      <form className="bw-filters" onSubmit={submit} aria-label={label}>
        {Object.entries(options).map(([k, vs]) => (
          <label key={k} className="lbl">{k}
            <select className="input" name={k} defaultValue={values[k] ?? ''}>
              <option value="">همه</option>
              {vs.map(v => <option key={v} value={v}>{v}</option>)}
            </select>
          </label>
        ))}
        <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
      </form>
    </Card>
  )
}

/** Server cursor controls. */
export function Pager({ page, depth, onBack, onNext, label }:
  { page: PageInfo; depth: number; onBack: () => void; onNext: (cursor: string) => void; label: string }) {
  return (
    <nav className="bw-pager" aria-label={label}>
      <Button disabled={depth === 0} onClick={onBack}>صفحه قبل</Button>
      <Button disabled={!page.has_more || !page.next_cursor} onClick={() => { if (page.next_cursor) onNext(page.next_cursor) }}>صفحه بعد</Button>
    </nav>
  )
}
