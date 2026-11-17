import type { Role } from '@/shared/api/types'

export const PROTECTED_ROUTES = {
  '/admin': 'admin',
  '/cadastrar-usuarios': 'admin',
  '/minha-conta': 'comum'
} as const satisfies Record<string, Role>
