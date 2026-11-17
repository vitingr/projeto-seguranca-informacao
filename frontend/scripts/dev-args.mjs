// Pure helpers for scripts/dev.mjs, kept apart so they can be unit-tested.
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const defaultCertDir = fileURLToPath(
  new URL('../../backend/certs/', import.meta.url)
)

/**
 * Paths of the backend development certificate and key.
 * @param {string | undefined} certDir BACKEND_CERT_DIR, or undefined for ../backend/certs
 */
export function certPaths(certDir) {
  const dir = path.resolve(certDir || defaultCertDir)
  return {
    crt: path.join(dir, 'server.crt'),
    key: path.join(dir, 'server.key')
  }
}

/**
 * The error to print when a file is missing, or null when both exist.
 * `next dev` does not stop on a missing file: it logs an unhandled
 * rejection and keeps running without serving, so the launcher checks first.
 * @param {{ crt: string, key: string }} paths
 * @param {(file: string) => boolean} exists
 */
export function missingCertMessage(paths, exists) {
  const missing = [paths.crt, paths.key].filter(file => !exists(file))
  if (missing.length === 0) return null
  return `missing ${missing.join(' and ')}: run "go tool mage cert" in backend/ (or set BACKEND_CERT_DIR)`
}

/**
 * Arguments for `next dev`: HTTPS with the backend certificate, bound to
 * localhost only. `--experimental-https-ca` is what makes the Next server
 * trust the backend: with HTTPS on, next dev sets the server process's
 * NODE_EXTRA_CA_CERTS to that file and ignores the environment value.
 * @param {{ crt: string, key: string }} paths
 * @param {string[]} extra arguments passed to `pnpm dev`
 */
export function devArgs(paths, extra = []) {
  return [
    'dev',
    '--experimental-https',
    '--experimental-https-key',
    paths.key,
    '--experimental-https-cert',
    paths.crt,
    '--experimental-https-ca',
    paths.crt,
    '-H',
    'localhost',
    ...extra
  ]
}
