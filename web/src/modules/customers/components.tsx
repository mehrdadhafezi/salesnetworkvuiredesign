import type { ResolutionState } from '../../shared/api'
import { StatusPill } from '../../shared/ui/primitives'
import { stateIcon, stateLabel, stateTone } from './labels'

export function StatePill({ state }: { state: ResolutionState }) {
  return <StatusPill tone={stateTone[state]} icon={stateIcon[state]} label={stateLabel[state]} />
}

/** States a fact the read surfaces must always show: there is no merge here. */
export function NoMergeNote() {
  return (
    <p className="muted cu-note" role="note">
      این نما فقط خواندنی است. تلفن هرگز هویت مشتری نیست؛ هیچ ادغام، انتخاب مشتری اصلی یا جابه‌جایی سابقه‌ای انجام نمی‌شود.
    </p>
  )
}
