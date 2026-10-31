import { ReadStates as SharedReadStates } from '../../../shared/read/ReadStates'
import type { ReadState } from '../../../shared/read/load'

export { FailureState } from '../../../shared/read/ReadStates'

/** HR read states: the shared renderer with the HR feature flag named. */
export function ReadStates<T>({ state, what }: { state: ReadState<T>; what: string }) {
  return <SharedReadStates state={state} what={what} blockedDetail="خواندن منابع انسانی پشت پرچم ویژگی است (FEATURE_DISABLED)." />
}
