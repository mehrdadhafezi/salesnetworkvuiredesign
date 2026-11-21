import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useApi } from '../../shared/api'
import { useSession } from '../../shared/auth/currentSession'
import { resolveNavigation, type Evaluation, type NavAccess } from './access'

/**
 * Loads the server's effective-access evaluation for the authenticated
 * principal (one read-only GET; never a write) and resolves the navigation
 * with it. It re-evaluates whenever the session answer changes (login, logout,
 * expiry, revocation) and never caches across identities. Anonymous visitors
 * never trigger the call.
 */
const Ctx = createContext<NavAccess>({ status: 'LOADING' })
/** RM-P04: the same evaluation, for read-only access visibility (no second request). */
const EvalCtx = createContext<Evaluation>({ state: 'IDLE' })

export function NavigationAccessProvider({ children }: { children: ReactNode }) {
  const api = useApi()
  const session = useSession()
  const ready = session.status === 'ready' ? session.session : undefined
  const authenticated = ready?.authenticated === true
  // Keyed on the session answer object: a new session answer always re-evaluates.
  const [result, setResult] = useState<{ for: unknown; ev: Evaluation }>({ for: null, ev: { state: 'IDLE' } })

  useEffect(() => {
    if (!ready || !authenticated) return
    let live = true
    api.GET('/api/v1/authorization/effective-access').then(({ data }) => {
      if (live) setResult({ for: ready, ev: data && data.outcome === 'SUCCESS' ? { state: 'READY', data: data.data } : { state: 'FAILED' } })
    }, () => {
      if (live) setResult({ for: ready, ev: { state: 'FAILED' } })
    })
    return () => { live = false }
  }, [api, ready, authenticated])

  // An evaluation made for an earlier session answer is never reused.
  const evaluation = useMemo<Evaluation>(() => (!authenticated ? { state: 'IDLE' } : result.for === ready ? result.ev : { state: 'LOADING' }),
    [authenticated, ready, result])
  const access = useMemo(() => resolveNavigation(session, evaluation), [session, evaluation])
  return <Ctx.Provider value={access}><EvalCtx.Provider value={evaluation}>{children}</EvalCtx.Provider></Ctx.Provider>
}

/**
 * The server's effective-access evaluation for the CURRENT principal, exactly
 * as received (IDLE when anonymous). Presentation only: a page may hide a
 * section it would be refused, but every read is still authorised by the server.
 */
export function useEffectiveAccess(): Evaluation {
  return useContext(EvalCtx)
}

export function useNavigationAccess(): NavAccess {
  return useContext(Ctx)
}
