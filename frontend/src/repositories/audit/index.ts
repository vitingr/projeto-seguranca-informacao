import { isAuditEventType } from '@/domain/audit'
import type {
  AuditActor,
  AuditEvent,
  AuditEventType,
  AuditPage,
  AuditResult,
  AuditTarget
} from '@/domain/audit/types'
import { loginUrl } from '@/domain/navigation'
import { isSessionValue } from '@/lib/backend'
import type { BackendGet } from '@/lib/backend/types'

const AUDIT_ROUTE = '/admin/auditoria'

const isPositiveInteger = (value: unknown): value is number =>
  typeof value === 'number' && Number.isInteger(value) && value > 0

const isString = (value: unknown): value is string => typeof value === 'string'

const isObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null

/** An optional login: absent is null; anything but a string is invalid. */
const optionalLogin = (value: unknown): string | null | undefined =>
  value === undefined ? null : isString(value) ? value : undefined

const parseActor = (value: unknown): AuditActor | null | undefined => {
  if (value === null) return null
  if (!isObject(value)) return undefined
  const login = optionalLogin(value.login)
  if (!isPositiveInteger(value.user_id) || login === undefined) return undefined
  return { login, userId: value.user_id }
}

const parseTarget = (value: unknown): AuditTarget | null | undefined => {
  if (value === null) return null
  if (!isObject(value)) return undefined
  const { id, type } = value
  const login = optionalLogin(value.login)
  if (
    !isPositiveInteger(id) ||
    (type !== 'post' && type !== 'user') ||
    login === undefined
  ) {
    return undefined
  }
  return { id, login, type }
}

const parseEvent = (value: unknown): AuditEvent | null => {
  if (!isObject(value)) return null
  const { created_at, event, id, ip } = value
  const actor = parseActor(value.actor)
  const target = parseTarget(value.target)
  if (
    !isPositiveInteger(id) ||
    !isAuditEventType(event) ||
    !isString(created_at) ||
    !(ip === null || isString(ip)) ||
    actor === undefined ||
    target === undefined
  ) {
    return null
  }
  return { actor, createdAt: created_at, event, id, ip, target }
}

/** The GET /api/audit-events body as a domain page, or null when it is not. */
export const parseAuditPage = (value: unknown): AuditPage | null => {
  if (!isObject(value)) return null
  const { has_next, items, page, page_size } = value
  if (
    !Array.isArray(items) ||
    !isPositiveInteger(page) ||
    !isPositiveInteger(page_size) ||
    typeof has_next !== 'boolean'
  ) {
    return null
  }
  const events = items.map(parseEvent)
  if (events.some(e => e === null)) return null
  return {
    hasNext: has_next,
    items: events as AuditEvent[],
    page,
    pageSize: page_size
  }
}

/**
 * One page of the audit log, optionally of one event type. The backend
 * stays the authority (admin only, 006 S-3): 401 → login, 403 → forbidden.
 * Anything else that is not a valid 200 throws (fail closed).
 */
export const loadAuditEvents = async (
  type: AuditEventType | null,
  page: number,
  cookie: string,
  get: BackendGet
): Promise<AuditResult> => {
  if (!isSessionValue(cookie)) {
    return { kind: 'login', url: loginUrl(AUDIT_ROUTE) }
  }
  const query = new URLSearchParams({ page: String(page) })
  if (type) query.set('type', type)
  const answer = await get(`/api/audit-events?${query}`, cookie)
  if (answer.status === 401) {
    return { kind: 'login', url: loginUrl(AUDIT_ROUTE, { expired: true }) }
  }
  if (answer.status === 403) return { kind: 'forbidden' }
  const parsed = answer.status === 200 ? parseAuditPage(answer.body) : null
  if (!parsed) {
    throw new Error(`unexpected GET /api/audit-events answer: ${answer.status}`)
  }
  return { kind: 'ok', page: parsed }
}
