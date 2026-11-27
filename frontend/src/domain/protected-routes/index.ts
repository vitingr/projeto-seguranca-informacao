import type { Role } from '@/domain/roles/types'

import type { ProtectedRoute } from './types'

export const PROTECTED_ROUTES = {
  '/administradores': 'admin',
  // The admin ticket page (spec 003-ticket-tracking, US-5).
  '/administradores/tickets/[id]': 'admin',
  '/cadastro': 'admin',
  '/usuarios': 'comum',
  // The user's ticket page (spec 003-ticket-tracking, US-2).
  '/usuarios/tickets/[id]': 'comum',
  // The ticket builder (spec 001-ticket-builder, US-1).
  '/usuarios/tickets/novo': 'comum'
} as const satisfies Record<string, Role>

/**
 * The dynamic routes and the paths they accept: a ticket number with no
 * sign and no leading zero, at most 19 digits (spec S-15, Q-3).
 */
const DYNAMIC_ROUTES = [
  {
    pattern: /^\/administradores\/tickets\/[1-9]\d{0,18}$/,
    route: '/administradores/tickets/[id]'
  },
  {
    pattern: /^\/usuarios\/tickets\/[1-9]\d{0,18}$/,
    route: '/usuarios/tickets/[id]'
  }
] as const satisfies readonly { pattern: RegExp; route: ProtectedRoute }[]

const isDynamic = (route: string) => route.includes('[')

/**
 * The protected route a path belongs to: an exact static route first, then
 * the dynamic patterns; `null` for any other path. A dynamic key itself
 * (`/usuarios/tickets/[id]`) is not a path and never matches.
 */
export const matchProtectedRoute = (
  pathname: string
): null | ProtectedRoute => {
  if (Object.hasOwn(PROTECTED_ROUTES, pathname) && !isDynamic(pathname)) {
    return pathname as ProtectedRoute
  }
  return (
    DYNAMIC_ROUTES.find(({ pattern }) => pattern.test(pathname))?.route ?? null
  )
}
