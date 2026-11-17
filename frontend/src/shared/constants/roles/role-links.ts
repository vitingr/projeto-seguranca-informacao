import type { Role } from '@/shared/api/types'
import type { ProtectedRoute } from '@/shared/types/navigation/protected-route'

export const ROLE_LINKS: Record<
  Role,
  { href: ProtectedRoute; label: string }[]
> = {
  admin: [
    { href: '/admin', label: 'Área do administrador' },
    { href: '/cadastrar-usuarios', label: 'Cadastrar usuários' }
  ],
  comum: [{ href: '/minha-conta', label: 'Minha conta' }]
} as const
