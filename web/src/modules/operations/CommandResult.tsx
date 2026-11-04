import type { components } from '../../shared/api'
import { reasonText } from '../writes/labels'
import { StateBlock, StatusPill } from '../../shared/ui/primitives'
import { modeLabel } from './labels'

type Env = components['schemas']['WorkCommandEnvelope']

/**
 * The honest result of one work-case command. SHADOW_COMPARE is never shown
 * as applied; UNKNOWN is never shown as success and must not be resent.
 */
export function CommandResult({ env }: { env: Env }) {
  const c = env.command
  const blockedBy = c?.blocked_by?.length ? ` · تصمیم‌های باز: ${c.blocked_by.join('، ')}` : ''
  switch (env.outcome) {
    case 'SUCCESS':
      return (
        <div className="op-banner op-result" role="status" aria-label="نتیجه دستور">
          <strong>{c?.applied ? 'روی پرونده سایه سامانه جدید اعمال شد' : 'ارزیابی و ثبت قصد شد (اعمال نشد)'}{c?.replayed ? ' · تکرار همان درخواست' : ''}</strong>
          {c && <StatusPill tone="slate" icon="lock" label={modeLabel[c.rollout_mode] ?? c.rollout_mode} />}
          <span className="muted">هیچ بازنویسی به سامانه قدیمی، تماس با ارائه‌دهنده یا اثر مالی انجام نشد.</span>
        </div>
      )
    case 'BLOCKED':
      return <StateBlock kind="locked" title="این دستور در حال حاضر مجاز نیست" detail={`${reasonText(env.reasons)}${blockedBy}`} />
    case 'CONFLICT':
      return <StateBlock kind="error" title="تعارض: وضعیت پرونده تغییر کرده یا کلید تکراری برای درخواست دیگری است"
        detail={`اطلاعات تازه بارگذاری شد؛ پیش از هر اقدام دوباره بررسی کنید. ${reasonText(env.reasons)}`} />
    case 'UNKNOWN':
      return <StateBlock kind="unknown" title="نتیجه نامعلوم است و باید تطبیق شود"
        detail="این درخواست را دوباره ارسال نکنید؛ نتیجه نامعلوم هرگز موفق نمایش داده نمی‌شود." />
    case 'UNAUTHORIZED':
      return <StateBlock kind="locked" title="اجازه این دستور را ندارید" detail={reasonText(env.reasons)} />
    default:
      return <StateBlock kind="error" title="دستور انجام نشد" detail={reasonText(env.reasons)} />
  }
}

export function ShadowNotice() {
  return (
    <div className="op-banner" role="note">
      <strong>SHADOW · کنترل‌شده</strong>
      <span>پرونده‌ها فقط از فعال‌سازی سایه ساخته می‌شوند. فعال‌سازی واقعی (D-35)، شارژ کیف پول (D-38)، تماس با بیاوین و بازنویسی به سامانه قدیمی غیرفعال است.</span>
    </div>
  )
}

/** One idempotency key per intent (reused for a retry of the same intent). */
export function newKey(): string {
  return crypto.randomUUID().replaceAll('-', '')
}
