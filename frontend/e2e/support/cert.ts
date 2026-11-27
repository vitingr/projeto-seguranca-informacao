import { X509Certificate, createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import path from 'node:path'

// Playwright loads this as CommonJS: __dirname, not import.meta.url
const certDir =
  process.env.BACKEND_CERT_DIR ||
  path.join(__dirname, '..', '..', '..', 'backend', 'certs')

// Chromium trusts only this public key: any other certificate error still fails
export const backendSpkiHash = (): string => {
  const crt = path.resolve(certDir, 'server.crt')
  const der = new X509Certificate(readFileSync(crt)).publicKey.export({
    format: 'der',
    type: 'spki'
  })
  return createHash('sha256').update(der).digest('base64')
}
