import 'server-only'

import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'

import { parseApiUrl } from '@/config/api'
import { env } from '@/config/env'
import type { AuditEventType } from '@/domain/audit/types'
import { SESSION_COOKIE, createBackendGet } from '@/lib/backend'
import { loadAuditEvents } from '@/repositories/audit'

// Its own backendGet, like services/posts: a service does not import
// another service (005 deviation 7).
const backendGet = createBackendGet(
  parseApiUrl({ raw: env.apiBaseUrl }),
  (...a) => fetch(...a)
)

/**
 * One page of the audit log for the current admin. Redirects to login if
 * the session ended after the page guard (requireRole) ran.
 */
export const requireAuditEvents = async (
  type: AuditEventType | null,
  page: number
) => {
  const cookie = (await cookies()).get(SESSION_COOKIE)?.value ?? ''
  const result = await loadAuditEvents(type, page, cookie, backendGet)
  if (result.kind === 'login') redirect(result.url)
  return result
}
