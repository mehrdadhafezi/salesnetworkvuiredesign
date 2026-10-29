/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  /** DEV/TEST only: synthetic identity such as dev-user-001. Ignored in production builds. */
  readonly VITE_DEV_AUTH_USER?: string
}
