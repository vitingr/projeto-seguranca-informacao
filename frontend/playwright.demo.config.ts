import { defineConfig, devices } from '@playwright/test'

import base from './playwright.config'

export default defineConfig({
  ...base,
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  reporter: [['list'], ['html', { open: 'never' }]],
  testMatch: 'enunciado.spec.ts',
  // slowMo needs more than the default 30 s
  timeout: 120_000,
  use: {
    ...base.use,
    headless: false,
    launchOptions: { ...base.use?.launchOptions, slowMo: 400 }
  },
  workers: 1
})
