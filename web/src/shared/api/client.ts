// The single network adapter of the SPA. Every request goes through the
// generated, contract-typed openapi-fetch client: base URL, correlation id
// propagation and (DEV/TEST only) the synthetic identity header live here.
import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './types'

export const CORRELATION_HEADER = 'X-Correlation-ID'
export const DEV_USER_HEADER = import.meta.env.DEV ? 'X-Dev-User' : ''

export interface ApiOptions {
  baseUrl: string
  /** Synthetic dev identity; must be undefined in production builds. */
  devUser?: string | undefined
  /** Injected for tests; defaults to the platform fetch. */
  fetch?: typeof globalThis.fetch
  newCorrelationId?: () => string
}

export type ApiClient = ReturnType<typeof createClient<paths>>

export function createApi(options: ApiOptions): ApiClient {
  const client = createClient<paths>({
    baseUrl: options.baseUrl,
    ...(options.fetch ? { fetch: options.fetch } : {}),
  })
  const newId = options.newCorrelationId ?? (() => crypto.randomUUID())
  const middleware: Middleware = {
    onRequest({ request }) {
      if (!request.headers.has(CORRELATION_HEADER)) {
        request.headers.set(CORRELATION_HEADER, newId())
      }
      // Build-time guard: production bundles never contain dev identity code.
      if (import.meta.env.DEV && options.devUser) {
        request.headers.set(DEV_USER_HEADER, options.devUser)
      }
      return request
    },
  }
  client.use(middleware)
  return client
}
