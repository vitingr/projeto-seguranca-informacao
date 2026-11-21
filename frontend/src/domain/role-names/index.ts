import type { Role } from '@/domain/roles/types'

export const ROLE_NAMES: Record<Role, string> = {
  admin: 'administrador',
  comum: 'usuário comum'
} as const
