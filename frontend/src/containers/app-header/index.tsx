import Link from 'next/link'

import { BrandMark } from '@/components/brand'
import { Badge } from '@/components/ui/shadcn/badge'
import { AppNav } from '@/containers/app-nav'
import { LogoutButton } from '@/containers/logout-button'
import { ROLE_NAMES } from '@/domain/role-names'

import type { AppHeaderProps } from './types'

// Mobile: brand and actions on top, the links in a scrollable row below.
// From md: one row.
export const AppHeader: React.FC<AppHeaderProps> = ({ role }) => (
  <header className="bg-background/80 supports-backdrop-filter:bg-background/70 sticky top-0 z-40 border-b backdrop-blur-md">
    <div className="mx-auto flex w-full max-w-7xl flex-wrap items-center justify-between gap-x-4 gap-y-1 px-4 py-2 sm:px-6 md:flex-nowrap md:py-3 lg:px-8">
      <Link
        aria-label="Sistemas da Informação"
        className="focus-visible:ring-ring/50 rounded-lg outline-none focus-visible:ring-3"
        href="/"
      >
        <BrandMark />
      </Link>
      <div className="order-last w-full min-w-0 md:order-none md:w-auto md:flex-1">
        <AppNav role={role} />
      </div>
      <div className="flex items-center gap-3">
        <Badge className="h-6 px-2.5" variant="outline">
          <span
            aria-hidden="true"
            className="bg-success size-1.5 rounded-full"
          />
          Papel: {ROLE_NAMES[role]}
        </Badge>
        <LogoutButton />
      </div>
    </div>
  </header>
)
