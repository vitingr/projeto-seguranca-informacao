import type { TicketStatus } from '@/domain/tickets/types'
import type { ApiClient } from '@/repositories/api/types'

export interface StatusActionProps {
  /** The status of the ticket as the page loaded it (BR-7: sent as `de`). */
  current: TicketStatus
  ticketId: number
}

/** A row of the status menu: the current status (checked) or a next one. */
export interface StatusMenuItem {
  current: boolean
  label: string
  /** "Fechado": the end of the work, in discreet red (AC-6.1). */
  risky: boolean
  status: TicketStatus
}

export interface StatusDeps {
  change: ApiClient['changeStatus']
}

export interface StatusChangeInput {
  from: TicketStatus
  ticketId: number
  to: TicketStatus
}

export type StatusOutcome =
  | { kind: 'changed'; message: string }
  | { kind: 'failed'; message: string; refresh: boolean }
