// `pnpm dev`: next dev over HTTPS with the backend development certificate
// (backend/certs, created by "go tool mage cert"). See scripts/dev-args.mjs.
import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { createRequire } from 'node:module'

import { certPaths, devArgs, missingCertMessage } from './dev-args.mjs'

const paths = certPaths(process.env.BACKEND_CERT_DIR)
const missing = missingCertMessage(paths, existsSync)
if (missing) {
  console.error(missing)
  process.exit(1)
}

const nextBin = createRequire(import.meta.url).resolve('next/dist/bin/next')
const child = spawn(
  process.execPath,
  [nextBin, ...devArgs(paths, process.argv.slice(2))],
  {
    stdio: 'inherit'
  }
)

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => child.kill(signal))
}
child.on('exit', (code, signal) => process.exit(code ?? (signal ? 1 : 0)))
