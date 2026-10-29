import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { toFailure, useApi, type ApiFailure, type SessionContext } from '../api'

/**
 * Session context for shell display only (who is shown in the header).
 * It is not a security boundary: the server authorises every request.
 */
export type SessionState =
  | { status: 'loading' }
  | { status: 'ready'; session: SessionContext }
  | { status: 'failed'; failure: ApiFailure }

const SessionCtx = createContext<SessionState>({ status: 'loading' })

export function SessionProvider({ children }: { children: ReactNode }) {
  const api = useApi()
  const [state, setState] = useState<SessionState>({ status: 'loading' })
  useEffect(() => {
    let live = true
    api.GET('/api/v1/session').then(({ data, error, response }) => {
      if (!live) return
      if (data) setState({ status: 'ready', session: data.data })
      else setState({ status: 'failed', failure: toFailure(error, response) })
    }, (err: unknown) => {
      if (live) setState({ status: 'failed', failure: toFailure(err, undefined) })
    })
    return () => { live = false }
  }, [api])
  return <SessionCtx.Provider value={state}>{children}</SessionCtx.Provider>
}

export function useSession(): SessionState {
  return useContext(SessionCtx)
}
