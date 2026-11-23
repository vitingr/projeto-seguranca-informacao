import type { ProtectedRoute } from '@/domain/protected-routes/types'
import type { Role } from '@/domain/roles/types'

export const ROLE_LINKS: Record<
  Role,
  { href: ProtectedRoute; label: string }[]
> = {
  admin: [
    { href: '/admin', label: 'Área do administrador' },
    { href: '/cadastrar-usuarios', label: 'Cadastrar usuários' },
    { href: '/mural', label: 'Mural' },
    { href: '/admin/auditoria', label: 'Auditoria' }
  ],
  comum: [
    { href: '/minha-conta', label: 'Minha conta' },
    { href: '/mural', label: 'Mural' }
  ]
} as const
