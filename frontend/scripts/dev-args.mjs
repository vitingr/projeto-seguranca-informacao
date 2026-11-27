import path from 'node:path'
import { fileURLToPath } from 'node:url'

const defaultCertDir = fileURLToPath(
  new URL('../../backend/certs/', import.meta.url)
)

/** @param {string | undefined} certDir */
export function certPaths(certDir) {
  const dir = path.resolve(certDir || defaultCertDir)
  return {
    crt: path.join(dir, 'server.crt'),
    key: path.join(dir, 'server.key')
  }
}

/**
 * @param {{ crt: string, key: string }} paths
 * @param {(file: string) => boolean} exists
 */
export function missingCertMessage(paths, exists) {
  const missing = [paths.crt, paths.key].filter(file => !exists(file))
  if (missing.length === 0) return null
  return `missing ${missing.join(' and ')}: run "go tool mage cert" in backend/ (or set BACKEND_CERT_DIR)`
}

/**
 * @param {{ crt: string, key: string }} paths
 * @param {string[]} extra
 */
export function devArgs(paths, extra = []) {
  return [
    'dev',
    '--experimental-https',
    '--experimental-https-key',
    paths.key,
    '--experimental-https-cert',
    paths.crt,
    // Makes the Next server trust the backend certificate
    '--experimental-https-ca',
    paths.crt,
    '-H',
    'localhost',
    ...extra
  ]
}
