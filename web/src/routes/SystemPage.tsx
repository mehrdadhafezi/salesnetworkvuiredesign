import { useEffect, useState } from 'react'
import { toFailure, useApi, type ApiFailure, type HealthLive, type HealthReady } from '../shared/api'
import { useSession } from '../shared/auth/currentSession'
import { outcomePresentation } from '../shared/ui/outcome'
import { Card, PageHeader, StateBlock, StatusPill } from '../shared/ui/primitives'
import { formatInstant } from '../shared/ui/time'

type Loaded<T> = { status: 'loading' } | { status: 'ok'; value: T } | { status: 'failed'; failure: ApiFailure }

/** Technical status only: process health and the session surface. */
export function SystemPage() {
  const api = useApi()
  const session = useSession()
  const [live, setLive] = useState<Loaded<HealthLive>>({ status: 'loading' })
  const [ready, setReady] = useState<Loaded<HealthReady>>({ status: 'loading' })

  useEffect(() => {
    let active = true
    const settle = <T,>(set: (v: Loaded<T>) => void) =>
      ({ data, error, response }: { data?: T | undefined; error?: unknown; response: Response }) => {
        if (!active) return
        set(data ? { status: 'ok', value: data } : { status: 'failed', failure: toFailure(error, response) })
      }
    const fail = <T,>(set: (v: Loaded<T>) => void) => (err: unknown) => {
      if (active) set({ status: 'failed', failure: toFailure(err, undefined) })
    }
    api.GET('/health/live').then(settle(setLive), fail(setLive))
    api.GET('/health/ready').then(settle(setReady), fail(setReady))
    return () => { active = false }
  }, [api])

  return (
    <>
      <PageHeader title="وضعیت سامانه" subtitle="وضعیت فنی سرویس؛ نتیجه کسب‌وکار نیست." />
      <div className="grid">
        <Card title="زنده بودن سرویس">
          <HealthValue state={live} ok={v => v.status === 'live'} okLabel="فعال" />
        </Card>
        <Card title="آمادگی سرویس">
          <HealthValue state={ready} ok={v => v.status === 'ready'} okLabel="آماده" />
        </Card>
        <Card title="نشست">
          {session.status === 'loading' && <StateBlock kind="loading" title="در حال دریافت نشست…" />}
          {session.status === 'failed' && <FailureState failure={session.failure} />}
          {session.status === 'ready' && (
            <dl className="kv">
              <dt>حالت احراز هویت</dt>
              <dd>{session.session.auth_mode === 'dev_stub' ? 'شبیه‌ساز توسعه (ساختگی)' : 'غیرفعال'}</dd>
              <dt>هویت</dt>
              <dd>{session.session.principal ? `${session.session.principal.display_name} (ساختگی)` : '—'}</dd>
              <dt>زمان سرور</dt>
              <dd>{formatInstant(session.session.server_time, session.session.display_zone)}</dd>
            </dl>
          )}
        </Card>
      </div>
    </>
  )
}

function HealthValue<T>({ state, ok, okLabel }: { state: Loaded<T>; ok: (v: T) => boolean; okLabel: string }) {
  if (state.status === 'loading') return <StateBlock kind="loading" title="در حال بررسی…" />
  if (state.status === 'failed') return <FailureState failure={state.failure} />
  return ok(state.value)
    ? <StatusPill tone="green" icon="check" label={okLabel} />
    : <StatusPill tone="amber" icon="alert" label="آماده نیست" />
}

function FailureState({ failure }: { failure: ApiFailure }) {
  if (failure.kind === 'problem') {
    return <StateBlock kind="error" title={failure.problem.title}
      detail={`کد: ${failure.problem.reason_code} · شناسه پیگیری: ${failure.problem.correlation_id}`} />
  }
  const unknown = outcomePresentation.UNKNOWN
  return <StateBlock kind="unknown" title="ارتباط با سرویس برقرار نشد" detail={unknown.label} />
}
