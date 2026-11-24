import { AUDIT_EVENT_LABELS } from '@/domain/audit'
import type { AuditActor, AuditTarget } from '@/domain/audit/types'
import { formatDateTime } from '@/utils/helpers/format-date-time'

import type { AuditCellProps, AuditTableProps } from './types'

const COLUMNS = ['Evento', 'Ator', 'Alvo', 'IP', 'Data'] as const

const actorText = (actor: AuditActor | null) =>
  actor === null ? '—' : (actor.login ?? `Usuário #${actor.userId} (excluído)`)

const targetText = (target: AuditTarget | null) => {
  if (target === null) return '—'
  if (target.type === 'post') return `Post #${target.id}`
  return target.login ?? `Usuário #${target.id} (excluído)`
}

// One cell: a table cell from md up; below, a labelled line of a stacked
// card, so the log reads from 360px wide without horizontal scrolling.
const Cell: React.FC<AuditCellProps> = ({ children, label, testId }) => (
  <td
    className="flex flex-wrap gap-1 wrap-anywhere md:table-cell md:px-3 md:py-2"
    data-testid={testId}
  >
    <span className="text-muted-foreground md:hidden">{label}:</span>
    {children}
  </td>
)

/**
 * The audit log (006). Every value is rendered as React text. The events
 * hold no free text (D-4): the logins come from the users table.
 */
export const AuditTable: React.FC<AuditTableProps> = ({ events }) => (
  <table className="flex w-full flex-col text-sm md:table">
    <thead className="hidden md:table-header-group">
      <tr className="border-border border-b text-left">
        {COLUMNS.map(column => (
          <th className="px-3 py-2 font-medium" key={column} scope="col">
            {column}
          </th>
        ))}
      </tr>
    </thead>
    <tbody className="flex flex-col gap-3 md:table-row-group">
      {events.map(event => (
        <tr
          className="border-border flex flex-col gap-1 rounded-lg border p-3 md:table-row md:rounded-none md:border-0 md:border-b md:p-0"
          data-event={event.event}
          data-testid="audit-row"
          key={event.id}
        >
          <Cell label="Evento" testId="audit-event">
            <span className="font-medium">
              {AUDIT_EVENT_LABELS[event.event]}
            </span>
          </Cell>
          <Cell label="Ator" testId="audit-actor">
            {actorText(event.actor)}
          </Cell>
          <Cell label="Alvo" testId="audit-target">
            {targetText(event.target)}
          </Cell>
          <Cell label="IP" testId="audit-ip">
            <span className="font-mono">{event.ip ?? '—'}</span>
          </Cell>
          <Cell label="Data" testId="audit-date">
            <time dateTime={event.createdAt}>
              {formatDateTime(event.createdAt)}
            </time>
          </Cell>
        </tr>
      ))}
    </tbody>
  </table>
)
