import { STALE_STATUS } from '@/domain/api-error'
import { messageFor } from '@/domain/messages'
import { ALLOWED_NEXT_STATUSES, TICKET_STATUS_LABELS } from '@/domain/tickets'
import type { TicketStatus } from '@/domain/tickets/types'

import type {
  StatusChangeInput,
  StatusDeps,
  StatusMenuItem,
  StatusOutcome
} from './types'

/**
 * The rows of the status menu (spec 003 AC-6.1, BR-3): the current status
 * with its check, then only the transitions allowed from it; "Fechado" is
 * told apart as the end of the work. A closed ticket has no rows (BR-5).
 */
export const statusMenuItems = (current: TicketStatus): StatusMenuItem[] => {
  const next: readonly TicketStatus[] = ALLOWED_NEXT_STATUSES[current]
  if (next.length === 0) return []
  return [current, ...next].map(status => ({
    current: status === current,
    label: TICKET_STATUS_LABELS[status],
    risky: status === 'fechado',
    status
  }))
}

/** "Status alterado para Em andamento." (AC-6.2). */
export const changedText = (status: TicketStatus) =>
  `Status alterado para ${TICKET_STATUS_LABELS[status]}.`

/**
 * One status change (AC-6.2 to AC-6.6), carrying the status the admin saw
 * (BR-7). A 409 means someone changed it meanwhile: its message, and the
 * page is loaded again (AC-6.3); a 404 also reloads (the not-found screen).
 * Every other error keeps the current status on screen.
 */
export const applyStatus = async (
  deps: StatusDeps,
  { from, ticketId, to }: StatusChangeInput
): Promise<StatusOutcome> => {
  const result = await deps.change({ de: from, para: to, ticketId })
  if (result.ok) return { kind: 'changed', message: changedText(to) }
  const { error } = result
  return {
    kind: 'failed',
    message: messageFor(error, 'status'),
    refresh: error.code === STALE_STATUS || error.status === 404
  }
}
