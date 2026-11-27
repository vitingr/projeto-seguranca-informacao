// `pnpm demo`: the class demo (playwright.demo.config.ts), then the HTML
// report. Playwright opens the report by itself only when stdin is a TTY,
// which Node often does not get under Git Bash (mintty), so it is opened
// here, pass or fail. Ctrl+C stops the report server; the exit code is the tests' one.
import { spawnSync } from 'node:child_process'
import { createRequire } from 'node:module'

const cli = createRequire(import.meta.url).resolve('@playwright/test/cli')

const playwright = args =>
  spawnSync(process.execPath, [cli, ...args], { stdio: 'inherit' })

const run = playwright([
  'test',
  '--config',
  'playwright.demo.config.ts',
  '--project=chromium',
  '--headed',
  '--workers=1',
  ...process.argv.slice(2)
])
playwright(['show-report'])
process.exit(run.status ?? 1)
