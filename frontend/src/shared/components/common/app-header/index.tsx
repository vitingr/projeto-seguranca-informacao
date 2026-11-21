import Link from 'next/link'

import { ROLE_LINKS } from '@/domain/role-links'
import { ROLE_NAMES } from '@/domain/role-names'
import { LogoutButton } from '@/shared/components/common/logout-button'

import type { AppHeaderProps } from './types'

export const AppHeader: React.FC<AppHeaderProps> = ({ role }) => (
  <header className="border-border border-b">
    <nav
      aria-label="Principal"
      className="mx-auto flex w-full max-w-3xl flex-wrap items-center gap-4 px-4 py-3"
    >
      <Link className="font-semibold" href="/">
        Início
      </Link>
      {ROLE_LINKS[role].map((link, index: number) => (
        <Link
          className="text-sm hover:underline"
          href={link.href}
          key={`${link.href}-${index}`}
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
