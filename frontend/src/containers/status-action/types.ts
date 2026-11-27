import type { TicketStatus } from '@/domain/tickets/types'
import type { ApiClient } from '@/repositories/api/types'

export interface StatusActionProps {
  current: TicketStatus
  ticketId: number
}

export interface StatusMenuItem {
  current: boolean
  label: string
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
