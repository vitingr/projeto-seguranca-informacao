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

export const changedText = (status: TicketStatus) =>
  `Status alterado para ${TICKET_STATUS_LABELS[status]}.`

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
