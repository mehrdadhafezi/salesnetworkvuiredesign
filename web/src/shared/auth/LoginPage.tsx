import { useState, type FormEvent } from 'react'
import { Button, Card, PageHeader, StateBlock } from '../ui/primitives'
import { useSessionActions, type LoginResult } from './currentSession'

const messages: Record<Exclude<LoginResult['kind'], 'ok'>, string> = {
  invalid: 'نام کاربری یا گذرواژه درست نیست.',
  throttled: 'تلاش‌های ناموفق زیاد بوده است. کمی بعد دوباره تلاش کنید.',
  disabled: 'ورود به سامانه در حال حاضر فعال نیست.',
  rejected: 'درخواست ورود پذیرفته نشد. صفحه را دوباره باز کنید.',
  failed: 'ارتباط با سرور برقرار نشد. دوباره تلاش کنید.',
}

/**
 * R1 password login (Persian RTL). The password lives only in the form
 * field and the single request; it is cleared after every attempt. There
 * is deliberately no "forgot password", sign-up or remember-me control:
 * recovery, enrolment and persistence policies are open decisions (D-34).
 */
export function LoginPage({ expired, enabled }: { expired: boolean; enabled: boolean }) {
  const { login } = useSessionActions()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = e.currentTarget
    const data = new FormData(form)
    const name = String(data.get('login') ?? '').trim()
    const password = String(data.get('password') ?? '')
    const pw = form.elements.namedItem('password')
    if (pw instanceof HTMLInputElement) pw.value = ''
    if (!name || !password) {
      setError('نام کاربری و گذرواژه را وارد کنید.')
      return
    }
    setBusy(true)
    setError(null)
    const r = await login(name, password)
    setBusy(false)
    if (r.kind !== 'ok') setError(messages[r.kind])
  }

  return (
    <>
      <PageHeader title="ورود به سامانه" subtitle="برای ادامه وارد شوید." />
      {expired && <StateBlock kind="locked" title="نشست شما پایان یافته است" detail="برای ادامه دوباره وارد شوید." />}
      {!enabled && <StateBlock kind="locked" title="ورود در حال حاضر فعال نیست" detail="سامانه ورود هنوز فعال نشده است." />}
      <Card title="ورود">
        <form className="login-form" onSubmit={e => { void submit(e) }} aria-label="فرم ورود" noValidate>
          <label className="lbl">نام کاربری
            <input className="input" name="login" autoComplete="username" dir="ltr" required maxLength={256} disabled={!enabled || busy} />
          </label>
          <label className="lbl">گذرواژه
            <input className="input" name="password" type="password" autoComplete="current-password" dir="ltr" required maxLength={1024}
              disabled={!enabled || busy} />
          </label>
          {error && <p className="login-error" role="alert">{error}</p>}
          <div><Button type="submit" variant="primary" disabled={!enabled || busy}>{busy ? 'در حال ورود…' : 'ورود'}</Button></div>
        </form>
      </Card>
    </>
  )
}
