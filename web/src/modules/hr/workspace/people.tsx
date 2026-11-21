import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useApi, type PersonSummary } from '../../../shared/api'
import { settle, type ReadState } from '../../../shared/read/load'
import { StatusPill } from '../../../shared/ui/primitives'
import { useCanResolveUsers } from './access'

interface Found { items: PersonSummary[]; more: boolean }
type Lookup = Promise<ReadState<Found>>

/**
 * Per-page cache of user → person lookups (one `GET /organization/persons?user_id=`
 * per distinct user). Scoped to the mounted page, never shared across sessions.
 */
const Cache = createContext<Map<string, Lookup> | null>(null)

export function PersonResolver({ children }: { children: ReactNode }) {
  const [cache] = useState(() => new Map<string, Lookup>())
  return <Cache.Provider value={cache}>{children}</Cache.Provider>
}

function useUserPersons(userId: string | null | undefined, enabled: boolean): ReadState<Found> | { status: 'off' } {
  const api = useApi()
  const cache = useContext(Cache)
  const [state, setState] = useState<{ key: string; value: ReadState<Found> } | null>(null)
  const key = enabled && userId ? userId : ''
  useEffect(() => {
    if (!key) return
    let live = true
    let p = cache?.get(key)
    if (!p) {
      p = api.GET('/api/v1/organization/persons', { params: { query: { user_id: key, limit: 5 } } })
        .then(r => settle(r, e => e.items ? { items: e.items, more: e.page?.has_more === true } : undefined))
      cache?.set(key, p)
    }
    p.then(v => { if (live) setState({ key, value: v }) }, () => { if (live) setState({ key, value: { status: 'failed', failure: { kind: 'transport', message: 'lookup failed' } } }) })
    return () => { live = false }
  }, [api, cache, key])
  if (!key) return { status: 'off' }
  return state?.key === key ? state.value : { status: 'loading' }
}

function RawId({ id }: { id: string }) {
  return <span className="hr-ltr num" title="شناسه کاربر (حساب)">{id}</span>
}

/**
 * Shows the person behind an identity user id when the caller may read the
 * employment user link; otherwise the raw user id. A user linked to several
 * persons is a conflict and is never collapsed into one name. The user id
 * (account) is never presented as the person itself.
 */
export function UserPerson({ userId, empty = '—' }: { userId: string | null | undefined; empty?: string }) {
  const canResolve = useCanResolveUsers()
  const st = useUserPersons(userId, canResolve)
  if (!userId) return <span className="muted">{empty}</span>
  if (st.status === 'off') return <RawId id={userId} />
  if (st.status === 'loading') return <span className="muted" aria-busy="true"><RawId id={userId} /></span>
  if (st.status !== 'ok') return <span><RawId id={userId} /> <span className="muted">(نام قابل نمایش نیست)</span></span>
  const people = st.data.items
  const only = people[0]
  if (people.length === 1 && only) return <Link to={`/hr/people/${only.person_id}`}>{only.display_name ?? 'بدون نام'}</Link>
  if (people.length === 0) return <span><RawId id={userId} /> <StatusPill tone="amber" icon="question" label="پرونده نیروی متصل یافت نشد" /></span>
  return (
    <span>
      <StatusPill tone="orange" icon="alert" label={`تعارض: حساب مشترک بین ${people.length}${st.data.more ? '+' : ''} پرونده`} />{' '}
      <Link to={`/hr?user_id=${encodeURIComponent(userId)}`}>مشاهده پرونده‌ها</Link>
    </span>
  )
}
