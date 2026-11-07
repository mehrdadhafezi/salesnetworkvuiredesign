// The single network adapter of the SPA. Every request goes through the
// generated, contract-typed openapi-fetch client: base URL, correlation id
// propagation, the R1 session-bound CSRF header and (DEV/TEST only) the
// synthetic identity header live here.
import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './types'

export const CORRELATION_HEADER = 'X-Correlation-ID'
export const CSRF_HEADER = 'X-CSRF-Token'
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

/**
 * Per-client R1 session state kept in memory only (never storage): the
 * session-bound CSRF token sent on unsafe requests, and a listener told
 * when the server denies a request so the session can be re-checked.
 * The session cookie itself is HttpOnly and never visible to script.
 */
interface AuthState { csrf: string | null; onDenied: (() => void) | null }
const authState = new WeakMap<ApiClient, AuthState>()

function state(client: ApiClient): AuthState {
  let s = authState.get(client)
  if (!s) {
    s = { csrf: null, onDenied: null }
    authState.set(client, s)
  }
  return s
}

/** Sets (or clears) the CSRF token for this client's unsafe requests. */
export function setCsrfToken(client: ApiClient, token: string | null) {
  state(client).csrf = token
}

/** Registers the callback run after a 401/403 response (null clears it). */
export function onAuthDenied(client: ApiClient, fn: (() => void) | null) {
  state(client).onDenied = fn
}

const SAFE = new Set(['GET', 'HEAD', 'OPTIONS'])

export function createApi(options: ApiOptions): ApiClient {
  const client = createClient<paths>({
    baseUrl: options.baseUrl,
    // Same-origin only: the session cookie never goes to another origin.
    credentials: 'same-origin',
    ...(options.fetch ? { fetch: options.fetch } : {}),
  })
  const newId = options.newCorrelationId ?? (() => crypto.randomUUID())
  const middleware: Middleware = {
    onRequest({ request }) {
      if (!request.headers.has(CORRELATION_HEADER)) {
        request.headers.set(CORRELATION_HEADER, newId())
      }
      const csrf = authState.get(client)?.csrf
      if (csrf && !SAFE.has(request.method.toUpperCase())) {
        request.headers.set(CSRF_HEADER, csrf)
      }
      // Build-time guard: production bundles never contain dev identity code.
      if (import.meta.env.DEV && options.devUser) {
        request.headers.set(DEV_USER_HEADER, options.devUser)
      }
      return request
    },
    onResponse({ request, response }) {
      if ((response.status === 401 || response.status === 403) && !new URL(request.url).pathname.startsWith('/api/v1/auth/')) {
        authState.get(client)?.onDenied?.()
      }
      return response
    },
  }
  client.use(middleware)
  return client
}
