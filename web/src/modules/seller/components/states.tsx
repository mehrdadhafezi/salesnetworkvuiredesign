import { useMemo, type ReactNode } from 'react'
import { Banner, Drawer, DsIcon, PageHead, StatePanel, formatFaNumber, toFaDigits, type DsCount } from '../../../shared/ui/ds'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { SellerUiState, StatePolicy } from '../state/uiState'

/*
 * RM-E04C / A7 — shared Seller presentation states (screen-design §0), built on RM-E01 components.
 * Every piece reads the current presentation state from the preview context; none decides scope,
 * fetches data or grants anything. Blocking states render no screen body at all, so no stale or
 * sensitive value (list rows, drawers, amounts, counts) survives a denial or an expired session.
 */

export const NOT_FOUND_TITLE = 'یافت نشد'
/** One sentence for missing and out-of-scope alike: it never says which (SR-06). */
export const NOT_FOUND_DETAIL = 'این مورد وجود ندارد یا برای شما در دسترس نیست.'
export const RESTRICTED_LABEL = 'محدود'
export const SETTLEMENT_UNDEFINED = '— تعریف نشده'

const BLOCKING: Partial<Record<SellerUiState, { title: string; detail: string }>> = {
  feature_disabled: { title: 'این بخش در حال حاضر فعال نیست', detail: 'قابلیت این بخش خاموش است (FEATURE_DISABLED). هیچ داده‌ای نمایش داده نمی‌شود.' },
  action_denied: { title: 'دسترسی به این بخش ندارید', detail: 'مجوز لازم برای دیدن این بخش وجود ندارد (ACTION_DENIED).' },
  row_scope_denied: { title: 'هویت کاربری شما برای این بخش تأیید نشده است', detail: 'تا هویت شما تأیید نشود هیچ ردیفی نمایش داده نمی‌شود (ROW_SCOPE_DENIED).' },
  session_expired: { title: 'نشست شما به پایان رسیده است', detail: 'اطلاعات این صفحه پاک شد. برای ادامه باید دوباره وارد شوید. (پیش‌نمایش نشست واقعی ندارد.)' },
  failure: { title: 'امکان دریافت لیست وجود ندارد', detail: 'خطای سرور یا شبکه. خواندن داده بدون تغییر چیزی دوباره قابل تلاش است.' },
}

/** Server-answer contracts that do not exist yet (screen-design S2, S3, S5). */
export type ContractNotice = 'no_list_source' | 'no_source'

const CONTRACT_NOTICE: Record<ContractNotice, string> = {
  no_list_source: 'پیش‌نمایش ساختگی: قرارداد فهرست این بخش در سرور هنوز وجود ندارد (NO SERVER SOURCE). ردیف‌ها نمونه‌اند و هیچ وضعیت کسب‌وکاری از آن‌ها استنتاج نمی‌شود.',
  no_source: 'پیش‌نمایش ساختگی: این بخش هیچ منبع داده‌ای در سرور ندارد (NO SERVER SOURCE). مقصد تولیدی آن در دسترس نیست.',
}

/**
 * Screen frame. `head` is the full page head (with KPIs) for normal states; a blocking state renders
 * only the plain title and the state panel — KPIs, lists and drawers are not mounted.
 */
export function ScreenFrame({ title, head, contract, children }: {
  title: string
  head: ReactNode
  contract?: ContractNotice | undefined
  children: ReactNode
}) {
  const { ui, setSim } = useSellerPreview()
  const block = ui.policy.blocksContent ? BLOCKING[ui.state] : undefined
  if (block) {
    const corr = ui.answer.correlationId
    return (
      <>
        <PageHead title={title} />
        <div data-ui-state={ui.state}>
          <StatePanel
            kind={ui.state === 'failure' ? 'error' : 'locked'}
            title={block.title}
            detail={<>{block.detail}{corr && ui.state === 'failure' ? <> کد پیگیری: <span className="mono" dir="ltr">{corr}</span></> : null}</>}
            actions={ui.policy.retry
              ? <button type="button" className="btn btn-primary" onClick={() => setSim('none')}><DsIcon name="refresh" />تلاش مجدد</button>
              : undefined}
          />
        </div>
      </>
    )
  }
  return (
    <>
      {head}
      {contract ? <Banner kind="info">{CONTRACT_NOTICE[contract]}</Banner> : null}
      {ui.state === 'partial'
        ? <Banner kind="incomplete">داده‌های این بخش ناقص است (COVERAGE_INCOMPLETE). مبالغ و جمع‌هایی که به داده ناقص وابسته‌اند نمایش داده نمی‌شوند.</Banner>
        : null}
      {ui.state === 'unknown'
        ? <Banner kind="unknown">برخی وضعیت‌ها با واژگان شناخته‌شده نگاشت نشده‌اند و «نامعلوم» نمایش داده می‌شوند.</Banner>
        : null}
      {ui.state === 'field_denied'
        ? <Banner kind="locked">برخی فیلدها برای شما محدود است (FIELD_DENIED) و هیچ بخشی از مقدارشان نمایش داده نمی‌شود.</Banner>
        : null}
      <div data-ui-state={ui.state} className="sl-frame-body">{children}</div>
    </>
  )
}

/** «محدود» — a withheld field. No partial value, no length hint. */
export function Restricted({ what }: { what: string }) {
  return (
    <span className="pill t-slate sl-restricted" data-restricted="true">
      <DsIcon name="lock" /><span aria-hidden="true">{RESTRICTED_LABEL}</span><span className="sr">{`${what}: محدود — برای شما نمایش داده نمی‌شود`}</span>
    </span>
  )
}

/** A withheld amount («—», reason announced). Distinct from unknown: the value may exist but is not shown. */
export function Withheld({ reason = 'به‌دلیل داده ناقص محاسبه نشد' }: { reason?: string }) {
  return <span className="sl-amount sl-unknown" data-withheld="true"><span aria-hidden="true">—</span><span className="sr">{`مبلغ نمایش داده نمی‌شود: ${reason}`}</span></span>
}

/** Detail lookup: missing and out-of-scope ids share one result. */
export function useDetail<T>(requested: string | null, find: (id: string) => T | undefined): { record: T | null; notFound: boolean } {
  const { ui } = useSellerPreview()
  if (requested === null) return { record: null, notFound: false }
  const found = ui.policy.detailNotFound ? undefined : find(requested)
  return found === undefined ? { record: null, notFound: true } : { record: found, notFound: false }
}

/** The generic not-found drawer. It does not echo the requested id. */
export function NotFoundDrawer({ open, kicker, onClose }: { open: boolean; kicker: string; onClose: () => void }) {
  return (
    <Drawer open={open} onClose={onClose} title={NOT_FOUND_TITLE} kicker={kicker}
      footer={<button type="button" className="btn btn-lg btn-ghost" onClick={onClose}>بستن</button>}>
      {open ? <div data-not-found="true"><StatePanel kind="noresult" title={NOT_FOUND_TITLE} detail={NOT_FOUND_DETAIL} headingLevel={3} /></div> : null}
    </Drawer>
  )
}

/** Counts and data under the current state: exact (0 is a fact on an exact empty answer) or absent («—»). */
const NO_ROWS: readonly never[] = Object.freeze([])

export function useStatePresentation() {
  const { ui } = useSellerPreview()
  return useMemo(() => presentation(ui.state, ui.policy), [ui.state, ui.policy])
}

function presentation(state: SellerUiState, policy: StatePolicy) {
  const absent = policy.countsAbsent
  const empty = state === 'empty'
  return {
    state,
    policy,
    /** The synthetic list as the answer would carry it: an exact empty answer has no rows. Stable references. */
    data<T>(rows: readonly T[]): readonly T[] { return empty ? NO_ROWS : rows },
    count(n: number): DsCount { return absent ? 'unknown' : n },
    countText(n: number): string { return absent ? '—' : toFaDigits(n) },
    /** Props for an RM-E01 Metric holding a count. */
    kpi(n: number): { value?: string | undefined; state?: 'ok' | 'unknown' | undefined; meaning?: string | undefined } {
      return absent ? { state: 'unknown', meaning: 'تعداد در پاسخ سرور نیامده است' } : { value: formatFaNumber(n) }
    },
  }
}
