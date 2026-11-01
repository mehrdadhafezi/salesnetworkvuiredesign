import '../reporting.css'
import { useApi, type SourceQualityRow } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { ReadOnlyNote } from '../components'
import { blockedDetail, coverageLabel, coverageTone, sourceLabel } from '../labels'

/** null = not applicable for this grain (never zero); a missing source has no counts. */
function cell(v: number | null | undefined, cov: string) {
  if (cov !== 'AVAILABLE' && cov !== 'PARTIAL') return <span className="muted">نامعلوم</span>
  if (v === null || v === undefined) return <span className="muted">قابل اعمال نیست</span>
  return <span className="num">{v}</span>
}

/**
 * Source quality per published source at an explicit grain. Duplicate
 * candidates share only a phone and are never merged into one customer.
 */
export function SourceQualityPage() {
  const api = useApi()
  const st = useRead('mis-sq', async () => settle(await api.GET('/api/v1/mis/source-quality'), e => e.data))
  return (
    <>
      <PageHeader title="کیفیت منبع" subtitle="معتبر / تکراری احتمالی / نامعتبر / نامعلوم / تعارض، با دانه صریح.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <ReadStates state={st} what="کیفیت منبع" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <Card title="منابع">
          <div className="rp-table-wrap">
            <table className="rp-table">
              <caption className="sr">کیفیت منبع</caption>
              <thead>
                <tr><th scope="col">منبع</th><th scope="col">دانه</th><th scope="col">پوشش</th><th scope="col">معتبر</th><th scope="col">تکراری احتمالی</th>
                  <th scope="col">نامعتبر</th><th scope="col">نامعلوم</th><th scope="col">تعارض</th><th scope="col">توضیح</th></tr>
              </thead>
              <tbody>
                {st.data.rows.map((r: SourceQualityRow) => (
                  <tr key={r.source}>
                    <td>{sourceLabel[r.source] ?? r.source}</td>
                    <td>{r.grain}</td>
                    <td><StatusPill tone={coverageTone[r.coverage] ?? 'slate'} icon={r.coverage === 'AVAILABLE' ? 'check' : 'alert'} label={coverageLabel[r.coverage] ?? r.coverage} /></td>
                    <td>{cell(r.valid, r.coverage)}</td>
                    <td>{cell(r.duplicate_candidates, r.coverage)}</td>
                    <td>{cell(r.invalid, r.coverage)}</td>
                    <td>{cell(r.unknown, r.coverage)}</td>
                    <td>{cell(r.conflict, r.coverage)}</td>
                    <td className="muted">{r.note}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </>
  )
}
