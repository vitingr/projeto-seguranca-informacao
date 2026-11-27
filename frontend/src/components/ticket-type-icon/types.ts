import type { TicketType } from '@/domain/tickets/types'

export interface TicketTypeIconProps {
  className?: string
  selected?: boolean
  type: null | TicketType
}
