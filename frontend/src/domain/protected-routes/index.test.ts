import { describe, expect, it } from 'vitest'

import { matchProtectedRoute } from '.'

describe('matchProtectedRoute', () => {
  it.each([
    ['/usuarios', '/usuarios'],
    ['/usuarios/tickets/novo', '/usuarios/tickets/novo'],
    ['/usuarios/tickets/12', '/usuarios/tickets/[id]'],
    ['/administradores/tickets/7', '/administradores/tickets/[id]'],
    ['/usuarios/tickets/9223372036854775807', '/usuarios/tickets/[id]']
  ])('%s → %s', (path, route) => {
    expect(matchProtectedRoute(path)).toBe(route)
  })

  it.each([
    '/usuarios/tickets/[id]',
    '/usuarios/tickets/0',
    '/usuarios/tickets/012',
    '/usuarios/tickets/abc',
    '/usuarios/tickets/12/',
    '/usuarios/tickets/12?x=1',
    '/usuarios/tickets/12/../x',
    '/usuarios/tickets/%31%32',
    '/usuarios/tickets/12345678901234567890',
    '//evil.example',
    '__proto__'
  ])('%s matches nothing', path => {
    expect(matchProtectedRoute(path)).toBeNull()
  })
})
