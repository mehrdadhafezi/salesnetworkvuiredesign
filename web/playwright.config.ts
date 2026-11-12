import { defineConfig, devices } from '@playwright/test'

// R7 full-stack E2E. The Go orchestrators (internal/integration/legacy/shadow,
// build tag e2e: TestBrowserFullStack for R7-1, TestBrowserPaymentsFullStack
// for R7-2, TestBrowserFulfillmentFullStack for R7-3) start one disposable stack each and pass E2E_* variables; this
// suite never starts servers and never runs against anything but that stack.
// Each project belongs to exactly one orchestrator and one flag state.
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
    // R7-1 (TestBrowserFullStack): contact outcome on the sales stack.
    { name: 'write-flag-on', testMatch: 'seller-contact-outcome.spec.ts', grepInvert: /@flag-off/, use: { ...devices['Desktop Chrome'] } },
    { name: 'write-flag-off', testMatch: 'seller-contact-outcome.spec.ts', grep: /@flag-off/, use: { ...devices['Desktop Chrome'] } },
    // R7-2 (TestBrowserPaymentsFullStack): invoices and payment submission.
    { name: 'payments-flags-on', testMatch: 'payments.spec.ts', grepInvert: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
    { name: 'payments-flags-off', testMatch: 'payments.spec.ts', grep: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
    // R7-3 (TestBrowserFulfillmentFullStack): work cases and provider operations.
    { name: 'fulfillment-flags-on', testMatch: 'fulfillment.spec.ts', grepInvert: /@flags-off|@flags-mixed/, use: { ...devices['Desktop Chrome'] } },
    { name: 'fulfillment-flags-mixed', testMatch: 'fulfillment.spec.ts', grep: /@flags-mixed/, use: { ...devices['Desktop Chrome'] } },
    { name: 'fulfillment-flags-off', testMatch: 'fulfillment.spec.ts', grep: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
  ],
})
