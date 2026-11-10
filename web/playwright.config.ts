import { defineConfig, devices } from '@playwright/test'

// R7-1 full-stack E2E. The Go orchestrator (internal/integration/legacy/shadow,
// build tag e2e) starts the disposable stack and passes E2E_* variables; this
// suite never starts servers and never runs against anything but that stack.
const baseURL = process.env.E2E_BASE_URL ?? ''

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: true,
  timeout: 30_000,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  outputDir: 'test-results',
  use: {
    baseURL,
    // Failure artifacts: screenshots only. Traces and videos are off because
    // they would record login request bodies and session cookies.
    screenshot: 'only-on-failure',
    trace: 'off',
    video: 'off',
    locale: 'fa-IR',
    serviceWorkers: 'block',
  },
  projects: [
    { name: 'write-flag-on', grepInvert: /@flag-off/, use: { ...devices['Desktop Chrome'] } },
    { name: 'write-flag-off', grep: /@flag-off/, use: { ...devices['Desktop Chrome'] } },
  ],
})
