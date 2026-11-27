import 'server-only'

import { loginUrl } from '@/domain/navigation'
import { LIST_PATHS } from '@/domain/ticket-query'
import type { ListQuery } from '@/domain/ticket-query/types'
import { isSessionValue } from '@/lib/backend'
import {
  parseAdminThreadPage,
  parseAdminTicketDetail,
  parseAuthorList,
  parseQueuePage,
  parseThreadPage,
  parseTicketDetail,
  parseTicketPage
} from '@/repositories/tickets'

import type { LoadResult, TicketLoaderDeps } from './types'

const TICKET_ID = /^[1-9]\d{0,18}$/

const requestIdOf = (body: unknown): string | undefined => {
  const id = (body as null | { request_id?: unknown })?.request_id
  return typeof id === 'string' && id !== '' ? id : undefined
}

const listSearch = (query: ListQuery): string => {
  const params = new URLSearchParams()
  if (query.status) params.set('status', query.status)
  if (query.type) params.set('tipo', query.type)
  if (query.scope === 'queue' && query.user) params.set('usuario', query.user)
  if (query.page > 1) params.set('pagina', String(query.page))
  const search = params.toString()
  return search === '' ? '' : `?${search}`
}

export const createTicketLoaders = (deps: TicketLoaderDeps) => {
  const load = async <T>(
    apiPath: string,
    pagePath: string,
    parse: (body: unknown) => null | T,
    { ticket = false } = {}
  ): Promise<LoadResult<T>> => {
    const cookie = await deps.cookie()
    if (!isSessionValue(cookie)) deps.redirect(loginUrl(pagePath))
    let answer
    try {
      answer = await deps.get(apiPath, cookie)
    } catch {
      return { kind: 'error' }
    }
    if (answer.status === 401) {
      deps.redirect(loginUrl(pagePath, { expired: true }))
    }
    if (answer.status === 403) return { kind: 'forbidden' }
    if (answer.status === 404 && ticket) deps.notFound()
    if (answer.status === 200) {
      const data = parse(answer.body)
      if (data !== null) return { data, kind: 'ok' }
    }
    return { kind: 'error', requestId: requestIdOf(answer.body) }
  }

  const ticketId = (raw: string): string => {
    if (!TICKET_ID.test(raw)) deps.notFound()
    return raw
  }

  return {
    loadAdminThread: async (raw: string) => {
      const id = ticketId(raw)
      return load(
        `/api/administradores/tickets/${id}/mensagens`,
        `/administradores/tickets/${id}`,
        parseAdminThreadPage,
        { ticket: true }
      )
    },

    loadAdminTicket: async (raw: string) => {
      const id = ticketId(raw)
      return load(
        `/api/administradores/tickets/${id}`,
        `/administradores/tickets/${id}`,
        parseAdminTicketDetail,
        { ticket: true }
      )
    },

    loadOwnThread: async (raw: string) => {
      const id = ticketId(raw)
      return load(
        `/api/tickets/${id}/mensagens`,
        `/usuarios/tickets/${id}`,
        parseThreadPage,
        { ticket: true }
      )
    },

    loadOwnTicket: async (raw: string) => {
      const id = ticketId(raw)
      return load(
        `/api/tickets/${id}`,
        `/usuarios/tickets/${id}`,
        parseTicketDetail,
        { ticket: true }
      )
    },

    loadOwnTickets: (query: ListQuery) =>
      load(
        `/api/tickets${listSearch({ ...query, scope: 'own' })}`,
        LIST_PATHS.own,
        parseTicketPage
      ),

    loadQueue: (query: ListQuery) =>
      load(
        `/api/administradores/tickets${listSearch({ ...query, scope: 'queue' })}`,
        LIST_PATHS.queue,
        parseQueuePage
      ),

    loadQueueAuthors: () =>
      load(
        '/api/administradores/tickets/autores',
        LIST_PATHS.queue,
        parseAuthorList
      )
  }
}
