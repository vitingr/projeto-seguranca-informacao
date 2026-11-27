// Parsers of the ticket tracking answers (spec 003), from the backend JSON
// to domain types. They fail closed (S-13, AC-1.10): a missing or extra
// key, an unknown code or `kind`, a malformed date, a `data` that is not
// the key set of its type, or an admin field in a `comum` answer makes the
// whole answer `null`, never a partly shown one.

import { LOGIN_PATTERN } from '@/domain/credentials'
import type {
  StatusChange,
  StatusCounts,
  ThreadEntry,
  ThreadPage,
  TicketDetail,
  TicketListPage,
  TicketStatus,
  TicketSummary,
  TicketType
} from '@/domain/tickets/types'
import {
  ACCESS_LEVELS,
  ADMIN_TICKET_DETAIL_FIELDS,
  ADMIN_TICKET_SUMMARY_FIELDS,
  AUTHOR_LIST_FIELDS,
  ENTRY_SIDES,
  ENVIRONMENTS,
  QUEUE_PAGE_FIELDS,
  SEVERITIES,
  STATUS_CHANGE_RESPONSE_FIELDS,
  THREAD_PAGE_FIELDS,
  THREAD_PAGE_SIZE,
  TICKET_DATA_FIELDS,
  TICKET_DETAIL_FIELDS,
  TICKET_PAGE_FIELDS,
  TICKET_PAGE_SIZE,
  TICKET_STATUSES,
  TICKET_SUMMARY_FIELDS,
  TICKET_TYPES
} from '@/lib/api/contract'

import type { Fields, Scope } from './types'

const RFC3339 =
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/

const isRecord = (value: unknown): value is Fields =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

/** Exactly these keys, in any order. */
const hasKeys = (value: Fields, keys: readonly string[]) => {
  const own = Object.keys(value)
  return (
    own.length === keys.length && keys.every(key => Object.hasOwn(value, key))
  )
}

const isId = (value: unknown): value is number =>
  typeof value === 'number' && Number.isSafeInteger(value) && value >= 1

const isCount = (value: unknown): value is number =>
  typeof value === 'number' && Number.isSafeInteger(value) && value >= 0

const isDate = (value: unknown): value is string =>
  typeof value === 'string' &&
  RFC3339.test(value) &&
  !Number.isNaN(Date.parse(value))

const isLogin = (value: unknown): value is string =>
  typeof value === 'string' && LOGIN_PATTERN.test(value)

const oneOf =
  <const Code extends string>(codes: readonly Code[]) =>
  (value: unknown): value is Code =>
    typeof value === 'string' && (codes as readonly string[]).includes(value)

const isType = oneOf(TICKET_TYPES)
const isStatus = oneOf(TICKET_STATUSES)
const isSide = oneOf(ENTRY_SIDES)

/** The fixed-choice fields of `data` and their codes. */
const CHOICES: Record<string, (value: unknown) => boolean> = {
  ambiente: oneOf(ENVIRONMENTS),
  nivel_acesso: oneOf(ACCESS_LEVELS),
  severidade: oneOf(SEVERITIES)
}

/** Fields that may be `null` when left empty. */
const OPTIONAL = new Set(['observacoes'])

const parseData = (
  type: TicketType,
  value: unknown
): null | Record<string, null | string> => {
  const keys = TICKET_DATA_FIELDS[type]
  if (!isRecord(value) || !hasKeys(value, keys)) return null
  const data: Record<string, null | string> = {}
  for (const key of keys) {
    const field = value[key]
    if (field === null && OPTIONAL.has(key)) {
      data[key] = null
      continue
    }
    if (typeof field !== 'string') return null
    if (CHOICES[key] && !CHOICES[key](field)) return null
    data[key] = field
  }
  return data
}

const summaryFields = (scope: Scope) =>
  scope === 'queue' ? ADMIN_TICKET_SUMMARY_FIELDS : TICKET_SUMMARY_FIELDS

const parseSummary = (value: unknown, scope: Scope): null | TicketSummary => {
  if (!isRecord(value) || !hasKeys(value, summaryFields(scope))) return null
  const { author_login, created_at, id, status, title, type, updated_at } =
    value
  if (
    !isId(id) ||
    !isType(type) ||
    !isStatus(status) ||
    typeof title !== 'string' ||
    !isDate(created_at) ||
    !isDate(updated_at)
  ) {
    return null
  }
  if (scope === 'queue' && !isLogin(author_login)) return null
  return {
    createdAt: created_at,
    id,
    status,
    title,
    type,
    updatedAt: updated_at,
    ...(scope === 'queue' ? { authorLogin: author_login as string } : {})
  }
}

const parseCounts = (value: unknown): null | StatusCounts => {
  if (!isRecord(value) || !hasKeys(value, TICKET_STATUSES)) return null
  if (!TICKET_STATUSES.every(status => isCount(value[status]))) return null
  return value as StatusCounts
}

const parseListPage = (body: unknown, scope: Scope): null | TicketListPage => {
  const keys = scope === 'queue' ? QUEUE_PAGE_FIELDS : TICKET_PAGE_FIELDS
  if (!isRecord(body) || !hasKeys(body, keys)) return null
  const { items, page, page_size, pages, status_counts, total } = body
  if (
    !Array.isArray(items) ||
    items.length > TICKET_PAGE_SIZE ||
    !isId(page) ||
    !isId(pages) ||
    !isCount(total) ||
    page_size !== TICKET_PAGE_SIZE
  ) {
    return null
  }
  const rows = items.map(item => parseSummary(item, scope))
  if (rows.some(row => row === null)) return null
  const counts = scope === 'queue' ? parseCounts(status_counts) : undefined
  if (counts === null) return null
  return {
    items: rows as TicketSummary[],
    page,
    pageSize: page_size,
    pages,
    total,
    ...(counts ? { statusCounts: counts } : {})
  }
}

/** GET /api/tickets (`TicketPage`). */
export const parseTicketPage = (body: unknown) => parseListPage(body, 'own')

/** GET /api/administradores/tickets (`QueuePage`). */
export const parseQueuePage = (body: unknown) => parseListPage(body, 'queue')

/** GET /api/administradores/tickets/autores (`AuthorList`). */
export const parseAuthorList = (body: unknown): null | string[] => {
  if (!isRecord(body) || !hasKeys(body, AUTHOR_LIST_FIELDS)) return null
  const { logins } = body
  if (!Array.isArray(logins) || !logins.every(isLogin)) return null
  return logins
}

const parseDetail = (body: unknown, scope: Scope): null | TicketDetail => {
  const keys =
    scope === 'queue' ? ADMIN_TICKET_DETAIL_FIELDS : TICKET_DETAIL_FIELDS
  if (!isRecord(body) || !hasKeys(body, keys)) return null
  const { data, ...rest } = body
  const summary = parseSummary(rest, scope)
  if (!summary) return null
  const fields = parseData(summary.type, data)
  return fields ? { ...summary, data: fields } : null
}

/** GET /api/tickets/{id} (`TicketDetail`). */
export const parseTicketDetail = (body: unknown) => parseDetail(body, 'own')

/** GET /api/administradores/tickets/{id} (`AdminTicketDetail`). */
export const parseAdminTicketDetail = (body: unknown) =>
  parseDetail(body, 'queue')

const parseEntry = (value: unknown, scope: Scope): null | ThreadEntry => {
  if (!isRecord(value)) return null
  const admin = scope === 'queue' ? ['actor_login'] : []
  const { actor_login, created_at, id, kind, mine, side } = value
  if (
    !isId(id) ||
    !isSide(side) ||
    typeof mine !== 'boolean' ||
    !isDate(created_at) ||
    (scope === 'queue' && !isLogin(actor_login))
  ) {
    return null
  }
  const base = {
    createdAt: created_at,
    id,
    mine,
    side,
    ...(scope === 'queue' ? { actorLogin: actor_login as string } : {})
  }
  const common = ['id', 'kind', 'side', 'mine', 'created_at', ...admin]
  if (kind === 'message') {
    const { text } = value
    if (!hasKeys(value, [...common, 'text'])) return null
    if (typeof text !== 'string' || text === '') return null
    return { ...base, kind, text }
  }
  if (kind === 'status_change') {
    const { from, to } = value
    if (!hasKeys(value, [...common, 'from', 'to'])) return null
    if (!isStatus(from) || !isStatus(to)) return null
    return { ...base, from: from as TicketStatus, kind, to: to as TicketStatus }
  }
  return null
}

const parseThread = (body: unknown, scope: Scope): null | ThreadPage => {
  if (!isRecord(body) || !hasKeys(body, THREAD_PAGE_FIELDS)) return null
  const { entries, has_more } = body
  if (
    !Array.isArray(entries) ||
    entries.length > THREAD_PAGE_SIZE ||
    typeof has_more !== 'boolean'
  ) {
    return null
  }
  const parsed = entries.map(entry => parseEntry(entry, scope))
  if (parsed.some(entry => entry === null)) return null
  return { entries: parsed as ThreadEntry[], hasMore: has_more }
}

/** GET /api/tickets/{id}/mensagens (`ThreadPage`). */
export const parseThreadPage = (body: unknown) => parseThread(body, 'own')

/** GET /api/administradores/tickets/{id}/mensagens (`AdminThreadPage`). */
export const parseAdminThreadPage = (body: unknown) =>
  parseThread(body, 'queue')

/** POST …/mensagens → 201 (`ThreadEntry` / `AdminThreadEntry`). */
export const parseThreadEntry = (body: unknown, scope: Scope) =>
  parseEntry(body, scope)

/** PATCH /api/administradores/tickets/{id}/status (`StatusChangeResponse`). */
export const parseStatusChange = (body: unknown): null | StatusChange => {
  if (!isRecord(body) || !hasKeys(body, STATUS_CHANGE_RESPONSE_FIELDS)) {
    return null
  }
  const { id, status, updated_at } = body
  if (!isId(id) || !isStatus(status) || !isDate(updated_at)) return null
  return { id, status, updatedAt: updated_at }
}
