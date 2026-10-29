import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import { fakeApi, healthy, json, session } from '../test/fakeApi'
import { App } from './App'
import { configFromEnv, type AppConfig } from './config'
import { ErrorBoundary } from './ErrorBoundary'
import { buildRoutes } from './routes'

const prodConfig: AppConfig = configFromEnv({ PROD: true, DEV: false })
const devConfig: AppConfig = configFromEnv({ PROD: false, DEV: true, VITE_DEV_AUTH_USER: 'dev-user-001' })

function renderAt(path: string, config = prodConfig, routes = healthy) {
  const { client, calls } = fakeApi(routes, config.devUser)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

describe('shell', () => {
  it('renders landmarks, skip link and one h1', async () => {
    renderAt('/')
    expect(await screen.findByRole('heading', { level: 1, name: 'خانه' })).toBeTruthy()
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
    expect(screen.getByRole('banner')).toBeTruthy()
    expect(screen.getByRole('navigation', { name: 'ناوبری اصلی' })).toBeTruthy()
    expect(screen.getByRole('main').id).toBe('ws')
    expect(screen.getByRole('link', { name: 'پرش به محتوای اصلی' }).getAttribute('href')).toBe('#ws')
    expect(screen.getByRole('link', { name: 'خانه' }).getAttribute('aria-current')).toBe('page')
  })

  it('routes between pages and shows 404 for unknown paths', async () => {
    renderAt('/')
    fireEvent.click(await screen.findByRole('link', { name: 'وضعیت سامانه' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'وضعیت سامانه' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'وضعیت سامانه' }).getAttribute('aria-current')).toBe('page')
    renderAt('/api/v1/customers')
    expect(await screen.findByRole('heading', { level: 1, name: 'صفحه پیدا نشد' })).toBeTruthy()
  })

  it('shows technical health through the generated client', async () => {
    const { calls } = renderAt('/system')
    expect(await screen.findByText('فعال')).toBeTruthy()
    expect(await screen.findByText('آماده')).toBeTruthy()
    const paths = calls.map(c => `${c.method} ${c.url}`)
    expect(paths).toEqual(expect.arrayContaining(['GET /health/live', 'GET /health/ready', 'GET /api/v1/session']))
    for (const c of calls) expect(c.headers.get('X-Correlation-ID')).toMatch(/^test-corr-\d{4}$/)
  })

  it('renders a problem response as an error state, and transport failure as unknown', async () => {
    renderAt('/system', prodConfig, {
      ...healthy,
      'GET /health/live': () => json({ type: '/problems/internal', title: 'Internal server error', status: 500, reason_code: 'OUTCOME_UNKNOWN', correlation_id: 'srv-corr-0001' }, 500, 'application/problem+json'),
      'GET /health/ready': () => { throw new TypeError('network down') },
    })
    expect(await screen.findByText('Internal server error')).toBeTruthy()
    expect(screen.getByText(/OUTCOME_UNKNOWN/)).toBeTruthy()
    expect(await screen.findByText('ارتباط با سرویس برقرار نشد')).toBeTruthy()
  })

  it('collapses navigation behind an accessible toggle', async () => {
    renderAt('/')
    const toggle = await screen.findByRole('button', { name: 'منو' })
    const nav = screen.getByRole('navigation', { name: 'ناوبری اصلی' })
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    expect(toggle.getAttribute('aria-controls')).toBe(nav.id)
    fireEvent.click(toggle)
    expect(toggle.getAttribute('aria-expanded')).toBe('true')
    expect(nav.getAttribute('data-open')).toBe('true')
  })
})

describe('dev auth context', () => {
  it('production: no dev header, no dev route, header shows no identity', async () => {
    const { calls } = renderAt('/', prodConfig)
    expect(await screen.findByText('ورود هنوز فعال نیست')).toBeTruthy()
    for (const c of calls) expect(c.headers.has('X-Dev-User')).toBe(false)
    expect(screen.queryByRole('link', { name: /نمونه اجزا/ })).toBeNull()
  })

  it('development: sends the synthetic identity and displays it as synthetic', async () => {
    const { calls } = renderAt('/', devConfig, {
      ...healthy,
      'GET /api/v1/session': () => json(session({ authenticated: true, auth_mode: 'dev_stub', principal: { principal_id: 'dev-user-001', display_name: 'Dev Operator', synthetic: true } })),
    })
    expect(await screen.findByText('Dev Operator')).toBeTruthy()
    expect(screen.getByText('DEV · ساختگی')).toBeTruthy()
    expect(calls.every(c => c.headers.get('X-Dev-User') === 'dev-user-001')).toBe(true)
    expect(screen.getByRole('link', { name: /نمونه اجزا/ })).toBeTruthy()
  })
})

describe('error boundary', () => {
  it('shows an error state instead of a blank page', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    function Boom(): never { throw new Error('render failure') }
    render(<ErrorBoundary><Boom /></ErrorBoundary>)
    expect(screen.getByRole('alert').textContent).toContain('خطا در نمایش صفحه')
    expect(spy).toHaveBeenCalled()
  })
})

describe('dev primitives route', () => {
  it('renders all seven outcomes in development only', async () => {
    renderAt('/dev/primitives', devConfig)
    for (const o of ['SUCCESS', 'PARTIAL', 'FAILED', 'UNKNOWN', 'CONFLICT', 'BLOCKED', 'UNAUTHORIZED']) {
      expect(await screen.findByText(o)).toBeTruthy()
    }
  })

  it('is not mounted in production', async () => {
    renderAt('/dev/primitives', prodConfig)
    await waitFor(() => expect(screen.getByRole('heading', { level: 1, name: 'صفحه پیدا نشد' })).toBeTruthy())
  })
})
