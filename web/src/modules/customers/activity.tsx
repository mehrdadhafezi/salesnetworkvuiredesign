import type { CustomerDetail, InvoiceSummary } from '../../shared/api'
import type { ReadState } from '../../shared/read/load'
import { invoicesBlockedDetail } from '../../shared/relations/labels'
import type { RelatedInvoicesRead } from '../../shared/relations/RelatedRecords'
import { ActivityTimeline, mapRead, type TimelineEvent, type TimelineSource } from '../../shared/timeline/ActivityTimeline'
import { instantTime, legacyTime } from '../../shared/timeline/model'
import { blockedDetail, reasonLabel } from './labels'

/**
 * RM-P03 customer activity — the safe subset only. Sources:
 *  - the customer record's own observation (customers module);
 *  - identity merge cases of this customer, only when the server's
 *    customers.merge_case.read hint allows (they arrive with the detail);
 *  - issuance of invoices whose own customer link is RESOLVED to this
 *    customer id, taken from the page's RM-P01 invoice read (exact
 *    customer_id filter, billing-authorized on its own; no extra request)
 *    and shown only after the customer detail read succeeded; rows that do
 *    not confirm the link themselves are dropped, never shown.
 * Case events and payment histories stay on their own pages: aggregating
 * them here would need one read per case / attempt and is not claimed.
 */

export function customerRecordEvents(d: CustomerDetail): TimelineEvent[] {
  const p = d.customer.provenance
  return [{
    id: d.customer.customer_id, type: 'customer.observed', label: 'ثبت مشتری در منبع (زمان مشاهده خام)', icon: 'user',
    time: legacyTime(d.customer.observed_raw), provenance: `${p.source_system} · ${p.source_spec} #${p.source_key} · batch ${p.batch_id}`,
  }]
}

export function mergeCaseEvents(d: CustomerDetail): TimelineEvent[] {
  return d.merge_cases.map(m => ({
    id: m.merge_case_id, type: 'customer.merge_case_created', label: `ایجاد پرونده تطبیق هویت: ${reasonLabel[m.reason] ?? m.reason}`, icon: 'swap',
    time: instantTime(m.created_at, 'TARGET_UTC'),
    related: { label: 'پرونده تطبیق', href: `/customers/merge-cases?case=${m.merge_case_id}` },
    facts: [{ label: 'وضعیت فعلی', value: <span className="cu-ltr">{m.state}</span> }],
    otherTimes: [{ label: 'آخرین بازسازی', time: instantTime(m.refreshed_at, 'TARGET_UTC') }],
    provenance: `customers.merge_case · ${m.member_customer_ids.length} مشتری`,
  }))
}

/** Only rows whose own link confirms RESOLVED to this exact customer. */
export function invoiceIssuedEvents(items: InvoiceSummary[], customerId: string): TimelineEvent[] {
  return items.filter(v => v.customer.state === 'RESOLVED' && v.customer.id === customerId).map(v => {
    const name = `${v.source.source_spec} #${v.source.source_key}`
    return {
      id: v.invoice_id, type: 'invoice.issued', label: 'صدور فاکتور (طبق منبع)', icon: 'file',
      time: legacyTime(v.issued_at_raw),
      actor: v.issued_by ? <span className="cu-ltr">{`کاربر قدیمی #${v.issued_by.legacy_key}`}</span> : null,
      related: { label: `فاکتور ${name}${v.code_raw ? ` · ${v.code_raw}` : ''}`, href: `/billing/invoices/${v.invoice_id}` },
      provenance: `${v.source.source_system} · ${name}`,
    }
  })
}

export function CustomerActivity({ customerId, detail, rel, zone }: {
  customerId: string
  detail: ReadState<CustomerDetail>
  /** The page's RM-P01 invoice read (one request per customer and cursor, billing-authorized on its own). */
  rel: RelatedInvoicesRead
  zone: string
}) {
  const { invoices, c } = rel
  const canMerge = detail.status === 'ok' && detail.allowed?.['customers.merge_case.read']?.allowed === true
  const dropped = detail.status === 'ok' && invoices.status === 'ok' ? invoices.data.items.length - invoiceIssuedEvents(invoices.data.items, customerId).length : 0

  const sources: TimelineSource[] = [
    {
      id: 'customer', label: 'رکورد مشتری', icon: 'user', order: 'RECORD_FIELDS', blockedDetail,
      provenance: 'ماژول مشتری · customers.customer (زمان مشاهده خام منبع)',
      read: mapRead(detail, customerRecordEvents),
    },
    {
      id: 'merge', label: 'پرونده‌های تطبیق هویت', icon: 'swap', order: 'LIST_ORDER', blockedDetail,
      provenance: 'ماژول مشتری · customers.merge_case (سامانه جدید؛ ادغامی انجام نمی‌شود)',
      read: detail.status !== 'ok' ? detail : canMerge ? mapRead(detail, mergeCaseEvents) : { status: 'locked', title: 'پرونده‌های تطبیق در خط زمانی نمی‌آیند (دسترسی ندارید)' },
    },
    {
      id: 'invoices', label: 'صدور فاکتورهای با پیوند تأییدشده', icon: 'file', order: 'LIST_ORDER', blockedDetail: invoicesBlockedDetail,
      provenance: 'ماژول صورت‌حساب · فهرست فاکتورها با فیلتر دقیق customer_id (فقط پیوند RESOLVED)',
      // Shown only after the customer detail read succeeded; the billing read keeps its own authorization state.
      read: detail.status !== 'ok' ? { status: 'awaiting', title: 'این منبع فقط پس از دریافت موفق رکورد مشتری نمایش داده می‌شود' }
        : mapRead(invoices, d => invoiceIssuedEvents(d.items, customerId)),
      paging: detail.status === 'ok' && invoices.status === 'ok' ? { depth: c.depth, page: invoices.data.page, back: c.back, next: c.next } : undefined,
      notes: dropped > 0 ? [{ kind: 'conflict', text: `${dropped} ردیف بدون تأیید پیوند در خود ردیف کنار گذاشته شد.` }] : undefined,
    },
  ]
  return (
    <ActivityTimeline label="خط زمانی فعالیت مشتری" zone={zone} sources={sources}
      coverage={`زیرمجموعه امن: ثبت رکورد مشتری، پرونده‌های تطبیق هویت و صدور فاکتورهایی که پیوند مشتری‌شان تأییدشده (RESOLVED) است. رویدادهای پرونده‌ها (سابقه وضعیت، نگهداری، پیامد تماس) و تاریخچه پرداخت‌ها فقط در صفحه هر پرونده و فاکتور است و اینجا تجمیع نمی‌شود؛ این خط زمانی کامل مشتری نیست. تلفن و نام هرگز پیوند نمی‌سازند.`} />
  )
}
