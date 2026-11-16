import { defineConfig, devices } from '@playwright/test'

import { backendSpkiHash } from './e2e/support/cert'

// End-to-end tests against the real backend stack ("go tool mage dockerUp",
// "migrateUp" and "seed" in backend/) and `pnpm dev` (HTTPS, backend
// certificate). Every call goes through the browser and the /api proxy.
export default defineConfig({
  // next dev compiles each page on its first visit.
  expect: { timeout: 10_000 },
  forbidOnly: !!process.env.CI,
  fullyParallel: false,
  reporter: process.env.CI
    ? [['list'], ['html', { open: 'never' }]]
    : [['list']],
  retries: 0,
  testDir: 'e2e',
  use: {
    baseURL: 'https://localhost:3000',
    launchOptions: {
      args: [`--ignore-certificate-errors-spki-list=${backendSpkiHash()}`]
    },
    screenshot: 'off',
    // Traces would record typed passwords and evaluate() arguments.
    trace: 'off',
    video: 'off'
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  // `port` is a plain TCP check, so Node never has to trust the certificate.
  webServer: {
    command: 'node scripts/dev.mjs',
    port: 3000,
    reuseExistingServer: !process.env.CI,
    stdout: 'pipe',
    wait: { stdout: /Ready in/ }
  },
  workers: 1
})
