import type { TicketStatus } from '@/domain/tickets/types'

export interface ThreadEventProps {
  actor: string
  now: string
  status?: TicketStatus
  time: string
}
