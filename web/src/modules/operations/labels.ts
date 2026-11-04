import type { Tone } from '../../shared/ui/primitives'

export const blockedDetail = 'سطوح عملیات اجرای خدمت پشت پرچم‌های fulfillment.work_read و fulfillment.work_write هستند و به‌طور پیش‌فرض خاموش‌اند.'

export const stateLabel: Record<string, string> = {
  QUEUED: 'در صف', ASSIGNED: 'تخصیص‌یافته', IN_PROGRESS: 'در حال انجام', COMPLETED: 'انجام شد', NOT_EXECUTED: 'انجام نشد (با دلیل)',
}
export const stateTone: Record<string, Tone> = { QUEUED: 'slate', ASSIGNED: 'blue', IN_PROGRESS: 'amber', COMPLETED: 'green', NOT_EXECUTED: 'orange' }

export const eventLabel: Record<string, string> = {
  CREATED: 'ایجاد از رویداد فعال‌سازی آیتم (SHADOW)', ASSIGNED: 'تخصیص به مجری', ACTION_RECORDED: 'ثبت اقدام عملیاتی',
  EVIDENCE_ATTACHED: 'ثبت مدرک (فقط اثر انگشت SHA-256)', COMPLETED: 'ثبت نتیجه نهایی',
}

export const actionLabel: Record<string, string> = { NO_ANSWER: 'عدم پاسخ', FOLLOW_UP: 'پیگیری بعدی', NOTE: 'یادداشت' }
export const outcomeLabel: Record<string, string> = { EXECUTED: 'انجام شد', NOT_EXECUTED: 'انجام نشد' }

export const blockedCommandLabel: Record<string, string> = {
  'fulfillment.cancel_case': 'لغو پرونده', 'fulfillment.reopen_case': 'بازگشایی پرونده', 'fulfillment.submit_review': 'ارسال برای بازبینی',
  'fulfillment.reassign_started': 'تخصیص مجدد پرونده در حال انجام', 'biawin.wallet_charge': 'شارژ کیف پول بیاوین',
  'biawin.activate_production': 'فعال‌سازی واقعی اشتراک',
}

export const financialLabel: Record<string, string> = {
  NONE: 'بدون اثر مالی', CONSUMES_CREDIT: 'مصرف اعتبار (مسدود — D-35)', CHARGES_LEDGER: 'اثر دفتر مالی (مسدود — D-35/D-38)',
}

export const modeLabel: Record<string, string> = {
  TARGET_WRITE_DISABLED: 'نوشتن غیرفعال', SHADOW_COMPARE: 'SHADOW_COMPARE · ارزیابی و ثبت قصد؛ اعمال نمی‌شود',
  CONTROLLED_WRITEBACK: 'CONTROLLED · اعمال روی پرونده سایه سامانه جدید؛ بدون بازنویسی به سامانه قدیمی',
}

export const routeLabel: Record<string, string> = {
  ADDITIONAL_SERVICES: 'خدمات تکمیلی', AFTER_SALES_COMPLAINT: 'شکایت پس از فروش (به بیاوین نمی‌رود)', ARCHIVE: 'بایگانی (حذف یا آزادسازی نیست)',
  UNKNOWN: 'نامعلوم',
}
export const decisionLabel: Record<string, string> = {
  SHADOW_ACTIVATED: 'فعال‌سازی سایه (بدون اثر واقعی)', BLOCKED: 'مسدود', NOT_APPLICABLE: 'نامرتبط با بیاوین',
}
export const decisionTone: Record<string, Tone> = { SHADOW_ACTIVATED: 'teal', BLOCKED: 'red', NOT_APPLICABLE: 'slate' }
export const linkLabel: Record<string, string> = { RESOLVED: 'اثبات‌شده', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض' }

export const opStateLabel: Record<string, string> = {
  PREPARED: 'آماده (ارسال نشده)', DISPATCHING: 'در حال ارسال — نتیجه هنوز معلوم نیست', ACKNOWLEDGED: 'پذیرش اولیه — تحویل تأیید نشده',
  CONFIRMED: 'تأییدشده با شواهد', REJECTED: 'رد شده توسط ارائه‌دهنده', UNKNOWN: 'نامعلوم — تحویل تأیید نشده؛ ارسال مجدد ممنوع',
  RECONCILIATION_REQUIRED: 'نیازمند تطبیق انسانی', CONFIRMED_ABSENT: 'تأیید عدم وجود سفارش', BLOCKED: 'مسدود (تصمیم باز)',
}
export const opStateTone: Record<string, Tone> = {
  PREPARED: 'slate', DISPATCHING: 'amber', ACKNOWLEDGED: 'blue', CONFIRMED: 'green', REJECTED: 'red', UNKNOWN: 'amber', RECONCILIATION_REQUIRED: 'orange',
  CONFIRMED_ABSENT: 'slate', BLOCKED: 'red',
}
export const opTypeLabel: Record<string, string> = { SERVICE_ORDER: 'سفارش خدمت', WALLET_CHARGE: 'شارژ کیف پول (مسدود — D-38)' }

export const short = (id: string | null | undefined) => (id ? id.slice(0, 8) : '—')
