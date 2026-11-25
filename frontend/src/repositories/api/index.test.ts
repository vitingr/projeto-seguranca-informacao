import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { createCsrfStore } from '@/lib/api/csrf'
import {
  type Reply,
  csrfReply,
  emptyResponse,
  errorResponse,
  fakeFetch,
  jsonResponse,
  settled,
  textResponse
} from '@/test/fake-fetch'

import { createApiClient } from '.'

const CSRF = 'GET /api/csrf'
const LOGIN = 'POST /api/login'
const LOGOUT = 'POST /api/logout'
const CADASTRO = 'POST /api/cadastro'
const TICKETS = 'POST /api/tickets'

const cadastro = {
  login: 'maria',
  papel: 'comum',
  senha: 'a'.repeat(15)
} as const
const user = { login: 'maria', role: 'comum', user_id: 2 }

const setup = (routes: Record<string, Reply[]>, path = '/cadastro') => {
  const f = fakeFetch(routes)
  const navigations: string[] = []
  const csrf = createCsrfStore({ fetchFn: f.fetch })
  const client = createApiClient({
    csrf,
    fetch: f.fetch,
    navigation: {
      currentPath: () => path,
      go: url => navigations.push(`go ${url}`),
      replace: url => navigations.push(`replace ${url}`)
    }
  })
  const sequence = () => f.calls.map(c => `${c.method} ${c.url}`)
  const tokens = () =>
    f.calls.filter(c => c.method === 'POST').map(c => c.headers['x-csrf-token'])
  return { calls: f.calls, client, csrf, navigations, sequence, tokens }
}

// The token must only ever live in memory (D-3): any storage access throws.
const storageNames = ['localStorage', 'sessionStorage', 'indexedDB', 'document']
beforeEach(() => {
  for (const name of storageNames) {
    Object.defineProperty(globalThis, name, {
      configurable: true,
      get() {
        throw new Error(`${name} must not be used`)
      }
    })
  }
})
afterEach(() => {
  for (const name of storageNames) {
    Reflect.deleteProperty(globalThis, name)
  }
})

describe('register', () => {
  it('fetches a token first and sends it with the JSON body', async () => {
    const s = setup({
      [CADASTRO]: [jsonResponse(201, user)],
      [CSRF]: [csrfReply()]
    })
    expect(await s.client.register(cadastro)).toEqual({ data: user, ok: true })
    expect(s.sequence()).toEqual([CSRF, CADASTRO])
    const post = s.calls[1]
    expect(post?.headers['x-csrf-token']).toMatch(/^token-/)
    expect(post?.headers['content-type']).toBe('application/json')
    expect(post?.body).toEqual(cadastro)
  })

  it('reuses the token on the next call', async () => {
    const s = setup({
      [CADASTRO]: [jsonResponse(201, user), jsonResponse(201, user)],
      [CSRF]: [csrfReply()]
    })
    await s.client.register(cadastro)
    await s.client.register(cadastro)
    expect(s.sequence()).toEqual([CSRF, CADASTRO, CADASTRO])
  })

  it.each([
    [409, 'conflict'],
    [422, 'unprocessable_entity'],
    [400, 'bad_request'],
    [500, 'internal']
  ])('returns %i %s as an error', async (status, code) => {
    const s = setup({
      [CADASTRO]: [errorResponse(status, code)],
      [CSRF]: [csrfReply()]
    })
    const result = await s.client.register(cadastro)
    expect(result).toMatchObject({ error: { code, status }, ok: false })
    expect(s.navigations).toEqual([])
  })

  it('on 403 refreshes the token and retries once', async () => {
    const s = setup({
      [CADASTRO]: [errorResponse(403, 'forbidden'), jsonResponse(201, user)],
      [CSRF]: [csrfReply(), csrfReply()]
    })
    expect(await s.client.register(cadastro)).toEqual({ data: user, ok: true })
    expect(s.sequence()).toEqual([CSRF, CADASTRO, CSRF, CADASTRO])
    const [first, second] = s.tokens()
    expect(second).not.toBe(first)
  })

  it('does not retry a second 403', async () => {
    const s = setup({
      [CADASTRO]: [
        errorResponse(403, 'forbidden'),
        errorResponse(403, 'forbidden')
      ],
      [CSRF]: [csrfReply(), csrfReply()]
    })
    const result = await s.client.register(cadastro)
    expect(result).toMatchObject({ error: { code: 'forbidden' }, ok: false })
    expect(s.sequence().filter(c => c === CADASTRO)).toHaveLength(2)
  })

  it('expired session: 403 → refresh → 401 goes to login, nothing registered', async () => {
    const s = setup({
      [CADASTRO]: [
        errorResponse(403, 'forbidden'),
        errorResponse(401, 'unauthorized')
      ],
      [CSRF]: [csrfReply(), csrfReply()]
    })
    expect(await settled(s.client.register(cadastro))).toBe('pending')
    expect(s.navigations).toEqual([
      'go /login?next=%2Fcadastro&sessao=expirada'
    ])
    expect(s.sequence().filter(c => c === CADASTRO)).toHaveLength(2)
  })

  it('a 401 discards the token', async () => {
    const s = setup({
      [CADASTRO]: [errorResponse(401, 'unauthorized'), jsonResponse(201, user)],
      [CSRF]: [csrfReply(), csrfReply()]
    })
    await settled(s.client.register(cadastro))
    await s.client.register(cadastro)
    expect(s.sequence()).toEqual([CSRF, CADASTRO, CSRF, CADASTRO])
  })

  it.each([
    ['a failed fetch', new TypeError('fetch failed')],
    ['the proxy 500', textResponse(500, 'Internal Server Error')]
  ])('reports %s as a network error', async (_name, reply) => {
    const s = setup({ [CADASTRO]: [reply], [CSRF]: [csrfReply()] })
    const result = await s.client.register(cadastro)
    expect(result).toMatchObject({ error: { code: 'network' }, ok: false })
  })

  it('reports a failed token fetch as an error, without posting', async () => {
    const s = setup({ [CSRF]: [textResponse(500, 'Internal Server Error')] })
    const result = await s.client.register(cadastro)
    expect(result).toMatchObject({ error: { code: 'network' }, ok: false })
    expect(s.sequence()).toEqual([CSRF])
  })
})

describe('login', () => {
  const input = { login: 'admin', senha: 'secret-password-123' }
  const admin = { login: 'admin', role: 'admin', user_id: 1 }

  it('fetches a new token right after a successful login (002 D-4)', async () => {
    const s = setup({
      [CSRF]: [csrfReply(), csrfReply()],
      [LOGIN]: [jsonResponse(200, admin)]
    })
    expect(await s.client.login(input)).toEqual({ data: admin, ok: true })
    expect(s.sequence()).toEqual([CSRF, LOGIN, CSRF])
  })

  it('the old token is never sent again after login', async () => {
    const s = setup({
      [CADASTRO]: [jsonResponse(201, user)],
      [CSRF]: [csrfReply(), csrfReply()],
      [LOGIN]: [jsonResponse(200, admin)]
    })
    await s.client.login(input)
    await s.client.register(cadastro)
    const [loginToken, registerToken] = s.tokens()
    expect(registerToken).not.toBe(loginToken)
  })

  it('a 401 is the answer (bad credentials), not a lost session', async () => {
    const s = setup({
      [CSRF]: [csrfReply()],
      [LOGIN]: [errorResponse(401, 'unauthorized')]
    })
    const result = await s.client.login(input)
    expect(result).toMatchObject({ error: { code: 'unauthorized' }, ok: false })
    expect(s.navigations).toEqual([])
  })

  it('an expired anonymous session is fixed by the 403 retry', async () => {
    const s = setup({
      [CSRF]: [csrfReply(), csrfReply(), csrfReply()],
      [LOGIN]: [errorResponse(403, 'forbidden'), jsonResponse(200, admin)]
    })
    expect(await s.client.login(input)).toEqual({ data: admin, ok: true })
    expect(s.sequence()).toEqual([CSRF, LOGIN, CSRF, LOGIN, CSRF])
  })
})

describe('logout', () => {
  it.each([
    ['204', emptyResponse(204)],
    ['401 (already ended)', errorResponse(401, 'unauthorized')]
  ])(
    'on %s discards the token and replaces the page with /login',
    async (_n, reply) => {
      const s = setup({ [CSRF]: [csrfReply(), csrfReply()], [LOGOUT]: [reply] })
      expect(await settled(s.client.logout())).toBe('pending')
      expect(s.navigations).toEqual(['replace /login'])
      expect(s.calls[1]?.headers['x-csrf-token']).toMatch(/^token-/)
      expect(s.calls[1]?.body).toBeUndefined()
      // The token is gone: the next call fetches a new one.
      await s.csrf.get()
      expect(s.sequence()).toEqual([CSRF, LOGOUT, CSRF])
    }
  )

  it('returns other errors and stays on the page', async () => {
    const s = setup({
      [CSRF]: [csrfReply()],
      [LOGOUT]: [errorResponse(500, 'internal')]
    })
    const result = await s.client.logout()
    expect(result).toMatchObject({ error: { code: 'internal' }, ok: false })
    expect(s.navigations).toEqual([])
  })
})

describe('createTicket', () => {
  const BUILDER = '/usuarios/tickets/novo'
  const input = {
    dados: {
      justificativa: 'Preciso ver os painéis da equipe.',
      nivel_acesso: 'leitura',
      sistema: 'Grafana'
    },
    tipo: 'acesso'
  } as const
  const ticket = {
    created_at: '2026-11-25T14:30:00.123456Z',
    id: 7,
    status: 'aberto',
    title: 'Acesso a Grafana',
    type: 'acesso',
    updated_at: '2026-11-25T14:30:00.123456Z'
  }

  it('201: sends the body with the token and returns the ticket', async () => {
    const s = setup(
      { [CSRF]: [csrfReply()], [TICKETS]: [jsonResponse(201, ticket)] },
      BUILDER
    )
    expect(await s.client.createTicket(input)).toEqual({
      data: ticket,
      ok: true
    })
    expect(s.sequence()).toEqual([CSRF, TICKETS])
    const post = s.calls[1]
    expect(post?.headers['x-csrf-token']).toMatch(/^token-/)
    expect(post?.headers['content-type']).toBe('application/json')
    expect(post?.body).toEqual(input)
    expect(s.navigations).toEqual([])
  })

  it.each([
    [422, 'unprocessable_entity'],
    [500, 'internal']
  ])('%i: returns the error with its request_id', async (status, code) => {
    const s = setup(
      {
        [CSRF]: [csrfReply()],
        [TICKETS]: [errorResponse(status, code, { request_id: 'req-42' })]
      },
      BUILDER
    )
    expect(await s.client.createTicket(input)).toEqual({
      error: { code, message: `msg ${code}`, requestId: 'req-42', status },
      ok: false
    })
    expect(s.sequence()).toEqual([CSRF, TICKETS])
    expect(s.navigations).toEqual([])
  })

  it('429: returns the error and never retries', async () => {
    const s = setup(
      {
        [CSRF]: [csrfReply()],
        [TICKETS]: [errorResponse(429, 'too_many_requests')]
      },
      BUILDER
    )
    const result = await s.client.createTicket(input)
    expect(result).toMatchObject({
      error: { code: 'too_many_requests', status: 429 },
      ok: false
    })
    // A second reply would be "unexpected request": only one POST was sent.
    expect(s.sequence()).toEqual([CSRF, TICKETS])
    expect(s.navigations).toEqual([])
  })

  it('403: refreshes the token and retries once', async () => {
    const s = setup(
      {
        [CSRF]: [csrfReply(), csrfReply()],
        [TICKETS]: [errorResponse(403, 'forbidden'), jsonResponse(201, ticket)]
      },
      BUILDER
    )
    expect(await s.client.createTicket(input)).toEqual({
      data: ticket,
      ok: true
    })
    expect(s.sequence()).toEqual([CSRF, TICKETS, CSRF, TICKETS])
    const [first, second] = s.tokens()
    expect(second).not.toBe(first)
  })

  it('403 twice (lost permission): returns forbidden after one retry', async () => {
    const s = setup(
      {
        [CSRF]: [csrfReply(), csrfReply()],
        [TICKETS]: [
          errorResponse(403, 'forbidden'),
          errorResponse(403, 'forbidden')
        ]
      },
      BUILDER
    )
    const result = await s.client.createTicket(input)
    expect(result).toMatchObject({ error: { code: 'forbidden' }, ok: false })
    expect(s.sequence().filter(c => c === TICKETS)).toHaveLength(2)
  })

  it('401: discards the token and goes to login with sessao=expirada', async () => {
    const s = setup(
      {
        [CSRF]: [csrfReply(), csrfReply()],
        [TICKETS]: [errorResponse(401, 'unauthorized')]
      },
      BUILDER
    )
    expect(await settled(s.client.createTicket(input))).toBe('pending')
    expect(s.navigations).toEqual([
      'go /login?next=%2Fusuarios%2Ftickets%2Fnovo&sessao=expirada'
    ])
    // The token is gone: the next call fetches a new one.
    await s.csrf.get()
    expect(s.sequence()).toEqual([CSRF, TICKETS, CSRF])
  })

  it.each([
    ['a failed fetch', new TypeError('fetch failed')],
    ['the proxy 500', textResponse(500, 'Internal Server Error')]
  ])('reports %s as a network error', async (_name, reply) => {
    const s = setup({ [CSRF]: [csrfReply()], [TICKETS]: [reply] }, BUILDER)
    const result = await s.client.createTicket(input)
    expect(result).toMatchObject({ error: { code: 'network' }, ok: false })
    expect(s.sequence()).toEqual([CSRF, TICKETS])
  })
})
