// Routes, the role each protected route needs, and the login redirect URLs.
// Used by the server guard and the browser client.

import { LOGIN_PATH } from '@/domain/login-path'
import { PROTECTED_ROUTES } from '@/domain/protected-routes'
import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { Role } from '@/domain/roles/types'

/** Where a user goes after login when there is no valid `next`. */
export const landingFor = (role: Role): ProtectedRoute =>
  role === 'admin' ? '/admin' : '/minha-conta'

const isProtectedRoute = (value: string): value is ProtectedRoute =>
  Object.hasOwn(PROTECTED_ROUTES, value)

/**
 * The page to open after login. `next` is accepted only when it is exactly
 * one of the protected routes and that route allows the user's role; any
 * other value (absolute URL, `//host`, `/login`, unknown path) is ignored,
 * so the login page cannot redirect anywhere else.
 */
export const safeNext = (next: unknown, role: Role): ProtectedRoute => {
  if (
    typeof next === 'string' &&
    isProtectedRoute(next) &&
    PROTECTED_ROUTES[next] === role
  ) {
    return next
  }
  return landingFor(role)
}

/** The login page, remembering where the user was. */
export const loginUrl = (next: string, { expired = false } = {}): string => {
  const query = new URLSearchParams({ next })
  if (expired) query.set('sessao', 'expirada')
  return `${LOGIN_PATH}?${query}`
}
