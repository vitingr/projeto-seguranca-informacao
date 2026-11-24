'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

import { ROLE_LINKS } from '@/domain/role-links'

import type { AppNavProps } from './types'

const LINK =
  'text-muted-foreground hover:text-foreground hover:bg-muted focus-visible:ring-ring/50 aria-[current=page]:bg-muted aria-[current=page]:text-foreground flex h-8 shrink-0 items-center rounded-full px-3 text-sm whitespace-nowrap outline-none focus-visible:ring-3 motion-safe:transition-colors'

// The header links. Only marks the current page (aria-current) for styling.
export const AppNav: React.FC<AppNavProps> = ({ role }) => {
  const pathname = usePathname()
  const links = [{ href: '/', label: 'Início' }, ...ROLE_LINKS[role]]

  return (
    <nav
      aria-label="Principal"
      className="-mx-1 flex min-w-0 items-center gap-1 overflow-x-auto px-1 py-1"
    >
      {links.map(link => (
        <Link
          aria-current={pathname === link.href ? 'page' : undefined}
          className={LINK}
          href={link.href}
          key={link.href}
        >
          {link.label}
        </Link>
      ))}
    </nav>
  )
}
