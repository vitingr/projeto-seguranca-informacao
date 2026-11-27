import { LOGIN_PATH } from '@/domain/login-path'
import {
  PROTECTED_ROUTES,
  matchProtectedRoute
} from '@/domain/protected-routes'
import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { Role } from '@/domain/roles/types'
import type { Session } from '@/domain/session/types'

export const landingFor = (role: Role): ProtectedRoute =>
  role === 'admin' ? '/administradores' : '/usuarios'

// Only a known route of the user's role is accepted: no open redirect.
export const safeNext = (next: unknown, role: Role): string => {
  if (typeof next === 'string') {
    const route = matchProtectedRoute(next)
    if (route !== null && PROTECTED_ROUTES[route] === role) return next
  }
  return landingFor(role)
}

export const rootRedirect = (
  session: Session
): ProtectedRoute | typeof LOGIN_PATH =>
  session.kind === 'user' ? landingFor(session.user.role) : LOGIN_PATH

export const loginUrl = (next: string, { expired = false } = {}): string => {
  const query = new URLSearchParams({ next })
  if (expired) query.set('sessao', 'expirada')
  return `${LOGIN_PATH}?${query}`
}
