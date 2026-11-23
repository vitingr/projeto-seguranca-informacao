// The events of the audit log (006 D-4) and their Portuguese labels.

import type { AuditEventType } from './types'

/** The five events, in the order of the contract. */
export const AUDIT_EVENT_TYPES = [
  'login_succeeded',
  'login_failed',
  'logout',
  'user_created',
  'post_removed'
] as const

export const AUDIT_EVENT_LABELS: Record<AuditEventType, string> = {
  login_failed: 'Falha de login',
  login_succeeded: 'Login',
  logout: 'Logout',
  post_removed: 'Post removido',
  user_created: 'Usuário cadastrado'
}

export const isAuditEventType = (value: unknown): value is AuditEventType =>
  typeof value === 'string' &&
  (AUDIT_EVENT_TYPES as readonly string[]).includes(value)

/**
 * The type filter in the URL (`?tipo=`): one of the five events, or null
 * (every event) for anything else or no value. The backend answers 400 for
 * an unknown type, so an unknown value never reaches it.
 */
export const parseAuditTypeParam = (
  raw: string | string[] | undefined
): AuditEventType | null => (isAuditEventType(raw) ? raw : null)
