import { defineConfig, devices } from '@playwright/test'

// RM-E02: real-browser checks of the role-oriented navigation shell on the PRODUCTION
// bundle (vite build → vite preview; no DEV route, no dev identity). Every /api and
// /health request is answered inside the test by synthetic fixtures (page.route); no
// API process, database or stack is involved, so a passing run is presentation
// evidence only — never proof of business authorization (that stays with the
// Go full-stack suites, playwright.config.ts).
const port = 4183

export default defineConfig({
  testDir: './e2e',
  testMatch: /role-navigation\.spec\.ts$/,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  forbidOnly: true,
  timeout: 30_000,
  reporter: [['list']],
  outputDir: 'test-results-nav',
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    screenshot: 'only-on-failure',
    trace: 'off',
    video: 'off',
    locale: 'fa-IR',
    serviceWorkers: 'block',
  },
  webServer: {
    command: `npx vite build --outDir dist-nav --emptyOutDir && npx vite preview --outDir dist-nav --host 127.0.0.1 --port ${port} --strictPort`,
    url: `http://127.0.0.1:${port}/`,
    reuseExistingServer: false,
    timeout: 180_000,
  },
  projects: [
    { name: 'nav-desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1366, height: 768 } } },
    { name: 'nav-mobile', use: { ...devices['Desktop Chrome'], viewport: { width: 390, height: 844 }, hasTouch: true } },
  ],
})
