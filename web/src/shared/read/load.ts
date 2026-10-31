import { useEffect, useState } from 'react'
import { toFailure, type AllowedActions, type ApiFailure, type Outcome, type ReasonCode } from '../api'

/**
 * The state of one server read. A read never becomes "empty" on failure:
 * transport/problem failures, UNAUTHORIZED and BLOCKED (feature disabled)
 * are distinct states, and the server's allowed_actions travel with data.
 */
export type ReadState<T> =
  | { status: 'loading' }
  | { status: 'ok'; data: T; outcome: Outcome; allowed: AllowedActions | undefined }
  | { status: 'denied'; reasons: ReasonCode[] }
  | { status: 'blocked'; reasons: ReasonCode[] }
  | { status: 'failed'; failure: ApiFailure }

interface Envelope { outcome: Outcome; reasons: ReasonCode[]; allowed_actions?: AllowedActions }

type Fetched<E> = { data?: E | undefined; error?: unknown; response: Response }

/** Settles a generated-client call into a ReadState. */
export function settle<E extends Envelope, T>(r: Fetched<E>, pick: (e: E) => T | undefined): ReadState<T> {
  if (r.data) {
    const env = r.data
    if (env.outcome === 'BLOCKED') return { status: 'blocked', reasons: env.reasons }
    if (env.outcome === 'UNAUTHORIZED') return { status: 'denied', reasons: env.reasons }
    const data = pick(env)
    if (data === undefined) return { status: 'failed', failure: { kind: 'transport', message: 'response without data' } }
    return { status: 'ok', data, outcome: env.outcome, allowed: env.allowed_actions }
  }
  if (r.response.status === 403 && r.error && typeof r.error === 'object' && 'outcome' in r.error) {
    return { status: 'denied', reasons: (r.error as Envelope).reasons }
  }
  return { status: 'failed', failure: toFailure(r.error, r.response) }
}

/** Runs one read whenever key changes; stale responses are ignored. */
export function useRead<T>(key: string, run: () => Promise<ReadState<T>>): ReadState<T> {
  const [state, setState] = useState<{ key: string; value: ReadState<T> }>({ key, value: { status: 'loading' } })
  useEffect(() => {
    let live = true
    run().then(v => { if (live) setState({ key, value: v }) },
      (err: unknown) => { if (live) setState({ key, value: { status: 'failed', failure: toFailure(err, undefined) } }) })
    return () => { live = false }
    // run is recreated every render; key captures every input that matters.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key])
  return state.key === key ? state.value : { status: 'loading' }
}

type Defined<T> = { [K in keyof T]?: Exclude<T[K], undefined> }

/** Drops undefined values so optional query parameters are simply absent. */
export function compact<T extends Record<string, unknown>>(o: T): Defined<T> {
  return Object.fromEntries(Object.entries(o).filter(([, v]) => v !== undefined)) as Defined<T>
}
