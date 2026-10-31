import type { ApiFailure, ReasonCode } from '../../../shared/api'
import { outcomePresentation } from '../../../shared/ui/outcome'
import { StateBlock } from '../../../shared/ui/primitives'
import type { ReadState } from '../load'

/** Renders every non-data state honestly; returns null when data is ready. */
export function ReadStates<T>({ state, what }: { state: ReadState<T>; what: string }) {
  switch (state.status) {
    case 'loading':
      return <StateBlock kind="loading" title={`در حال دریافت ${what}…`} />
    case 'denied':
      return <StateBlock kind="locked" title="دسترسی به این بخش ندارید" detail={reasonText(state.reasons)} />
    case 'blocked':
      return <StateBlock kind="locked" title="این بخش هنوز فعال نشده است" detail="خواندن منابع انسانی پشت پرچم ویژگی است (FEATURE_DISABLED)." />
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
