import type { RouteObject } from 'react-router'
import type { AppConfig } from './config'
import { Shell, type NavItem } from './Shell'

/**
 * Shell routes only: home, system status, 404 — plus a DEV-only primitives
 * demonstration route that production builds never mount. Pages load lazily.
 * No business route trees exist yet.
 */
export function buildRoutes(config: AppConfig): RouteObject[] {
  const nav: NavItem[] = [
    { to: '/', label: 'خانه' },
    { to: '/system', label: 'وضعیت سامانه' },
  ]
  const children: RouteObject[] = [
    { index: true, lazy: async () => ({ Component: (await import('../routes/HomePage')).HomePage }) },
    { path: 'system', lazy: async () => ({ Component: (await import('../routes/SystemPage')).SystemPage }) },
  ]
  // import.meta.env.DEV is a build-time constant: production bundles drop this branch entirely.
  if (import.meta.env.DEV && config.devTools) {
    nav.push({ to: '/dev/primitives', label: 'نمونه اجزا', dev: true })
    children.push({ path: 'dev/primitives', lazy: async () => ({ Component: (await import('../routes/dev/PrimitivesPage')).PrimitivesPage }) })
  }
  children.push({ path: '*', lazy: async () => ({ Component: (await import('../routes/NotFoundPage')).NotFoundPage }) })
  return [{ path: '/', element: <Shell nav={nav} />, children }]
}
