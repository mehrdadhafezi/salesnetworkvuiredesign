import { defineConfig, devices } from '@playwright/test'

// R7 full-stack E2E. The Go orchestrators (internal/integration/legacy/shadow,
// build tag e2e: TestBrowserFullStack for R7-1, TestBrowserPaymentsFullStack
// for R7-2, TestBrowserFulfillmentFullStack for R7-3, TestBrowserHRFullStack and TestBrowserHRReadOverlay for R7-6,
// TestBrowserReportingBiawinFullStack and TestBrowserReportingBiawinReadOverlay for R7-7,
// TestBrowserCustomer360ReadOverlay for RM-P01, TestBrowserInvoiceWorkspaceReadOverlay for RM-P02, TestBrowserActivityTimelineReadOverlay for RM-P03, TestBrowserMISWorkspaceReadOverlay for RM-P05, TestBrowserMISEvidenceReadOverlay for RM-P06) start one disposable stack each and pass E2E_* variables; this
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
    // R7-6 (TestBrowserHRFullStack): HR reads and requests, production subject resolver.
    { name: 'hr-flags-on', testMatch: 'hr.spec.ts', grepInvert: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
    { name: 'hr-flags-off', testMatch: 'hr.spec.ts', grep: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
    // R7-6 (TestBrowserHRReadOverlay): HR reads through the TEST-ONLY dev auth stub (backend-only evidence).
    // RM-P04: the HR operational workspace runs on the same overlay stack (same dataset, same orchestrator).
    { name: 'hr-read-overlay', testMatch: /hr-(read|workspace)-overlay\.spec\.ts/, use: { ...devices['Desktop Chrome'] } },
    // R7-7 (TestBrowserReportingBiawinFullStack): reporting / MIS and Biawin reads, production subject resolver.
    { name: 'reporting-biawin-flags-on', testMatch: 'reporting-biawin.spec.ts', grepInvert: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
    { name: 'reporting-biawin-flags-off', testMatch: 'reporting-biawin.spec.ts', grep: /@flags-off/, use: { ...devices['Desktop Chrome'] } },
    // R7-7 (TestBrowserReportingBiawinReadOverlay): the same reads through the TEST-ONLY dev auth stub (backend-only evidence).
    { name: 'reporting-biawin-read-overlay', testMatch: 'reporting-biawin-overlay.spec.ts', use: { ...devices['Desktop Chrome'] } },
    // RM-P01 (TestBrowserCustomer360ReadOverlay): Customer ⇄ Case ⇄ Invoice navigation through the TEST-ONLY dev auth stub.
    { name: 'customer-360-read-overlay', testMatch: 'customer-360-overlay.spec.ts', use: { ...devices['Desktop Chrome'] } },
    // RM-P02 (TestBrowserInvoiceWorkspaceReadOverlay): the invoice 360° workspace through the TEST-ONLY dev auth stub.
    { name: 'invoice-workspace-read-overlay', testMatch: 'invoice-workspace-overlay.spec.ts', use: { ...devices['Desktop Chrome'] } },
    // RM-P03 (TestBrowserActivityTimelineReadOverlay): the unified activity timeline through the TEST-ONLY dev auth stub.
    { name: 'activity-timeline-read-overlay', testMatch: 'activity-timeline-overlay.spec.ts', use: { ...devices['Desktop Chrome'] } },
    // RM-P05 (TestBrowserMISWorkspaceReadOverlay): the MIS operational workspace through the TEST-ONLY dev auth stub.
    { name: 'mis-workspace-read-overlay', testMatch: 'mis-workspace-overlay.spec.ts', use: { ...devices['Desktop Chrome'] } },
    // RM-P06 (TestBrowserMISEvidenceReadOverlay): custody-to-case evidence continuity through the TEST-ONLY dev auth stub.
    { name: 'mis-evidence-read-overlay', testMatch: 'mis-evidence-overlay.spec.ts', use: { ...devices['Desktop Chrome'] } },
  ],
})
