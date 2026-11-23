import '../mis.css'
import { Link, useSearchParams } from 'react-router'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { DsIcon } from '../../../shared/ui/ds'
import { Count } from '../components'
import { EXCEPTIONS, EXCEPTION_GROUPS, exceptionById, type ExceptionDef } from '../labels'
import { CaseList, CustodyList } from '../workspace/lists'
import { countOf, sourceOf, unavailableWhy, useSalesReports, type Reports } from '../workspace/reports'
import { MISSectionNav } from '../workspace/SectionNav'

/**
 * RM-P05 exceptions and reconciliation queue. Each queue is a stored
 * finding or state, opened with an existing server filter; its count comes
 * from a named aggregate (or is shown as unavailable). Investigation only:
 * nothing here resolves, repairs, merges, reassigns or re-derives anything.
 */
export function ExceptionsPage() {
  const [params] = useSearchParams()
  const reports = useSalesReports()
  const selected = exceptionById(params.get('type'))
  return (
    <>
      <PageHeader title="استثناها و تطبیق" subtitle="یافته‌ها و حالت‌های ثبت‌شده برای بررسی؛ هیچ مورد خودکار حل یا اصلاح نمی‌شود." />
      <MISSectionNav />
      <div className="mis-exc">
        <nav className="mis-exc-nav" aria-label="صف‌های استثنا">
          {EXCEPTION_GROUPS.map(g => (
            <section key={g.id} className="mis-exc-group" aria-label={g.label}>
              <h2>{g.label}</h2>
              <ul>
                {EXCEPTIONS.filter(x => x.group === g.id).map(x => (
                  <li key={x.id}>
                    <Link to={`/mis/exceptions?type=${x.id}`} aria-current={selected?.id === x.id ? 'true' : undefined} className="mis-exc-link">
                      <DsIcon name={x.icon} />
                      <span className="mis-exc-label">{x.label}</span>
                      <Count value={countOf(x.count, reports)} unavailable={unavailableWhy(x.count, reports)} />
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </nav>
        <div className="mis-exc-main">
          {selected ? <ExceptionDetail x={selected} reports={reports} /> : (
            <Card title="یک صف را انتخاب کنید">
              <p className="muted">هر صف یک یافته یا حالت ذخیره‌شده است که با فیلتر سرور فهرست می‌شود. شمارنده‌ها از گزارش‌های تجمیعی موجود خوانده می‌شوند و منبع هر کدام کنار آن آمده است.</p>
              <ul className="mis-list">
                <li><StatusPill tone="slate" icon="lock" label="فقط بررسی" /> حل، ادغام، بازتخصیص یا اصلاح در این صفحه وجود ندارد.</li>
                <li><StatusPill tone="amber" icon="question" label="نامعلوم ≠ صفر" /> شمارنده‌ای که منبع ندارد «در دسترس نیست» نمایش داده می‌شود.</li>
              </ul>
            </Card>
          )}
        </div>
      </div>
    </>
  )
}

function ExceptionDetail({ x, reports }: { x: ExceptionDef; reports: Reports }) {
  const value = countOf(x.count, reports)
  return (
    <>
      <Card title={x.label}>
        <p>{x.meaning}</p>
        <dl className="mis-kv">
          <dt>شمارنده</dt><dd><Count value={value} unavailable={unavailableWhy(x.count, reports)} />
            {value === null && x.count.kind !== 'none' ? <span className="muted"> — {unavailableWhy(x.count, reports)}</span> : null}
            {x.count.kind === 'none' ? <span className="muted"> — {x.count.why}</span> : null}</dd>
          <dt>منبع شمارنده</dt><dd className="mis-ltr mis-src">{sourceOf(x.count)}</dd>
          {x.count.kind !== 'none' ? <><dt>دامنه شمارش</dt><dd>{x.count.scope}</dd></> : null}
          <dt>فهرست</dt><dd className="mis-ltr mis-src">{x.list.read === 'none' ? '—' : `GET /api/v1/${x.list.read === 'custody' ? 'custody/assignments' : 'cases'}?${new URLSearchParams(x.list.params).toString()}`}</dd>
        </dl>
        <p className="muted mis-hint">این صفحه فقط برای بررسی است؛ برای هر پرونده به شواهد نگهداری یا نمای ۳۶۰ بروید. هیچ حل یا اصلاح خودکاری وجود ندارد.</p>
      </Card>
      <Card title="موارد">
        {x.list.read === 'custody' && <CustodyList key={x.id} query={x.list.params} empty="موردی برای این صف ثبت نشده است" />}
        {x.list.read === 'cases' && <CaseList key={x.id} query={x.list.params} empty="موردی برای این صف ثبت نشده است" />}
        {x.list.read === 'none' && <StateBlock kind="unknown" title="فهرست این صف در قرارداد موجود نیست" detail={x.list.why} />}
      </Card>
    </>
  )
}
