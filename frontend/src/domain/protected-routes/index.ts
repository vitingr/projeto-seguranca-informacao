import type { Role } from '@/domain/roles/types'

export const PROTECTED_ROUTES = {
  '/administradores': 'admin',
  '/cadastro': 'admin',
  '/usuarios': 'comum'
} as const satisfies Record<string, Role>
