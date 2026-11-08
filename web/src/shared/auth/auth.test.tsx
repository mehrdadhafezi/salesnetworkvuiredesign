import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { afterEach, describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// All accounts and tokens are SYNTHETIC.
const config = configFromEnv({ PROD: true, DEV: false })
const USER = '00000000-0000-4000-8000-000000001501'
const CSRF = 'c'.repeat(43)
const authSession = { user_id: USER, display_name: 'فروشنده آزمایشی', csrf_token: CSRF, idle_expires_at: '2099-01-01T00:00:00Z',
  absolute_expires_at: '2099-01-01T08:00:00Z' }

function serverState() {
  return { authenticated: false, state: 'none' as 'none' | 'active' | 'invalid', loginEnabled: true }
}

function routes(s: ReturnType<typeof serverState>, extra: Record<string, (r: Request) => Response | Promise<Response>> = {}) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ auth_mode: 'session', authenticated: s.authenticated, session_state: s.state,
      login_enabled: s.loginEnabled, ...(s.authenticated ? { session: authSession } : {}) })),
    ...extra,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

afterEach(() => {
  localStorage.clear()
  sessionStorage.clear()
})

describe('R1 login and session (Persian RTL)', () => {
  it('protects every route: unauthenticated session mode shows only the login form', async () => {
    renderAt('/system', routes(serverState()))
    expect(await screen.findByRole('form', { name: 'فرم ورود' })).toBeTruthy()
    expect(screen.queryByRole('heading', { level: 1, name: 'وضعیت سامانه' })).toBeNull()
    expect(screen.getByText('وارد نشده‌اید')).toBeTruthy()
    // No enrolment, recovery or remember-me control exists.
    expect(screen.queryByText(/فراموشی|ثبت‌نام|مرا به خاطر/)).toBeNull()
    expect(screen.queryByRole('checkbox')).toBeNull()
  })

  it('logs in, clears the password, restores the session and sends the CSRF token on logout', async () => {
    const s = serverState()
    const posts: Request[] = []
    const { calls } = renderAt('/', routes(s, {
      'POST /api/v1/auth/login': async (req: Request) => {
        posts.push(req.clone())
        s.authenticated = true
        s.state = 'active'
        return json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-1501', session: authSession })
      },
      'POST /api/v1/auth/logout': async (req: Request) => {
        posts.push(req.clone())
        s.authenticated = false
        s.state = 'none'
        return json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-1502' })
      },
    }))
    const form = await screen.findByRole('form', { name: 'فرم ورود' })
    fireEvent.change(within(form).getByLabelText('نام کاربری'), { target: { value: ' seller.one ' } })
    const pw = within(form).getByLabelText('گذرواژه') as HTMLInputElement
    expect(pw.type).toBe('password')
    expect(pw.autocomplete).toBe('current-password')
    fireEvent.change(pw, { target: { value: 'synthetic-Pass-1' } })
    fireEvent.click(within(form).getByRole('button', { name: 'ورود' }))
    expect(await screen.findByText('فروشنده آزمایشی')).toBeTruthy()
    expect(await screen.findByRole('heading', { level: 1, name: 'خانه' })).toBeTruthy()
    const [login] = posts
    if (!login) throw new Error('no login request')
    expect(await login.json()).toEqual({ login: 'seller.one', password: 'synthetic-Pass-1' })
    expect(login.headers.get('X-CSRF-Token')).toBeNull()
    // Nothing secret persisted in browser storage.
    const stored = JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage })
    expect(stored).not.toContain('synthetic-Pass-1')
    expect(stored).not.toContain(CSRF)
    fireEvent.click(screen.getByRole('button', { name: 'خروج' }))
    expect(await screen.findByRole('form', { name: 'فرم ورود' })).toBeTruthy()
    const logout = posts[1]
    if (!logout) throw new Error('no logout request')
    expect(logout.headers.get('X-CSRF-Token')).toBe(CSRF)
    expect(calls.filter(c => c.url === '/api/v1/session').length).toBeGreaterThanOrEqual(3)
  })

  it('reports invalid credentials, throttling, a disabled login and a network failure without leaking which part was wrong', async () => {
    let mode: 'invalid' | 'throttled' | 'blocked' | 'down' = 'invalid'
    renderAt('/', routes(serverState(), {
      'POST /api/v1/auth/login': () => {
        if (mode === 'invalid') return json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-1' }, 401)
        if (mode === 'throttled') return json({ type: '/problems/too-many-attempts', title: 'Too many attempts', status: 429, reason_code: 'ACTION_DENIED', correlation_id: 'c-2' }, 429, 'application/problem+json')
        if (mode === 'blocked') return json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-3' })
        throw new TypeError('network down')
      },
    }))
    const form = await screen.findByRole('form', { name: 'فرم ورود' })
    const attempt = async (expected: string) => {
      fireEvent.change(within(form).getByLabelText('نام کاربری'), { target: { value: 'seller.one' } })
      fireEvent.change(within(form).getByLabelText('گذرواژه'), { target: { value: 'wrong' } })
      fireEvent.click(within(form).getByRole('button', { name: 'ورود' }))
      expect(await screen.findByText(expected)).toBeTruthy()
      expect((within(form).getByLabelText('گذرواژه') as HTMLInputElement).value).toBe('')
    }
    await attempt('نام کاربری یا گذرواژه درست نیست.')
    mode = 'throttled'
    await attempt('تلاش‌های ناموفق زیاد بوده است. کمی بعد دوباره تلاش کنید.')
    mode = 'blocked'
    await attempt('ورود به سامانه در حال حاضر فعال نیست.')
    mode = 'down'
    await attempt('ارتباط با سرور برقرار نشد. دوباره تلاش کنید.')
  })

  it('shows an expired-session notice and a disabled form when login is off', async () => {
    const s = serverState()
    s.state = 'invalid'
    s.loginEnabled = false
    renderAt('/', routes(s))
    expect(await screen.findByText('نشست شما پایان یافته است')).toBeTruthy()
    expect(screen.getByText('ورود در حال حاضر فعال نیست')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'ورود' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('a denied request re-checks the session; an ended session returns to login', async () => {
    const s = serverState()
    s.authenticated = true
    s.state = 'active'
    renderAt('/system', routes(s, {
      'GET /health/ready': () => {
        s.authenticated = false
        s.state = 'invalid'
        return json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-9' }, 403)
      },
    }))
    expect(await screen.findByText('نشست شما پایان یافته است')).toBeTruthy()
  })

  it('a session that cannot be checked is an error, never a silent anonymous page', async () => {
    renderAt('/', { ...healthy, 'GET /api/v1/session': () => { throw new TypeError('network down') } })
    expect(await screen.findByText('وضعیت نشست مشخص نشد')).toBeTruthy()
    await waitFor(() => expect(screen.queryByRole('heading', { level: 1, name: 'خانه' })).toBeNull())
  })
})
