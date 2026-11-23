import { describe, expect, it } from 'vitest'

import type { BackendAnswer, BackendGet } from '@/lib/backend/types'

import { loadAuditEvents, parseAuditPage } from '.'

const SESSION = 'MZXW6YTBOI4DCMRTGQ2TMNZYHE'

const removed = {
  actor: { login: 'admin', user_id: 1 },
  created_at: '2026-11-22T11:30:00Z',
  event: 'post_removed',
  id: 3,
  ip: '172.18.0.1',
  target: { id: 2, type: 'post' }
}
const failed = {
  actor: null,
  created_at: '2026-11-22T11:00:00Z',
  event: 'login_failed',
  id: 2,
  ip: null,
  target: { id: 2, login: 'maria', type: 'user' }
}
const apiPage = {
  has_next: false,
  items: [removed, failed],
  page: 1,
  page_size: 20
}

const backend = (answer: BackendAnswer) => {
  const calls: string[] = []
  const get: BackendGet = async (path, cookie) => {
    calls.push(`${path} ${cookie}`)
    return answer
  }
  return { calls, get }
}

describe('parseAuditPage', () => {
  it('converts the API page into domain types', () => {
    expect(parseAuditPage(apiPage)).toEqual({
      hasNext: false,
      items: [
        {
          actor: { login: 'admin', userId: 1 },
          createdAt: '2026-11-22T11:30:00Z',
          event: 'post_removed',
          id: 3,
          ip: '172.18.0.1',
          target: { id: 2, login: null, type: 'post' }
        },
        {
          actor: null,
          createdAt: '2026-11-22T11:00:00Z',
          event: 'login_failed',
          id: 2,
          ip: null,
          target: { id: 2, login: 'maria', type: 'user' }
        }
      ],
      page: 1,
      pageSize: 20
    })
  })

  it('an actor whose user is gone has no login', () => {
    const gone = { ...removed, actor: { user_id: 9 } }
    expect(
      parseAuditPage({ ...apiPage, items: [gone] })?.items[0]?.actor
    ).toEqual({ login: null, userId: 9 })
  })

  it.each([
    ['null', null],
    ['no items', { ...apiPage, items: undefined }],
    ['page 0', { ...apiPage, page: 0 }],
    ['an unknown event', { ...apiPage, items: [{ ...removed, event: 'x' }] }],
    ['a numeric ip', { ...apiPage, items: [{ ...removed, ip: 1 }] }],
    ['a missing ip', { ...apiPage, items: [{ ...removed, ip: undefined }] }],
    ['an actor without id', { ...apiPage, items: [{ ...removed, actor: {} }] }],
    [
      'a numeric actor login',
      { ...apiPage, items: [{ ...removed, actor: { login: 1, user_id: 1 } }] }
    ],
    [
      'an unknown target type',
      { ...apiPage, items: [{ ...removed, target: { id: 2, type: 'x' } }] }
    ],
    [
      'a missing target',
      { ...apiPage, items: [{ ...removed, target: undefined }] }
    ]
  ])('rejects %s', (_, value) => {
    expect(parseAuditPage(value)).toBeNull()
  })
})

describe('loadAuditEvents', () => {
  it('asks the page and the type with the session cookie', async () => {
    const b = backend({ body: apiPage, status: 200 })
    expect(
      (await loadAuditEvents('login_failed', 2, SESSION, b.get)).kind
    ).toBe('ok')
    expect(b.calls).toEqual([
      `/api/audit-events?page=2&type=login_failed ${SESSION}`
    ])
  })

  it('sends no type when there is no filter', async () => {
    const b = backend({ body: apiPage, status: 200 })
    await loadAuditEvents(null, 1, SESSION, b.get)
    expect(b.calls).toEqual([`/api/audit-events?page=1 ${SESSION}`])
  })

  it('a malformed cookie goes to login without calling the API', async () => {
    const b = backend({ body: apiPage, status: 200 })
    expect(await loadAuditEvents(null, 1, 'nope', b.get)).toEqual({
      kind: 'login',
      url: '/login?next=%2Fadmin%2Fauditoria'
    })
    expect(b.calls).toEqual([])
  })

  it('401 goes to login with the expired notice', async () => {
    const b = backend({ body: null, status: 401 })
    expect(await loadAuditEvents(null, 1, SESSION, b.get)).toEqual({
      kind: 'login',
      url: '/login?next=%2Fadmin%2Fauditoria&sessao=expirada'
    })
  })

  it('403 is forbidden', async () => {
    const b = backend({ body: null, status: 403 })
    expect(await loadAuditEvents(null, 1, SESSION, b.get)).toEqual({
      kind: 'forbidden'
    })
  })

  it.each([
    [500, { error: 'internal' }],
    [400, { error: 'bad_request' }],
    [200, { items: 'not a list' }]
  ])('throws on %i %j (fail closed)', async (status, body) => {
    const b = backend({ body, status })
    await expect(loadAuditEvents(null, 1, SESSION, b.get)).rejects.toThrow(
      `unexpected GET /api/audit-events answer: ${status}`
    )
  })
})
