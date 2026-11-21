import { describe, expect, it } from 'vitest'

import { ApiRequestError } from '@/lib/api/errors'
import {
  csrfReply,
  errorResponse,
  fakeFetch,
  jsonResponse,
  textResponse
} from '@/test/fake-fetch'

import { createCsrfStore } from '.'

const GET = 'GET /api/csrf'

describe('csrf store', () => {
  it('fetches the token once and keeps it in memory', async () => {
    const f = fakeFetch({ [GET]: [csrfReply()] })
    const store = createCsrfStore(f.fetch)
    const first = await store.get()
    expect(await store.get()).toBe(first)
    expect(f.calls).toHaveLength(1)
  })

  it('shares one request between concurrent callers', async () => {
    const f = fakeFetch({ [GET]: [csrfReply()] })
    const store = createCsrfStore(f.fetch)
    const [a, b] = await Promise.all([store.get(), store.get()])
    expect(a).toBe(b)
    expect(f.calls).toHaveLength(1)
  })

  it('refresh always fetches a new token', async () => {
    const f = fakeFetch({ [GET]: [csrfReply(), csrfReply()] })
    const store = createCsrfStore(f.fetch)
    const first = await store.get()
    const second = await store.refresh()
    expect(second).not.toBe(first)
    expect(await store.get()).toBe(second)
  })

  it('clear discards the token, and the next get fetches again', async () => {
    const f = fakeFetch({ [GET]: [csrfReply(), csrfReply()] })
    const store = createCsrfStore(f.fetch)
    const first = await store.get()
    store.clear()
    expect(await store.get()).not.toBe(first)
    expect(f.calls).toHaveLength(2)
  })

  it('clear discards a token that is still being fetched', async () => {
    const f = fakeFetch({ [GET]: [csrfReply(), csrfReply()] })
    const store = createCsrfStore(f.fetch)
    const inFlight = store.get()
    store.clear()
    const discarded = await inFlight
    expect(await store.get()).not.toBe(discarded)
  })

  it.each([
    ['a JSON error', errorResponse(500, 'internal'), 'internal'],
    ['plain text', textResponse(500, 'Internal Server Error'), 'network'],
    ['a body without a token', jsonResponse(200, { header: 'x' }), 'network'],
    ['a failed fetch', new TypeError('fetch failed'), 'network']
  ])('fails with %s, and the next get tries again', async (_n, reply, code) => {
    const f = fakeFetch({ [GET]: [reply, csrfReply()] })
    const store = createCsrfStore(f.fetch)
    const err = await store.get().catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiRequestError)
    expect((err as ApiRequestError).error.code).toBe(code)
    expect(await store.get()).toMatch(/^token-/)
  })

  it('asks the browser not to cache the token', async () => {
    let init: RequestInit | undefined
    const store = createCsrfStore(async (_url, i) => {
      init = i
      return csrfReply()
    })
    await store.get()
    expect(init?.cache).toBe('no-store')
  })
})
