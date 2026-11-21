import type { InvoiceDetail, PageInfo, PaymentAttempt, ReviewAttempt } from '../../shared/api'
import type { ReadState } from '../../shared/read/load'
import { ActivityTimeline, mapRead, type TimelineEvent, type TimelineSource } from '../../shared/timeline/ActivityTimeline'
import { instantTime, legacyTime } from '../../shared/timeline/model'
import { money } from '../payments/labels'
import { AmountText, UserRefText } from './components'
import { attemptStateLabel, blockedDetail, mappingLabel, text } from './labels'
import { AttemptHistory, reviewBlocked, type InvoiceReview } from './workspace'

/**
 * RM-P03 invoice activity. Sources, each separately authorized:
 *  - the invoice record's own time fields (billing.invoice.read);
 *  - the legacy payment attempts page already read by the workspace
 *    (payments.attempt.read hint; not requested otherwise), each with its
 *    RM-P02 per-attempt state history opened on demand;
 *  - attempts submitted in the target system for this invoice, read through
 *    the Finance review queue filtered by the attempt's own invoice id
 *    (finance.review.read on the server; mounted only after the invoice
 *    detail read succeeded).
 * No relationship is inferred; no amount is summed.
 */

type Cursors = { depth: number; back: () => void; next: (c: string) => void }

export function invoiceRecordEvents(d: InvoiceDetail): TimelineEvent[] {
  const v = d.invoice
  const prov = `${v.source.source_system} · ${v.source.source_spec} #${v.source.source_key}`
  const out: TimelineEvent[] = [{
    id: 'issued', type: 'invoice.issued', label: 'صدور فاکتور (طبق منبع)', icon: 'file',
    time: legacyTime(v.issued_at_raw), actor: v.issued_by ? <UserRefText user={v.issued_by} /> : null, provenance: prov,
    facts: [{ label: 'کد (خام)', value: <span className="bi-ltr">{text(v.code_raw)}</span> }],
  }]
  // A last-update stamp is an event only when the source recorded one; its content is unknown.
  if (d.updated_at_raw) {
    out.push({ id: 'updated', type: 'invoice.source_row_updated', label: 'آخرین به‌روزرسانی ردیف فاکتور در منبع (محتوای تغییر نامعلوم)', icon: 'repeat',
      time: legacyTime(d.updated_at_raw), provenance: prov })
  }
  out.push({ id: 'first-seen', type: 'invoice.first_observed', label: 'نخستین مشاهده فاکتور در واردسازی سایه', icon: 'layers',
    time: instantTime(v.source.first_seen_at, 'IMPORT_OBSERVED'), provenance: `batch ${v.source.first_batch_id}` })
  return out
}

export function attemptEvents(items: PaymentAttempt[], zone: string): TimelineEvent[] {
  const out: TimelineEvent[] = []
  for (const a of items) {
    const prov = `${a.source.source_spec} #${a.source.source_key}`
    const state = a.state ? attemptStateLabel[a.state] ?? a.state : mappingLabel[a.state_mapping]
    out.push({
      id: `${a.attempt_id}:created`, type: 'payment.attempt_created', label: 'ثبت تلاش پرداخت (طبق منبع)', icon: 'wallet',
      time: legacyTime(a.created_at_raw), provenance: prov,
      related: { label: 'ردیف در بخش «تلاش‌های پرداخت»', href: '#inv-attempts' },
      facts: [
        { label: 'مبلغ', value: <AmountText amount={a.amount} /> },
        { label: 'وضعیت فعلی (خام → نگاشت)', value: <><span className="bi-ltr">{text(a.state_raw)}</span> · {state}</> },
        { label: 'بارگذارنده (منبع)', value: <UserRefText user={a.uploaded_by} /> },
      ],
      // RM-P02 per-attempt history, read only when opened.
      details: () => <AttemptHistory attemptId={a.attempt_id} zone={zone} />,
    })
    if (a.updated_at_raw) {
      out.push({ id: `${a.attempt_id}:updated`, type: 'payment.attempt_source_row_updated', label: 'آخرین به‌روزرسانی ردیف تلاش در منبع (محتوای تغییر نامعلوم)',
        icon: 'repeat', time: legacyTime(a.updated_at_raw), provenance: prov })
    }
  }
  return out
}

export function submissionEvents(items: ReviewAttempt[]): TimelineEvent[] {
  return items.map(a => ({
    id: a.attempt_id, type: 'payment.submitted', label: 'ثبت پرداخت در سامانه جدید (در انتظار بررسی مالی)', icon: 'send',
    time: instantTime(a.submitted_at, 'TARGET_UTC'),
    actor: <span className="bi-ltr">{`کاربر سامانه ${a.submitted_by_user_id}`}</span>,
    related: { label: 'پرونده بررسی مالی', href: `/finance/review/${a.attempt_id}` },
    facts: [{ label: 'مبلغ · مرحله', value: <span className="bi-ltr">{`${money(a.amount, a.unit)} · #${a.stage_no}`}</span> }],
    flags: [{ tone: 'slate', icon: 'dashed', label: 'تصمیم مالی ثبت نشده' }],
    provenance: `payments · batch ${a.batch_id}`,
  }))
}

export function InvoiceActivity({ detail, pays, pc, canPay, review, zone }: {
  detail: ReadState<InvoiceDetail>
  pays: ReadState<{ items: PaymentAttempt[]; page: PageInfo } | null>
  pc: Cursors
  canPay: boolean
  /** The same Finance review read the workspace section uses (one request). */
  review: InvoiceReview
  zone: string
}) {
  const invoiceOk = detail.status === 'ok'
  const { q: subs, c: rc } = review
  const awaiting = { status: 'awaiting', title: 'این منبع فقط پس از دریافت موفق فاکتور خوانده می‌شود' } as const

  const sources: TimelineSource[] = [
    {
      id: 'invoice', label: 'رکورد فاکتور', icon: 'file', order: 'RECORD_FIELDS', blockedDetail,
      provenance: 'ماژول صورت‌حساب · billing.invoice (فیلدهای زمانی رکورد منبع)',
      read: mapRead(detail, invoiceRecordEvents),
    },
    {
      id: 'attempts', label: 'تلاش‌های پرداخت (منبع قدیمی)', icon: 'wallet', order: 'LIST_ORDER', blockedDetail,
      provenance: 'ماژول پرداخت · payments.payment_attempt؛ سابقه وضعیت هر تلاش در «جزئیات» (زمان مشاهده واردسازی)',
      read: !invoiceOk ? awaiting : !canPay ? { status: 'locked', title: 'دسترسی به شواهد پرداخت ندارید' }
        : pays.status === 'ok' ? (pays.data ? { ...pays, data: attemptEvents(pays.data.items, zone) } : awaiting) : pays,
      paging: pays.status === 'ok' && pays.data ? { ...pc, page: pays.data.page } : undefined,
    },
    {
      id: 'submissions', label: 'پرداخت‌های ثبت‌شده در سامانه جدید', icon: 'send', order: 'LIST_ORDER',
      blockedDetail: reviewBlocked,
      provenance: 'صف بررسی مالی، فیلترشده با شناسه فاکتور خود تلاش (فقط خواندن)',
      read: !invoiceOk ? awaiting : subs.status === 'ok' ? (subs.data ? { ...subs, data: submissionEvents(subs.data.items) } : awaiting) : subs,
      paging: subs.status === 'ok' && subs.data ? { ...rc, page: subs.data.page } : undefined,
    },
  ]
  return (
    <ActivityTimeline label="خط زمانی فعالیت فاکتور" zone={zone} sources={sources}
      coverage="فیلدهای زمانی خود فاکتور، تلاش‌های پرداخت منبع قدیمی (با سابقه وضعیت هر تلاش) و پرداخت‌های ثبت‌شده در سامانه جدید برای همین فاکتور. تاریخچه تغییرات اقلام، مراحل و رسیدها در منبع وجود ندارد و نمایش داده نمی‌شود؛ تصمیم مالی در این نسخه وجود ندارد." />
  )
}
