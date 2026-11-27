import { describe, expect, it } from 'vitest'

import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { Role } from '@/domain/roles/types'

import { decideAccess } from '.'
import type { Session } from './types'

describe('decideAccess: the page guard matrix', () => {
  const anonymous: Session = { kind: 'anonymous' }
  const as = (role: Role): Session => ({
    kind: 'user',
    user: { role, userId: 1 }
  })

  it.each<[ProtectedRoute, Session, string]>([
    ['/administradores', anonymous, 'login /login?next=%2Fadministradores'],
    ['/administradores', as('comum'), 'forbidden'],
    ['/administradores', as('admin'), 'allowed'],
    ['/cadastro', anonymous, 'login /login?next=%2Fcadastro'],
    ['/cadastro', as('comum'), 'forbidden'],
    ['/cadastro', as('admin'), 'allowed'],
    ['/usuarios', anonymous, 'login /login?next=%2Fusuarios'],
    ['/usuarios', as('comum'), 'allowed'],
    ['/usuarios', as('admin'), 'forbidden']
  ])('%s', (route, session, want) => {
    const access = decideAccess(session, route)
    expect(access.kind === 'login' ? `login ${access.url}` : access.kind).toBe(
      want
    )
  })

  it('a ticket page sends its real path as next', () => {
    expect(
      decideAccess(anonymous, '/usuarios/tickets/[id]', '/usuarios/tickets/12')
    ).toEqual({ kind: 'login', url: '/login?next=%2Fusuarios%2Ftickets%2F12' })
    expect(
      decideAccess(
        as('admin'),
        '/usuarios/tickets/[id]',
        '/usuarios/tickets/12'
      ).kind
    ).toBe('forbidden')
    expect(
      decideAccess(
        as('admin'),
        '/administradores/tickets/[id]',
        '/administradores/tickets/3'
      ).kind
    ).toBe('allowed')
  })

  it('an expired session goes to login with the notice', () => {
    expect(decideAccess({ kind: 'expired' }, '/administradores')).toEqual({
      kind: 'login',
      url: '/login?next=%2Fadministradores&sessao=expirada'
    })
  })
})
