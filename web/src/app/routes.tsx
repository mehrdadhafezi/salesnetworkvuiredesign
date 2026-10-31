import type { RouteObject } from 'react-router'
import type { AppConfig } from './config'
import { Shell, type NavItem } from './Shell'

/**
 * Shell routes (home, system status, 404), the Slice 4 HR read routes, and a
 * DEV-only primitives route that production builds never mount. Pages load
 * lazily. HR navigation is shown only when the server reports the HR read
 * feature enabled; the routes themselves render the server's BLOCKED /
 * UNAUTHORIZED answer, so hiding the link is a convenience, not security.
 */
export function buildRoutes(config: AppConfig): RouteObject[] {
  const nav: NavItem[] = [
    { to: '/', label: 'خانه' },
    { to: '/hr', label: 'نیروها', feature: 'hr_read' },
    { to: '/hr/structure', label: 'ساختار سازمانی', feature: 'hr_read' },
    { to: '/hr/exceptions', label: 'استثناهای سلسله‌مراتب', feature: 'hr_read' },
    { to: '/system', label: 'وضعیت سامانه' },
  ]
  const children: RouteObject[] = [
    { index: true, lazy: async () => ({ Component: (await import('../routes/HomePage')).HomePage }) },
    { path: 'system', lazy: async () => ({ Component: (await import('../routes/SystemPage')).SystemPage }) },
    { path: 'hr', lazy: async () => ({ Component: (await import('../modules/hr/pages/WorkforcePage')).WorkforcePage }) },
    { path: 'hr/people/:personId', lazy: async () => ({ Component: (await import('../modules/hr/pages/PersonPage')).PersonPage }) },
    { path: 'hr/structure', lazy: async () => ({ Component: (await import('../modules/hr/pages/StructurePage')).StructurePage }) },
    { path: 'hr/exceptions', lazy: async () => ({ Component: (await import('../modules/hr/pages/ExceptionsPage')).ExceptionsPage }) },
  ]
  // import.meta.env.DEV is a build-time constant: production bundles drop this branch entirely.
  if (import.meta.env.DEV && config.devTools) {
    nav.push({ to: '/dev/primitives', label: 'نمونه اجزا', dev: true })
    children.push({ path: 'dev/primitives', lazy: async () => ({ Component: (await import('../routes/dev/PrimitivesPage')).PrimitivesPage }) })
  }
  children.push({ path: '*', lazy: async () => ({ Component: (await import('../routes/NotFoundPage')).NotFoundPage }) })
  return [{ path: '/', element: <Shell nav={nav} />, children }]
}
