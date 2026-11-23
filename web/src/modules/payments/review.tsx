import type { ReviewAttempt } from '../../shared/api'
import { StatusPill } from '../../shared/ui/primitives'
import { codes, evidenceStateLabel, evidenceStateTone, gateActionLabel } from './labels'

/** The decision gate of one attempt: why no Finance decision is available. */
export function Gates({ a }: { a: ReviewAttempt }) {
  return (
    <ul className="wr-list" aria-label="دلایل در دسترس نبودن تصمیم مالی">
      {a.gates.map(g => (
        <li key={g.action}>
          <StatusPill tone={g.outcome === 'UNAUTHORIZED' ? 'red' : 'slate'} icon="lock" label={`${gateActionLabel[g.action] ?? g.action}: در دسترس نیست`} />{' '}
          <span className="muted">{codes(g.blocked_by)}</span>
        </li>
      ))}
    </ul>
  )
}

export function ReviewFlags({ a }: { a: ReviewAttempt }) {
  return (
    <span className="pay-row">
      {a.evidence_ready
        ? <StatusPill tone="green" icon="check" label="همه فیش‌ها آماده (D-43 برقرار)" />
        : <StatusPill tone="amber" icon="alert" label="فیش آماده نیست؛ تصمیم مالی مسدود (D-43)" />}
      {a.reviewer_is_submitter && <StatusPill tone="red" icon="ban" label="شما ثبت‌کننده این پرداخت هستید؛ بررسی‌کننده مستقل نیستید" />}
      <StatusPill tone={evidenceStateTone[a.evidence_state] ?? 'slate'} icon="inbox" label={evidenceStateLabel[a.evidence_state] ?? a.evidence_state} />
    </span>
  )
}

