import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { onAuthDenied, setCsrfToken, toFailure, useApi, type ApiFailure, type SessionContext } from '../api'

/**
 * Session context. In R1 session mode it carries the authenticated identity
 * and the CSRF token (kept in memory only); it is still not a security
 * boundary: the server authorises every request and holds the session.
 */
export type SessionState =
  | { status: 'loading' }
  | { status: 'ready'; session: SessionContext }
  | { status: 'failed'; failure: ApiFailure }

export type LoginResult =
  | { kind: 'ok' }
  | { kind: 'invalid' }
  | { kind: 'throttled' }
  | { kind: 'disabled' }
  | { kind: 'rejected' }
  | { kind: 'failed' }

export interface SessionActions {
  refresh: () => void
  login: (login: string, password: string) => Promise<LoginResult>
  logout: () => Promise<void>
}

const SessionCtx = createContext<SessionState>({ status: 'loading' })
const ActionsCtx = createContext<SessionActions>({
  refresh: () => {},
  login: () => Promise.resolve({ kind: 'failed' }),
  logout: () => Promise.resolve(),
})

export function SessionProvider({ children }: { children: ReactNode }) {
  const api = useApi()
  const [state, setState] = useState<SessionState>({ status: 'loading' })
  const [nonce, setNonce] = useState(0)
  const refresh = useCallback(() => setNonce(n => n + 1), [])

  useEffect(() => {
    let live = true
    api.GET('/api/v1/session').then(({ data, error, response }) => {
      if (!live) return
      if (data) {
        setCsrfToken(api, data.data.session?.csrf_token ?? null)
        setState({ status: 'ready', session: data.data })
      } else {
        setState({ status: 'failed', failure: toFailure(error, response) })
      }
    }, (err: unknown) => {
      if (live) setState({ status: 'failed', failure: toFailure(err, undefined) })
    })
    return () => { live = false }
  }, [api, nonce])

  // A denied request may mean the session ended (expired, revoked,
  // logged out elsewhere): re-check, never assume.
  const authenticated = state.status === 'ready' && state.session.authenticated && state.session.auth_mode === 'session'
  useEffect(() => {
    onAuthDenied(api, authenticated ? refresh : null)
    return () => onAuthDenied(api, null)
  }, [api, authenticated, refresh])

  // Re-check exactly when the idle window ends (no polling: polling would
  // keep an idle session alive).
  const idleAt = state.status === 'ready' ? state.session.session?.idle_expires_at : undefined
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current)
    if (!idleAt) return
    const ms = Math.min(Math.max(new Date(idleAt).getTime() - Date.now() + 1000, 1000), 2 ** 31 - 1)
    timer.current = setTimeout(refresh, ms)
    return () => { if (timer.current) clearTimeout(timer.current) }
  }, [idleAt, refresh])

  const actions = useMemo<SessionActions>(() => ({
    refresh,
    async login(login, password) {
      try {
        const { data, error, response } = await api.POST('/api/v1/auth/login', { body: { login, password } })
        if (data?.outcome === 'SUCCESS') {
          setCsrfToken(api, data.session?.csrf_token ?? null)
          refresh()
          return { kind: 'ok' }
        }
        if (data?.outcome === 'BLOCKED') return { kind: 'disabled' }
        if (response.status === 401) return { kind: 'invalid' }
        if (response.status === 429) return { kind: 'throttled' }
        if (response.status === 403) return { kind: 'rejected' }
        void error
        return { kind: 'failed' }
      } catch {
        return { kind: 'failed' }
      }
    },
    async logout() {
      try {
        await api.POST('/api/v1/auth/logout')
      } finally {
        setCsrfToken(api, null)
        refresh()
      }
    },
  }), [api, refresh])

  return (
    <ActionsCtx.Provider value={actions}>
      <SessionCtx.Provider value={state}>{children}</SessionCtx.Provider>
    </ActionsCtx.Provider>
  )
}

export function useSession(): SessionState {
  return useContext(SessionCtx)
}

export function useSessionActions(): SessionActions {
  return useContext(ActionsCtx)
}
