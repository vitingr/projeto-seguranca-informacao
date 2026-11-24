import { cn } from 'cn'
import { ShieldCheck } from 'lucide-react'

import { AreaCard } from '@/components/area-card'
import { Eyebrow } from '@/components/eyebrow'
import { Forbidden } from '@/components/forbidden'
import { staggerAt } from '@/components/stagger'
import { AppHeader } from '@/containers/app-header'
import { ROLE_LINKS } from '@/domain/role-links'
import { ROLE_NAMES } from '@/domain/role-names'
import { AREA_DETAILS } from '@/screens/home/constants'
import { requireArea, requireRole } from '@/services/session'

import type { RoleAreaProps } from './types'

const RISE = 'motion-safe:animate-rise'

/**
 * A role's own area (/admin, /minha-conta). It opens only after the server
 * answers the area endpoint; the panel shows what was checked. The other
 * areas of the role follow as shortcuts.
 */
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
  const checks = [
    { label: 'Papel', value: ROLE_NAMES[user.role] },
    { label: 'Página', mono: true, value: route },
    { label: 'Endpoint conferido', mono: true, value: endpoint },
    { label: 'Resposta', mono: true, value: '200 OK' }
  ]

  return (
    <>
      <AppHeader role={user.role} />
      <main className="mx-auto flex w-full max-w-7xl flex-col gap-12 px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
        <section className="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:items-start lg:gap-10">
          <div className={cn(RISE, 'flex flex-col gap-4 lg:col-span-6')}>
            <Eyebrow>
              {user.role === 'admin' ? 'Administração' : 'Conta'}
            </Eyebrow>
            <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
              {title}
            </h1>
            <p className="text-muted-foreground max-w-xl text-lg text-pretty">
              {description}
            </p>
          </div>

          <section
            className={cn(
              RISE,
              'motion-safe:anim-delay-75',
              'bg-card flex flex-col gap-5 rounded-3xl border p-6 sm:p-8 lg:col-span-6'
            )}
            aria-labelledby="server-check"
          >
            <div className="flex items-center gap-3">
              <span
                aria-hidden="true"
                className="bg-success-soft text-success flex size-10 items-center justify-center rounded-xl"
              >
                <ShieldCheck className="size-5" strokeWidth={1.75} />
              </span>
              <h2
                className="text-lg font-semibold tracking-tight"
                id="server-check"
              >
                Verificado pelo servidor
              </h2>
            </div>
            <p className="text-sm font-medium" data-testid="area">
              Área liberada pelo servidor: {area.area}
            </p>
            <dl className="flex flex-col border-t">
              {checks.map(check => (
                <div
                  className="flex items-center justify-between gap-4 border-b py-3 text-sm"
                  key={check.label}
                >
                  <dt className="text-muted-foreground">{check.label}</dt>
                  <dd
                    className={cn(
                      'min-w-0 truncate',
                      check.mono ? 'font-mono text-xs' : 'font-medium'
                    )}
                  >
                    {check.value}
                  </dd>
                </div>
              ))}
            </dl>
            <p className="text-muted-foreground text-xs text-pretty">
              A página só abre depois dessa resposta. Com outro papel, ela
              mostra o aviso de acesso negado, e a API responde 403.
            </p>
          </section>
        </section>

        {shortcuts.length > 0 ? (
          <section aria-labelledby="shortcuts" className="flex flex-col gap-5">
            <h2
              className="text-muted-foreground text-xs font-medium tracking-wider uppercase"
              id="shortcuts"
            >
              Outras áreas do seu papel
            </h2>
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
