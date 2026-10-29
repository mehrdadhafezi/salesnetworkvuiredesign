import { createApi, type ApiClient } from '../shared/api'

export interface Captured { url: string; method: string; headers: Headers }

type Route = (req: Request) => Response | Promise<Response>

export function json(body: unknown, status = 200, contentType = 'application/json'): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': contentType } })
}

export const session = (over: Record<string, unknown> = {}) => ({
  outcome: 'SUCCESS',
  reasons: [],
  correlation_id: 'test-corr-0001',
  data: { authenticated: false, auth_mode: 'none', server_time: '2026-10-29T10:00:00Z', display_zone: 'Asia/Tehran', ...over },
})

/** A generated client wired to an in-memory fake server (no network). */
export function fakeApi(routes: Record<string, Route>, devUser?: string): { client: ApiClient; calls: Captured[] } {
  const calls: Captured[] = []
  let n = 0
  const client = createApi({
    baseUrl: 'http://api.test',
    devUser,
    newCorrelationId: () => `test-corr-${String(++n).padStart(4, '0')}`,
    fetch: async (input: RequestInfo | URL) => {
      const req = input as Request
      const url = new URL(req.url)
      calls.push({ url: url.pathname, method: req.method, headers: req.headers })
      const route = routes[`${req.method} ${url.pathname}`]
      return route ? route(req) : json({ type: '/problems/not-found', title: 'Resource not found', status: 404, reason_code: 'CONTRACT_INVALID', correlation_id: 'x-not-found-01' }, 404, 'application/problem+json')
    },
  })
  return { client, calls }
}

export const healthy: Record<string, Route> = {
  'GET /health/live': () => json({ status: 'live' }),
  'GET /health/ready': () => json({ status: 'ready', checks: [] }),
  'GET /api/v1/session': () => json(session()),
}
