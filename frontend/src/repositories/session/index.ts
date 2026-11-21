import { loginUrl } from '@/domain/navigation'
import type { ProtectedRoute } from '@/domain/protected-routes/types'
import { ROLES } from '@/domain/roles'
import type { Role } from '@/domain/roles/types'
import type {
  AreaPath,
  AreaResult,
  CurrentUser,
  Session
} from '@/domain/session/types'
import { isSessionValue } from '@/lib/backend'
import type { BackendGet } from '@/lib/backend/types'

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
