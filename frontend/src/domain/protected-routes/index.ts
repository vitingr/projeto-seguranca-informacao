import type { Role } from '@/domain/roles/types'

import type { ProtectedRoute } from './types'

/** The roles each protected page allows. The backend checks them again. */
export const PROTECTED_ROUTES = {
  '/admin': ['admin'],
  '/admin/auditoria': ['admin'],
  '/cadastrar-usuarios': ['admin'],
  '/minha-conta': ['comum'],
  // Admins moderate from the wall (006 D-1).
  '/mural': ['admin', 'comum']
} as const satisfies Record<string, readonly Role[]>

/** Whether route lets role in. */
export const routeAllows = (route: ProtectedRoute, role: Role): boolean =>
  (PROTECTED_ROUTES[route] as readonly Role[]).includes(role)
