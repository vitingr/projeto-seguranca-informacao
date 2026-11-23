import Link from 'next/link'

import { AuditTable } from '@/components/audit-table'
import { Forbidden } from '@/components/forbidden'
import { Button, buttonVariants } from '@/components/ui/shadcn/button'
import { AppHeader } from '@/containers/app-header'
import { AUDIT_EVENT_LABELS, AUDIT_EVENT_TYPES } from '@/domain/audit'
import type { AuditEventType } from '@/domain/audit/types'
import { requireAuditEvents } from '@/services/audit'
import { requireRole } from '@/services/session'

import type { AuditLogProps } from './types'

const ROUTE = '/admin/auditoria'

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
      <main className="mx-auto w-full max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-6">
          <div className="flex flex-col gap-2">
            <h1 className="text-2xl font-semibold">Auditoria</h1>
            <p className="text-muted-foreground">
              Eventos de segurança, do mais recente ao mais antigo. O registro
              só recebe novos eventos: nada pode ser alterado ou apagado.
            </p>
          </div>
          {/* A plain GET form: the filter works without JavaScript. */}
          <form
            action={ROUTE}
            aria-label="Filtrar eventos"
            className="flex flex-col gap-3 sm:flex-row sm:items-end"
            method="get"
          >
            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-medium" htmlFor="audit-type">
                Tipo de evento
              </label>
              <select
                className="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 h-9 rounded-lg border px-3 text-sm outline-none focus-visible:ring-3"
                defaultValue={type ?? ''}
                id="audit-type"
                name="tipo"
              >
                <option value="">Todos</option>
                {AUDIT_EVENT_TYPES.map(t => (
                  <option key={t} value={t}>
                    {AUDIT_EVENT_LABELS[t]}
                  </option>
                ))}
              </select>
            </div>
            <Button type="submit" variant="outline">
              Filtrar
            </Button>
          </form>
          {log.items.length === 0 ? (
            <p className="text-muted-foreground" data-testid="audit-empty">
              Nenhum evento encontrado.
            </p>
          ) : (
            <AuditTable events={log.items} />
          )}
          {log.page > 1 || log.hasNext ? (
            <nav aria-label="Paginação" className="flex flex-wrap gap-2">
              {log.page > 1 ? (
                <Link
                  className={buttonVariants({ variant: 'outline' })}
                  href={pageHref(type, log.page - 1)}
                >
                  Mais recentes
                </Link>
              ) : null}
              {log.hasNext ? (
                <Link
                  className={buttonVariants({ variant: 'outline' })}
                  href={pageHref(type, log.page + 1)}
                >
                  Anteriores
                </Link>
              ) : null}
            </nav>
          ) : null}
        </div>
      </main>
    </>
  )
}
