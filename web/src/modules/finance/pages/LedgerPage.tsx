import '../finance.css'
import { useApi, type LedgerEntry } from '../../../shared/api'
import { compact, settle, useRead, type ReadState } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { AmountCell, MoneyCell, Pager, RunBar, ShadowBanner } from '../components'
import { Filters } from '../Filters'
import { basisLabel, blockedDetail, categoryLabel, entryStatusLabel, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

const options = {
  basis: ['ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW'],
  unit: ['IRT', 'IRR'],
  state: ['CONSISTENT', 'CONFLICT', 'UNRESOLVED_CORRECTION'],
  source_type: ['billing.invoice', 'payments.attempt', 'legacy.wallet_transaction'],
  account_category: ['customer_receivable', 'revenue_by_product_kind', 'cash_bank_clearing', 'commission_payable', 'legacy_wallet_unclassified', 'legacy_wallet_contra'],
} as const

const timeLabel: Record<string, string> = {
  BUSINESS_TIME_ABSENT: 'زمان مالی ثبت نشده (نامعلوم)', SOURCE_TIME_ZONE_UNASSERTED: 'زمان منبع بدون منطقه زمانی اثبات‌شده', SOURCE_TIME_UTC: 'زمان منبع UTC',
}

/**
 * SHADOW journal (read only): balanced, single-unit entries with their
 * lines, business key, basis, provenance, source references and reversal
 * relation. No edit, post, reverse or approve control exists. Detail is
 * read from the same run as the list.
 */
export function LedgerPage() {
  const api = useApi()
  const ctx = useQueryContext(Object.keys(options))
  const entry = ctx.params.get('entry') ?? undefined
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 25 }
  const st = useRead(`fn-led:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/finance/ledger', { params: { query: compact(q) as never } }),
    e => e.meta && e.items && e.page ? { meta: e.meta, items: e.items, page: e.page, summary: e.summary ?? [] } : undefined))
  const det = useRead(`fn-ent:${entry ?? ''}:${ctx.runId ?? ''}`, async (): Promise<ReadState<LedgerEntry | null>> => entry
    ? settle(await api.GET('/api/v1/finance/ledger/{entry_id}', { params: { path: { entry_id: entry }, query: compact({ run_id: ctx.runId }) } }),
      e => e.entry ? e.entry : undefined)
    : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined })
  return (
    <>
      <PageHeader title="دفتر روزنامه سایه" subtitle="ثبت‌های متوازن تک‌واحدی با سطرها، کلید کسب‌وکار و ارجاع منبع.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Filters options={options} values={ctx.filters} onApply={ctx.setFilters} label="فیلتر دفتر" />
      <ReadStates state={st} what="دفتر" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <div className="fn-sections">
          <RunBar run={st.data.meta.run} />
          <Card title="خلاصه (کل عضویت؛ به تفکیک مبنا و واحد)">
            <ul className="fn-list" aria-label="خلاصه دفتر">
              {st.data.summary.map(g => (
                <li key={`${g.basis}|${g.unit}`}>{basisLabel[g.basis] ?? g.basis} · <span className="fn-ltr">{g.unit}</span> · {g.entries} ثبت · بدهکار <AmountCell a={g.debits} />
                  {g.unknown_effective_at ? <span className="muted">{` · ${g.unknown_effective_at} بدون زمان مالی`}</span> : null}</li>
              ))}
            </ul>
          </Card>
          <Card title="ثبت‌ها">
            {st.data.items.length === 0
              ? <StateBlock kind="empty" title="هیچ ثبتی با این فیلتر در این اجرا نیست" />
              : (
                <div className="fn-table-wrap">
                  <table className="fn-table">
                    <caption className="sr">ثبت‌های سایه</caption>
                    <thead><tr>
                      <th scope="col">کلید کسب‌وکار</th><th scope="col">مبنا</th><th scope="col">واحد</th><th scope="col">بدهکار / بستانکار</th>
                      <th scope="col">منشأ</th><th scope="col">زمان</th><th scope="col">برگشت</th><th scope="col">وضعیت</th><th scope="col">جزئیات</th>
                    </tr></thead>
                    <tbody>
                      {st.data.items.map(e => (
                        <tr key={e.entry_id}>
                          <td className="fn-ltr">{e.business_key}</td>
                          <td>{basisLabel[e.basis] ?? e.basis}</td>
                          <td className="fn-ltr">{e.unit}</td>
                          <td><MoneyCell m={e.debits} /> / <MoneyCell m={e.credits} /></td>
                          <td className="fn-ltr">{e.provenance}</td>
                          <td>{timeLabel[e.time_evidence] ?? e.time_evidence}{e.source_time_raw ? <span className="muted fn-ltr">{` · ${e.source_time_raw}`}</span> : null}</td>
                          <td className="fn-ltr">{e.reverses_entry_id ? `↩ ${e.reverses_entry_id.slice(0, 8)}` : e.reversed_by_entry_id ? `↪ ${e.reversed_by_entry_id.slice(0, 8)}` : '—'}</td>
                          <td>{entryStatusLabel[e.reconciliation_status] ?? e.reconciliation_status}</td>
                          <td><Button onClick={() => ctx.pinned(st.data.meta.query.run_id, p => p.set('entry', e.entry_id))}>سطرها</Button></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <Pager page={st.data.page} depth={ctx.cursors.length} onBack={ctx.back}
              onNext={n => ctx.pinned(st.data.meta.query.run_id, p => p.append('cursor', n))} label="صفحه‌بندی دفتر" />
          </Card>
          {entry && (
            <Card title="سطرهای ثبت (همین اجرا)">
              <ReadStates state={det} what="ثبت" blockedDetail={blockedDetail} />
              {det.status === 'ok' && det.data && (
                <>
                  <p className="fn-ltr">{det.data.business_key}</p>
                  <p className="muted">ارجاع منبع: <span className="fn-ltr">{text(det.data.source_refs.map(r => `${r.role}=${r.domain}:${r.ref}`).join(' · '))}</span></p>
                  <ul className="fn-list" aria-label="سطرهای ثبت">
                    {det.data.lines.map(l => (
                      <li key={l.line_no}>{l.side === 'DEBIT' ? 'بدهکار' : 'بستانکار'} · <span className="fn-ltr">{l.account_key}</span>
                        {` (${categoryLabel[l.category] ?? l.category}) · `}<MoneyCell m={l.amount} /></li>
                    ))}
                  </ul>
                </>
              )}
            </Card>
          )}
        </div>
      )}
    </>
  )
}
