import { describe, expect, it } from 'vitest'

import type { Role } from '@/shared/api/types'

import { landingFor, loginUrl, safeNext } from '.'

describe('landingFor', () => {
  it.each<[Role, string]>([
    ['admin', '/admin'],
    ['comum', '/minha-conta']
  ])('%s → %s', (role, want) => {
    expect(landingFor(role)).toBe(want)
  })
})

describe('safeNext', () => {
  it.each<[unknown, Role, string]>([
    ['/admin', 'admin', '/admin'],
    ['/cadastrar-usuarios', 'admin', '/cadastrar-usuarios'],
    ['/minha-conta', 'comum', '/minha-conta'],
    // a route of another role
    ['/admin', 'comum', '/minha-conta'],
    ['/cadastrar-usuarios', 'comum', '/minha-conta'],
    ['/minha-conta', 'admin', '/admin'],
    // no open redirect
    ['https://evil.example/admin', 'admin', '/admin'],
    ['//evil.example', 'admin', '/admin'],
    ['/\\evil.example', 'comum', '/minha-conta'],
    ['/login', 'admin', '/admin'],
    ['/', 'comum', '/minha-conta'],
    ['/admin/', 'admin', '/admin'],
    ['/admin?x=1', 'admin', '/admin'],
    ['/ADMIN', 'admin', '/admin'],
    ['toString', 'admin', '/admin'],
    ['__proto__', 'comum', '/minha-conta'],
    // not a single string
    [undefined, 'admin', '/admin'],
    [['/admin', '/minha-conta'], 'admin', '/admin'],
    ['', 'comum', '/minha-conta']
  ])('next=%j for %s → %s', (next, role, want) => {
    expect(safeNext(next, role)).toBe(want)
  })
})

describe('loginUrl', () => {
  it('keeps the path to come back to', () => {
    expect(loginUrl('/admin')).toBe('/login?next=%2Fadmin')
  })

  it('marks an expired session', () => {
    expect(loginUrl('/minha-conta', { expired: true })).toBe(
      '/login?next=%2Fminha-conta&sessao=expirada'
    )
  })
})
