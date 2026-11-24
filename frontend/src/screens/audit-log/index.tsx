import { cn } from 'cn'
import { ArrowLeft, ArrowRight, Lock, ScrollText } from 'lucide-react'
import Link from 'next/link'

import { AUDIT_EVENT_BADGE, AuditTable } from '@/components/audit-table'
import { Eyebrow } from '@/components/eyebrow'
import { Forbidden } from '@/components/forbidden'
import { Badge } from '@/components/ui/shadcn/badge'
import { Button, buttonVariants } from '@/components/ui/shadcn/button'
import {
  NativeSelect,
  NativeSelectOption
} from '@/components/ui/shadcn/native-select'
import { AppHeader } from '@/containers/app-header'
import { AUDIT_EVENT_LABELS, AUDIT_EVENT_TYPES } from '@/domain/audit'
import type { AuditEventType } from '@/domain/audit/types'
import { requireAuditEvents } from '@/services/audit'
import { requireRole } from '@/services/session'

import type { AuditLogProps } from './types'

const ROUTE = '/admin/auditoria'

const RISE = 'motion-safe:animate-rise'

// The dot of each filter chip, in the color of its event badge.
const DOT = {
  destructive: 'bg-destructive',
  info: 'bg-info',
  secondary: 'bg-muted-foreground',
  success: 'bg-success',
  warning: 'bg-warning'
} as const

const CHIP =
  'hover:border-foreground/30 focus-visible:ring-ring/50 aria-[current=page]:border-primary aria-[current=page]:bg-primary aria-[current=page]:text-primary-foreground bg-card flex h-8 shrink-0 items-center gap-2 rounded-full border px-3 text-sm whitespace-nowrap outline-none focus-visible:ring-3 motion-safe:transition-colors'

const pageHref = (type: AuditEventType | null, page: number) => {
  const query = new URLSearchParams()
  if (type) query.set('tipo', type)
  if (page > 1) query.set('pagina', String(page))
  const qs = query.toString()
  return qs ? `${ROUTE}?${qs}` : ROUTE
}

// Admin only (006 S-3). The backend checks the role again on every call and
// answers 403 if it changed in between.
export const AuditLog: React.FC<AuditLogProps> = async ({ page, type }) => {
  const { allowed, user } = await requireRole(ROUTE)
  const result = allowed ? await requireAuditEvents(type, page) : null

  if (result?.kind !== 'ok') {
    return (
      <>
        <AppHeader role={user.role} />
        <Forbidden role={user.role} />
      </>
    )
  }

  const log = result.page
  return (
    <>
      <AppHeader role={user.role} />
      <main className="mx-auto flex w-full max-w-7xl flex-col gap-8 px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
        <div
          className={cn(
            RISE,
            'flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between'
          )}
        >
          <div className="flex flex-col gap-4">
            <Eyebrow>Registro</Eyebrow>
            <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
              Auditoria
            </h1>
            <p className="text-muted-foreground max-w-2xl text-pretty">
              Eventos de segurança, do mais recente ao mais antigo. O registro
              só recebe novos eventos: nada pode ser alterado ou apagado.
            </p>
          </div>
          <Badge className="h-7 gap-1.5 px-3" variant="outline">
            <Lock aria-hidden="true" />
            Somente leitura
          </Badge>
        </div>

        <div
          className={cn(
            RISE,
            'motion-safe:anim-delay-75',
            'bg-card flex flex-col gap-5 rounded-2xl border p-4 sm:p-5 lg:flex-row lg:items-end lg:justify-between'
          )}
        >
          <nav
            aria-label="Filtros rápidos"
            className="-mx-1 flex min-w-0 gap-2 overflow-x-auto px-1 py-1 lg:flex-wrap"
          >
            <Link
              aria-current={type === null ? 'page' : undefined}
              className={CHIP}
              href={pageHref(null, 1)}
            >
              Todos
            </Link>
            {AUDIT_EVENT_TYPES.map(t => (
              <Link
                aria-current={type === t ? 'page' : undefined}
                className={CHIP}
                href={pageHref(t, 1)}
                key={t}
              >
                <span
                  className={cn(
                    'size-2 rounded-full',
                    DOT[AUDIT_EVENT_BADGE[t]]
                  )}
                  aria-hidden="true"
                />
                {AUDIT_EVENT_LABELS[t]}
              </Link>
            ))}
          </nav>

          {/* A plain GET form: the filter works without JavaScript. */}
          <form
            action={ROUTE}
            aria-label="Filtrar eventos"
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
            method="get"
          >
            <div className="flex flex-col gap-2">
              <label className="text-sm font-medium" htmlFor="audit-type">
                Tipo de evento
              </label>
              <NativeSelect
                className="w-full sm:w-56"
                defaultValue={type ?? ''}
                id="audit-type"
                name="tipo"
                size="lg"
              >
                <NativeSelectOption value="">Todos</NativeSelectOption>
                {AUDIT_EVENT_TYPES.map(t => (
                  <NativeSelectOption key={t} value={t}>
                    {AUDIT_EVENT_LABELS[t]}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </div>
            <Button size="xl" type="submit" variant="outline">
              Filtrar
            </Button>
          </form>
        </div>

        <section aria-labelledby="events" className="flex flex-col gap-4">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h2 className="text-lg font-semibold tracking-tight" id="events">
              {type ? AUDIT_EVENT_LABELS[type] : 'Todos os eventos'}
            </h2>
            <span className="text-muted-foreground text-sm">
              Página {log.page}
            </span>
          </div>

          {log.items.length === 0 ? (
            <div
              className={cn(
                RISE,
                'flex flex-col items-center gap-4 rounded-3xl border border-dashed px-6 py-16 text-center'
              )}
            >
              <span
                aria-hidden="true"
                className="bg-brand-soft text-brand-ink flex size-12 items-center justify-center rounded-2xl"
              >
                <ScrollText className="size-6" strokeWidth={1.75} />
              </span>
              <p
                className="text-lg font-semibold tracking-tight"
                data-testid="audit-empty"
              >
                Nenhum evento encontrado.
              </p>
              <p className="text-muted-foreground max-w-sm text-sm text-pretty">
                {type
                  ? 'Nenhum evento deste tipo nesta página. Tente outro filtro.'
                  : 'Os eventos aparecem aqui assim que alguém entrar ou sair.'}
              </p>
            </div>
          ) : (
            <AuditTable events={log.items} />
          )}

          {log.page > 1 || log.hasNext ? (
            <nav
              aria-label="Paginação"
              className="flex items-center justify-between gap-3 border-t pt-6"
            >
              {log.page > 1 ? (
                <Link
                  className={buttonVariants({ size: 'lg', variant: 'outline' })}
                  href={pageHref(type, log.page - 1)}
                >
                  <ArrowLeft aria-hidden="true" data-icon="inline-start" />
                  Mais recentes
                </Link>
              ) : (
                <span />
              )}
              <span className="text-muted-foreground hidden text-sm sm:inline">
                Página {log.page}
              </span>
              {log.hasNext ? (
                <Link
                  className={buttonVariants({ size: 'lg', variant: 'outline' })}
                  href={pageHref(type, log.page + 1)}
                >
                  Anteriores
                  <ArrowRight aria-hidden="true" data-icon="inline-end" />
                </Link>
              ) : (
                <span />
              )}
            </nav>
          ) : null}
        </section>
      </main>
    </>
  )
}
