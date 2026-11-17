import type { Role } from '@/shared/api/types'

export const ROLE_NAMES: Record<Role, string> = {
  admin: 'administrador',
  comum: 'usuário comum'
} as const
