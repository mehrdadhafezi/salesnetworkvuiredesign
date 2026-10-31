import type { CaseReason, EvidenceKind, ResolutionState, Strength } from '../../shared/api'
import type { IconName } from '../../shared/ui/icons'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 5 contract enums. States are evidence
 * outcomes, never actions: there is no "merge" or "approve" wording.
 */
export const stateLabel = {
  RESOLVED: 'هم‌ارزی تأییدشده (بدون ادغام)', RELATED: 'مرتبط', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست',
} as const satisfies Record<ResolutionState, string>

export const stateTone: Record<ResolutionState, Tone> = { RESOLVED: 'green', RELATED: 'blue', UNKNOWN: 'amber', CONFLICT: 'orange', NOT_RECOVERABLE: 'slate' }

export const stateIcon: Record<ResolutionState, IconName> = { RESOLVED: 'check', RELATED: 'split', UNKNOWN: 'question', CONFLICT: 'alert', NOT_RECOVERABLE: 'ban' }

export const strengthLabel = { STRONG: 'قوی', MEDIUM: 'متوسط', WEAK: 'ضعیف' } as const satisfies Record<Strength, string>

export const reasonLabel = {
  SHARED_PHONE: 'تلفن مشترک (فقط کشف)', SHARED_ACCOUNT_REF: 'حساب مشترک مشتق از تلفن',
  SHARED_EXCLUSIVE_IDENTIFIER: 'شناسه انحصاری مشترک', SUPPLIED_EVIDENCE: 'شواهد ارائه‌شده',
} as const satisfies Record<CaseReason, string>

export const evidenceLabel = {
  PHONE_MATCH: 'تطابق تلفن', ACCOUNT_REF_MATCH: 'تطابق حساب مشتق از تلفن', SEQUENTIAL_PHONE_REUSE: 'استفاده متوالی از یک تلفن',
  OVERLAPPING_PHONE_OWNERSHIP: 'مالکیت هم‌زمان تلفن', EXCLUSIVE_IDENTIFIER_SHARED: 'شناسه انحصاری مشترک', OTP_CONTACT_CONTROL: 'کنترل تماس با کد یک‌بارمصرف',
  VERIFIED_EQUALITY: 'هم‌ارزی تأییدشده', VERIFIED_DISTINCT: 'تمایز تأییدشده', SOURCE_NOT_RECOVERABLE: 'منبع قابل بازیابی نیست',
} as const satisfies Record<EvidenceKind, string>

export const schemeLabel: Record<string, string> = {
  LEGACY_WP_ACCOUNT: 'حساب قدیمی (مشتق از تلفن)', WOO_CUSTOMER: 'مشتری فروشگاه', BIAWIN: 'بیاوین',
  LEGACY_AGGREGATE_V1: 'شناسه تجمیعی قدیمی', LEGACY_MAP_V2: 'نگاشت قدیمی', CRM_CODE: 'کد CRM',
}

export const blockedDetail = 'خواندن هویت مشتری پشت پرچم ویژگی است (customers.identity_shadow_read · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
