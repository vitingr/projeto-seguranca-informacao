import { X509Certificate, createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import path from 'node:path'

// Playwright loads these files as CommonJS (the package has no "type":
// "module"), so paths are built from __dirname, not import.meta.url. The
// default folder matches scripts/dev-args.mjs.
const certDir =
  process.env.BACKEND_CERT_DIR ||
  path.join(__dirname, '..', '..', '..', 'backend', 'certs')

/**
 * Base64 SHA-256 of the backend certificate's public key (SPKI). Chromium's
 * --ignore-certificate-errors-spki-list then trusts only this key: any other
 * certificate error still fails the test (spec D-16).
 */
export const backendSpkiHash = (): string => {
  const crt = path.resolve(certDir, 'server.crt')
  const der = new X509Certificate(readFileSync(crt)).publicKey.export({
    format: 'der',
    type: 'spki'
  })
  return createHash('sha256').update(der).digest('base64')
}
