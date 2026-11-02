import type { CommandEnvelope } from '../../shared/api'
import { StateBlock, StatusPill } from '../../shared/ui/primitives'
import { modeLabel, reasonText, writebackLabel, writebackTone } from './labels'

/** The honest result of one command: applied ≠ written to legacy. */
export function CommandBanner({ env }: { env: CommandEnvelope }) {
  const c = env.command
  const blockedBy = c?.blocked_by?.length ? ` · تصمیم‌های باز: ${c.blocked_by.join('، ')}` : ''
  switch (env.outcome) {
    case 'SUCCESS':
      return (
        <div className="wr-banner" role="status" aria-label="نتیجه دستور">
          <strong>{c?.applied ? 'در سامانه جدید ثبت شد' : 'ارزیابی و ثبت قصد شد (اعمال نشد)'}{c?.replayed ? ' · تکرار همان درخواست' : ''}</strong>
          {c && <StatusPill tone={writebackTone[c.writeback_state] ?? 'slate'} icon="alert" label={writebackLabel[c.writeback_state] ?? c.writeback_state} />}
          {c && <span className="muted">{modeLabel[c.rollout_mode]}</span>}
        </div>
      )
    case 'BLOCKED':
      return <StateBlock kind="locked" title="این دستور در حال حاضر مجاز نیست" detail={`${reasonText(env.reasons)}${blockedBy}`} />
    case 'CONFLICT':
      return <StateBlock kind="error" title="تعارض: وضعیت تغییر کرده یا کلید تکراری برای درخواست دیگری است"
        detail={`اطلاعات تازه بارگذاری شد؛ پیش از هر اقدام دوباره بررسی کنید. ${reasonText(env.reasons)}`} />
    case 'UNKNOWN':
      return <StateBlock kind="unknown" title="نتیجه نامعلوم است و در حال تطبیق است"
        detail="این درخواست را دوباره ارسال نکنید؛ وضعیت پس از تطبیق نمایش داده می‌شود." />
    case 'UNAUTHORIZED':
      return <StateBlock kind="locked" title="اجازه این دستور را ندارید" detail={reasonText(env.reasons)} />
    default:
      return <StateBlock kind="error" title="دستور انجام نشد" detail={reasonText(env.reasons)} />
  }
}

/** Generates one idempotency key per intent (reused for a retry of the same intent). */
export function newIdempotencyKey(): string {
  return crypto.randomUUID().replaceAll('-', '')
}
