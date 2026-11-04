import '../operations.css'
import { useSearchParams } from 'react-router'
import { useApi } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ShadowNotice } from '../CommandResult'
import { blockedDetail, decisionLabel, decisionTone, linkLabel, routeLabel, short } from '../labels'

const decisions = ['SHADOW_ACTIVATED', 'BLOCKED', 'NOT_APPLICABLE'] as const

/**
 * Activation decisions recorded from fully-paid invoice events (SHADOW
 * only, D-35). Complaint and archive branches never enter Biawin; unknown
 * lineage or route stays BLOCKED with its reason. Read only.
 */
export function ActivationDecisionsPage() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [params, setParams] = useSearchParams()
  const decision = params.get('decision') ?? undefined
  const st = useRead(`op-act:${decision ?? ''}`, async () => settle(await api.GET('/api/v1/biawin/activation-decisions', {
    params: { query: compact({ decision, limit: 100 }) as never } }), e => e.items))
  return (
    <>
      <PageHeader title="تصمیم‌های فعال‌سازی اشتراک" subtitle="فقط فعال‌سازی سایه؛ هیچ عضویت، اعتبار یا اثر مالی واقعی ایجاد نمی‌شود (D-35).">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowNotice />
      <Card title="فیلتر">
        <label className="lbl">تصمیم
          <select className="input" value={decision ?? ''} onChange={e => { const n = new URLSearchParams(params); if (e.target.value) n.set('decision', e.target.value); else n.delete('decision'); setParams(n) }}>
            <option value="">همه</option>
            {decisions.map(d => <option key={d} value={d}>{decisionLabel[d]}</option>)}
          </select>
        </label>
      </Card>
      <ReadStates state={st} what="تصمیم‌ها" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <Card title="تصمیم‌ها">
          {st.data.length === 0 ? <StateBlock kind="empty" title="هنوز تصمیمی ثبت نشده است" /> : (
            <div className="op-table-wrap">
              <table className="op-table">
                <caption className="sr">تصمیم‌های فعال‌سازی</caption>
                <thead><tr>
                  <th scope="col">محصول</th><th scope="col">شاخه مشتری</th><th scope="col">پیوند مشتری</th><th scope="col">تصمیم</th><th scope="col">دلیل / تصمیم باز</th><th scope="col">زمان</th>
                </tr></thead>
                <tbody>
                  {st.data.map(d => (
                    <tr key={d.activation_id}>
                      <td className="op-ltr">{`${d.product_key} v${d.product_version}`}<br /><span className="muted">{d.invoice_ref ?? short(d.invoice_id)}</span></td>
                      <td>{routeLabel[d.route] ?? d.route}</td>
                      <td>{linkLabel[d.customer_link_state] ?? d.customer_link_state}{d.existing_customer ? ' · مشتری موجود' : ''}</td>
                      <td><StatusPill tone={decisionTone[d.decision] ?? 'slate'} icon={d.decision === 'BLOCKED' ? 'ban' : 'check'} label={decisionLabel[d.decision] ?? d.decision} />
                        {d.decision === 'SHADOW_ACTIVATED' && <><br /><span className="muted">{`${d.item_count} آیتم سایه`}</span></>}</td>
                      <td className="op-ltr">{[...d.reasons, ...d.blocked_by].join(', ') || '—'}</td>
                      <td>{formatInstant(d.decided_at, zone)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      )}
    </>
  )
}
