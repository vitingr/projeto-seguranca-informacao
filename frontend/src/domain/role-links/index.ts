import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { Role } from '@/domain/roles/types'

export const ROLE_LINKS: Record<
  Role,
  { href: ProtectedRoute; label: string }[]
> = {
  admin: [
    { href: '/administradores', label: 'Área do administrador' },
    { href: '/cadastro', label: 'Cadastrar usuários' }
  ],
  comum: [{ href: '/usuarios', label: 'Minha conta' }]
} as const
