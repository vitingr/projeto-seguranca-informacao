import { describe, expect, it } from 'vitest'

import { createBackendGet, isSessionValue } from './backend'

const SESSION = 'MZXW6YTBOI4DCMRTGQ2TMNZYHE'

describe('isSessionValue', () => {
  it.each([
    [SESSION, true],
    [undefined, false],
    ['', false],
    ['mzxw6ytboi4dcmrtgq2tmnzyhe', false],
    [`${SESSION}A`, false],
    [`${SESSION.slice(0, 25)};`, false],
    [`${SESSION.slice(0, 24)}\r\n`, false]
  ])('%j → %s', (value, want) => {
    expect(isSessionValue(value)).toBe(want)
  })
})

describe('createBackendGet', () => {
  it('sends only the session cookie, uncached, to the backend origin', async () => {
    let seen: { init?: RequestInit; url: string } | undefined
    const get = createBackendGet('https://localhost', async (url, init) => {
      seen = { init, url: String(url) }
      return new Response(JSON.stringify({ role: 'admin', user_id: 1 }), {
        status: 200
      })
    })
    const answer = await get('/api/me', SESSION)
    expect(answer).toEqual({ body: { role: 'admin', user_id: 1 }, status: 200 })
    expect(seen?.url).toBe('https://localhost/api/me')
    expect(seen?.init?.cache).toBe('no-store')
    expect(seen?.init?.redirect).toBe('error')
    expect(seen?.init?.headers).toEqual({ cookie: `__Host-session=${SESSION}` })
  })

  it('returns a null body when the answer is not JSON', async () => {
    const get = createBackendGet(
      'https://localhost',
      async () => new Response('Internal Server Error', { status: 500 })
    )
    expect(await get('/api/me', SESSION)).toEqual({ body: null, status: 500 })
  })

  it('refuses to forward a malformed cookie', async () => {
    let called = false
    const get = createBackendGet('https://localhost', async () => {
      called = true
      return new Response(null, { status: 200 })
    })
    await expect(get('/api/me', 'x; other=1')).rejects.toThrow(/malformed/)
    expect(called).toBe(false)
  })
})
