import path from 'node:path'
import { describe, expect, it } from 'vitest'

import { certPaths, devArgs, missingCertMessage } from './dev-args.mjs'

describe('certPaths', () => {
  it('defaults to backend/certs next to the frontend', () => {
    const { crt, key } = certPaths(undefined)
    expect(crt.endsWith(path.join('backend', 'certs', 'server.crt'))).toBe(true)
    expect(key.endsWith(path.join('backend', 'certs', 'server.key'))).toBe(true)
  })

  it('uses BACKEND_CERT_DIR when set', () => {
    const dir = path.resolve('some', 'dir')
    expect(certPaths(dir)).toEqual({
      crt: path.join(dir, 'server.crt'),
      key: path.join(dir, 'server.key')
    })
  })
})

describe('missingCertMessage', () => {
  const paths = { crt: '/c/server.crt', key: '/c/server.key' }

  it.each([
    ['both exist', [paths.crt, paths.key], null],
    ['crt missing', [paths.key], /missing \/c\/server\.crt:/],
    ['key missing', [paths.crt], /missing \/c\/server\.key:/],
    ['both missing', [], /missing \/c\/server\.crt and \/c\/server\.key:/]
  ])('%s', (_name, present, want) => {
    const message = missingCertMessage(paths, file => present.includes(file))
    if (want === null) {
      expect(message).toBeNull()
    } else {
      expect(message).toMatch(want)
      expect(message).toContain('go tool mage cert')
    }
  })
})

describe('devArgs', () => {
  const paths = { crt: 'C.crt', key: 'K.key' }
  const args = devArgs(paths, ['-p', '3001'])
  const valueOf = flag => args[args.indexOf(flag) + 1]

  it('serves HTTPS with the backend key and certificate', () => {
    expect(args[0]).toBe('dev')
    expect(args).toContain('--experimental-https')
    expect(valueOf('--experimental-https-key')).toBe('K.key')
    expect(valueOf('--experimental-https-cert')).toBe('C.crt')
  })

  it('trusts the backend certificate through the CA flag (D-4)', () => {
    expect(valueOf('--experimental-https-ca')).toBe('C.crt')
  })

  it('binds to localhost only', () => {
    expect(valueOf('-H')).toBe('localhost')
  })

  it('never disables TLS verification', () => {
    expect(args.join(' ')).not.toMatch(/insecure|reject|NODE_TLS/i)
  })

  it('appends extra arguments last', () => {
    expect(args.slice(-2)).toEqual(['-p', '3001'])
  })
})
