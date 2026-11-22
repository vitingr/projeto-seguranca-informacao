import { describe, expect, it } from 'vitest'

import type { BackendAnswer, BackendGet } from '@/lib/backend/types'

import { loadPosts, parsePostPage } from '.'

const SESSION = 'MZXW6YTBOI4DCMRTGQ2TMNZYHE'

const apiPost = {
  author: { login: 'maria', user_id: 2 },
  body: 'linha 1\nlinha 2',
  created_at: '2026-11-22T12:00:00Z',
  id: 7,
  mine: true,
  title: 'Olá',
  updated_at: '2026-11-22T12:00:00Z'
}
const apiPage = { has_next: true, items: [apiPost], page: 2, page_size: 20 }

const backend = (answer: BackendAnswer) => {
  const calls: string[] = []
  const get: BackendGet = async (path, cookie) => {
    calls.push(`${path} ${cookie}`)
    return answer
  }
  return { calls, get }
}

describe('parsePostPage', () => {
  it('converts the API page into domain types', () => {
    expect(parsePostPage(apiPage)).toEqual({
      hasNext: true,
      items: [
        {
          author: { login: 'maria', userId: 2 },
          body: 'linha 1\nlinha 2',
          createdAt: '2026-11-22T12:00:00Z',
          id: 7,
          mine: true,
          title: 'Olá',
          updatedAt: '2026-11-22T12:00:00Z'
        }
      ],
      page: 2,
      pageSize: 20
    })
  })

  it.each([
    ['null', null],
    ['no items', { ...apiPage, items: undefined }],
    ['page 0', { ...apiPage, page: 0 }],
    ['has_next not boolean', { ...apiPage, has_next: 'yes' }],
    [
      'post without author',
      { ...apiPage, items: [{ ...apiPost, author: null }] }
    ],
    ['post id as string', { ...apiPage, items: [{ ...apiPost, id: '7' }] }],
    [
      'post title not string',
      { ...apiPage, items: [{ ...apiPost, title: 1 }] }
    ],
    ['mine missing', { ...apiPage, items: [{ ...apiPost, mine: undefined }] }]
  ])('rejects %s', (_, value) => {
    expect(parsePostPage(value)).toBeNull()
  })
})

describe('loadPosts', () => {
  it('asks the page with the session cookie', async () => {
    const b = backend({ body: apiPage, status: 200 })
    const result = await loadPosts(2, SESSION, b.get)
    expect(result.kind).toBe('ok')
    expect(b.calls).toEqual([`/api/posts?page=2 ${SESSION}`])
  })

  it('a malformed cookie goes to login without calling the API', async () => {
    const b = backend({ body: apiPage, status: 200 })
    expect(await loadPosts(1, 'nope', b.get)).toEqual({
      kind: 'login',
      url: '/login?next=%2Fmural'
    })
    expect(b.calls).toEqual([])
  })

  it('401 goes to login with the expired notice', async () => {
    const b = backend({ body: null, status: 401 })
    expect(await loadPosts(1, SESSION, b.get)).toEqual({
      kind: 'login',
      url: '/login?next=%2Fmural&sessao=expirada'
    })
  })

  it('403 is forbidden', async () => {
    const b = backend({ body: null, status: 403 })
    expect(await loadPosts(1, SESSION, b.get)).toEqual({ kind: 'forbidden' })
  })

  it.each([
    [500, { error: 'internal' }],
    [400, { error: 'bad_request' }],
    [200, { items: 'not a list' }]
  ])('throws on %i %j (fail closed)', async (status, body) => {
    const b = backend({ body, status })
    await expect(loadPosts(1, SESSION, b.get)).rejects.toThrow(
      `unexpected GET /api/posts answer: ${status}`
    )
  })
})
