import { cn } from 'cn'

import { staggerAt } from '@/components/stagger'
import { Badge } from '@/components/ui/shadcn/badge'
import { AUDIT_EVENT_LABELS } from '@/domain/audit'
import type { AuditActor, AuditTarget } from '@/domain/audit/types'
import { formatDateTime } from '@/utils/helpers/format-date-time'

import type { AuditCellProps, AuditTableProps } from './types'

const COLUMNS = ['Evento', 'Ator', 'Alvo', 'IP', 'Data'] as const

/** One badge color per event, also used by the filter chips. */
export const AUDIT_EVENT_BADGE = {
  login_failed: 'destructive',
  login_succeeded: 'success',
  logout: 'secondary',
  post_removed: 'warning',
  user_created: 'info'
} as const

const actorText = (actor: AuditActor | null) =>
  actor === null ? '—' : (actor.login ?? `Usuário #${actor.userId} (excluído)`)

const targetText = (target: AuditTarget | null) => {
  if (target === null) return '—'
  if (target.type === 'post') return `Post #${target.id}`
  return target.login ?? `Usuário #${target.id} (excluído)`
}

// One cell: a table cell from md up; below, a labelled line of a stacked
// card, so the log reads from 360px wide without horizontal scrolling.
const Cell: React.FC<AuditCellProps> = ({
  children,
  className,
  label,
  testId
}) => (
  <td
    className={cn(
      'flex gap-3 text-sm wrap-anywhere md:table-cell md:px-4 md:py-3.5 md:align-middle',
      className
    )}
    data-testid={testId}
  >
    <span className="text-muted-foreground w-12 shrink-0 md:hidden">
      {label}
    </span>
    {children}
  </td>
)

/**
 * The audit log (006). Every value is rendered as React text. The events
 * hold no free text (D-4): the logins come from the users table.
 */
export const AuditTable: React.FC<AuditTableProps> = ({ events }) => (
  <div className="md:bg-card md:overflow-hidden md:rounded-2xl md:border">
    <table className="flex w-full flex-col md:table">
      <thead className="hidden md:table-header-group">
        <tr className="bg-muted text-muted-foreground border-b text-left">
          {COLUMNS.map(column => (
            <th
              className="px-4 py-3 text-xs font-medium tracking-wider uppercase"
              key={column}
              scope="col"
            >
              {column}
            </th>
          ))}
        </tr>
      </thead>
      <tbody className="flex flex-col gap-3 md:table-row-group">
        {events.map((event, index) => (
          <tr
            className={cn(
              'motion-safe:animate-rise bg-card md:hover:bg-muted/50 flex flex-col gap-2 rounded-2xl border p-4 md:table-row md:rounded-none md:border-0 md:border-b md:p-0 md:last:border-b-0 motion-safe:md:transition-colors',
              staggerAt(index)
            )}
            data-event={event.event}
            data-testid="audit-row"
            key={event.id}
          >
            <Cell
              className="pb-1 md:pb-3.5"
              label="Evento"
              testId="audit-event"
            >
              <Badge
                className="h-6 px-2.5"
                variant={AUDIT_EVENT_BADGE[event.event]}
              >
                {AUDIT_EVENT_LABELS[event.event]}
              </Badge>
            </Cell>
            <Cell label="Ator" testId="audit-actor">
              <span className="font-medium">{actorText(event.actor)}</span>
            </Cell>
            <Cell label="Alvo" testId="audit-target">
              {targetText(event.target)}
            </Cell>
            <Cell label="IP" testId="audit-ip">
              <span className="text-muted-foreground font-mono text-xs md:text-sm">
                {event.ip ?? '—'}
              </span>
            </Cell>
            <Cell label="Data" testId="audit-date">
              <time
                className="text-muted-foreground font-mono text-xs whitespace-nowrap md:text-sm"
                dateTime={event.createdAt}
              >
                {formatDateTime(event.createdAt)}
              </time>
            </Cell>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
)
