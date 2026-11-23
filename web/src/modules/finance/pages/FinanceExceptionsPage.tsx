import '../workspace/workspace.css'
import { Link, useSearchParams } from 'react-router'
import { useSession } from '../../../shared/auth/currentSession'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { DsIcon } from '../../../shared/ui/ds'
import { Count, InvoiceList, LegacyAttemptList, ReviewList } from '../workspace/components'
import { EXCEPTIONS, EXCEPTION_GROUPS, UNIT_LABEL, exceptionById, listUnit, type ExceptionDef } from '../workspace/exceptions'
import { countOf, sourceOf, unavailableWhy, useFinanceReports, type FinanceReports } from '../workspace/reports'
import { FinanceSectionNav } from '../workspace/sections'

const LIST_PATH = { attempts: 'payments/attempts', invoices: 'invoices', review: 'finance/review/attempts' } as const

/**
 * RM-P07 Finance exceptions (read-only investigation). Each queue is a
 * stored state listed with an existing server filter; its count comes from
 * a named aggregate or is shown as unavailable. Nothing is reconciled,
 * relinked, corrected or retried; no mismatch is derived client-side.
 */
export function FinanceExceptionsPage() {
  const [params] = useSearchParams()
  const reports = useFinanceReports()
  const selected = exceptionById(params.get('type'))
  return (
    <>
      <PageHeader title="استثناها و شواهد حل‌نشده" subtitle="حالت‌های ثبت‌شده برای بررسی؛ هیچ رکورد مالی اینجا تطبیق، اصلاح یا بازپیوند نمی‌شود." />
      <FinanceSectionNav />
      <div className="fw-exc">
        <nav className="fw-exc-nav" aria-label="صف‌های استثنای مالی">
          {EXCEPTION_GROUPS.map(g => (
            <section key={g.id} className="fw-exc-group" aria-label={g.label}>
              <h2>{g.label}</h2>
              <ul>
                {EXCEPTIONS.filter(x => x.group === g.id).map(x => (
                  <li key={x.id}>
                    <Link to={`/finance/exceptions?type=${x.id}`} aria-current={selected?.id === x.id ? 'true' : undefined} className="fw-exc-link">
                      <DsIcon name={x.icon} />
                      <span className="fw-exc-label">{x.label}</span>
                      <Count value={countOf(x.count, reports)} unavailable={unavailableWhy(x.count, reports)} unit={UNIT_LABEL[x.countUnit]} />
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </nav>
        <div className="fw-exc-main">
          {selected ? <ExceptionDetail key={selected.id} x={selected} reports={reports} /> : (
            <Card title="یک صف را انتخاب کنید">
              <p className="muted">هر صف یک حالت ذخیره‌شده در قراردادهای موجود است که با فیلتر سرور فهرست می‌شود. شمارنده‌ها از گزارش‌های تجمیعی خوانده می‌شوند و منبع هر کدام کنار آن آمده است.</p>
              <ul className="fw-list">
                <li><StatusPill tone="slate" icon="lock" label="فقط بررسی" /> تطبیق، اصلاح، بازپیوند یا تلاش دوباره در این صفحه وجود ندارد.</li>
                <li><StatusPill tone="amber" icon="question" label="نامعلوم ≠ صفر" /> شمارنده‌ای که منبع ندارد «در دسترس نیست» نمایش داده می‌شود.</li>
                <li><StatusPill tone="amber" icon="alert" label="ناهم‌خوانی ساخته نمی‌شود" /> فقط حالتی فهرست می‌شود که سرور ذخیره کرده است.</li>
              </ul>
            </Card>
          )}
        </div>
      </div>
    </>
  )
}

function ExceptionDetail({ x, reports }: { x: ExceptionDef; reports: FinanceReports }) {
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const why = unavailableWhy(x.count, reports)
  const unit = listUnit(x)
  const empty = 'موردی برای این صف ثبت نشده است'
  return (
    <>
      <Card title={x.label}>
        <p>{x.meaning}</p>
        <dl className="fw-kv">
          <dt>شمارنده</dt><dd><Count value={countOf(x.count, reports)} unavailable={why} unit={UNIT_LABEL[x.countUnit]} />{why ? <span className="muted"> — {why}</span> : null}</dd>
          <dt>واحد شمارنده</dt><dd data-testid="count-unit">{UNIT_LABEL[x.countUnit]}</dd>
          <dt>واحد فهرست</dt><dd data-testid="list-unit">{unit ? UNIT_LABEL[unit] : 'فهرست قراردادی ندارد'}</dd>
          <dt>منبع شمارنده</dt><dd className="fw-ltr fw-src">{sourceOf(x.count)}</dd>
          {x.count.kind !== 'none' ? <><dt>دامنه شمارش</dt><dd>{x.count.scope}{x.count.note ? ` — ${x.count.note}` : ''}</dd></> : null}
          <dt>فهرست</dt><dd className="fw-ltr fw-src">{x.list.read === 'none' ? '—' : `GET /api/v1/${LIST_PATH[x.list.read]}?${new URLSearchParams(x.list.params).toString()}`}</dd>
        </dl>
        {x.cardinality ? <p className="fw-hint" role="note" data-testid="cardinality"><StatusPill tone="amber" icon="alert" label="دو مجموعه متفاوت" /> {x.cardinality}</p> : null}
        <p className="muted fw-hint">برای هر ردیف به جزئیات فاکتور یا مورد بررسی بروید؛ هیچ اقدام اصلاحی وجود ندارد.</p>
      </Card>
      <Card title="موارد">
        {x.list.read === 'attempts' && <LegacyAttemptList query={x.list.params} empty={empty} />}
        {x.list.read === 'invoices' && <InvoiceList query={x.list.params} empty={empty} />}
        {x.list.read === 'review' && <ReviewList query={x.list.params} zone={zone} empty={empty} />}
        {x.list.read === 'none' && <StateBlock kind="unknown" title="فهرست این صف در قرارداد موجود نیست" detail={x.list.why} />}
      </Card>
    </>
  )
}
