import { randomBytes } from 'node:crypto'

import { defineConfig, devices } from '@playwright/test'

import { backendSpkiHash } from './e2e/support/cert'
import { generateComumCredentials } from './e2e/support/env'

if (!process.env.E2E_COMUM_LOGIN || !process.env.E2E_COMUM_PASSWORD) {
  const comum = generateComumCredentials(randomBytes)
  process.env.E2E_COMUM_LOGIN = comum.login
  process.env.E2E_COMUM_PASSWORD = comum.password
}

export default defineConfig({
  expect: { timeout: 10_000 },
  forbidOnly: !!process.env.CI,
  fullyParallel: false,
  reporter: process.env.CI
    ? [['list'], ['html', { open: 'never' }]]
    : [['list']],
  retries: 0,
  testDir: 'e2e',
  testMatch: '**/*.spec.ts',
  use: {
    baseURL: 'https://localhost:3000',
    launchOptions: {
      args: [`--ignore-certificate-errors-spki-list=${backendSpkiHash()}`]
    },
    screenshot: 'off',
    // Traces would record typed passwords
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
  webServer: {
    command: 'node scripts/dev.mjs',
    port: 3000,
    reuseExistingServer: !process.env.CI,
    stdout: 'pipe',
    wait: { stdout: /Ready in/ }
  },
  workers: 1
})
