// Routes, the role each protected route needs, and the login redirect URLs.
// Used by the server guard and the browser client.

import type { Role } from '@/shared/api/types'

export const LOGIN_PATH = '/login'

export const PROTECTED_ROUTES = {
  '/admin': 'admin',
  '/cadastrar-usuarios': 'admin',
  '/minha-conta': 'comum'
} as const satisfies Record<string, Role>

export type ProtectedRoute = keyof typeof PROTECTED_ROUTES

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
export function safeNext(next: unknown, role: Role): ProtectedRoute {
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
export function loginUrl(next: string, { expired = false } = {}): string {
  const query = new URLSearchParams({ next })
  if (expired) query.set('sessao', 'expirada')
  return `${LOGIN_PATH}?${query}`
}
