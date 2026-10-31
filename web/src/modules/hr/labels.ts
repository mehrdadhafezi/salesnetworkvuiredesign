import type { EmploymentStatus, FindingKind } from '../../shared/api'
import type { Tone } from '../../shared/ui/primitives'

/** Persian labels for contract enums. Raw legacy values are shown separately, never relabelled. */
export const statusLabel = {
  ACTIVE: 'فعال', INACTIVE: 'غیرفعال', SUSPENDED: 'تعلیق', RESIGNED: 'استعفا', TERMINATED: 'قطع همکاری',
  PROBATION: 'آزمایشی', UNMAPPED: 'نگاشت‌نشده (مقدار قدیمی حفظ شد)', NOT_RECORDED: 'ثبت نشده',
} as const satisfies Record<EmploymentStatus, string>

export const findingLabel = {
  MISSING_PARENT: 'مدیر ثبت‌شده وجود ندارد', SELF_PARENT: 'گزارش به خود', CYCLE: 'چرخه در سلسله‌مراتب',
  DUPLICATE_CURRENT: 'انتساب جاری تکراری', MULTIPLE_CURRENT_PARENTS: 'چند مدیر جاری', INTERVAL_OVERLAP: 'هم‌پوشانی بازه',
  HISTORY_GAP: 'شکاف با سابقه پیوسته', UNEXPLAINED_GAP: 'شکاف بدون توضیح', MISSING_CHILD: 'نیروی انتساب وجود ندارد',
  ORG_UNIT_PARENT_MISSING: 'والد واحد وجود ندارد', ORG_UNIT_PARENT_NOT_RECORDED: 'والد واحد ثبت نشده',
  MEMBERSHIP_MISMATCH: 'ناسازگاری دپارتمان/واحد/تیم', CURRENT_FLAG_INCONSISTENT: 'پرچم جاری با تاریخ‌ها ناسازگار',
  HISTORY_CURRENT_DISAGREE: 'ناسازگاری سابقه و وضعیت جاری', UNREADABLE_DATE: 'تاریخ ناخوانا', UNMAPPED_RELATIONSHIP: 'نوع رابطه نگاشت‌نشده',
  MISSING_REFERENCE: 'ارجاع ناموجود', SHARED_USER: 'حساب مشترک بین چند پرونده', USER_NOT_IMPORTED: 'حساب کاربری وارد نشده',
  HISTORY_CHANGED: 'تغییر در سابقه منبع',
} as const satisfies Record<FindingKind, string>

export const severityTone: Record<'CONFLICT' | 'UNKNOWN' | 'INFO', Tone> = { CONFLICT: 'orange', UNKNOWN: 'amber', INFO: 'slate' }

export const severityLabel: Record<'CONFLICT' | 'UNKNOWN' | 'INFO', string> = { CONFLICT: 'تعارض', UNKNOWN: 'نامعلوم', INFO: 'اطلاع' }

export const refStateLabel: Record<'RESOLVED' | 'NOT_RECORDED' | 'MISSING', string> = {
  RESOLVED: 'ثبت‌شده', NOT_RECORDED: 'ثبت نشده', MISSING: 'ارجاع به مورد ناموجود',
}

export const linkLabel: Record<'LINKED' | 'USER_NOT_IMPORTED' | 'USER_NOT_RECORDED' | 'SHARED_USER', string> = {
  LINKED: 'متصل', USER_NOT_IMPORTED: 'حساب وارد نشده', USER_NOT_RECORDED: 'حساب ثبت نشده', SHARED_USER: 'تعارض: حساب مشترک',
}

export const kindLabel: Record<'DEPARTMENT' | 'UNIT' | 'TEAM', string> = { DEPARTMENT: 'دپارتمان', UNIT: 'واحد', TEAM: 'تیم' }

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
