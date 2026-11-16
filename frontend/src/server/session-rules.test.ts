import { describe, expect, it } from 'vitest'

import type { Role } from '@/shared/api/types'
import type { ProtectedRoute } from '@/shared/navigation/routes'

import type { BackendAnswer, BackendGet } from './backend'
import {
  type Session,
  decideAccess,
  loadArea,
  loadSession
} from './session-rules'

const SESSION = 'MZXW6YTBOI4DCMRTGQ2TMNZYHE'

function backend(answers: Record<string, BackendAnswer>) {
  const calls: string[] = []
  const get: BackendGet = async (path, cookie) => {
    calls.push(`${path} ${cookie}`)
    const answer = answers[path]
    if (!answer) throw new Error(`unexpected ${path}`)
    return answer
  }
  return { calls, get }
}

describe('loadSession', () => {
  it('without a cookie is anonymous and calls nothing', async () => {
    const b = backend({})
    expect(await loadSession(undefined, b.get)).toEqual({ kind: 'anonymous' })
    expect(b.calls).toEqual([])
  })

  it('with a malformed cookie is anonymous and calls nothing', async () => {
    const b = backend({})
    expect(await loadSession('not-a-token', b.get)).toEqual({
      kind: 'anonymous'
    })
    expect(b.calls).toEqual([])
  })

  it('asks /api/me with the cookie', async () => {
    const b = backend({
      '/api/me': { body: { role: 'comum', user_id: 7 }, status: 200 }
    })
    expect(await loadSession(SESSION, b.get)).toEqual({
      kind: 'user',
      user: { role: 'comum', userId: 7 }
    })
    expect(b.calls).toEqual([`/api/me ${SESSION}`])
  })

  it('401 means the session expired', async () => {
    const b = backend({
      '/api/me': { body: { error: 'unauthorized' }, status: 401 }
    })
    expect(await loadSession(SESSION, b.get)).toEqual({ kind: 'expired' })
  })

  it.each<[string, BackendAnswer]>([
    ['a 500', { body: { error: 'internal' }, status: 500 }],
    ['a 403', { body: { error: 'forbidden' }, status: 403 }],
    ['a non-JSON 500', { body: null, status: 500 }],
    ['an unknown role', { body: { role: 'root', user_id: 1 }, status: 200 }],
    ['a missing user id', { body: { role: 'admin' }, status: 200 }],
    ['a zero user id', { body: { role: 'admin', user_id: 0 }, status: 200 }],
    ['a string user id', { body: { role: 'admin', user_id: '1' }, status: 200 }]
  ])('fails closed on %s', async (_name, answer) => {
    const b = backend({ '/api/me': answer })
    await expect(loadSession(SESSION, b.get)).rejects.toThrow(/\/api\/me/)
  })
})

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

describe('loadArea', () => {
  const area = (answer: BackendAnswer) =>
    loadArea(
      '/api/administradores',
      '/admin',
      SESSION,
      backend({ '/api/administradores': answer }).get
    )

  it('200 returns the area', async () => {
    expect(
      await area({ body: { area: 'administradores' }, status: 200 })
    ).toEqual({ area: 'administradores', kind: 'ok' })
  })

  it('403 after an allowed role means no permission', async () => {
    expect(await area({ body: { error: 'forbidden' }, status: 403 })).toEqual({
      kind: 'forbidden'
    })
  })

  it('401 (session ended in between) goes to login with the notice', async () => {
    expect(
      await area({ body: { error: 'unauthorized' }, status: 401 })
    ).toEqual({ kind: 'login', url: '/login?next=%2Fadmin&sessao=expirada' })
  })

  it.each<[string, BackendAnswer]>([
    ['a 500', { body: { error: 'internal' }, status: 500 }],
    ['a 200 without area', { body: {}, status: 200 }]
  ])('fails closed on %s', async (_name, answer) => {
    await expect(area(answer)).rejects.toThrow(/administradores/)
  })
})
