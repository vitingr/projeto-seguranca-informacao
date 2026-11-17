import 'server-only'

import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { cache } from 'react'

import { parseApiUrl } from '@/config/api'
import type { ProtectedRoute } from '@/shared/types/navigation/protected-route'
import { loginUrl } from '@/shared/utils/navigation'

import { SESSION_COOKIE, createBackendGet, isSessionValue } from './backend'
import {
  type AreaPath,
  type CurrentUser,
  type Session,
  decideAccess,
  loadArea,
  loadSession
} from './session-rules'

const backendGet = createBackendGet(parseApiUrl(process.env.API_URL), (...a) =>
  fetch(...a)
)

const sessionCookie = async () => (await cookies()).get(SESSION_COOKIE)?.value

/** The current session, checked once per request (React cache). */
export const getSession = cache(async (): Promise<Session> =>
  loadSession(await sessionCookie(), backendGet)
)

/**
 * Guard for a protected page: redirects to login when there is no valid
 * session, and tells the page whether the role is allowed. Call it in each
 * page, not in a layout (layouts do not re-render on navigation).
 */
export async function requireRole(
  route: ProtectedRoute
): Promise<{ allowed: boolean; user: CurrentUser }> {
  const access = decideAccess(await getSession(), route)
  if (access.kind === 'login') redirect(access.url)
  return { allowed: access.kind === 'allowed', user: access.user }
}

/** Loads a role area; redirects to login if the session ended meanwhile. */
export async function requireArea(path: AreaPath, route: ProtectedRoute) {
  const cookie = await sessionCookie()
  if (!isSessionValue(cookie)) {
    redirect(loginUrl(route))
  }
  const result = await loadArea(path, route, cookie, backendGet)
  if (result.kind === 'login') redirect(result.url)
  return result
}
