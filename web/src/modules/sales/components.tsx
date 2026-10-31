import type { Parity, UserRef } from '../../shared/api'
import { StatusPill } from '../../shared/ui/primitives'
import { parityIcon, parityLabel, parityTone, resolutionLabel, resolutionTone } from './labels'

export function ParityPill({ parity }: { parity: Parity }) {
  return <StatusPill tone={parityTone[parity]} icon={parityIcon[parity]} label={parityLabel[parity]} />
}

export function ResolutionPill({ resolution }: { resolution: string }) {
  return <StatusPill tone={resolutionTone[resolution] ?? 'slate'} icon={resolution === 'RESOLVED' ? 'check' : 'question'} label={resolutionLabel[resolution] ?? resolution} />
}

/** A legacy user reference: never a name, never a phone. */
export function UserRefText({ user }: { user: UserRef | undefined }) {
  if (!user) return <span className="muted">—</span>
  return <span className="sa-ltr">{`کاربر قدیمی #${user.legacy_key}`}{user.user_id ? '' : ' (حساب هدف نامعلوم)'}</span>
}

/** Read-only statement shown on every sales surface. */
export function ReadOnlyNote() {
  return (
    <p className="muted sa-note" role="note">
      این نما فقط خواندنی و از مدل سایه است. هیچ تخصیص، انتقال، برگشت یا اقدامی از اینجا انجام نمی‌شود؛ تلفن هرگز شناسه پرونده نیست.
    </p>
  )
}
