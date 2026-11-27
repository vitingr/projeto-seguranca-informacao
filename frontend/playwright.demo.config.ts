import { defineConfig, devices } from '@playwright/test'

import base from './playwright.config'

// `pnpm demo` (scripts/demo.mjs): the class demo of the assignment
// requirements (e2e/enunciado.spec.ts) in a visible Chromium, one test at a
// time and slowed down. The script opens the HTML report afterwards.
export default defineConfig({
  ...base,
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  reporter: [['list'], ['html', { open: 'never' }]],
  testMatch: 'enunciado.spec.ts',
  // slowMo makes every action take longer than the default 30 s allows.
  timeout: 120_000,
  use: {
    ...base.use,
    headless: false,
    launchOptions: { ...base.use?.launchOptions, slowMo: 400 }
  },
  workers: 1
})
