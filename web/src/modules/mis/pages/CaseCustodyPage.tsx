import '../mis.css'
import { Link, useParams } from 'react-router'
import { useApi, type AttributionSnapshot, type CaseCustody, type CaseSummary, type CustodyEvent } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { formatFaNumber } from '../../../shared/ui/ds'
import { ParityPill, ResolutionPill } from '../../sales/components'
import { eventKindLabel, facetLabel, findingLabel, text, workflowLabel } from '../../sales/labels'
import { Holder, PresencePill, SourcePresence, case360, caseTimeline, presenceText } from '../components'
import { blockedSales, contextLabel, derivationLabel } from '../labels'
import { OperationsStatus } from '../workspace/Operations'
import { MISSectionNav } from '../workspace/SectionNav'

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/

/**
 * RM-P05 custody evidence of one Case: current custody (derived from
 * append-only events, beside the legacy holder of record), the transfer
 * history in source order, custody findings and the immutable historical
 * attribution, each from its own read and authorised on its own. Current
 * custody never overwrites or recomputes historical attribution.
 */
export function CaseCustodyPage() {
  const { caseId = '' } = useParams()
  const valid = UUID.test(caseId)
  const api = useApi()
  // An invalid id is refused here without a request (the server would answer 400).
  const kase = useRead(`mc:${caseId}`, async () => valid ? settle(await api.GET('/api/v1/cases/{case_id}', { params: { path: { case_id: caseId } } }), e => e.data) : { status: 'loading' })
  const custody = useRead(`mk:${caseId}`, async () => valid ? settle(await api.GET('/api/v1/custody/cases/{case_id}', { params: { path: { case_id: caseId } } }), e => e.data) : { status: 'loading' })
  const attribution = useRead(`ma:${caseId}`, async () => valid
    ? settle(await api.GET('/api/v1/cases/{case_id}/attribution', { params: { path: { case_id: caseId } } }), e => e.items) : { status: 'loading' })
  if (!valid) {
    return (
      <>
        <PageHeader title="شواهد نگهداری پرونده" />
        <MISSectionNav />
        <StateBlock kind="error" title="شناسه پرونده نامعتبر است" detail="شناسه باید یک UUID کامل باشد." />
      </>
    )
  }
  const title = kase.status === 'ok' ? `${kase.data.case.source.source_spec} #${kase.data.case.source.source_key}` : 'پرونده'
  const absent = (kase.status === 'ok' && kase.data.case.presence !== 'PRESENT') || (custody.status === 'ok' && custody.data.assignment.presence !== 'PRESENT')
  return (
    <>
      <PageHeader title={`شواهد نگهداری: ${title}`} subtitle="نگهدار فعلی، سابقه انتقال، یافته‌ها و انتساب تاریخی — جدا از هم و بدون ادغام." />
      <MISSectionNav />
      <p className="mis-links mis-nav-links">
        <Link to="/mis/custody">بازگشت به نگهداری و انتساب</Link>
        <Link to={case360(caseId)}>نمای ۳۶۰ پرونده</Link>
        <Link to={caseTimeline(caseId)}>خط زمانی فعالیت پرونده</Link>
      </p>
      <div className="mis-sections">
        <Card title="پرونده و منبع">
          <ReadStates state={kase} what="پرونده" blockedDetail={blockedSales} />
          {kase.status === 'ok' && (
            <dl className="mis-kv">
              <dt>نوع منبع</dt><dd>{workflowLabel[kase.data.case.workflow_type]}</dd>
              <dt>سامانه · مشخصه · کلید</dt><dd className="mis-ltr">{`${kase.data.case.source.source_system} · ${kase.data.case.source.source_spec} · ${kase.data.case.source.source_key}`}</dd>
              <dt>حضور</dt><dd><PresencePill presence={kase.data.case.presence} /> <span className="muted">{presenceText(kase.data.case.presence)}</span></dd>
              <dt>وضعیت (خام)</dt><dd><span className="mis-ltr">{`${kase.data.case.state_namespace}: ${text(kase.data.case.state_raw)}`}</span>
                {kase.data.case.state_mapping === 'UNKNOWN' ? <span className="muted"> (نگاشت‌نشده؛ خام حفظ شد)</span> : null}</dd>
              <dt>شناسه بومی پرونده</dt><dd className="mis-ltr mis-uuid">{kase.data.case.case_id}</dd>
            </dl>
          )}
        </Card>
        <Card title="خاستگاه و دسته‌های واردسازی (Provenance)">
          <ReadStates state={kase} what="خاستگاه پرونده" blockedDetail={blockedSales} />
          {kase.status === 'ok' && <Provenance c={kase.data.case} />}
        </Card>
        <Card title="نگهدار فعلی (Current Custody)">
          <ReadStates state={custody} what="نگهداری" blockedDetail={blockedSales} />
          {custody.status === 'ok' && <CurrentCustody c={custody.data} />}
        </Card>
        <Card title="سابقه انتقال (رویدادهای منبع، به ترتیب منبع)">
          <ReadStates state={custody} what="سابقه انتقال" blockedDetail={blockedSales} />
          {custody.status === 'ok' && (custody.data.events.length === 0
            ? <StateBlock kind="unknown" title="هیچ رویداد نگهداری ثبت نشده است" detail="نبود رویداد یعنی سابقه ناشناخته است، نه اینکه پرونده آزاد یا بدون نگهدار است." />
            : <Events events={custody.data.events} />)}
        </Card>
        <Card title="یافته‌های نگهداری">
          <ReadStates state={custody} what="یافته‌ها" blockedDetail={blockedSales} />
          {custody.status === 'ok' && (custody.data.findings.length === 0
            ? <p className="muted">یافته‌ای ثبت نشده است.</p>
            : (
              <ul className="mis-list" aria-label="یافته‌های نگهداری">
                {custody.data.findings.map(f => (
                  <li key={f.finding_id}>
                    <StatusPill tone={f.presence === 'PRESENT' ? 'orange' : 'slate'} icon="alert" label={findingLabel[f.kind] ?? f.kind} />
                    <span> {f.detail}</span>
                    {f.presence !== 'PRESENT' ? <span className="muted"> (در آخرین واردسازی نیست)</span> : null}
                  </li>
                ))}
              </ul>
            ))}
        </Card>
        <Card title="انتساب تاریخی (تغییرناپذیر)">
          <p className="muted mis-hint">هنگام واردسازی از شواهد ذخیره‌شده منجمد شده است؛ با نگهدار فعلی یا چارت سازمانی امروز بازنویسی یا محاسبه نمی‌شود. صاحب اعتبار از نگهدار فعلی مشتق نمی‌شود.</p>
          <ReadStates state={attribution} what="انتساب" blockedDetail={blockedSales} />
          {attribution.status === 'ok' && (attribution.data.length === 0
            ? <StateBlock kind="unknown" title="انتسابی ثبت نشده است" detail="نبود انتساب یعنی نامعلوم، نه مالک خالی." />
            : <Attribution items={attribution.data} />)}
        </Card>
        <Card title="عملیات روی این پرونده">
          <OperationsStatus absent={absent} />
        </Card>
      </div>
    </>
  )
}

/**
 * RM-P06 recorded provenance of the Case's own source row, exactly as the
 * cases read returns it. Import-run status, file and batch detail have no
 * read contract (integration batches are not exposed over the API), so they
 * are stated as unavailable — never shown as confirmed or complete.
 */
function Provenance({ c }: { c: CaseSummary }) {
  const s = c.source
  const sameBatch = s.first_batch_id === s.last_batch_id
  return (
    <>
      <dl className="mis-kv" aria-label="خاستگاه ثبت‌شده">
        <dt>حضور ردیف منبع</dt><dd><SourcePresence presence={s.presence} /> <span className="muted">{presenceText(s.presence)}</span></dd>
        <dt>اولین دسته مشاهده</dt><dd><span className="mis-ltr mis-uuid">{s.first_batch_id}</span> <span className="muted mis-ltr">{s.first_seen_at}</span></dd>
        <dt>آخرین دسته مشاهده</dt><dd><span className="mis-ltr mis-uuid">{s.last_batch_id}</span> <span className="muted mis-ltr">{s.last_seen_at}</span>
          {sameBatch ? <span className="muted"> (همان دسته اول)</span> : null}</dd>
        <dt>کانال · مرجع مبدأ (خام)</dt><dd className="mis-ltr">{`${text(s.channel_raw)} · ${text(s.origin_ref_raw)}`}</dd>
        <dt>واردسازی · تازه‌سازی پرونده</dt><dd className="mis-ltr">{`${c.imported_at} · ${c.refreshed_at}`}</dd>
        <dt>وضعیت اجرای واردسازی</dt><dd><StatusPill tone="slate" icon="question" label="در دسترس نیست" />
          <span className="muted"> قرارداد خواندن اجرای واردسازی، فایل و دسته در API وجود ندارد؛ کامل یا موفق فرض نمی‌شود.</span></dd>
        <dt>کامل‌بودن و تازگی</dt><dd><span className="muted">{s.presence === 'PRESENT'
          ? 'ردیف در آخرین واردسازی کامل دیده شده است. تازگی پروجکشن‌های گزارشی در «تطبیق و پوشش» جداگانه و با دسترسی خودش نمایش داده می‌شود.'
          : 'ردیف در آخرین واردسازی کامل دیده نشده است؛ آخرین شاهد همان دسته بالا است. این به معنی آزاد یا قابل استفاده مجدد نیست.'}</span></dd>
      </dl>
      <p className="muted mis-hint">خاستگاه فقط مرجع ردیف منبع است، نه هویت؛ هیچ پیوندی از تلفن، نام یا کلید قدیمی استنتاج نمی‌شود.</p>
    </>
  )
}

function CurrentCustody({ c }: { c: CaseCustody }) {
  const a = c.assignment
  return (
    <>
      <div className="mis-meta"><ParityPill parity={a.parity} /><PresencePill presence={a.presence} /></div>
      <div className="mis-owners">
        <section className="mis-owner" aria-label="استنتاج از رویدادها">
          <h3>استنتاج از رویدادها</h3>
          <p><Holder user={a.derived_holder} none="بدون نگهدار استنتاجی" /></p>
          <p className="muted">{derivationLabel[a.derivation] ?? a.derivation}</p>
          <p className="muted">جایگاه (خام): <span className="mis-ltr">{text(a.derived_position_raw)}</span></p>
        </section>
        <section className="mis-owner" aria-label="ثبت‌شده در منبع">
          <h3>ثبت‌شده در منبع</h3>
          <p><Holder user={a.legacy_holder} none="منبع نگهداری ثبت نکرده است" /></p>
          <p className="muted">جایگاه (خام): <span className="mis-ltr">{text(a.legacy_position_raw)}</span></p>
          <p className="muted mis-ltr">{a.legacy_source_spec}</p>
        </section>
      </div>
      <p className="muted mis-hint">{`${formatFaNumber(a.event_count)} رویداد · ${formatFaNumber(a.finding_count)} یافته حاضر. `}
        {a.presence !== 'PRESENT' ? 'پروجکشن در آخرین واردسازی کامل حاضر نیست؛ نگهدار پاک یا آزاد نشده است.' : ''}</p>
    </>
  )
}

function Events({ events }: { events: CustodyEvent[] }) {
  return (
    <div className="mis-table-wrap">
      <table className="mis-table mis-stack">
        <caption className="sr">سابقه انتقال</caption>
        <thead>
          <tr><th scope="col">#</th><th scope="col">نوع (شاهد)</th><th scope="col">از → به</th><th scope="col">انجام‌دهنده</th>
            <th scope="col">زمان (خام منبع)</th><th scope="col">کنش خام · زمینه</th><th scope="col">منبع</th></tr>
        </thead>
        <tbody>
          {events.map((e, i) => (
            <tr key={e.event_id}>
              <td data-label="#" className="num">{formatFaNumber(i + 1)}</td>
              <td data-label="نوع (شاهد)">{eventKindLabel[e.kind] ?? e.kind}</td>
              <td data-label="از → به"><span className="mis-transfer"><Holder user={e.from} none="—" /><span aria-hidden="true">←</span><span className="sr">به</span><Holder user={e.to} none="—" /></span></td>
              <td data-label="انجام‌دهنده"><Holder user={e.actor} none="—" /></td>
              <td data-label="زمان (خام منبع)" className="mis-ltr">{text(e.occurred_raw)}</td>
              <td data-label="کنش خام · زمینه"><span className="mis-ltr">{e.legacy_action}</span><span className="muted"> · {contextLabel[e.context] ?? e.context}</span></td>
              <td data-label="منبع" className="mis-ltr">{`${e.source_spec} #${e.source_key}`}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function Attribution({ items }: { items: AttributionSnapshot[] }) {
  return (
    <div className="mis-table-wrap">
      <table className="mis-table mis-stack">
        <caption className="sr">انتساب تاریخی</caption>
        <thead>
          <tr><th scope="col">جنبه</th><th scope="col">وضعیت تعیین</th><th scope="col">فرد</th><th scope="col">جایگاه (خام)</th><th scope="col">زمان منبع</th><th scope="col">مبنا و شاهد</th></tr>
        </thead>
        <tbody>
          {items.map(s => (
            <tr key={s.snapshot_id}>
              <td data-label="جنبه">{facetLabel[s.facet]}</td>
              <td data-label="وضعیت تعیین"><ResolutionPill resolution={s.resolution} /></td>
              <td data-label="فرد"><Holder user={s.subject} /></td>
              <td data-label="جایگاه (خام)" className="mis-ltr">{text(s.position_raw)}</td>
              <td data-label="زمان منبع" className="mis-ltr">{text(s.source_time_raw)}</td>
              <td data-label="مبنا و شاهد">{s.basis}<span className="muted mis-ltr"> · {`${s.evidence_spec ?? '—'} #${s.evidence_key ?? '—'}`}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
