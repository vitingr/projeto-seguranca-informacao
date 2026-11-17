import { ROLES, type Role } from '@/shared/api/types'
import { PROTECTED_ROUTES } from '@/shared/constants/navigation/protected-routes'
import type { ProtectedRoute } from '@/shared/types/navigation/protected-route'
import { loginUrl } from '@/shared/utils/navigation'

import { type BackendGet, isSessionValue } from './backend'

export interface CurrentUser {
  role: Role
  userId: number
}

export type Session =
  | { kind: 'anonymous' }
  | { kind: 'expired' }
  | { kind: 'user'; user: CurrentUser }

function parseMe(body: unknown): CurrentUser | null {
  if (typeof body !== 'object' || body === null) return null
  const { role, user_id: userId } = body as Record<string, unknown>
  if (typeof userId !== 'number' || !Number.isInteger(userId) || userId <= 0) {
    return null
  }
  if (!ROLES.includes(role as Role)) return null
  return { role: role as Role, userId }
}

/**
 * No (well-formed) cookie → anonymous, without calling the API. Otherwise
 * GET /api/me: 401 → expired, 200 → user. Anything else throws (fail closed:
 * the page shows an error, never its content).
 */
export async function loadSession(
  cookie: string | undefined,
  get: BackendGet
): Promise<Session> {
  if (!isSessionValue(cookie)) return { kind: 'anonymous' }
  const answer = await get('/api/me', cookie)
  if (answer.status === 401) return { kind: 'expired' }
  const user = answer.status === 200 ? parseMe(answer.body) : null
  if (!user) {
    throw new Error(`unexpected GET /api/me answer: ${answer.status}`)
  }
  return { kind: 'user', user }
}

export type Access =
  | { kind: 'allowed'; user: CurrentUser }
  | { kind: 'forbidden'; user: CurrentUser }
  | { kind: 'login'; url: string }

export function decideAccess(session: Session, route: ProtectedRoute): Access {
  switch (session.kind) {
    case 'anonymous':
      return { kind: 'login', url: loginUrl(route) }
    case 'expired':
      return { kind: 'login', url: loginUrl(route, { expired: true }) }
    case 'user':
      return session.user.role === PROTECTED_ROUTES[route]
        ? { kind: 'allowed', user: session.user }
        : { kind: 'forbidden', user: session.user }
  }
}

export type AreaPath = '/api/administradores' | '/api/usuarios'

export type AreaResult =
  | { area: string; kind: 'ok' }
  | { kind: 'forbidden' }
  | { kind: 'login'; url: string }

/**
 * The backend stays the authority: even after /api/me allowed the role,
 * the area endpoint can answer 403 (the role changed in between) or 401
 * (the session ended in between).
 */
export async function loadArea(
  path: AreaPath,
  route: ProtectedRoute,
  cookie: string,
  get: BackendGet
): Promise<AreaResult> {
  const answer = await get(path, cookie)
  if (answer.status === 401) {
    return { kind: 'login', url: loginUrl(route, { expired: true }) }
  }
  if (answer.status === 403) return { kind: 'forbidden' }
  const area = (answer.body as { area?: unknown } | null)?.area
  if (answer.status !== 200 || typeof area !== 'string') {
    throw new Error(`unexpected GET ${path} answer: ${answer.status}`)
  }
  return { area, kind: 'ok' }
}
