import type { Role } from '@/domain/roles/types'

import type { ProtectedRoute } from './types'

export const PROTECTED_ROUTES = {
  '/administradores': 'admin',
  '/administradores/tickets/[id]': 'admin',
  '/cadastro': 'admin',
  '/usuarios': 'comum',
  '/usuarios/tickets/[id]': 'comum',
  '/usuarios/tickets/novo': 'comum'
} as const satisfies Record<string, Role>

// Ticket number: no sign, no leading zero, at most 19 digits.
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
