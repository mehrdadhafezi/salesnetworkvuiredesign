import { defineConfig, devices } from '@playwright/test'

// RM-E01: browser checks of the DEV-only design-system showcase (/dev/design-system).
// Starts its own Vite dev server on a dedicated port; touches no API, database or stack.
// Separate from playwright.config.ts, whose projects belong to the Go full-stack orchestrators.
const port = 5181

export default defineConfig({
  testDir: './e2e',
  testMatch: /design-system(-evidence)?\.spec\.ts$/,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: true,
  timeout: 30_000,
  reporter: [['list']],
  outputDir: 'test-results-ds',
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    screenshot: 'only-on-failure',
    trace: 'off',
    video: 'off',
    locale: 'fa-IR',
    serviceWorkers: 'block',
  },
  webServer: {
    command: `npx vite --host 127.0.0.1 --port ${port} --strictPort`,
    url: `http://127.0.0.1:${port}/dev/design-system`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
  projects: [
    { name: 'ds-desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1366, height: 768 } } },
    { name: 'ds-mobile', use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 }, hasTouch: true } },
  ],
})
