import Link from 'next/link'

import type { Role } from '@/shared/api/types'
import { LogoutButton } from '@/shared/components/logout-button'
import type { ProtectedRoute } from '@/shared/navigation/routes'

export const ROLE_LINKS: Record<
  Role,
  { href: ProtectedRoute; label: string }[]
> = {
  admin: [
    { href: '/admin', label: 'Área do administrador' },
    { href: '/cadastrar-usuarios', label: 'Cadastrar usuários' }
  ],
  comum: [{ href: '/minha-conta', label: 'Minha conta' }]
}

export const ROLE_NAMES: Record<Role, string> = {
  admin: 'administrador',
  comum: 'usuário comum'
}

/** Navigation for a logged-in user: the links of their role and logout. */
export const AppHeader = ({ role }: { role: Role }) => (
  <header className="border-border border-b">
    <nav
      aria-label="Principal"
      className="mx-auto flex w-full max-w-3xl flex-wrap items-center gap-4 px-4 py-3"
    >
      <Link className="font-semibold" href="/">
        Início
      </Link>
      {ROLE_LINKS[role].map(link => (
        <Link
          className="text-sm hover:underline"
          href={link.href}
          key={link.href}
        >
          {link.label}
        </Link>
      ))}
      <span className="text-muted-foreground ml-auto text-sm">
        Papel: {ROLE_NAMES[role]}
      </span>
      <LogoutButton />
    </nav>
  </header>
)
