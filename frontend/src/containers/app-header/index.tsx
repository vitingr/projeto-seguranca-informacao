import Link from 'next/link'

import { LogoutButton } from '@/containers/logout-button'
import { MobileMenu } from '@/containers/mobile-menu'
import { ROLE_LINKS } from '@/domain/role-links'
import { ROLE_NAMES } from '@/domain/role-names'

import type { AppHeaderProps } from './types'

// One line from 360 px: below `md` the role links and the role name move
// into the "Menu" panel; "Início" and "Sair" stay visible at every width.
export const AppHeader: React.FC<AppHeaderProps> = ({ role }) => (
  <header className="border-border relative border-b">
    <nav
      aria-label="Principal"
      className="mx-auto flex w-full max-w-3xl items-center justify-between gap-4 px-4 py-3"
    >
      <div className="flex items-center gap-4">
        <Link className="font-semibold" href="/">
          Início
        </Link>
        <div className="hidden items-center gap-4 md:flex">
          {ROLE_LINKS[role].map((link, index: number) => (
            <Link
              className="text-sm hover:underline"
              href={link.href}
              key={`${link.href}-${index}`}
            >
              {link.label}
            </Link>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-2 md:gap-4">
        <span className="text-muted-foreground hidden text-sm md:inline">
          Papel: {ROLE_NAMES[role]}
        </span>
        <MobileMenu links={ROLE_LINKS[role]} roleName={ROLE_NAMES[role]} />
        <LogoutButton />
      </div>
    </nav>
  </header>
)
