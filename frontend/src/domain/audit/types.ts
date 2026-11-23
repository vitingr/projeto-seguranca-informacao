import type { AUDIT_EVENT_TYPES } from '.'

export type AuditEventType = (typeof AUDIT_EVENT_TYPES)[number]

export interface AuditActor {
  /** Null when the user no longer exists. */
  login: string | null
  userId: number
}

export interface AuditTarget {
  id: number
  /** Only for a user that still exists. */
  login: string | null
  type: 'post' | 'user'
}

export interface AuditEvent {
  /** Null for a failed login. */
  actor: AuditActor | null
  /** ISO 8601 timestamp. */
  createdAt: string
  event: AuditEventType
  id: number
  /** The TCP peer the backend saw; null when it did not parse. */
  ip: string | null
  /** Null for a login, a logout or a failed login of an unknown login. */
  target: AuditTarget | null
}

export interface AuditPage {
  hasNext: boolean
  items: AuditEvent[]
  page: number
  pageSize: number
}

export type AuditResult =
  | { kind: 'forbidden' }
  | { kind: 'login'; url: string }
  | { kind: 'ok'; page: AuditPage }
