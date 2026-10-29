/** Runtime configuration of the SPA, derived from Vite's build environment. */
export interface AppConfig {
  apiBaseUrl: string
  /** Synthetic DEV/TEST identity sent as X-Dev-User. Always undefined in production builds. */
  devUser: string | undefined
  /** Whether development-only demonstration routes are mounted. */
  devTools: boolean
  /** Presentation default zone until a profile display_zone exists. Not storage truth. */
  displayZone: string
}

interface BuildEnv {
  PROD: boolean
  DEV: boolean
  VITE_API_BASE_URL?: string
  VITE_DEV_AUTH_USER?: string
}

const DEV_USER_PATTERN = /^dev-user-\d{3}$/

/**
 * Production builds never send a dev identity and never mount dev routes,
 * whatever VITE_DEV_AUTH_USER says. The API refuses the stub in production too.
 */
export function configFromEnv(env: BuildEnv): AppConfig {
  // import.meta.env.DEV is a build-time constant, so production bundles contain
  // no dev identity handling at all; the env flags guard tests and dev builds.
  const devBuild = import.meta.env.DEV && !env.PROD && env.DEV
  const devUser = devBuild && env.VITE_DEV_AUTH_USER && DEV_USER_PATTERN.test(env.VITE_DEV_AUTH_USER)
    ? env.VITE_DEV_AUTH_USER
    : undefined
  return {
    apiBaseUrl: env.VITE_API_BASE_URL ?? '',
    devUser,
    devTools: devBuild,
    displayZone: 'Asia/Tehran',
  }
}
