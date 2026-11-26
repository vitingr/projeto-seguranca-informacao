import { describe, expect, it } from 'vitest'

import type { Role } from '@/domain/roles/types'

import { landingFor, loginUrl, safeNext } from '.'

describe('landingFor', () => {
  it.each<[Role, string]>([
    ['admin', '/administradores'],
    ['comum', '/usuarios']
  ])('%s → %s', (role, want) => {
    expect(landingFor(role)).toBe(want)
  })
})

describe('safeNext', () => {
  it.each<[unknown, Role, string]>([
    ['/administradores', 'admin', '/administradores'],
    ['/cadastro', 'admin', '/cadastro'],
    ['/usuarios', 'comum', '/usuarios'],
    // a route of another role
    ['/administradores', 'comum', '/usuarios'],
    ['/cadastro', 'comum', '/usuarios'],
    ['/usuarios', 'admin', '/administradores'],
    // the old page paths no longer exist (spec AC-0.7)
    ['/admin', 'admin', '/administradores'],
    ['/cadastrar-usuarios', 'admin', '/administradores'],
    ['/minha-conta', 'comum', '/usuarios'],
    ['/admin', 'comum', '/usuarios'],
    ['/minha-conta', 'admin', '/administradores'],
    // no open redirect
    ['https://evil.example/administradores', 'admin', '/administradores'],
    ['//evil.example', 'admin', '/administradores'],
    ['/\\evil.example', 'comum', '/usuarios'],
    ['/login', 'admin', '/administradores'],
    ['/', 'comum', '/usuarios'],
    ['/administradores/', 'admin', '/administradores'],
    ['/administradores?x=1', 'admin', '/administradores'],
    ['/ADMINISTRADORES', 'admin', '/administradores'],
    ['toString', 'admin', '/administradores'],
    ['__proto__', 'comum', '/usuarios'],
    // not a single string
    [undefined, 'admin', '/administradores'],
    [['/administradores', '/usuarios'], 'admin', '/administradores'],
    ['', 'comum', '/usuarios']
  ])('next=%j for %s → %s', (next, role, want) => {
    expect(safeNext(next, role)).toBe(want)
  })
})

describe('loginUrl', () => {
  it('keeps the path to come back to', () => {
    expect(loginUrl('/administradores')).toBe('/login?next=%2Fadministradores')
  })

  it('marks an expired session', () => {
    expect(loginUrl('/usuarios', { expired: true })).toBe(
      '/login?next=%2Fusuarios&sessao=expirada'
    )
  })
})
