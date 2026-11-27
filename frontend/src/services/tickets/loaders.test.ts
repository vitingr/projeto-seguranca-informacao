import { describe, expect, it } from 'vitest'

import { readListQuery } from '@/domain/ticket-query'
import { createBackendGet } from '@/lib/backend'
import { errorResponse, fakeFetch, jsonResponse } from '@/test/fake-fetch'

import { createTicketLoaders } from './loaders'

const API = 'https://api.test'
const COOKIE = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
const at = '2026-11-26T17:04:00Z'

class Redirect extends Error {
  constructor(readonly url: string) {
    super(`redirect ${url}`)
  }
}
class NotFound extends Error {}

const setup = (routes: Parameters<typeof fakeFetch>[0], cookie = COOKIE) => {
  const fake = fakeFetch(routes)
  const loaders = createTicketLoaders({
    cookie: async () => cookie,
    get: createBackendGet(API, fake.fetch),
    notFound: () => {
      throw new NotFound()
    },
    redirect: url => {
      throw new Redirect(url)
    }
  })
  return { calls: fake.calls, loaders }
}

const emptyPage = { items: [], page: 1, page_size: 20, pages: 1, total: 0 }

const detail = {
  created_at: at,
  data: { descricao: 'Abrir a porta', titulo: 'Liberar porta' },
  id: 12,
  status: 'aberto',
  title: 'Liberar porta',
  type: 'outro',
  updated_at: at
}

describe('createTicketLoaders', () => {
  it('forwards the cookie and sends only the parsed codes', async () => {
    const query = readListQuery(
      { pagina: '2', status: 'aberto', tipo: 'xyz', usuario: 'maria' },
      'own'
    )
    const { calls, loaders } = setup({
      [`GET ${API}/api/tickets?status=aberto&pagina=2`]: [
        jsonResponse(200, emptyPage)
      ]
    })
    expect(await loaders.loadOwnTickets(query)).toMatchObject({
      data: { items: [], total: 0 },
      kind: 'ok'
    })
    expect(calls[0]?.headers.cookie).toBe(`__Host-session=${COOKIE}`)
  })

  it('the queue sends the user filter', async () => {
    const query = readListQuery({ usuario: 'maria.souza' }, 'queue')
    const { loaders } = setup({
      [`GET ${API}/api/administradores/tickets?usuario=maria.souza`]: [
        jsonResponse(200, {
          ...emptyPage,
          status_counts: {
            aberto: 0,
            em_andamento: 0,
            fechado: 0,
            resolvido: 0
          }
        })
      ]
    })
    expect((await loaders.loadQueue(query)).kind).toBe('ok')
  })

  it('reads a ticket', async () => {
    const { loaders } = setup({
      [`GET ${API}/api/tickets/12`]: [jsonResponse(200, detail)]
    })
    expect(await loaders.loadOwnTicket('12')).toMatchObject({
      data: { data: detail.data, id: 12 },
      kind: 'ok'
    })
  })

  it('401 → login with the expired notice and the page as next', async () => {
    const { loaders } = setup({
      [`GET ${API}/api/tickets/12/mensagens`]: [
        errorResponse(401, 'unauthorized')
      ]
    })
    await expect(loaders.loadOwnThread('12')).rejects.toMatchObject({
      url: '/login?next=%2Fusuarios%2Ftickets%2F12&sessao=expirada'
    })
  })

  it('no valid session cookie → login, without calling the backend', async () => {
    const { calls, loaders } = setup({}, 'tampered; x=1')
    await expect(
      loaders.loadOwnTickets(readListQuery({}, 'own'))
    ).rejects.toMatchObject({ url: '/login?next=%2Fusuarios' })
    expect(calls).toHaveLength(0)
  })

  it('404 and a malformed id → not found', async () => {
    const { calls, loaders } = setup({
      [`GET ${API}/api/administradores/tickets/99`]: [
        errorResponse(404, 'not_found')
      ]
    })
    for (const load of [
      () => loaders.loadAdminTicket('99'),
      () => loaders.loadAdminTicket('012'),
      () => loaders.loadOwnTicket('abc')
    ]) {
      await expect(load()).rejects.toBeInstanceOf(NotFound)
    }
    expect(calls).toHaveLength(1)
  })

  it('403 → the forbidden screen', async () => {
    const { loaders } = setup({
      [`GET ${API}/api/administradores/tickets/autores`]: [
        errorResponse(403, 'forbidden')
      ]
    })
    expect(await loaders.loadQueueAuthors()).toEqual({ kind: 'forbidden' })
  })

  it('500, an answer that does not parse and no answer → the error', async () => {
    const { loaders } = setup({
      [`GET ${API}/api/tickets`]: [
        errorResponse(500, 'internal', { request_id: 'req-1' }),
        jsonResponse(200, { ...emptyPage, extra: 1 }),
        new TypeError('fetch failed')
      ]
    })
    const own = readListQuery({}, 'own')
    expect(await loaders.loadOwnTickets(own)).toEqual({
      kind: 'error',
      requestId: 'req-1'
    })
    expect(await loaders.loadOwnTickets(own)).toEqual({ kind: 'error' })
    expect(await loaders.loadOwnTickets(own)).toEqual({ kind: 'error' })
  })
})
