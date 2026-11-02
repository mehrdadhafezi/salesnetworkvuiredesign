import '../biawin.css'
import { Link, useParams } from 'react-router'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { RawTime, RunBar, ShadowBanner } from '../components'
import { blockedDetail, caseStatusLabel, caseStatusTone, evidenceLabel, kindLabel, roleLabel, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

/**
 * One shadow fulfillment case of one run: assignees, evidence metadata and
 * append-only history. External references are digests only.
 */
export function CasePage() {
  const api = useApi()
  const { caseId = '' } = useParams()
  const ctx = useQueryContext([])
  const run = ctx.runId
  const st = useRead(`fw-case:${caseId}:${run ?? ''}`, async () => settle(await api.GET('/api/v1/fulfillment/cases/{case_id}', {
    params: { path: { case_id: caseId }, query: compact({ run_id: run }) as never },
  }), e => e.meta && e.case ? { meta: e.meta, c: e.case, evidence: e.evidence ?? [] } : undefined))
  const pinnedRun = st.status === 'ok' ? st.data.meta.query.run_id : run
  // Waits for the detail only when no run is pinned yet (the key then
  // changes once the detail names its run); a pinned run loads at once.
  const hist = useRead(`fw-hist:${caseId}:${pinnedRun ?? ''}`, async () => pinnedRun === undefined
    ? { status: 'loading' as const }
    : settle(await api.GET('/api/v1/fulfillment/cases/{case_id}/history', {
      params: { path: { case_id: caseId }, query: compact({ run_id: pinnedRun, limit: 200 }) as never },
    }), e => e.items ? e.items : undefined))
  return (
    <>
      <PageHeader title="پرونده اجرای خدمت" subtitle="جزئیات، مسئولان فعلی، شاهد و تاریخچه از یک اجرای ورود.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <p><Link to={`/fulfillment/cases${pinnedRun ? `?run_id=${pinnedRun}` : ''}`}>بازگشت به فهرست</Link></p>
      <ReadStates state={st} what="پرونده" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <div className="bw-sections">
          <RunBar run={st.data.meta.run} />
          <Card title={`${kindLabel[st.data.c.kind] ?? st.data.c.kind} #${st.data.c.source_key}`}>
            <dl className="bw-dl">
              <dt>وضعیت</dt>
              <dd>
                <StatusPill tone={caseStatusTone[st.data.c.status_mapped] ?? 'slate'} icon={st.data.c.status_mapped === 'UNKNOWN' ? 'question' : 'check'}
                  label={caseStatusLabel[st.data.c.status_mapped] ?? st.data.c.status_mapped} />
                <span className="muted bw-ltr">{` raw: ${text(st.data.c.status_raw)} · ${st.data.c.status_namespace}`}</span>
              </dd>
              {st.data.c.status_mapped === 'COMPLETED' && <><dt>معنا</dt><dd>فقط همین پرونده تکمیل شده است؛ درباره فاکتور، پرداخت، اشتراک، فروش یا مالی چیزی نمی‌گوید.</dd></>}
              <dt>زیرنوع</dt><dd className="bw-ltr">{text(st.data.c.subtype_raw)}</dd>
              <dt>وضعیت ارسال / تأمین‌کننده / تماس</dt>
              <dd className="bw-ltr">{`${text(st.data.c.shipping_status_raw)} / ${text(st.data.c.provider_status_raw)} / ${text(st.data.c.contact_status_raw)}`}</dd>
              <dt>تکمیل</dt><dd><RawTime v={st.data.c.completed_at_raw} /></dd>
              <dt>ارسال / تحویل</dt><dd><RawTime v={st.data.c.shipped_at_raw} /> · <RawTime v={st.data.c.delivered_at_raw} /></dd>
              <dt>کد رهگیری</dt><dd>{st.data.c.tracking_ref_sha256 ? <span className="muted bw-ltr">{`sha256 ${st.data.c.tracking_ref_sha256.slice(0, 12)}…`}</span> : <span className="muted">ندارد</span>}</dd>
              <dt>مرجع تأمین‌کننده</dt><dd>{st.data.c.provider_ref_sha256 ? <span className="muted bw-ltr">{`${text(st.data.c.provider_ref_scheme)} sha256 ${st.data.c.provider_ref_sha256.slice(0, 12)}…`}</span> : <span className="muted">ندارد</span>}</dd>
              <dt>پیوند فاکتور</dt><dd>{st.data.c.invoice_link_state === 'RESOLVED' ? 'اثبات‌شده' : 'نامعلوم'} <span className="muted bw-ltr">{text(st.data.c.invoice_key)}</span></dd>
              {st.data.c.subscription_id && <><dt>اشتراک</dt><dd><Link to={`/biawin/subscriptions/${st.data.c.subscription_id}`}>مشاهده اشتراک</Link></dd></>}
            </dl>
          </Card>
          <Card title="مسئولان فعلی (نه مالک، نه ذی‌نفع پورسانت)">
            {st.data.c.assignments.length === 0 ? <StateBlock kind="empty" title="مسئولی ثبت نشده است" />
              : (
                <ul className="bw-list" aria-label="مسئولان">
                  {st.data.c.assignments.map(a => (
                    <li key={a.role}>{roleLabel[a.role] ?? a.role}: <span className="bw-ltr">{`#${a.user_key}`}</span>{a.user_id ? '' : ' (کاربر نامعلوم)'} · <RawTime v={a.assigned_at_raw} /></li>
                  ))}
                </ul>
              )}
          </Card>
          <Card title="شاهد خدمت (فقط فراداده)">
            {st.data.evidence.length === 0 ? <StateBlock kind="empty" title="شاهدی ثبت نشده است" />
              : (
                <ul className="bw-list" aria-label="شاهد خدمت">
                  {st.data.evidence.map(ev => (
                    <li key={ev.source_key}>{evidenceLabel[ev.outcome_mapped] ?? ev.outcome_mapped} <span className="muted bw-ltr">{`raw: ${text(ev.outcome_raw)}`}</span> · <RawTime v={ev.submitted_at_raw} /></li>
                  ))}
                </ul>
              )}
          </Card>
          <Card title="تاریخچه (فقط افزودنی)">
            <ReadStates state={hist} what="تاریخچه" blockedDetail={blockedDetail} />
            {hist.status === 'ok' && (hist.data.length === 0 ? <StateBlock kind="empty" title="رویدادی ثبت نشده است" />
              : (
                <ol className="bw-list" aria-label="تاریخچه پرونده">
                  {hist.data.map(e => (
                    <li key={`${e.source_spec}:${e.source_key}`}>
                      <span className="bw-ltr">{e.event_key_raw}</span> <span className="muted bw-ltr">{`${text(e.old_status_raw)} → ${text(e.new_status_raw)}`}</span> · <RawTime v={e.occurred_at_raw} />
                    </li>
                  ))}
                </ol>
              ))}
          </Card>
        </div>
      )}
    </>
  )
}
