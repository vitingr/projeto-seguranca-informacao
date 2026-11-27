// Routes, the role each protected route needs, and the login redirect URLs.
// Used by the server guard and the browser client.

import { LOGIN_PATH } from '@/domain/login-path'
import {
  PROTECTED_ROUTES,
  matchProtectedRoute
} from '@/domain/protected-routes'
import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { Role } from '@/domain/roles/types'
import type { Session } from '@/domain/session/types'

/** Where a user goes after login when there is no valid `next`. */
export const landingFor = (role: Role): ProtectedRoute =>
  role === 'admin' ? '/administradores' : '/usuarios'

/**
 * The page to open after login. `next` is accepted only when it is exactly
 * one of the protected routes, or a ticket page with a well-formed number
 * (spec 003 S-15), and that route allows the user's role; any other value
 * (absolute URL, `//host`, `/login`, unknown path) is ignored, so the login
 * page cannot redirect anywhere else.
 */
export const safeNext = (next: unknown, role: Role): string => {
  if (typeof next === 'string') {
    const route = matchProtectedRoute(next)
    if (route !== null && PROTECTED_ROUTES[route] === role) return next
  }
  return landingFor(role)
}

/**
 * Where `/` sends each visitor (spec AC-0.6): there is no public home, so
 * an anonymous visitor or an expired session goes to the login page and a
 * user goes to the landing page of their role.
 */
export const rootRedirect = (
  session: Session
): ProtectedRoute | typeof LOGIN_PATH =>
  session.kind === 'user' ? landingFor(session.user.role) : LOGIN_PATH

/** The login page, remembering where the user was. */
export const loginUrl = (next: string, { expired = false } = {}): string => {
  const query = new URLSearchParams({ next })
  if (expired) query.set('sessao', 'expirada')
  return `${LOGIN_PATH}?${query}`
}
