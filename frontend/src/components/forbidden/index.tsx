import { cn } from 'cn'
import { ArrowRight, ShieldX } from 'lucide-react'
import Link from 'next/link'

import { buttonVariants } from '@/components/ui/shadcn/button'
import { MESSAGES } from '@/domain/messages'
import { landingFor } from '@/domain/navigation'
import { ROLE_NAMES } from '@/domain/role-names'

import type { ForbiddenProps } from './types'

const RISE = 'motion-safe:animate-rise'

export const Forbidden: React.FC<ForbiddenProps> = ({ role }) => (
  <main className="mx-auto flex w-full max-w-7xl flex-1 items-center justify-center px-4 py-16 sm:px-6 lg:px-8">
    <div className="bg-card flex w-full max-w-lg flex-col items-center gap-6 rounded-3xl border px-6 py-12 text-center sm:px-12">
      <span
        className={cn(
          RISE,
          'bg-destructive-soft text-destructive flex size-14 items-center justify-center rounded-2xl'
        )}
        aria-hidden="true"
      >
        <ShieldX className="size-7" strokeWidth={1.75} />
      </span>
      <div
        className={cn(
          RISE,
          'motion-safe:anim-delay-75',
          'flex flex-col items-center gap-3'
        )}
      >
        <span className="text-muted-foreground font-mono text-xs">
          403 · Forbidden
        </span>
        <h1 className="text-3xl font-semibold tracking-tight">Acesso negado</h1>
        <p className="text-lg text-pretty" data-testid="forbidden" role="alert">
          {MESSAGES.forbiddenPage}
        </p>
        <p className="text-muted-foreground text-sm text-pretty">
          O servidor conferiu o seu papel ({ROLE_NAMES[role]}) e esta página não
          faz parte dele. Nada foi aberto nem alterado.
        </p>
      </div>
      <Link
        className={cn(
          RISE,
          'motion-safe:anim-delay-150',
          buttonVariants({ size: 'xl' }),
          'group/cta'
        )}
        href={landingFor(role)}
      >
        Ir para a minha página inicial
        <ArrowRight
          aria-hidden="true"
          className="motion-safe:transition-transform motion-safe:group-hover/cta:translate-x-1"
        />
      </Link>
    </div>
  </main>
)
