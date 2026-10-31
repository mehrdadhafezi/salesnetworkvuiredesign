import type { ApiFailure, ReasonCode } from '../api'
import { outcomePresentation } from '../ui/outcome'
import { StateBlock } from '../ui/primitives'
import type { ReadState } from './load'

/**
 * Renders every non-data state honestly; returns null when data is ready.
 * blockedDetail names the feature flag behind a BLOCKED / FEATURE_DISABLED read.
 */
export function ReadStates<T>({ state, what, blockedDetail }: { state: ReadState<T>; what: string; blockedDetail: string }) {
  switch (state.status) {
    case 'loading':
      return <StateBlock kind="loading" title={`در حال دریافت ${what}…`} />
    case 'denied':
      return <StateBlock kind="locked" title="دسترسی به این بخش ندارید" detail={reasonText(state.reasons)} />
    case 'blocked':
      // FEATURE_DISABLED is the flag; any other BLOCKED reason (e.g. a
      // LINKAGE_CONFLICT the server refuses to report around) is shown as such.
      return state.reasons.length === 0 || state.reasons.includes('FEATURE_DISABLED')
        ? <StateBlock kind="locked" title="این بخش هنوز فعال نشده است" detail={blockedDetail} />
        : <StateBlock kind="error" title="سرور این خواندن را مسدود کرده است" detail={reasonText(state.reasons)} />
    case 'failed':
      return <FailureState failure={state.failure} />
    case 'ok':
      return null
  }
}

function reasonText(reasons: ReasonCode[]): string {
  return reasons.length ? `کد: ${reasons.join('، ')}` : 'سرور اجازه این خواندن را نمی‌دهد.'
}

export function FailureState({ failure }: { failure: ApiFailure }) {
  if (failure.kind === 'problem') {
    return <StateBlock kind="error" title={failure.problem.title}
      detail={`کد: ${failure.problem.reason_code} · شناسه پیگیری: ${failure.problem.correlation_id}`} />
  }
  return <StateBlock kind="unknown" title="ارتباط با سرویس برقرار نشد" detail={outcomePresentation.UNKNOWN.label} />
}
