import type { TicketStatus } from '@/domain/tickets/types'

export interface StatusBadgeProps {
  className?: string
  status: TicketStatus
}
