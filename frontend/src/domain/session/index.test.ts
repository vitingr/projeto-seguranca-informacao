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
    ['/admin', anonymous, 'login /login?next=%2Fadmin'],
    ['/admin', as('comum'), 'forbidden'],
    ['/admin', as('admin'), 'allowed'],
    [
      '/cadastrar-usuarios',
      anonymous,
      'login /login?next=%2Fcadastrar-usuarios'
    ],
    ['/cadastrar-usuarios', as('comum'), 'forbidden'],
    ['/cadastrar-usuarios', as('admin'), 'allowed'],
    ['/minha-conta', anonymous, 'login /login?next=%2Fminha-conta'],
    ['/minha-conta', as('comum'), 'allowed'],
    ['/minha-conta', as('admin'), 'forbidden']
  ])('%s', (route, session, want) => {
    const access = decideAccess(session, route)
    expect(access.kind === 'login' ? `login ${access.url}` : access.kind).toBe(
      want
    )
  })

  it('an expired session goes to login with the notice', () => {
    expect(decideAccess({ kind: 'expired' }, '/admin')).toEqual({
      kind: 'login',
      url: '/login?next=%2Fadmin&sessao=expirada'
    })
  })
})
