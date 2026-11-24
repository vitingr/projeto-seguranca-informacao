import { cn } from 'cn'
import { ArrowRight, Check } from 'lucide-react'
import Link from 'next/link'

import { AreaCard } from '@/components/area-card'
import { Brand } from '@/components/brand'
import { SecurityDemo } from '@/components/security-demo'
import { STAGGER } from '@/components/stagger'
import { buttonVariants } from '@/components/ui/shadcn/button'
import { AppHeader } from '@/containers/app-header'
import { Reveal } from '@/containers/reveal'
import { ROLE_LINKS } from '@/domain/role-links'

import {
  AREA_DETAILS,
  CONTAINER,
  CONTROLS,
  FACTS,
  RISE,
  ROLES,
  SECTION_LINK,
  SECTION_TITLE
} from './constants'
import type { RoleHomeProps } from './types'

export const PublicHome: React.FC = () => (
  <div className="flex min-h-dvh flex-col">
    <header className="border-b bg-white">
      <div
        className={cn(
          CONTAINER,
          'flex h-16 items-center justify-between gap-6'
        )}
      >
        <Brand />
        <nav aria-label="Seções" className="hidden items-center gap-1 sm:flex">
          <a className={SECTION_LINK} href="#como-funciona">
            Como funciona
          </a>
          <a className={SECTION_LINK} href="#papeis">
            Papéis
          </a>
          <a className={SECTION_LINK} href="#seguranca">
            Segurança
          </a>
        </nav>
      </div>
    </header>

    <main className="flex flex-1 flex-col">
      <section className="border-b bg-white">
        <div
          className={cn(
            CONTAINER,
            'grid items-center gap-12 py-12 sm:py-16 lg:grid-cols-12 lg:gap-10 lg:py-24'
          )}
        >
          <div className="flex flex-col items-start gap-7 lg:col-span-5">
            <h1
              className={cn(
                RISE,
                STAGGER[1],
                'text-4xl leading-tight font-semibold text-balance sm:text-5xl lg:text-5xl xl:text-6xl'
              )}
            >
              Cada acesso conferido no servidor. Cada ação{' '}
              <span className="text-brand-ink">registrada.</span>
            </h1>
            <p
              className={cn(
                RISE,
                STAGGER[2],
                'text-muted-foreground max-w-lg text-base text-pretty sm:text-lg'
              )}
            >
              Administradores cadastram usuários, moderam o mural e consultam a
              auditoria. Usuários comuns publicam no mural e cuidam da própria
              conta. Entre para acessar as áreas do seu papel.
            </p>
            <div
              className={cn(
                RISE,
                STAGGER[3],
                'flex flex-wrap items-center gap-3'
              )}
            >
              <Link
                className={cn(buttonVariants({ size: 'xl' }), 'group/cta')}
                href="/login"
              >
                Entrar
                <ArrowRight
                  aria-hidden="true"
                  className="motion-safe:transition-transform motion-safe:group-hover/cta:translate-x-1"
                />
              </Link>
              <a
                className={buttonVariants({ size: 'xl', variant: 'ghost' })}
                href="#seguranca"
              >
                Ver os controles de segurança
              </a>
            </div>
            <dl
              className={cn(
                RISE,
                STAGGER[4],
                'grid w-full grid-cols-3 gap-4 border-t pt-6'
              )}
            >
              {FACTS.map(fact => (
                <div
                  className="flex flex-col-reverse justify-end gap-1"
                  key={fact.value}
                >
                  <dt className="text-xs text-neutral-500">{fact.label}</dt>
                  <dd className="text-lg font-semibold text-neutral-700 sm:text-xl">
                    {fact.value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>

          <div
            className={cn(RISE, STAGGER[3], 'scroll-mt-20 lg:col-span-7')}
            id="como-funciona"
          >
            <SecurityDemo />
          </div>
        </div>
      </section>

      <section className="scroll-mt-4" id="papeis">
        <div className={cn(CONTAINER, 'flex flex-col gap-10 py-16 sm:py-24')}>
          <Reveal className="flex max-w-2xl flex-col gap-4">
            <h2 className={SECTION_TITLE}>
              Dois papéis, cada um com a sua área
            </h2>
            <p className="text-muted-foreground text-pretty">
              O papel é definido no cadastro. Ele decide quais páginas abrem e
              quais ações o servidor aceita.
            </p>
          </Reveal>
          <div className="grid gap-4 md:grid-cols-2">
            {ROLES.map((role, index) => (
              <Reveal className={cn('h-full', STAGGER[index])} key={role.name}>
                <article className="bg-card hover:border-foreground/25 flex h-full flex-col gap-6 rounded-sm border p-4 motion-safe:transition-colors motion-safe:duration-300 sm:p-6">
                  <div className="flex flex-col gap-1">
                    <h3 className="text-xl font-semibold">{role.name}</h3>
                    <p className="text-muted-foreground text-sm">
                      {role.summary}
                    </p>
                  </div>
                  <ul className="flex flex-col gap-3 border-t pt-6">
                    {role.items.map(item => (
                      <li className="flex gap-3 text-sm" key={item}>
                        <Check
                          aria-hidden="true"
                          className="text-brand-ink size-4 shrink-0 translate-y-0.5"
                        />
                        {item}
                      </li>
                    ))}
                  </ul>
                </article>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-card scroll-mt-4 border-y" id="seguranca">
        <div className={cn(CONTAINER, 'flex flex-col gap-10 py-16 sm:py-24')}>
          <Reveal className="flex max-w-2xl flex-col gap-4">
            <h2 className={SECTION_TITLE}>O que protege cada sessão</h2>
            <p className="text-muted-foreground text-pretty">
              Controles que valem para todos os papéis, aplicados pelo backend.
            </p>
          </Reveal>
          <Reveal>
            <ol className="bg-border grid gap-px overflow-hidden rounded-sm border sm:grid-cols-2 lg:grid-cols-3">
              {CONTROLS.map((control, index) => (
                <li
                  className="bg-card flex flex-col gap-4 p-6"
                  key={control.title}
                >
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-brand-ink text-sm font-semibold">
                      {String(index + 1).padStart(2, '0')}
                    </span>
                    <code className="bg-muted text-muted-foreground rounded-md border px-2 py-0.5 font-mono text-xs">
                      {control.tag}
                    </code>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <h3 className="text-base font-semibold">{control.title}</h3>
                    <p className="text-muted-foreground text-sm text-pretty">
                      {control.description}
                    </p>
                  </div>
                </li>
              ))}
            </ol>
          </Reveal>
        </div>
      </section>

      <Reveal className={cn(CONTAINER, 'py-16 sm:py-24')}>
        <section className="bg-primary text-primary-foreground flex flex-col gap-8 rounded-3xl p-8 sm:p-12 md:flex-row md:items-end md:justify-between">
          <div className="flex max-w-xl flex-col gap-3">
            <h2 className="text-2xl font-semibold sm:text-3xl">
              Acesso só por convite
            </h2>
            <p className="text-primary-foreground/70 text-pretty">
              Não há cadastro aberto: as contas são criadas por um
              administrador, que define o papel de cada pessoa.
            </p>
          </div>
          <Link
            className="group/cta bg-primary-foreground text-primary hover:bg-primary-foreground/90 focus-visible:ring-brand/60 flex h-11 w-fit shrink-0 items-center gap-2 rounded-full px-5 text-sm font-medium outline-none focus-visible:ring-3 motion-safe:transition-colors"
            href="/login"
          >
            Ir para o login
            <ArrowRight
              aria-hidden="true"
              className="size-4 motion-safe:transition-transform motion-safe:group-hover/cta:translate-x-1"
            />
          </Link>
        </section>
      </Reveal>
    </main>

    <footer className="border-t">
      <div
        className={cn(
          CONTAINER,
          'text-muted-foreground flex flex-col gap-2 py-8 text-xs sm:flex-row sm:justify-between'
        )}
      >
        <span>Sistemas da Informação</span>
        <span>Administradores · Usuários comuns</span>
      </div>
    </footer>
  </div>
)

export const RoleHome: React.FC<RoleHomeProps> = ({ role }) => {
  const links = ROLE_LINKS[role]

  return (
    <>
      <AppHeader role={role} />
      <main className={cn(CONTAINER, 'flex flex-col gap-10 py-12 lg:py-20')}>
        <section className="grid gap-8 lg:grid-cols-12 lg:items-end">
          <div className="flex flex-col gap-4 lg:col-span-8">
            <h1
              className={cn(
                RISE,
                STAGGER[1],
                'text-4xl font-semibold sm:text-5xl'
              )}
            >
              Início
            </h1>
            <p
              className={cn(
                RISE,
                STAGGER[2],
                'text-muted-foreground max-w-xl text-pretty'
              )}
            >
              Escolha uma área para continuar. Você vê só as áreas do seu papel.
            </p>
          </div>
        </section>
        <ul
          className={cn(
            'grid gap-4 sm:grid-cols-2',
            links.length > 2 && 'lg:grid-cols-4'
          )}
          data-testid="home-links"
        >
          {links.map((link, index) => (
            <li key={link.href}>
              <AreaCard
                {...AREA_DETAILS[link.href]}
                className={cn(RISE, STAGGER[index + 2])}
                href={link.href}
                label={link.label}
              />
            </li>
          ))}
        </ul>
      </main>
    </>
  )
}
