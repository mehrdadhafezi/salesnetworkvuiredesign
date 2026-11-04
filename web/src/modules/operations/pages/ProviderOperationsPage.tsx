import '../operations.css'
import { useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router'
import { useApi, type components } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { reasonText } from '../../writes/labels'
import { newKey } from '../CommandResult'
import { blockedDetail, opStateLabel, opStateTone, opTypeLabel, short } from '../labels'

type Op = components['schemas']['ProviderOperation']
const states = ['PREPARED', 'DISPATCHING', 'ACKNOWLEDGED', 'CONFIRMED', 'REJECTED', 'UNKNOWN', 'RECONCILIATION_REQUIRED', 'CONFIRMED_ABSENT', 'BLOCKED'] as const
const resolvable = new Set(['UNKNOWN', 'RECONCILIATION_REQUIRED'])

/**
 * External provider operations (write-ahead ledger). The connector is not
 * configured; nothing here dispatches or retries. UNKNOWN is never shown
 * as delivered; only a human with evidence may resolve it.
 */
export function ProviderOperationsPage() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [params, setParams] = useSearchParams()
  const state = params.get('state') ?? undefined
  const [reload, setReload] = useState(0)
  const st = useRead(`op-prov:${state ?? ''}:${reload}`, async () => settle(await api.GET('/api/v1/integration/provider-operations', {
    params: { query: compact({ state, limit: 100 }) as never } }), e => e.items ? { items: e.items, connector: e.connector } : undefined))
  const [target, setTarget] = useState<{ op: Op; present: boolean; note: string; ext: string; key: string } | null>(null)
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null)

  async function resolve() {
    if (!target) return
    const r = await api.POST('/api/v1/integration/provider-operations/{operation_id}/resolution', {
      params: { path: { operation_id: target.op.operation_id }, header: { 'Idempotency-Key': target.key, 'If-Match': String(target.op.version) } },
      body: { present: target.present, evidence_note: target.note, ...(target.ext ? { external_ref: target.ext } : {}) } })
    const env = (r.data ?? r.error) as { outcome?: string; reasons?: string[] } | undefined
    if (env?.outcome === 'SUCCESS') {
      setMessage({ ok: true, text: 'تطبیق انسانی ثبت شد.' })
      setTarget(null)
      setReload(n => n + 1)
    } else {
      setMessage({ ok: false, text: `تطبیق ثبت نشد ${env?.outcome ?? ''} ${reasonText(env?.reasons ?? [])}` })
      if (env?.outcome === 'CONFLICT') { setTarget(null); setReload(n => n + 1) }
    }
  }

  return (
    <>
      <PageHeader title="عملیات ارائه‌دهنده خارجی" subtitle="دفتر پیش‌ثبت عملیات بیاوین؛ هیچ ارسال یا تلاش مجددی از این صفحه انجام نمی‌شود.">
        <StatusPill tone="slate" icon="lock" label="اتصال پیکربندی نشده" />
      </PageHeader>
      <div className="op-banner" role="note">
        <strong>اتصال بیاوین: پیکربندی نشده (NOT_CONFIGURED)</strong>
        <span>نتیجه نامعلوم هرگز به‌عنوان تحویل موفق نمایش داده نمی‌شود. شارژ کیف پول تا تصمیم D-38 مسدود است.</span>
      </div>
      <Card title="فیلتر">
        <label className="lbl">وضعیت
          <select className="input" value={state ?? ''} onChange={e => { const n = new URLSearchParams(params); if (e.target.value) n.set('state', e.target.value); else n.delete('state'); setParams(n) }}>
            <option value="">همه</option>
            {states.map(x => <option key={x} value={x}>{opStateLabel[x]}</option>)}
          </select>
        </label>
      </Card>
      <ReadStates state={st} what="عملیات" blockedDetail={blockedDetail} />
      {message && <StateBlock kind={message.ok ? 'empty' : 'error'} title={message.text} />}
      {st.status === 'ok' && (
        <Card title="عملیات">
          {st.data.items.length === 0 ? <StateBlock kind="empty" title="هیچ عملیاتی ثبت نشده است" /> : (
            <div className="op-table-wrap">
              <table className="op-table">
                <caption className="sr">عملیات ارائه‌دهنده</caption>
                <thead><tr>
                  <th scope="col">نوع</th><th scope="col">مرجع</th><th scope="col">وضعیت</th><th scope="col">تلاش‌ها</th><th scope="col">مسئول تطبیق</th><th scope="col">آخرین تغییر</th><th scope="col">اقدام</th>
                </tr></thead>
                <tbody>
                  {st.data.items.map(o => (
                    <tr key={o.operation_id}>
                      <td>{opTypeLabel[o.op_type] ?? o.op_type}</td>
                      <td className="op-ltr">{o.reference}{o.external_ref ? <><br />{o.external_ref}</> : null}</td>
                      <td><StatusPill tone={opStateTone[o.state] ?? 'slate'} icon={o.state === 'CONFIRMED' ? 'check' : o.state === 'BLOCKED' ? 'ban' : 'question'} label={opStateLabel[o.state] ?? o.state} />
                        {o.blocked_by && <><br /><span className="muted">{o.blocked_by}</span></>}</td>
                      <td className="op-ltr">{o.attempts}</td>
                      <td>{o.reconciliation_owner}</td>
                      <td>{formatInstant(o.updated_at, zone)}</td>
                      <td>{resolvable.has(o.state) && actionHint(st.allowed, 'integration.provider_operation.reconcile').allowed
                        ? <Button small onClick={() => setTarget({ op: o, present: false, note: '', ext: '', key: newKey() })}>{`تطبیق انسانی ${short(o.operation_id)}`}</Button>
                        : <span className="muted">—</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      )}
      {target && (
        <Card title="تطبیق انسانی با شواهد">
          <form className="op-form" aria-label="تطبیق انسانی" onSubmit={(e: FormEvent<HTMLFormElement>) => { e.preventDefault(); void resolve() }}>
            <p className="muted op-ltr">{target.op.reference}</p>
            <label className="lbl">یافته
              <select className="input" value={target.present ? 'present' : 'absent'} onChange={e => setTarget({ ...target, present: e.target.value === 'present' })}>
                <option value="absent">سفارش نزد ارائه‌دهنده وجود ندارد</option>
                <option value="present">سفارش نزد ارائه‌دهنده وجود دارد</option>
              </select>
            </label>
            <label className="lbl">شرح شواهد (الزامی)<textarea className="input" required minLength={10} value={target.note} onChange={e => setTarget({ ...target, note: e.target.value })} /></label>
            <label className="lbl">شناسه خارجی (در صورت وجود)<input className="input op-ltr" value={target.ext} onChange={e => setTarget({ ...target, ext: e.target.value })} /></label>
            <div className="op-actions">
              <Button type="submit" variant="primary">ثبت تطبیق</Button>
              <Button onClick={() => setTarget(null)}>انصراف</Button>
            </div>
          </form>
        </Card>
      )}
    </>
  )
}
