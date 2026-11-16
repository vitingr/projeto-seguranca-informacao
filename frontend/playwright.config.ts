import { randomBytes } from 'node:crypto'

import { defineConfig, devices } from '@playwright/test'

import { backendSpkiHash } from './e2e/support/cert'
import { generateComumCredentials } from './e2e/support/env'

// The config is loaded by the main process and again by each worker; the
// workers inherit these variables, so every process sees the same user.
if (!process.env.E2E_COMUM_LOGIN || !process.env.E2E_COMUM_PASSWORD) {
  const comum = generateComumCredentials(randomBytes)
  process.env.E2E_COMUM_LOGIN = comum.login
  process.env.E2E_COMUM_PASSWORD = comum.password
}

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
  // *.test.ts files under e2e/support are Vitest unit tests.
  testMatch: '**/*.spec.ts',
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
  projects: [
    { name: 'setup', testMatch: /.*\.setup\.ts/ },
    {
      dependencies: ['setup'],
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] }
    }
  ],
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
