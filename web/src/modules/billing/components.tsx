import type { BillingAmount, BillingSource, Money, TotalClass, UserRef } from '../../shared/api'
import { StatusPill } from '../../shared/ui/primitives'
import { classIcon, classLabel, classTone, receiptStateLabel, text, unitLabel, vocabularyLabel, vocabularyTone } from './labels'

/** Exact money with its explicit unit; never converted, never rounded. */
export function MoneyText({ money }: { money: Money | undefined }) {
  if (!money) return <span className="muted">—</span>
  return <span className="bi-ltr">{`${money.amount} ${unitLabel[money.unit] ?? money.unit}`}</span>
}

/** Raw legacy text beside the money it represents (absent is not zero). */
export function AmountText({ amount }: { amount: BillingAmount }) {
  if (amount.representation === 'ABSENT') return <span className="muted">ثبت نشده (صفر نیست)</span>
  if (amount.representation === 'UNREPRESENTABLE') {
    return <span><span className="bi-ltr">{text(amount.raw)}</span> <span className="muted">(قابل نمایش دقیق نیست؛ گرد نشد)</span></span>
  }
  return <span><MoneyText money={amount.money} /> <span className="muted bi-ltr">{`خام: ${text(amount.raw)}`}</span></span>
}

export function VocabularyPill({ mapping }: { mapping: string }) {
  return <StatusPill tone={vocabularyTone[mapping] ?? 'slate'} icon={mapping === 'UNKNOWN' ? 'question' : 'check'} label={vocabularyLabel[mapping] ?? mapping} />
}

export function ClassPill({ cls }: { cls: TotalClass }) {
  return <StatusPill tone={classTone[cls]} icon={classIcon[cls]} label={classLabel[cls]} />
}

export function ReceiptPill({ state }: { state: string }) {
  return <StatusPill tone={state === 'REFERENCE_ONLY' ? 'blue' : 'slate'} icon={state === 'REFERENCE_ONLY' ? 'inbox' : 'question'} label={receiptStateLabel[state] ?? state} />
}

/** A legacy user reference: never a name, never a phone. */
export function UserRefText({ user }: { user: UserRef | undefined }) {
  if (!user) return <span className="muted">—</span>
  return <span className="bi-ltr">{`کاربر قدیمی #${user.legacy_key}`}{user.user_id ? '' : ' (حساب هدف نامعلوم)'}</span>
}

export function SourceText({ source }: { source: BillingSource }) {
  return (
    <span className="bi-ltr">{`${source.source_spec} #${source.source_key}`}
      {source.presence !== 'PRESENT' && <span className="muted"> (در آخرین دسته کامل نیست؛ حذف نشده)</span>}</span>
  )
}

/** Read-only statement shown on every billing surface. */
export function ReadOnlyNote() {
  return (
    <p className="muted bi-note" role="note">
      این نما فقط خواندنی و از مدل سایه است. هیچ صدور، لغو، تأیید مالی، پرداخت، بارگذاری رسید یا ثبت دفتری از اینجا انجام نمی‌شود. وضعیت قطعی فاکتور و جمع قطعی تعیین نشده است.
    </p>
  )
}
