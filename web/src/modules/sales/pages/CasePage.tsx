import '../sales.css'
import { Link, useParams } from 'react-router'
import { useApi, type AttributionSnapshot, type CaseCustody, type Facet } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead, type ReadState } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ParityPill, ReadOnlyNote, ResolutionPill, UserRefText } from '../components'
import { blockedDetail, eventKindLabel, facetLabel, findingLabel, text, workflowLabel } from '../labels'
import { Pager, useCursors } from '../pager'

/**
 * Case detail (read only). The four owners (GATE-0 INV-004) are shown as
 * separate, never-collapsed facts: Current Custody (derived, with its parity
 * against the legacy holder), Original Owner/Seller (immutable historical
 * snapshots), Next Actor (not modelled in Slice 6) and Credit Owner (not
 * evaluated). Historical attribution is shown apart from any current
 * organisation view and is never recomputed from it.
 */
export function CasePage() {
  const { caseId = '' } = useParams()
  const api = useApi()
  const detail = useRead(`c:${caseId}`, async () => settle(await api.GET('/api/v1/cases/{case_id}', { params: { path: { case_id: caseId } } }), e => e.data))
  // History is paged with the server cursor (source order); nothing is truncated silently.
  const hc = useCursors()
  const history = useRead(`h:${caseId}:${hc.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/cases/{case_id}/history', {
    params: { path: { case_id: caseId }, query: { limit: 100, ...(hc.cursor ? { cursor: hc.cursor } : {}) } } }),
  e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  const attribution = useRead(`a:${caseId}`, async () => settle(await api.GET('/api/v1/cases/{case_id}/attribution', {
    params: { path: { case_id: caseId } } }), e => e.items))
  const canCustody = detail.status === 'ok' && actionHint(detail.allowed, 'custody.custody.read').allowed
  const custody = useRead(`k:${caseId}:${canCustody}`, async (): Promise<ReadState<CaseCustody | null>> => canCustody
    ? settle(await api.GET('/api/v1/custody/cases/{case_id}', { params: { path: { case_id: caseId } } }), e => e.data)
    : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined })

  const title = detail.status === 'ok' ? `${detail.data.case.source.source_spec} #${detail.data.case.source.source_key}` : '…'
  return (
    <>
      <PageHeader title={`پرونده: ${title}`} subtitle="مدل سایه فروش">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <p><Link to="/sales/cases">بازگشت به فهرست پرونده‌ها</Link></p>
      <ReadOnlyNote />
      <div className="sa-sections">
        <Card title="پرونده">
          <ReadStates state={detail} what="پرونده" blockedDetail={blockedDetail} />
          {detail.status === 'ok' && (
            <dl className="sa-kv">
              <dt>نوع منبع</dt><dd>{workflowLabel[detail.data.case.workflow_type]}</dd>
              <dt>وضعیت (خام)</dt><dd><span className="sa-ltr">{`${detail.data.case.state_namespace}: ${text(detail.data.case.state_raw)}`}</span>
                {detail.data.case.state_mapping === 'UNKNOWN' && <span className="muted"> (نگاشت‌نشده؛ مقدار خام حفظ شد)</span>}</dd>
              <dt>پیوند مشتری</dt><dd><ResolutionPill resolution={detail.data.case.customer_link_state} /> <span className="muted">{detail.data.case.customer_link_reason}</span></dd>
            </dl>
          )}
        </Card>
        <Card title="چهار مالک (بدون ادغام)">
          <FourOwners custody={custody} attribution={attribution} canCustody={canCustody} />
        </Card>
        <Card title="منبع و پیوندها">
          {detail.status === 'ok' && (
            <>
              <dl className="sa-kv">
                <dt>سامانه · مشخصه · کلید</dt><dd className="sa-ltr">{`${detail.data.case.source.source_system} · ${detail.data.case.source.source_spec} · ${detail.data.case.source.source_key}`}</dd>
                <dt>کانال (خام)</dt><dd className="sa-ltr">{text(detail.data.case.source.channel_raw)}</dd>
                <dt>حضور</dt><dd>{detail.data.case.presence === 'PRESENT' ? 'در آخرین دسته کامل حاضر است' : 'در آخرین دسته کامل نیست (حذف نشده است)'}</dd>
              </dl>
              {detail.data.aliases.length === 0 ? <p className="muted">پیوند صریحی با نمایش دیگری ثبت نشده است.</p> : (
                <ul className="sa-list" aria-label="پیوندها">
                  {detail.data.aliases.map(a => (
                    <li key={a.alias_id}>
                      <ResolutionPill resolution={a.resolution} /> <span className="sa-ltr">{`${a.related_source_spec} #${a.related_source_key}`}</span>
                      {a.related_case_id && <> · <Link to={`/sales/cases/${a.related_case_id === detail.data.case.case_id ? a.case_id : a.related_case_id}`}>پرونده مرتبط</Link></>}
                      <span className="muted"> — {a.basis}</span>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </Card>
        <Card title="سابقه وضعیت (فقط افزودنی)">
          <ReadStates state={history} what="سابقه" blockedDetail={blockedDetail} />
          {history.status === 'ok' && (history.data.items.length === 0 ? <StateBlock kind="empty" title="سابقه وضعیتی ثبت نشده است" /> : (
            <ol className="sa-list" aria-label="سابقه وضعیت">
              {history.data.items.map(h => (
                <li key={h.history_id}><span className="sa-ltr">{`${text(h.old_state_raw)} → ${text(h.new_state_raw)}`}</span>
                  <span className="muted"> · {text(h.occurred_raw)} · </span><UserRefText user={h.actor} /></li>
              ))}
            </ol>
          ))}
          {history.status === 'ok' && <Pager page={history.data.page} c={hc} />}
        </Card>
        <Card title="خط زمانی نگهداری (فقط افزودنی)">
          {!canCustody ? <StateBlock kind="locked" title="دسترسی به نگهداری ندارید" /> : (
            <>
              <ReadStates state={custody} what="نگهداری" blockedDetail={blockedDetail} />
              {custody.status === 'ok' && custody.data && <Timeline c={custody.data} />}
            </>
          )}
        </Card>
      </div>
    </>
  )
}

function snapshotOf(list: AttributionSnapshot[], facet: Facet): AttributionSnapshot | undefined {
  const of = list.filter(s => s.facet === facet)
  return of[of.length - 1]
}

function FourOwners({ custody, attribution, canCustody }: { custody: ReadState<CaseCustody | null>; attribution: ReadState<AttributionSnapshot[]>; canCustody: boolean }) {
  const a = custody.status === 'ok' && custody.data ? custody.data.assignment : undefined
  return (
    <div className="sa-owners">
      <section className="sa-owner" aria-label="نگهدار فعلی">
        <h3>نگهدار فعلی (Current Custody)</h3>
        {!canCustody ? <p className="muted">دسترسی ندارید</p> : custody.status !== 'ok' ? <p className="muted">…</p> : !a ? <p className="muted">—</p> : (
          <>
            <ParityPill parity={a.parity} />
            <p>استنتاج از رویدادها: <UserRefText user={a.derived_holder} /></p>
            <p>نگهدار ثبت‌شده در منبع: <UserRefText user={a.legacy_holder} /></p>
          </>
        )}
      </section>
      <section className="sa-owner" aria-label="مالک و فروشنده اولیه">
        <h3>مالک / فروشنده اولیه (تاریخی، تغییرناپذیر)</h3>
        {attribution.status !== 'ok' ? <ReadStates state={attribution} what="انتساب" blockedDetail={blockedDetail} /> : (['ORIGINAL_OWNER', 'ORIGINAL_SELLER'] as Facet[]).map(f => {
          const s = snapshotOf(attribution.data, f)
          return (
            <p key={f}>{facetLabel[f]}: {s ? <><ResolutionPill resolution={s.resolution} /> <UserRefText user={s.subject} />
              <span className="muted"> · {text(s.position_raw)} · {text(s.source_time_raw)} — {s.basis}</span></> : <span className="muted">ثبت نشده</span>}</p>
          )
        })}
        <p className="muted">از ساختار سازمانی فعلی محاسبه نمی‌شود؛ سرپرست امروز سرپرست دیروز نیست.</p>
      </section>
      <section className="sa-owner" aria-label="اقدام‌کننده بعدی">
        <h3>اقدام‌کننده بعدی (Next Actor)</h3>
        <StatusPill tone="slate" icon="question" label="در این برش مدل نشده است" />
      </section>
      <section className="sa-owner" aria-label="صاحب اعتبار">
        <h3>صاحب اعتبار (Credit Owner)</h3>
        {attribution.status === 'ok' && snapshotOf(attribution.data, 'CREDIT_OWNER')
          ? <ResolutionPill resolution={snapshotOf(attribution.data, 'CREDIT_OWNER')?.resolution ?? 'UNKNOWN'} />
          : <StatusPill tone="slate" icon="question" label="ارزیابی نشده" />}
        <p className="muted">بدون شاهد مالی؛ از نگهدار فعلی مشتق نمی‌شود.</p>
      </section>
    </div>
  )
}

function Timeline({ c }: { c: CaseCustody }) {
  return (
    <>
      {c.events.length === 0 ? <StateBlock kind="empty" title="رویداد نگهداری ثبت نشده است" /> : (
        <ol className="sa-list" aria-label="رویدادهای نگهداری">
          {c.events.map(e => (
            <li key={e.event_id}>
              <span className="sa-ltr">{e.legacy_action}</span> — {eventKindLabel[e.kind] ?? e.kind}
              <span className="muted"> · {text(e.occurred_raw)} · </span>
              <UserRefText user={e.from} /> → <UserRefText user={e.to} /> <span className="muted">(اقدام‌کننده: </span><UserRefText user={e.actor} /><span className="muted">)</span>
            </li>
          ))}
        </ol>
      )}
      {c.findings.length > 0 && (
        <>
          <h3 className="sa-sub">یافته‌ها (خودکار اصلاح نمی‌شوند)</h3>
          <ul className="sa-list" aria-label="یافته‌های نگهداری">
            {c.findings.map(f => <li key={f.finding_id}><StatusPill tone="orange" icon="alert" label={findingLabel[f.kind] ?? f.kind} /> <span className="muted">{f.detail}</span></li>)}
          </ul>
        </>
      )}
    </>
  )
}
