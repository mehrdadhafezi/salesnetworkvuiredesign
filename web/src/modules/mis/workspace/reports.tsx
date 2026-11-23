import { useApi, type CustodyParityReport, type SalesCoverageReport } from '../../../shared/api'
import { settle, useRead, type ReadState } from '../../../shared/read/load'
import type { CountSource } from '../labels'

/**
 * The two existing aggregate reads every MIS counter comes from
 * (both require sales.report.read and the sales.read_shadow flag):
 *   GET /api/v1/sales/coverage-report  → SalesCoverageReport
 *   GET /api/v1/sales/custody-parity   → CustodyParityReport
 * No counter is computed from a list page.
 */
export interface Reports {
  coverage: ReadState<SalesCoverageReport>
  parity: ReadState<CustodyParityReport>
}

export function useSalesReports(): Reports {
  const api = useApi()
  const coverage = useRead('mis-coverage', async () => settle(await api.GET('/api/v1/sales/coverage-report'), e => e.data))
  const parity = useRead('mis-parity', async () => settle(await api.GET('/api/v1/sales/custody-parity'), e => e.data))
  return { coverage, parity }
}

/** A counter's value, or null when its source is not readable or has no aggregate. */
export function countOf(src: CountSource, r: Reports): number | null {
  switch (src.kind) {
    case 'parity':
      // A parity class missing from current_holder_parity has no present projection: the map is a full GROUP BY, so 0 is real.
      return r.parity.status === 'ok' ? r.parity.data.current_holder_parity[src.key] ?? 0 : null
    case 'finding':
      return r.parity.status === 'ok' ? r.parity.data.findings_by_kind[src.key] ?? 0 : null
    case 'coverage':
      return r.coverage.status === 'ok' ? r.coverage.data[src.field] : null
    case 'coverage-absent':
      return r.coverage.status === 'ok' ? r.coverage.data.cases_total - r.coverage.data.cases_present : null
    case 'coverage-map':
      return r.coverage.status === 'ok' ? r.coverage.data[src.map][src.key] ?? 0 : null
    case 'none':
      return null
  }
}

/** The documented data source of a counter (shown next to it). */
export function sourceOf(src: CountSource): string {
  switch (src.kind) {
    case 'parity': return `GET /api/v1/sales/custody-parity · current_holder_parity.${src.key}`
    case 'finding': return `GET /api/v1/sales/custody-parity · findings_by_kind.${src.key}`
    case 'coverage': return `GET /api/v1/sales/coverage-report · ${src.field}`
    case 'coverage-absent': return 'GET /api/v1/sales/coverage-report · cases_total − cases_present'
    case 'coverage-map': return `GET /api/v1/sales/coverage-report · ${src.map}.${src.key}`
    case 'none': return 'بدون منبع تجمیعی'
  }
}

/** Why a counter has no value right now. */
export function unavailableWhy(src: CountSource, r: Reports): string {
  if (src.kind === 'none') return src.why
  const st = src.kind === 'parity' || src.kind === 'finding' ? r.parity : r.coverage
  switch (st.status) {
    case 'loading': return 'در حال دریافت'
    case 'denied': return 'دسترسی به گزارش منبع ندارید (sales.report.read)'
    case 'blocked': return 'گزارش منبع غیرفعال است (sales.read_shadow)'
    case 'failed': return 'خواندن گزارش منبع ناموفق بود'
    case 'ok': return ''
  }
}
