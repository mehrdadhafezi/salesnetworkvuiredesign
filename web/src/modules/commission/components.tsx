import type { CommissionRun } from '../../shared/api'
import { useSession } from '../../shared/auth/currentSession'
import { formatInstant } from '../../shared/ui/time'
import { StateBlock, StatusPill } from '../../shared/ui/primitives'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 13 commission controlled SHADOW read. No
 * label ever presents a stored shadow calculation as an approved, posted,
 * credited or payable commission; uncertainty states keep their own label.
 */
export const parityLabel: Record<string, string> = {
  MATCH: 'منطبق', AMOUNT_MISMATCH: 'مغایرت مبلغ', RECIPIENT_MISMATCH: 'مغایرت گیرنده', POLICY_MISMATCH: 'مغایرت سیاست (نرخ فعلی ≠ نرخ زمان رویداد)',
  ATTRIBUTION_UNKNOWN: 'انتساب نامعلوم', DUPLICATE_RISK: 'خطر پرداخت تکراری', ENGINE_ROUTING_UNKNOWN: 'موتور نامعلوم', SOURCE_INCOMPLETE: 'منبع ناقص',
  NOT_COMPARABLE: 'قابل مقایسه نیست',
}
export const parityTone: Record<string, Tone> = {
  MATCH: 'green', AMOUNT_MISMATCH: 'red', RECIPIENT_MISMATCH: 'orange', POLICY_MISMATCH: 'amber', ATTRIBUTION_UNKNOWN: 'slate', DUPLICATE_RISK: 'red',
  ENGINE_ROUTING_UNKNOWN: 'slate', SOURCE_INCOMPLETE: 'amber', NOT_COMPARABLE: 'neutral',
}
export const engineLabel: Record<string, string> = {
  PURPOSE_MATRIX: 'ماتریس هدف پرداخت', RULES_AUTO: 'قوانین (اعتباردهی خودکار)', DRY_RUN_POSTING: 'پیش‌نمایش تأییدشده', PROJECT_CARD: 'کارت پروژه بیاوین',
  ENGINE_ROUTING_UNKNOWN: 'موتور نامعلوم',
}
export const eligibilityLabel: Record<string, string> = { ELIGIBLE: 'واجد شرایط (فقط محاسبه سایه)', INELIGIBLE: 'فاقد شرایط', BLOCKED: 'مسدود (تصمیم باز)' }
export const eligibilityTone: Record<string, Tone> = { ELIGIBLE: 'blue', INELIGIBLE: 'neutral', BLOCKED: 'slate' }
export const decisionLabel: Record<string, string> = {
  APPROVED: 'تأییدشده', SAFE_DEFAULT_AVAILABLE: 'پیش‌فرض ایمن موجود', IMPLEMENTATION_BLOCKED: 'پیاده‌سازی مسدود', REQUIRES_HUMAN_OWNER_DECISION: 'نیازمند تصمیم مالک',
  EVIDENCE_INCOMPLETE: 'شواهد ناقص',
}
export const decisionTone: Record<string, Tone> = {
  APPROVED: 'green', SAFE_DEFAULT_AVAILABLE: 'blue', IMPLEMENTATION_BLOCKED: 'orange', REQUIRES_HUMAN_OWNER_DECISION: 'red', EVIDENCE_INCOMPLETE: 'amber',
}
export const runStateLabel: Record<string, string> = { GENERATED: 'تولیدشده', APPROVED: 'تأییدشده (قدیمی)', REJECTED: 'ردشده (قدیمی)', UNKNOWN: 'نامعلوم' }

export const blockedDetail = 'نمای خواندنی پورسانت پشت پرچم ویژگی است (commission.read_shadow · FEATURE_DISABLED).'

export const text = (v: string | number | null | undefined) => (v === null || v === undefined || v === '' ? '—' : String(v))

/** An exact amount with its unit; a missing amount is never shown as zero. */
export function Amount({ v, unit }: { v?: string | null | undefined; unit: string }) {
  if (v === null || v === undefined) return <span className="muted">مبلغ ثبت نشده (صفر نیست)</span>
  return <span className="cm-ltr">{`${v} ${unit}`}</span>
}

/** The SHADOW statement shown on every commission surface. */
export function ShadowBanner() {
  return (
    <div className="cm-banner" role="note" aria-label="هشدار نمای سایه پورسانت">
      <strong>SHADOW · سایه</strong>
      <span>
        این نما فقط نتایج ذخیره‌شده محاسبه سایه و شواهد قدیمی را نشان می‌دهد. هیچ پورسانتی از اینجا محاسبه، تأیید، ثبت، به کیف پول واریز، تسویه یا برگشت
        داده نمی‌شود؛ سامانه قدیمی همچنان مرجع است. وضعیت ثبت: مسدود — مجاز نیست.
      </span>
    </div>
  )
}

export function When({ utc }: { utc: string }) {
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  return <span>{formatInstant(utc, zone)}</span>
}

/** Run, cutoff, rounding, outcome and per-source coverage of an answer. */
export function RunBar({ run }: { run: CommissionRun }) {
  return (
    <div className="cm-meta" aria-label="اجرای سایه پورسانت">
      <span className="muted cm-ltr">{`run ${run.run_id.slice(0, 8)} · ${run.snapshot_id} · v${run.normalization_version}`}</span>
      <span className="muted">برش منبع: <When utc={run.source_cutoff} /></span>
      <StatusPill tone={run.rounding === 'NONE' ? 'orange' : 'slate'} icon="alert"
        label={run.rounding === 'NONE' ? 'بدون قاعده گرد کردن: همه محاسبات مسدود' : 'گرد کردن قدیمی (فقط برای تطبیق؛ OPD-06 باز)'} />
      <StatusPill tone={run.outcome === 'SUCCESS' ? 'green' : 'amber'} icon={run.outcome === 'SUCCESS' ? 'check' : 'alert'}
        label={run.outcome === 'SUCCESS' ? 'اجرای فنی کامل' : 'اجرای فنی ناقص'} />
      {run.sources.filter(s => s.status !== 'AVAILABLE').map(s => (
        <StatusPill key={s.source} tone={s.status === 'MISSING' ? 'red' : 'amber'} icon="alert" label={`${s.source}: ${s.status === 'MISSING' ? 'منبع موجود نیست' : 'پوشش ناقص'}`} />
      ))}
    </div>
  )
}

export function NoRun() {
  return <StateBlock kind="unknown" title="هنوز هیچ اجرای سایه پورسانتی ثبت نشده است"
    detail="نامعلوم است، نه صفر. اجرا فقط با کار آفلاین crm-commission-shadow ساخته می‌شود؛ هیچ صفحه‌ای محاسبه را آغاز نمی‌کند." />
}

export function Count({ label, n, testId }: { label: string; n: number; testId: string }) {
  return (
    <div className="cm-count">
      <span className="muted">{label}</span>
      <strong data-testid={testId}>{n}</strong>
    </div>
  )
}
