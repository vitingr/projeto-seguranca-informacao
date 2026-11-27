import type { TICKET_TYPES } from '@/domain/tickets'
import type { TicketType } from '@/domain/tickets/types'

export interface TicketTypePickerProps {
  error?: string
  labelledBy: string
  onValueChange: (type: TicketType) => void
  options: typeof TICKET_TYPES
  value: TicketType | null
}
