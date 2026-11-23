import '../finance.css'
import '../workspace/workspace.css'
import { Link } from 'react-router'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { DsIcon, type DsIconName } from '../../../shared/ui/ds'
import { Count, OperationsStatus } from '../workspace/components'
import { EXCEPTIONS } from '../workspace/exceptions'
import { C, countOf, sourceOf, unavailableWhy, useFinanceReports, type CountSource, type FinanceReports } from '../workspace/reports'
import { FinanceSectionNav, useFinanceSections } from '../workspace/sections'
import { FinanceShadowSections } from './FinanceDashboardPage'

const TILES: { id: string; to: string; title: string; icon: DsIconName; text: string }[] = [
  { id: 'finance.invoices', to: '/billing/invoices', title: 'فاکتورها', icon: 'receipt', text: 'فاکتورهای واردشده با منشأ، پیوند مشتری و پرونده و میز کار ۳۶۰ هر فاکتور.' },
  { id: 'finance.attempts', to: '/billing/attempts', title: 'تلاش‌های پرداخت (قدیمی)', icon: 'repeat', text: 'وضعیت خام و نگاشت‌شده، کانال، منشأ تأیید و شواهد درگاه؛ شاهد قدیمی، نه تصمیم مالی.' },
  { id: 'finance.review', to: '/finance/review', title: 'صف بررسی و پرداخت‌های ثبت‌شده', icon: 'checkCircle', text: 'پرداخت‌های ثبت‌شده در سامانه جدید، آمادگی فیش (D-43) و تفکیک وظایف؛ بدون تأیید یا رد.' },
  { id: 'finance.receipts', to: '/finance/receipts', title: 'شواهد رسید', icon: 'inbox', text: 'ارجاع‌های رسید قدیمی و فیش‌های سامانه جدید، جدا از هم و فقط به صورت شمارش.' },
  { id: 'finance.exceptions', to: '/finance/exceptions', title: 'استثناها و شواهد حل‌نشده', icon: 'alert', text: 'پیوند حل‌نشده، وضعیت نامعلوم، تلاش ردشده، فیش ناقص و منشأ ناقص برای بررسی.' },
]

type Counter = { label: string; source: CountSource }

const GROUPS: { title: string; label: string; counters: Counter[] }[] = [
  { title: 'فاکتورها', label: 'شمارنده‌های فاکتور', counters: [
    { label: 'فاکتورهای حاضر', source: C.invoicesPresent },
    { label: 'پیوند مشتری در تعارض', source: C.customerLink('CONFLICT') },
    { label: 'پیوند پرونده در تعارض', source: C.caseLink('CONFLICT') },
    { label: 'فاکتور بدون مرحله پرداخت', source: C.invoicesWithoutStages },
  ] },
  { title: 'تلاش‌های پرداخت قدیمی', label: 'شمارنده‌های تلاش قدیمی', counters: [
    { label: 'تلاش‌های حاضر', source: C.attemptsPresent },
    { label: '«approved» خام (بررسی مالی یا تسویه نیست)', source: C.approvedRaw },
    { label: 'تلاش بدون فاکتور حل‌شده', source: C.attemptsWithoutInvoice },
    { label: 'نیازمند تطبیق', source: C.attemptState('RECONCILIATION_REQUIRED') },
  ] },
  { title: 'پرداخت‌های ثبت‌شده در سامانه جدید', label: 'شمارنده‌های پرداخت ثبت‌شده', counters: [
    { label: 'در انتظار بررسی مالی', source: C.reviewQueue },
    { label: 'فیش آماده (هش بررسی شد)', source: C.targetReceipt('READY') },
    { label: 'فیش غیرقابل بازیابی', source: C.targetReceipt('NOT_RECOVERABLE') },
  ] },
  { title: 'ارجاع‌های رسید قدیمی', label: 'شمارنده‌های رسید قدیمی', counters: [
    { label: 'فقط ارجاع (فایل بررسی نشده)', source: C.receiptsReferenced },
    { label: 'نامعلوم', source: C.receiptsUnknown },
    { label: 'مفقود', source: C.receiptsMissing },
  ] },
]

/** A labelled list of counters, each with its named source and scope (or why it is unavailable). */
export function Counters({ label, counters, reports }: { label: string; counters: Counter[]; reports: FinanceReports }) {
  return (
    <ul className="fw-counters" aria-label={label}>
      {counters.map(c => {
        const why = unavailableWhy(c.source, reports)
        return (
          <li key={c.label}>
            <span>{c.label}</span>
            <Count value={countOf(c.source, reports)} unavailable={why} />
            {why ? <span className="fw-why">{why}</span> : null}
            <span className="fw-src fw-ltr">{sourceOf(c.source)}{c.source.kind !== 'none' ? ` · ${c.source.scope}` : ''}</span>
          </li>
        )
      })}
    </ul>
  )
}

/**
 * RM-P07 Finance workspace landing (route /finance, unchanged URL). Section
 * entry tiles (shown when the shell allows the section), counters bound to
 * the two existing aggregates with their source and scope, the separation
 * of invoice amount / submitted amount / recorded payment evidence, the
 * exception summary, every financial operation listed as unavailable, and
 * the existing shadow-ledger dashboard unchanged below. Nothing here sums
 * money, computes a balance, or decides anything.
 */
export function FinanceHomePage() {
  const granted = useFinanceSections()
  const reports = useFinanceReports()
  const tiles = TILES.filter(t => granted.has(t.id))
  return (
    <>
      <PageHeader title="میز کار مالی" subtitle="فاکتورها، تلاش‌های پرداخت، پرداخت‌های ثبت‌شده، صف بررسی، شواهد رسید و استثناها — فقط خواندنی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <FinanceSectionNav />
      <div className="fw-sections">
        {tiles.length > 0 && (
          <nav className="fw-tiles" aria-label="ورود به بخش‌های مالی">
            {tiles.map(t => (
              <Link key={t.id} to={t.to} className="fw-tile">
                <span className="fw-tile-h"><DsIcon name={t.icon} /><b>{t.title}</b></span>
                <span className="muted">{t.text}</span>
              </Link>
            ))}
          </nav>
        )}
        <div className="fw-two">
          {GROUPS.map(g => (
            <Card key={g.title} title={g.title}>
              <Counters label={g.label} counters={g.counters} reports={reports} />
            </Card>
          ))}
        </div>
        <p className="muted fw-hint" role="note">شمارنده‌ها از گزارش‌های تجمیعی موجود خوانده می‌شوند و منبع و دامنه هر کدام کنارش آمده است. شمارنده‌ها افراز کل نیستند و با هم جمع نمی‌شوند؛ منبعی که در دسترس نیست «در دسترس نیست» نمایش داده می‌شود، نه صفر.</p>
        <Card title="سه مفهوم جدا">
          <div className="fw-concepts">
            <section className="fw-concept" aria-label="مبلغ فاکتور">
              <h3>مبلغ فاکتور</h3>
              <p className="muted">مبالغ اعلامی منبع، جمع اقلام و جمع مراحل، به تفکیک واحد. جمع قطعی فاکتور تعیین نشده است (TAD #19).</p>
            </section>
            <section className="fw-concept" aria-label="مبلغ ثبت‌شده">
              <h3>مبلغ ثبت‌شده</h3>
              <p className="muted">مبلغی که فروشنده در سامانه جدید ثبت کرده است؛ ادعا است، در انتظار بررسی مالی، و وصول وجه نیست.</p>
            </section>
            <section className="fw-concept" aria-label="شواهد پرداخت ثبت‌شده">
              <h3>شواهد پرداخت ثبت‌شده</h3>
              <p className="muted">تلاش‌های قدیمی (درگاه یا دستی) و ارجاع رسید؛ پذیرش درگاه یا تأیید قدیمی، تأیید مالی، تسویه یا ثبت حسابداری نیست.</p>
            </section>
          </div>
        </Card>
        {granted.has('finance.exceptions') && (
          <Card title="استثناها و شواهد حل‌نشده">
            <ExceptionSummary reports={reports} />
          </Card>
        )}
        <Card title="عملیات مالی">
          <OperationsStatus />
        </Card>
      </div>
      <h2 className="fw-h2">دفتر سایه مالی</h2>
      <FinanceShadowSections />
    </>
  )
}

function ExceptionSummary({ reports }: { reports: FinanceReports }) {
  return (
    <ul className="fw-counters" aria-label="خلاصه استثناها">
      {EXCEPTIONS.map(x => (
        <li key={x.id}>
          <Link to={`/finance/exceptions?type=${x.id}`}>{x.label}</Link>
          <Count value={countOf(x.count, reports)} unavailable={unavailableWhy(x.count, reports)} />
          <span className="fw-src fw-ltr">{sourceOf(x.count)}</span>
        </li>
      ))}
    </ul>
  )
}
