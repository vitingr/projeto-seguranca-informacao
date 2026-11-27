import { LOGIN_PATTERN } from '@/domain/credentials'
import { TICKET_STATUS_LABELS, TICKET_TYPES } from '@/domain/tickets'
import type { TicketStatus, TicketType } from '@/domain/tickets/types'

import type {
  ListQuery,
  ListQueryChange,
  ListScope,
  SearchParams,
  TicketTab
} from './types'

export const LIST_PATHS = {
  own: '/usuarios',
  queue: '/administradores'
} as const

// The highest page the backend accepts
export const MAX_PAGE = 10_000

const STATUS_CODES: readonly string[] = Object.keys(TICKET_STATUS_LABELS)
const TYPE_CODES: readonly string[] = TICKET_TYPES.map(type => type.code)
const PAGE = /^[1-9]\d{0,4}$/

const single = (params: SearchParams, key: string): null | string => {
  const value = params[key]
  return typeof value === 'string' && value !== '' ? value : null
}

const isStatus = (value: null | string): value is TicketStatus =>
  value !== null && STATUS_CODES.includes(value)

const isType = (value: null | string): value is TicketType =>
  value !== null && TYPE_CODES.includes(value)

const isLogin = (value: null | string): value is string =>
  value !== null && LOGIN_PATTERN.test(value)

const isPage = (value: number) =>
  Number.isInteger(value) && value >= 1 && value <= MAX_PAGE

export const readListQuery = (
  params: SearchParams,
  scope: ListScope
): ListQuery => {
  const status = single(params, 'status')
  const type = single(params, 'tipo')
  const user = single(params, 'usuario')
  const page = single(params, 'pagina')
  const pageNumber = page !== null && PAGE.test(page) ? Number(page) : 1
  return {
    page: isPage(pageNumber) ? pageNumber : 1,
    scope,
    status: isStatus(status) ? status : null,
    type: isType(type) ? type : null,
    user: scope === 'queue' && isLogin(user) ? user : null
  }
}

export const listQueryHref = (
  query: ListQuery,
  change: ListQueryChange = {}
): string => {
  const filterChanged =
    'status' in change || 'type' in change || 'user' in change
  const next = {
    ...query,
    ...change,
    page: change.page ?? (filterChanged ? 1 : query.page)
  }
  const params = new URLSearchParams()
  if (isStatus(next.status)) params.set('status', next.status)
  if (isType(next.type)) params.set('tipo', next.type)
  if (query.scope === 'queue' && isLogin(next.user)) {
    params.set('usuario', next.user)
  }
  if (isPage(next.page) && next.page > 1) {
    params.set('pagina', String(next.page))
  }
  const search = params.toString()
  return search === ''
    ? LIST_PATHS[query.scope]
    : `${LIST_PATHS[query.scope]}?${search}`
}

export const TICKET_TABS = ['detalhes', 'conversa'] as const

export const readTicketTab = (aba: unknown): TicketTab =>
  aba === 'conversa' ? 'conversa' : 'detalhes'

// Only a same-origin list path is accepted, rebuilt from its known keys.
export const backToListHref = (de: unknown, scope: ListScope): string => {
  const list = LIST_PATHS[scope]
  if (typeof de !== 'string' || !de.startsWith(`${list}?`)) return list
  const params = Object.fromEntries(
    new URLSearchParams(de.slice(list.length + 1))
  )
  return listQueryHref(readListQuery(params, scope))
}
