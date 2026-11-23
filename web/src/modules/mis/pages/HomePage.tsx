import '../mis.css'
import { Link } from 'react-router'
import { ReadStates } from '../../../shared/read/ReadStates'
import type { ReadState } from '../../../shared/read/load'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { DsIcon, Metric, MetricGroup, formatFaNumber, type DsIconName } from '../../../shared/ui/ds'
import { MisOverviewSections } from '../../reporting/pages/MisOverviewPage'
import { Count } from '../components'
import { EXCEPTIONS, blockedSales } from '../labels'
import { OperationsStatus } from '../workspace/Operations'
import { countOf, sourceOf, unavailableWhy, useSalesReports } from '../workspace/reports'
import { MISSectionNav, useMisSections } from '../workspace/SectionNav'

const TILES: { id: string; to: string; title: string; icon: DsIconName; text: string }[] = [
  { id: 'mis.cases', to: '/mis/cases', title: 'موجودی پرونده‌ها', icon: 'inbox', text: 'جستجو با کلید منبع، فیلتر حضور، نوع منبع، نگاشت وضعیت و پیوند مشتری؛ نگهداری هر ردیف کنار آن.' },
  { id: 'mis.custody', to: '/mis/custody', title: 'نگهداری و انتساب', icon: 'target', text: 'نگهدار استنتاجی در کنار نگهدار منبع، سابقه انتقال و انتساب تاریخی جدا از هم.' },
  { id: 'mis.exceptions', to: '/mis/exceptions', title: 'استثناها و تطبیق', icon: 'alert', text: 'تعارض و نامعلومی نگهدار، شکاف سابقه، هویت حل‌نشده و پرونده‌های غایب برای بررسی.' },
  { id: 'mis.quality', to: '/mis/source-quality', title: 'کیفیت منبع', icon: 'check', text: 'کیفیت ورودی هر منبع و دانه داده.' },
  { id: 'mis.reconciliation', to: '/mis/reconciliation', title: 'تطبیق و پوشش', icon: 'swap', text: 'اجراهای پروجکشن، پوشش منبع و قراردادهای شاخص.' },
]

const n = (v: number | null | undefined) => (v === null || v === undefined ? undefined : formatFaNumber(v))

/**
 * RM-P05 MIS landing page. Every counter names the existing aggregate it
 * reads (GET /api/v1/sales/coverage-report or /sales/custody-parity, each
 * with its own authorization) and its scope; a counter whose source is not
 * readable is shown as unavailable, never as zero. No KPI, target,
 * eligibility or total is computed here. The existing MIS overview
 * (projection health and FINAL metrics) follows unchanged.
 */
export function HomePage() {
  const granted = useMisSections()
  const reports = useSalesReports()
  const cov = reports.coverage.status === 'ok' ? reports.coverage.data : undefined
  const par = reports.parity.status === 'ok' ? reports.parity.data : undefined
  const covState = (v: number | undefined) => (cov && v !== undefined ? 'ok' as const : 'unknown' as const)
  const parState = (v: number | undefined) => (par && v !== undefined ? 'ok' as const : 'unknown' as const)
  const tiles = TILES.filter(t => granted.has(t.id))
  return (
    <>
      <PageHeader title="امروز MIS" subtitle="فضای کار عملیاتی MIS: موجودی پرونده، نگهداری، استثناها و سلامت داده.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <MISSectionNav />
      <div className="mis-sections">
        {tiles.length > 0 && (
          <nav className="mis-tiles" aria-label="ورود به بخش‌ها">
            {tiles.map(t => (
              <Link key={t.id} to={t.to} className="mis-tile">
                <span className="mis-tile-h"><DsIcon name={t.icon} /><b>{t.title}</b></span>
                <span className="muted">{t.text}</span>
              </Link>
            ))}
          </nav>
        )}
        <Card title="موجودی پرونده‌ها">
          <ReportState state={reports.coverage} what="گزارش پوشش پرونده‌ها" />
          <MetricGroup label="شمارنده‌های موجودی">
            <Metric label="همه پرونده‌های واردشده" value={n(cov?.cases_total)} state={covState(cov?.cases_total)} meaning="coverage-report · cases_total" />
            <Metric label="حاضر در آخرین واردسازی" value={n(cov?.cases_present)} state={covState(cov?.cases_present)} meaning="coverage-report · cases_present" />
            <Metric label="غایب در آخرین واردسازی" value={n(cov ? cov.cases_total - cov.cases_present : undefined)} state={covState(cov?.cases_total)}
              meaning="cases_total − cases_present (حذف یا آزاد نشده)" />
            <Metric label="وضعیت نگاشت‌نشده" value={n(cov?.cases_with_unknown_state_mapping)} state={covState(cov?.cases_with_unknown_state_mapping)}
              meaning="coverage-report · cases_with_unknown_state_mapping" />
          </MetricGroup>
          {cov ? <p className="muted mis-hint">{`سرنخ‌های قدیمی (Legacy Leads Only، نه کل پرونده‌ها): ${cov.legacy_leads_only.count === null || cov.legacy_leads_only.count === undefined ? 'ارزیابی نشده' : formatFaNumber(cov.legacy_leads_only.count)}. «کل پرونده‌ها» (F03) ${cov.total_cases.status === 'EVALUATED' ? 'ارزیابی شده' : 'نهایی نیست'} و اینجا عدد ساخته نمی‌شود.`}</p> : null}
        </Card>
        <Card title="نگهداری">
          <ReportState state={reports.parity} what="گزارش تطابق نگهداری" />
          <MetricGroup label="شمارنده‌های نگهداری">
            <Metric label="پروجکشن‌های نگهداری حاضر" value={n(par?.assignments_present)} state={parState(par?.assignments_present)} meaning="custody-parity · assignments_present" />
            <Metric label="نگهدار تأییدشده (منبع = رویدادها)" value={n(par ? par.current_holder_parity.MATCH ?? 0 : undefined)} state={parState(par?.assignments_present)}
              meaning="custody-parity · current_holder_parity.MATCH" />
            <Metric label="رویدادهای نگهداری" value={n(par?.custody_events)} state={parState(par?.custody_events)} meaning="custody-parity · custody_events" />
            <Metric label="پرونده دارای رویداد" value={n(par?.cases_with_custody_events)} state={parState(par?.cases_with_custody_events)} meaning="custody-parity · cases_with_custody_events" />
          </MetricGroup>
          {par ? (
            <p className="mis-meta">
              {par.full_parity
                ? <StatusPill tone="green" icon="check" label="تطابق کامل همه پروجکشن‌های حاضر" />
                : <StatusPill tone="amber" icon="alert" label="تطابق کامل ادعا نمی‌شود" />}
              <span className="muted">تطابق کامل فقط وقتی ادعا می‌شود که همه پروجکشن‌های حاضر MATCH باشند.</span>
            </p>
          ) : null}
        </Card>
        {granted.has('mis.exceptions') && <Card title="استثناهای ثبت‌شده">
          <ul className="mis-exc-summary" aria-label="خلاصه استثناها">
            {EXCEPTIONS.map(x => (
              <li key={x.id}>
                <Link to={`/mis/exceptions?type=${x.id}`}>{x.label}</Link>
                <Count value={countOf(x.count, reports)} unavailable={unavailableWhy(x.count, reports)} />
                <span className="muted mis-src mis-ltr">{sourceOf(x.count)}</span>
              </li>
            ))}
          </ul>
          <p className="muted mis-hint">شمارنده یافته‌ها تعداد یافته است، نه تعداد پرونده؛ شمارنده‌ها افراز کل نیستند و جمع آن‌ها معنا ندارد.</p>
        </Card>}
        <Card title="عملیات موجودی">
          <OperationsStatus />
        </Card>
      </div>
      <h2 className="mis-h2">سلامت پروجکشن و شاخص‌های نهایی</h2>
      <MisOverviewSections />
    </>
  )
}

function ReportState<T>({ state, what }: { state: ReadState<T>; what: string }) {
  if (state.status === 'ok') return null
  return <ReadStates state={state} what={what} blockedDetail={blockedSales} />
}
