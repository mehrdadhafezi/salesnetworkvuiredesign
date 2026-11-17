import { defineConfig, devices } from '@playwright/test'

// RM-E03: real-browser checks of the DEV/TEST-only Seller V1 preview (/dev/seller/*).
// Starts its own Vite dev server on a dedicated port. Every /api and /health request is answered
// inside the test (and recorded) — no API process, database or stack is involved, so a passing run
// is presentation evidence only, never business authorization.
// Viewports match the frozen prototype renders: 1366 desktop, 1920 wide, 768 tablet, 390 mobile.
const port = 5182

export default defineConfig({
  testDir: './e2e',
  testMatch: /seller-preview\.spec\.ts$/,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: true,
  timeout: 30_000,
  reporter: [['list']],
  outputDir: 'test-results-seller',
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
    url: `http://127.0.0.1:${port}/dev/seller/leads`,
    reuseExistingServer: false,
    timeout: 120_000,
  },
  projects: [
    { name: 'seller-desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1366, height: 768 } } },
    { name: 'seller-wide', use: { ...devices['Desktop Chrome'], viewport: { width: 1920, height: 1080 } } },
    { name: 'seller-tablet', use: { ...devices['Desktop Chrome'], viewport: { width: 768, height: 1024 }, hasTouch: true } },
    { name: 'seller-mobile', use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 }, hasTouch: true } },
  ],
})
