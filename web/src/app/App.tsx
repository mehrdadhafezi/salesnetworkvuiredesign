import { useMemo } from 'react'
import { RouterProvider, createBrowserRouter, type createMemoryRouter } from 'react-router'
import { ApiProvider, createApi, type ApiClient } from '../shared/api'
import { SessionProvider } from '../shared/auth/session'
import type { AppConfig } from './config'
import { ErrorBoundary } from './ErrorBoundary'
import { buildRoutes } from './routes'

type DataRouter = ReturnType<typeof createBrowserRouter> | ReturnType<typeof createMemoryRouter>

export interface AppProps {
  config: AppConfig
  /** Injected in tests; built from config otherwise. */
  client?: ApiClient
  router?: DataRouter
}

export function App({ config, client, router }: AppProps) {
  const api = useMemo(() => client ?? createApi({ baseUrl: config.apiBaseUrl, devUser: config.devUser }), [client, config])
  const r = useMemo(() => router ?? createBrowserRouter(buildRoutes(config)), [router, config])
  return (
    <ErrorBoundary>
      <ApiProvider client={api}>
        <SessionProvider>
          <RouterProvider router={r} />
        </SessionProvider>
      </ApiProvider>
    </ErrorBoundary>
  )
}
