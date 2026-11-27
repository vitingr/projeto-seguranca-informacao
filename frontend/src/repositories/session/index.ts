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

const parseMe = (body: unknown): CurrentUser | null => {
  if (typeof body !== 'object' || body === null) return null
  const {
    created_at: createdAt,
    login,
    role,
    user_id: userId
  } = body as Record<string, unknown>
  if (typeof userId !== 'number' || !Number.isInteger(userId) || userId <= 0) {
    return null
  }
  if (!ROLES.includes(role as Role)) return null
  return {
    role: role as Role,
    userId,
    ...(typeof login === 'string' && login !== '' ? { login } : {}),
    ...(typeof createdAt === 'string' && !Number.isNaN(Date.parse(createdAt))
      ? { createdAt }
      : {})
  }
}

export const loadSession = async (
  cookie: string | undefined,
  get: BackendGet
): Promise<Session> => {
  if (!isSessionValue(cookie)) return { kind: 'anonymous' }
  const answer = await get('/api/me', cookie)
  if (answer.status === 401) return { kind: 'expired' }
  const user = answer.status === 200 ? parseMe(answer.body) : null
  if (!user) {
    throw new Error(`unexpected GET /api/me answer: ${answer.status}`)
  }
  return { kind: 'user', user }
}

export const loadArea = async (
  path: AreaPath,
  route: ProtectedRoute,
  cookie: string,
  get: BackendGet
): Promise<AreaResult> => {
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
