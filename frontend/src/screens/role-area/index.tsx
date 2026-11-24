import { cn } from 'cn'

import { AreaCard } from '@/components/area-card'
import { Forbidden } from '@/components/forbidden'
import { staggerAt } from '@/components/stagger'
import { AppHeader } from '@/containers/app-header'
import { ROLE_LINKS } from '@/domain/role-links'
import { AREA_DETAILS } from '@/screens/home/constants'
import { requireArea, requireRole } from '@/services/session'

import type { RoleAreaProps } from './types'

const RISE = 'motion-safe:animate-rise'

export const RoleArea: React.FC<RoleAreaProps> = async ({
  description,
  endpoint,
  route,
  title
}) => {
  const { allowed, user } = await requireRole(route)
  const area = allowed ? await requireArea(endpoint, route) : null

  if (area?.kind !== 'ok') {
    return (
      <>
        <AppHeader role={user.role} />
        <Forbidden role={user.role} />
      </>
    )
  }

  const shortcuts = ROLE_LINKS[user.role].filter(link => link.href !== route)

  return (
    <>
      <AppHeader role={user.role} />
      <main className="mx-auto flex w-full max-w-7xl flex-col gap-12 px-4 py-12 lg:py-20 xl:px-0">
        <section className="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:items-start lg:gap-10">
          <div className={cn(RISE, 'flex flex-col gap-4 lg:col-span-6')}>
            <h1 className="text-3xl font-semibold tracking-tight lg:text-4xl">
              {title}
            </h1>
            <p className="text-muted-foreground max-w-xl text-sm text-pretty lg:text-base">
              {description}
            </p>
          </div>
        </section>

        {shortcuts.length > 0 ? (
          <section aria-labelledby="shortcuts" className="flex flex-col gap-5">
            <ul
              className={cn(
                'grid grid-cols-1 gap-4 sm:grid-cols-2',
                shortcuts.length > 2 && 'lg:grid-cols-3'
              )}
            >
              {shortcuts.map((link, index) => (
                <li key={link.href}>
                  <AreaCard
                    {...AREA_DETAILS[link.href]}
                    className={cn(RISE, staggerAt(index + 2))}
                    href={link.href}
                    label={link.label}
                  />
                </li>
              ))}
            </ul>
          </section>
        ) : null}
      </main>
    </>
  )
}
