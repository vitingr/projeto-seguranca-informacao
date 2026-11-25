import type { Role } from '@/domain/roles/types'

export const PROTECTED_ROUTES = {
  '/administradores': 'admin',
  '/cadastro': 'admin',
  '/usuarios': 'comum',
  // The ticket builder (spec 001-ticket-builder, US-1).
  '/usuarios/tickets/novo': 'comum'
} as const satisfies Record<string, Role>
